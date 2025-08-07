package users

import (
	"backend/api/auth"
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// @Security     AdminAuth
// GetUsers returns a paginated list of users for your campus.
// @Summary      Get User List
// @Description  Returns all available users for your campus, with optional filtering, sorting, and pagination.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        filter           query   string  false  "Filter expression (e.g. \"ft_login=heinz\")"
// @Param        next_page_token  query   string  false  "Pagination token for the next page"
// @Param        order            query   string  false  "Sort order: asc or desc"            Enums(asc,desc)  default(desc)
// @Param        limit            query   int     false  "Maximum number of items per page"   default(50)
// @Success      200              {object} UserGetResponse
// @Failure      500              {string} string  "Internal server error"
// @Router       /admin/users [get]
func GetUsers(w http.ResponseWriter, r *http.Request) {
	var err error
	var users []core.User
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

	dest := UserGetResponse{}
	pagination := core.UserPagination{
		OrderBy:  core.GenerateUserOrderBy(order),
		Filter:   filter,
		LastUser: nil,
		Limit:    limit,
	}
	if pageToken != "" {
		pagination, err = core.DecodeUserPaginationToken(pageToken)
		if err != nil {
			http.Error(w, "Failed in core.GetUsers()", http.StatusInternalServerError)
			fmt.Printf("Couldn't decode token:\n%s\n: %s\n", pageToken, err.Error())
			return
		}
	}
	users, nextToken, err = core.GetUsers(pagination)
	if err != nil {
		http.Error(w, "Failed in core.GetUsers()", http.StatusInternalServerError)
		return
	}
	dest.NextPageToken = nextToken
	dest.Users = api.UsersToAPIUsers(users)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// GetUser returns details for a specific user by ID or login.
// @Summary      Get User
// @Description  Retrieves a user’s details given their ID or login identifier.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        identifier  path      string     true   "User identifier (ID or login)"
// @Success      200         {object}  api.User   "The requested user object"
// @Failure      400         {string}  string     "Identifier is required"
// @Failure      404         {string}  string     "User not found"
// @Failure      500         {string}  string     "Internal server error"
// @Router       /admin/users/{identifier} [get]
func GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	identifier := chi.URLParam(r, "identifier")
	if strings.TrimSpace(identifier) == "" {
		http.Error(w, "Identifier is required", http.StatusBadRequest)
		return
	}

	user, err := core.GetUser(identifier)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			log.Printf("error fetching user %s: %v\n", identifier, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	apiUser := api.UserToAPIUser(user)
	if err := json.NewEncoder(w).Encode(apiUser); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// GetUserMe returns the details of the currently authenticated user.
// @Summary      Get Current User
// @Description  Retrieves the user profile for the authenticated session.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.User  "The current user’s profile"
// @Failure      401  {string}  string    "Unauthorized"
// @Failure      500  {string}  string    "Internal server error"
// @Router       /users/me [get]
func GetUserMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	u := r.Context().Value(auth.UserCtxKey)
	if u == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	coreUser, ok := u.(*core.User)
	if !ok || coreUser == nil {
		http.Error(w, "Invalid user context", http.StatusInternalServerError)
		return
	}

	apiUser := api.UserToAPIUser(*coreUser)
	if err := json.NewEncoder(w).Encode(apiUser); err != nil {
		http.Error(w, "Failed to encode user to JSON", http.StatusInternalServerError)
	}
}

// GetUserPages returns all module pages accessible to a specific user.
// @Summary      Get Pages for a User
// @Description  Retrieves all module pages accessible to the specified user by ID or ft_login.
// @Tags         Pages
// @Accept       json
// @Produce      json
// @Param        userID path string true "User ID or ft_login"
// @Success      200    {object} modules.ModulePagesGetResponse
// @Failure      400    {string} string "ID not found"
// @Failure      500    {string} string "Internal server error"
// @Router       /users/{userID}/pages [get]
func GetUserPages(w http.ResponseWriter, r *http.Request) {
	var err error
	var pages []core.ModulePage

	w.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}
	pages, err = core.GetUserPages(userID)
	if err != nil {
		log.Printf("error while getting user pages: %s\n", err.Error())
		http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
		return
	}

	dest := api.ModulePagesToAPIModulePages(pages)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// GetContextUserPages returns all module pages accessible to the current user.
// @Summary      Get User Pages
// @Description  Retrieves all module pages accessible to the currently logged-in user.
// @Tags         Pages
// @Accept       json
// @Produce      json
// @Success      200  {object} modules.ModulePagesGetResponse
// @Failure      404  {string} string  "User not found"
// @Failure      500  {string} string  "Internal server error"
// @Router       /users/me/pages [get]
func GetContextUserPages(w http.ResponseWriter, r *http.Request) {
	var err error
	var pages []core.ModulePage

	w.Header().Set("Content-Type", "application/json")

	u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
	if !ok {
		log.Printf("No user logged with /me request\n")
		http.Error(w, "Failed in core.GetModules()", http.StatusNotFound)
		return
	}

	pages, err = core.GetUserPages(u.ID)
	if err != nil {
		log.Printf("error while getting modules: %s\n", err.Error())
		http.Error(w, "Failed in core.GetModules()", http.StatusInternalServerError)
		return
	}

	dest := api.ModulePagesToAPIModulePages(pages)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
