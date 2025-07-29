package modules

import (
	"backend/core"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
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

	moduleID := chi.URLParam(r, "moduleID")

	if moduleID == "" {
		http.Error(w, "Missing field moduleID", http.StatusBadRequest)
		return
	}

	err := core.DeleteModule(moduleID)
	if err != nil {
		fmt.Printf("Error while deleting module: %s\n", err.Error())
		http.Error(w, "error while deleting module", http.StatusInternalServerError)
	}

	fmt.Fprint(w, "OK")
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
