package users

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// PatchUser modifies specific fields of a user (admin only).
// @Summary      Patch User (staff only)
// @Description  Modify specific user fields such as staff status or assigned roles.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        identifier  path      string             true  "User ID or login"
// @Param        input       body      UserPatchInput true  "Fields to update"
// @Success      200         {object}  api.User          "The updated user object"
// @Failure      400         {string}  string            "Invalid JSON or bad input"
// @Failure      404         {string}  string            "User not found"
// @Failure      500         {string}  string            "Internal server error"
// @Router       /admin/users/{identifier} [patch]
func PatchUser(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	w.Header().Set("Content-Type", "application/json")

	var input UserPatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Resolve identifier → internal ID
	id, err := core.ResolveUserIdentifier(identifier)
	if err != nil {
		http.Error(w, fmt.Sprintf("User not found: %v", err), http.StatusNotFound)
		return
	}

	// Build core patch struct from API input
	patch := core.UserPatch{
		ID:    id,
		Roles: input.Roles,
	}

	updated, err := core.PatchUser(patch)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to patch user: %v", err), http.StatusInternalServerError)
		return
	}

	apiUser := api.UserToAPIUser(*updated)
	json.NewEncoder(w).Encode(apiUser)
}
