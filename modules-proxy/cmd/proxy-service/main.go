package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"modules-proxy/internal/types"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
)

const (
	sessionCookieName  = "session_id"
	defaultGatewayPort = 8080
	moduleSessionPath  = "/_pb/session"
)

var (
	errNoSession          = errors.New("no session")
	errSessionExpired     = errors.New("session expired")
	errInvalidToken       = errors.New("invalid session token")
	errTokenExpired       = errors.New("session token expired")
	sessionCookieSameSite = http.SameSiteNoneMode
	authDebugEnabled      = strings.EqualFold(os.Getenv("PROXY_DEBUG_AUTH"), "1")
)

type config struct {
	PostgresURL        string
	ListenAddr         string
	Channel            string
	AllowedDomains     []string
	IframeAllowedHosts []string
	NetControllerURL   string
	GatewayPort        int
	SessionSecret      []byte
	SessionCookieTTL   time.Duration
	LoginURL           string
}

type cachedPage struct {
	Slug              string
	ModuleID          string
	ModuleSlug        string
	ModuleStatus      string
	IframeOnly        bool
	NeedAuth          bool
	HasPageRoles      bool
	HasForbiddenRoles bool
	Network           string
	TargetContainer   string
	TargetPort        int
}

type sessionUser struct {
	ID        string
	FtLogin   string
	LastSeen  time.Time
	ExpiresAt time.Time
	SessionID string
}

