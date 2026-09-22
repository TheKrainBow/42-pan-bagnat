package redirections

import (
	"backend/core"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// PostRedirectionRole assigns a role to a redirection.
// @Summary      Add role to redirection
// @Tags         Redirections,Roles
// @Param        redirectionID  path  string  true  "Redirection ID"
// @Param        roleID         path  string  true  "Role ID"
// @Success      201  "Role successfully assigned to redirection"
// @Failure      404  {string}  string  "Redirection not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /admin/redirections/{redirectionID}/roles/{roleID} [post]
func PostRedirectionRole(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	roleID := chi.URLParam(r, "roleID")

	if _, err := core.GetRedirectionByID(redirectionID); err != nil {
		http.Error(w, "Redirection not found", http.StatusNotFound)
		return
	}

	if err := core.AssignRoleToRedirection(redirectionID, roleID); err != nil {
		http.Error(w, "Failed to assign role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// DeleteRedirectionRole removes a role from a redirection.
// @Summary      Remove role from redirection
// @Tags         Redirections,Roles
// @Param        redirectionID  path  string  true  "Redirection ID"
// @Param        roleID         path  string  true  "Role ID"
// @Success      204  "Role successfully removed from redirection"
// @Failure      404  {string}  string  "Redirection not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /admin/redirections/{redirectionID}/roles/{roleID} [delete]
func DeleteRedirectionRole(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	roleID := chi.URLParam(r, "roleID")

	if _, err := core.GetRedirectionByID(redirectionID); err != nil {
		http.Error(w, "Redirection not found", http.StatusNotFound)
		return
	}

	if err := core.RemoveRoleFromRedirection(redirectionID, roleID); err != nil {
		http.Error(w, "Failed to remove role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostRedirectionForbiddenRole adds a role to a redirection's forbidden list.
// @Summary      Add forbidden role to redirection
// @Tags         Redirections,Roles
// @Param        redirectionID  path  string  true  "Redirection ID"
// @Param        roleID         path  string  true  "Role ID"
// @Success      201  "Role successfully added to redirection's forbidden list"
// @Failure      404  {string}  string  "Redirection not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /admin/redirections/{redirectionID}/forbidden-roles/{roleID} [post]
func PostRedirectionForbiddenRole(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	roleID := chi.URLParam(r, "roleID")

	if _, err := core.GetRedirectionByID(redirectionID); err != nil {
		http.Error(w, "Redirection not found", http.StatusNotFound)
		return
	}

	if err := core.AssignForbiddenRoleToRedirection(redirectionID, roleID); err != nil {
		http.Error(w, "Failed to assign forbidden role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// DeleteRedirectionForbiddenRole removes a role from a redirection's forbidden list.
// @Summary      Remove forbidden role from redirection
// @Tags         Redirections,Roles
// @Param        redirectionID  path  string  true  "Redirection ID"
// @Param        roleID         path  string  true  "Role ID"
// @Success      204  "Role successfully removed from redirection's forbidden list"
// @Failure      404  {string}  string  "Redirection not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /admin/redirections/{redirectionID}/forbidden-roles/{roleID} [delete]
func DeleteRedirectionForbiddenRole(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	roleID := chi.URLParam(r, "roleID")

	if _, err := core.GetRedirectionByID(redirectionID); err != nil {
		http.Error(w, "Redirection not found", http.StatusNotFound)
		return
	}

	if err := core.RemoveForbiddenRoleFromRedirection(redirectionID, roleID); err != nil {
		http.Error(w, "Failed to remove forbidden role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
