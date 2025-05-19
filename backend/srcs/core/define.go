package core

import "time"

type ModuleStatus string

const (
	Enabled     ModuleStatus = "enabled"
	Disabled    ModuleStatus = "disabled"
	Downloading ModuleStatus = "downloading"
)

type User struct {
	ID        string    `json:"id"`
	FtLogin   string    `json:"ftLogin"`
	FtID      string    `json:"ftID"`
	FtIsStaff bool      `json:"ftIsStaff"`
	PhotoURL  string    `json:"url"`
	LastSeen  time.Time `json:"lastUpdate"`
	IsStaff   bool      `json:"isStaff"`
	Roles     []Role    `json:"roles"`
}

type Role struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Color   string   `json:"color"`
	Users   []User   `json:"users"`
	Modules []Module `json:"modules"`
}

type Module struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Version       string       `json:"version"`
	Status        ModuleStatus `json:"status"`
	URL           string       `json:"url"`
	LatestVersion string       `json:"lastestVersion"`
	LateCommits   int          `json:"lateCommits"`
	LastUpdate    time.Time    `json:"lastUpdate"`
	Roles         []Role       `json:"roles"`
}

type ModulePostInput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	GitBranch string `json:"gitBranch"`
}

type ModulePatchInput struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	GitBranch string `json:"gitBranch"`
}
