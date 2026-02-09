package core

import (
	"backend/database"
	"backend/websocket"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

var (
	dockerClientOnce sync.Once
	dockerClientInst *client.Client
	dockerClientErr  error
)

func getDockerClient() (*client.Client, error) {
	dockerClientOnce.Do(func() {
		dockerClientInst, dockerClientErr = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	return dockerClientInst, dockerClientErr
}

// Compose now uses the repository's docker-compose.yml directly.

func GetModuleConfig(module Module) (string, error) {
	root, err := ModuleRepoPath(module)
	if err != nil {
		return "", LogModule(module.ID, "ERROR", "invalid module slug", nil, err)
	}
	modPath := filepath.Join(root, "docker-compose.yml")

	data, err := os.ReadFile(modPath)
	if err != nil {
		return "", LogModule(module.ID, "ERROR", fmt.Sprintf("failed to read docker-compose.yml from %s", modPath), nil, err)
	}

	return string(data), nil
}

func SaveModuleConfig(module Module, content string) error {
	LogModule(module.ID, "INFO", "Saving config to docker-compose.yml", nil, nil)
	root, err := ModuleRepoPath(module)
	if err != nil {
		return LogModule(module.ID, "ERROR", "invalid module slug", nil, err)
	}
	modPath := filepath.Join(root, "docker-compose.yml")

	if err := os.WriteFile(modPath, []byte(content), 0o644); err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			fmt.Sprintf("failed to write docker-compose.yml to %s", modPath),
			nil,
			err,
		)
	}
	return nil
}

func DeployModule(module Module) error {
	dir, err := ModuleRepoPath(module)
	if err != nil {
		return LogModule(module.ID, "ERROR", "invalid module slug", nil, err)
	}
	file := "docker-compose.yml"

	// Mark deployment as in progress
	pending := "pending"
	_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, IsDeploying: ptrBool(true), LastDeployStatus: &pending})

	// Emit WS: deployment starting
	websocket.SendModuleDeployStatus(module.ID, true, "pending", "")

	// Step 1: docker compose build
	cmdBuild := exec.Command("docker", "compose", "-f", file, "build")
	cmdBuild.Dir = dir
	err = runAndLog(module.ID, cmdBuild)
	if err != nil {
		_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, IsDeploying: ptrBool(false), LastDeployStatus: strPtr("failed")})
		websocket.SendModuleDeployStatus(module.ID, false, "failed", "")
		return LogModule(module.ID, "ERROR", "Failed to docker build", nil, err)
	}

	// Step 2: docker compose up -d
	cmdUp := exec.Command("docker", "compose", "-f", file, "up", "-d")
	cmdUp.Dir = dir
	err = runAndLog(module.ID, cmdUp)
	if err != nil {
		_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, IsDeploying: ptrBool(false), LastDeployStatus: strPtr("failed")})
		websocket.SendModuleDeployStatus(module.ID, false, "failed", "")
		return LogModule(module.ID, "ERROR", "Failed to docker up", nil, err)
	}

	// Mark success: set last_deploy to now, clear in-progress, and set success status
	now := time.Now().UTC()
	_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, IsDeploying: ptrBool(false), LastDeploy: &now, LastDeployStatus: strPtr("success")})
	websocket.SendModuleDeployStatus(module.ID, false, "success", now.Format(time.RFC3339))
	SetModuleStatus(module.ID, Enabled, true)
	LogModule(module.ID, "INFO", "docker compose up succeeded", nil, nil)
	notifyContainersChanged(module)
	return nil
}

// ComposeRebuild rebuilds images without cache and restarts containers.
func ComposeRebuild(module Module) error {
	dir, err := ModuleRepoPath(module)
	if err != nil {
		return err
	}
	file := "docker-compose.yml"
	LogModule(module.ID, "INFO", "docker compose build --no-cache", nil, nil)
	cmdBuild := exec.Command("docker", "compose", "-f", file, "--project-name", module.Slug, "build", "--no-cache")
	cmdBuild.Dir = dir
	if err := runAndLog(module.ID, cmdBuild); err != nil {
		return err
	}
	LogModule(module.ID, "INFO", "docker compose up -d", nil, nil)
	cmdUp := exec.Command("docker", "compose", "-f", file, "--project-name", module.Slug, "up", "-d")
	cmdUp.Dir = dir
	if err := runAndLog(module.ID, cmdUp); err != nil {
		return err
	}
	notifyContainersChanged(module)
	return nil
}

