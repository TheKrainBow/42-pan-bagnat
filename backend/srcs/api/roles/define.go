package roles

import api "backend/api/dto"

// Define the model for the API Role input
// @Description API Role model
type RolePostInput struct {
	Name  string `json:"name" example:"IT"`
	Color string `json:"color" example:"#FF00FF"`
}

// Define the model for the API Role input
// @Description API Role model
type RolePatchInput struct {
	// Name is the new name for the role.
	Name *string `json:"name,omitempty"  example:"Support"`
	// Color is the new hex color code for the role.
	Color *string `json:"color,omitempty" example:"#00FF00"`
	// IsDefault toggles whether this role is the default for new users.
	IsDefault *bool `json:"is_default,omitempty" example:"false"`
}

// RoleGetResponse is the paginated wrapper for a role list.
// swagger:model RoleGetResponse
type RoleGetResponse struct {
	// NextPageToken is the token to retrieve the next page of results.
	NextPageToken string `json:"next_page_token,omitempty" example:"eyJhbGciOi..."`

	// Roles is the list of roles on this page.
	Roles []api.Role `json:"roles"`
}

// RoleCreateInput defines the payload for creating a new role.
// swagger:model RoleCreateInput
type RoleCreateInput struct {
	// Name is the human-readable name of the role.
	Name string `json:"name"      example:"Support"`
	// Color is the hex color code for the role.
	Color string `json:"color"     example:"#00FF00"`
	// IsDefault indicates whether this role is the default for new users.
	IsDefault bool `json:"is_default" example:"false"`
	// Modules lists the IDs of modules this role should have access to.
	Modules []string `json:"modules"   example:"[\"module_01\",\"module_02\"]"`
}
