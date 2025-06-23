package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// @Summary      Post Module List
// @Description  Download a new module for your campus
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body ModuleGitInput true "Git URL and SSH key"
// @Success      200 {object} Module
// @Router       /modules [post]
func PostModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse input
	var input struct {
		GitURL    string `json:"git_url"`
		GitBranch string `json:"git_branch"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.GitURL) == "" || strings.TrimSpace(input.Name) == "" {
		http.Error(w, "Missing git_url or name", http.StatusBadRequest)
		return
	}

	module, err := core.ImportModule(input.Name, input.GitURL, input.GitBranch)
	if err != nil {
		log.Printf("failed to import module: %v", err)
		http.Error(w, "Failed to import module", http.StatusInternalServerError)
		return
	}

	dest := api.ModuleToAPIModule(module)
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
