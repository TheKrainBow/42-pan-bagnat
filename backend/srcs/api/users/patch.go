package users

import (
	"backend/core"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// @Summary      Patch user (staff only)
// @Description  Modify specific user fields like is_staff
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        identifier path string true "User ID or login"
// @Param        input body api.UserPatchInput true "Patch payload"
// @Success      200 {object} api.User
// @Failure      400 {object} api.ErrorResponse
// @Failure      404 {object} api.ErrorResponse
// @Router       /admin/users/{identifier} [patch]
func PatchUser(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	w.Header().Set("Content-Type", "application/json")

	var input core.UserPatch
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Always store the resolved ID inside the patch struct
	id, err := core.ResolveUserIdentifier(identifier)
	if err != nil {
		http.Error(w, fmt.Sprintf("User not found: %v", err), http.StatusNotFound)
		return
	}
	input.ID = id

	updated, err := core.PatchUser(input)
	if err != nil || updated == nil {
		http.Error(w, fmt.Sprintf("Failed to patch user: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(*updated)
}
