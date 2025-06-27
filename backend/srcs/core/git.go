package core

import (
	"backend/database"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func CloneModuleRepo(module Module) error {
	LogModule(module.ID, "INFO", fmt.Sprintf("Cloning repo %s in repos/%s", module.GitURL, module.Slug), nil)
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, module.Slug)

	tmpKey, err := os.CreateTemp("", "id_rsa_")

	if err != nil {
		return LogModule(module.ID, "error", "failed to create temp ssh key file", err)
	}

	defer os.Remove(tmpKey.Name())

	if err := os.WriteFile(tmpKey.Name(), []byte(module.SSHPrivateKey), 0600); err != nil {
		return LogModule(module.ID, "error", "failed to write private ssh key", err)
	}

	sshCommand := "ssh -i " + tmpKey.Name() + " -o StrictHostKeyChecking=no"
	cmd := exec.Command("git", "clone", module.GitURL, targetDir)
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return LogModule(module.ID, "ERROR", "git clone failed", fmt.Errorf("%s", string(output)))
	}

	newStatus := "disabled"
	err = database.PatchModule(database.ModulePatch{
		ID:     module.ID,
		Status: &newStatus,
	})
	if err != nil {
		return LogModule(module.ID, "ERROR", "error while updating status to database", err)
	}
	err = InitModuleForDocker(module)
	if err != nil {
		return err
	}
	return nil
}

func PullModuleRepo(module Module) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, module.Slug)

	tmpKey, err := os.CreateTemp("", "id_rsa_")
	if err != nil {
		return LogModule(module.ID, "error", "failed to create temp key file", err)
	}
	defer os.Remove(tmpKey.Name())

	if err := os.WriteFile(tmpKey.Name(), []byte(module.SSHPrivateKey), 0600); err != nil {
		return LogModule(module.ID, "error", "failed to write private key", err)
	}

	sshCommand := "ssh -i " + tmpKey.Name() + " -o StrictHostKeyChecking=no"

	cmd := exec.Command("git", "-C", targetDir, "pull", "--rebase")
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	output, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("%w | output: %s", err, output)
		return LogModule(module.ID, "ERROR", "git pull failed", wrappedErr)
	}
	return LogModule(module.ID, "INFO", fmt.Sprintf("Pulled module from URL %s", module.GitURL), nil)
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
