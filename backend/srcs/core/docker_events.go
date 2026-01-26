package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"backend/websocket"

	dockercontainer "github.com/docker/docker/api/types/container"
	eventtypes "github.com/docker/docker/api/types/events"
	dockerfilters "github.com/docker/docker/api/types/filters"
)

type composeContainerRuntime struct {
	ID        string
	Name      string
	ModuleID  string
	Module    string
	Service   string
	Action    string
	Status    ContainerStatus
	UpdatedAt time.Time
}

var (
	eventWatcherOnce  sync.Once
	runtimeStateMu    sync.Mutex
	runtimeState      = make(map[string]composeContainerRuntime)
	moduleCacheMu     sync.RWMutex
	moduleCacheBySlug = make(map[string]Module)
)

// StartDockerEventWatcher spawns a goroutine that watches docker events and logs container states.
func StartDockerEventWatcher() {
	eventWatcherOnce.Do(func() {
		preloadComposeContainers()
		go dockerEventLoop()
	})
}

func dockerEventLoop() {
	for {
		if err := streamDockerEvents(); err != nil {
			log.Printf("[DockerEvents] stream error: %v", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func streamDockerEvents() error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	args := dockerfilters.NewArgs()
	args.Add("type", "container")
	args.Add("label", "com.docker.compose.project")
	messages, errs := cli.Events(ctx, eventtypes.ListOptions{Filters: args})
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("docker events channel closed")
			}
			handleDockerEventMessage(msg)
		case err, ok := <-errs:
			if !ok {
				return fmt.Errorf("docker events error channel closed")
			}
			return err
		}
	}
}

func handleDockerEventMessage(msg eventtypes.Message) {
	action := string(msg.Action)
	if strings.HasPrefix(action, "exec_") {
		return
	}
	moduleSlug, service, ok := moduleServiceFromAttributes(msg.Actor.Attributes)
	if !ok {
		return
	}
	module, ok := moduleFromSlug(moduleSlug)
	if !ok {
		return
	}
	key := fmt.Sprintf("%s/%s", moduleSlug, service)
	ts := eventTimestamp(msg)
	var payload *websocket.ContainerPayload
	stateChanged := false
	runtimeStateMu.Lock()
	defer func() {
		if stateChanged {
			logRuntimeStateLocked()
		}
		runtimeStateMu.Unlock()
		if payload != nil {
			websocket.SendContainerStatusEvent(module.ID, *payload)
		}
	}()

	if action == "destroy" {
		if _, exists := runtimeState[key]; exists {
			delete(runtimeState, key)
			stateChanged = true
			payload = &websocket.ContainerPayload{
				Name:   service,
				Status: string(ContainerDead),
				Reason: action,
				Since:  ts.Format(time.RFC3339),
			}
		}
		return
	}

	entry := runtimeState[key]
	entry.ID = msg.Actor.ID
	entry.Name = msg.Actor.Attributes["name"]
	entry.Module = moduleSlug
	entry.ModuleID = module.ID
	entry.Service = service
	entry.Action = action
	entry.Status = statusFromEvent(action, msg.Actor.Attributes, entry.Status)
	entry.UpdatedAt = ts
	runtimeState[key] = entry
	stateChanged = true
	payload = &websocket.ContainerPayload{
		Name:   service,
		Status: string(entry.Status),
		Reason: entry.Action,
		Since:  entry.UpdatedAt.Format(time.RFC3339),
	}
}

func eventTimestamp(msg eventtypes.Message) time.Time {
	if msg.TimeNano > 0 {
		return time.Unix(0, msg.TimeNano)
	}
	if msg.Time > 0 {
		return time.Unix(msg.Time, 0)
	}
	return time.Now()
}

func logRuntimeStateLocked() {
	if len(runtimeState) == 0 {
		log.Println("[DockerEvents] container map is empty")
		log.Println()
		return
	}
	log.Println("[DockerEvents] container map:")
	keys := make([]string, 0, len(runtimeState))
	for k := range runtimeState {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := runtimeState[key]
		shortID := entry.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		log.Printf("  %s | id=%s | status=%s | action=%s | updated=%s", key, shortID, entry.Status, entry.Action, entry.UpdatedAt.Format(time.RFC3339))
	}
	log.Println()
}

