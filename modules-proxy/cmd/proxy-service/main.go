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
	sessionCookieSameSite = func() http.SameSite {
		raw := strings.TrimSpace(strings.ToLower(os.Getenv("SESSION_COOKIE_SAMESITE")))
		switch raw {
		case "strict":
			return http.SameSiteStrictMode
		case "lax":
			return http.SameSiteLaxMode
		case "none", "":
			return http.SameSiteNoneMode
		default:
			return http.SameSiteNoneMode
		}
	}()
	authDebugEnabled = strings.EqualFold(os.Getenv("PROXY_DEBUG_AUTH"), "1")
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
}

type cachedPage struct {
	Slug            string
	ModuleID        string
	ModuleSlug      string
	IframeOnly      bool
	NeedAuth        bool
	Network         string
	TargetContainer string
	TargetPort      int
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
	netClient        *netControllerClient
	gatewayPort      int
	sessionSecret    []byte
	sessionCookieTTL time.Duration
}

func newProxyService(db *sqlx.DB, connInfo, channel string, suffixes []string, iframeHosts []string, netClient *netControllerClient, gatewayPort int, sessionSecret []byte, cookieTTL time.Duration) *proxyService {
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
		netClient:        netClient,
		gatewayPort:      gatewayPort,
		sessionSecret:    sessionSecret,
		sessionCookieTTL: cookieTTL,
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
		       m.slug AS module_slug,
		       COALESCE(mp.network_name, '') AS network_name
		FROM module_page mp
		JOIN modules m ON m.id = mp.module_id
        WHERE mp.target_container IS NOT NULL
          AND mp.target_port IS NOT NULL
	`
	type row struct {
		Slug            string `db:"slug"`
		TargetContainer string `db:"target_container"`
		TargetPort      int    `db:"target_port"`
		ModuleID        string `db:"module_id"`
		IframeOnly      bool   `db:"iframe_only"`
		NeedAuth        bool   `db:"need_auth"`
		ModuleSlug      string `db:"module_slug"`
		NetworkName     string `db:"network_name"`
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
			Slug:            slug,
			ModuleID:        r.ModuleID,
			ModuleSlug:      r.ModuleSlug,
			IframeOnly:      r.IframeOnly,
			NeedAuth:        r.NeedAuth,
			Network:         strings.TrimSpace(r.NetworkName),
			TargetContainer: strings.TrimSpace(r.TargetContainer),
			TargetPort:      r.TargetPort,
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
	if r.URL.Path == moduleSessionPath {
		p.handleSessionBootstrap(w, r, slug)
		return
	}
	if _, allowed, reason := p.authorizePageRequest(w, r, page); !allowed {
		log.Printf("[proxy-service] denied slug=%q host=%q referer=%q reason=%s", slug, r.Host, r.Referer(), reason)
		return
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

	proxy := newReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

func (p *proxyService) handleModulePageStatus(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "missing module slug", http.StatusBadRequest)
		return
	}
	page, ok := p.store.get(slug)
	if !ok {
		log.Printf("[proxy-service] status request for unknown slug=%q", slug)
		http.NotFound(w, r)
		return
	}
	if _, allowed, reason := p.authorizePageRequest(w, r, page); !allowed {
		log.Printf("[proxy-service] denied status slug=%q host=%q reason=%s", slug, r.Host, reason)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var (
		payload types.ProxyStatusPayload
		err     error
	)
	switch r.Method {
	case http.MethodPost:
		payload, err = p.netClient.Reconcile(ctx, slug)
	default:
		payload, err = p.netClient.Status(ctx, slug)
	}
	if err != nil {
		log.Printf("[proxy-service] status fetch failed for %s: %v", slug, err)
		writeJSONError(w, http.StatusBadGateway, "net_controller_unavailable", "Unable to fetch gateway status")
		return
	}
	payload.Network = page.Network
	respondJSON(w, payload)
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
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Session expired. Please sign in again.")
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
			writeJSONError(w, http.StatusForbidden, "blacklisted", "Your account is currently blacklisted. Contact your bocal.")
			return nil, false, "user is blacklisted"
		}
	}

	if page.IframeOnly && !p.isRefererAllowed(r.Header.Get("Referer")) {
		writeJSONError(w, http.StatusForbidden, "iframe_required", "This page must be loaded from Pan Bagnat.")
		return nil, false, "missing iframe referer"
	}
	if page.NeedAuth && user == nil {
		if authDebugEnabled {
			log.Printf("[proxy-service][auth-debug] user nil but referer=%q need_auth=%v", r.Header.Get("Referer"), page.NeedAuth)
		}
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Please sign in.")
		return nil, false, "not authenticated"
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

func (p *proxyService) isRefererAllowed(raw string) bool {
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

func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
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
		return config{}, errors.New("MODULES_PROXY_ALLOWED_DOMAINS is required")
	}
	rawIframe := strings.TrimSpace(os.Getenv("MODULES_IFRAME_ALLOWED_HOSTS"))
	if rawIframe == "" {
		rawIframe = "localhost,panbagnat.42nice.fr"
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

	service := newProxyService(db, cfg.PostgresURL, cfg.Channel, cfg.AllowedDomains, cfg.IframeAllowedHosts, netClient, cfg.GatewayPort, cfg.SessionSecret, cfg.SessionCookieTTL)
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
	router.Get("/module-page/_status/{slug}", service.handleModulePageStatus)
	router.Post("/module-page/_status/{slug}", service.handleModulePageStatus)
	router.NotFound(service.handleGatewayRequest)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
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
