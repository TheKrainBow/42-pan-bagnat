package core

import (
	"backend/database"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultOIDCScopes = "openid profile email roles"
	oidcBearerPrefix  = "Bearer "
)

var (
	oidcConfigOnce sync.Once
	oidcConfig     OIDCConfig
	oidcConfigErr  error
)

type OIDCConfig struct {
	Issuer         string
	JWTPrivateKey  *rsa.PrivateKey
	JWTKeyID       string
	AuthCodeTTL    time.Duration
	AccessTokenTTL time.Duration
	IDTokenTTL     time.Duration
	DefaultScopes  []string
}

type OIDCUserClaims struct {
	Sub               string         `json:"sub"`
	Email             string         `json:"email,omitempty"`
	EmailVerified     bool           `json:"email_verified,omitempty"`
	Name              string         `json:"name,omitempty"`
	PreferredUsername string         `json:"preferred_username,omitempty"`
	Picture           string         `json:"picture,omitempty"`
	Module            map[string]any `json:"module,omitempty"`
	Roles             []string       `json:"roles,omitempty"`
	RoleSlugs         []string       `json:"role_slugs,omitempty"`
	Nonce             string         `json:"nonce,omitempty"`
}

type oidcIDTokenClaims struct {
	Issuer            string         `json:"iss"`
	Subject           string         `json:"sub"`
	Audience          string         `json:"aud"`
	ExpiresAt         int64          `json:"exp"`
	IssuedAt          int64          `json:"iat"`
	AuthTime          int64          `json:"auth_time"`
	Nonce             string         `json:"nonce,omitempty"`
	Email             string         `json:"email,omitempty"`
	EmailVerified     bool           `json:"email_verified,omitempty"`
	Name              string         `json:"name,omitempty"`
	PreferredUsername string         `json:"preferred_username,omitempty"`
	Picture           string         `json:"picture,omitempty"`
	Module            map[string]any `json:"module,omitempty"`
	Roles             []string       `json:"roles,omitempty"`
	RoleSlugs         []string       `json:"role_slugs,omitempty"`
}

type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type OIDCTokenResponse = oidcTokenResponse

func resolveOIDCConfig() (OIDCConfig, error) {
	oidcConfigOnce.Do(func() {
		issuer := strings.TrimSpace(os.Getenv("OIDC_ISSUER"))
		if issuer == "" {
			host := strings.TrimSpace(os.Getenv("HOST_NAME"))
			if host != "" {
				issuer = "https://" + host
			}
		}
		if issuer == "" {
			oidcConfigErr = errors.New("OIDC_ISSUER is not configured")
			return
		}
		issuer = strings.TrimRight(issuer, "/")

		key, kid, err := resolveOIDCPrivateKey()
		if err != nil {
			oidcConfigErr = err
			return
		}
		if overrideKid := strings.TrimSpace(os.Getenv("OIDC_JWT_KEY_ID")); overrideKid != "" {
			kid = overrideKid
		}

		oidcConfig = OIDCConfig{
			Issuer:         issuer,
			JWTPrivateKey:  key,
			JWTKeyID:       kid,
			AuthCodeTTL:    parseOIDCDuration("OIDC_AUTH_CODE_TTL_SECONDS", 120*time.Second),
			AccessTokenTTL: parseOIDCDuration("OIDC_ACCESS_TOKEN_TTL_SECONDS", 15*time.Minute),
			IDTokenTTL:     parseOIDCDuration("OIDC_ID_TOKEN_TTL_SECONDS", 15*time.Minute),
			DefaultScopes:  parseOIDCScopes(),
		}
	})
	return oidcConfig, oidcConfigErr
}

func ResolveOIDCConfigForAdmin() (OIDCConfig, error) {
	return resolveOIDCConfig()
}

func parseOIDCDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if strings.ContainsAny(raw, "hms") {
		if d, err := time.ParseDuration(raw); err == nil {
			return d
		}
	}
	if secs, err := time.ParseDuration(raw + "s"); err == nil {
		return secs
	}
	return fallback
}

func parseOIDCScopes() []string {
	raw := strings.TrimSpace(os.Getenv("OIDC_DEFAULT_SCOPES"))
	if raw == "" {
		raw = defaultOIDCScopes
	}
	return normalizeStringList(strings.Fields(raw))
}

