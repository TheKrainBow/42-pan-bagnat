package users

import (
	"backend/api/auth"
	"backend/core"
	"backend/database"
	"encoding/json"
	"net/http"
)

const sidebarPrefKey = "sidebar_prefs"

// GET /api/v1/users/me/prefs/sidebar
func GetSidebarPrefs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
	if !ok || u == nil || u.ID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	raw, err := database.GetUserPref(r.Context(), u.ID, sidebarPrefKey)
	if err != nil {
		// default prefs if missing
		w.Write([]byte(`{"order":[],"hidden":{}}`))
		return
	}
	w.Write(raw)
}

// PUT /api/v1/users/me/prefs/sidebar
func PutSidebarPrefs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
	if !ok || u == nil || u.ID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// minimal validation: expect object with keys order(array) and hidden(object)
	if _, ok := payload["order"].([]interface{}); !ok {
		payload["order"] = []string{}
	}
	if _, ok := payload["hidden"].(map[string]interface{}); !ok {
		payload["hidden"] = map[string]interface{}{}
	}
	raw, _ := json.Marshal(payload)
	if err := database.PutUserPref(r.Context(), u.ID, sidebarPrefKey, raw); err != nil {
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}
	w.Write(raw)
}
