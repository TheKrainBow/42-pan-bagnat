package modules

import (
	api "backend/api/dto"
	"time"
)

// Define the model for the API Module input
// @Description API Module model
type ModulePostInput struct {
	ID        string `json:"id" example:"module_01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name      string `json:"name" example:"captain-hook"`
	GitURL    string `json:"git_url" example:"https://github.com/some-user/some-repo"`
	GitBranch string `json:"gitBranch" example:"main"`
}

// Define the model for the API Module input
// @Description API Module model
type ModulePatchInput struct {
	Name      *string `json:"name" example:"captain-hook"`
	GitURL    *string `json:"git_url" example:"https://github.com/some-user/some-repo"`
	GitBranch *string `json:"gitBranch" example:"main"`
}

// ModuleGetResponse is the paginated wrapper for a module list.
// swagger:model ModuleGetResponse
type ModuleGetResponse struct {
	// NextPageToken is the token to retrieve the next page of results.
	NextPageToken string `json:"next_page_token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"`
	// Modules is the list of modules on this page.
	Modules []api.Module `json:"modules"`
}

// ModuleLogsGetResponse is the paginated wrapper for module log entries.
// swagger:model ModuleLogsGetResponse
type ModuleLogsGetResponse struct {
	// NextPageToken is the token to retrieve the next page of logs.
	NextPageToken string `json:"next_page_token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"`
	// ModuleLogs is the list of log entries for the module.
	ModuleLogs []api.ModuleLog `json:"logs"`
}

// Define the model for the API GET User response
// @Description API User model
type ModulePagesGetResponse struct {
	ModulePages []api.ModulePage `json:"pages"`
	NextPage    string           `json:"next_page_token" example:"BAD87as"`
}

// ModulePageSessionResponse represents the payload returned when minting a module session token.
type ModulePageSessionResponse struct {
	Token     string    `json:"token" example:"eyJzaWQiOiJzZXNzaW9uIn0.MEUCIQ..."`
	ExpiresAt time.Time `json:"expires_at" example:"2026-01-19T15:42:00Z"`
}

// ModuleNetworksResponse lists docker networks detected for a module.
type ModuleNetworksResponse struct {
	Networks []string `json:"networks"`
}

// ConfigResponse is the wrapper for a module’s YAML config.
// swagger:model ConfigResponse
type ConfigResponse struct {
	// Config is the raw module.yml content, with newlines preserved
	Config string `json:"config" example:"foo: bar\nbaz: qux\n"`
}

// ModulePageUpdateInput defines the fields you can patch on a module page.
// swagger:model ModulePageUpdateInput
type ModulePageUpdateInput struct {
	// Name is the new name for the page.
	Name *string `json:"name,omitempty" example:"Home"`

	// TargetContainer is the docker container name this page proxies to.
	TargetContainer *string `json:"target_container,omitempty" example:"frontend"`

	// TargetPort is the port exposed by the container to proxy.
	TargetPort *int `json:"target_port,omitempty" example:"80"`

	// IframeOnly enforces that the page is only reachable from the Pan Bagnat iframe.
	IframeOnly *bool `json:"iframe_only,omitempty" example:"true"`

	// NeedAuth toggles whether authentication is required.
	NeedAuth *bool `json:"need_auth,omitempty" example:"true"`

	// NetworkName is the docker network to which the reverse proxy must attach.
	NetworkName *string `json:"network_name,omitempty" example:"piscine-monitor-net"`
}

// ModuleGitInput describes the payload for importing a new module.
// swagger:model ModuleGitInput
type ModuleGitInput struct {
	// Name is the human-readable title you want for this module.
	Name string `json:"name"      example:"Captain Hook"`
	// GitURL is the repository URL to clone the module from.
	GitURL string `json:"git_url"   example:"https://github.com/some-user/some-repo"`
	// GitBranch is the branch to check out. Defaults to "main" if omitted.
	GitBranch string `json:"git_branch,omitempty" example:"main"`
	// SSHKeyID allows reusing an existing SSH key managed by Pan Bagnat. Leave empty to generate a new one.
	SSHKeyID string `json:"ssh_key_id,omitempty" example:"ssh-key_01H..."`
}

// ModuleRemoteUpdateInput describes the payload for updating a module’s Git remote.
// swagger:model ModuleRemoteUpdateInput
type ModuleRemoteUpdateInput struct {
	// GitURL is the new Git repository URL
	GitURL string `json:"git_url" example:"https://github.com/some-user/some-repo.git"`
}

// ComposeRequest represents the payload for deploying a module’s configuration.
// swagger:model ComposeRequest
type ComposeRequest struct {
	// Config is the full module.yml content to deploy (newlines preserved)
	Config string `json:"config" example:"version: '3'\nservices:\n  app:\n    image: my-app:latest\n"`
}

// ModulePageInput describes the payload for creating a new module page.
// swagger:model ModulePageInput
type ModulePageInput struct {
	// Name is the slug/identifier for this page (e.g. "home")
	Name string `json:"name"      example:"home"`
	// TargetContainer is the docker container name this page proxies to.
	TargetContainer *string `json:"target_container,omitempty" example:"frontend"`
	// TargetPort is the port exposed by the container to proxy.
	TargetPort *int `json:"target_port,omitempty" example:"80"`
	// IframeOnly enforces iframe usage for the page.
	IframeOnly bool `json:"iframe_only" example:"true"`
	// NeedAuth controls whether authentication is required.
	NeedAuth bool `json:"need_auth" example:"true"`
	// NetworkName is the docker network the proxy should join for this page (optional)
	NetworkName *string `json:"network_name,omitempty" example:"piscine-monitor-net"`
}
