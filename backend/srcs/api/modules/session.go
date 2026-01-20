package modules

import (
	"backend/api/auth"
	"backend/core"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// IssueModulePageSession exchanges the current Pan Bagnat session for a temporary
// token that the modules proxy can validate to recreate the session cookie on the
// module subdomain.
// @Security     SessionAuth
// @Summary      Issue module page session token
// @Description  Generates a short-lived token that allows the proxy to authenticate requests to the specified module page.
// @Tags         Pages
// @Accept       json
// @Produce      json
// @Param        slug   path      string  true  "Module page slug"
// @Success      200    {object}  ModulePageSessionResponse
// @Failure      400    {string}  string  "Bad request"
// @Failure      401    {string}  string  "Unauthorized"
// @Failure      403    {string}  string  "Forbidden"
// @Failure      500    {string}  string  "Internal server error"
// @Router       /modules/pages/{slug}/session [post]
func IssueModulePageSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		auth.WriteJSONError(w, http.StatusBadRequest, "missing_slug", "Module page slug is required.")
		return
	}

	u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
	if !ok || u == nil {
		auth.WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Please sign in.")
		return
	}

	allowed, err := core.UserCanAccessPage(u.ID, slug)
	if err != nil {
		log.Printf("module session: failed to check access for user %s on %s: %v", u.ID, slug, err)
		auth.WriteJSONError(w, http.StatusInternalServerError, "access_check_failed", "Unable to verify if you can access this module page.")
		return
	}
	if !allowed {
		auth.WriteJSONError(w, http.StatusForbidden, "forbidden", "You are not allowed to access this module page.")
		return
	}

	sessionID := core.ReadSessionIDFromCookie(r)
	if sessionID == "" {
		auth.WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Please sign in.")
		return
	}

	token, expiresAt, err := core.GenerateModuleAccessToken(sessionID, slug)
	if err != nil {
		if errors.Is(err, core.ErrModuleAccessSecretMissing) {
			auth.WriteJSONError(w, http.StatusInternalServerError, "server_error", "Modules session secret is not configured.")
			return
		}
		log.Printf("module session: failed to create token for %s: %v", slug, err)
		auth.WriteJSONError(w, http.StatusInternalServerError, "token_generation_failed", "Unable to generate access token.")
		return
	}

	resp := ModulePageSessionResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("module session: failed to encode response: %v", err)
	}
}
