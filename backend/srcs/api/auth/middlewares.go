package auth

import (
	"backend/core"
	"backend/database"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type contextKey string

const UserCtxKey contextKey = "user"
const PageCtxKey contextKey = "page"

type APIError struct {
	Error   string `json:"error"`             // e.g. "forbidden"
	Code    string `json:"code,omitempty"`    // e.g. "blacklisted"
	Message string `json:"message,omitempty"` // user-friendly text
}

func WriteJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	if code != "" {
		w.Header().Set("X-Error-Code", code) // optional: easy to read from fetch()
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{
		Error:   http.StatusText(status),
		Code:    code,
		Message: message,
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := core.ReadSessionIDFromCookie(r)
		if sid == "" {
			log.Println("[auth] no session_id")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		session, err := database.GetSession(sid)
		if err != nil {
			// Only clear cookie if session definitely doesn't exist; for transient DB errors keep cookie
			if err == sql.ErrNoRows {
				log.Println("[auth] no such session, clearing cookie")
				core.ClearSessionCookie(w)
			} else {
				log.Printf("[auth] session lookup error: %v", err)
			}
			go database.PurgeExpiredSessions()
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if session.ExpiresAt.Before(time.Now()) {
			log.Println("[auth] expired session")
			go database.PurgeExpiredSessions()
			core.ClearSessionCookie(w)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := core.GetUser(session.Login)
		if err != nil {
			log.Println("[auth] user not found for session:", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("[auth] user %s authenticated via session", user.FtLogin)

		if time.Since(user.LastSeen) > time.Minute {
			go core.TouchUserLastSeen(user.FtLogin)
		}

		ctx := context.WithValue(r.Context(), UserCtxKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(UserCtxKey).(*core.User)
		if !ok || u == nil {
			// Not authenticated → 401 so the SPA can redirect to /login
			WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Please sign in.")
			return
		}

		isAdmin, err := database.UserHasRoleByID(r.Context(), u.ID, core.RoleIDAdmin)
		if err != nil {
			log.Printf("admin check failed for user %s: %v", u.ID, err)
			WriteJSONError(w, http.StatusInternalServerError, "server_error", "Unable to verify permissions.")
			return
		}
		if !isAdmin {
			WriteJSONError(w, http.StatusForbidden, "admin_required", "You are not allowed to view this content.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func BlackListMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(UserCtxKey).(*core.User)
		if !ok {
			fmt.Printf("Couldn't get user\n")
			http.Redirect(w, r, "/", http.StatusForbidden)
			return
		}
		hasBlacklist, err := database.UserHasRoleByID(r.Context(), u.ID, "roles_blacklist")
		if err != nil {
			log.Printf("BlacklistGuard: role check failed for user %s: %v", u.ID, err)
			next.ServeHTTP(w, r)
			return
		}
		if !hasBlacklist {
			next.ServeHTTP(w, r)
			return
		}

		n, err := database.DeleteUserSessions(r.Context(), u.ID)
		if err != nil {
			log.Printf("couldn't delete user %s sessions: %s\n", u.FtLogin, err.Error())
		} else {
			log.Printf("deleted %d sessions for user %s\n", n, u.FtLogin)
		}
		core.ClearSessionCookie(w)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		WriteJSONError(w, http.StatusForbidden, "blacklisted", "Your account is currently blacklisted. Contact your bocal.")
		fmt.Printf("[auth] user %s is blacklisted, returned 403 Forbidden\n", u.FtLogin)
	})
}

func PageAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageName, ok := r.Context().Value(PageCtxKey).(string)
		if !ok || pageName == "" {
			http.Error(w, "Missing page name", http.StatusBadRequest)
			return
		}

		page, err := core.GetPage(pageName)
		if err != nil {
			http.Error(w, "Page not found: "+err.Error(), http.StatusNotFound)
			return
		}

		if page.IsPublic {
			next.ServeHTTP(w, r)
			return
		}

		user, ok := r.Context().Value(UserCtxKey).(*core.User)
		if !ok || user == nil {
			http.Error(w, "Unauthorized — this page requires login", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
