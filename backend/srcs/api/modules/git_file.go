package modules

import (
	"backend/core"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// POST /admin/modules/{moduleID}/git/file/checkout { path, ref }
func GitFileCheckout(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	var body struct {
		Path string `json:"path"`
		Ref  string `json:"ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := core.GitCheckoutFile(mod, body.Path, body.Ref); err != nil {
		http.Error(w, "checkout file failed", http.StatusInternalServerError)
		return
	}
	logSSHKeyUsage(r, mod, fmt.Sprintf("git checkout -- %s", body.Path))
	w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/git/file/resolve/ours { path }
func GitFileResolveOurs(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := core.GitResolveOurs(mod, body.Path); err != nil {
		http.Error(w, "resolve ours failed", http.StatusInternalServerError)
		return
	}
	logSSHKeyUsage(r, mod, fmt.Sprintf("git checkout --ours %s", body.Path))
	w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/git/file/resolve/theirs { path }
func GitFileResolveTheirs(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	mod, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := core.GitResolveTheirs(mod, body.Path); err != nil {
		http.Error(w, "resolve theirs failed", http.StatusInternalServerError)
		return
	}
	logSSHKeyUsage(r, mod, fmt.Sprintf("git checkout --theirs %s", body.Path))
	w.WriteHeader(http.StatusNoContent)
}
