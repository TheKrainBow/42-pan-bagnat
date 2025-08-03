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

// @Summary      Post User List
// @Description  Download a new user for your campus
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        input body UserPostInput true "User input"
// @Success      200 {object} User
// @Router       /users [post]
func PostUser(w http.ResponseWriter, r *http.Request) {
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

// @Summary      Add role to user
// @Description  Grants a role to a user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        identifier path string true "User identifier"
// @Param        role_id path string true "Role ID"
// @Success      204
// @Failure      400 {object} api.ErrorResponse
// @Router       /users/{identifier}/roles/{role_id} [post]
func PostUserRole(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	roleID := chi.URLParam(r, "role_id")

	err := core.AddRoleToUser(roleID, identifier)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assign role: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error assigning role %s to user %s: %v\n", roleID, identifier, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Role %s successfully assigned to user %s\n", roleID, identifier)
}
