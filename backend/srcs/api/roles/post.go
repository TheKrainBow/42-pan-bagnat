package roles

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// PostRole creates a new role for your campus.
// @Summary      Create Role
// @Description  Imports a new role with the given name, color, default status, and module assignments.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        input  body      RoleCreateInput  true  "Role creation payload"
// @Success      200    {object}  api.Role         "The newly created role"
// @Failure      400    {string}  string           "Invalid JSON input or missing required fields"
// @Failure      500    {string}  string           "Internal server error"
// @Router       /admin/roles [post]
func PostRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse input
	var input struct {
		Name      string   `json:"name"`
		Color     string   `json:"color"`
		IsDefault bool     `json:"is_default"`
		Modules   []string `json:"modules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Color) == "" {
		http.Error(w, "Missing name or color", http.StatusBadRequest)
		return
	}

	role, err := core.ImportRole(input.Name, input.Color, input.IsDefault, input.Modules)
	if err != nil {
		log.Printf("failed to import role: %v", err)
		http.Error(w, "Failed to import role", http.StatusInternalServerError)
		return
	}

	dest := api.RoleToAPIRole(role)
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