func resolveOIDCPrivateKey() (*rsa.PrivateKey, string, error) {
	pemData := strings.TrimSpace(os.Getenv("OIDC_JWT_PRIVATE_KEY_PEM"))
	pemPath := strings.TrimSpace(os.Getenv("OIDC_JWT_PRIVATE_KEY_PATH"))

	if pemData == "" && pemPath != "" {
		content, err := os.ReadFile(filepath.Clean(pemPath))
		if err != nil {
			return nil, "", fmt.Errorf("read OIDC private key path: %w", err)
		}
		pemData = string(content)
	}

	if pemData == "" {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, "", fmt.Errorf("generate OIDC private key: %w", err)
		}
		return key, fingerprintRSAKey(&key.PublicKey), nil
	}

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, "", errors.New("OIDC private key PEM is invalid")
	}

	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		if pkcs1, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
			return pkcs1, fingerprintRSAKey(&pkcs1.PublicKey), nil
		}
		return nil, "", fmt.Errorf("parse OIDC private key: %w", err)
	}

	key, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, "", errors.New("OIDC private key must be RSA")
	}
	return key, fingerprintRSAKey(&key.PublicKey), nil
}

func fingerprintRSAKey(pub *rsa.PublicKey) string {
	if pub == nil {
		return "panbagnat-main"
	}
	sum := sha256.Sum256(append(pub.N.Bytes(), byte(pub.E)))
	return hex.EncodeToString(sum[:8])
}

func normalizeStringList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeScopeValues(values []string) []string {
	return normalizeStringList(values)
}

func normalizeClientID(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	slug = strings.ReplaceAll(slug, "-", "_")
	slug = strings.ReplaceAll(slug, " ", "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		slug = "module"
	}
	return "panbagnat_" + slug
}

func normalizeRoleSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "role"
	}
	return out
}

func OIDCBearerPrefix() string {
	return oidcBearerPrefix
}

func BuildOIDCDiscoveryDocument() (map[string]any, error) {
	cfg, err := resolveOIDCConfig()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"issuer":                                cfg.Issuer,
		"authorization_endpoint":                cfg.Issuer + "/oauth/authorize",
		"token_endpoint":                        cfg.Issuer + "/oauth/token",
		"userinfo_endpoint":                     cfg.Issuer + "/oauth/userinfo",
		"jwks_uri":                              cfg.Issuer + "/.well-known/jwks.json",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      cfg.DefaultScopes,
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
		"claims_supported": []string{
			"sub", "email", "email_verified", "name", "preferred_username", "picture", "module", "roles", "role_slugs",
		},
	}, nil
}

func BuildOIDCJWKS() (map[string]any, error) {
	cfg, err := resolveOIDCConfig()
	if err != nil {
		return nil, err
	}
	pub := cfg.JWTPrivateKey.Public().(*rsa.PublicKey)
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": cfg.JWTKeyID,
				"n":   n,
				"e":   e,
			},
		},
	}, nil
}

