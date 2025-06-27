package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
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

	err = core.CloneModuleRepo(module)
	if err != nil {
		log.Printf("error while cloning module %s: %s\n", module.ID, err.Error())
	}

	fmt.Fprint(w, string(destJSON))
}

// GitClone clones a new Git repository for the module
// @Summary      Clone Git Repository
// @Description  Clone the Git repository for the specified module (only allowed if not already cloned)
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      202 {string} string "Cloning module..."
// @Router       /modules/{moduleID}/git/clone [post]
func GitClone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")

	module, err := core.GetModule(moduleID)
	if err != nil {
		log.Printf("error while getting module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while getting module " + moduleID))
		return
	}

	if module.ID == "" {
		log.Printf("error while cloning module %s: module doesn't exist\n", moduleID)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("module " + moduleID + " doesn't exist"))
		return
	}

	if !module.LastUpdate.IsZero() {
		log.Printf("error while cloning module %s: module is already cloned\n", moduleID)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("module " + moduleID + " is already cloned"))
		return
	}

	err = core.CloneModuleRepo(module)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error while cloning module " + moduleID))
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Cloning module: " + moduleID))
}

// GitPull pulls latest changes from the Git repository
// @Summary      Pull Git Repository
// @Description  Pull the latest changes for a previously cloned module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        moduleID path string true "Module ID"
// @Success      202 {string} string "Pulling latest changes..."
// @Router       /modules/{moduleID}/git/pull [post]
func GitPull(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	module, err := core.GetModule(moduleID)
	if err != nil {
		log.Printf("error while getting module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while getting module " + moduleID))
		return
	}

	if module.ID == "" {
		log.Printf("error while cloning module %s: module doesn't exist\n", moduleID)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("module " + moduleID + " doesn't exist"))
		return
	}

	if module.LastUpdate.IsZero() {
		log.Printf("error while cloning module %s: module is not cloned yet\n", moduleID)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("module " + moduleID + " is not cloned yet"))
		return
	}

	err = core.PullModuleRepo(module)
	if err != nil {
		log.Printf("error while cloning module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error while cloning module " + moduleID))
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Cloning module: " + moduleID))
}

// GitUpdateRemote updates the remote Git URL for the module
// @Summary      Update Git Remote
// @Description  Change the remote Git URL of the specified module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        moduleID path string true "Module ID"
// @Success      202 {string} string "Updating remote..."
// @Router       /modules/{moduleID}/git/update-remote [post]
func GitUpdateRemote(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	module, err := core.GetModule(moduleID)
	if err != nil {
		log.Printf("error while getting module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while getting module " + moduleID))
		return
	}

	if module.ID == "" {
		log.Printf("error while cloning module %s: module doesn't exist\n", moduleID)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("module " + moduleID + " doesn't exist"))
		return
	}

	var input struct {
		NewURL string `json:"git_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.NewURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid or missing 'git_url' in body"))
		return
	}

	err = core.UpdateModuleGitRemote(module.ID, module.Name, input.NewURL)
	if err != nil {
		log.Printf("error while cloning module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error while cloning module " + moduleID))
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Cloning module: " + moduleID))
}
