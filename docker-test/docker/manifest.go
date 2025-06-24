package docker

type ModuleManifest struct {
	Module     ModuleInfo     `yaml:"module"`
	Containers []ContainerDef `yaml:"containers"`
}

type ModuleInfo struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ContainerDef struct {
	Name    string   `yaml:"name"` // e.g. "backend", "frontend", "db"
	Image   string   `yaml:"image,omitempty"`
	Build   *Build   `yaml:"build,omitempty"`
	Expose  int      `yaml:"expose"`
	Env     []string `yaml:"env"`
	Volumes []Volume `yaml:"volumes"`
}

type Build struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

type Volume struct {
	Name          string `yaml:"name,omitempty"`
	HostPath      string `yaml:"hostPath,omitempty"`
	ContainerPath string `yaml:"containerPath"`
}
