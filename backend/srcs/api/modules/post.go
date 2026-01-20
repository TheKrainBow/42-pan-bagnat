package modules

import (
	"backend/api/auth"
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// PostModule imports a new module into your campus by cloning its Git repo.
// @Summary      Post Module
// @Description  Downloads and registers a new module for your campus by cloning from the provided Git URL.
// @Tags         Modules
// @Accept       json
// @Produce      json
// @Param        input  body      ModuleGitInput  true  "Git URL, branch, and module name"
// @Success      200    {object}  api.Module          "The newly imported module"
// @Failure      400    {string}  string              "Invalid or missing fields"
// @Failure      500    {string}  string              "Failed to import module"
// @Router       /admin/modules [post]
func PostModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse input
	var input struct {
		GitURL    string `json:"git_url"`
		GitBranch string `json:"git_branch"`
		Name      string `json:"name"`
		SSHKeyID  string `json:"ssh_key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.GitURL) == "" || strings.TrimSpace(input.Name) == "" {
		http.Error(w, "Missing git_url or name", http.StatusBadRequest)
		return
	}

	var actor *core.User
	if u, ok := r.Context().Value(auth.UserCtxKey).(*core.User); ok && u != nil {
		actor = u
	}

	module, err := core.ImportModule(actor, input.Name, input.GitURL, input.GitBranch, input.SSHKeyID)
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
	logSSHKeyUsage(r, module, "git clone")

	fmt.Fprint(w, string(destJSON))
}

// GitClone initiates cloning of the module’s Git repository.
// @Summary      Clone Module Repository
// @Description  Starts an asynchronous clone of the Git repository for the specified module. Only allowed if the module isn’t already cloned.
// @Tags         Modules,Git
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Success      202       {string}  string  "Cloning module: {moduleID}"
// @Failure      404       {string}  string  "module {moduleID} doesn't exist"
// @Failure      409       {string}  string  "module {moduleID} is already cloned"
// @Failure      500       {string}  string  "error while cloning module {moduleID}"
// @Router       /admin/modules/{moduleID}/git/clone [post]
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

	logSSHKeyUsage(r, module, "git pull")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Cloning module: " + moduleID))
}

// GitPull pulls the latest changes from a module’s Git repository.
// @Summary      Pull Module Repository
// @Description  Pulls the latest commits for a previously cloned module.
// @Tags         Modules,Git
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Success      202       {string}  string  "Pulling latest changes for module {moduleID}"
// @Failure      404       {string}  string  "module {moduleID} doesn't exist"
// @Failure      409       {string}  string  "module {moduleID} is not cloned yet"
// @Failure      500       {string}  string  "error while pulling module {moduleID}"
// @Router       /admin/modules/{moduleID}/git/pull [post]
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

	logSSHKeyUsage(r, module, "git update remote")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Cloning module: " + moduleID))
}

// GitUpdateRemote updates the remote Git URL for the specified module.
// @Summary      Update Module Git Remote
// @Description  Changes the Git remote URL for a previously imported module.
// @Tags         Modules,Git
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string                          true  "Module ID"
// @Param        input     body      ModuleRemoteUpdateInput         true  "New Git URL"
// @Success      202       {string}  string                          "Updating remote for module {moduleID}"
// @Failure      400       {string}  string                          "Invalid or missing 'git_url'"
// @Failure      404       {string}  string                          "module {moduleID} doesn't exist"
// @Failure      500       {string}  string                          "Error updating remote for module {moduleID}"
// @Router       /admin/modules/{moduleID}/git/update-remote [post]
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

	logSSHKeyUsage(r, module, "ssh key reassigned")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Cloning module: " + moduleID))
}

// GitSetSSHKey assigns an existing SSH key to the module.
// @Summary      Assign Module SSH Key
// @Description  Replaces the module's SSH key reference with an existing Pan Bagnat SSH key.
// @Tags         Modules,Git
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Param        input     body      struct{SSHKeyID string `json:"ssh_key_id"`}  true  "SSH key to assign"
// @Success      200       {object}  api.Module
// @Failure      400       {string}  string  "Invalid payload"
// @Failure      404       {string}  string  "Module not found"
// @Router       /admin/modules/{moduleID}/git/ssh-key [post]
func GitSetSSHKey(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	var input struct {
		SSHKeyID string `json:"ssh_key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.SSHKeyID) == "" {
		http.Error(w, "Missing ssh_key_id", http.StatusBadRequest)
		return
	}
	var actor *core.User
	if u, ok := r.Context().Value(auth.UserCtxKey).(*core.User); ok && u != nil {
		actor = u
	}
	module, err := core.AssignModuleSSHKey(moduleID, input.SSHKeyID, actor)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logSSHKeyUsage(r, module, "git import")
	json.NewEncoder(w).Encode(api.ModuleToAPIModule(module))
}

// DeployConfig saves and deploys the configuration for a given module.
// @Summary      Deploy Module Configuration
// @Description  Saves the provided YAML config and triggers a deployment for the specified module.
// @Tags         Modules,Docker
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string           true  "Module ID"
// @Param        input     body      ComposeRequest   true  "YAML configuration payload"
// @Success      202       {string}  string           "Deployment started for module {moduleID}"
// @Failure      400       {string}  string           "Invalid request payload or missing moduleID"
// @Failure      404       {string}  string           "module {moduleID} doesn't exist"
// @Failure      500       {string}  string           "Internal server error"
// @Router       /admin/modules/{moduleID}/docker/deploy [post]
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

	if module.IsDeploying {
		http.Error(w, "deployment already in progress", http.StatusConflict)
		return
	}

	var req ComposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	core.SaveModuleConfig(module, req.Config)
	go core.DeployModule(module)
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Deployment started for module " + moduleID))
}

