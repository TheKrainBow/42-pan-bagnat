package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	modulesAccessSecret          = []byte(strings.TrimSpace(os.Getenv("MODULES_SESSION_SECRET")))
	modulesAccessTTL             = parseModulesAccessTTL()
	ErrModuleAccessSecretMissing = errors.New("MODULES_SESSION_SECRET is not configured")
)

type moduleAccessClaims struct {
	SessionID string `json:"sid"`
	Slug      string `json:"slug"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Nonce     string `json:"n"`
}

func parseModulesAccessTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("MODULES_SESSION_TOKEN_TTL"))
	if raw == "" {
		return 1 * time.Minute
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 1 * time.Minute
	}
	if d < time.Second {
		return time.Second
	}
	return d
}

// GenerateModuleAccessToken signs a short-lived token that allows the module proxy
// to recreate the user session cookie on the modules subdomain.
func GenerateModuleAccessToken(sessionID, slug string) (string, time.Time, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", time.Time{}, errors.New("missing session id")
	}
	if slug = strings.TrimSpace(slug); slug == "" {
		return "", time.Time{}, errors.New("missing page slug")
	}
	if len(modulesAccessSecret) == 0 {
		return "", time.Time{}, ErrModuleAccessSecretMissing
	}

	expiresAt := time.Now().Add(modulesAccessTTL)
	nonce, err := GenerateSecureSessionID()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate nonce: %w", err)
	}

	claims := moduleAccessClaims{
		SessionID: sessionID,
		Slug:      slug,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: expiresAt.Unix(),
		Nonce:     nonce,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	sig := hmac.New(sha256.New, modulesAccessSecret)
	sig.Write(payload)
	token := fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(sig.Sum(nil)),
	)
	return token, expiresAt, nil
}
