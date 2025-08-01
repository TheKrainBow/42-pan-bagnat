package database

import (
	"fmt"
	"time"
)

type Session struct {
	SessionID string
	Login     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func AddSession(sess Session) error {
	if sess.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if sess.Login == "" {
		return fmt.Errorf("login cannot be empty")
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now()
	}
	if sess.ExpiresAt.IsZero() {
		sess.ExpiresAt = sess.CreatedAt.Add(24 * time.Hour) // default TTL
	}

	_, err := mainDB.Exec(`
		INSERT INTO sessions (session_id, ft_login, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`, sess.SessionID, sess.Login, sess.CreatedAt, sess.ExpiresAt)

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
