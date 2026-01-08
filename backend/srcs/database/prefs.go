package database

import (
	"context"
	"encoding/json"
	"time"
)

// user_prefs schema (created on demand):
//   user_id TEXT
//   pref_key TEXT
//   value JSONB
//   updated_at TIMESTAMP
// PK (user_id, pref_key)

func ensureUserPrefsTable(ctx context.Context) error {
	_, err := mainDB.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS user_prefs (
            user_id    TEXT NOT NULL,
            pref_key   TEXT NOT NULL,
            value      JSONB NOT NULL,
            updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
            PRIMARY KEY (user_id, pref_key)
        )
    `)
	return err
}

// GetUserPref returns raw JSON for a user preference key. If not found, returns sql.ErrNoRows.
func GetUserPref(ctx context.Context, userID, key string) (json.RawMessage, error) {
	if err := ensureUserPrefsTable(ctx); err != nil {
		return nil, err
	}
	var data []byte
	err := mainDB.QueryRowContext(ctx, `
        SELECT value
          FROM user_prefs
         WHERE user_id = $1 AND pref_key = $2
    `, userID, key).Scan(&data)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// PutUserPref upserts the raw JSON value for a user preference key.
func PutUserPref(ctx context.Context, userID, key string, value json.RawMessage) error {
	if err := ensureUserPrefsTable(ctx); err != nil {
		return err
	}
	_, err := mainDB.ExecContext(ctx, `
        INSERT INTO user_prefs (user_id, pref_key, value, updated_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id, pref_key)
        DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
    `, userID, key, []byte(value), time.Now())
	return err
}

// DeleteUserPrefs deletes all preferences for a user (used when deleting account)
func DeleteUserPrefs(ctx context.Context, userID string) error {
	if err := ensureUserPrefsTable(ctx); err != nil {
		return err
	}
	_, err := mainDB.ExecContext(ctx, `
        DELETE FROM user_prefs WHERE user_id = $1
    `, userID)
	return err
}
