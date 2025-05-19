package users

import (
	"backend/core"
	"backend/handlers/api"
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

// @Summary      Get User List
// @Description  Returns all the available users for your campus
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200 {object} User
// @Router       /users/{userID} [get]
func GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "userID")
	log.Printf("Received ID: '%s'", id) // This should print the ID

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}
	// for _, param := range chi.RouteContext(r.Context()).URLParams.Values {
	// 	log.Printf("Param key: %s, value: %s", param, param)
	// }
	// log.Printf("Backend id: %+v", chi.RouteContext(r.Context()).URLParams)

	dest := api.User{
		ID: id,
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
