package docker

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func LoadManifest(path string) (*ModuleManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m ModuleManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func GenerateDockerComposeFromConfig(moduleSlug string, configYAML string) (string, error) {
	baseRepoPath := os.Getenv("TEMLATES_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../templates" // fallback for local dev
	}
	templatePath := filepath.Join(baseRepoPath, "compose.tmpl")
	fmt.Printf("Full path: `%s`\n", templatePath)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse compose template: %w", err)
	}

	var manifest ModuleManifest
	if err := yaml.Unmarshal([]byte(configYAML), &manifest); err != nil {
		return "", fmt.Errorf("failed to unmarshal module config: %w", err)
	}

	manifest.Module.Name = moduleSlug

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, manifest); err != nil {
		return "", fmt.Errorf("failed to render compose template: %w", err)
	}
	return buf.String(), nil
}
