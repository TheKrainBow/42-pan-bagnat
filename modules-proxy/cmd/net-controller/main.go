package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"modules-proxy/internal/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	docker "github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
)

const (
	gatewayLabel        = "com.panbagnat.gateway"
	gatewaySlugLabel    = "com.panbagnat.gateway.slug"
	gatewayTargetLabel  = "com.panbagnat.gateway.target"
	gatewayNetworkLabel = "com.panbagnat.gateway.network"
	defaultGatewayPort  = 8080
)

type config struct {
	PostgresURL string
	ListenAddr  string
	Channel     string
	SharedNet   string
	GatewayImg  string
	GatewayPort int
}

type gatewaySpec struct {
	Slug            string
	ModuleSlug      string
	ModuleID        string
	Network         string
	TargetContainer string
	TargetPort      int
}

type controller struct {
	db          *sqlx.DB
	connInfo    string
	channelName string
	docker      *docker.Client
	sharedNet   string
	gatewayImg  string
	gatewayPort int

	statusMu sync.RWMutex
	statuses map[string]types.ProxyStatusPayload

	reconcileMu sync.Mutex
	triggerCh   chan struct{}
}

func newController(db *sqlx.DB, connInfo, channel string, cli *docker.Client, sharedNet, image string, gatewayPort int) *controller {
	return &controller{
		db:          db,
		connInfo:    connInfo,
		channelName: channel,
		docker:      cli,
		sharedNet:   sharedNet,
		gatewayImg:  image,
		gatewayPort: gatewayPort,
		statuses:    make(map[string]types.ProxyStatusPayload),
		triggerCh:   make(chan struct{}, 1),
	}
}

func (c *controller) replaceStatuses(next map[string]types.ProxyStatusPayload) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	c.statuses = next
}

func (c *controller) getStatus(slug string) (types.ProxyStatusPayload, bool) {
	c.statusMu.RLock()
	defer c.statusMu.RUnlock()
	payload, ok := c.statuses[slug]
	return payload, ok
}

func (c *controller) triggerReconcile() {
	select {
	case c.triggerCh <- struct{}{}:
	default:
	}
}

func (c *controller) listenForChanges(ctx context.Context) {
	report := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[net-controller] listener event=%v error=%v", ev, err)
		}
	}
	listener := pq.NewListener(c.connInfo, 5*time.Second, time.Minute, report)
	if err := listener.Listen(c.channelName); err != nil {
		log.Printf("[net-controller] failed to LISTEN %s: %v", c.channelName, err)
		return
	}
	log.Printf("[net-controller] listening for notifications on %q", c.channelName)

	for {
		select {
		case <-ctx.Done():
			listener.Close()
			return
		case <-listener.Notify:
			c.triggerReconcile()
		case <-time.After(2 * time.Minute):
			go func() {
				if err := listener.Ping(); err != nil {
					log.Printf("[net-controller] listener ping failed: %v", err)
				}
			}()
		}
	}
}

func (c *controller) run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.reconcile(context.Background()); err != nil {
				log.Printf("[net-controller] reconcile failed: %v", err)
			}
		case <-c.triggerCh:
			if err := c.reconcile(context.Background()); err != nil {
				log.Printf("[net-controller] reconcile failed: %v", err)
			}
		}
	}
}

func (c *controller) reconcile(ctx context.Context) error {
	c.reconcileMu.Lock()
	defer c.reconcileMu.Unlock()

	specs, err := c.fetchSpecs(ctx)
	if err != nil {
		return fmt.Errorf("fetch specs: %w", err)
	}
	existing, err := c.listGateways(ctx)
	if err != nil {
		return fmt.Errorf("list gateways: %w", err)
	}

	desired := make(map[string]gatewaySpec, len(specs))
	for _, spec := range specs {
		desired[spec.Slug] = spec
	}

	nextStatuses := make(map[string]types.ProxyStatusPayload, len(desired))

	// Remove containers for deleted pages
	for slug, cont := range existing {
		if _, ok := desired[slug]; !ok {
			log.Printf("[net-controller] removing stale gateway %s", cont.ID)
			if err := c.removeGateway(ctx, cont.ID); err != nil {
				log.Printf("[net-controller] remove gateway failed: %v", err)
			}
		}
	}

	for slug, spec := range desired {
		status := c.ensureGateway(ctx, spec, existing[slug])
		nextStatuses[slug] = status
	}

	c.replaceStatuses(nextStatuses)
	return nil
}

