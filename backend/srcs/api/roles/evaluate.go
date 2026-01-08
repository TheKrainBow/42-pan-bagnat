package roles

import (
	"backend/core"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type evaluateRequest struct {
	Rules   any `json:"rules"`
	Payload any `json:"payload"`
}

type evaluateResponse struct {
	Matched   bool           `json:"matched"`
	Canonical any            `json:"canonical,omitempty"`
	Trace     core.TraceNode `json:"trace"`
}

// EvaluateRoleRules evaluates a rule tree against a single payload (e.g., a user JSON) without saving.
// @Summary Evaluate Role Rules against a payload
// @Description Returns whether the provided payload matches the rule tree. Does not persist.
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param input body evaluateRequest true "Rules and payload"
// @Success 200 {object} evaluateResponse
// @Failure 400 {string} string "Invalid input"
// @Router /admin/roles/{roleID}/rules/evaluate [post]
func EvaluateRoleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	roleID := chi.URLParam(r, "roleID")
	if strings.TrimSpace(roleID) == "" {
		http.Error(w, "Missing roleID", http.StatusBadRequest)
		return
	}

	var req evaluateRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.Rules == nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	rawRules, err := json.Marshal(req.Rules)
	if err != nil {
		http.Error(w, "Invalid rules payload", http.StatusBadRequest)
		return
	}

	matched, canonical, trace, err := core.EvaluateRoleRulesJSONTrace(rawRules, req.Payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var can any
	_ = json.Unmarshal(canonical, &can)
	_ = json.NewEncoder(w).Encode(evaluateResponse{Matched: matched, Canonical: can, Trace: trace})
}
