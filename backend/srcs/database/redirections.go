package database

import (
	"database/sql"
	"errors"
	"fmt"
)

type Redirection struct {
	ID        string `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	Slug      string `json:"slug" db:"slug"`
	TargetURL string `json:"target_url" db:"target_url"`
	IconURL   string `json:"icon_url" db:"icon_url"`
	NeedAuth  bool   `json:"need_auth" db:"need_auth"`
	IsVisible bool   `json:"is_visible" db:"is_visible"`
}

type RedirectionPatch struct {
	ID        string
	Name      *string
	Slug      *string
	TargetURL *string
	IconURL   *string
	NeedAuth  *bool
	IsVisible *bool
}

func GetRedirection(slug string) (*Redirection, error) {
	row := mainDB.QueryRow(`
		SELECT id, name, slug, target_url, icon_url, need_auth, is_visible
		FROM redirections
		WHERE slug = $1
	`, slug)

	var redirection Redirection
	if err := row.Scan(
		&redirection.ID,
		&redirection.Name,
		&redirection.Slug,
		&redirection.TargetURL,
		&redirection.IconURL,
		&redirection.NeedAuth,
		&redirection.IsVisible,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &redirection, nil
}

func GetRedirectionByID(redirectionID string) (*Redirection, error) {
	row := mainDB.QueryRow(`
		SELECT id, name, slug, target_url, icon_url, need_auth, is_visible
		FROM redirections
		WHERE id = $1
	`, redirectionID)

	var redirection Redirection
	if err := row.Scan(
		&redirection.ID,
		&redirection.Name,
		&redirection.Slug,
		&redirection.TargetURL,
		&redirection.IconURL,
		&redirection.NeedAuth,
		&redirection.IsVisible,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &redirection, nil
}

func GetAllRedirections() ([]Redirection, error) {
	rows, err := mainDB.Query(`
		SELECT id, name, slug, target_url, icon_url, need_auth, is_visible
		FROM redirections
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirections []Redirection
	for rows.Next() {
		var redirection Redirection
		if err := rows.Scan(
			&redirection.ID,
			&redirection.Name,
			&redirection.Slug,
			&redirection.TargetURL,
			&redirection.IconURL,
			&redirection.NeedAuth,
			&redirection.IsVisible,
		); err != nil {
			return nil, err
		}
		redirections = append(redirections, redirection)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return redirections, nil
}

func IsRedirectionSlugTaken(slug string) (bool, error) {
	var exists bool
	err := mainDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM redirections WHERE slug = $1
		)
	`, slug).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func InsertRedirection(r Redirection) error {
	_, err := mainDB.Exec(`
		INSERT INTO redirections (id, name, slug, target_url, icon_url, need_auth, is_visible)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, r.ID, r.Name, r.Slug, r.TargetURL, r.IconURL, r.NeedAuth, r.IsVisible)
	return err
}

func PatchRedirection(p RedirectionPatch) (Redirection, error) {
	row := mainDB.QueryRow(`
		UPDATE redirections
		   SET name       = COALESCE($1, name),
		       slug       = COALESCE($2, slug),
		       target_url = COALESCE($3, target_url),
		       icon_url   = COALESCE($4, icon_url),
		       need_auth  = COALESCE($5, need_auth),
		       is_visible = COALESCE($6, is_visible),
		       updated_at = NOW()
		 WHERE id = $7
	 RETURNING id, name, slug, target_url, icon_url, need_auth, is_visible
	`, p.Name, p.Slug, p.TargetURL, p.IconURL, p.NeedAuth, p.IsVisible, p.ID)

	var out Redirection
	if err := row.Scan(
		&out.ID,
		&out.Name,
		&out.Slug,
		&out.TargetURL,
		&out.IconURL,
		&out.NeedAuth,
		&out.IsVisible,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Redirection{}, fmt.Errorf("redirection %s doesn't exist", p.ID)
		}
		return Redirection{}, fmt.Errorf("PatchRedirection scan: %w", err)
	}
	return out, nil
}

func DeleteRedirection(redirectionID string) error {
	_, err := mainDB.Exec(`
		DELETE FROM redirections
		 WHERE id = $1
	`, redirectionID)
	return err
}

