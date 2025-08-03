package users

import (
	"backend/api/auth"
	api "backend/api/dto"
	"backend/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// @Summary      Get User List
// @Description  Returns all the available users for your campus
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200 {object} []User
// @Router       /users [get]
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
	dest.NextPage = nextToken
	dest.Users = api.UsersToAPIUsers(users)

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

func GetUserMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user := r.Context().Value(auth.UserCtxKey)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	coreUser, ok := user.(core.User)
	if !ok {
		http.Error(w, "Invalid user context", http.StatusInternalServerError)
		return
	}

	apiUser := api.UserToAPIUser(coreUser)
	destJSON, err := json.Marshal(apiUser)
	if err != nil {
		http.Error(w, "Failed to convert user to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Summary      Get User List
// @Description  Returns all the available users for your campus
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200 {object} User
// @Router       /users/{identifier} [get]
func GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	identifier := chi.URLParam(r, "identifier")
	log.Printf("Received identifier: '%s'", identifier)

	if identifier == "" {
		http.Error(w, "Identifier is required", http.StatusBadRequest)
		return
	}

	user, err := core.GetUser(identifier)
	if err != nil {
		http.Error(w, fmt.Sprintf("User not found: %s", err.Error()), http.StatusNotFound)
		return
	}

	apiUser := api.UserToAPIUser(user)

	destJSON, err := json.Marshal(apiUser)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
