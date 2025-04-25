package users

import (
	"backend/api/roles"
	"time"
)

// Define the model for the API User input
// @Description API User model
type UserPostInput struct {
	ID        string `json:"id" example:"user_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Login     string `json:"login" example:"heinz"`
	PhotoURL  string `json:"url" example:"https://intra.42.fr/some-login/some-id"`
	GitBranch string `json:"gitBranch" example:"main"`
}

// Define the model for the API User Object
// @Description API User model
type User struct {
	ID        string       `json:"id" example:"user_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtID      string       `json:"ft_id" example:"1492"`
	FtLogin   string       `json:"ft_login" example:"heinz"`
	FtIsStaff bool         `json:"ft_is_staff" example:"true"`
	PhotoURL  string       `json:"ft_photo" example:"https://intra.42.fr/some-login/some-id"`
	LastSeen  time.Time    `json:"last_seen" example:"2025-02-18T15:00:00Z"`
	IsStaff   bool         `json:"is_staff" example:"true"`
	Roles     []roles.Role `json:"roles"`
}

// Define the model for the API GET User response
// @Description API User model
type UserGetResponse struct {
	Users    []User `json:"users"`
	NextPage string `json:"next_page_token" example:"BAD87as"`
}
