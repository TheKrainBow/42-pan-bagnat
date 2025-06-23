package users

import api "backend/api/dto"

// Define the model for the API User input
// @Description API User model
type UserPostInput struct {
	ID        string `json:"id" example:"user_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Login     string `json:"login" example:"heinz"`
	PhotoURL  string `json:"photo_url" example:"https://intra.42.fr/some-login/some-id"`
	GitBranch string `json:"gitBranch" example:"main"`
}

// Define the model for the API GET User response
// @Description API User model
type UserGetResponse struct {
	Users    []api.User `json:"users"`
	NextPage string     `json:"next_page_token" example:"BAD87as"`
}
