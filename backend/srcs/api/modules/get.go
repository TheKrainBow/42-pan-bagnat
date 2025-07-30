package modules

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

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
		log.Printf("error while getting modules: %s\n", err.Error())
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

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}
	// for _, param := range chi.RouteContext(r.Context()).URLParams.Values {
	// 	log.Printf("Param key: %s, value: %s", param, param)
	// }
	// log.Printf("Backend id: %+v", chi.RouteContext(r.Context()).URLParams)

	module, err := core.GetModule(id)
	if err != nil {
		log.Printf("Failed fetching module: %s\n", err.Error())
		http.Error(w, "Failed fetching module info", http.StatusInternalServerError)
		return
	}

	dest := api.ModuleToAPIModule(module)
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
// @Router       /modules/{moduleID}/logs [get]
func GetModuleLogs(w http.ResponseWriter, r *http.Request) {
	var err error
	var roles []core.ModuleLog
	var nextToken string

	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

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

	dest := ModuleLogsGetResponse{}
	pagination := core.ModuleLogsPagination{
		ModuleID:      moduleID,
		OrderBy:       core.GenerateModuleLogsOrderBy(order),
		Filter:        filter,
		LastModuleLog: nil,
		Limit:         limit,
	}

	if pageToken != "" {
		pagination, err = core.DecodeModuleLogsPaginationToken(pageToken)
		if err != nil {
			http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
			fmt.Printf("Couldn't decode token:\n%s\n: %s\n", pageToken, err.Error())
			return
		}
	}
	roles, nextToken, err = core.GetModuleLogs(pagination)
	if err != nil {
		log.Printf("error while getting modules: %s\n", err.Error())
		http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
		return
	}
	dest.NextPage = nextToken
	dest.ModuleLogs = api.ModuleLogsToAPIModuleLogs(roles)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Summary      Get Module List
// @Description  Return the module.yml of a given module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      200 {object} Module
// @Router       /modules/{moduleID}/config [get]
func GetModuleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "moduleID")

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}
	// for _, param := range chi.RouteContext(r.Context()).URLParams.Values {
	// 	log.Printf("Param key: %s, value: %s", param, param)
	// }
	// log.Printf("Backend id: %+v", chi.RouteContext(r.Context()).URLParams)

	module, err := core.GetModule(id)
	if err != nil {
		http.Error(w, "Failed fetching module info", http.StatusInternalServerError)
		return
	}

	cfg, err := core.GetModuleConfig(module)
	if err != nil {
		http.Error(w, "cannot read config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"config": cfg, // this is the entire YAML, newlines preserved
	})
}

// @Summary      Get Module List
// @Description  Return the module.yml of a given module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      200 {object} Module
// @Router       /modules/{moduleID}/config [get]
func GetModulePages(w http.ResponseWriter, r *http.Request) {
	var err error
	var pages []core.ModulePage
	var nextToken string

	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

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

	dest := ModulePagesGetResponse{}
	pagination := core.ModulePagesPagination{
		ModuleID:       &moduleID,
		OrderBy:        core.GenerateModulePagesOrderBy(order),
		Filter:         filter,
		LastModulePage: nil,
		Limit:          limit,
	}

	if pageToken != "" {
		pagination, err = core.DecodeModulePagesPaginationToken(pageToken)
		if err != nil {
			http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
			fmt.Printf("Couldn't decode token:\n%s\n: %s\n", pageToken, err.Error())
			return
		}
	}
	pages, nextToken, err = core.GetModulePages(pagination)
	if err != nil {
		log.Printf("error while getting modules: %s\n", err.Error())
		http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
		return
	}
	dest.NextPage = nextToken
	dest.ModulePages = api.ModulePagesToAPIModulePages(pages)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

func GetContainerLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

	containerName := chi.URLParam(r, "containerName")
	if containerName == "" {
		http.Error(w, "Container name not found", http.StatusBadRequest)
		return
	}

	module, err := core.GetModule(moduleID)
	if err != nil {
		log.Printf("error while getting module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while getting module " + moduleID))
		return
	}

	logs, err := core.GetContainerLogs(module, containerName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// @Summary      Get Module List
// @Description  Return the module.yml of a given module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      200 {object} Module
// @Router       /modules/{moduleID}/config [get]
func GetModuleContainers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

	module, err := core.GetModule(moduleID)
	if err != nil {
		log.Printf("error while getting module %s: %s\n", moduleID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while getting module " + moduleID))
		return
	}

	containers, err := core.GetModuleContainers(module)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get containers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(containers); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Get Module List
// @Description  Return the module.yml of a given module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Success      200 {object} Module
// @Router       /pages [get]
func GetPages(w http.ResponseWriter, r *http.Request) {
	var err error
	var pages []core.ModulePage
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

	dest := ModulePagesGetResponse{}
	pagination := core.ModulePagesPagination{
		ModuleID:       nil,
		OrderBy:        core.GenerateModulePagesOrderBy(order),
		Filter:         filter,
		LastModulePage: nil,
		Limit:          limit,
	}

	if pageToken != "" {
		pagination, err = core.DecodeModulePagesPaginationToken(pageToken)
		if err != nil {
			http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
			fmt.Printf("Couldn't decode token:\n%s\n: %s\n", pageToken, err.Error())
			return
		}
	}
	pages, nextToken, err = core.GetModulePages(pagination)
	if err != nil {
		log.Printf("error while getting modules: %s\n", err.Error())
		http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
		return
	}
	dest.NextPage = nextToken
	dest.ModulePages = api.ModulePagesToAPIModulePages(pages)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

func PageRedirection(w http.ResponseWriter, r *http.Request) {
	pageName := chi.URLParam(r, "pageName")

	// Fetch up to 1 page for this module
	pages, err := core.GetPage(pageName)
	if err != nil {
		http.Error(w, "error looking up module pages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	targetURL, err := url.Parse(pages.URL)
	if err != nil {
		http.Error(w, "bad module URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set up the reverse‐proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Strip off "/module-page/{mod}" so that
	// /module-page/toto/foo/bar → /foo/bar on the target.
	suffix := strings.TrimPrefix(r.URL.Path, "/module-page/"+pageName)
	if suffix == "" {
		suffix = "/"
	}
	r.URL.Path = suffix

	fmt.Printf("url: %s\n", r.URL.Path)
	proxy.ServeHTTP(w, r)
}
