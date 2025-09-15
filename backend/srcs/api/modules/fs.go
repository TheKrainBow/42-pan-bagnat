package modules

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
    "backend/core"
    "strings"
)

// GET /admin/modules/{moduleID}/fs/tree?path=.
func GetFsTree(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    rel := r.URL.Query().Get("path")
    entries, err := core.ListModuleDir(mod, rel)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    json.NewEncoder(w).Encode(entries)
}

// GET /admin/modules/{moduleID}/fs/read?path=...
func ReadFsFile(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    rel := r.URL.Query().Get("path")
    data, err := core.ReadModuleFile(mod, rel)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    json.NewEncoder(w).Encode(map[string]string{"content": string(data)})
}

// POST /admin/modules/{moduleID}/fs/write  { path, content }
func WriteFsFile(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    var body struct { Path string `json:"path"`; Content string `json:"content"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
    if err := core.WriteModuleFile(mod, body.Path, []byte(body.Content)); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/fs/rename  { old_path, new_path }
func RenameFsPath(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    var body struct { OldPath string `json:"old_path"`; NewPath string `json:"new_path"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
    if err := core.RenameModulePath(mod, body.OldPath, body.NewPath); err != nil {
        status := http.StatusBadRequest
        if strings.Contains(strings.ToLower(err.Error()), "exists") {
            status = http.StatusConflict
        }
        http.Error(w, err.Error(), status)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/fs/delete  { path }
func DeleteFsPath(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    var body struct { Path string `json:"path"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
    if err := core.DeleteModulePath(mod, body.Path); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusNoContent)
}

// POST /admin/modules/{moduleID}/fs/mkdir  { path }
func MkdirFsPath(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    var body struct { Path string `json:"path"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
    if err := core.MkdirModule(mod, body.Path); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusNoContent)
}

// GET /admin/modules/{moduleID}/fs/root
func GetFsRoot(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    root, err := core.HostModuleRepoRoot(mod)
    if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
    json.NewEncoder(w).Encode(map[string]string{"abs_path": root})
}
