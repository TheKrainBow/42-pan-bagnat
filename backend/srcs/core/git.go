package core

import (
	"backend/database"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func CloneModuleRepo(moduleID, gitURL, moduleSlug, privateKey string) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, moduleSlug)

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

	newStatus := "disabled"
	err = database.PatchModule(database.ModulePatch{
		ID:     moduleID,
		Status: &newStatus,
	})
	if err != nil {
		log.Printf("error while updating status to database: %s\n", err.Error())
		return fmt.Errorf("error while updating status to database: %w", err)
	}
	return nil
}

func PullModuleRepo(moduleSlug, privateKey string) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, moduleSlug)

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

func UpdateModuleGitRemote(moduleID, moduleSlug, newGitURL string) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos"
	}
	targetDir := filepath.Join(baseRepoPath, moduleSlug)

	cmd := exec.Command("git", "-C", targetDir, "remote", "set-url", "origin", newGitURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update remote url: %w\nOutput: %s", err, output)
	}

	err = database.PatchModule(database.ModulePatch{
		ID:     moduleID,
		GitURL: &newGitURL,
	})
	if err != nil {
		log.Printf("error while updating git_url to database: %s\n", err.Error())
		return fmt.Errorf("error while updating git_url to database: %w", err)
	}
	return nil
}
