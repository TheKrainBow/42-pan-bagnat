package modules

import (
	"backend/core"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GET /admin/modules/{moduleID}/git/behind
func GitBehind(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	behind, err := core.GitBehind(mod)
	if err != nil {
		http.Error(w, "failed to compute behind", http.StatusInternalServerError)
		return
	}
	logSSHKeyUsage(r, mod, "git behind")
	_ = json.NewEncoder(w).Encode(map[string]int{"behind": behind})
}
