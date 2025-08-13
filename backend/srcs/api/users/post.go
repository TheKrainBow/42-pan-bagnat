package users

import (
	"backend/api/auth"
	"backend/core"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// PostUser creates a new user for your campus.
// @Summary      Create User
// @Description  Imports a new user with the given 42-intranet details and optional role assignments.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        input  body      UserPostInput  true  "User creation payload"
// @Success      200    {object}  api.User           "The newly created user"
// @Failure      400    {string}  string             "Invalid JSON input or missing required fields"
// @Failure      500    {string}  string             "Internal server error"
// @Router       /admin/users [post]
func PostUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse and validate input
	var input UserPostInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.FtLogin) == "" || input.FtID <= 0 {
		http.Error(w, "Missing ft_id or ft_login", http.StatusBadRequest)
		return
	}

	// user, err := core.CreateUser(input.FtID, input.FtLogin, input.FtPhoto, input.IsStaff, input.Roles)
	// if err != nil {
	// 	log.Printf("failed to create user: %v", err)
	// 	http.Error(w, "Failed to create user", http.StatusInternalServerError)
	// 	return
	// }

	// // Convert to API model & respond
	// resp := api.UserToAPIUser(*user)
	// if err := json.NewEncoder(w).Encode(resp); err != nil {
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// 	return
	// }
}

// PostUserRole grants a role to a user.
// @Summary      Add Role to User
// @Description  Assigns the specified role to the user identified by ID or login.
// @Tags         Users,Roles
// @Accept       json
// @Produce      json
// @Param        identifier  path      string  true   "User identifier (ID or login)"
// @Param        roleID      path      string  true   "Role ID"
// @Success      201         {string}  string  "Role successfully assigned to user"
// @Failure      400         {string}  string  "Invalid identifier or roleID"
// @Failure      404         {string}  string  "User or role not found"
// @Failure      500         {string}  string  "Internal server error"
// @Router       /admin/users/{identifier}/roles/{roleID} [post]
func PostUserRole(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	roleID := chi.URLParam(r, "roleID")

	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(roleID) == "" {
		http.Error(w, "Invalid identifier or roleID", http.StatusBadRequest)
		return
	}

	err := core.AddRoleToUser(roleID, identifier)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotFound):
			http.Error(w, "User or role not found", http.StatusNotFound)
		case errors.Is(err, core.ErrRoleAlreadyAssigned):
			http.Error(w, "Role already assigned to user", http.StatusConflict)
		case errors.Is(err, core.ErrWouldBlacklistLastAdmin):
			auth.WriteJSONError(w, http.StatusConflict, "Conflict", "Cannot blacklist the last admin user")
		default:
			log.Printf("error assigning role %s to user %s: %v\n", roleID, identifier, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Role %s successfully assigned to user %s", roleID, identifier)
}
