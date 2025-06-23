package git

import (
	"backend/core"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

// GitClone clones a new Git repository for the module
// @Summary      Clone Git Repository
// @Description  Clone the Git repository for the specified module (only allowed if not already cloned)
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        moduleID path string true "Module ID"
// @Success      202 {string} string "Cloning module..."
// @Router       /modules/{moduleID}/git/clone [post]
func GitClone(w http.ResponseWriter, r *http.Request) {
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

	err = core.CloneModuleRepo(module.ID, module.GitURL, module.Slug, module.SSHPrivateKey)
	if err != nil {
		log.Printf("error while cloning module %s: %s\n", moduleID, err.Error())
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

	err = core.PullModuleRepo(module.Name, module.SSHPrivateKey)
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
