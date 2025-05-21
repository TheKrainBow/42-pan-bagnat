package roles

import api "backend/api/dto"

// Define the model for the API Role input
// @Description API Role model
type RolePostInput struct {
	Name  string `json:"name" example:"IT"`
	Color string `json:"color" example:"0xFF00FF"`
}

// Define the model for the API Role input
// @Description API Role model
type RolePatchInput struct {
	Name  string `json:"name" example:"IT"`
	Color string `json:"color" example:"0xFF00FF"`
}

// Define the model for the API GET User response
// @Description API User model
type RoleGetResponse struct {
	Roles    []api.Role `json:"roles"`
	NextPage string     `json:"next_page_token" example:"BAD87as"`
}
