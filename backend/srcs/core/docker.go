package core

import (
	"backend/database"
	"backend/websocket"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("empty container name")
	}
	cmd := exec.Command("docker", "rm", "-f", name)
	return cmd.Run()
}

func ptrBool(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func GetModuleContainers(module Module) ([]ModuleContainer, error) {
	project := module.Slug

	cmd := exec.Command("docker", "ps", "-a",
		"--filter", fmt.Sprintf("label=com.docker.compose.project=%s", project),
		"--format", "{{.Names}}|{{.Status}}",
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var containers []ModuleContainer

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}

		fullName := parts[0]
		rawStatus := parts[1]

		// Simplify container name
		name := strings.TrimPrefix(fullName, project+"-")
		name = strings.TrimSuffix(name, "-1")

		// Parse status into enum
		var status ContainerStatus
		var reason, since string
		lowered := strings.ToLower(rawStatus)

		switch {
		case strings.HasPrefix(lowered, "up"):
			status = ContainerRunning
			reason = "Up"
			since = strings.TrimPrefix(rawStatus, "Up ")
		case strings.HasPrefix(lowered, "exited"):
			status = ContainerExited
			parts := strings.SplitN(rawStatus, " ", 2)
			reason = parts[0]
			if len(parts) > 1 {
				since = parts[1]
			}
		case strings.HasPrefix(lowered, "paused"):
			status = ContainerPaused
			reason = "Paused"
			since = strings.TrimPrefix(rawStatus, "Paused ")
		case strings.HasPrefix(lowered, "created"):
			status = ContainerCreated
			reason = "Created"
			since = strings.TrimPrefix(rawStatus, "Created ")
		case strings.HasPrefix(lowered, "restarting"):
			status = ContainerRestarting
			reason = "Restarting"
			since = strings.TrimPrefix(rawStatus, "Restarting ")
		case strings.HasPrefix(lowered, "dead"):
			status = ContainerDead
			reason = "Dead"
			since = strings.TrimPrefix(rawStatus, "Dead ")
		default:
			status = ContainerUnknown
			reason = rawStatus
		}

		containers = append(containers, ModuleContainer{
			Name:   name,
			Status: status,
			Reason: reason,
			Since:  since,
		})
	}

	return containers, nil
}

