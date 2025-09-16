package api

import (
	"time"
)

// Role represents a user role in the system
// swagger:model Role
type Role struct {
	// ID is the unique identifier of the role
	ID string `json:"id" example:"role_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`

	// Name is the human‐readable name of the role (e.g. "IT", "Admin")
	Name string `json:"name" example:"IT"`

	// Color is the hex color code associated with the role in the UI
	Color string `json:"color" example:"#FF00FF"`

	// IsDefault indicates whether this role is assigned by default to new users
	IsDefault bool `json:"is_default" example:"true"`

	// UsersCount is the total number of users assigned to this role
	UsersCount int `json:"usersCount,omitempty" example:"42"`

	// Users lists the User objects that have this role
	Users []User `json:"users,omitempty"`

	// Modules lists the Module objects this role grants access to
	Modules []Module `json:"modules,omitempty"`
}

// User represents a 42-intranet user in the system
// swagger:model User
type User struct {
	// ID is the unique identifier of the user
	ID string `json:"id" example:"user_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`

	// FtID is the numeric 42-intranet ID of the user
	FtID int `json:"ft_id" example:"1492"`

	// FtLogin is the 42-intranet login handle (e.g. "heinz")
	FtLogin string `json:"ft_login" example:"heinz"`

	// FtIsStaff indicates whether the user is a 42-intranet staff member
	FtIsStaff bool `json:"ft_is_staff" example:"true"`

	// PhotoURL is the URL to the user’s 42-intranet profile picture
	PhotoURL string `json:"ft_photo" example:"https://intra.42.fr/some-login/some-id"`

	// LastSeen is the UTC timestamp of the user’s last activity
	LastSeen time.Time `json:"last_seen" example:"2025-02-18T15:00:00Z"`

	// IsStaff indicates whether the user has staff privileges within Pan Bagnat
	IsStaff bool `json:"is_staff" example:"true"`

	// Roles lists the roles assigned to the user
	Roles []Role `json:"roles,omitempty"`
}

// Module represents a feature module in the system
// swagger:model Module
type Module struct {
	// ID is the unique identifier of the module
	ID string `json:"id" example:"module_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`

	// SSHPublicKey is the RSA public key used for module deployments
	SSHPublicKey string `json:"ssh_public_key" example:"ssh-rsa AAAA..."`

	// Name is the human-readable title of the module
	Name string `json:"name" example:"Captain Hook"`

	// Slug is the URL-friendly identifier for the module
	Slug string `json:"slug" example:"captain-hook-main"`

	// Version is the currently deployed version of the module
	Version string `json:"version" example:"1.2"`

	// Status indicates whether the module is enabled or disabled
	Status ModuleStatus `json:"status" example:"enabled"`

	// GitURL is the repository URL where the module’s source lives
	GitURL string `json:"git_url" example:"https://github.com/some-user/some-repo"`

	// GitBranch is the branch currently tracked for deployments
	GitBranch string `json:"git_branch" example:"main"`

	// IconURL is the link to the module’s icon or logo
	IconURL string `json:"icon_url" example:"https://someURL/image.png"`

	// LatestVersion is the most recent version available upstream
	LatestVersion string `json:"latest_version" example:"1.7"`

	// LateCommits is the number of commits behind the latest version
	LateCommits int `json:"late_commits" example:"2"`

	// LastUpdate is the UTC timestamp of the module’s last update check
	LastUpdate time.Time `json:"last_update" example:"2025-02-18T15:00:00Z"`

	// Roles lists the roles that have access to this module
	Roles []Role `json:"roles,omitempty"`

	// IsDeploying indicates a deployment is currently running
	IsDeploying bool `json:"is_deploying" example:"false"`

	// LastDeploy is the timestamp of the latest successful deployment
    LastDeploy time.Time `json:"last_deploy" example:"2025-06-01T10:15:00Z"`

    // LastDeployStatus is the status of the latest deployment ("success", "failed", or "")
    LastDeployStatus string `json:"last_deploy_status" example:"success"`

    // Note: Git live details (current/ latest commit, behind, last fetch/pull)
    // are provided by /git/status and are intentionally not duplicated here
}

type ModuleLog struct {
	ID        int64          `json:"id"`
	ModuleID  string         `json:"module_id"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Meta      map[string]any `json:"meta"`
	CreatedAt time.Time      `json:"created_at"`
}

type ModulePage struct {
    ID       string `json:"id" example:"page_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
    Name     string `json:"name"`
    Slug     string `json:"slug"`
    URL      string `json:"url"`
    IsPublic bool   `json:"is_public"`
    ModuleID string `json:"module_id"`
    IconURL  string `json:"icon_url,omitempty"`
}

// Session represents a user session (device) in the system
// swagger:model Session
type Session struct {
    ID          string    `json:"id" example:"eyJhbGciOiJIUz..."`
    UserAgent   string    `json:"user_agent" example:"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)..."`
    IP          string    `json:"ip" example:"192.168.0.12"`
    DeviceLabel string    `json:"device_label" example:"MacBook Pro"`
    CreatedAt   time.Time `json:"created_at"`
    LastSeen    time.Time `json:"last_seen"`
    ExpiresAt   time.Time `json:"expires_at"`
    IsCurrent   bool      `json:"is_current"`
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