// ComposeDown stops and removes the project's resources (without volumes).
func ComposeDown(module Module) error {
	dir, err := ModuleRepoPath(module)
	if err != nil {
		return err
	}
	file := "docker-compose.yml"
	LogModule(module.ID, "INFO", "docker compose down --remove-orphans", nil, nil)
	cmd := exec.Command("docker", "compose", "-f", file, "--project-name", module.Slug, "down", "--remove-orphans")
	cmd.Dir = dir
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	notifyContainersChanged(module)
	return nil
}

// RemoveContainer force-removes a container by name.
func RemoveContainer(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("empty container name")
	}
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	return cli.ContainerRemove(ctx, name, dockercontainer.RemoveOptions{Force: true})
}

func ptrBool(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func GetModuleContainers(module Module) ([]ModuleContainer, error) {
	project := strings.TrimSpace(module.Slug)
	if project == "" {
		return nil, fmt.Errorf("missing module slug")
	}
	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	filter := dockerfilters.NewArgs()
	filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", project))
	items, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true, Filters: filter})
	if err != nil {
		return nil, fmt.Errorf("list containers failed: %w", err)
	}
	var containers []ModuleContainer
	for _, item := range items {
		name := firstDockerName(item.Names)
		if name == "" {
			name = item.ID
		}
		status, reason, since := parseContainerStatus(item.Status)
		ports := containerPortsFromSummary(item)
		ports = ensureExposedPorts(ctx, cli, item.ID, ports)
		containers = append(containers, ModuleContainer{
			Name:   name,
			Status: status,
			Reason: reason,
			Since:  since,
			Ports:  ports,
		})
	}
	return containers, nil
}

func findModuleContainer(ctx context.Context, cli *client.Client, module Module, containerName string) (dockercontainer.Summary, bool) {
	containerName = strings.TrimSpace(containerName)
	if cli == nil || containerName == "" {
		return dockercontainer.Summary{}, false
	}
	project := strings.TrimSpace(module.Slug)
	if project == "" {
		return dockercontainer.Summary{}, false
	}
	filter := dockerfilters.NewArgs()
	filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", project))
	items, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true, Filters: filter})
	if err != nil || len(items) == 0 {
		return dockercontainer.Summary{}, false
	}

	score := func(item dockercontainer.Summary) int {
		pts := 0
		name := firstDockerName(item.Names)
		service := strings.TrimSpace(item.Labels["com.docker.compose.service"])
		canonical := ""
		if service != "" {
			canonical = fmt.Sprintf("%s-%s-1", project, service)
		}
		switch {
		case strings.EqualFold(containerName, name):
			pts += 200
		case service != "" && strings.EqualFold(containerName, service):
			pts += 150
		case canonical != "" && strings.EqualFold(containerName, canonical):
			pts += 120
		}
		if strings.EqualFold(strings.TrimSpace(item.State), "running") {
			pts += 20
		}
		return pts
	}

	bestIdx := -1
	bestScore := -1
	bestCreated := int64(-1)
	for i := range items {
		s := score(items[i])
		if s <= 0 {
			continue
		}
		if s > bestScore || (s == bestScore && items[i].Created > bestCreated) {
			bestIdx = i
			bestScore = s
			bestCreated = items[i].Created
		}
	}
	if bestIdx < 0 {
		return dockercontainer.Summary{}, false
	}
	return items[bestIdx], true
}

func containerPortsFromSummary(c dockercontainer.Summary) []ContainerPort {
	if len(c.Ports) == 0 {
		return nil
	}
	ports := make([]ContainerPort, 0, len(c.Ports))
	for _, p := range c.Ports {
		entry := ContainerPort{
			ContainerPort: int(p.PrivatePort),
			HostPort:      int(p.PublicPort),
			Protocol:      strings.ToLower(strings.TrimSpace(p.Type)),
		}
		if p.PublicPort > 0 {
			entry.Scope = "host"
		} else {
			entry.Scope = "internal"
		}
		ports = append(ports, entry)
	}
	return ports
}

func ensureExposedPorts(ctx context.Context, cli *client.Client, containerID string, ports []ContainerPort) []ContainerPort {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return ports
	}
	if cli == nil {
		return ports
	}
	report, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return ports
	}
	exposed := report.Config.ExposedPorts
	if len(exposed) == 0 {
		return ports
	}
	if len(exposed) == 0 {
		return ports
	}
	existing := make(map[string]struct{}, len(ports))
	for _, port := range ports {
		key := fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol)
		existing[key] = struct{}{}
	}
	for key := range exposed {
		num, proto := parsePortProto(key.Port())
		if num <= 0 {
			continue
		}
		normalized := fmt.Sprintf("%d/%s", num, proto)
		if _, ok := existing[normalized]; ok {
			continue
		}
		ports = append(ports, ContainerPort{
			ContainerPort: num,
			Protocol:      proto,
			Scope:         "internal",
		})
		existing[normalized] = struct{}{}
	}
	return ports
}

