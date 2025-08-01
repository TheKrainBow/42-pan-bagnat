package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func InitModuleForDocker(module Module) error {
	LogModule(module.ID, "INFO", "Setting up docker-compose-panbagnat.yml", nil, nil)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, module.Slug)

	defaultPath := filepath.Join(targetDir, "docker-compose.yml")
	pbPath := filepath.Join(targetDir, "docker-compose-panbagnat.yml")

	out, err := os.OpenFile(pbPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return LogModule(module.ID, "ERROR", "failed to create docker-compose-panbagnat.yml", nil, err)
	}
	defer out.Close()

	if _, err := os.Stat(defaultPath); err == nil {
		in, err := os.Open(defaultPath)
		if err != nil {
			return LogModule(module.ID, "ERROR", "failed to open docker-compose.yml", nil, err)
		}
		defer in.Close()

		if _, err := io.Copy(out, in); err != nil {
			return LogModule(module.ID, "ERROR", "failed to copy docker-compose.yml to docker-compose-panbagnat.yml", nil, err)
		}
		LogModule(module.ID, "INFO", "docker-compose-panbagnat.yml created using docker-compose.yml", nil, nil)
	} else {
		LogModule(module.ID, "INFO", "docker-compose.yml not found, created empty docker-compose-panbagnat.yml", nil, nil)
	}
	return nil
}

func GetModuleConfig(module Module) (string, error) {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	modPath := filepath.Join(baseRepoPath, module.Slug, "docker-compose-panbagnat.yml")

	data, err := os.ReadFile(modPath)
	if err != nil {
		return "", LogModule(module.ID, "ERROR", fmt.Sprintf("failed to read docker-compose-panbagnat.yml from %s", modPath), nil, err)
	}

	return string(data), nil
}

func SaveModuleConfig(module Module, content string) error {
	LogModule(module.ID, "INFO", "Saving config to docker-compose-panbagnat.yml", nil, nil)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	modPath := filepath.Join(baseRepoPath, module.Slug, "docker-compose-panbagnat.yml")

	if err := os.WriteFile(modPath, []byte(content), 0o644); err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			fmt.Sprintf("failed to write docker-compose-panbagnat.yml to %s", modPath),
			nil,
			err,
		)
	}
	return nil
}

func DeployModule(module Module) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	dir := filepath.Join(baseRepoPath, module.Slug)
	file := "docker-compose-panbagnat.yml"

	// Step 1: docker compose build
	cmdBuild := exec.Command("docker", "compose", "-f", file, "build")
	cmdBuild.Dir = dir
	err := runAndLog(module.ID, cmdBuild)
	if err != nil {
		return LogModule(module.ID, "ERROR", "Failed to docker build", nil, err)
	}

	// Step 2: docker compose up -d
	cmdUp := exec.Command("docker", "compose", "-f", file, "up", "-d")
	cmdUp.Dir = dir
	err = runAndLog(module.ID, cmdUp)
	if err != nil {
		return LogModule(module.ID, "ERROR", "Failed to docker up", nil, err)
	}

	SetModuleStatus(module.ID, Enabled, true)
	LogModule(module.ID, "INFO", "docker compose up succeeded", nil, nil)
	return nil
}

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

func GetContainerLogs(module Module, containerName string) ([]string, error) {
	fullName := fmt.Sprintf("%s-%s-1", module.Slug, containerName)

	cmd := exec.Command("docker", "logs", "--tail=1000", fullName)

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
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	dir := filepath.Join(baseRepoPath, module.Slug)
	file := "docker-compose-panbagnat.yml"

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
	return nil
}

func StartContainer(module Module, containerName string) error {
	return runDockerCommand(module, containerName, "start")
}

func StopContainer(module Module, containerName string) error {
	return runDockerCommand(module, containerName, "stop")
}

func RestartContainer(module Module, containerName string) error {
	return runDockerCommand(module, containerName, "restart")
}

func DeleteContainer(module Module, containerName string) error {
	return runDockerCommand(module, containerName, "rm")
}

func runDockerCommand(module Module, containerName, action string) error {
	fullName := fmt.Sprintf("%s-%s-1", module.Slug, containerName)
	cmd := exec.Command("docker", action, fullName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s failed: %v – %s", action, err, stderr.String())
	}
	return nil
}
