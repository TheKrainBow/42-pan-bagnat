package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
)

const sessionCookieName = "session_id"

var (
	errNoSession      = errors.New("no session")
	errSessionExpired = errors.New("session expired")
)

type config struct {
	PostgresURL string
	ListenAddr  string
	Channel     string
	ContainerID string
}

type cachedPage struct {
	Slug       string
	URL        string
	ModuleID   string
	ModuleSlug string
	IsPublic   bool
	Target     *url.URL
	Network    string
}

type proxyStatusPayload struct {
	OK                bool     `json:"ok"`
	Message           string   `json:"message,omitempty"`
	Network           string   `json:"network_name,omitempty"`
	ConnectedNetworks []string `json:"connected_networks,omitempty"`
	MissingNetworks   []string `json:"missing_networks,omitempty"`
	Reattached        bool     `json:"reattached,omitempty"`
}

type sessionUser struct {
	ID        string
	FtLogin   string
	LastSeen  time.Time
	ExpiresAt time.Time
	SessionID string
}

type dockerManager struct {
	client      *http.Client
	containerID string
}

func newDockerManager(containerName string) (*dockerManager, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.DialTimeout("unix", "/var/run/docker.sock", 5*time.Second)
		},
	}
	cli := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	dm := &dockerManager{client: cli}
	id, err := dm.lookupContainerID(context.Background(), containerName)
	if err != nil {
		return nil, err
	}
	dm.containerID = id
	return dm, nil
}

func (d *dockerManager) Close() {}

func (d *dockerManager) lookupContainerID(ctx context.Context, name string) (string, error) {
	resp, err := d.doRequest(ctx, http.MethodGet, fmt.Sprintf("/containers/%s/json", name), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("inspect container failed: %s", strings.TrimSpace(string(body)))
	}
	var payload struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.ID, nil
}

func (d *dockerManager) ensureNetworks(ctx context.Context, required map[string]map[string]struct{}) error {
	if d == nil {
		return nil
	}
	for slug, wanted := range required {
		if slug == "" {
			continue
		}
		networks, err := d.listComposeNetworks(ctx, slug)
		if err != nil {
			log.Printf("[modules-proxy] list networks for %s failed: %v", slug, err)
			continue
		}
		for _, netInfo := range networks {
			_, need := wanted[netInfo.Name]
			if need {
				if netInfo.isConnected(d.containerID) {
					continue
				}
				log.Printf("[modules-proxy] connecting to network %s", netInfo.Name)
				if err := d.connect(ctx, netInfo.ID); err != nil {
					log.Printf("[modules-proxy] connect to %s failed: %v", netInfo.Name, err)
				}
			} else {
				if netInfo.isConnected(d.containerID) {
					log.Printf("[modules-proxy] detaching from unused network %s", netInfo.Name)
					if err := d.disconnect(ctx, netInfo.ID); err != nil {
						log.Printf("[modules-proxy] disconnect from %s failed: %v", netInfo.Name, err)
					}
				}
			}
		}
	}
	return nil
}

func (d *dockerManager) listComposeNetworks(ctx context.Context, slug string) ([]dockerNetwork, error) {
	filters := map[string][]string{
		"label": {"com.docker.compose.project=" + slug},
	}
	filterJSON, _ := json.Marshal(filters)
	path := fmt.Sprintf("/networks?filters=%s", url.QueryEscape(string(filterJSON)))
	resp, err := d.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list networks failed: %s", strings.TrimSpace(string(body)))
	}
	var nets []dockerNetwork
	if err := json.NewDecoder(resp.Body).Decode(&nets); err != nil {
		return nil, err
	}
	detailed := make([]dockerNetwork, 0, len(nets))
	for _, netInfo := range nets {
		if netInfo.ID == "" {
			continue
		}
		inspected, err := d.inspectNetwork(ctx, netInfo.ID)
		if err != nil {
			log.Printf("[modules-proxy] inspect network %s failed: %v", netInfo.Name, err)
			continue
		}
		detailed = append(detailed, inspected)
	}
	return detailed, nil
}

