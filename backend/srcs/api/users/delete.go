package users

import (
	"backend/core"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DeleteUser deletes a user by ID.
// @Summary      Delete User
// @Description  Deletes the specified user and all associated data.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        userID  path      string  true  "User ID"
// @Success      204     {string}  string  "No Content"
// @Failure      400     {string}  string  "Invalid user ID"
// @Failure      404     {string}  string  "User not found"
// @Failure      500     {string}  string  "Internal server error"
// @Router       /admin/users/{userID} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(r, "userID")
	if strings.TrimSpace(userID) == "" {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// err := core.DeleteUser(userID)
	// if err != nil {
	// 	if errors.Is(err, core.ErrNotFound) {
	// 		http.Error(w, "User not found", http.StatusNotFound)
	// 	} else {
	// 		log.Printf("error deleting user %s: %v\n", userID, err)
	// 		http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	}
	// 	return
	// }

	w.WriteHeader(http.StatusNoContent)

}

// DeleteUserRole revokes a specific role from a user.
// @Summary      Remove Role from User
// @Description  Revokes the specified role from the user identified by ID or login.
// @Tags         Users,Roles
// @Accept       json
// @Produce      json
// @Param        identifier  path      string  true  "User identifier (ID or login)"
// @Param        roleID      path      string  true  "Role ID"
// @Success      204         {string}  string  "No Content"
// @Failure      400         {string}  string  "Invalid identifier or roleID"
// @Failure      404         {string}  string  "User or role not found"
// @Failure      500         {string}  string  "Internal server error"
// @Router       /admin/users/{identifier}/roles/{roleID} [delete]
func DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	roleID := chi.URLParam(r, "roleID")

	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(roleID) == "" {
		http.Error(w, "Invalid identifier or roleID", http.StatusBadRequest)
		return
	}

	err := core.DeleteRoleFromUser(roleID, identifier)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotFound):
			http.Error(w, "User or role not found", http.StatusNotFound)
		default:
			log.Printf("error deleting role %s from user %s: %v\n", roleID, identifier, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
