package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// UpdateRoleRulesJSON stores (or clears) the JSON rules for a role.
func UpdateRoleRulesJSON(ctx context.Context, roleID string, rulesJSON []byte) error {
	// if roleID <= 0 {
	// 	return fmt.Errorf("invalid roleID")
	// }

	var res sql.Result
	var err error

	// If nil/empty → store NULL to indicate "no rules"
	if len(rulesJSON) == 0 {
		res, err = mainDB.ExecContext(ctx,
			`UPDATE roles
			   SET rules_json = NULL,
			       rules_updated_at = NOW()
			 WHERE id = $1`,
			roleID,
		)
	} else {
		// Cast to jsonb explicitly; driver will pass as text.
		res, err = mainDB.ExecContext(ctx,
			`UPDATE roles
			   SET rules_json = $2::jsonb,
			       rules_updated_at = NOW()
			 WHERE id = $1`,
			roleID, string(rulesJSON),
		)
	}

	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

// ListActiveUsers returns users to evaluate. Tweak the WHERE to your "active" definition.
func ListActiveUsers(ctx context.Context) ([]User, error) {
	// Minimal filter: only users with a 42 login
	rows, err := mainDB.QueryContext(ctx, `
		SELECT id, ft_login
		  FROM users
		 WHERE ft_login IS NOT NULL
		   AND ft_login <> ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.FtLogin); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// EnsureUserRole makes membership match shouldHave. Returns true if a change occurred.
func EnsureUserRole(ctx context.Context, userID, roleID string, shouldHave bool) (bool, error) {
	if shouldHave {
		_, err := mainDB.ExecContext(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			VALUES ($1, $2)
			ON CONFLICT (user_id, role_id) DO NOTHING
		`, userID, roleID)
		if err != nil {
			return false, err
		}
		var exists bool
		err = mainDB.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2
			)`, userID, roleID).Scan(&exists)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	res, err := mainDB.ExecContext(ctx, `
		DELETE FROM user_roles
		 WHERE user_id = $1 AND role_id = $2
	`, userID, roleID)
	if err != nil {
		return false, err
	}
	ra, _ := res.RowsAffected()
	return ra > 0, nil
}

// RemoveRoleFromAllUsers strips a role from everyone, returning how many rows were removed.
func RemoveRoleFromAllUsers(ctx context.Context, roleID string) (int, error) {
	// if roleID <= 0 {
	// 	return 0, fmt.Errorf("invalid roleID")
	// }
	res, err := mainDB.ExecContext(ctx, `
		DELETE FROM user_roles
		 WHERE role_id = $1
	`, roleID)
	if err != nil {
		return 0, err
	}
	ra, _ := res.RowsAffected()
	return int(ra), nil
}

func ListRolesWithRules() ([]Role, error) {
	rows, err := mainDB.Query(`
		SELECT id, is_default, rules_json
		FROM roles
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list roles with rules: %w", err)
	}
	defer rows.Close()

	out := make([]Role, 0, 32)
	for rows.Next() {
		var r Role
		var rulesBytes sql.NullString
		if err := rows.Scan(&r.ID, &r.IsDefault, &rulesBytes); err != nil {
			return nil, fmt.Errorf("scan role with rules: %w", err)
		}
		if rulesBytes.Valid && rulesBytes.String != "" {
			r.Rules = json.RawMessage(rulesBytes.String)
		} else {
			// keep nil to mean "no rules set"
			r.Rules = nil
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles with rules: %w", err)
	}
	return out, nil
}

// GetRoleRulesJSON fetches the stored rules JSON (nil if NULL).
func GetRoleRulesJSON(roleID string) ([]byte, *time.Time, error) {
	const q = `
		SELECT rules_json, rules_updated_at
		FROM roles
		WHERE id = $1
	`
	var rules []byte
	var updatedAt *time.Time
	err := mainDB.QueryRow(q, roleID).Scan(&rules, &updatedAt)
	if err != nil {
		return nil, nil, err // sql.ErrNoRows bubbles up for "role not found"
	}
	// Note: if rules_json is NULL, "rules" will be nil (which we treat as null in the handler)
	return rules, updatedAt, nil
}

func BulkAddUserRoles(userID string, roleIDs []string) error {
	if userID == "" || len(roleIDs) == 0 {
		return nil
	}

	tx, err := mainDB.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const ins = `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	stmt, err := tx.Prepare(ins)
	if err != nil {
		return fmt.Errorf("prepare insert user_roles: %w", err)
	}
	defer stmt.Close()

	for _, rid := range roleIDs {
		if rid == "" {
			continue
		}
		if _, err := stmt.Exec(userID, rid); err != nil {
			return fmt.Errorf("insert user_role (%s,%s): %w", userID, rid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit add user roles: %w", err)
	}
	return nil
}
