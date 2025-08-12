package roles

import (
	"backend/core"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// PutRoleRules updates the assignment rules for a role and can optionally apply them to existing users.
// @Summary      Update Role Rules
// @Description  Replace the conditions that assign the role. Optionally apply to existing users immediately.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        roleID  path      int                       true  "Role ID"
// @Param        input   body      RoleRulesUpdateInput      true  "Rules payload"
// @Success      200     {object}  RoleRulesUpdateResponse   "Saved rules"
// @Failure      400     {string}  string                    "Invalid input"
// @Failure      404     {string}  string                    "Role not found"
// @Failure      500     {string}  string                    "Internal server error"
// @Router       /admin/roles/{roleID}/rules [put]
func PutRoleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type RoleRulesUpdateInput struct {
		// The rule tree built by the UI (group/array/scalar nodes).
		// We accept it as a generic object to keep the handler decoupled from core structs.
		Rules           map[string]interface{} `json:"rules"`
		ApplyToExisting bool                   `json:"applyToExisting"`
	}

	type RoleRulesUpdateResponse struct {
		RoleID            string                 `json:"role_id"`
		Rules             map[string]interface{} `json:"rules"`
		AppliedToExisting bool                   `json:"applied_to_existing"`
		UpdatedUsersCount int                    `json:"updated_users_count,omitempty"`
	}

	// Extract roleID
	roleID := chi.URLParam(r, "roleID")
	if strings.TrimSpace(roleID) == "" {
		http.Error(w, "Missing roleID", http.StatusBadRequest)
		return
	}

	// Decode body
	var input RoleRulesUpdateInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	if input.Rules == nil {
		http.Error(w, "rules is required", http.StatusBadRequest)
		return
	}

	// Marshal the generic map back to JSON for storage in core.
	rulesJSON, err := json.Marshal(input.Rules)
	if err != nil {
		http.Error(w, "Failed to encode rules", http.StatusBadRequest)
		return
	}

	if err := core.SetRoleRulesJSON(r.Context(), roleID, rulesJSON); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
			return
		}
		log.Printf("SetRoleRulesJSON failed (role %s): %v", roleID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Optionally apply to existing users now.
	updated := 0
	applied := false
	if input.ApplyToExisting {
		n, err := core.ApplyRoleRulesNow(r.Context(), roleID)
		if err != nil {
			// Application failure shouldn’t hide that rules were saved successfully.
			log.Printf("ApplyRoleRulesNow failed (role %s): %v", roleID, err)
			http.Error(w, "Rules saved, but failed to apply to existing users", http.StatusInternalServerError)
			return
		}
		applied = true
		updated = n
	}

	resp := RoleRulesUpdateResponse{
		RoleID:            roleID,
		Rules:             input.Rules,
		AppliedToExisting: applied,
		UpdatedUsersCount: updated,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
