package sshkeys

import (
	"backend/api/auth"
	"backend/core"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", listSSHKeys)
	r.Post("/", createSSHKey)
	r.Get("/{sshKeyID}/modules", getSSHKeyModules)
	r.Get("/{sshKeyID}/events", getSSHKeyEvents)
	r.Post("/{sshKeyID}/regenerate", regenerateSSHKey)
	r.Delete("/{sshKeyID}", deleteSSHKey)
}

func listSSHKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	keys, err := core.ListSSHKeys()
	if err != nil {
		http.Error(w, "Failed to list SSH keys", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"ssh_keys": keys})
}

func createSSHKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var input SSHKeyCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	var userID *string
	var actor *core.User
	if u, ok := r.Context().Value(auth.UserCtxKey).(*core.User); ok && u != nil {
		actor = u
		userID = &u.ID
	}
	key, err := core.CreateSSHKey(input.Name, input.SSHPrivateKey, userID, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if actor != nil {
		_ = core.AppendSSHKeyEvent(key.ID, actor, nil, "ssh key created")
	}
	json.NewEncoder(w).Encode(key)
}

func regenerateSSHKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sshKeyID := chi.URLParam(r, "sshKeyID")
	var input SSHKeyRegenerateInput
	_ = json.NewDecoder(r.Body).Decode(&input)
	var userID *string
	var actor *core.User
	if u, ok := r.Context().Value(auth.UserCtxKey).(*core.User); ok && u != nil {
		actor = u
		userID = &u.ID
	}
	key, err := core.RegenerateSSHKey(sshKeyID, input.SSHPrivateKey, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if actor != nil {
		_ = core.AppendSSHKeyEvent(key.ID, actor, nil, "ssh key regenerated")
	}
	json.NewEncoder(w).Encode(key)
}

func getSSHKeyModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sshKeyID := chi.URLParam(r, "sshKeyID")
	mods, err := core.GetModulesBySSHKey(sshKeyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"modules": mods})
}

func deleteSSHKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sshKeyID := chi.URLParam(r, "sshKeyID")
	err := core.DeleteSSHKey(sshKeyID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			http.Error(w, "SSH key is still assigned to modules", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getSSHKeyEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sshKeyID := chi.URLParam(r, "sshKeyID")
	events, err := core.ListSSHKeyEvents(sshKeyID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"events": events})
}