type moduleAccessClaims struct {
	SessionID string `json:"sid"`
	Slug      string `json:"slug"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Nonce     string `json:"n"`
}

type pageStore struct {
	mu    sync.RWMutex
	pages map[string]cachedPage
}

func newPageStore() *pageStore {
	return &pageStore{pages: make(map[string]cachedPage)}
}

func (s *pageStore) replace(pages []cachedPage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]cachedPage, len(pages))
	for _, p := range pages {
		next[p.Slug] = p
	}
	s.pages = next
}

func (s *pageStore) get(slug string) (cachedPage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pages[slug]
	return p, ok
}

type netControllerClient struct {
	baseURL string
	client  *http.Client
}

func newNetControllerClient(rawURL string) *netControllerClient {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return nil
	}
	return &netControllerClient{
		baseURL: strings.TrimRight(raw, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *netControllerClient) buildURL(slug string) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s/gateways/%s", c.baseURL, url.PathEscape(slug))
}

func (c *netControllerClient) Status(ctx context.Context, slug string) (types.ProxyStatusPayload, error) {
	if c == nil {
		return types.ProxyStatusPayload{Message: "network controller disabled"}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURL(slug), nil)
	if err != nil {
		return types.ProxyStatusPayload{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return types.ProxyStatusPayload{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return types.ProxyStatusPayload{}, fmt.Errorf("net-controller status failed: %s", resp.Status)
	}
	var payload types.ProxyStatusPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return types.ProxyStatusPayload{}, err
	}
	return payload, nil
}

func (c *netControllerClient) Reconcile(ctx context.Context, slug string) (types.ProxyStatusPayload, error) {
	if c == nil {
		return types.ProxyStatusPayload{Message: "network controller disabled"}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(slug), nil)
	if err != nil {
		return types.ProxyStatusPayload{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return types.ProxyStatusPayload{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return types.ProxyStatusPayload{}, fmt.Errorf("net-controller reconcile failed: %s", resp.Status)
	}
	var payload types.ProxyStatusPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return types.ProxyStatusPayload{}, err
	}
	return payload, nil
}

type proxyService struct {
	db               *sqlx.DB
	store            *pageStore
	connInfo         string
	channelName      string
	allowedSuffix    []string
	iframeHosts      map[string]struct{}
	frameAncestors   string
	netClient        *netControllerClient
	gatewayPort      int
	sessionSecret    []byte
	sessionCookieTTL time.Duration
	loginURL         string
	activityMu       sync.Mutex
	activityLastLog  map[string]time.Time
}

func newProxyService(db *sqlx.DB, connInfo, channel string, suffixes []string, iframeHosts []string, netClient *netControllerClient, gatewayPort int, sessionSecret []byte, cookieTTL time.Duration, loginURL string) *proxyService {
	hostMap := make(map[string]struct{})
	for _, h := range iframeHosts {
		h = strings.TrimSpace(strings.ToLower(h))
		if h != "" {
			hostMap[h] = struct{}{}
		}
	}
	return &proxyService{
		db:               db,
		store:            newPageStore(),
		connInfo:         connInfo,
		channelName:      channel,
		allowedSuffix:    suffixes,
		iframeHosts:      hostMap,
		frameAncestors:   buildFrameAncestorsDirective(hostMap),
		netClient:        netClient,
		gatewayPort:      gatewayPort,
		sessionSecret:    sessionSecret,
		sessionCookieTTL: cookieTTL,
		loginURL:         strings.TrimSpace(loginURL),
		activityLastLog:  make(map[string]time.Time),
	}
}

func (p *proxyService) refreshPages(ctx context.Context) error {
	pages, err := p.fetchPages(ctx)
	if err != nil {
		return err
	}
	p.store.replace(pages)
	log.Printf("[proxy-service] loaded %d module pages", len(pages))
	return nil
}

func (p *proxyService) fetchPages(ctx context.Context) ([]cachedPage, error) {
	const query = `
		SELECT mp.slug,
		       mp.target_container,
		       mp.target_port,
		       mp.module_id,
		       mp.iframe_only,
		       mp.need_auth,
		       EXISTS (
		           SELECT 1
		             FROM module_page_roles pr
		            WHERE pr.page_id = mp.id
		       ) AS has_page_roles,
		       EXISTS (
		           SELECT 1
		             FROM module_page_forbidden_roles pfr
		            WHERE pfr.page_id = mp.id
		       ) AS has_forbidden_roles,
		       m.slug AS module_slug,
		       m.status AS module_status,
		       COALESCE(mp.network_name, '') AS network_name
		FROM module_page mp
		JOIN modules m ON m.id = mp.module_id
        WHERE mp.target_container IS NOT NULL
          AND mp.target_port IS NOT NULL
	`
	type row struct {
		Slug              string `db:"slug"`
		TargetContainer   string `db:"target_container"`
		TargetPort        int    `db:"target_port"`
		ModuleID          string `db:"module_id"`
		IframeOnly        bool   `db:"iframe_only"`
		NeedAuth          bool   `db:"need_auth"`
		HasPageRoles      bool   `db:"has_page_roles"`
		HasForbiddenRoles bool   `db:"has_forbidden_roles"`
		ModuleSlug        string `db:"module_slug"`
		ModuleStatus      string `db:"module_status"`
		NetworkName       string `db:"network_name"`
	}
	var rows []row
	if err := p.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	cached := make([]cachedPage, 0, len(rows))
	for _, r := range rows {
		slug := strings.TrimSpace(r.Slug)
		if slug == "" {
			continue
		}
		cached = append(cached, cachedPage{
			Slug:              slug,
			ModuleID:          r.ModuleID,
			ModuleSlug:        r.ModuleSlug,
			ModuleStatus:      r.ModuleStatus,
			IframeOnly:        r.IframeOnly,
			NeedAuth:          r.NeedAuth,
			HasPageRoles:      r.HasPageRoles,
			HasForbiddenRoles: r.HasForbiddenRoles,
			Network:           strings.TrimSpace(r.NetworkName),
			TargetContainer:   strings.TrimSpace(r.TargetContainer),
			TargetPort:        r.TargetPort,
		})
	}
	return cached, nil
}

func (p *proxyService) listenForChanges(ctx context.Context) {
	report := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[proxy-service] listener event=%v error=%v", ev, err)
		}
	}
	listener := pq.NewListener(p.connInfo, 5*time.Second, time.Minute, report)
	if err := listener.Listen(p.channelName); err != nil {
		log.Printf("[proxy-service] failed to LISTEN %s: %v", p.channelName, err)
		return
	}
	log.Printf("[proxy-service] listening for notifications on %q", p.channelName)

	for {
		select {
		case <-ctx.Done():
			listener.Close()
			return
		case n := <-listener.Notify:
			if n == nil {
				continue
			}
			log.Printf("[proxy-service] received notification %s", n.Channel)
			if err := p.refreshPages(context.Background()); err != nil {
				log.Printf("[proxy-service] failed to refresh pages: %v", err)
			}
		case <-time.After(2 * time.Minute):
			go func() {
				if err := listener.Ping(); err != nil {
					log.Printf("[proxy-service] listener ping failed: %v", err)
				}
			}()
		}
	}
}

func (p *proxyService) handleSessionBootstrap(w http.ResponseWriter, r *http.Request, slug string) {
	if len(p.sessionSecret) == 0 {
		http.Error(w, "session exchange disabled", http.StatusServiceUnavailable)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	claims, err := parseModuleAccessToken(token, p.sessionSecret)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errTokenExpired) {
			status = http.StatusUnauthorized
		}
		http.Error(w, "invalid session token", status)
		return
	}
	if claims.Slug != slug {
		http.Error(w, "slug mismatch", http.StatusForbidden)
		return
	}
	p.setModuleSessionCookie(w, claims.SessionID, isHTTPS(r))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<!doctype html><title>ok</title><body></body>"))
}

func (p *proxyService) handleGatewayRequest(w http.ResponseWriter, r *http.Request) {
	slug, ok := p.extractSlug(r.Host)
	if !ok {
		log.Printf("[proxy-service] rejected host=%q path=%s: unable to derive slug (allowed suffixes=%v)", r.Host, r.URL.Path, p.allowedSuffix)
		http.NotFound(w, r)
		return
	}
	page, ok := p.store.get(slug)
	if !ok {
		log.Printf("[proxy-service] slug=%q not found for host=%q", slug, r.Host)
		http.NotFound(w, r)
		return
	}
	if page.ModuleStatus != "" && page.ModuleStatus != "enabled" {
		log.Printf("[proxy-service] slug=%q module_status=%q: serving maintenance page", slug, page.ModuleStatus)
		writeModuleDisabledResponse(w, r)
		return
	}
	if r.URL.Path == moduleSessionPath {
		p.handleSessionBootstrap(w, r, slug)
		return
	}
	user, allowed, reason := p.authorizePageRequest(w, r, page)
	if !allowed {
		log.Printf("[proxy-service] denied slug=%q host=%q referer=%q reason=%s", slug, r.Host, r.Referer(), reason)
		return
	}
	if user != nil {
		p.recordActivity(r.Context(), page.ModuleID, user.ID)
	}

	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", gatewayName(slug), p.gatewayPort),
	}
	log.Printf("[proxy-service] routing slug=%q host=%q referer=%q to gateway %s", slug, r.Host, r.Referer(), target.Host)

	originalProto := "http"
	if isHTTPS(r) {
		originalProto = "https"
	}
	r.Header.Set("X-Forwarded-Proto", originalProto)
	r.Header.Set("X-Forwarded-Host", r.Host)

	proxy := newReverseProxy(target, p.frameAncestors)
	proxy.ServeHTTP(w, r)
}

func (p *proxyService) extractSlug(hostPort string) (string, bool) {
	host := strings.ToLower(strings.TrimSpace(hostPort))
	if host == "" {
		return "", false
	}
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	for _, suffix := range p.allowedSuffix {
		sfx := strings.ToLower(strings.TrimSpace(suffix))
		if sfx == "" {
			continue
		}
		if !strings.HasPrefix(sfx, ".") {
			sfx = "." + sfx
		}
		if strings.HasSuffix(host, sfx) && len(host) > len(sfx) {
			slug := host[:len(host)-len(sfx)]
			if isValidSlug(slug) {
				return slug, true
			}
		}
	}
	return "", false
}

func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

func gatewayName(slug string) string {
	return fmt.Sprintf("gateway-%s", dnsSafeSlug(slug))
}

func dnsSafeSlug(slug string) string {
	slug = strings.ToLower(slug)
	slug = strings.TrimSpace(slug)
	var builder strings.Builder
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-':
			builder.WriteRune('-')
		default:
			builder.WriteRune('-')
		}
	}
	cleaned := strings.Trim(builder.String(), "-")
	if cleaned == "" {
		return "page"
	}
	return cleaned
}

// recordActivity logs a usage ping for a module/user pair, throttled to at
// most one row per minute per (module, user) so chatty modules don't flood
// the activity table. The 60-minute inactivity gap used to sessionize these
// pings into "activity sessions" is computed at query time by the stats API.
func (p *proxyService) recordActivity(ctx context.Context, moduleID, userID string) {
	if moduleID == "" || userID == "" {
		return
	}
	key := moduleID + "|" + userID
	now := time.Now()

	p.activityMu.Lock()
	last, seen := p.activityLastLog[key]
	if seen && now.Sub(last) < time.Minute {
		p.activityMu.Unlock()
		return
	}
	p.activityLastLog[key] = now
	p.activityMu.Unlock()

	const query = `INSERT INTO module_activity (module_id, user_id, created_at) VALUES ($1, $2, $3)`
	if _, err := p.db.ExecContext(ctx, query, moduleID, userID, now); err != nil {
		log.Printf("[proxy-service] failed to record activity module=%s user=%s: %v", moduleID, userID, err)
	}
}

func (p *proxyService) authorizePageRequest(w http.ResponseWriter, r *http.Request, page cachedPage) (*sessionUser, bool, string) {
	if authDebugEnabled {
		var cookieNames []string
		for _, c := range r.Cookies() {
			cookieNames = append(cookieNames, c.Name)
		}
		log.Printf("[proxy-service][auth-debug] slug=%s host=%s referer=%q cookies=%v session_id=%q",
			page.Slug, r.Host, r.Header.Get("Referer"), cookieNames, readSessionID(r))
	}

	ctx := r.Context()
	user, err := p.authenticateRequest(ctx, r)
	switch {
	case err == errNoSession:
		if authDebugEnabled {
			log.Printf("[proxy-service][auth-debug] authenticateRequest -> no session")
		}
		user = nil
	case err == errSessionExpired:
		if authDebugEnabled {
			log.Printf("[proxy-service][auth-debug] authenticateRequest -> session expired")
		}
		clearSessionCookie(w, isHTTPS(r))
		writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized", accentRed, iconSVGLock,
			"Session expired",
			"Your session has expired.",
			"Please sign in again to continue.",
			"")
		return nil, false, "session expired"
	case err != nil:
		log.Printf("[proxy-service] auth failure: %v", err)
		http.Error(w, "internal authentication error", http.StatusInternalServerError)
		return nil, false, "authentication backend error"
	}

	if user != nil {
		blacklisted, err := p.isUserBlacklisted(ctx, user.ID)
		if err != nil {
			log.Printf("[proxy-service] blacklist check failed for user %s: %v", user.ID, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return nil, false, "blacklist lookup failed"
		}
		if blacklisted {
			_ = p.deleteUserSessions(ctx, user.ID)
			clearSessionCookie(w, isHTTPS(r))
			writeErrorResponse(w, r, http.StatusForbidden, "blacklisted", accentRed, iconSVGLock,
				"Account blacklisted",
				"Your account is currently blacklisted.",
				"Contact your bocal for more information.",
				"")
			return nil, false, "user is blacklisted"
		}
	}

	moduleHost := hostWithoutPort(r.Host)
	if page.IframeOnly && !p.isIframeRefererAllowed(r, r.Header.Get("Referer"), moduleHost) {
		writeErrorResponse(w, r, http.StatusForbidden, "iframe_required", accentBlue, iconSVGFrame,
			"Embedded access only",
			"This page must be loaded from Pan Bagnat.",
			"Open it from your Pan Bagnat sidebar instead of visiting this link directly.",
			actionButtonHTML("Go back to Pan Bagnat", p.panBagnatModuleURL(page.Slug)))
		return nil, false, "missing iframe referer"
	}
	requiresLogin := page.NeedAuth
	if requiresLogin && user == nil {
		if authDebugEnabled {
			log.Printf("[proxy-service][auth-debug] user nil but referer=%q need_auth=%v has_page_roles=%v", r.Header.Get("Referer"), page.NeedAuth, page.HasPageRoles)
		}
		if p.redirectToLoginIfPossible(w, r, r.Header.Get("Referer")) {
			return nil, false, "redirect_login"
		}
		writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized", accentRed, iconSVGLock,
			"Sign-in required",
			"Please sign in to continue.",
			"You need to be authenticated to view this page.",
			"")
		return nil, false, "not authenticated"
	}
	if user != nil && page.NeedAuth && page.HasForbiddenRoles {
		forbidden, err := p.userHasForbiddenRole(ctx, user.ID, page.Slug)
		if err != nil {
			log.Printf("[proxy-service] forbidden-role check failed for user %s on %s: %v", user.ID, page.Slug, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return nil, false, "forbidden role lookup failed"
		}
		if forbidden {
			writeErrorResponse(w, r, http.StatusForbidden, "forbidden", accentRed, iconSVGLock,
				"Access restricted",
				"You don't have access to this page.",
				"You may not have the required role for this module.",
				"")
			return nil, false, "forbidden"
		}
	}
	if user != nil && page.NeedAuth && page.HasPageRoles {
		allowed, err := p.userCanAccessPage(ctx, user.ID, page.Slug)
		if err != nil {
			log.Printf("[proxy-service] access check failed for user %s on %s: %v", user.ID, page.Slug, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return nil, false, "access lookup failed"
		}
		if !allowed {
			writeErrorResponse(w, r, http.StatusForbidden, "forbidden", accentRed, iconSVGLock,
				"Access restricted",
				"You don't have access to this page.",
				"You may not have the required role for this module.",
				"")
			return nil, false, "forbidden"
		}
	}
	if authDebugEnabled {
		userID := "<nil>"
		if user != nil {
			userID = user.ID
		}
		log.Printf("[proxy-service][auth-debug] access granted slug=%s user=%s", page.Slug, userID)
	}
	return user, true, ""
}

func (p *proxyService) userCanAccessPage(ctx context.Context, userID, slug string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			  FROM users u
			 WHERE u.id = $1
			   AND (
			        EXISTS (
			            SELECT 1
			              FROM user_roles ur_admin
			             WHERE ur_admin.user_id = u.id
			               AND ur_admin.role_id = 'roles_admin'
			        )
			        OR EXISTS (
			            SELECT 1
			              FROM user_roles ur
			              JOIN module_page_roles pr ON pr.role_id = ur.role_id
			              JOIN module_page mp ON mp.id = pr.page_id
			             WHERE ur.user_id = u.id
			               AND mp.slug = $2
			        )
			   )
		)
	`
	var exists bool
	if err := p.db.QueryRowContext(ctx, query, userID, slug).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (p *proxyService) userHasForbiddenRole(ctx context.Context, userID, slug string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			  FROM user_roles ur
			  JOIN module_page_forbidden_roles pfr ON pfr.role_id = ur.role_id
			  JOIN module_page mp ON mp.id = pfr.page_id
			 WHERE ur.user_id = $1
			   AND mp.slug = $2
		)
	`
	var exists bool
	if err := p.db.QueryRowContext(ctx, query, userID, slug).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (p *proxyService) isRefererFromParent(raw string) bool {
	if len(p.iframeHosts) == 0 {
		return false
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	_, ok := p.iframeHosts[host]
	return ok
}

func (p *proxyService) isIframeRefererAllowed(r *http.Request, raw string, moduleHost string) bool {
	// Sec-Fetch-Dest is set by the browser itself (page script cannot forge
	// it). Only "document" unambiguously means a real top-level navigation —
	// that's the one case we must reject, since a top-level visit can still
	// carry a Referer that matches Pan Bagnat's own host (e.g. the user
	// opened the sidebar link in a new tab, or pasted it). Every other
	// value — "iframe"/"frame" (a genuine embed), and non-navigation
	// subresource fetches ("empty" for XHR/fetch, "script", "style",
	// "image", ...) — must be let through: those can only originate from a
	// document that already loaded successfully, which means it already
	// passed this very check at its own navigation time. Rejecting "empty"
	// here would break every same-origin API call a legitimately embedded
	// module makes to its own backend.
	dest := strings.TrimSpace(strings.ToLower(r.Header.Get("Sec-Fetch-Dest")))
	if dest != "" {
		return dest != "document"
	}

	// No Sec-Fetch-Dest (older browsers): fall back to best-effort Referer
	// matching.
	if p.isRefererFromParent(raw) {
		return true
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	if moduleHost != "" && host == moduleHost {
		return true
	}
	return false
}

func (p *proxyService) redirectToLoginIfPossible(w http.ResponseWriter, r *http.Request, referer string) bool {
	if p.loginURL == "" {
		return false
	}
	if r.Method != http.MethodGet {
		return false
	}
	if p.shouldSkipLoginRedirectReferer(referer, r) {
		return false
	}
	if !acceptsHTML(r) {
		return false
	}
	loginURL, err := url.Parse(p.loginURL)
	if err != nil || loginURL.Scheme == "" || loginURL.Host == "" {
		return false
	}
	next := buildOriginalURL(r)
	values := loginURL.Query()
	if next != "" {
		values.Set("next", next)
	}
	loginURL.RawQuery = values.Encode()
	http.Redirect(w, r, loginURL.String(), http.StatusFound)
	return true
}

// statusPageTemplate renders a dark, dotted-grid backdrop with a centered
// card (icon tile, status pill, heading, description) — the same visual
// language used by the frontend's ModuleStatusCard component, for the
// server-rendered pages this proxy has to serve on its own (e.g. before any
// React app has loaded, or for direct browser navigation to a module host).
const statusPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>%s</title>
<style>
  html, body { height: 100%%; margin: 0; }
  body {
    display: flex; align-items: center; justify-content: center;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background-color: #141414;
    background-image:
      linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
      linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
    background-size: 36px 36px;
    color: #e6e6e6;
    padding: 24px;
    box-sizing: border-box;
  }
  .card {
    width: 100%%;
    max-width: 420px;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 40px 32px;
    border-radius: 16px;
    background-color: #252525;
    border: 1px solid #333;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.25);
    box-sizing: border-box;
  }
  .icon {
    width: 56px;
    height: 56px;
    border-radius: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 20px;
    background: rgba(%s, 0.14);
    color: rgb(%s);
  }
  .icon svg { width: 26px; height: 26px; }
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 12px;
    margin-bottom: 20px;
    border-radius: 999px;
    font-size: 11px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    background: rgba(255, 255, 255, 0.06);
    color: rgb(%s);
  }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: currentColor; }
  h1 { font-size: 22px; font-weight: 700; line-height: 1.3; margin: 0 0 12px; color: #fff; }
  p { font-size: 14px; line-height: 1.6; color: #9aa0a6; margin: 0; }
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-top: 22px;
    padding: 10px 20px;
    border-radius: 10px;
    background: #3b82f6;
    color: #fff;
    font-size: 14px;
    font-weight: 600;
    text-decoration: none;
  }
  .btn:hover { background: #2563eb; }
</style>
</head>
<body>
  <div class="card">
    <div class="icon">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">%s</svg>
    </div>
    <div class="badge"><span class="dot"></span>%s</div>
    <h1>%s</h1>
    <p>%s</p>
    %s
  </div>
</body>
</html>`

// SVG path data (Feather-style icons) shared with the frontend's ModuleStatusCard.
const (
	iconSVGWrench = `<path d="M14.7 6.3a4 4 0 0 0-5.4 5.4L3 18l3 3 6.3-6.3a4 4 0 0 0 5.4-5.4l-2.1 2.1a2 2 0 0 1-2.8-2.8l2.1-2.1Z" />`
	iconSVGLock   = `<rect x="3" y="11" width="18" height="10" rx="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" />`
	iconSVGFrame  = `<rect x="3" y="4" width="18" height="16" rx="2" /><path d="M3 9h18" />`
)

// statusAccent bundles the RGB triplets used for an accent color's icon/badge tint.
type statusAccent struct {
	rgb string // e.g. "59, 130, 246"
}

var (
	accentBlue = statusAccent{rgb: "59, 130, 246"}
	accentRed  = statusAccent{rgb: "239, 68, 68"}
)

// actionButtonHTML renders an optional call-to-action button for the status
// page, or an empty string when no href is available.
func actionButtonHTML(label, href string) string {
	if strings.TrimSpace(href) == "" {
		return ""
	}
	return fmt.Sprintf(`<a class="btn" href="%s">%s</a>`, html.EscapeString(href), html.EscapeString(label))
}

// renderStatusPage fills the shared dark-card template.
func renderStatusPage(title string, accent statusAccent, iconSVG, badge, heading, description, actionHTML string) string {
	return fmt.Sprintf(statusPageTemplate,
		html.EscapeString(title),
		accent.rgb, accent.rgb, accent.rgb,
		iconSVG,
		html.EscapeString(badge),
		html.EscapeString(heading),
		html.EscapeString(description),
		actionHTML,
	)
}

// writeStatusPage writes the rendered card as the full HTML response.
func writeStatusPage(w http.ResponseWriter, status int, accent statusAccent, iconSVG, badge, heading, description, actionHTML string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(renderStatusPage(heading, accent, iconSVG, badge, heading, description, actionHTML)))
}