func (c *controller) ensureGateway(ctx context.Context, spec gatewaySpec, current containerSummary) types.ProxyStatusPayload {
	status := types.ProxyStatusPayload{
		Network: spec.Network,
	}
	containerName := strings.TrimSpace(spec.TargetContainer)
	if containerName == "" {
		status.Message = "invalid target container"
		return status
	}
	if spec.TargetPort <= 0 || spec.TargetPort > 65535 {
		status.Message = "invalid target port"
		return status
	}
	if strings.TrimSpace(spec.Network) == "" {
		status.Message = "no network selected"
		return status
	}

	needsCreate := current.ID == ""
	targetString := formatTarget(containerName, spec.TargetPort)

	if !needsCreate {
		if !strings.EqualFold(strings.TrimSpace(current.Network), spec.Network) ||
			strings.TrimSpace(current.Target) != targetString {
			log.Printf("[net-controller] gateway %s spec changed, recreating", current.Name)
			if err := c.removeGateway(ctx, current.ID); err != nil {
				log.Printf("[net-controller] remove gateway failed: %v", err)
			}
			needsCreate = true
		}
	}

	if needsCreate {
		if err := c.createGateway(ctx, spec, targetString); err != nil {
			log.Printf("[net-controller] create gateway for %s failed: %v", spec.Slug, err)
			status.Message = "failed to start gateway"
			return status
		}
	}

	info, err := c.inspectGateway(ctx, spec.Slug)
	if err != nil {
		log.Printf("[net-controller] inspect gateway %s failed: %v", spec.Slug, err)
		status.Message = "gateway unreachable"
		return status
	}

	missing := []string{}
	connected := []string{}
	if !info.connectedTo(c.sharedNet) {
		missing = append(missing, c.sharedNet)
	}
	if !info.connectedTo(spec.Network) {
		missing = append(missing, spec.Network)
	} else {
		connected = append(connected, spec.Network)
	}

	if len(missing) > 0 {
		status.Message = "missing networks"
		status.MissingNetworks = missing
		status.ConnectedNetworks = connected
		return status
	}

	if !info.Running {
		status.Message = "gateway stopped"
		return status
	}

	status.OK = true
	status.Message = "gateway attached"
	status.ConnectedNetworks = connected
	return status
}

func (c *controller) fetchSpecs(ctx context.Context) ([]gatewaySpec, error) {
	const query = `
		SELECT mp.slug,
		       mp.target_container,
		       mp.target_port,
		       mp.module_id,
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
		ModuleSlug      string `db:"module_slug"`
		NetworkName     string `db:"network_name"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	specs := make([]gatewaySpec, 0, len(rows))
	for _, r := range rows {
		slug := strings.TrimSpace(r.Slug)
		if slug == "" {
			continue
		}
		specs = append(specs, gatewaySpec{
			Slug:            slug,
			ModuleSlug:      r.ModuleSlug,
			ModuleID:        r.ModuleID,
			Network:         strings.TrimSpace(r.NetworkName),
			TargetContainer: strings.TrimSpace(r.TargetContainer),
			TargetPort:      r.TargetPort,
		})
	}
	return specs, nil
}

type containerSummary struct {
	ID      string
	Name    string
	Target  string
	Network string
}

func (c *controller) listGateways(ctx context.Context) (map[string]containerSummary, error) {
	args := filters.NewArgs()
	args.Add("label", gatewayLabel+"=true")
	containers, err := c.docker.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]containerSummary, len(containers))
	for _, cont := range containers {
		slug := cont.Labels[gatewaySlugLabel]
		if slug == "" {
			continue
		}
		result[slug] = containerSummary{
			ID:      cont.ID,
			Name:    strings.TrimPrefix(cont.Names[0], "/"),
			Target:  cont.Labels[gatewayTargetLabel],
			Network: cont.Labels[gatewayNetworkLabel],
		}
	}
	return result, nil
}

type gatewayInfo struct {
	Running  bool
	Networks map[string]string
}

func (info gatewayInfo) connectedTo(name string) bool {
	if info.Networks == nil {
		return false
	}
	_, ok := info.Networks[name]
	return ok
}

func (c *controller) inspectGateway(ctx context.Context, slug string) (gatewayInfo, error) {
	containerName := gatewayName(slug)
	resp, err := c.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return gatewayInfo{}, err
	}
	networks := make(map[string]string)
	for name, endpoint := range resp.NetworkSettings.Networks {
		if endpoint.NetworkID != "" {
			networks[name] = endpoint.NetworkID
		}
	}
	return gatewayInfo{
		Running:  resp.State.Running,
		Networks: networks,
	}, nil
}

func (c *controller) createGateway(ctx context.Context, spec gatewaySpec, target string) error {
	containerName := gatewayName(spec.Slug)
	cmd := c.gatewayCommand(target, spec)
	config := &container.Config{
		Image: c.gatewayImg,
		Labels: map[string]string{
			gatewayLabel:        "true",
			gatewaySlugLabel:    spec.Slug,
			gatewayTargetLabel:  target,
			gatewayNetworkLabel: spec.Network,
		},
		Cmd: []string{"sh", "-c", cmd},
	}
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	networking := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			c.sharedNet: {
				Aliases: []string{containerName},
			},
		},
	}
	resp, err := c.docker.ContainerCreate(ctx, config, hostConfig, networking, nil, containerName)
	if err != nil {
		return err
	}
	if err := c.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	if err := c.docker.NetworkConnect(ctx, spec.Network, resp.ID, nil); err != nil {
		_ = c.docker.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return err
	}
	return nil
}

func (c *controller) removeGateway(ctx context.Context, id string) error {
	return c.docker.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false,
		RemoveLinks:   false,
	})
}

