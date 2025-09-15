package modules

import (
    "backend/core"
    "encoding/json"
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
)

// GET /admin/modules/{moduleID}/git/status
func GitStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    st, err := core.GitStatusModule(mod)
    if err != nil {
        log.Printf("git status error for %s: %v\n", moduleID, err)
        http.Error(w, "git status failed", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(st)
}

// POST /admin/modules/{moduleID}/git/fetch
func GitFetch(w http.ResponseWriter, r *http.Request) {
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    if err := core.GitFetchModule(mod); err != nil {
        http.Error(w, "git fetch failed", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusAccepted)
}

// POST /admin/modules/{moduleID}/git/add { paths: []string }
func GitAdd(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    var body struct { Paths []string `json:"paths"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
    if err := core.GitAddPaths(mod, body.Paths); err != nil {
        http.Error(w, "git add failed", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/git/merge/continue
func GitMergeContinueHandler(w http.ResponseWriter, r *http.Request) {
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    if err := core.GitMergeContinue(mod); err != nil {
        http.Error(w, "merge continue failed", http.StatusConflict)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/git/merge/abort
func GitMergeAbortHandler(w http.ResponseWriter, r *http.Request) {
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    if err := core.GitMergeAbort(mod); err != nil {
        http.Error(w, "merge abort failed", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