// writeErrorResponse renders the styled status-card page for browser
// navigations, or falls back to a plain JSON error for API/fetch clients.
func writeErrorResponse(w http.ResponseWriter, r *http.Request, status int, code string, accent statusAccent, iconSVG, badge, heading, description, actionHTML string) {
	if !acceptsHTML(r) {
		writeJSONError(w, status, code, description)
		return
	}
	writeStatusPage(w, status, accent, iconSVG, badge, heading, description, actionHTML)
}

// panBagnatModuleURL builds a link back to the module's page inside the main
// Pan Bagnat app (https://{host}/modules/{slug}), derived from the same host
// used to build the login URL. Returns "" when that host isn't configured.
func (p *proxyService) panBagnatModuleURL(slug string) string {
	base, err := url.Parse(p.loginURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return ""
	}
	base.Path = "/modules/" + url.PathEscape(slug)
	base.RawQuery = ""
	base.Fragment = ""
	return base.String()
}

// writeModuleDisabledResponse renders a maintenance page (or JSON error for
// non-browser clients) instead of proxying to a disabled module's container.
func writeModuleDisabledResponse(w http.ResponseWriter, r *http.Request) {
	writeErrorResponse(w, r, http.StatusServiceUnavailable, "module_disabled", accentBlue, iconSVGWrench,
		"Module status: disabled",
		"This module is currently disabled by an administrator.",
		"If you were expecting to use it, let them know and they may turn it back on.",
		"")
}

