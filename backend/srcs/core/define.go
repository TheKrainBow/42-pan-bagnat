package core

import "time"

type ModuleStatus string

const (
	Enabled          ModuleStatus = "enabled"
	Disabled         ModuleStatus = "disabled"
	Downloading      ModuleStatus = "downloading"
	WaitingForAction ModuleStatus = "waiting_for_action"
)

type User struct {
	ID        string    `json:"id"`
	FtLogin   string    `json:"ftLogin"`
	FtID      string    `json:"ft_id"`
	FtIsStaff bool      `json:"ft_is_staff"`
	PhotoURL  string    `json:"photo_url"`
	LastSeen  time.Time `json:"last_update"`
	IsStaff   bool      `json:"is_staff"`
	Roles     []Role    `json:"roles"`
}

type Role struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color"`
	Users      []User   `json:"users"`
	UsersCount int      `json:"usersCount"`
	Modules    []Module `json:"modules"`
}

type Module struct {
	ID            string       `json:"id"`
	SSHPublicKey  string       `json:"ssh_public_key"`
	SSHPrivateKey string       `json:"ssh_private_key"`
	Name          string       `json:"name"`
	Slug          string       `json:"slug"`
	Version       string       `json:"version"`
	Status        ModuleStatus `json:"status"`
	GitURL        string       `json:"git_url"`
	IconURL       string       `json:"icon_url"`
	LatestVersion string       `json:"latest_Version"`
	LateCommits   int          `json:"late_commits"`
	LastUpdate    time.Time    `json:"last_update"`
	Roles         []Role       `json:"roles"`
}

type ModulePostInput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GitURL    string `json:"git_url"`
	GitBranch string `json:"gitBranch"`
}

type ModulePatchInput struct {
	Name      string `json:"name"`
	GitURL    string `json:"git_url"`
	GitBranch string `json:"gitBranch"`
}
