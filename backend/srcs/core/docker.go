package core

import (
	"io"
	"os"
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
