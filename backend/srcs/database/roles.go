package database

import (
	"backend/utils"
	"fmt"
	"strings"
)

type Role struct {
	ID                string `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X" db:"id"`
	Name              string `json:"name" example:"captain-hook" db:"name"`
	Color             string `json:"color" example:"0xFF00FF" db:"color"`
	AssignedByDefault bool   `json:"assigned_by_default" example:"true" db:"assigned_by_default"`
}

type RolePatch struct {
	ID    string  `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name  *string `json:"name" example:"captain-hook"`
	Color *string `json:"color" example:"0xFF00FF"`
}

type RoleOrderField string

const (
	RoleID    RoleOrderField = "id"
	RoleName  RoleOrderField = "name"
	RoleColor RoleOrderField = "color"
)

type RoleOrder struct {
	Field RoleOrderField
	Order OrderDirection
}

func AddRole(role Role) error {
	if role.ID == "" {
		role.ID = utils.GenerateULID(utils.Role)
	}
	if role.Color == "" || role.Name == "" {
		return fmt.Errorf("some fields are missing")
	}
	_, err := mainDB.Exec(`
		INSERT INTO roles (id, name, color)
		VALUES ($1, $2, $3)
	`, role.ID, role.Name, role.Color)
	return err
}

func DeleteRole(id string) error {
	_, err := mainDB.Exec(`DELETE FROM roles WHERE id = $1`, id)
	return err
}

func PatchRole(toPatch RolePatch) error {
	query := "UPDATE roles SET "
	params := []any{}
	paramCount := 1

	if toPatch.Name != nil {
		query += fmt.Sprintf("name = $%d, ", paramCount)
		params = append(params, *toPatch.Name)
		paramCount++
	}
	if toPatch.Color != nil {
		query += fmt.Sprintf("color = $%d, ", paramCount)
		params = append(params, *toPatch.Color)
		paramCount++
	}

	if len(params) == 0 {
		return nil
	}

	query = query[:len(query)-2]

	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	params = append(params, toPatch.ID)

	_, err := mainDB.Exec(query, params...)
	return err
}

func PutRole(role Role) error {
	_, err := mainDB.Exec(`
		UPDATE users
		SET name = $1, color = $2
		WHERE id = $3
	`, role.Name, role.Color, role.ID)
	return err
}

func GetAllRoles(
	orderBy *[]RoleOrder,
	filter string,
	lastRole *Role,
	limit int,
) ([]Role, error) {
	// 1) Default to ordering by ID ASC if none provided
	if orderBy == nil || len(*orderBy) == 0 {
		tmp := []RoleOrder{{Field: RoleID, Order: Asc}}
		orderBy = &tmp
	}

	// 2) Build ORDER BY clauses
	hasID := false
	var orderClauses []string
	for _, ord := range *orderBy {
		orderClauses = append(orderClauses,
			fmt.Sprintf("%s %s", ord.Field, ord.Order),
		)
		if ord.Field == RoleID {
			hasID = true
		}
	}
	// Always append ID as tie-breaker in same direction as first field
	firstOrder := (*orderBy)[0].Order
	if !hasID {
		orderClauses = append(orderClauses,
			fmt.Sprintf("id %s", firstOrder),
		)
	}

	// 3) Build WHERE conditions and collect args
	var whereConds []string
	var args []any
	argPos := 1

	// Cursor pagination: tuple comparison
	if lastRole != nil {
		var cols, placeholders []string

		// use the same order fields as in ORDER BY
		for _, ord := range *orderBy {
			cols = append(cols, string(ord.Field))
			placeholders = append(placeholders,
				fmt.Sprintf("$%d", argPos),
			)
			switch ord.Field {
			case RoleID:
				args = append(args, lastRole.ID)
			case RoleName:
				args = append(args, lastRole.Name)
			case RoleColor:
				args = append(args, lastRole.Color)
			}
			argPos++
		}

		if !hasID {
			cols = append(cols, "id")
			placeholders = append(placeholders,
				fmt.Sprintf("$%d", argPos),
			)
			args = append(args, lastRole.ID)
			argPos++
		}

		// Asc vs Desc on the first order field
		dir := ">"
		if firstOrder == Desc {
			dir = "<"
		}

		whereConds = append(whereConds,
			fmt.Sprintf("(%s) %s (%s)",
				strings.Join(cols, ", "),
				dir,
				strings.Join(placeholders, ", "),
			),
		)
	}

	// Text‐filter only on Name
	if filter != "" {
		whereConds = append(whereConds,
			fmt.Sprintf("name ILIKE '%%' || $%d || '%%'", argPos),
		)
		args = append(args, filter)
		argPos++
	}

	// 4) Assemble SQL
	var sb strings.Builder
	sb.WriteString(
		`SELECT id, name, color
FROM roles`,
	)
	if len(whereConds) > 0 {
		sb.WriteString("\nWHERE ")
		sb.WriteString(strings.Join(whereConds, " AND "))
	}
	sb.WriteString("\nORDER BY ")
	sb.WriteString(strings.Join(orderClauses, ", "))
	if limit > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d", limit))
	}
	sb.WriteString(";")

	query := sb.String()

	// 5) Execute and scan
	rows, err := mainDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Role{}
	for rows.Next() {
		var r Role
		if err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.Color,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func GetRoleUsers(roleID string) ([]User, error) {
	rows, err := mainDB.Query(`
		SELECT u.id, u.ft_login, u.ft_id, u.ft_is_staff, u.photo_url, u.last_seen, u.is_staff
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.FtLogin,
			&user.FtID,
			&user.FtIsStaff,
			&user.PhotoURL,
			&user.LastSeen,
			&user.IsStaff,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func GetRoleUserCount(roleID string) (int, error) {
	rows, err := mainDB.Query(`
		SELECT COUNT(*) 
		FROM user_roles 
		WHERE role_id = $1;
	`, roleID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var usersCount int
	if rows.Next() {
		if err := rows.Scan(&usersCount); err != nil {
			return 0, err
		}
	}
	return usersCount, nil
}

func GetRoleModules(roleID string) ([]Module, error) {
	rows, err := mainDB.Query(`
		SELECT mod.id, mod.name, mod.version, mod.status, mod.git_url, mod.icon_url, mod.latest_version, mod.late_commits, mod.last_update
		FROM modules mod
		JOIN module_roles ur ON ur.module_id = mod.id
		WHERE ur.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []Module
	for rows.Next() {
		var module Module
		if err := rows.Scan(
			&module.ID,
			&module.Name,
			&module.Version,
			&module.Status,
			&module.GitURL,
			&module.IconURL,
			&module.LatestVersion,
			&module.LateCommits,
			&module.LastUpdate,
		); err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func AssignRoleToUser(roleID, userIdentifier string) error {
	var userID string

	err := mainDB.QueryRow(`
		SELECT id FROM users
		WHERE id = $1 OR ft_login = $1
	`, userIdentifier).Scan(&userID)

	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	_, err = mainDB.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, roleID)

	return err
}

func RemoveRoleFromUser(roleID, userIdentifier string) error {
	var userID string

	err := mainDB.QueryRow(`
		SELECT id FROM users
		WHERE id = $1 OR ft_login = $1
	`, userIdentifier).Scan(&userID)

	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	_, err = mainDB.Exec(`
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = $2
	`, userID, roleID)

	return err
}

func AssignRoleToModule(roleID, moduleID string) error {
	fmt.Printf("Adding module_roles for module (%s) and role (%s)\n", moduleID, roleID)
	_, err := mainDB.Exec(`
		INSERT INTO module_roles (module_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, moduleID, roleID)

	return err
}

func RemoveRoleFromModule(roleID, moduleID string) error {
	_, err := mainDB.Exec(`
		DELETE FROM module_roles
		WHERE module_id = $1 AND role_id = $2
	`, moduleID, roleID)

	return err
}
