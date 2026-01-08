package modules

import (
	"backend/core"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// GET /admin/modules/{moduleID}/git/commits?limit=50&ref=branchOrRemote
func GitCommits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 500 {
			limit = n
		}
	}
	ref := r.URL.Query().Get("ref")
	list, err := core.GitListCommitsRef(mod, ref, limit)
	if err != nil {
		http.Error(w, "git log failed", http.StatusInternalServerError)
		return
	}
	logSSHKeyUsage(r, mod, "git log")
	json.NewEncoder(w).Encode(list)
}

// GET /admin/modules/{moduleID}/git/branches
func GitBranches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	list, err := core.GitListBranches(mod)
	if err != nil {
		http.Error(w, "git branches failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	logSSHKeyUsage(r, mod, "git branch list")
	json.NewEncoder(w).Encode(list)
}

// POST /admin/modules/{moduleID}/git/checkout { ref }
func GitCheckout(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	var body struct {
		Ref string `json:"ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Ref == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := core.GitCheckout(mod, body.Ref); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	logSSHKeyUsage(r, mod, fmt.Sprintf("git checkout %s", body.Ref))

	// Refetch module to get fresh DB-backed fields (branch, commit snapshot)
	updated, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found after checkout", http.StatusInternalServerError)
		return
	}
	// Build live status and lists for immediate UI update
	st, _ := core.GitStatusModule(updated)
	branches, _ := core.GitListBranches(updated)
	// Use current branch name for commit list; fallback to DB branch
	ref := st.Branch
	if ref == "" || ref == "HEAD" {
		ref = updated.GitBranch
	}
	commits, _ := core.GitListCommitsRef(updated, ref, 20)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   st,
		"branches": branches,
		"commits":  commits,
	})
}

// POST /admin/modules/{moduleID}/git/branch { name, from }
func GitCreateBranch(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	var body struct {
		Name string `json:"name"`
		From string `json:"from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := core.GitCreateBranch(mod, body.Name, body.From); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	logSSHKeyUsage(r, mod, fmt.Sprintf("git branch %s", body.Name))
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /admin/modules/{moduleID}/git/branch { name }
func GitDeleteBranch(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := core.GitDeleteBranch(mod, body.Name); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	logSSHKeyUsage(r, mod, fmt.Sprintf("git branch -D %s", body.Name))
	w.WriteHeader(http.StatusNoContent)
}
