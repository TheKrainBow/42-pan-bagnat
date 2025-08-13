package core

import (
	"backend/database"
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

type DeviceMeta struct {
	UserAgent   string
	IP          string
	DeviceLabel string
}

const sessionExpireCooldown = 24 * time.Hour
const sessionMaxExpire = 30 * 24 * time.Hour
const sessionUpdateThrottle = 60 * time.Second

func TouchSession(ctx context.Context, sessionID string) error {
	_, _, _, _, err := database.TouchSessionMaybe(ctx, sessionID, time.Now(), sessionExpireCooldown, sessionMaxExpire, sessionUpdateThrottle)
	return err
}

func EnsureDeviceSession(ctx context.Context, login string, meta DeviceMeta) (string, error) {
	now := time.Now()

	// Try reuse an active session for this device
	if s, err := database.FindActiveSessionForDevice(login, meta.UserAgent, meta.IP, now); err == nil && s != nil {
		database.TouchSessionMaybe(ctx, s.SessionID, time.Now(), sessionExpireCooldown, sessionMaxExpire, sessionUpdateThrottle)
		return s.SessionID, nil
	}

	// Else, create a fresh session
	sid, err := GenerateSecureSessionID()
	if err != nil {
		return "", err
	}
	s := database.Session{
		SessionID:   sid,
		Login:       login,
		CreatedAt:   now,
		ExpiresAt:   now.Add(sessionExpireCooldown),
		UserAgent:   meta.UserAgent,
		IP:          meta.IP,
		DeviceLabel: meta.DeviceLabel,
		LastSeen:    now,
	}
	if err := database.AddSession(s); err != nil {
		return "", err
	}
	return sid, nil
}

func GenerateSecureSessionID() (string, error) {
	b := make([]byte, 32) // 256-bit
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

const SessionCookieName = "session_id"

// ReadSessionIDFromCookie returns the session ID from the cookie if present.
// Falls back to "X-Session-Id" header or "Authorization: Bearer <id>" for dev/tools.
func ReadSessionIDFromCookie(r *http.Request) string {
	if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if v := r.Header.Get("X-Session-Id"); v != "" {
		return v
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

// WriteSessionCookie sets the session cookie with reasonable defaults.
// Call this right after you create (or reuse) a session.
func WriteSessionCookie(w http.ResponseWriter, sessionID string, ttl time.Duration, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()), // or omit for session-only
		Expires:  time.Now().Add(ttl),
	})
}

// ClearSessionCookie removes the cookie (e.g., on logout or blacklist).
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
