package database

import (
	"backend/utils"
	"fmt"
	"time"
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

func AddUser(user User) error {
	if user.ID == "" {
		user.ID = utils.GenerateULID(utils.User)
	}
	if user.LastSeen.IsZero() {
		user.LastSeen = time.Now()
	}
	if user.FtLogin == "" && user.FtID == "" {
		return fmt.Errorf("you must provide ftlogin or ftid")
	}
	_, err := db.Exec(`
		INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.FtLogin, user.FtID, user.FtIsStaff, user.PhotoURL, user.LastSeen, user.IsStaff)
	return err
}

func DeleteUser(id string) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}