func (d *dockerManager) inspectNetwork(ctx context.Context, networkID string) (dockerNetwork, error) {
	resp, err := d.doRequest(ctx, http.MethodGet, fmt.Sprintf("/networks/%s", networkID), nil)
	if err != nil {
		return dockerNetwork{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return dockerNetwork{}, fmt.Errorf("inspect network failed: %s", strings.TrimSpace(string(body)))
	}
	var netInfo dockerNetwork
	if err := json.NewDecoder(resp.Body).Decode(&netInfo); err != nil {
		return dockerNetwork{}, err
	}
	return netInfo, nil
}

func (d *dockerManager) connect(ctx context.Context, networkID string) error {
	payload := map[string]string{"Container": d.containerID}
	body, _ := json.Marshal(payload)
	resp, err := d.doRequest(ctx, http.MethodPost, fmt.Sprintf("/networks/%s/connect", networkID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("network connect failed: %s", strings.TrimSpace(string(msg)))
	}
	return nil
}

func (d *dockerManager) disconnect(ctx context.Context, networkID string) error {
	payload := map[string]string{"Container": d.containerID}
	body, _ := json.Marshal(payload)
	resp, err := d.doRequest(ctx, http.MethodPost, fmt.Sprintf("/networks/%s/disconnect", networkID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("network disconnect failed: %s", strings.TrimSpace(string(msg)))
	}
	return nil
}

func (d *dockerManager) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return d.client.Do(req)
}

type dockerNetwork struct {
	ID         string                            `json:"Id"`
	Name       string                            `json:"Name"`
	Containers map[string]dockerNetworkContainer `json:"Containers"`
}

type dockerNetworkContainer struct{}

func (n dockerNetwork) isConnected(containerID string) bool {
	for id := range n.Containers {
		if strings.HasPrefix(containerID, id) || strings.HasPrefix(id, containerID) {
			return true
		}
	}
	return false
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

type proxyService struct {
	db          *sqlx.DB
	connInfo    string
	store       *pageStore
	channelName string
	docker      *dockerManager
	knownSlugs  map[string]struct{}
}

func newProxyService(db *sqlx.DB, connInfo, channel string, dockerMgr *dockerManager) *proxyService {
	return &proxyService{
		db:          db,
		connInfo:    connInfo,
		store:       newPageStore(),
		channelName: channel,
		docker:      dockerMgr,
		knownSlugs:  make(map[string]struct{}),
	}
}

func (p *proxyService) refreshPages(ctx context.Context) error {
	pages, required, slugs, err := p.fetchPages(ctx)
	if err != nil {
		return err
	}
	p.store.replace(pages)
	log.Printf("[modules-proxy] loaded %d module pages", len(pages))
	if p.docker != nil {
		if required == nil {
			required = make(map[string]map[string]struct{})
		}
		for slug := range p.knownSlugs {
			if _, ok := required[slug]; !ok {
				required[slug] = make(map[string]struct{})
			}
		}
		if err := p.docker.ensureNetworks(ctx, required); err != nil {
			log.Printf("[modules-proxy] failed to ensure networks: %v", err)
		}
	}
	p.knownSlugs = slugs
	return nil
}

func (p *proxyService) fetchPages(ctx context.Context) ([]cachedPage, map[string]map[string]struct{}, map[string]struct{}, error) {
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
		return nil, nil, nil, err
	}
	cached := make([]cachedPage, 0, len(rows))
	required := make(map[string]map[string]struct{})
	slugSet := make(map[string]struct{})
	for _, r := range rows {
		u, err := url.Parse(r.URL)
		if err != nil {
			log.Printf("[modules-proxy] skipping page %s: invalid URL %q (%v)", r.Slug, r.URL, err)
			continue
		}
		slugSet[r.ModuleSlug] = struct{}{}
		if _, ok := required[r.ModuleSlug]; !ok {
			required[r.ModuleSlug] = make(map[string]struct{})
		}
		networkName := strings.TrimSpace(r.NetworkName)
		if networkName != "" {
			required[r.ModuleSlug][networkName] = struct{}{}
		}
		cached = append(cached, cachedPage{
			Slug:       r.Slug,
			URL:        r.URL,
			ModuleID:   r.ModuleID,
			ModuleSlug: r.ModuleSlug,
			IsPublic:   r.IsPublic,
			Target:     u,
			Network:    networkName,
		})
	}
	return cached, required, slugSet, nil
}

func (p *proxyService) listenForChanges(ctx context.Context) {
	report := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[modules-proxy] listener event=%v error=%v", ev, err)
		}
	}
	listener := pq.NewListener(p.connInfo, 5*time.Second, time.Minute, report)
	if err := listener.Listen(p.channelName); err != nil {
		log.Printf("[modules-proxy] failed to LISTEN %s: %v", p.channelName, err)
		return
	}
	log.Printf("[modules-proxy] listening for notifications on channel %q", p.channelName)

	for {
		select {
		case <-ctx.Done():
			listener.Close()
			return
		case n := <-listener.Notify:
			if n == nil {
				continue
			}
			log.Printf("[modules-proxy] received notification %s", n.Channel)
			if err := p.refreshPages(context.Background()); err != nil {
				log.Printf("[modules-proxy] failed to refresh pages: %v", err)
			}
		case <-time.After(2 * time.Minute):
			go func() {
				if err := listener.Ping(); err != nil {
					log.Printf("[modules-proxy] listener ping failed: %v", err)
				}
			}()
		}
	}
}

