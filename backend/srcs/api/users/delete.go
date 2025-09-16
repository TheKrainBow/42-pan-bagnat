package users

import (
    "backend/api/auth"
    "backend/core"
    "backend/database"
    "errors"
    "log"
    "net/http"
    "strings"

    "github.com/go-chi/chi/v5"
)

// DeleteUser deletes a user by ID.
// @Summary      Delete User
// @Description  Deletes the specified user and all associated data.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        userID  path      string  true  "User ID"
// @Success      204     {string}  string  "No Content"
// @Failure      400     {string}  string  "Invalid user ID"
// @Failure      404     {string}  string  "User not found"
// @Failure      500     {string}  string  "Internal server error"
// @Router       /admin/users/{userID} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(r, "userID")
	if strings.TrimSpace(userID) == "" {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// err := core.DeleteUser(userID)
	// if err != nil {
	// 	if errors.Is(err, core.ErrNotFound) {
	// 		http.Error(w, "User not found", http.StatusNotFound)
	// 	} else {
	// 		log.Printf("error deleting user %s: %v\n", userID, err)
	// 		http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	}
	// 	return
	// }

	w.WriteHeader(http.StatusNoContent)

}

// DeleteUserRole revokes a specific role from a user.
// @Summary      Remove Role from User
// @Description  Revokes the specified role from the user identified by ID or login.
// @Tags         Users,Roles
// @Accept       json
// @Produce      json
// @Param        identifier  path      string  true  "User identifier (ID or login)"
// @Param        roleID      path      string  true  "Role ID"
// @Success      204         {string}  string  "No Content"
// @Failure      400         {string}  string  "Invalid identifier or roleID"
// @Failure      404         {string}  string  "User or role not found"
// @Failure      500         {string}  string  "Internal server error"
// @Router       /admin/users/{identifier}/roles/{roleID} [delete]
func DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	roleID := chi.URLParam(r, "roleID")

	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(roleID) == "" {
		http.Error(w, "Invalid identifier or roleID", http.StatusBadRequest)
		return
	}

	err := core.DeleteRoleFromUser(roleID, identifier)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotFound):
			http.Error(w, "User or role not found", http.StatusNotFound)
		case errors.Is(err, core.ErrWouldRemoveLastAdmin):
			auth.WriteJSONError(w, http.StatusConflict, "Conflict", "Cannot remove the last admin user")
		default:
			log.Printf("error deleting role %s from user %s: %v\n", roleID, identifier, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteUserMe deletes the currently authenticated user.
// @Summary      Delete Current User
// @Description  Deletes the authenticated user account, revoking sessions and removing role links.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      204  {string}  string  "No Content"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /users/me [delete]
func DeleteUserMe(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
    if !ok || u == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    if err := core.DeleteUserAndAssociations(r.Context(), u.ID); err != nil {
        log.Printf("error deleting current user %s: %v\n", u.ID, err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Also clear any session cookie in the client
    core.ClearSessionCookie(w)
    w.WriteHeader(http.StatusNoContent)
}

// DeleteUserSession revokes a specific session for the current user
// @Summary      Revoke a session
// @Description  Deletes one session (device) of the current user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        sessionID  path      string  true  "Session ID"
// @Success      204        {string}  string  "No Content"
// @Failure      401        {string}  string  "Unauthorized"
// @Failure      403        {string}  string  "Forbidden"
// @Failure      500        {string}  string  "Internal server error"
// @Router       /users/me/sessions/{sessionID} [delete]
func DeleteUserSession(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
    if !ok || u == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    sessionID := chi.URLParam(r, "sessionID")
    if strings.TrimSpace(sessionID) == "" {
        http.Error(w, "Invalid session ID", http.StatusBadRequest)
        return
    }

    sess, err := database.GetSession(sessionID)
    if err == nil && sess != nil {
        if sess.Login != u.FtLogin {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
    }
    _ = database.DeleteSession(sessionID)
    if core.ReadSessionIDFromCookie(r) == sessionID {
        core.ClearSessionCookie(w)
    }
    w.WriteHeader(http.StatusNoContent)
}

// DeleteUserSessions revokes all sessions for the current user
// @Summary      Revoke all sessions
// @Description  Deletes all sessions (devices) for the current user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      204        {string}  string  "No Content"
// @Failure      401        {string}  string  "Unauthorized"
// @Failure      500        {string}  string  "Internal server error"
// @Router       /users/me/sessions [delete]
func DeleteUserSessions(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    u, ok := r.Context().Value(auth.UserCtxKey).(*core.User)
    if !ok || u == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    _, err := database.DeleteUserSessions(r.Context(), u.ID)
    if err != nil {
        log.Printf("error deleting sessions for user %s: %v\n", u.ID, err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    core.ClearSessionCookie(w)
    w.WriteHeader(http.StatusNoContent)
}
