package database

import (
	"backend/utils"
	"fmt"
	"strings"
	"time"
)

type UserOrderField string
type OrderDirection string

const (
	Desc OrderDirection = "DESC"
	Asc  OrderDirection = "ASC"
)

const (
	UserID        UserOrderField = "id"
	UserFtLogin   UserOrderField = "ft_login"
	UserLastSeen  UserOrderField = "last_seen"
	UserIsStaff   UserOrderField = "is_staff"
	UserFtIsStaff UserOrderField = "ft_is_staff"
	UserFtID      UserOrderField = "ft_id"
)

type UserOrder struct {
	Field UserOrderField
	Order OrderDirection
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

func GetAllUsers(
	orderBy *[]UserOrder,
	filter string,
	lastUser *User,
	limit int,
) ([]User, error) {
	// 1) Default to ordering by ID ASC if none provided
	if orderBy == nil || len(*orderBy) == 0 {
		tmp := []UserOrder{{Field: UserID, Order: Asc}}
		orderBy = &tmp
	}

	// 2) Build ORDER BY clauses
	hasID := false
	var orderClauses []string
	for _, ord := range *orderBy {
		orderClauses = append(orderClauses,
			fmt.Sprintf("%s %s", ord.Field, ord.Order),
		)
		if ord.Field == UserID {
			hasID = true
		}
	}
	// Always append ID as a tie-breaker for deterministic pagination
	if !hasID {
		orderClauses = append(orderClauses, "id "+string((*orderBy)[0].Order))
	}

	// 3) Build WHERE conditions and collect args
	var whereConds []string
	var args []any
	argPos := 1

	// Cursor pagination: tuple comparison
	if lastUser != nil {
		var cols []string
		var placeholders []string
		// use the same order fields as in ORDER BY
		for _, ord := range *orderBy {
			cols = append(cols, string(ord.Field))
			placeholders = append(placeholders,
				fmt.Sprintf("$%d", argPos),
			)
			switch ord.Field {
			case UserID:
				args = append(args, lastUser.ID)
			case UserFtLogin:
				args = append(args, lastUser.FtLogin)
			case UserFtID:
				args = append(args, lastUser.FtID)
			case UserFtIsStaff:
				args = append(args, lastUser.FtIsStaff)
			case UserIsStaff:
				args = append(args, lastUser.IsStaff)
			case UserLastSeen:
				// pass time.Time directly
				args = append(args, lastUser.LastSeen)
			}
			argPos++
		}
		if !hasID {
			cols = append(cols, "id")
			placeholders = append(placeholders,
				fmt.Sprintf("$%d", argPos),
			)
			args = append(args, lastUser.ID)
			argPos++
		}

		// Asc vs Desc on the first order field
		dir := ">"
		if (*orderBy)[0].Order == Desc {
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

	// Text‐filter on login and 42id
	if filter != "" {
		whereConds = append(whereConds,
			fmt.Sprintf("(ft_login ILIKE '%%' || $%d || '%%' OR ft_id::text ILIKE '%%' || $%d || '%%')", argPos, argPos),
		)
		args = append(args, filter)
		argPos++
	}

	// 4) Assemble SQL
	var sb strings.Builder
	sb.WriteString(
		`SELECT id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff
FROM users`,
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

	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID,
			&u.FtLogin,
			&u.FtID,
			&u.FtIsStaff,
			&u.PhotoURL,
			&u.LastSeen,
			&u.IsStaff,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
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
	_, err := mainDB.Exec(`
		INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.FtLogin, user.FtID, user.FtIsStaff, user.PhotoURL, user.LastSeen, user.IsStaff)
	return err
}

func DeleteUser(id string) error {
	_, err := mainDB.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}