func (p *proxyService) handleModulePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "missing module slug", http.StatusBadRequest)
		return
	}
	page, ok := p.store.get(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if _, ok := p.authorizePageRequest(w, r, page); !ok {
		return
	}

	prefix := "/module-page/" + slug
	suffix := strings.TrimPrefix(r.URL.Path, prefix)
	if suffix == "" {
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, prefix+"/", http.StatusTemporaryRedirect)
			return
		}
		suffix = "/"
	}

	originalPath, originalRaw := r.URL.Path, r.URL.RawPath
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	r.URL.Path = joinURLPath(suffix)
	r.URL.RawPath = r.URL.Path

	log.Printf("[modules-proxy] connection on %s redirected to %s%s", slug, page.Target.Host, r.URL.Path)

	proxy := newReverseProxy(page.Target)
	proxy.ServeHTTP(w, r)

	r.URL.Path, r.URL.RawPath = originalPath, originalRaw
}

func (p *proxyService) handleModulePageStatus(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "missing module slug", http.StatusBadRequest)
		return
	}
	page, ok := p.store.get(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, ok := p.authorizePageRequest(w, r, page); !ok {
		return
	}
	resp := proxyStatusPayload{
		Network: strings.TrimSpace(page.Network),
	}
	if p.docker == nil {
		resp.Message = "docker integration disabled"
		respondJSON(w, resp)
		return
	}
	if r.Method == http.MethodPost {
		if resp.Network == "" {
			resp.Message = "no network selected"
			respondJSON(w, resp)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		payload := map[string]map[string]struct{}{
			page.ModuleSlug: {resp.Network: {}},
		}
		if err := p.docker.ensureNetworks(ctx, payload); err != nil {
			log.Printf("[modules-proxy] manual attach failed for %s: %v", resp.Network, err)
			writeJSONError(w, http.StatusInternalServerError, "docker_error", "Failed to attach network")
			return
		}
		resp.Reattached = true
		respondJSON(w, resp)
		return
	}
	if resp.Network == "" {
		resp.Message = "no network selected"
		respondJSON(w, resp)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	nets, err := p.docker.listComposeNetworks(ctx, page.ModuleSlug)
	if err != nil {
		log.Printf("[modules-proxy] status check failed for %s: %v", slug, err)
		writeJSONError(w, http.StatusInternalServerError, "docker_error", "Failed to inspect networks")
		return
	}
	if len(nets) == 0 {
		resp.Message = "no compose networks found"
		respondJSON(w, resp)
		return
	}
	var target *dockerNetwork
	for i := range nets {
		if nets[i].Name == resp.Network {
			target = &nets[i]
			break
		}
	}
	if target == nil {
		resp.Message = fmt.Sprintf("network %s not found", resp.Network)
		resp.MissingNetworks = []string{resp.Network}
		respondJSON(w, resp)
		return
	}
	if target.isConnected(p.docker.containerID) {
		resp.OK = true
		resp.Message = "reverse proxy attached"
		resp.ConnectedNetworks = []string{resp.Network}
	} else {
		resp.Message = fmt.Sprintf("network %s missing", resp.Network)
		resp.MissingNetworks = []string{resp.Network}
	}
	respondJSON(w, resp)
}

func (p *proxyService) authorizePageRequest(w http.ResponseWriter, r *http.Request, page cachedPage) (*sessionUser, bool) {
	ctx := r.Context()
	user, err := p.authenticateRequest(ctx, r)
	switch {
	case err == errNoSession:
		user = nil
	case err == errSessionExpired:
		clearSessionCookie(w, isHTTPS(r))
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Session expired. Please sign in again.")
		return nil, false
	case err != nil:
		log.Printf("[modules-proxy] auth failure: %v", err)
		http.Error(w, "internal authentication error", http.StatusInternalServerError)
		return nil, false
	}

	if user != nil {
		blacklisted, err := p.isUserBlacklisted(ctx, user.ID)
		if err != nil {
			log.Printf("[modules-proxy] blacklist check failed for user %s: %v", user.ID, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return nil, false
		}
		if blacklisted {
			_ = p.deleteUserSessions(ctx, user.ID)
			clearSessionCookie(w, isHTTPS(r))
			writeJSONError(w, http.StatusForbidden, "blacklisted", "Your account is currently blacklisted. Contact your bocal.")
			return nil, false
		}
	}

	if !page.IsPublic {
		if r.Header.Get("Referer") == "" {
			writeJSONError(w, http.StatusForbidden, "iframe_required", "This page must be loaded from Pan Bagnat.")
			return nil, false
		}
		if user == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Please sign in.")
			return nil, false
		}
	}

	return user, true
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

func joinURLPath(pth string) string {
	cleaned := path.Clean(pth)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if pth == "/" {
		return "/"
	}
	if strings.HasSuffix(pth, "/") && !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}

func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[modules-proxy] proxy error for %s: %v", target.Host, err)
		http.Error(w, "module upstream error", http.StatusBadGateway)
	}
	return proxy
}

func loadConfig() (config, error) {
	pg := os.Getenv("POSTGRES_URL")
	if strings.TrimSpace(pg) == "" {
		return config{}, errors.New("POSTGRES_URL is required")
	}
	port := os.Getenv("MODULES_PROXY_PORT")
	if port == "" {
		port = "9090"
	}
	channel := os.Getenv("MODULES_PROXY_CHANNEL")
	if channel == "" {
		channel = "module_page_changed"
	}
	container := os.Getenv("MODULES_PROXY_CONTAINER")
	if container == "" {
		container = os.Getenv("HOSTNAME")
	}
	return config{
		PostgresURL: pg,
		ListenAddr:  ":" + port,
		Channel:     channel,
		ContainerID: container,
	}, nil
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
		log.Printf("[modules-proxy] failed to write json response: %v", err)
	}
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
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	var dockerMgr *dockerManager
	if cfg.ContainerID != "" {
		dockerMgr, err = newDockerManager(cfg.ContainerID)
		if err != nil {
			log.Printf("[modules-proxy] docker integration disabled: %v", err)
		}
	}

	service := newProxyService(db, cfg.PostgresURL, cfg.Channel, dockerMgr)
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
	router.HandleFunc("/module-page/{slug}", service.handleModulePage)
	router.HandleFunc("/module-page/{slug}/*", service.handleModulePage)

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
			log.Printf("[modules-proxy] graceful shutdown failed: %v", err)
		}
		_ = db.Close()
		if dockerMgr != nil {
			dockerMgr.Close()
		}
	}()

	log.Printf("[modules-proxy] listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped unexpectedly: %v", err)
	}
}
