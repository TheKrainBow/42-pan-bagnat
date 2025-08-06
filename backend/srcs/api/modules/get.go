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

// @Security     SessionAuth
// GetModules returns a paginated list of modules available for your campus.
// @Summary      Get Module List
// @Description  Returns all available modules for your campus, with optional filtering, sorting, and pagination.
// @Tags         Modules
// @Accept       json
// @Produce      json
// @Param        filter           query   string  false  "Filter expression (e.g. \"status=enabled\")"
// @Param        next_page_token  query   string  false  "Pagination token for the next page"
// @Param        order            query   string  false  "Sort order: asc or desc"                      Enums(asc,desc)  default(desc)
// @Param        limit            query   int     false  "Maximum number of items per page"             default(50)
// @Success      200              {object} ModuleGetResponse
// @Failure      400              {string} string    "Bad request"
// @Failure      500              {string} string    "Internal server error"
// @Router       /admin/modules [get]
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
	dest.NextPageToken = nextToken
	dest.Modules = api.ModulesToAPIModules(roles)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Security     SessionAuth
// GetModule returns the details for a specific module.
// @Summary      Get Module
// @Description  Returns all information about a module given its ID.
// @Tags         Modules
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true  "Module ID"
// @Success      200       {object}  api.Module
// @Failure      400       {string}  string  "ID not found"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /admin/modules/{moduleID} [get]
func GetModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "moduleID")

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

	module, err := core.GetModule(id)
	if err != nil {
		log.Printf("Failed fetching module: %s\n", err.Error())
		http.Error(w, "Failed fetching module info", http.StatusInternalServerError)
		return
	}

	dest := api.ModuleToAPIModule(module)
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Security     SessionAuth
// GetModuleLogs returns a paginated list of log entries for a module.
// @Summary      Get Module Logs
// @Description  Retrieves logs for the specified module, with optional filtering, ordering, and pagination.
// @Tags         Modules,Logs
// @Accept       json
// @Produce      json
// @Param        moduleID         path    string  true   "Module ID"
// @Param        filter           query   string  false  "Filter expression (e.g. \"level=INFO\")"
// @Param        next_page_token  query   string  false  "Pagination token for the next page"
// @Param        order            query   string  false  "Sort order: asc or desc"                default(desc)
// @Param        limit            query   int     false  "Maximum number of items per page"       default(50)
// @Success      200              {object} ModuleLogsGetResponse
// @Failure      400              {string} string   "ID not found or bad query parameter"
// @Failure      500              {string} string   "Internal server error"
// @Router       /admin/modules/{moduleID}/logs [get]
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
	dest.NextPageToken = nextToken
	dest.ModuleLogs = api.ModuleLogsToAPIModuleLogs(roles)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Security     SessionAuth
// GetModuleConfig returns the YAML configuration for a given module.
// @Summary      Get Module Configuration
// @Description  Returns the module’s config.yml as a YAML string under the `config` field.
// @Tags         Modules,Docker
// @Accept       json
// @Produce      json
// @Param        moduleID path string true "Module ID"
// @Success      200 {object} ConfigResponse
// @Failure      400 {string} string "ID not found"
// @Failure      500 {string} string "Internal server error"
// @Router       /admin/modules/{moduleID}/docker/config [get]
func GetModuleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "moduleID")

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}

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

// @Security     SessionAuth
// GetModulePages returns a paginated list of pages for a given module.
// @Summary      Get Module Pages
// @Description  Retrieves all pages for the specified module, with optional filtering, sorting, and pagination.
// @Tags         Modules,Pages
// @Accept       json
// @Produce      json
// @Param        moduleID         path    string  true   "Module ID"
// @Param        filter           query   string  false  "Filter expression (e.g. \"title=Home\")"
// @Param        next_page_token  query   string  false  "Pagination token for the next page"
// @Param        order            query   string  false  "Sort order: asc or desc"                Enums(asc,desc)  default(desc)
// @Param        limit            query   int     false  "Maximum number of items per page"       default(50)
// @Success      200              {object} ModulePagesGetResponse
// @Failure      400              {string} string   "ID not found or invalid pagination token"
// @Failure      500              {string} string   "Internal server error"
// @Router       /admin/modules/{moduleID}/pages [get]
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

