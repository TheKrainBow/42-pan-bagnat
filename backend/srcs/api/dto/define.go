package api

import (
	"time"
)

// Define the model for the API Role response
// @Description API Role model
type Role struct {
	ID         string   `json:"id" example:"role_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name       string   `json:"name" example:"IT"`
	Color      string   `json:"color" example:"0xFF00FF"`
	UsersCount int      `json:"usersCount" example:"42"`
	Users      []User   `json:"users,omitempty"`
	Modules    []Module `json:"modules,omitempty"`
}

// Define the model for the API User Object
// @Description API User model
type User struct {
	ID        string    `json:"id" example:"user_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtID      int       `json:"ft_id" example:"1492"`
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
	SSHPublicKey  string       `json:"ssh_public_key" example:"ssh-rsa AAAA..."`
	Name          string       `json:"name" example:"Captain Hook"`
	Slug          string       `json:"slug" example:"captain-hook-main"`
	Version       string       `json:"version" example:"1.2"`
	Status        ModuleStatus `json:"status" example:"enabled"`
	GitURL        string       `json:"git_url" example:"https://github.com/some-user/some-repo"`
	GitBranch     string       `json:"git_branch" example:"main"`
	IconURL       string       `json:"icon_url" example:"https://someURL/image.png"`
	LatestVersion string       `json:"latest_version" example:"1.7"`
	LateCommits   int          `json:"late_commits" example:"2"`
	LastUpdate    time.Time    `json:"last_update" example:"2025-02-18T15:00:00Z"`
	Roles         []Role       `json:"roles,omitempty"`
}

type ModuleLog struct {
	ID        int64                  `json:"id"`
	ModuleID  string                 `json:"module_id"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Meta      map[string]interface{} `json:"meta"`
	CreatedAt time.Time              `json:"created_at"`
}

type ModulePage struct {
	ID       string `json:"id" example:"page_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	URL      string `json:"url"`
	IsPublic bool   `json:"is_public"`
	ModuleID string `json:"module_id"`
}

type ModuleContainer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

const (
	Cloning       ModuleStatus = "cloning"
	WaitingDeploy ModuleStatus = "waiting_for_deploy"
	Disabled      ModuleStatus = "disabled"
	Enabled       ModuleStatus = "enabled"
)

type ModuleStatus string