func parsePortProto(value string) (int, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ""
	}
	proto := ""
	if idx := strings.Index(value, "/"); idx >= 0 {
		proto = strings.ToLower(strings.TrimSpace(value[idx+1:]))
		value = value[:idx]
	}
	port, _ := strconv.Atoi(strings.TrimSpace(value))
	return port, proto
}

func parseContainerStatus(raw string) (ContainerStatus, string, string) {
	lowered := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(lowered, "up"):
		return ContainerRunning, "Up", strings.TrimPrefix(raw, "Up ")
	case strings.HasPrefix(lowered, "exited"):
		parts := strings.SplitN(raw, " ", 2)
		since := ""
		if len(parts) > 1 {
			since = parts[1]
		}
		return ContainerExited, parts[0], since
	case strings.HasPrefix(lowered, "paused"):
		return ContainerPaused, "Paused", strings.TrimPrefix(raw, "Paused ")
	case strings.HasPrefix(lowered, "created"):
		return ContainerCreated, "Created", strings.TrimPrefix(raw, "Created ")
	case strings.HasPrefix(lowered, "restarting"):
		return ContainerRestarting, "Restarting", strings.TrimPrefix(raw, "Restarting ")
	case strings.HasPrefix(lowered, "dead"):
		return ContainerDead, "Dead", strings.TrimPrefix(raw, "Dead ")
	default:
		return ContainerUnknown, raw, ""
	}
}

func firstDockerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

func networkNamesFromSummary(networks map[string]*networktypes.EndpointSettings) []string {
	if len(networks) == 0 {
		return nil
	}
	names := make([]string, 0, len(networks))
	for name := range networks {
		if strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// CollectModuleNetworkAliases maps every known hostname/alias of the module containers
// to the list of docker networks they are attached to, and returns all unique networks.
func CollectModuleNetworkAliases(module Module) (map[string][]string, []string, error) {
	slug := strings.TrimSpace(module.Slug)
	if slug == "" {
		return nil, nil, fmt.Errorf("missing module slug")
	}
	cli, err := getDockerClient()
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	filter := dockerfilters.NewArgs()
	filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", slug))
	list, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true, Filters: filter})
	if err != nil {
		return nil, nil, fmt.Errorf("list containers failed: %w", err)
	}
	aliases := make(map[string][]string)
	networkSet := make(map[string]struct{})
	for _, item := range list {
		inspect, err := cli.ContainerInspect(ctx, item.ID)
		if err != nil {
			continue
		}
		containerName := strings.Trim(strings.TrimPrefix(inspect.Name, "/"), " ")
		if containerName == "" {
			containerName = inspect.ID
		}
		baseAliases := []string{}
		if containerName != "" {
			baseAliases = append(baseAliases, containerName)
		}
		if hn := strings.TrimSpace(inspect.Config.Hostname); hn != "" {
			baseAliases = append(baseAliases, hn)
		}
		for netName, netInfo := range inspect.NetworkSettings.Networks {
			if netName == "" {
				continue
			}
			networkSet[netName] = struct{}{}
			names := append([]string{}, baseAliases...)
			if netInfo != nil {
				names = append(names, netInfo.Aliases...)
				names = append(names, netInfo.DNSNames...)
			}
			for _, alias := range names {
				alias = strings.ToLower(strings.TrimSpace(alias))
				if alias == "" {
					continue
				}
				list := aliases[alias]
				already := false
				for _, existing := range list {
					if existing == netName {
						already = true
						break
					}
				}
				if !already {
					aliases[alias] = append(list, netName)
				}
			}
		}
	}
	networks := make([]string, 0, len(networkSet))
	for n := range networkSet {
		networks = append(networks, n)
	}
	sort.Strings(networks)
	return aliases, networks, nil
}

