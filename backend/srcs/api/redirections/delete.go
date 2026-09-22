package redirections

import (
	"backend/core"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DeleteRedirection deletes a redirection by its ID.
// @Summary      Delete Redirection
// @Description  Deletes the specified redirection (all its role assignments are lost).
// @Tags         Redirections
// @Param        redirectionID  path  string  true  "Redirection ID"
// @Success      204  "No Content"
// @Failure      400  {string}  string  "Invalid redirection ID"
// @Failure      404  {string}  string  "Redirection not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /admin/redirections/{redirectionID} [delete]
func DeleteRedirection(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	if strings.TrimSpace(redirectionID) == "" {
		http.Error(w, "redirectionID is required", http.StatusBadRequest)
		return
	}

	if err := core.DeleteRedirectionByID(redirectionID); err != nil {
		http.Error(w, "Redirection not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
