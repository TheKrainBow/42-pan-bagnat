package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// PatchModule updates the metadata of an existing module.
// @Summary      Patch Module
// @Description  Updates the specified fields of a module (e.g. name, Git URL/branch, status).
// @Tags         Modules
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string              true  "Module ID"
// @Param        input     body      ModulePatchInput    true  "Fields to update"
// @Success      200       {object}  api.Module          "The updated module"
// @Failure      400       {string}  string              "Invalid module ID or JSON body"
// @Failure      404       {string}  string              "Module not found"
// @Failure      500       {string}  string              "Internal server error"
// @Router       /admin/modules/{moduleID} [patch]
func PatchModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1️⃣ Path parameter
	moduleID := chi.URLParam(r, "moduleID")
	if strings.TrimSpace(moduleID) == "" {
		http.Error(w, "moduleID is required", http.StatusBadRequest)
		return
	}

	// 2️⃣ Decode and validate input
	var input ModulePatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Name != nil {
		*input.Name = strings.TrimSpace(*input.Name)
	}
	if input.GitURL != nil {
		*input.GitURL = strings.TrimSpace(*input.GitURL)
	}
	if input.GitBranch != nil {
		*input.GitBranch = strings.TrimSpace(*input.GitBranch)
	}

	// 3️⃣ Perform the patch via core
	updated, err := core.PatchModule(core.ModulePatch{
		ID:        moduleID,
		Name:      input.Name,
		GitURL:    input.GitURL,
		GitBranch: input.GitBranch,
	})
	if err != nil || updated == nil {
		log.Printf("error patching module %s: %v\n", moduleID, err)
		http.Error(w, "Failed to update module", http.StatusInternalServerError)
		return
	}

	// 4️⃣ Map to API and respond
	resp := api.ModuleToAPIModule(*updated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("error encoding updated module %s: %v\n", moduleID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// PatchModulePage updates an existing front-page of a module.
// @Summary      Patch Module Page
// @Description  Updates the metadata of a module’s page (name, URL, visibility).
// @Tags         Modules,Pages
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string                    true  "Module ID"
// @Param        pageID    path      string                    true  "Page ID"
// @Param        input     body      ModulePageUpdateInput     true  "Fields to update"
// @Success      200       {object}  api.ModulePage            "The updated module page"
// @Failure      400       {string}  string                    "Bad request (missing or invalid parameters)"
// @Failure      404       {string}  string                    "Page not found"
// @Failure      500       {string}  string                    "Internal server error"
// @Router       /admin/modules/{moduleID}/pages/{pageID} [patch]
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var input ModulePageUpdateInput
	if err := json.Unmarshal(body, &input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}
	targetContainerSet := false
	if _, ok := raw["target_container"]; ok {
		targetContainerSet = true
	}
	targetPortSet := false
	if _, ok := raw["target_port"]; ok {
		targetPortSet = true
	}
	networkSet := false
	if _, ok := raw["network_name"]; ok {
		networkSet = true
	}

	// trim & validate required fields
	if input.Name != nil {
		*input.Name = strings.TrimSpace(*input.Name)
	}

	if input.TargetContainer != nil {
		trimmed := strings.TrimSpace(*input.TargetContainer)
		input.TargetContainer = &trimmed
	}

	if input.NetworkName != nil {
		trimmed := strings.TrimSpace(*input.NetworkName)
		input.NetworkName = &trimmed
	}

	// perform update
	modulePage, err := core.UpdateModulePage(pageID, input.Name, input.TargetContainer, targetContainerSet, input.TargetPort, targetPortSet, input.IframeOnly, input.NeedAuth, input.NetworkName, networkSet)
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
				"-> name":        apiPage.Name,
				"-> target":      describeTarget(apiPage.TargetContainer, apiPage.TargetPort),
				"-> iframe_only": apiPage.IframeOnly,
				"-> need_auth":   apiPage.NeedAuth,
			}, nil)
		w.Write(b)
	}
}

func describeTarget(container *string, port *int) string {
	if container != nil && port != nil {
		return fmt.Sprintf("%s:%d", *container, *port)
	}
	return "pending target"
}
