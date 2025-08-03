package roles

import (
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// @Summary      Get Role List
// @Description  Returns all the available roles for your campus
// @Tags         roles
// @Accept       json
// @Produce      json
// @Success      200 {object} []Role
// @Router       /roles [get]
func GetRoles(w http.ResponseWriter, r *http.Request) {
	var err error
	var roles []core.Role
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

	dest := RoleGetResponse{}
	pagination := core.RolePagination{
		OrderBy:  core.GenerateRoleOrderBy(order),
		Filter:   filter,
		LastRole: nil,
		Limit:    limit,
	}
	if pageToken != "" {
		pagination, err = core.DecodeRolePaginationToken(pageToken)
		if err != nil {
			http.Error(w, "Failed in core.GetRoles()", http.StatusInternalServerError)
			fmt.Printf("Couldn't decode token:\n%s\n: %s\n", pageToken, err.Error())
			return
		}
	}
	roles, nextToken, err = core.GetRoles(pagination)
	if err != nil {
		http.Error(w, "Failed in core.GetRoles()", http.StatusInternalServerError)
		return
	}
	dest.NextPage = nextToken
	dest.Roles = api.RolesToAPIRoles(roles)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Summary      Get Role
// @Description  Returns details about a specific role and its linked modules
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        roleID path string true "Role ID"
// @Success      200 {object} api.Role
// @Failure      400 {object} api.ErrorResponse
// @Failure      404 {object} api.ErrorResponse
// @Router       /roles/{roleID} [get]
func GetRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	roleID := chi.URLParam(r, "roleID")
	log.Printf("Received roleID: '%s'", roleID)

	if roleID == "" {
		http.Error(w, "Role ID is required", http.StatusBadRequest)
		return
	}

	role, err := core.GetRole(roleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Role not found: %s", err.Error()), http.StatusNotFound)
		return
	}

	apiRole := api.RoleToAPIRole(role)

	destJSON, err := json.Marshal(apiRole)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
