package roles

// Define the model for the API Role input
// @Description API Role model
type RolePostInput struct {
	Name  string `json:"name" example:"captain-hook"`
	Color string `json:"color" example:"0xFF00FF"`
}

// Define the model for the API Role input
// @Description API Role model
type RolePatchInput struct {
	Name  string `json:"name" example:"captain-hook"`
	Color string `json:"color" example:"0xFF00FF"`
}

// Define the model for the API Role response
// @Description API Role model
type Role struct {
	ID    string `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name  string `json:"name" example:"captain-hook"`
	Color string `json:"color" example:"0xFF00FF"`
}