// GetContainerLogs retrieves the logs for a specific container in a module.
// @Summary      Get Container Logs
// @Description  Returns the log lines for the specified container within a module.
// @Tags         Modules,Docker,Logs
// @Accept       json
// @Produce      json
// @Param        moduleID       path     string  true   "Module ID"
// @Param        containerName  path     string  true   "Container name"
// @Success      200            {array}  string  "An array of log lines"
// @Failure      400            {string} string  "ID not found or container name not found"
// @Failure      500            {string} string  "Internal server error"
// @Router       /admin/modules/{moduleID}/docker/{containerName}/logs [get]
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

// GetModuleContainers returns the list of containers for a given module.
// @Summary      Get Module Containers
// @Description  Retrieves all container names and metadata for the specified module.
// @Tags         Modules,Docker
// @Accept       json
// @Produce      json
// @Param        moduleID  path      string  true   "Module ID"
// @Success      200       {array}   core.ModuleContainer   "List of containers"
// @Failure      400       {string}  string          "ID not found"
// @Failure      500       {string}  string          "Internal server error"
// @Router       /admin/modules/{moduleID}/docker/ls [get]
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

// GetPages returns a paginated list of all module pages.
// @Summary      Get Pages
// @Description  Retrieves all pages across modules, with optional filtering, sorting, and pagination.
// @Tags         Pages
// @Accept       json
// @Produce      json
// @Param        filter           query   string  false  "Filter expression (e.g. \"title=Home\")"
// @Param        next_page_token  query   string  false  "Pagination token for the next page"
// @Param        order            query   string  false  "Sort order: asc or desc"                Enums(asc,desc)  default(desc)
// @Param        limit            query   int     false  "Maximum number of items per page"       default(50)
// @Success      200              {object} ModulePagesGetResponse
// @Failure      400              {string} string                         "Invalid pagination token"
// @Failure      500              {string} string                         "Internal server error"
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

// PageRedirection proxies the root of a module page.
// @Summary      Proxy Module Page (root)
// @Description  Reverse-proxies /module-page/{pageName} to the module’s configured URL.
// @Tags         Pages
// @Param        pageName  path  string  true   "Name of the module page"
// @Success      200       {string}  string  "Proxied content"
// @Failure      400       {string}  string  "Module page name not provided"
// @Failure      500       {string}  string  "Error looking up or proxying the module page"
// @Router       /admin/module-page/{pageName} [get]
func PageRedirectionRoot(w http.ResponseWriter, r *http.Request) {
	PageRedirection(w, r)
}

// PageRedirection proxies any sub-path under a module page.
// @Summary      Proxy Module Page (sub-paths)
// @Description  Reverse-proxies /module-page/{pageName}/{path}/* to the module’s configured URL, stripping the prefix.
// @Tags         Pages
// @Param        pageName  path  string  true   "Name of the module page"
// @Param        path      path  string  true   "Sub-path under the module page (may include slashes)"
// @Success      200       {string}  string  "Proxied content"
// @Failure      400       {string}  string  "Module page name or path not provided"
// @Failure      500       {string}  string  "Error looking up or proxying the module page"
// @Router       /admin/module-page/{pageName}/{path} [get]
func PageRedirectionSub(w http.ResponseWriter, r *http.Request) {
	PageRedirection(w, r)
}

func PageRedirection(w http.ResponseWriter, r *http.Request) {
	pageName := chi.URLParam(r, "pageName")

	page, err := core.GetPage(pageName)
	if err != nil {
		http.Error(w, "error looking up module page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reject non-public pages if not loaded in iframe
	isIframe := r.Header.Get("Referer") != ""

	if !page.IsPublic && !isIframe {
		http.Error(w, "This page is not public", http.StatusForbidden)
		return
	}

	targetURL, err := url.Parse(page.URL)
	if err != nil {
		http.Error(w, "bad module URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	suffix := strings.TrimPrefix(r.URL.Path, "/module-page/"+pageName)
	if suffix == "" {
		suffix = "/"
	}
	r.URL.Path = suffix

	proxy.ServeHTTP(w, r)
}
