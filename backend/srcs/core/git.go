package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var invalidFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.TrimSpace(name)
	name = invalidFileNameChars.ReplaceAllString(name, "-")
	return strings.Trim(name, "-._")
}

func CloneModuleRepo(moduleName, gitURL, privateKey string) error {
	moduleName = sanitizeFileName(moduleName)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, moduleName)

	tmpKey, err := os.CreateTemp("", "id_rsa_")
	if err != nil {
		return fmt.Errorf("failed to create temp key file: %w", err)
	}
	defer os.Remove(tmpKey.Name())

	if err := os.WriteFile(tmpKey.Name(), []byte(privateKey), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	sshCommand := "ssh -i " + tmpKey.Name() + " -o StrictHostKeyChecking=no"
	cmd := exec.Command("git", "clone", gitURL, targetDir)
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func PullModuleRepo(moduleName, privateKey string) error {
	moduleName = sanitizeFileName(moduleName)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, moduleName)

	tmpKey, err := os.CreateTemp("", "id_rsa_")
	if err != nil {
		return fmt.Errorf("failed to create temp key file: %w", err)
	}
	defer os.Remove(tmpKey.Name())

	if err := os.WriteFile(tmpKey.Name(), []byte(privateKey), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	sshCommand := "ssh -i " + tmpKey.Name() + " -o StrictHostKeyChecking=no"

	cmd := exec.Command("git", "-C", targetDir, "pull", "--rebase")
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func UpdateModuleGitRemote(moduleName, newGitURL string) error {
	moduleName = sanitizeFileName(moduleName)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos"
	}
	targetDir := filepath.Join(baseRepoPath, moduleName)

	cmd := exec.Command("git", "-C", targetDir, "remote", "set-url", "origin", newGitURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update remote url: %w\nOutput: %s", err, output)
	}
	return nil
}
