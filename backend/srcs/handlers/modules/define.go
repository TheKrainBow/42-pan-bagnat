package modules

import "time"

const (
	Enabled     ModuleStatus = "enabled"
	Disabled    ModuleStatus = "disabled"
	Downloading ModuleStatus = "downloading"
)

type ModuleStatus string

// Define the model for the API Module input
// @Description API Module model
type ModuleInput struct {
	Name      string `json:"name" example:"captain-hook"`
	URL       string `json:"url" example:"https://github.com/some-user/some-repo"`
	GitBranch string `json:"gitBranch" example:"main"`
}

// Define the model for the API Module response
// @Description API Module model
type Module struct {
	ID            string       `json:"id" example:""`
	Name          string       `json:"name" example:"captain-hook"`
	Version       string       `json:"version" example:"1.2"`
	Status        ModuleStatus `json:"status" example:"enabled"`
	URL           string       `json:"url" example:"https://github.com/some-user/some-repo"`
	LatestVersion string       `json:"lastestVersion" example:"1.7"`
	LateCommits   int          `json:"lateCommits" example:"2"`
	LastUpdate    time.Time    `json:"lastUpdate" example:"2025-02-18T15:00:00Z"`
}