func GetRedirectionRoles(redirectionID string) ([]Role, error) {
	rows, err := mainDB.Query(`
		SELECT r.id, r.name, r.color
		FROM roles r
		JOIN redirection_roles rr ON rr.role_id = r.id
		WHERE rr.redirection_id = $1
		ORDER BY r.name ASC
	`, redirectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Color); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func AssignRoleToRedirection(roleID, redirectionID string) error {
	_, err := mainDB.Exec(`
		INSERT INTO redirection_roles (redirection_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, redirectionID, roleID)
	return err
}

func RemoveRoleFromRedirection(roleID, redirectionID string) error {
	_, err := mainDB.Exec(`
		DELETE FROM redirection_roles
		WHERE redirection_id = $1 AND role_id = $2
	`, redirectionID, roleID)
	return err
}

func GetRedirectionForbiddenRoles(redirectionID string) ([]Role, error) {
	rows, err := mainDB.Query(`
		SELECT r.id, r.name, r.color
		FROM roles r
		JOIN redirection_forbidden_roles rr ON rr.role_id = r.id
		WHERE rr.redirection_id = $1
		ORDER BY r.name ASC
	`, redirectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Color); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func AssignForbiddenRoleToRedirection(roleID, redirectionID string) error {
	_, err := mainDB.Exec(`
		INSERT INTO redirection_forbidden_roles (redirection_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, redirectionID, roleID)
	return err
}

func RemoveForbiddenRoleFromRedirection(roleID, redirectionID string) error {
	_, err := mainDB.Exec(`
		DELETE FROM redirection_forbidden_roles
		WHERE redirection_id = $1 AND role_id = $2
	`, redirectionID, roleID)
	return err
}

// GetUserRedirections returns the redirections a user (by id or ft_login) can access,
// mirroring GetUserPages' access rules for module pages.
func GetUserRedirections(identifier string) ([]Redirection, error) {
	rows, err := mainDB.Query(`
        SELECT DISTINCT re.id,
                        re.name,
                        re.slug,
                        re.target_url,
                        COALESCE(re.icon_url, '') AS icon_url,
                        re.need_auth,
                        re.is_visible
        FROM users u
        CROSS JOIN redirections re
        WHERE (u.id = $1 OR u.ft_login = $1)
          AND (
                EXISTS (
                    SELECT 1
                      FROM user_roles ur_admin
                     WHERE ur_admin.user_id = u.id
                       AND ur_admin.role_id = 'roles_admin'
                )
                OR
                re.need_auth = FALSE
                OR EXISTS (
                    SELECT 1
                      FROM user_roles ur
                      JOIN redirection_roles rr ON rr.role_id = ur.role_id
                     WHERE ur.user_id = u.id
                       AND rr.redirection_id = re.id
                )
              )
          AND NOT EXISTS (
                SELECT 1
                  FROM user_roles ur_forbidden
                  JOIN redirection_forbidden_roles rfr ON rfr.role_id = ur_forbidden.role_id
                 WHERE ur_forbidden.user_id = u.id
                   AND rfr.redirection_id = re.id
              )
    `, identifier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirections []Redirection
	for rows.Next() {
		var redirection Redirection
		if err := rows.Scan(
			&redirection.ID,
			&redirection.Name,
			&redirection.Slug,
			&redirection.TargetURL,
			&redirection.IconURL,
			&redirection.NeedAuth,
			&redirection.IsVisible,
		); err != nil {
			return nil, err
		}
		redirections = append(redirections, redirection)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return redirections, nil
}

func UserCanAccessRedirection(identifier, slug string) (bool, error) {
	var exists bool
	err := mainDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			  FROM redirections re
			  JOIN users u ON (u.id = $1 OR u.ft_login = $1)
			 WHERE re.slug = $2
			   AND (
			        EXISTS (
			            SELECT 1
			              FROM user_roles ur_admin
			             WHERE ur_admin.user_id = u.id
			               AND ur_admin.role_id = 'roles_admin'
			        )
			        OR
			        re.need_auth = FALSE
			        OR EXISTS (
			            SELECT 1
			              FROM user_roles ur
			              JOIN redirection_roles rr ON rr.role_id = ur.role_id
			             WHERE ur.user_id = u.id
			               AND rr.redirection_id = re.id
			        )
			   )
			   AND NOT EXISTS (
			        SELECT 1
			          FROM user_roles ur_forbidden
			          JOIN redirection_forbidden_roles rfr ON rfr.role_id = ur_forbidden.role_id
			         WHERE ur_forbidden.user_id = u.id
			           AND rfr.redirection_id = re.id
			   )
		)
	`, identifier, slug).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
