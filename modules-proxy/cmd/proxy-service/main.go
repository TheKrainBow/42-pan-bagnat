package main

import (
	"context"
	"database/sql"
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
)

var (
	errNoSession      = errors.New("no session")
	errSessionExpired = errors.New("session expired")
)

type config struct {
	PostgresURL      string
	ListenAddr       string
	Channel          string
	AllowedDomains   []string
	NetControllerURL string
	GatewayPort      int
}

type cachedPage struct {
	Slug       string
	URL        string
	ModuleID   string
	ModuleSlug string
	IsPublic   bool
	Network    string
}

type sessionUser struct {
	ID        string
	FtLogin   string
	LastSeen  time.Time
	ExpiresAt time.Time
	SessionID string
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
	db            *sqlx.DB
	store         *pageStore
	connInfo      string
	channelName   string
	allowedSuffix []string
	netClient     *netControllerClient
	gatewayPort   int
}

func newProxyService(db *sqlx.DB, connInfo, channel string, suffixes []string, netClient *netControllerClient, gatewayPort int) *proxyService {
	return &proxyService{
		db:            db,
		store:         newPageStore(),
		connInfo:      connInfo,
		channelName:   channel,
		allowedSuffix: suffixes,
		netClient:     netClient,
		gatewayPort:   gatewayPort,
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
		       mp.url,
		       mp.module_id,
		       mp.is_public,
		       m.slug AS module_slug,
		       COALESCE(mp.network_name, '') AS network_name
		FROM module_page mp
		JOIN modules m ON m.id = mp.module_id
	`
	type row struct {
		Slug        string `db:"slug"`
		URL         string `db:"url"`
		ModuleID    string `db:"module_id"`
		IsPublic    bool   `db:"is_public"`
		ModuleSlug  string `db:"module_slug"`
		NetworkName string `db:"network_name"`
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
			Slug:       slug,
			URL:        r.URL,
			ModuleID:   r.ModuleID,
			ModuleSlug: r.ModuleSlug,
			IsPublic:   r.IsPublic,
			Network:    strings.TrimSpace(r.NetworkName),
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
	ctx := r.Context()
	user, err := p.authenticateRequest(ctx, r)
	switch {
	case err == errNoSession:
		user = nil
	case err == errSessionExpired:
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

	if !page.IsPublic {
		if r.Header.Get("Referer") == "" {
			writeJSONError(w, http.StatusForbidden, "iframe_required", "This page must be loaded from Pan Bagnat.")
			return nil, false, "missing iframe referer"
		}
		if user == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Please sign in.")
			return nil, false, "not authenticated"
		}
	}
	return user, true, ""
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
		SameSite: http.SameSiteLaxMode,
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
	netURL := strings.TrimSpace(os.Getenv("MODULES_NET_CONTROLLER_URL"))

	gatewayPort := defaultGatewayPort
	if raw := strings.TrimSpace(os.Getenv("MODULES_GATEWAY_PORT")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			gatewayPort = v
		}
	}

	return config{
		PostgresURL:      pg,
		ListenAddr:       ":" + port,
		Channel:          channel,
		AllowedDomains:   domains,
		NetControllerURL: netURL,
		GatewayPort:      gatewayPort,
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

	service := newProxyService(db, cfg.PostgresURL, cfg.Channel, cfg.AllowedDomains, netClient, cfg.GatewayPort)
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
