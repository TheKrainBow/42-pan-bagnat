package modules

import (
    "backend/core"
    "encoding/json"
    "net/http"
    "os/exec"

    "github.com/go-chi/chi/v5"
)

// GET /admin/modules/{moduleID}/git/commit/current
func GitCurrentCommit(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    repoDir := core.HostRepoDir(mod)
    _ = core.EnsureSafeDir(mod, repoDir)
    hashB, _ := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").CombinedOutput()
    subjB, _ := exec.Command("git", "-C", repoDir, "log", "-1", "--pretty=%s").CombinedOutput()
    resp := map[string]string{
        "hash":   string(core.BytesTrimSpace(hashB)),
        "subject": string(core.BytesTrimSpace(subjB)),
    }
    _ = json.NewEncoder(w).Encode(resp)
}

// GET /admin/modules/{moduleID}/git/commit/latest
func GitLatestCommit(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    moduleID := chi.URLParam(r, "moduleID")
    mod, err := core.GetModule(moduleID)
    if err != nil { http.Error(w, "module not found", http.StatusNotFound); return }
    repoDir := core.HostRepoDir(mod)
    _ = core.EnsureSafeDir(mod, repoDir)
    // Determine upstream
    up := ""
    if b, e := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").CombinedOutput(); e == nil {
        up = string(core.BytesTrimSpace(b))
    }
    if up == "" && mod.GitBranch != "" { up = "origin/" + mod.GitBranch }
    if up == "" {
        http.Error(w, "no upstream branch", http.StatusConflict)
        return
    }
    // Fetch the ref to ensure online view
    _ = exec.Command("git", "-C", repoDir, "fetch", "--all", "--prune").Run()
    line, _ := exec.Command("git", "-C", repoDir, "log", "-1", up, "--pretty=%H%x1f%s").CombinedOutput()
    s := string(core.BytesTrimSpace(line))
    hash, subj := "", ""
    if s != "" {
        cur := ""
        arr := []string{}
        for i := 0; i < len(s); i++ { if s[i]==0x1f { arr = append(arr, cur); cur = "" } else { cur += string(s[i]) } }
        arr = append(arr, cur)
        if len(arr) >= 2 { hash = arr[0]; subj = arr[1] }
    }
    _ = json.NewEncoder(w).Encode(map[string]string{"hash": hash, "subject": subj})
}

