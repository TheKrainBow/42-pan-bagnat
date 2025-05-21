package api

import (
	"time"
)

// Define the model for the API Role response
// @Description API Role model
type Role struct {
	ID      string   `json:"id" example:"role_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name    string   `json:"name" example:"IT"`
	Color   string   `json:"color" example:"0xFF00FF"`
	Users   []User   `json:"users,omitempty"`
	Modules []Module `json:"modules,omitempty"`
}

// Define the model for the API User Object
// @Description API User model
type User struct {
	ID        string    `json:"id" example:"user_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtID      string    `json:"ft_id" example:"1492"`
	FtLogin   string    `json:"ft_login" example:"heinz"`
	FtIsStaff bool      `json:"ft_is_staff" example:"true"`
	PhotoURL  string    `json:"ft_photo" example:"https://intra.42.fr/some-login/some-id"`
	LastSeen  time.Time `json:"last_seen" example:"2025-02-18T15:00:00Z"`
	IsStaff   bool      `json:"is_staff" example:"true"`
	Roles     []Role    `json:"roles,omitempty"`
}

// Define the model for the API Module response
// @Description API Module model
type Module struct {
	ID            string       `json:"id" example:"module_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name          string       `json:"name" example:"captain-hook"`
	Version       string       `json:"version" example:"1.2"`
	Status        ModuleStatus `json:"status" example:"enabled"`
	URL           string       `json:"url" example:"https://github.com/some-user/some-repo"`
	IconURL       string       `json:"icon_url" example:"https://someURL/image.png"`
	LatestVersion string       `json:"latest_version" example:"1.7"`
	LateCommits   int          `json:"late_commits" example:"2"`
	LastUpdate    time.Time    `json:"last_update" example:"2025-02-18T15:00:00Z"`
	Roles         []Role       `json:"roles,omitempty"`
}

const (
	Enabled     ModuleStatus = "enabled"
	Disabled    ModuleStatus = "disabled"
	Downloading ModuleStatus = "downloading"
)

type ModuleStatus string
