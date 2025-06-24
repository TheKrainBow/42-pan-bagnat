package docker

import (
	"os"

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
