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

	// Slug is the hostname segment used to expose the page.
	Slug *string `json:"slug,omitempty" example:"watchdog"`

	// TargetContainer is the docker container name this page proxies to.
	TargetContainer *string `json:"target_container,omitempty" example:"frontend"`

	// TargetPort is the port exposed by the container to proxy.
	TargetPort *int `json:"target_port,omitempty" example:"80"`

	// IframeOnly enforces that the page is only reachable from the Pan Bagnat iframe.
	IframeOnly *bool `json:"iframe_only,omitempty" example:"true"`

	// PageOnly enforces that the page is only reachable directly on the module domain.
	PageOnly *bool `json:"page_only,omitempty" example:"true"`

	// NeedAuth toggles whether authentication is required.
	NeedAuth *bool `json:"need_auth,omitempty" example:"true"`

	// IsVisible controls whether the page appears in user-facing navigation.
	IsVisible *bool `json:"is_visible,omitempty" example:"true"`

	// NetworkName is the docker network to which the reverse proxy must attach.
	NetworkName *string `json:"network_name,omitempty" example:"piscine-monitor-net"`

	// MaxUploadBodySize is the nginx-style max request body size accepted by
	// this page's gateway (e.g. "1m", "50m"). Defaults to "1m" when unset.
	MaxUploadBodySize *string `json:"max_upload_body_size,omitempty" example:"10m"`

	// ProxyTimeoutSeconds is the read/send timeout applied by this page's gateway.
	// Defaults to 60s when unset.
	ProxyTimeoutSeconds *int `json:"proxy_timeout_seconds,omitempty" example:"60"`

	// RateLimitRPS is the max requests/second allowed per client IP. 0 or unset
	// disables rate limiting.
	RateLimitRPS *int `json:"rate_limit_rps,omitempty" example:"0"`

	// RateLimitBurst is the burst allowance on top of RateLimitRPS.
	RateLimitBurst *int `json:"rate_limit_burst,omitempty" example:"0"`

	// DisableRequestBuffering streams uploads directly to the module instead of
	// buffering the full request body first.
	DisableRequestBuffering *bool `json:"disable_request_buffering,omitempty" example:"false"`
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
	// Name is the human-readable label displayed in the admin.
	Name string `json:"name" example:"Watchdog"`
	// Slug is the hostname segment used to expose the page. If omitted, it is generated from the name.
	Slug *string `json:"slug,omitempty" example:"watchdog"`
	// TargetContainer is the docker container name this page proxies to.
	TargetContainer *string `json:"target_container,omitempty" example:"frontend"`
	// TargetPort is the port exposed by the container to proxy.
	TargetPort *int `json:"target_port,omitempty" example:"80"`
	// IframeOnly enforces iframe usage for the page.
	IframeOnly bool `json:"iframe_only" example:"true"`
	// PageOnly enforces direct access on the module domain.
	PageOnly bool `json:"page_only" example:"false"`
	// NeedAuth controls whether authentication is required.
	NeedAuth bool `json:"need_auth" example:"true"`
	// IsVisible controls whether the page appears in user-facing navigation.
	IsVisible bool `json:"is_visible" example:"true"`
	// NetworkName is the docker network the proxy should join for this page (optional)
	NetworkName *string `json:"network_name,omitempty" example:"piscine-monitor-net"`
	// MaxUploadBodySize is the nginx-style max request body size accepted by
	// this page's gateway (e.g. "1m", "50m"). Defaults to "1m" when omitted.
	MaxUploadBodySize *string `json:"max_upload_body_size,omitempty" example:"10m"`

	// ProxyTimeoutSeconds is the read/send timeout applied by this page's gateway.
	// Defaults to 60s when omitted.
	ProxyTimeoutSeconds *int `json:"proxy_timeout_seconds,omitempty" example:"60"`

	// RateLimitRPS is the max requests/second allowed per client IP. 0 or
	// omitted disables rate limiting.
	RateLimitRPS *int `json:"rate_limit_rps,omitempty" example:"0"`

	// RateLimitBurst is the burst allowance on top of RateLimitRPS.
	RateLimitBurst *int `json:"rate_limit_burst,omitempty" example:"0"`

	// DisableRequestBuffering streams uploads directly to the module instead of
	// buffering the full request body first.
	DisableRequestBuffering *bool `json:"disable_request_buffering,omitempty" example:"false"`
}
