package oidc

import (
	"backend/api/auth"
	"backend/core"
	"backend/database"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type moduleOIDCPatchRequest struct {
	Enabled             *bool    `json:"enabled"`
	AllowedRedirectURIs []string `json:"allowed_redirect_uris"`
	AllowedScopes       []string `json:"allowed_scopes"`
}

type oidcSecretResponse struct {
	ClientID            string  `json:"client_id"`
	ClientSecret        string  `json:"client_secret"`
	LastSecretRotatedAt *string `json:"last_secret_rotated_at,omitempty"`
}

type oidcAuthorizeError struct {
	Error string
	State string
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func GetDiscovery(w http.ResponseWriter, r *http.Request) {
	doc, err := core.BuildOIDCDiscoveryDocument()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func GetJWKS(w http.ResponseWriter, r *http.Request) {
	doc, err := core.BuildOIDCJWKS()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	responseType := strings.TrimSpace(q.Get("response_type"))
	clientID := strings.TrimSpace(q.Get("client_id"))
	redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
	scope := strings.TrimSpace(q.Get("scope"))
	state := q.Get("state")
	nonce := strings.TrimSpace(q.Get("nonce"))

	if responseType != "code" {
		writeAuthorizeError(w, r, redirectURI, state, "unsupported_response_type", http.StatusBadRequest)
		return
	}
	if clientID == "" {
		writeAuthorizeError(w, r, redirectURI, state, "invalid_client", http.StatusBadRequest)
		return
	}

	client, err := database.GetOIDCClientByClientID(clientID)
	if err != nil {
		writeAuthorizeError(w, r, redirectURI, state, "invalid_client", http.StatusBadRequest)
		return
	}
	if !client.Enabled {
		writeAuthorizeError(w, r, redirectURI, state, "unauthorized_client", http.StatusBadRequest)
		return
	}

	module, err := core.GetModule(client.ModuleID)
	if err != nil || strings.ToLower(string(module.Status)) != string(core.Enabled) {
		writeAuthorizeError(w, r, redirectURI, state, "unauthorized_client", http.StatusBadRequest)
		return
	}

	if !isRedirectURIAuthorized(client.AllowedRedirectURIs, redirectURI) {
		http.Error(w, "redirect_uri is not allowed", http.StatusBadRequest)
		return
	}

	requestedScopes := parseRequestedScopes(scope, client.AllowedScopes)
	if !containsScope(requestedScopes, "openid") {
		writeAuthorizeError(w, r, redirectURI, state, "invalid_scope", http.StatusBadRequest)
		return
	}
	for _, requested := range requestedScopes {
		if !containsScope(client.AllowedScopes, requested) {
			writeAuthorizeError(w, r, redirectURI, state, "invalid_scope", http.StatusBadRequest)
			return
		}
	}

	user, _ := r.Context().Value(auth.UserCtxKey).(*core.User)
	if user == nil {
		next := r.URL.RequestURI()
		loginURL := "/auth/42/login?next=" + url.QueryEscape(next)
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	canAccess, err := database.UserCanAccessModule(user.ID, module.ID)
	if err != nil || !canAccess {
		writeAuthorizeError(w, r, redirectURI, state, "access_denied", http.StatusForbidden)
		return
	}

	if nonce == "" {
		nonce = ""
	}

	codeValue, err := generateOpaqueValue(32)
	if err != nil {
		writeAuthorizeError(w, r, redirectURI, state, "server_error", http.StatusInternalServerError)
		return
	}
	codeHash := database.HashOIDCSecret(codeValue)
	now := time.Now().UTC()
	ttl := 120 * time.Second
	if c, err := core.ResolveOIDCConfigForAdmin(); err == nil {
		ttl = c.AuthCodeTTL
	}
	authCode := database.OIDCAuthorizationCode{
		CodeHash:    codeHash,
		ClientID:    client.ClientID,
		ModuleID:    module.ID,
		UserID:      user.ID,
		RedirectURI: redirectURI,
		Scopes:      requestedScopes,
		ExpiresAt:   now.Add(ttl),
	}
	if nonce != "" {
		authCode.Nonce = sqlNullString(nonce)
	}
	if err := database.CreateOIDCAuthorizationCode(authCode); err != nil {
		writeAuthorizeError(w, r, redirectURI, state, "server_error", http.StatusInternalServerError)
		return
	}

	u, _ := url.Parse(redirectURI)
	vals := u.Query()
	vals.Set("code", codeValue)
	if state != "" {
		vals.Set("state", state)
	}
	u.RawQuery = vals.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	grantType := strings.TrimSpace(r.Form.Get("grant_type"))
	if grantType != "authorization_code" {
		http.Error(w, "unsupported_grant_type", http.StatusBadRequest)
		return
	}

	clientID, clientSecret := extractClientCredentials(r)
	if clientID == "" {
		http.Error(w, "invalid_client", http.StatusUnauthorized)
		return
	}
	client, err := database.GetOIDCClientByClientID(clientID)
	if err != nil {
		http.Error(w, "invalid_client", http.StatusUnauthorized)
		return
	}
	if client.ClientType == database.OIDCClientTypeConfidential {
		if clientSecret == "" || database.HashOIDCSecret(clientSecret) != client.ClientSecretHash.String {
			http.Error(w, "invalid_client", http.StatusUnauthorized)
			return
		}
	}

	code := strings.TrimSpace(r.Form.Get("code"))
	redirectURI := strings.TrimSpace(r.Form.Get("redirect_uri"))
	if code == "" || redirectURI == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	codeHash := database.HashOIDCSecret(code)
	authCode, err := database.GetOIDCAuthorizationCodeByHash(codeHash)
	if err != nil {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	if authCode.ClientID != client.ClientID || authCode.RedirectURI != redirectURI || authCode.ConsumedAt.Valid {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	if time.Now().UTC().After(authCode.ExpiresAt) {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	if _, err := database.ConsumeOIDCAuthorizationCode(codeHash); err != nil {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	user, err := core.GetUser(authCode.UserID)
	if err != nil {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	module, err := core.GetModule(authCode.ModuleID)
	if err != nil {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	idClaims := core.BuildOIDCIDTokenClaims(user, module, *client, authCode.Scopes, nullStringValue(authCode.Nonce), now)
	idToken, err := core.SignOIDCToken(idClaims)
	if err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}

	accessToken, accessHash, err := generateTokenAndHash(32)
	if err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}
	if err := database.CreateOIDCAccessToken(database.OIDCAccessToken{
		TokenHash: accessHash,
		ClientID:  client.ClientID,
		ModuleID:  module.ID,
		UserID:    user.ID,
		Scopes:    authCode.Scopes,
		ExpiresAt: now.Add(resolveAccessTokenTTL()),
	}); err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, core.OIDCTokenResponse{
		AccessToken: accessToken,
		IDToken:     idToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(resolveAccessTokenTTL().Seconds()),
	})
}

func UserInfo(w http.ResponseWriter, r *http.Request) {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authz, core.OIDCBearerPrefix()) {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}
	rawToken := strings.TrimSpace(strings.TrimPrefix(authz, core.OIDCBearerPrefix()))
	if rawToken == "" {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}
	token, err := database.GetOIDCAccessTokenByHash(database.HashOIDCSecret(rawToken))
	if err != nil || token.RevokedAt.Valid || time.Now().UTC().After(token.ExpiresAt) {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}

	user, err := core.GetUser(token.UserID)
	if err != nil {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}
	module, err := core.GetModule(token.ModuleID)
	if err != nil {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}
	client, err := database.GetOIDCClientByClientID(token.ClientID)
	if err != nil {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}

	claims := core.BuildOIDCUserClaims(user, module, *client, token.Scopes, "")
	writeJSON(w, http.StatusOK, claims)
}

func GetModuleOIDC(w http.ResponseWriter, r *http.Request) {
	view, err := core.GetModuleOIDCView(chi.URLParam(r, "moduleID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func PatchModuleOIDC(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	client, err := database.GetOIDCClientByModuleID(moduleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var req moduleOIDCPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validateRedirectURIList(req.AllowedRedirectURIs) {
		http.Error(w, "invalid redirect uri", http.StatusBadRequest)
		return
	}
	if !validateScopeList(req.AllowedScopes) {
		http.Error(w, "invalid scope", http.StatusBadRequest)
		return
	}
	if _, err := database.UpdateOIDCClient(database.OIDCClientPatch{
		ID:                  client.ID,
		AllowedRedirectURIs: &req.AllowedRedirectURIs,
		AllowedScopes:       &req.AllowedScopes,
		Enabled:             req.Enabled,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	view, err := core.GetModuleOIDCView(moduleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func GenerateModuleOIDCSecret(w http.ResponseWriter, r *http.Request) {
	client, err := database.GetOIDCClientByModuleID(chi.URLParam(r, "moduleID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	secret, err := core.SetOIDCClientSecret(client.ClientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	view, _ := core.GetModuleOIDCView(client.ModuleID)
	payload := map[string]any{
		"client_id":              client.ClientID,
		"client_secret":          secret,
		"last_secret_rotated_at": view["last_secret_rotated_at"],
	}
	writeJSON(w, http.StatusOK, payload)
}

func RotateModuleOIDCSecret(w http.ResponseWriter, r *http.Request) {
	GenerateModuleOIDCSecret(w, r)
}

func AddModuleOIDCRedirectURI(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	client, err := database.GetOIDCClientByModuleID(moduleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var req struct {
		RedirectURI string `json:"redirect_uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uri := strings.TrimSpace(req.RedirectURI)
	if !isRedirectURIValid(uri) {
		http.Error(w, "invalid redirect uri", http.StatusBadRequest)
		return
	}
	if err := database.AddOIDCClientRedirectURI(client.ClientID, uri); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	view, _ := core.GetModuleOIDCView(moduleID)
	writeJSON(w, http.StatusOK, view)
}

func DeleteModuleOIDCRedirectURI(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	client, err := database.GetOIDCClientByModuleID(moduleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	uri := strings.TrimSpace(r.URL.Query().Get("uri"))
	if uri == "" {
		var req struct {
			RedirectURI string `json:"redirect_uri"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		uri = strings.TrimSpace(req.RedirectURI)
	}
	if uri == "" {
		http.Error(w, "invalid redirect uri", http.StatusBadRequest)
		return
	}
	if err := database.RemoveOIDCClientRedirectURI(client.ClientID, uri); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	view, _ := core.GetModuleOIDCView(moduleID)
	writeJSON(w, http.StatusOK, view)
}

func writeAuthorizeError(w http.ResponseWriter, r *http.Request, redirectURI, state, code string, status int) {
	if redirectURI == "" || !isRedirectURIValid(redirectURI) {
		http.Error(w, code, status)
		return
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, code, status)
		return
	}
	vals := u.Query()
	vals.Set("error", code)
	if state != "" {
		vals.Set("state", state)
	}
	u.RawQuery = vals.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func isRedirectURIAuthorized(allowed []string, candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	for _, uri := range allowed {
		if strings.TrimSpace(uri) == candidate {
			return true
		}
	}
	return false
}

func isRedirectURIValid(uri string) bool {
	if strings.Contains(uri, "*") {
		return false
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return false
	}
	return parsed.Host != ""
}

func validateRedirectURIList(values []string) bool {
	for _, uri := range values {
		if !isRedirectURIValid(strings.TrimSpace(uri)) {
			return false
		}
	}
	return true
}

func validateScopeList(values []string) bool {
	if len(values) == 0 {
		return false
	}
	hasOpenID := false
	for _, scope := range values {
		scope = strings.TrimSpace(scope)
		switch scope {
		case "openid":
			hasOpenID = true
		case "profile", "email", "roles":
		default:
			return false
		}
	}
	return hasOpenID
}

func parseRequestedScopes(scope string, allowed []string) []string {
	if scope == "" {
		return normalizeScopes(allowed)
	}
	return normalizeScopes(strings.Fields(scope))
}

func normalizeScopes(scopes []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func containsScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func extractClientCredentials(r *http.Request) (string, string) {
	if u, p, ok := r.BasicAuth(); ok {
		return strings.TrimSpace(u), p
	}
	return strings.TrimSpace(r.Form.Get("client_id")), r.Form.Get("client_secret")
}

func generateOpaqueValue(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateTokenAndHash(n int) (string, string, error) {
	token, err := generateOpaqueValue(n)
	if err != nil {
		return "", "", err
	}
	return token, database.HashOIDCSecret(token), nil
}

func resolveAccessTokenTTL() time.Duration {
	if cfg, err := core.ResolveOIDCConfigForAdmin(); err == nil {
		return cfg.AccessTokenTTL
	}
	return 15 * time.Minute
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func sqlNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