// GetAllContainers lists all containers, grouping info by compose project and networks.
func GetAllContainers() ([]AllContainer, error) {
	mods, _, err := GetModules(ModulePagination{Limit: 10000})
	if err != nil {
		return nil, fmt.Errorf("failed to load modules: %w", err)
	}
	modBySlug := make(map[string]Module)
	for _, m := range mods {
		modBySlug[m.Slug] = m
	}

	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos"
	}

	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	type composeInfo struct {
		Services []string
		Networks []string
	}
	expects := make(map[string]composeInfo)

	for _, m := range mods {
		dir := filepath.Join(baseRepoPath, m.Slug)
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err != nil {
			if !errorsIsNotExist(err) {
			}
			continue
		}
		cfgCmd := exec.Command("docker", "compose", "-f", "docker-compose.yml", "--project-name", m.Slug, "config", "--format", "json")
		cfgCmd.Dir = dir
		var out bytes.Buffer
		cfgCmd.Stdout = &out
		if err := cfgCmd.Run(); err != nil {
			continue
		}
		var cfg map[string]any
		if err := json.Unmarshal(out.Bytes(), &cfg); err != nil {
			continue
		}
		services := []string{}
		if sv, ok := cfg["services"].(map[string]any); ok {
			for k := range sv {
				services = append(services, k)
			}
		}
		netsSet := map[string]bool{}
		if ns, ok := cfg["networks"].(map[string]any); ok {
			for k := range ns {
				netsSet[k] = true
			}
		}
		filter := dockerfilters.NewArgs()
		filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", m.Slug))
		if current, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true, Filters: filter}); err == nil {
			for _, c := range current {
				for name := range c.NetworkSettings.Networks {
					if name != "" {
						netsSet[name] = true
					}
				}
			}
		}
		networks := make([]string, 0, len(netsSet))
		for n := range netsSet {
			networks = append(networks, n)
		}
		expects[m.Slug] = composeInfo{Services: services, Networks: networks}
	}

	type containerInfo struct {
		ID       string
		Name     string
		Project  string
		Networks []string
		Status   string
	}

	all, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers failed: %w", err)
	}

	containersByNetwork := make(map[string][]*containerInfo)
	processed := make(map[string]bool)
	results := make(map[string]AllContainer)
	visitedNet := make(map[string]bool)
	var queue []string

	enqueueNet := func(netName string) {
		netName = strings.TrimSpace(netName)
		if netName == "" || visitedNet[netName] {
			return
		}
		visitedNet[netName] = true
		queue = append(queue, netName)
	}

	for _, ci := range expects {
		for _, n := range ci.Networks {
			enqueueNet(n)
		}
	}

	for _, item := range all {
		fullName := firstDockerName(item.Names)
		if fullName == "" {
			fullName = item.ID
		}
		info := &containerInfo{
			ID:       item.ID,
			Name:     fullName,
			Project:  item.Labels["com.docker.compose.project"],
			Networks: networkNamesFromSummary(item.NetworkSettings.Networks),
			Status:   item.Status,
		}
		for _, net := range info.Networks {
			containersByNetwork[net] = append(containersByNetwork[net], info)
		}
		if m, ok := modBySlug[info.Project]; ok {
			status, reason, since := parseContainerStatus(info.Status)
			results[info.Name] = AllContainer{
				Name:       info.Name,
				Status:     status,
				Reason:     reason,
				Since:      since,
				Project:    info.Project,
				Networks:   info.Networks,
				ModuleID:   m.ID,
				ModuleName: m.Name,
			}
			processed[info.ID] = true
			for _, net := range info.Networks {
				enqueueNet(net)
			}
		}
	}

	for len(queue) > 0 {
		net := queue[0]
		queue = queue[1:]
		for _, info := range containersByNetwork[net] {
			if info == nil || processed[info.ID] {
				continue
			}
			processed[info.ID] = true
			status, reason, since := parseContainerStatus(info.Status)
			if m, ok := modBySlug[info.Project]; ok {
				results[info.Name] = AllContainer{
					Name:       info.Name,
					Status:     status,
					Reason:     reason,
					Since:      since,
					Project:    info.Project,
					Networks:   info.Networks,
					ModuleID:   m.ID,
					ModuleName: m.Name,
				}
			} else {
				skip := true
				for _, n := range info.Networks {
					if n != "pan-bagnat-net" {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
				results[info.Name] = AllContainer{
					Name:     info.Name,
					Status:   status,
					Reason:   reason,
					Since:    since,
					Project:  "orphans",
					Networks: info.Networks,
					Orphan:   true,
				}
			}
			for _, n := range info.Networks {
				enqueueNet(n)
			}
		}
	}

	for slug, ci := range expects {
		for _, svc := range ci.Services {
			expectedName := fmt.Sprintf("%s-%s-1", slug, svc)
			if _, ok := results[expectedName]; ok {
				continue
			}
			m := modBySlug[slug]
			results[expectedName] = AllContainer{
				Name:       expectedName,
				Status:     ContainerUnknown,
				Reason:     "Not created",
				Project:    slug,
				Networks:   ci.Networks,
				ModuleID:   m.ID,
				ModuleName: m.Name,
				Missing:    true,
			}
		}
	}

	out := make([]AllContainer, 0, len(results))
	for _, v := range results {
		out = append(out, v)
	}
	return out, nil
}

func errorsIsNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func GetContainerLogs(module Module, containerName string, since string) ([]string, error) {
	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	target := strings.TrimSpace(containerName)
	if ref, ok := findModuleContainer(ctx, cli, module, containerName); ok {
		target = ref.ID
	}
	if target == "" {
		target = fmt.Sprintf("%s-%s-1", module.Slug, containerName)
	}
	opts := dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
	}
	if since != "" {
		opts.Since = since
	} else {
		opts.Tail = "1000"
	}
	reader, err := cli.ContainerLogs(ctx, target, opts)
	if err != nil {
		return nil, fmt.Errorf("container logs failed: %w", err)
	}
	defer reader.Close()
	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, reader); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("copy logs failed: %w", err)
	}
	text := strings.TrimRight(buf.String(), "\n")
	if text == "" {
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	return lines, nil
}

