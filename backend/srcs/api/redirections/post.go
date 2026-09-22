package redirections

import (
	"backend/core"
	"encoding/json"
	"net/http"
	"strings"
)

type RedirectionCreateInput struct {
	Name      string  `json:"name"`
	Slug      *string `json:"slug,omitempty"`
	TargetURL string  `json:"target_url"`
	NeedAuth  *bool   `json:"need_auth,omitempty"`
	IsVisible *bool   `json:"is_visible,omitempty"`
}

// PostRedirection creates a new redirection.
// @Summary      Create Redirection
// @Description  Creates a new redirection page that redirects the browser to an external URL.
// @Tags         Redirections
// @Accept       json
// @Produce      json
// @Param        input  body      RedirectionCreateInput  true  "Redirection creation payload"
// @Success      200    {object}  core.Redirection
// @Failure      400    {string}  string  "Invalid JSON input or missing required fields"
// @Failure      500    {string}  string  "Internal server error"
// @Router       /admin/redirections [post]
func PostRedirection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input RedirectionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	if input.Name = strings.TrimSpace(input.Name); input.Name == "" {
		http.Error(w, "Missing field name", http.StatusBadRequest)
		return
	}

	needAuth := true
	if input.NeedAuth != nil {
		needAuth = *input.NeedAuth
	}
	isVisible := true
	if input.IsVisible != nil {
		isVisible = *input.IsVisible
	}

	redirection, err := core.CreateRedirection(input.Name, input.Slug, input.TargetURL, needAuth, isVisible)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(redirection); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