func moduleServiceFromAttributes(attrs map[string]string) (string, string, bool) {
	module := strings.TrimSpace(attrs["com.docker.compose.project"])
	service := strings.TrimSpace(attrs["com.docker.compose.service"])
	if module == "" || service == "" {
		return "", "", false
	}
	if !pathsContainRepo(attrs) {
		return "", "", false
	}
	return module, service, true
}

func pathsContainRepo(attrs map[string]string) bool {
	workingDir := strings.TrimSpace(attrs["com.docker.compose.project.working_dir"])
	configFiles := strings.TrimSpace(attrs["com.docker.compose.project.config_files"])
	repoBase := normalizedRepoBase()
	candidates := []string{}
	if repoBase != "" {
		candidates = append(candidates, repoBase)
	}
	candidates = append(candidates, "/repos")
	for _, needle := range candidates {
		if needle == "" {
			continue
		}
		if strings.Contains(workingDir, needle) || strings.Contains(configFiles, needle) {
			return true
		}
	}
	return false
}

func normalizedRepoBase() string {
	base := strings.TrimSpace(os.Getenv("REPO_BASE_PATH"))
	if base == "" {
		return ""
	}
	if !filepath.IsAbs(base) {
		if abs, err := filepath.Abs(base); err == nil {
			base = abs
		}
	}
	return base
}

func preloadComposeContainers() {
	cli, err := getDockerClient()
	if err != nil {
		log.Printf("[DockerEvents] preload skipped: %v", err)
		return
	}
	ctx := context.Background()
	args := dockerfilters.NewArgs()
	args.Add("label", "com.docker.compose.project")
	containers, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true, Filters: args})
	if err != nil {
		log.Printf("[DockerEvents] preload failed: %v", err)
		return
	}
	changed := false
	runtimeStateMu.Lock()
	for _, c := range containers {
		module, service, ok := moduleServiceFromAttributes(c.Labels)
		if !ok {
			continue
		}
		mod, ok := moduleFromSlug(module)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s/%s", module, service)
		entry := composeContainerRuntime{
			ID:        c.ID,
			Name:      firstDockerName(c.Names),
			ModuleID:  mod.ID,
			Module:    module,
			Service:   service,
			Action:    c.State,
			Status:    statusFromContainerState(c.State),
			UpdatedAt: time.Now(),
		}
		runtimeState[key] = entry
		changed = true
	}
	if changed {
		logRuntimeStateLocked()
	}
	runtimeStateMu.Unlock()
}

func moduleFromSlug(slug string) (Module, bool) {
	moduleCacheMu.RLock()
	if mod, ok := moduleCacheBySlug[slug]; ok {
		moduleCacheMu.RUnlock()
		return mod, true
	}
	moduleCacheMu.RUnlock()
	mod, err := GetModuleBySlug(slug)
	if err != nil || mod.ID == "" {
		log.Printf("[DockerEvents] unknown module slug %s: %v", slug, err)
		return Module{}, false
	}
	moduleCacheMu.Lock()
	moduleCacheBySlug[slug] = mod
	moduleCacheMu.Unlock()
	return mod, true
}

func statusFromContainerState(state string) ContainerStatus {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running":
		return ContainerRunning
	case "paused":
		return ContainerPaused
	case "created":
		return ContainerCreated
	case "restarting":
		return ContainerRestarting
	case "exited", "stopped":
		return ContainerExited
	case "dead":
		return ContainerDead
	default:
		return ContainerUnknown
	}
}

func statusFromEvent(action string, attrs map[string]string, prev ContainerStatus) ContainerStatus {
	switch strings.ToLower(action) {
	case "start", "restart", "unpause":
		return ContainerRunning
	case "stop":
		return ContainerExited
	case "kill":
		if prev == ContainerRunning {
			return ContainerRestarting
		}
		return ContainerExited
	case "die":
		return ContainerExited
	case "destroy":
		return ContainerDead
	case "pause":
		return ContainerPaused
	case "create":
		return ContainerCreated
	case "health_status":
		health := strings.ToLower(strings.TrimSpace(attrs["health-status"]))
		if health == "unhealthy" {
			return ContainerDead
		}
		if health == "healthy" {
			return ContainerRunning
		}
		return prev
	default:
		return prev
	}
}