func CleanupModuleDockerResources(module Module) error {
	dir, err := ModuleRepoPath(module)
	if err != nil {
		return LogModule(module.ID, "ERROR", "invalid module slug", nil, err)
	}
	file := "docker-compose.yml"

	cmdDown := exec.Command("docker", "compose", "-f", file, "down", "--volumes", "--remove-orphans", "--rmi", "all")
	cmdDown.Dir = dir
	if err := runAndLog(module.ID, cmdDown); err != nil {
		return LogModule(module.ID, "ERROR", "Failed to docker compose down", nil, err)
	}

	cli, err := getDockerClient()
	if err != nil {
		return LogModule(module.ID, "ERROR", "Failed to init docker client", nil, err)
	}
	ctx := context.Background()
	args := dockerfilters.NewArgs()
	args.Add("dangling", "false")
	if _, err := cli.ImagesPrune(ctx, args); err != nil {
		return LogModule(module.ID, "ERROR", "Failed to prune images", nil, err)
	}

	LogModule(module.ID, "INFO", "docker cleanup completed", nil, nil)
	SetModuleStatus(module.ID, Disabled, false)
	notifyContainersChanged(module)
	return nil
}

func StartContainer(module Module, containerName string) error {
	if err := runDockerCommand(module, containerName, "start"); err != nil {
		return err
	}
	notifyContainersChanged(module)
	return nil
}

func StopContainer(module Module, containerName string) error {
	if err := runDockerCommand(module, containerName, "stop"); err != nil {
		return err
	}
	notifyContainersChanged(module)
	return nil
}

func RestartContainer(module Module, containerName string) error {
	if err := runDockerCommand(module, containerName, "restart"); err != nil {
		return err
	}
	notifyContainersChanged(module)
	return nil
}

func DeleteContainer(module Module, containerName string) error {
	if err := runDockerCommand(module, containerName, "rm"); err != nil {
		return err
	}
	notifyContainersChanged(module)
	return nil
}

func runDockerCommand(module Module, containerName, action string) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	target := strings.TrimSpace(containerName)
	if ref, ok := findModuleContainer(ctx, cli, module, containerName); ok {
		target = ref.ID
	}
	if target == "" {
		target = fmt.Sprintf("%s-%s-1", module.Slug, containerName)
	}
	fmt.Printf("[Docker] docker %s %s\n", action, target)
	switch action {
	case "start":
		return cli.ContainerStart(ctx, target, dockercontainer.StartOptions{})
	case "stop":
		return cli.ContainerStop(ctx, target, dockercontainer.StopOptions{})
	case "restart":
		return cli.ContainerRestart(ctx, target, dockercontainer.StopOptions{})
	case "rm":
		return cli.ContainerRemove(ctx, target, dockercontainer.RemoveOptions{Force: true})
	default:
		return fmt.Errorf("unsupported docker action %q", action)
	}
}

// notifyContainersChanged fetches the current containers and emits a WS event
func notifyContainersChanged(module Module) {
	containers, err := GetModuleContainers(module)
	if err != nil {
		return
	}
	payload := make([]websocket.ContainerPayload, 0, len(containers))
	for _, c := range containers {
		payload = append(payload, websocket.ContainerPayload{
			Name:   c.Name,
			Status: string(c.Status),
			Reason: c.Reason,
			Since:  c.Since,
		})
	}
	websocket.SendContainersUpdatedEvent(module.ID, payload)
}
