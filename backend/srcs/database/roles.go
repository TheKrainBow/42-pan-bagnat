package database

import (
	"backend/utils"
	"fmt"
)

func GetUserRoles(userID string) ([]Role, error) {
	rows, err := db.Query(`
		SELECT r.id, r.name, r.color
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`, userID)
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

func AddRole(role Role) error {
	if role.ID == "" {
		role.ID = utils.GenerateULID(utils.Role)
	}
	if role.Color == "" || role.Name == "" {
		return fmt.Errorf("some fields are missing")
	}
	_, err := db.Exec(`
		INSERT INTO roles (id, name, color)
		VALUES ($1, $2, $3)
	`, role.ID, role.Name, role.Color)
	return err
}

func DeleteRole(id string) error {
	_, err := db.Exec(`DELETE FROM roles WHERE id = $1`, id)
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

	_, err := db.Exec(query, params...)
	return err
}

func PutRole(role Role) error {
	_, err := db.Exec(`
		UPDATE users
		SET name = $1, color = $2
		WHERE id = $3
	`, role.Name, role.Color, role.ID)
	return err
}
