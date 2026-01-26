package auth

import (
	"backend/core"
	"backend/database"
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const loginRedirectCookieName = "pb_login_next"
const loginRedirectTTL = 5 * time.Minute

var allowedModuleRedirectDomains = parseEnvList("MODULES_PROXY_ALLOWED_DOMAINS")
var allowedLoginHosts = parseEnvList("MODULES_IFRAME_ALLOWED_HOSTS")

func getOAuthConf() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("FT_CLIENT_ID"),
		ClientSecret: os.Getenv("FT_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FT_CALLBACK_URL"),
		// Request the minimal scope required to read /v2/me
		Scopes: []string{"public"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.intra.42.fr/oauth/authorize",
			TokenURL: "https://api.intra.42.fr/oauth/token",
		},
	}
}

// GET /auth/42/login
func StartLogin(w http.ResponseWriter, r *http.Request) {
	nextParam := strings.TrimSpace(r.URL.Query().Get("next"))
	secure := isHTTPSRequest(r)
	if nextParam != "" {
		setLoginRedirectCookie(w, nextParam, secure)
	} else {
		clearLoginRedirectCookie(w, secure)
	}
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
	ua := r.UserAgent()
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}

	sessionID, err := core.HandleUser42Connection(r.Context(), token, core.DeviceMeta{
		UserAgent: ua,
		IP:        ip,
		// DeviceLabel: optionally read from a cookie/query param for named devices
	})
	if err != nil {
		http.Error(w, "Auth failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	isHTTPS := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	core.WriteSessionCookie(w, sessionID, 24*time.Hour, isHTTPS)

	nextRedirect := readLoginRedirectCookie(r)
	clearLoginRedirectCookie(w, isHTTPS)
	if target, ok := sanitizeRedirectURL(nextRedirect); ok {
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	redirectHome()
}

// POST /auth/logout
// Clears the current session cookie and deletes the server-side session.
func Logout(w http.ResponseWriter, r *http.Request) {
	sid := core.ReadSessionIDFromCookie(r)
	if sid != "" {
		_ = database.DeleteSession(sid)
	}
	core.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func setLoginRedirectCookie(w http.ResponseWriter, raw string, secure bool) {
	if raw == "" {
		clearLoginRedirectCookie(w, secure)
		return
	}
	value := url.QueryEscape(raw)
	domain := core.SessionCookieDomain()
	http.SetCookie(w, &http.Cookie{
		Name:     loginRedirectCookieName,
		Value:    value,
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(loginRedirectTTL.Seconds()),
		Expires:  time.Now().Add(loginRedirectTTL),
	})
}

func clearLoginRedirectCookie(w http.ResponseWriter, secure bool) {
	domain := core.SessionCookieDomain()
	http.SetCookie(w, &http.Cookie{
		Name:     loginRedirectCookieName,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func readLoginRedirectCookie(r *http.Request) string {
	c, err := r.Cookie(loginRedirectCookieName)
	if err != nil || c.Value == "" {
		return ""
	}
	val, err := url.QueryUnescape(c.Value)
	if err != nil {
		return ""
	}
	return val
}

func sanitizeRedirectURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "/") {
		return raw, true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if !isAllowedRedirectHost(host) {
		return "", false
	}
	return u.String(), true
}

func isAllowedRedirectHost(host string) bool {
	if host == "" {
		return false
	}
	host = strings.ToLower(host)
	for _, allowed := range allowedLoginHosts {
		if host == allowed {
			return true
		}
	}
	for _, suffix := range allowedModuleRedirectDomains {
		if host == suffix {
			return true
		}
		if strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func parseEnvList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		val := strings.ToLower(strings.TrimSpace(part))
		if val != "" {
			out = append(out, val)
		}
	}
	return out
}

func isHTTPSRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
