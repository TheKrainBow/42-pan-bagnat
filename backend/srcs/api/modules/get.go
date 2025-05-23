package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// @Summary      Get Module List
// @Description  Returns all the available modules for your campus
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      200 {object} []Module
// @Router       /modules [get]
func GetModules(w http.ResponseWriter, r *http.Request) {
	var err error
	var roles []core.Module
	var nextToken string

	w.Header().Set("Content-Type", "application/json")

	filter := r.URL.Query().Get("filter")
	pageToken := r.URL.Query().Get("next_page_token")
	order := r.URL.Query().Get("order")
	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	} else {
		limit = 50
	}

	dest := ModuleGetResponse{}
	pagination := core.ModulePagination{
		OrderBy:    core.GenerateModuleOrderBy(order),
		Filter:     filter,
		LastModule: nil,
		Limit:      limit,
	}
	if pageToken != "" {
		pagination, err = core.DecodeModulePaginationToken(pageToken)
		if err != nil {
			http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
			fmt.Printf("Couldn't decode token:\n%s\n: %s\n", pageToken, err.Error())
			return
		}
	}
	roles, nextToken, err = core.GetModules(pagination)
	if err != nil {
		http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
		return
	}
	dest.NextPage = nextToken
	dest.Modules = api.ModulesToAPIModules(roles)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Summary      Get Module List
// @Description  Returns all the available modules for your campus
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      200 {object} Module
// @Router       /modules/{moduleID} [get]
func GetModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "moduleID")
	log.Printf("Received ID: '%s'", id) // This should print the ID

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}
	// for _, param := range chi.RouteContext(r.Context()).URLParams.Values {
	// 	log.Printf("Param key: %s, value: %s", param, param)
	// }
	// log.Printf("Backend id: %+v", chi.RouteContext(r.Context()).URLParams)

	dest := api.Module{
		ID:            id,
		Name:          "Test",
		Version:       "1.2",
		Status:        api.Enabled,
		URL:           "https://github.com/some-user/some-repo",
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
