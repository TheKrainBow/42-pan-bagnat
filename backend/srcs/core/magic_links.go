package core

import (
	"backend/database"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	magicLinkTTL          = 15 * time.Minute
	magicLinkSendCooldown = 2 * time.Minute
)

var ErrMagicLinkInvalid = errors.New("invalid magic link")

func RequestMagicLink(ctx context.Context, email string, nextURL string) error {
	email = normalizeEmail(email)
	if email == "" {
		return nil
	}

	user, err := database.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("lookup user by email: %w", err)
	}

	now := time.Now().UTC()
	lastSentAt, err := database.GetLastMagicLinkSentAtForUser(ctx, user.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lookup last magic link sent time: %w", err)
	}
	if err == nil && now.Sub(lastSentAt) < magicLinkSendCooldown {
		return nil
	}

	rawToken := ""
	tokenHash := ""
	expiresAt := now.Add(magicLinkTTL)

	activeToken, err := database.GetActiveMagicLinkTokenForUser(ctx, user.ID, now)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lookup active magic link token: %w", err)
	}
	if activeToken != nil {
		rawToken = strings.TrimSpace(activeToken.TokenValue)
		if rawToken == "" {
			return nil
		}
		tokenHash = activeToken.TokenHash
		expiresAt = activeToken.ExpiresAt
	} else {
		rawToken, err = GenerateSecureSessionID()
		if err != nil {
			return err
		}
		tokenHash = hashMagicLinkToken(rawToken)
		if err := database.CreateMagicLinkToken(ctx, database.MagicLinkToken{
			TokenHash:  tokenHash,
			TokenValue: rawToken,
			UserID:     user.ID,
			NextURL:    nextURL,
			ExpiresAt:  expiresAt,
		}); err != nil {
			return fmt.Errorf("create magic link token: %w", err)
		}
	}

	link, err := buildMagicLinkURL(rawToken)
	if err != nil {
		return err
	}
	if err := sendMagicLinkEmail(email, link, expiresAt); err != nil {
		return err
	}
	if err := database.TouchMagicLinkTokenSentAt(ctx, tokenHash, now); err != nil {
		return fmt.Errorf("update magic link sent time: %w", err)
	}
	return nil
}

func ConsumeMagicLink(ctx context.Context, token string, meta DeviceMeta) (string, string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", ErrMagicLinkInvalid
	}

	user, nextURL, err := database.ConsumeMagicLinkToken(ctx, hashMagicLinkToken(token), time.Now().UTC())
	if err != nil {
		return "", "", ErrMagicLinkInvalid
	}

	_ = database.UpdateUserLastSeen(user.FtLogin, time.Now().UTC())
	sessionID, err := EnsureDeviceSession(ctx, user.FtLogin, meta)
	if err != nil {
		return "", "", fmt.Errorf("failed to ensure session: %w", err)
	}
	return sessionID, nextURL, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashMagicLinkToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func buildMagicLinkURL(token string) (string, error) {
	base := strings.TrimSpace(os.Getenv("MAGIC_LINK_BASE_URL"))
	if base == "" {
		host := strings.TrimSpace(os.Getenv("HOST_NAME"))
		if host == "" {
			return "", errors.New("MAGIC_LINK_BASE_URL or HOST_NAME must be set")
		}
		base = "https://" + host
	}
	base = strings.TrimRight(base, "/")
	u, err := url.Parse(base + "/auth/magic/callback")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func sendMagicLinkEmail(to string, link string, expiresAt time.Time) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if host == "" || from == "" {
		return errors.New("SMTP_HOST and SMTP_FROM must be set")
	}

	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	addr := host + ":" + port

	subject := "Your Pan Bagnat sign-in link"
	body := fmt.Sprintf(
		"Use this link to sign in to Pan Bagnat:\r\n\r\n%s\r\n\r\nThis link expires at %s and can only be used once.\r\n",
		link,
		expiresAt.Format(time.RFC1123),
	)
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	password := os.Getenv("SMTP_PASSWORD")
	if username != "" || password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid SMTP_PORT: %w", err)
	}
	if err := sendSMTP(addr, host, resolveSMTPHelloHost(from), auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send magic link email via %s: %w", addr, err)
	}
	return nil
}

func resolveSMTPHelloHost(from string) string {
	if explicit := strings.TrimSpace(os.Getenv("SMTP_HELO_HOST")); isUsableSMTPHelloHost(explicit) {
		return explicit
	}
	if explicit := strings.TrimSpace(os.Getenv("SMTP_HELO")); isUsableSMTPHelloHost(explicit) {
		return explicit
	}
	if host := strings.TrimSpace(os.Getenv("HOST_NAME")); isUsableSMTPHelloHost(host) {
		return host
	}
	if at := strings.LastIndex(from, "@"); at >= 0 && at < len(from)-1 {
		if domain := strings.TrimSpace(from[at+1:]); isUsableSMTPHelloHost(domain) {
			return domain
		}
	}
	if host, err := os.Hostname(); err == nil && isUsableSMTPHelloHost(host) {
		return host
	}
	return ""
}

func isUsableSMTPHelloHost(host string) bool {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if host == "" {
		return false
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasPrefix(lower, "localhost.") {
		return false
	}
	if net.ParseIP(host) != nil {
		return false
	}
	return strings.Contains(host, ".")
}

func sendSMTP(addr, host string, helloHost string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsMode := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_TLS_MODE")))
	if tlsMode == "" {
		tlsMode = "starttls"
	}

	var conn net.Conn
	var err error
	if tlsMode == "tls" {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer c.Close()

	if helloHost != "" {
		if err := c.Hello(helloHost); err != nil {
			return fmt.Errorf("hello: %w", err)
		}
	}

	if tlsMode == "starttls" || tlsMode == "auto" {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		} else if tlsMode == "starttls" {
			return errors.New("starttls: server does not advertise STARTTLS")
		}
	}

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("auth: server does not advertise AUTH")
		}
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt %s: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("write data: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	if err := c.Quit(); err != nil {
		return fmt.Errorf("quit: %w", err)
	}
	return nil
}
