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

type composeRequest struct {
	Config string `json:"config"`
}

func DeployConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "moduleID is required", http.StatusBadRequest)
		return
	}

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

	var req composeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	core.SaveModuleConfig(module, req.Config)

	core.DeployModule(module)
}

// @Summary      Post Module List
// @Description  Download a new module for your campus
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body ModuleGitInput true "Git URL and SSH key"
// @Success      200 {object} Module
// @Router       /modules [post]
func PostModulePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

	// Parse input
	var input struct {
		ModuleID string `json:"module_id"`
		Name     string `json:"name"`
		URL      string `json:"url"`
		IsPublic bool   `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	input.ModuleID = moduleID

	if input.URL = strings.TrimSpace(input.URL); input.URL == "" {
		http.Error(w, "Missing field url", http.StatusBadRequest)
		return
	}

	if input.Name = strings.TrimSpace(input.Name); input.Name == "" {
		http.Error(w, "Missing field name", http.StatusBadRequest)
		return
	}

	if input.ModuleID = strings.TrimSpace(input.ModuleID); input.ModuleID == "" {
		http.Error(w, "Missing field module_id", http.StatusBadRequest)
		return
	}

	modulePage, err := core.ImportModulePage(input.ModuleID, input.Name, input.URL, input.IsPublic)
	if err != nil {
		core.LogModule(moduleID, "ERROR", "Couldn't add a module Page", nil, err)
		http.Error(w, "Failed to import module page", http.StatusInternalServerError)
		return
	}

	dest := api.ModulePageToAPIModulePage(modulePage)
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	core.LogModule(moduleID, "INFO", fmt.Sprintf("Created page at '/module-page/%s' from '%s'", dest.Slug, dest.URL), nil, nil)
	fmt.Fprint(w, string(destJSON))
}

func StartModuleContainer(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	containerName := chi.URLParam(r, "containerName")

	module, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	if err := core.StartContainer(module, containerName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start container: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func StopModuleContainer(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	containerName := chi.URLParam(r, "containerName")

	module, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	if err := core.StopContainer(module, containerName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop container: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func RestartModuleContainer(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	containerName := chi.URLParam(r, "containerName")

	module, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	if err := core.RestartContainer(module, containerName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to restart container: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func DeleteModuleContainer(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	containerName := chi.URLParam(r, "containerName")

	module, err := core.GetModule(moduleID)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	if err := core.StopContainer(module, containerName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop container: %v", err), http.StatusInternalServerError)
		return
	}

	if err := core.DeleteContainer(module, containerName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete container: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