func acceptsHTML(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	if accept == "" {
		return true
	}
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}

func buildOriginalURL(r *http.Request) string {
	scheme := "https"
	if !isHTTPS(r) {
		scheme = "http"
	}
	host := r.Host
	if host == "" {
		return ""
	}
	u := &url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}
	return u.String()
}

func hostWithoutPort(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, ":"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.ToLower(raw)
}

func (p *proxyService) shouldSkipLoginRedirectReferer(referer string, r *http.Request) bool {
	referer = strings.TrimSpace(referer)
	if referer == "" {
		return false
	}
	if p.isRefererFromParent(referer) && isIframeFetch(r) {
		return true
	}
	original := buildOriginalURL(r)
	if original != "" && strings.EqualFold(referer, original) {
		return true
	}
	return false
}

func parseModuleAccessToken(raw string, secret []byte) (moduleAccessClaims, error) {
	var claims moduleAccessClaims
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return claims, errInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, errInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, errInvalidToken
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return claims, errInvalidToken
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, errInvalidToken
	}
	if claims.SessionID == "" || claims.Slug == "" {
		return claims, errInvalidToken
	}
	if time.Now().After(time.Unix(claims.ExpiresAt, 0)) {
		return claims, errTokenExpired
	}
	return claims, nil
}

