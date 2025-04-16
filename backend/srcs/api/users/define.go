package users

import "time"

// Define the model for the API User input
// @Description API User model
type UserPostInput struct {
	ID        string `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Login     string `json:"login" example:"heinz"`
	PhotoURL  string `json:"url" example:"https://intra.42.fr/some-login/some-id"`
	GitBranch string `json:"gitBranch" example:"main"`
}

type FtUser struct {
	Login   string `json:"login" example:"heinz"`
	ID      string `json:"ftId" example:"1492"`
	IsStaff bool   `json:"ftIsStaff" example:"true"`
}

// Define the model for the API User response
// @Description API User model
type User struct {
	ID       string    `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtUser   FtUser    `json:"ftUser"`
	PhotoURL string    `json:"url" example:"https://intra.42.fr/some-login/some-id"`
	LastSeen time.Time `json:"lastUpdate" example:"2025-02-18T15:00:00Z"`
	IsStaff  bool      `json:"isStaff" example:"true"`
}
