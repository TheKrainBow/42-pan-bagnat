package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	networktypes "github.com/docker/docker/api/types/network"
)

// A scoped docker API endpoint handed to a module shell instead of the real
// docker socket. It only lets the shell see and manage the resources of the
// module's compose project (containers, volumes, networks), and refuses anything
// that could be used to reach beyond them: no container/volume/network creation
// (bind mounts), no privileged exec, no image build, no daemon-wide endpoints.

const (
	composeProjectLabel = "com.docker.compose.project"
	shellProxySockName  = "docker.sock"
	// shellProxyDirName lives next to the module repos so that it is visible
	// at the same relative path from the host (REPO_HOST_BASE_PATH) and from
	// this container (REPO_BASE_PATH).
	shellProxyDirName = ".pb-shell"
	maxProxyBodyBytes = 1 << 20
)

var dockerAPIVersionRe = regexp.MustCompile(`^/v[0-9]+\.[0-9]+`)

type scopedDockerProxy struct {
	project string
	dir     string
	opts    dockerProxyOptions
	srv     *http.Server
	rp      *httputil.ReverseProxy
}

// dockerProxyOptions adapts a scoped proxy to the Claude tab container.
type dockerProxyOptions struct {
	// keepDir keeps the socket directory on Close: it is bind-mounted in a
	// container that outlives the proxy (backend restart).
	keepDir bool
	// socketUID/GID own the socket when the client does not run as root.
	socketUID, socketGID int
}

func upstreamDockerSocket() string {
	if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); strings.HasPrefix(host, "unix://") {
		return strings.TrimPrefix(host, "unix://")
	}
	return "/var/run/docker.sock"
}

// startScopedDockerProxy listens on <dir>/docker.sock and filters requests to project.
func startScopedDockerProxy(project, dir string) (*scopedDockerProxy, error) {
	return startScopedDockerProxyWith(project, dir, dockerProxyOptions{})
}

func startScopedDockerProxyWith(project, dir string, opts dockerProxyOptions) (*scopedDockerProxy, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	sock := filepath.Join(dir, shellProxySockName)
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		if !opts.keepDir {
			os.RemoveAll(dir)
		}
		return nil, err
	}
	_ = os.Chmod(sock, 0o600)
	if opts.socketUID != 0 || opts.socketGID != 0 {
		_ = os.Chown(sock, opts.socketUID, opts.socketGID)
	}

	upstream := upstreamDockerSocket()
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", upstream)
		},
	}
	p := &scopedDockerProxy{project: project, dir: dir, opts: opts}
	p.rp = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "docker"
			req.Header["X-Forwarded-For"] = nil
		},
		Transport:     transport,
		FlushInterval: -1, // stream logs/events/attach as they come
	}
	p.srv = &http.Server{Handler: p, ReadHeaderTimeout: 10 * time.Second}
	go p.srv.Serve(ln)
	return p, nil
}

func (p *scopedDockerProxy) Close() {
	_ = p.srv.Close()
	if p.opts.keepDir {
		_ = os.Remove(filepath.Join(p.dir, shellProxySockName))
		return
	}
	_ = os.RemoveAll(p.dir)
}

func proxyError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func (p *scopedDockerProxy) deny(w http.ResponseWriter, r *http.Request) {
	proxyError(w, http.StatusForbidden, "Pan Bagnat module shell: this docker action is not allowed ("+r.Method+" "+r.URL.Path+")")
}