func (p *proxyService) setModuleSessionCookie(w http.ResponseWriter, sessionID string, secure bool) {
	if sessionID == "" {
		return
	}
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure || sessionCookieSameSite == http.SameSiteNoneMode,
		SameSite: sessionCookieSameSite,
	}
	if p.sessionCookieTTL > 0 {
		cookie.MaxAge = int(p.sessionCookieTTL.Seconds())
		cookie.Expires = time.Now().Add(p.sessionCookieTTL)
	}
	http.SetCookie(w, cookie)
}

func (p *proxyService) authenticateRequest(ctx context.Context, r *http.Request) (*sessionUser, error) {
	sid := readSessionID(r)
	if sid == "" {
		return nil, errNoSession
	}

	const query = `
        SELECT u.id, u.ft_login, COALESCE(u.last_seen, NOW()), s.expires_at
          FROM sessions s
          JOIN users u ON u.ft_login = s.ft_login
         WHERE s.session_id = $1
    `

	var info sessionUser
	if err := p.db.QueryRowContext(ctx, query, sid).Scan(&info.ID, &info.FtLogin, &info.LastSeen, &info.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errNoSession
		}
		return nil, err
	}
	if time.Now().After(info.ExpiresAt) {
		return nil, errSessionExpired
	}
	info.SessionID = sid
	return &info, nil
}