// PostModulePage creates a new front-page for a module.
// @Summary      Create Module Page
// @Description  Adds a new page under the specified module.
// @Tags         Modules,Pages
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string                 true  "Module ID"
// @Param        input     body      ModulePageInput  true  "Page creation input"
// @Success      200       {object}  api.ModulePage         "The created module page"
// @Failure      400       {string}  string                 "Invalid input or missing moduleID"
// @Failure      500       {string}  string                 "Failed to create module page"
// @Router       /admin/modules/{moduleID}/pages [post]
func PostModulePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

	var input ModulePageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	if input.Name = strings.TrimSpace(input.Name); input.Name == "" {
		http.Error(w, "Missing field name", http.StatusBadRequest)
		return
	}

	var targetContainer *string
	if input.TargetContainer != nil {
		trimmed := strings.TrimSpace(*input.TargetContainer)
		if trimmed != "" {
			targetContainer = &trimmed
		}
	}

	var targetPort *int
	if input.TargetPort != nil {
		if *input.TargetPort <= 0 || *input.TargetPort > 65535 {
			http.Error(w, "Invalid target_port", http.StatusBadRequest)
			return
		}
		val := *input.TargetPort
		targetPort = &val
	}

	if (targetContainer == nil) != (targetPort == nil) {
		http.Error(w, "target_container and target_port must be defined together", http.StatusBadRequest)
		return
	}

	var network string
	if input.NetworkName != nil {
		trimmed := strings.TrimSpace(*input.NetworkName)
		if trimmed != "" {
			network = trimmed
		}
	}

	modulePage, err := core.ImportModulePage(moduleID, input.Name, targetContainer, targetPort, input.IframeOnly, input.NeedAuth, network)
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

	targetMsg := "pending target"
	if dest.TargetContainer != nil && dest.TargetPort != nil {
		targetMsg = fmt.Sprintf("%s:%d", *dest.TargetContainer, *dest.TargetPort)
	}
	core.LogModule(moduleID, "INFO", fmt.Sprintf("Created page '%s' targeting %s", dest.Slug, targetMsg), map[string]any{
		"iframe_only": dest.IframeOnly,
		"need_auth":   dest.NeedAuth,
	}, nil)
	fmt.Fprint(w, string(destJSON))
}

// StartModuleContainer starts a specific container within a module.
// @Summary      Start Module Container
// @Description  Initiates the start of the specified container for the given module.
// @Tags         Modules,Docker
// @Accept       json
// @Produce      json
// @Param        moduleID       path      string  true  "Module ID"
// @Param        containerName  path      string  true  "Container name"
// @Success      204            {string}  string  "No Content"
// @Failure      404            {string}  string  "Module not found"
// @Failure      500            {string}  string  "Failed to start container"
// @Router       /admin/modules/{moduleID}/docker/{containerName}/start [post]
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

// StopModuleContainer stops a specific container within a module.
// @Summary      Stop Module Container
// @Description  Stops the specified container for the given module.
// @Tags         Modules,Docker
// @Accept       json
// @Produce      json
// @Param        moduleID       path      string  true  "Module ID"
// @Param        containerName  path      string  true  "Container name"
// @Success      204            {string}  string  "No Content"
// @Failure      404            {string}  string  "Module not found"
// @Failure      500            {string}  string  "Failed to stop container"
// @Router       /admin/modules/{moduleID}/docker/{containerName}/stop [post]
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

// RestartModuleContainer restarts a specific container within a module.
// @Summary      Restart Module Container
// @Description  Restarts the specified container for the given module.
// @Tags         Modules,Docker
// @Accept       json
// @Produce       json
// @Param        moduleID       path      string  true  "Module ID"
// @Param        containerName  path      string  true  "Container name"
// @Success      204            {string}  string  "No Content"
// @Failure      404            {string}  string  "Module not found"
// @Failure      500            {string}  string  "Failed to restart container"
// @Router       /admin/modules/{moduleID}/docker/{containerName}/restart [post]
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

// DeleteModuleContainer deletes (stops and removes) a specific container within a module.
// @Summary      Delete Module Container
// @Description  Stops and removes the specified container for the given module.
// @Tags         Modules,Docker
// @Accept       json
// @Produce       json
// @Param        moduleID       path      string  true  "Module ID"
// @Param        containerName  path      string  true  "Container name"
// @Success      204            {string}  string  "No Content"
// @Failure      404            {string}  string  "Module not found"
// @Failure      500            {string}  string  "Failed to delete container"
// @Router       /admin/modules/{moduleID}/docker/{containerName}/delete [delete]
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

// PostModuleRole grants a role to a module.
// @Summary      Add Role to Module
// @Description  Assigns the specified role to the given module.
// @Tags         Modules,Roles
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Param        roleID    path      string  true  "Role ID"
// @Success      201       {string}  string  "Role successfully assigned to module"
// @Failure      400       {string}  string  "Bad request"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /admin/modules/{moduleID}/roles/{roleID} [post]
func PostModuleRole(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	roleID := chi.URLParam(r, "roleID")

	err := core.AddRoleToModule(roleID, moduleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assign role: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error assigning role %s to module %s: %v\n", roleID, moduleID, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Role %s successfully assigned to module %s\n", roleID, moduleID)
}
