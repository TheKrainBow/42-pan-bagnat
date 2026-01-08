package roles

import (
	"backend/core"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type validateRequest struct {
	Rules any `json:"rules"`
}

type validateError struct {
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type validateResponse struct {
	Ok        bool            `json:"ok"`
	Errors    []validateError `json:"errors,omitempty"`
	Canonical any             `json:"canonical,omitempty"`
}

// ValidateRoleRules checks the rule JSON shape and returns a canonical version without persisting.
// @Summary Validate Role Rules
// @Description Validate and canonicalize the rule tree without saving.
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param input body validateRequest true "Rules payload"
// @Success 200 {object} validateResponse
// @Failure 400 {string} string "Invalid input"
// @Router /admin/roles/{roleID}/rules/validate [post]
func ValidateRoleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	roleID := chi.URLParam(r, "roleID")
	if strings.TrimSpace(roleID) == "" {
		http.Error(w, "Missing roleID", http.StatusBadRequest)
		return
	}

	var req validateRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.Rules == nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	// Encode to bytes then canonicalize via core
	raw, err := json.Marshal(req.Rules)
	if err != nil {
		http.Error(w, "Invalid rules payload", http.StatusBadRequest)
		return
	}
	can, err := core.CanonicalizeRoleRulesJSON(raw)
	if err != nil {
		_ = json.NewEncoder(w).Encode(validateResponse{
			Ok:     false,
			Errors: []validateError{{Message: err.Error()}},
		})
		return
	}

	var canonical any
	_ = json.Unmarshal(can, &canonical)
	_ = json.NewEncoder(w).Encode(validateResponse{Ok: true, Canonical: canonical})
}
