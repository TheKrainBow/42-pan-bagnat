package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func InitModuleForDocker(moduleSlug string) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos" // fallback for local dev
	}
	targetDir := filepath.Join(baseRepoPath, moduleSlug)

	tplPath := filepath.Join(targetDir, "module-template.yml")
	modPath := filepath.Join(targetDir, "module.yml")

	out, err := os.OpenFile(modPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create module.yml: %w", err)
	}
	defer out.Close()

	if _, err := os.Stat(tplPath); err == nil {
		in, err := os.Open(tplPath)
		if err != nil {
			return fmt.Errorf("failed to open module-template.yml: %w", err)
		}
		defer in.Close()

		if _, err := io.Copy(out, in); err != nil {
			return fmt.Errorf("failed to copy template to module.yml: %w", err)
		}
	}
	return nil
}
