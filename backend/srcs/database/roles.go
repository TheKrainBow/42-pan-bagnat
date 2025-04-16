package database

import (
	"backend/utils"
	"fmt"
)

func GetUsersWithRole(roleID string) ([]User, error) {
	rows, err := db.Query(`
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