// GetAllContainers lists all containers, grouping info by compose project and networks.
func GetAllContainers() ([]AllContainer, error) {
	// Build module map and compose expectations (services, networks)
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

	type composeInfo struct {
		Services []string
		Networks []string
	}
	expects := make(map[string]composeInfo) // slug -> services, networks

	for _, m := range mods {
		dir := filepath.Join(baseRepoPath, m.Slug)
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err != nil {
			if !errorsIsNotExist(err) { /* ignore */
			}
			continue
		}
		// docker compose config --format json
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
		networks := []string{}
		if ns, ok := cfg["networks"].(map[string]any); ok {
			for k := range ns {
				networks = append(networks, k)
			}
		}
		// Also collect networks from existing containers of this project (in case defaults aren't declared)
		netsSet := map[string]bool{}
		for _, n := range networks {
			netsSet[n] = true
		}
		psCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", m.Slug),
			"--format", `{{.Networks}}`)
		psCmd.Dir = dir
		var psOut bytes.Buffer
		psCmd.Stdout = &psOut
		_ = psCmd.Run()
		for _, ln := range strings.Split(strings.TrimSpace(psOut.String()), "\n") {
			for _, n := range strings.Split(strings.TrimSpace(ln), ",") {
				n = strings.TrimSpace(n)
				if n == "" {
					continue
				}
				netsSet[n] = true
			}
		}
		networks = networks[:0]
		for n := range netsSet {
			networks = append(networks, n)
		}
		expects[m.Slug] = composeInfo{Services: services, Networks: networks}
	}

	// BFS over networks starting from compose networks
	visitedNet := make(map[string]bool)
	queue := []string{}
	for _, ci := range expects {
		for _, n := range ci.Networks {
			if !visitedNet[n] {
				visitedNet[n] = true
				queue = append(queue, n)
			}
		}
	}

	// helper to parse a docker ps line
	parseLine := func(line string) (name, rawStatus, project string, nets []string) {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			return "", "", "", nil
		}
		name = parts[0]
		rawStatus = parts[1]
		project = parts[2]
		if parts[3] != "" {
			nets = strings.Split(parts[3], ",")
		}
		return
	}
	// convert status
	toStatus := func(raw string) (ContainerStatus, string, string) {
		lowered := strings.ToLower(raw)
		switch {
		case strings.HasPrefix(lowered, "up"):
			return ContainerRunning, "Up", strings.TrimPrefix(raw, "Up ")
		case strings.HasPrefix(lowered, "exited"):
			p := strings.SplitN(raw, " ", 2)
			since := ""
			if len(p) > 1 {
				since = p[1]
			}
			return ContainerExited, p[0], since
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

	containers := make(map[string]AllContainer) // name -> item
	push := func(c AllContainer) { containers[c.Name] = c }

	for i := 0; i < len(queue); i++ {
		net := queue[i]
		cmd := exec.Command("docker", "ps", "-a", "--filter", "network="+net, "--format", `{{.Names}}|{{.Status}}|{{.Label "com.docker.compose.project"}}|{{.Networks}}`)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			continue
		}
		text := strings.TrimSpace(out.String())
		if text == "" {
			continue
		}
		for _, line := range strings.Split(text, "\n") {
			name, raw, project, nets := parseLine(line)
			if name == "" {
				continue
			}
			// Keep containers belonging to known module compose project;
			// otherwise, treat as orphan unless it's only infra (pan-bagnat-net)
			m, ok := modBySlug[project]
			if !ok {
				skip := true
				for _, n := range nets {
					if n != "pan-bagnat-net" {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
				st, reason, since := toStatus(raw)
				push(AllContainer{Name: name, Status: st, Reason: reason, Since: since, Project: "orphans", Networks: nets, Orphan: true})
				// still propagate networks
				for _, n := range nets {
					if !visitedNet[n] {
						visitedNet[n] = true
						queue = append(queue, n)
					}
				}
				continue
			}
			st, reason, since := toStatus(raw)
			c := AllContainer{Name: name, Status: st, Reason: reason, Since: since, Project: project, Networks: nets, ModuleID: m.ID, ModuleName: m.Name}
			push(c)
			// enqueue their networks
			for _, n := range nets {
				if !visitedNet[n] {
					visitedNet[n] = true
					queue = append(queue, n)
				}
			}
		}
	}

	// Add expected-but-missing containers (from compose services)
	for slug, ci := range expects {
		for _, svc := range ci.Services {
			// Compose default name pattern: project-service-1
			expectedName := fmt.Sprintf("%s-%s-1", slug, svc)
			if _, ok := containers[expectedName]; ok {
				continue
			}
			m := modBySlug[slug]
			push(AllContainer{Name: expectedName, Status: ContainerUnknown, Reason: "Not created", Since: "", Project: slug, Networks: ci.Networks, ModuleID: m.ID, ModuleName: m.Name, Missing: true})
		}
	}

	// Emit slice
	out := make([]AllContainer, 0, len(containers))
	for _, v := range containers {
		out = append(out, v)
	}
	return out, nil
}

func errorsIsNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func GetContainerLogs(module Module, containerName string, since string) ([]string, error) {
	fullName := fmt.Sprintf("%s-%s-1", module.Slug, containerName)

	// Include timestamps so clients can merge logs chronologically across sources
	var cmd *exec.Cmd
	if since != "" {
		// Use RFC3339 timestamp to fetch only newer logs
		cmd = exec.Command("docker", "logs", "--timestamps", "--since", since, fullName)
	} else {
		cmd = exec.Command("docker", "logs", "--timestamps", "--tail=1000", fullName)
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker logs failed: %w", err)
	}

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
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

	cmdPrune := exec.Command("docker", "image", "prune", "-a", "-f")
	cmdPrune.Dir = dir
	if err := runAndLog(module.ID, cmdPrune); err != nil {
		return LogModule(module.ID, "ERROR", "Failed to docker compose down", nil, err)
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
	fullName := fmt.Sprintf("%s-%s-1", module.Slug, containerName)
	fmt.Printf("[Docker] docker %s %s\n", action, fullName)
	cmd := exec.Command("docker", action, fullName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s failed: %v – %s", action, err, stderr.String())
	}
	return nil
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
