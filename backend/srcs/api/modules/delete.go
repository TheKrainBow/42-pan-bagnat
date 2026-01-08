package modules

import (
	"backend/core"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// @Security     SessionAuth
// @Summary      Delete Module
// @Description  Delete a module for your campus (All module datas will be lost!)
// @Tags         Modules
// @Accept       json
// @Produce      json
// @Param        input body ModulePatchInput true "Module input"
// @Success      200
// @Router       /admin/modules [delete]
func DeleteModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleID := chi.URLParam(r, "moduleID")

	if moduleID == "" {
		http.Error(w, "Missing field moduleID", http.StatusBadRequest)
		return
	}

	err := core.DeleteModule(moduleID)
	if err != nil {
		fmt.Printf("Error while deleting module: %s\n", err.Error())
		http.Error(w, "error while deleting module", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "OK")
}

// @Security     SessionAuth
// @Summary      Delete Module page
// @Description  Delete a module page for your campus (All page datas will be lost!)
// @Tags         Modules,Pages
// @Accept       json
// @Produce      json
// @Param        input body ModulePatchInput true "Module input"
// @Success      200
// @Router       /admin/modules/{moduleID}/pages/{pageID} [delete]
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

// @Security     SessionAuth
// @Summary      Remove role from module
// @Description  Revokes a specific role from a module (by login or ID)
// @Tags         Modules,Roles
// @Accept       json
// @Produce      json
// @Param        moduleID path string true "User moduleID (ID or login)"
// @Param        roleID path string true "Role ID"
// @Success      204 "Role successfully removed"
// @Router       /admin/modules/{moduleID}/roles/{roleID} [delete]
func DeleteModuleRole(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	roleID := chi.URLParam(r, "roleID")

	err := core.DeleteRoleFromModule(roleID, moduleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete role: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error delete role %s from module %s: %v\n", roleID, moduleID, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "Role %s successfully deleted role from module %s\n", roleID, moduleID)
}
