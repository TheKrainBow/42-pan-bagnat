package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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

// @Summary      Patch Module Page
// @Description  Update an existing front-page of a module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        moduleID path string true "Module ID"
// @Param        name     path string true "Existing page name"
// @Param        input    body ModulePageUpdateInput true "Fields to update"
// @Success      200      {object} api.ModulePage
// @Failure      400      {string} string "Bad request"
// @Failure      404      {string} string "Page not found"
// @Failure      500      {string} string "Internal error"
// @Router       /modules/{moduleID}/pages/{pageID} [patch]
func PatchModulePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleID := chi.URLParam(r, "moduleID")
	pageID := chi.URLParam(r, "pageID")
	if moduleID == "" || pageID == "" {
		log.Printf("%s | %s\n", moduleID, pageID)
		http.Error(w, "moduleID and pageID are required", http.StatusBadRequest)
		return
	}

	// parse body
	var input struct {
		ID       string  `json:"id"`
		Name     *string `json:"name"`
		URL      *string `json:"url"`
		IsPublic *bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	// trim & validate required fields
	if input.Name != nil {
		*input.Name = strings.TrimSpace(*input.Name)
	}

	if input.URL != nil {
		*input.URL = strings.TrimSpace(*input.URL)
	}

	// perform update
	modulePage, err := core.UpdateModulePage(pageID, input.Name, input.URL, input.IsPublic)
	if err != nil {
		core.LogModule(moduleID, "ERROR", "Failed to update module page", nil, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// convert to API model and respond
	apiPage := api.ModulePageToAPIModulePage(modulePage)
	if b, err := json.Marshal(apiPage); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	} else {
		core.LogModule(moduleID, "INFO",
			fmt.Sprintf("Updated page '%s'", apiPage.ID),
			map[string]any{
				"-> name":      apiPage.Name,
				"-> url":       apiPage.URL,
				"-> is_public": apiPage.IsPublic,
			}, nil)
		w.Write(b)
	}
}
