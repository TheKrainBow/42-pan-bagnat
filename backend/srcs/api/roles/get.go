package roles

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
	w.Header().Set("Content-Type", "application/json")

	dest := Role{
		ID:    "01HZ0MMK4S6VQW4WPHB6NZ7R7X",
		Name:  "Test",
		Color: "0xFF00FF",
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}

// @Summary      Get Role List
// @Description  Returns all the available roles for your campus
// @Tags         roles
// @Accept       json
// @Produce      json
// @Success      200 {object} Role
// @Router       /roles/{roleID} [get]
func GetRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "roleID")
	log.Printf("Received ID: '%s'", id) // This should print the ID

	if id == "" {
		http.Error(w, "ID not found", http.StatusBadRequest)
		return
	}
	// for _, param := range chi.RouteContext(r.Context()).URLParams.Values {
	// 	log.Printf("Param key: %s, value: %s", param, param)
	// }
	// log.Printf("Backend id: %+v", chi.RouteContext(r.Context()).URLParams)

	dest := Role{
		ID:    "01HZ0MMK4S6VQW4WPHB6NZ7R7X",
		Name:  "Test",
		Color: "0xFF00FF",
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
