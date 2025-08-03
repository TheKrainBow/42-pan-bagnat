package users

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

// @Summary      Delete User
// @Description  Delete a user for your campus (All user datas will be lost!)
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200
// @Router       /users [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := api.User{
		ID: id.String(),
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Summary      Remove role from user
// @Description  Revokes a specific role from a user (by login or ID)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        identifier path string true "User identifier (ID or login)"
// @Param        role_id path string true "Role ID"
// @Success      204 "Role successfully removed"
// @Failure      500 {object} api.ErrorResponse "Server error or user not found"
// @Router       /users/{identifier}/roles/{role_id} [delete]
func DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	roleID := chi.URLParam(r, "role_id")

	err := core.DeleteRoleFromUser(roleID, identifier)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete role: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error delete role %s from user %s: %v\n", roleID, identifier, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "Role %s successfully deleted role from user %s\n", roleID, identifier)
}
