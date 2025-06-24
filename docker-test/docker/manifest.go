package docker

// ModuleManifest mirrors module.yml
type ModuleManifest struct {
	Module   ModuleInfo   `yaml:"module"`
	Services []ServiceDef `yaml:"services"`
	Volumes  []string     `yaml:"volumes,omitempty"` // top-level list of volume names
}

type ModuleInfo struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// ServiceDef describes one service
type ServiceDef struct {
	Name      string     `yaml:"name"`
	Image     string     `yaml:"image,omitempty"`
	Build     *Build     `yaml:"build,omitempty"`
	Expose    []int      `yaml:"expose,omitempty"`
	Publish   []int      `yaml:"publish,omitempty"`
	Env       []EnvEntry `yaml:"env,omitempty"`
	DependsOn []string   `yaml:"depends_on,omitempty"`
	Volumes   []Volume   `yaml:"volumes,omitempty"`
}

type Build struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

type Volume struct {
	Name        string `yaml:"name,omitempty"`
	HostPath    string `yaml:"hostPath,omitempty"`
	ServicePath string `yaml:"servicePath"`
}

// EnvEntry is now just a flat key/value pair
type EnvEntry struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}