func gatewayName(slug string) string {
	return fmt.Sprintf("gateway-%s", dnsSafeSlug(slug))
}

func dnsSafeSlug(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
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

func formatTarget(container string, port int) string {
	return fmt.Sprintf("http://%s:%d/", container, port)
}

func (c *controller) gatewayCommand(target string, spec gatewaySpec) string {
	var buf bytes.Buffer
	buf.WriteString("set -euo pipefail\n")
	buf.WriteString("cat <<'EOF' >/etc/nginx/nginx.conf\n")
	fmt.Fprintf(&buf, "events {}\n")
	buf.WriteString("http {\n")
	buf.WriteString("    map $http_upgrade $connection_upgrade {\n")
	buf.WriteString("        default upgrade;\n")
	buf.WriteString("        ''      close;\n")
	buf.WriteString("    }\n")
	buf.WriteString("    map $host $module_target_host {\n")
	buf.WriteString("        default $host;\n")
	buf.WriteString("        ~\\.modules\\.localhost$ localhost;\n")
	buf.WriteString("    }\n")
	fmt.Fprintf(&buf, "    server {\n        listen %d;\n", c.gatewayPort)
	buf.WriteString("        location / {\n")
	fmt.Fprintf(&buf, "            proxy_pass %s;\n", target)
	buf.WriteString("            proxy_http_version 1.1;\n")
	buf.WriteString("            proxy_set_header Host $module_target_host;\n")
	buf.WriteString("            proxy_set_header X-Forwarded-Host $host;\n")
	fmt.Fprintf(&buf, "            proxy_set_header X-Upstream-Host %s:%d;\n", spec.TargetContainer, spec.TargetPort)
	buf.WriteString("            proxy_set_header X-Forwarded-Proto $http_x_forwarded_proto;\n")
	buf.WriteString("            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	buf.WriteString("            proxy_set_header Upgrade $http_upgrade;\n")
	buf.WriteString("            proxy_set_header Connection $connection_upgrade;\n")
	buf.WriteString("            proxy_set_header X-Real-IP $remote_addr;\n")
	buf.WriteString("            proxy_buffering off;\n")
	buf.WriteString("        }\n")
	buf.WriteString("    }\n")
	buf.WriteString("}\n")
	buf.WriteString("EOF\n")
	buf.WriteString("exec nginx -g 'daemon off;'\n")
	return buf.String()
}

func (c *controller) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "missing slug", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		if err := c.reconcile(r.Context()); err != nil {
			log.Printf("[net-controller] manual reconcile failed: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "reconcile_failed", "Unable to reconcile gateways")
			return
		}
	}

	status, ok := c.getStatus(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		status.Reattached = true
	}
	respondJSON(w, status)
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[net-controller] write json failed: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   http.StatusText(status),
		"code":    code,
		"message": message,
	})
}

func loadConfig() (config, error) {
	pg := strings.TrimSpace(os.Getenv("POSTGRES_URL"))
	if pg == "" {
		return config{}, errors.New("POSTGRES_URL is required")
	}
	channel := strings.TrimSpace(os.Getenv("MODULES_PROXY_CHANNEL"))
	if channel == "" {
		channel = "module_page_changed"
	}
	shared := strings.TrimSpace(os.Getenv("MODULES_SHARED_NETWORK"))
	if shared == "" {
		shared = "pan-bagnat-proxy-net"
	}
	image := strings.TrimSpace(os.Getenv("MODULES_GATEWAY_IMAGE"))
	if image == "" {
		image = "nginx:alpine"
	}
	port := 9091
	if raw := strings.TrimSpace(os.Getenv("MODULES_NET_CONTROLLER_PORT")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			port = v
		}
	}
	gatewayPort := defaultGatewayPort
	if raw := strings.TrimSpace(os.Getenv("MODULES_GATEWAY_PORT")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			gatewayPort = v
		}
	}

	return config{
		PostgresURL: pg,
		ListenAddr:  fmt.Sprintf(":%d", port),
		Channel:     channel,
		SharedNet:   shared,
		GatewayImg:  image,
		GatewayPort: gatewayPort,
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
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(15)
	db.SetConnMaxLifetime(30 * time.Minute)

	dockerCli, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("failed to init docker client: %v", err)
	}

	ctrl := newController(db, cfg.PostgresURL, cfg.Channel, dockerCli, cfg.SharedNet, cfg.GatewayImg, cfg.GatewayPort)
	if err := ctrl.reconcile(ctx); err != nil {
		log.Fatalf("initial reconcile failed: %v", err)
	}

	go ctrl.listenForChanges(ctx)
	go ctrl.run(ctx)

	router := chi.NewRouter()
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	router.Get("/gateways/{slug}", ctrl.handleGatewayStatus)
	router.Post("/gateways/{slug}", ctrl.handleGatewayStatus)

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
			log.Printf("[net-controller] graceful shutdown failed: %v", err)
		}
		_ = db.Close()
		_ = dockerCli.Close()
	}()

	log.Printf("[net-controller] listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped unexpectedly: %v", err)
	}
}
