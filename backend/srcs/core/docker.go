package core

import (
	"backend/docker"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func InitModuleForDocker(module Module) error {
	LogModule(module.ID, "INFO", "Setting up module.yml", nil)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, module.Slug)

	tplPath := filepath.Join(targetDir, "module-template.yml")
	modPath := filepath.Join(targetDir, "module.yml")

	out, err := os.OpenFile(modPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return LogModule(module.ID, "ERROR", "failed to create module.yml", err)
	}
	defer out.Close()

	if _, err := os.Stat(tplPath); err == nil {
		in, err := os.Open(tplPath)
		if err != nil {
			return LogModule(module.ID, "ERROR", "failed to open module-template.yml", err)
		}
		defer in.Close()

		if _, err := io.Copy(out, in); err != nil {
			return LogModule(module.ID, "ERROR", "failed to copy template to module.yml", err)
		}
	} else {
		LogModule(module.ID, "INFO", "no module-template.yml, generating an empty module.yml", err)
	}
	return nil
}

func GetModuleConfig(module Module) (string, error) {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	modPath := filepath.Join(baseRepoPath, module.Slug, "module.yml")

	data, err := os.ReadFile(modPath)
	if err != nil {
		return "", LogModule(module.ID, "ERROR", fmt.Sprintf("failed to read module.yml from %s", modPath), err)
	}

	return string(data), nil
}

func SaveModuleConfig(module Module, content string) error {
	LogModule(module.ID, "INFO", "Saving config to module.yml", nil)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	modPath := filepath.Join(baseRepoPath, module.Slug, "module.yml")
	composePath := filepath.Join(baseRepoPath, module.Slug, "docker-compose.yml")

	if err := os.WriteFile(modPath, []byte(content), 0o644); err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			fmt.Sprintf("failed to write module.yml to %s", modPath),
			err,
		)
	}

	composeYAML, err := docker.GenerateDockerComposeFromConfig(module.Slug, content)
	if err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			fmt.Sprintf("failed to generate docker compose file to %s", composePath),
			err,
		)
	}

	if err := os.WriteFile(composePath, []byte(composeYAML), 0o644); err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			fmt.Sprintf("failed to write docker-compose.yml to %s", modPath),
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

	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = dir
	err := runAndLog(module.ID, cmd)
	if err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			"Failed to docker up",
			err,
		)
	}

	SetModuleStatus(module.ID, Enabled)
	LogModule(module.ID, "INFO", "docker compose up succeeded", nil)
	return nil
}
