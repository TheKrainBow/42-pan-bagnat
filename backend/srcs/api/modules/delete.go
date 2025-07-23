package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

// @Summary      Delete Module
// @Description  Delete a module for your campus (All module datas will be lost!)
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body ModulePatchInput true "Module input"
// @Success      200
// @Router       /modules [delete]
func DeleteModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := api.Module{
		ID:            id.String(),
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

// @Summary      Delete Module page
// @Description  Delete a module page for your campus (All page datas will be lost!)
// @Tags         pages
// @Accept       json
// @Produce      json
// @Param        input body ModulePatchInput true "Module input"
// @Success      200
// @Router       /modules/{moduleID}/pages/{pageID} [delete]
func DeleteModulePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pageID := chi.URLParam(r, "pageID")
	moduleID := chi.URLParam(r, "moduleID")

	if pageID == "" {
		http.Error(w, "Missing field page_name", http.StatusBadRequest)
		return
	}

	err := core.DeleteModulePage(pageID)
	if err != nil {
		http.Error(w, "error while deleting module page", http.StatusInternalServerError)
	}

	core.LogModule(moduleID, "INFO", fmt.Sprintf("Deleted page '%s'", pageID), nil, nil)
	fmt.Fprint(w, "OK")
}
