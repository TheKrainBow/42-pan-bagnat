package auth

import (
	"backend/core"
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func getOAuthConf() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("FT_CLIENT_ID"),
		ClientSecret: os.Getenv("FT_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FT_CALLBACK_URL"),
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.intra.42.fr/oauth/authorize",
			TokenURL: "https://api.intra.42.fr/oauth/token",
		},
	}
}

// GET /auth/42/login
func StartLogin(w http.ResponseWriter, r *http.Request) {
	url := getOAuthConf().AuthCodeURL("random-state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
}

// GET /auth/42/callback
func Callback(w http.ResponseWriter, r *http.Request) {
	redirectHome := func() {
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	if c, err := r.Cookie("session_id"); err == nil && c.Value != "" {
		redirectHome()
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	token, err := getOAuthConf().Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sessionID, err := core.HandleUser42Connection(token)
	if err != nil {
		http.Error(w, "Auth failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	isHTTPS := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS, // only if served over HTTPS!
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	redirectHome()
}
