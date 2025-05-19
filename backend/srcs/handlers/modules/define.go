package modules

// Define the model for the API Module input
// @Description API Module model
type ModulePostInput struct {
	ID        string `json:"id" example:"module_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name      string `json:"name" example:"captain-hook"`
	URL       string `json:"url" example:"https://github.com/some-user/some-repo"`
	GitBranch string `json:"gitBranch" example:"main"`
}

// Define the model for the API Module input
// @Description API Module model
type ModulePatchInput struct {
	Name      string `json:"name" example:"captain-hook"`
	URL       string `json:"url" example:"https://github.com/some-user/some-repo"`
	GitBranch string `json:"gitBranch" example:"main"`
}
