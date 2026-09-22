package redirections

import (
	"backend/core"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type RedirectionUpdateInput struct {
	Name      *string `json:"name,omitempty"`
	Slug      *string `json:"slug,omitempty"`
	TargetURL *string `json:"target_url,omitempty"`
	NeedAuth  *bool   `json:"need_auth,omitempty"`
	IsVisible *bool   `json:"is_visible,omitempty"`
}

// PatchRedirection updates an existing redirection.
// @Summary      Patch Redirection
// @Description  Updates the specified fields of a redirection.
// @Tags         Redirections
// @Accept       json
// @Produce      json
// @Param        redirectionID  path  string                   true  "Redirection ID"
// @Param        input          body  RedirectionUpdateInput   true  "Fields to update"
// @Success      200  {object}  core.Redirection
// @Failure      400  {string}  string  "Invalid redirection ID or JSON body"
// @Failure      404  {string}  string  "Redirection not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /admin/redirections/{redirectionID} [patch]
func PatchRedirection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	redirectionID := chi.URLParam(r, "redirectionID")
	if strings.TrimSpace(redirectionID) == "" {
		http.Error(w, "redirectionID is required", http.StatusBadRequest)
		return
	}

	var input RedirectionUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Name != nil {
		*input.Name = strings.TrimSpace(*input.Name)
		if *input.Name == "" {
			http.Error(w, "name cannot be empty", http.StatusBadRequest)
			return
		}
	}
	if input.Slug != nil {
		*input.Slug = strings.TrimSpace(*input.Slug)
	}

	updated, err := core.UpdateRedirection(redirectionID, input.Name, input.Slug, input.TargetURL, input.NeedAuth, input.IsVisible)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(updated); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