func (p *scopedDockerProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := dockerAPIVersionRe.ReplaceAllString(r.URL.Path, "")
	segs := strings.Split(strings.Trim(path, "/"), "/")
	get := r.Method == http.MethodGet
	post := r.Method == http.MethodPost
	del := r.Method == http.MethodDelete

	switch segs[0] {
	case "_ping":
		if get || r.Method == http.MethodHead {
			p.rp.ServeHTTP(w, r)
			return
		}
	case "version", "info":
		if get && len(segs) == 1 {
			p.rp.ServeHTTP(w, r)
			return
		}
	case "events":
		if get && len(segs) == 1 && p.injectFilter(w, r) {
			p.rp.ServeHTTP(w, r)
			return
		}
	case "containers":
		switch {
		case len(segs) == 2 && segs[1] == "json" && get,
			len(segs) == 2 && segs[1] == "prune" && post:
			if p.injectFilter(w, r) {
				p.rp.ServeHTTP(w, r)
			}
			return
		case len(segs) == 2 && del:
			if p.containerInScope(r.Context(), segs[1]) {
				p.rp.ServeHTTP(w, r)
			} else {
				proxyError(w, http.StatusNotFound, "No such container: "+segs[1])
			}
			return
		case len(segs) == 3:
			if !(get && containerReadActions[segs[2]]) && !(post && containerWriteActions[segs[2]]) {
				break
			}
			if !p.containerInScope(r.Context(), segs[1]) {
				proxyError(w, http.StatusNotFound, "No such container: "+segs[1])
				return
			}
			if post && segs[2] == "exec" && !p.sanitizeExecCreate(w, r) {
				return
			}
			p.rp.ServeHTTP(w, r)
			return
		}
	case "exec":
		if len(segs) == 3 && ((get && segs[2] == "json") || (post && (segs[2] == "start" || segs[2] == "resize"))) {
			if p.execInScope(r.Context(), segs[1]) {
				p.rp.ServeHTTP(w, r)
			} else {
				proxyError(w, http.StatusNotFound, "No such exec instance: "+segs[1])
			}
			return
		}
	case "volumes":
		switch {
		case len(segs) == 1 && get, len(segs) == 2 && segs[1] == "prune" && post:
			if p.injectFilter(w, r) {
				p.rp.ServeHTTP(w, r)
			}
			return
		case len(segs) == 2 && (get || del):
			if p.volumeInScope(r.Context(), segs[1]) {
				p.rp.ServeHTTP(w, r)
			} else {
				proxyError(w, http.StatusNotFound, "No such volume: "+segs[1])
			}
			return
		}
	case "networks":
		switch {
		case len(segs) == 1 && get:
			if p.injectFilter(w, r) {
				p.rp.ServeHTTP(w, r)
			}
			return
		case len(segs) == 2 && (get || del):
			if p.networkInScope(r.Context(), segs[1]) {
				p.rp.ServeHTTP(w, r)
			} else {
				proxyError(w, http.StatusNotFound, "No such network: "+segs[1])
			}
			return
		}
	}
	p.deny(w, r)
}

var containerReadActions = map[string]bool{"json": true, "logs": true, "top": true, "stats": true, "changes": true}
var containerWriteActions = map[string]bool{
	"start": true, "stop": true, "restart": true, "kill": true, "pause": true,
	"unpause": true, "wait": true, "resize": true, "attach": true, "exec": true,
	"rename": true,
}

// injectFilter forces the compose project label into the request's "filters"
// query parameter so list/prune/events can only ever see the module's resources.
func (p *scopedDockerProxy) injectFilter(w http.ResponseWriter, r *http.Request) bool {
	want := composeProjectLabel + "=" + p.project
	q := r.URL.Query()
	filters := map[string]any{}
	if raw := q.Get("filters"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &filters); err != nil {
			proxyError(w, http.StatusBadRequest, "invalid filters")
			return false
		}
	}
	switch cur := filters["label"].(type) {
	case []any: // {"label":["a=b"]}
		filters["label"] = append(cur, want)
	case map[string]any: // {"label":{"a=b":true}}
		cur[want] = true
	default:
		filters["label"] = map[string]any{want: true}
	}
	b, _ := json.Marshal(filters)
	q.Set("filters", string(b))
	r.URL.RawQuery = q.Encode()
	return true
}

// sanitizeExecCreate rejects privileged execs, which would bypass container isolation.
func (p *scopedDockerProxy) sanitizeExecCreate(w http.ResponseWriter, r *http.Request) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxProxyBodyBytes))
	if err != nil {
		proxyError(w, http.StatusBadRequest, "invalid body")
		return false
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		proxyError(w, http.StatusBadRequest, "invalid body")
		return false
	}
	if priv, _ := cfg["Privileged"].(bool); priv {
		proxyError(w, http.StatusForbidden, "Pan Bagnat module shell: privileged exec is not allowed")
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	r.Header.Set("Content-Length", fmt.Sprint(len(body)))
	return true
}

func (p *scopedDockerProxy) containerInScope(ctx context.Context, id string) bool {
	cli, err := getDockerClient()
	if err != nil {
		return false
	}
	info, err := cli.ContainerInspect(ctx, id)
	return err == nil && info.Config != nil && info.Config.Labels[composeProjectLabel] == p.project
}

func (p *scopedDockerProxy) execInScope(ctx context.Context, execID string) bool {
	cli, err := getDockerClient()
	if err != nil {
		return false
	}
	info, err := cli.ContainerExecInspect(ctx, execID)
	return err == nil && p.containerInScope(ctx, info.ContainerID)
}

func (p *scopedDockerProxy) volumeInScope(ctx context.Context, name string) bool {
	cli, err := getDockerClient()
	if err != nil {
		return false
	}
	vol, err := cli.VolumeInspect(ctx, name)
	return err == nil && vol.Labels[composeProjectLabel] == p.project
}

func (p *scopedDockerProxy) networkInScope(ctx context.Context, id string) bool {
	cli, err := getDockerClient()
	if err != nil {
		return false
	}
	nw, err := cli.NetworkInspect(ctx, id, networktypes.InspectOptions{})
	return err == nil && nw.Labels[composeProjectLabel] == p.project
}
