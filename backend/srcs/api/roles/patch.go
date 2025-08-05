package roles

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
)

// PatchRole updates the specified fields of a role.
// @Summary      Patch Role
// @Description  Updates the name, color, and/or default status of an existing role.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        roleID  path      string              true  "Role ID"
// @Param        input   body      RolePatchInput  true  "Fields to update"
// @Success      200     {object}  api.Role            "The updated role"
// @Failure      400     {string}  string              "Invalid role ID or JSON body"
// @Failure      404     {string}  string              "Role not found"
// @Failure      500     {string}  string              "Internal server error"
// @Router       /admin/roles/{roleID} [patch]
func PatchRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1️⃣ Extract and validate path param
	roleID := chi.URLParam(r, "roleID")
	if strings.TrimSpace(roleID) == "" {
		http.Error(w, "roleID is required", http.StatusBadRequest)
		return
	}

	// 2️⃣ Decode and clean input
	var input RolePatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Name != nil {
		*input.Name = strings.TrimSpace(*input.Name)
	}
	if input.Color != nil {
		*input.Color = strings.TrimSpace(*input.Color)
	}
	// no trimming needed for *bool

	// 3️⃣ Perform the update (core.UpdateRole must accept the new flag)
	// updated, err := core.UpdateRole(roleID, input.Name, input.Color, input.IsDefault)
	// if err != nil {
	// 	if errors.Is(err, core.ErrNotFound) {
	// 		http.Error(w, "Role not found", http.StatusNotFound)
	// 	} else {
	// 		log.Printf("error updating role %s: %v\n", roleID, err)
	// 		http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	}
	// 	return
	// }

	// // 4️⃣ Convert to API model & respond
	// apiRole := api.RoleToAPIRole(updated)
	// if err := json.NewEncoder(w).Encode(apiRole); err != nil {
	// 	log.Printf("error encoding updated role %s: %v\n", roleID, err)
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// }
}
