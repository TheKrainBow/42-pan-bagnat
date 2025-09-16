package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Session struct {
	SessionID   string
	Login       string // ft_login
	CreatedAt   time.Time
	ExpiresAt   time.Time
	UserAgent   string
	IP          string
	DeviceLabel string
	LastSeen    time.Time
}

// New/updated queries
func ListSessionsByUserID(ctx context.Context, userID string) ([]Session, error) {
    rows, err := mainDB.QueryContext(ctx, `
        SELECT s.session_id, s.ft_login, s.created_at, s.expires_at, s.user_agent, s.ip, s.device_label, s.last_seen
          FROM sessions s
         WHERE s.ft_login = (SELECT ft_login FROM users WHERE id = $1)
         ORDER BY COALESCE(s.last_seen, s.created_at) DESC
    `, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []Session
    for rows.Next() {
        var s Session
        if err := rows.Scan(&s.SessionID, &s.Login, &s.CreatedAt, &s.ExpiresAt, &s.UserAgent, &s.IP, &s.DeviceLabel, &s.LastSeen); err != nil {
            return nil, err
        }
        out = append(out, s)
    }
    return out, nil
}
func FindActiveSessionForDevice(login, ua, ip string, now time.Time) (*Session, error) {
	row := mainDB.QueryRow(`
		SELECT session_id, ft_login, created_at, expires_at, user_agent, ip, device_label, last_seen
		FROM sessions
		WHERE ft_login = $1 AND user_agent = $2 AND ip = $3 AND expires_at > $4
		ORDER BY expires_at DESC
		LIMIT 1
	`, login, ua, ip, now)

	var s Session
	if err := row.Scan(&s.SessionID, &s.Login, &s.CreatedAt, &s.ExpiresAt, &s.UserAgent, &s.IP, &s.DeviceLabel, &s.LastSeen); err != nil {
		return nil, err
	}
	return &s, nil
}

func TouchSessionMaybe(
	ctx context.Context,
	sessionID string,
	now time.Time,
	idleWindow time.Duration,
	absoluteCap time.Duration,
	minWriteInterval time.Duration,
) (bool, bool, time.Time, time.Time, error) {

	var lastSeen time.Time
	var expiresAt time.Time

	// Try to update only if:
	//  - session exists and not expired
	//  - last_seen is older than now - minWriteInterval
	err := mainDB.QueryRowContext(ctx, `
        UPDATE sessions
           SET last_seen  = $2,
               expires_at = LEAST(created_at + $3::interval, $2 + $4::interval)
         WHERE session_id = $1
           AND now() < expires_at
           AND (last_seen IS NULL OR last_seen <= $2 - $5::interval)
     RETURNING last_seen, expires_at
    `,
		sessionID,
		now,
		pgInterval(absoluteCap),
		pgInterval(idleWindow),
		pgInterval(minWriteInterval),
	).Scan(&lastSeen, &expiresAt)

	if err == nil {
		// Updated & still valid
		return true, true, lastSeen, expiresAt, nil
	}
	if err != sql.ErrNoRows {
		return false, false, time.Time{}, time.Time{}, err
	}

	// No update occurred: either too fresh, expired, or missing.
	// Check validity without writing.
	err = mainDB.QueryRowContext(ctx, `
        SELECT last_seen, expires_at
          FROM sessions
         WHERE session_id = $1
    `, sessionID).Scan(&lastSeen, &expiresAt)

	if err == sql.ErrNoRows {
		return false, false, time.Time{}, time.Time{}, nil
	}
	if err != nil {
		return false, false, time.Time{}, time.Time{}, err
	}
	if now.After(expiresAt) {
		return false, false, lastSeen, expiresAt, nil
	}
	return true, false, lastSeen, expiresAt, nil
}

// Helper to pass Go durations as Postgres intervals.
func pgInterval(d time.Duration) string {
	sec := int64(d.Seconds())
	return fmt.Sprintf("%d seconds", sec)
}

func AddSession(s Session) error {
	if s.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if s.Login == "" {
		return fmt.Errorf("login cannot be empty")
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt = s.CreatedAt.Add(24 * time.Hour) // default TTL
	}

	_, err := mainDB.Exec(`
		INSERT INTO sessions (session_id, ft_login, created_at, expires_at, user_agent, ip, device_label, last_seen)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, s.SessionID, s.Login, s.CreatedAt, s.ExpiresAt, s.UserAgent, s.IP, s.DeviceLabel, s.LastSeen)

	return err
}

func GetSession(sessionID string) (*Session, error) {
	var sess Session
	err := mainDB.QueryRow(`
		SELECT session_id, ft_login, created_at, expires_at
		FROM sessions
		WHERE session_id = $1
	`, sessionID).Scan(&sess.SessionID, &sess.Login, &sess.CreatedAt, &sess.ExpiresAt)

	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func DeleteSession(sessionID string) error {
	_, err := mainDB.Exec(`
		DELETE FROM sessions
		WHERE session_id = $1
	`, sessionID)
	return err
}

func PurgeExpiredSessions() (int64, error) {
	res, err := mainDB.Exec(`
		DELETE FROM sessions
		WHERE expires_at < NOW()
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func DeleteUserSessions(ctx context.Context, userID string) (int64, error) {
	res, err := mainDB.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE ft_login = (SELECT ft_login FROM users WHERE id = $1)
	`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