func SignOIDCToken(claims any) (string, error) {
	cfg, err := resolveOIDCConfig()
	if err != nil {
		return "", err
	}
	header := map[string]any{
		"alg": "RS256",
		"typ": "JWT",
		"kid": cfg.JWTKeyID,
	}
	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encHeader := base64.RawURLEncoding.EncodeToString(hb)
	encPayload := base64.RawURLEncoding.EncodeToString(pb)
	unsigned := encHeader + "." + encPayload
	sum := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, cfg.JWTPrivateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func BuildOIDCUserClaims(user User, module Module, _ database.OIDCClient, scopes []string, nonce string) OIDCUserClaims {
	includeProfile := containsScope(scopes, "profile")
	includeEmail := containsScope(scopes, "email")
	includeRoles := containsScope(scopes, "roles")

	claims := OIDCUserClaims{
		Sub:   user.ID,
		Nonce: nonce,
	}
	if includeProfile {
		claims.Name = user.FtLogin
		claims.PreferredUsername = user.FtLogin
		claims.Picture = user.PhotoURL
	}
	if includeEmail {
		claims.Email = buildOIDCEmail(user)
		claims.EmailVerified = claims.Email != ""
	}
	if includeRoles {
		claims.Module = map[string]any{
			"id":   module.ID,
			"slug": module.Slug,
			"name": module.Name,
		}
		roles, err := database.GetUserModuleRoles(user.ID, module.ID)
		if err == nil {
			for _, role := range roles {
				claims.Roles = append(claims.Roles, role.Name)
				claims.RoleSlugs = append(claims.RoleSlugs, normalizeRoleSlug(role.Name))
			}
		}
	}
	return claims
}

func buildOIDCEmail(user User) string {
	login := strings.TrimSpace(strings.ToLower(user.FtLogin))
	if login == "" {
		return ""
	}
	if user.FtIsStaff {
		return login + "@42nice.fr"
	}
	return login + "@student.42nice.fr"
}

func BuildOIDCIDTokenClaims(user User, module Module, client database.OIDCClient, scopes []string, nonce string, now time.Time) oidcIDTokenClaims {
	claims := BuildOIDCUserClaims(user, module, client, scopes, nonce)
	cfg, _ := resolveOIDCConfig()
	return oidcIDTokenClaims{
		Issuer:            mustOIDCIssuer(),
		Subject:           claims.Sub,
		Audience:          client.ClientID,
		ExpiresAt:         now.Add(cfg.IDTokenTTL).Unix(),
		IssuedAt:          now.Unix(),
		AuthTime:          now.Unix(),
		Nonce:             claims.Nonce,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		Name:              claims.Name,
		PreferredUsername: claims.PreferredUsername,
		Picture:           claims.Picture,
		Module:            claims.Module,
		Roles:             claims.Roles,
		RoleSlugs:         claims.RoleSlugs,
	}
}

func resolveOIDCAccessTokenTTL() time.Duration {
	if cfg, err := resolveOIDCConfig(); err == nil {
		return cfg.AccessTokenTTL
	}
	return 15 * time.Minute
}

func mustOIDCIssuer() string {
	cfg, err := resolveOIDCConfig()
	if err != nil {
		return ""
	}
	return cfg.Issuer
}

func containsScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func ensureOIDCClientForModule(module Module) (*database.OIDCClient, error) {
	existing, err := database.GetOIDCClientByModuleID(module.ID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	client := database.OIDCClient{
		ModuleID:            module.ID,
		Name:                module.Name,
		ClientID:            normalizeClientID(module.Slug),
		ClientType:          database.OIDCClientTypeConfidential,
		AllowedRedirectURIs: []string{},
		AllowedScopes:       normalizeScopeValues(strings.Fields(defaultOIDCScopes)),
		Enabled:             module.Status == Enabled,
	}
	return database.InsertOIDCClient(client)
}

func BackfillOIDCClients() error {
	modules, err := database.GetAllModules(nil, "", nil, 0)
	if err != nil {
		return err
	}
	for _, module := range modules {
		if _, err := ensureOIDCClientForModule(DatabaseModuleToModule(module)); err != nil {
			return err
		}
	}
	return nil
}

func GetModuleOIDCView(moduleID string) (map[string]any, error) {
	module, err := GetModule(moduleID)
	if err != nil {
		return nil, err
	}
	client, err := database.GetOIDCClientByModuleID(moduleID)
	if err != nil {
		return nil, err
	}
	cfg, err := resolveOIDCConfig()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"module_id":              module.ID,
		"module_slug":            module.Slug,
		"enabled":                client.Enabled,
		"client_id":              client.ClientID,
		"client_type":            client.ClientType,
		"has_client_secret":      client.ClientSecretHash.Valid,
		"last_secret_rotated_at": nullTimeToString(client.LastSecretRotatedAt),
		"issuer":                 cfg.Issuer,
		"discovery_url":          cfg.Issuer + "/.well-known/openid-configuration",
		"authorization_url":      cfg.Issuer + "/oauth/authorize",
		"token_url":              cfg.Issuer + "/oauth/token",
		"userinfo_url":           cfg.Issuer + "/oauth/userinfo",
		"jwks_url":               cfg.Issuer + "/.well-known/jwks.json",
		"allowed_redirect_uris":  client.AllowedRedirectURIs,
		"allowed_scopes":         client.AllowedScopes,
	}, nil
}

func nullTimeToString(v sql.NullTime) any {
	if !v.Valid {
		return nil
	}
	return v.Time.UTC().Format(time.RFC3339)
}

func GenerateOIDCSecret() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	hash := database.HashOIDCSecret(secret)
	return secret, hash, nil
}

func SetOIDCClientSecret(clientID string) (string, error) {
	secret, hash, err := GenerateOIDCSecret()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := database.SetOIDCClientSecretHash(clientID, &hash, now); err != nil {
		return "", err
	}
	return secret, nil
}