func (p *proxyService) isUserBlacklisted(ctx context.Context, userID string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = 'roles_blacklist')`
	var exists bool
	if err := p.db.QueryRowContext(ctx, query, userID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (p *proxyService) deleteUserSessions(ctx context.Context, userID string) error {
	const query = `DELETE FROM sessions WHERE ft_login = (SELECT ft_login FROM users WHERE id = $1)`
	_, err := p.db.ExecContext(ctx, query, userID)
	return err
}

func buildFrameAncestorsDirective(hosts map[string]struct{}) string {
	if len(hosts) == 0 {
		return ""
	}
	values := []string{"'self'"}
	for host := range hosts {
		h := strings.TrimSpace(host)
		if h == "" {
			continue
		}
		if strings.HasPrefix(h, "http://") || strings.HasPrefix(h, "https://") {
			values = append(values, h)
			continue
		}
		values = append(values, "https://"+h, "http://"+h)
	}
	return strings.Join(values, " ")
}

func newReverseProxy(target *url.URL, frameAncestors string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	if frameAncestors != "" {
		policyValue := fmt.Sprintf("frame-ancestors %s", frameAncestors)
		proxy.ModifyResponse = func(resp *http.Response) error {
			resp.Header.Add("Content-Security-Policy", policyValue)
			return nil
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[proxy-service] proxy error for %s: %v", target.Host, err)
		http.Error(w, "module upstream error", http.StatusBadGateway)
	}
	return proxy
}

func readSessionID(r *http.Request) string {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if v := r.Header.Get("X-Session-Id"); v != "" {
		return v
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	if code != "" {
		w.Header().Set("X-Error-Code", code)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   http.StatusText(status),
		"code":    code,
		"message": message,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sessionCookieSameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func isIframeFetch(r *http.Request) bool {
	dest := strings.TrimSpace(strings.ToLower(r.Header.Get("Sec-Fetch-Dest")))
	if dest == "" {
		return false
	}
	return dest == "iframe" || dest == "frame"
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[proxy-service] failed to write json response: %v", err)
	}
}

func loadConfig() (config, error) {
	pg := strings.TrimSpace(os.Getenv("POSTGRES_URL"))
	if pg == "" {
		return config{}, errors.New("POSTGRES_URL is required")
	}
	hostName := strings.ToLower(strings.TrimSpace(os.Getenv("HOST_NAME")))
	port := strings.TrimSpace(os.Getenv("MODULES_PROXY_PORT"))
	if port == "" {
		port = "9090"
	}
	channel := strings.TrimSpace(os.Getenv("MODULES_PROXY_CHANNEL"))
	if channel == "" {
		channel = "module_page_changed"
	}
	rawDomains := strings.Split(os.Getenv("MODULES_PROXY_ALLOWED_DOMAINS"), ",")
	domains := make([]string, 0, len(rawDomains))
	for _, d := range rawDomains {
		trimmed := strings.ToLower(strings.TrimSpace(d))
		if trimmed != "" {
			domains = append(domains, trimmed)
		}
	}
	if len(domains) == 0 {
		if hostName == "" {
			return config{}, errors.New("MODULES_PROXY_ALLOWED_DOMAINS or HOST_NAME is required")
		}
		domains = []string{fmt.Sprintf("modules.%s", hostName)}
	}
	rawIframe := strings.TrimSpace(os.Getenv("MODULES_IFRAME_ALLOWED_HOSTS"))
	if rawIframe == "" {
		rawIframe = hostName
	}
	rawIframeHosts := strings.Split(rawIframe, ",")
	iframeHosts := make([]string, 0, len(rawIframeHosts))
	for _, h := range rawIframeHosts {
		trimmed := strings.ToLower(strings.TrimSpace(h))
		if trimmed != "" {
			iframeHosts = append(iframeHosts, trimmed)
		}
	}
	netURL := strings.TrimSpace(os.Getenv("MODULES_NET_CONTROLLER_URL"))
	if netURL == "" {
		netURL = "http://net-controller:9091"
	}
	loginURL := strings.TrimSpace(os.Getenv("MODULES_LOGIN_URL"))
	if loginURL == "" && hostName != "" {
		loginURL = fmt.Sprintf("https://%s/login", hostName)
	}

	gatewayPort := defaultGatewayPort
	if raw := strings.TrimSpace(os.Getenv("MODULES_GATEWAY_PORT")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			gatewayPort = v
		}
	}

	secret := strings.TrimSpace(os.Getenv("MODULES_SESSION_SECRET"))
	if secret == "" {
		return config{}, errors.New("MODULES_SESSION_SECRET is required")
	}
	cookieTTL := time.Hour
	if raw := strings.TrimSpace(os.Getenv("MODULES_SESSION_COOKIE_TTL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cookieTTL = d
		}
	}

	return config{
		PostgresURL:        pg,
		ListenAddr:         ":" + port,
		Channel:            channel,
		AllowedDomains:     domains,
		IframeAllowedHosts: iframeHosts,
		NetControllerURL:   netURL,
		GatewayPort:        gatewayPort,
		SessionSecret:      []byte(secret),
		SessionCookieTTL:   cookieTTL,
		LoginURL:           loginURL,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := sqlx.Connect("postgres", cfg.PostgresURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	netClient := newNetControllerClient(cfg.NetControllerURL)

	service := newProxyService(db, cfg.PostgresURL, cfg.Channel, cfg.AllowedDomains, cfg.IframeAllowedHosts, netClient, cfg.GatewayPort, cfg.SessionSecret, cfg.SessionCookieTTL, cfg.LoginURL)
	if err := service.refreshPages(ctx); err != nil {
		log.Fatalf("failed to load module pages: %v", err)
	}
	go service.listenForChanges(ctx)

	router := chi.NewRouter()
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	router.NotFound(service.handleGatewayRequest)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		// ReadTimeout/WriteTimeout are intentionally unset: Go sets these as a
		// one-shot deadline on the raw connection when a request starts and
		// never renews them. httputil.ReverseProxy hijacks that same
		// connection for WebSocket upgrades without clearing the deadline, so
		// any nonzero value here force-closes every proxied WebSocket at that
		// fixed age regardless of activity. ReadHeaderTimeout already guards
		// against slow-header attacks; per-request body/response timeouts are
		// enforced downstream by the module gateway's proxy_read_timeout.
		IdleTimeout: 60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[proxy-service] graceful shutdown failed: %v", err)
		}
		_ = db.Close()
	}()

	log.Printf("[proxy-service] listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped unexpectedly: %v", err)
	}
}
