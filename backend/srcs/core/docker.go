package core

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

	cmd := exec.Command("docker", "compose", "-f", file, "up", "-d")
	cmd.Dir = dir
	err := runAndLog(module.ID, cmd)
	if err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			"Failed to docker up",
			nil,
			err,
		)
	}

	SetModuleStatus(module.ID, Enabled)
	LogModule(module.ID, "INFO", "docker compose up succeeded", nil, nil)
	return nil
}
