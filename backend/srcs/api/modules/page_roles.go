package modules

import (
	"backend/core"
	"backend/database"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// @Security     SessionAuth
// @Summary      Add role to module page
// @Description  Assigns the specified role to the specified module page.
// @Tags         Modules,Pages,Roles
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Param        pageID    path      string  true  "Page ID"
// @Param        roleID    path      string  true  "Role ID"
// @Success      201       {string}  string  "Role successfully assigned to page"
// @Failure      400       {string}  string  "Bad request"
// @Failure      404       {string}  string  "Page not found"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /admin/modules/{moduleID}/pages/{pageID}/roles/{roleID} [post]
func PostModulePageRole(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "pageID")
	roleID := chi.URLParam(r, "roleID")
	moduleID := chi.URLParam(r, "moduleID")

	page, err := database.GetPageByID(pageID)
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	if page.ModuleID != moduleID {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	if err := database.AssignRoleToPage(roleID, pageID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to assign role: %v", err), http.StatusInternalServerError)
		return
	}

	core.LogModule(moduleID, "INFO", fmt.Sprintf("Assigned role '%s' to page '%s'", roleID, pageID), nil, nil)
	w.WriteHeader(http.StatusCreated)
}

// @Security     SessionAuth
// @Summary      Remove role from module page
// @Description  Removes the specified role from the specified module page.
// @Tags         Modules,Pages,Roles
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Param        pageID    path      string  true  "Page ID"
// @Param        roleID    path      string  true  "Role ID"
// @Success      204       {string}  string  "Role successfully removed from page"
// @Failure      400       {string}  string  "Bad request"
// @Failure      404       {string}  string  "Page not found"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /admin/modules/{moduleID}/pages/{pageID}/roles/{roleID} [delete]
func DeleteModulePageRole(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "pageID")
	roleID := chi.URLParam(r, "roleID")
	moduleID := chi.URLParam(r, "moduleID")

	page, err := database.GetPageByID(pageID)
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	if page.ModuleID != moduleID {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	if err := database.RemoveRoleFromPage(roleID, pageID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete role: %v", err), http.StatusInternalServerError)
		return
	}

	core.LogModule(moduleID, "INFO", fmt.Sprintf("Removed role '%s' from page '%s'", roleID, pageID), nil, nil)
	w.WriteHeader(http.StatusNoContent)
}
