package modules

import (
	api "backend/api/dto"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

// @Summary      Post Module List
// @Description  Download a new module for your campus
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body ModulePatchInput true "Module input"
// @Success      200 {object} Module
// @Router       /modules/{moduleID} [patch]
func PatchModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "moduleID")

	dest := api.Module{
		ID:            id,
		Name:          "Test",
		Version:       "1.2",
		Status:        api.Enabled,
		GitURL:        "https://github.com/some-user/some-repo",
		LatestVersion: "1.7",
		LastUpdate:    time.Date(2025, 02, 18, 15, 0, 0, 0, time.UTC),
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
