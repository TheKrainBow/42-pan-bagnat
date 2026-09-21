package stats

import (
	"backend/core"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", getGlobalStats)
	r.Get("/modules/{moduleID}", getModuleStats)
}

func getGlobalStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	from, to, err := core.ParseStatsRange(r)
	if err != nil {
		http.Error(w, "invalid from/to date", http.StatusBadRequest)
		return
	}
	result, err := core.GetGlobalStats(from, to)
	if err != nil {
		http.Error(w, "failed to load stats", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func getModuleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	from, to, err := core.ParseStatsRange(r)
	if err != nil {
		http.Error(w, "invalid from/to date", http.StatusBadRequest)
		return
	}
	result, err := core.GetModuleStats(moduleID, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(result)
}
