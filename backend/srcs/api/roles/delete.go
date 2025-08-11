package roles

import (
	"backend/core"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DeleteRole deletes a role by its ID.
// @Summary      Delete Role
// @Description  Deletes the specified role for your campus (all role data will be lost).
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        roleID  path      string  true  "Role ID"
// @Success      204      "No Content"
// @Failure      400      {string}  string  "Invalid role ID"
// @Failure      404      {string}  string  "Role not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /admin/roles/{roleID} [delete]
func DeleteRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	roleID := chi.URLParam(r, "roleID")
	if strings.TrimSpace(roleID) == "" {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	err := core.DeleteRole(roleID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
		} else {
			log.Printf("error deleting role %s: %v\n", roleID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
