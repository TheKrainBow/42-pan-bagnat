package database

import (
	"context"
	"time"
)

type MagicLinkToken struct {
	TokenHash  string
	TokenValue string
	UserID     string
	NextURL    string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSentAt time.Time
	ConsumedAt *time.Time
}

func CreateMagicLinkToken(ctx context.Context, token MagicLinkToken) error {
	var lastSentAt any
	if !token.LastSentAt.IsZero() {
		lastSentAt = token.LastSentAt
	}
	_, err := mainDB.ExecContext(ctx, `
		INSERT INTO magic_link_tokens (token_hash, token_value, user_id, next_url, expires_at, last_sent_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, token.TokenHash, token.TokenValue, token.UserID, token.NextURL, token.ExpiresAt, lastSentAt)
	return err
}

func GetLastMagicLinkSentAtForUser(ctx context.Context, userID string) (time.Time, error) {
	var sentAt time.Time
	err := mainDB.QueryRowContext(ctx, `
		SELECT last_sent_at
		FROM magic_link_tokens
		WHERE user_id = $1
		  AND last_sent_at IS NOT NULL
		ORDER BY last_sent_at DESC
		LIMIT 1
	`, userID).Scan(&sentAt)
	return sentAt, err
}

func GetActiveMagicLinkTokenForUser(ctx context.Context, userID string, now time.Time) (*MagicLinkToken, error) {
	var token MagicLinkToken
	err := mainDB.QueryRowContext(ctx, `
		SELECT token_hash, COALESCE(token_value, ''), user_id, next_url, created_at, expires_at, COALESCE(last_sent_at, created_at)
		FROM magic_link_tokens
		WHERE user_id = $1
		  AND consumed_at IS NULL
		  AND expires_at > $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, now).Scan(
		&token.TokenHash,
		&token.TokenValue,
		&token.UserID,
		&token.NextURL,
		&token.CreatedAt,
		&token.ExpiresAt,
		&token.LastSentAt,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func TouchMagicLinkTokenSentAt(ctx context.Context, tokenHash string, sentAt time.Time) error {
	_, err := mainDB.ExecContext(ctx, `
		UPDATE magic_link_tokens
		   SET last_sent_at = $2
		 WHERE token_hash = $1
	`, tokenHash, sentAt)
	return err
}

func ConsumeMagicLinkToken(ctx context.Context, tokenHash string, now time.Time) (*User, string, error) {
	var user User
	var nextURL string
	err := mainDB.QueryRowContext(ctx, `
		UPDATE magic_link_tokens m
		   SET consumed_at = $2
		  FROM users u
		 WHERE m.token_hash = $1
		   AND m.user_id = u.id
		   AND m.consumed_at IS NULL
		   AND m.expires_at > $2
		RETURNING u.id, u.ft_login, u.ft_id, u.ft_is_staff, u.photo_url, COALESCE(u.email, ''), u.last_seen, m.next_url
	`, tokenHash, now).Scan(
		&user.ID,
		&user.FtLogin,
		&user.FtID,
		&user.FtIsStaff,
		&user.PhotoURL,
		&user.Email,
		&user.LastSeen,
		&nextURL,
	)
	if err != nil {
		return nil, "", err
	}
	return &user, nextURL, nil
}
