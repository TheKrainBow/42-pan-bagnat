package auth

import (
	"backend/core"
	"backend/database"
	"context"
	"log"
	"net/http"
	"time"
)

type contextKey string

const UserCtxKey contextKey = "user"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			log.Println("[auth] no session_id cookie:", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		session, err := database.GetSession(cookie.Value)
		if err != nil || session.ExpiresAt.Before(time.Now()) {
			log.Println("[auth] invalid/expired session:", err)
			go database.PurgeExpiredSessions()
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
			go core.TouchUserLastSeen(user.ID)
		}

		ctx := context.WithValue(r.Context(), UserCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func PageAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: later check if the page is public or requires auth
		next.ServeHTTP(w, r)
	})
}
