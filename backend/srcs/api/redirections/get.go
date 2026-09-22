package redirections

import (
	"backend/core"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetRedirections returns all redirections.
// @Summary      Get Redirection List
// @Description  Returns all redirections.
// @Tags         Redirections
// @Produce      json
// @Success      200  {array}  core.Redirection
// @Failure      500  {string} string "Internal server error"
// @Router       /admin/redirections [get]
func GetRedirections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	redirections, err := core.ListRedirections()
	if err != nil {
		log.Printf("error while listing redirections: %s\n", err.Error())
		http.Error(w, "Failed to list redirections", http.StatusInternalServerError)
		return
	}
	if redirections == nil {
		redirections = []core.Redirection{}
	}

	if err := json.NewEncoder(w).Encode(redirections); err != nil {
		log.Printf("error encoding redirections: %s\n", err.Error())
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// GetRedirection returns a single redirection by ID.
// @Summary      Get Redirection
// @Description  Returns a single redirection by ID.
// @Tags         Redirections
// @Produce      json
// @Param        redirectionID  path  string  true  "Redirection ID"
// @Success      200  {object}  core.Redirection
// @Failure      400  {string}  string  "redirectionID is required"
// @Failure      404  {string}  string  "Redirection not found"
// @Router       /admin/redirections/{redirectionID} [get]
func GetRedirection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	redirectionID := chi.URLParam(r, "redirectionID")
	if redirectionID == "" {
		http.Error(w, "redirectionID is required", http.StatusBadRequest)
		return
	}

	redirection, err := core.GetRedirectionByID(redirectionID)
	if err != nil {
		http.Error(w, "Redirection not found", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(redirection); err != nil {
		log.Printf("error encoding redirection: %s\n", err.Error())
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
