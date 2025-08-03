package database

import (
	"backend/utils"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID        string    `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X" db:"id"`
	FtLogin   string    `json:"login" example:"heinz" db:"login"`
	FtID      int       `json:"ft_id" example:"1492" db:"ft_id"`
	FtIsStaff bool      `json:"ft_is_staff" example:"true" db:"ft_is_staff"`
	PhotoURL  string    `json:"photo_url" example:"https://intra.42.fr/some-login/some-id" db:"photo_url"`
	LastSeen  time.Time `json:"last_update" example:"2025-02-18T15:00:00Z" db:"last_update"`
	IsStaff   bool      `json:"is_staff" example:"true" db:"is_staff"`
}

type UserPatch struct {
	ID        string     `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtLogin   *string    `json:"login" example:"heinz"`
	FtID      *int       `json:"ft_id" example:"1492"`
	FtIsStaff *bool      `json:"ft_is_staff" example:"true"`
	PhotoURL  *string    `json:"photo_url" example:"https://intra.42.fr/some-login/some-id"`
	LastSeen  *time.Time `json:"last_update" example:"2025-02-18T15:00:00Z"`
	IsStaff   *bool      `json:"is_staff" example:"true"`
}

type UserOrderField string

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

func GetUser(identifier string) (*User, error) {
	if strings.HasPrefix(identifier, "user_") {
		return GetUserByID(identifier)
	}
	return GetUserByLogin(identifier)
}

func GetUserByID(id string) (*User, error) {
	var user User
	err := mainDB.QueryRow(`
		SELECT id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.FtLogin, &user.FtID, &user.FtIsStaff, &user.PhotoURL, &user.LastSeen, &user.IsStaff)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByLogin(login string) (*User, error) {
	var user User
	err := mainDB.QueryRow(`
		SELECT id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff
		FROM users
		WHERE ft_login = $1
	`, login).Scan(&user.ID, &user.FtLogin, &user.FtID, &user.FtIsStaff, &user.PhotoURL, &user.LastSeen, &user.IsStaff)

	if err != nil {
		return nil, err
	}
	return &user, nil
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
	if user.FtLogin == "" || user.FtID == 0 {
		return fmt.Errorf("you must provide ftlogin and ftid")
	}
	_, err := mainDB.Exec(`
		INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.FtLogin, user.FtID, user.FtIsStaff, user.PhotoURL, user.LastSeen, user.IsStaff)
	return err
}

func UpdateUserLastSeen(login string, ts time.Time) error {
	_, err := mainDB.Exec(`
        UPDATE users SET last_seen = $1 WHERE ft_login = $2
    `, ts, login)
	return err
}

func DeleteUser(id string) error {
	_, err := mainDB.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}

func GetUserRoles(userID string) ([]Role, error) {
	rows, err := mainDB.Query(`
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

func PatchUser(patch UserPatch) error {
	if patch.ID == "" {
		return fmt.Errorf("user ID is required")
	}

	var sets []string
	var args []any
	argPos := 1

	if patch.FtLogin != nil {
		sets = append(sets, fmt.Sprintf("ft_login = $%d", argPos))
		args = append(args, *patch.FtLogin)
		argPos++
	}
	if patch.FtID != nil {
		sets = append(sets, fmt.Sprintf("ft_id = $%d", argPos))
		args = append(args, *patch.FtID)
		argPos++
	}
	if patch.FtIsStaff != nil {
		sets = append(sets, fmt.Sprintf("ft_is_staff = $%d", argPos))
		args = append(args, *patch.FtIsStaff)
		argPos++
	}
	if patch.PhotoURL != nil {
		sets = append(sets, fmt.Sprintf("photo_url = $%d", argPos))
		args = append(args, *patch.PhotoURL)
		argPos++
	}
	if patch.LastSeen != nil {
		sets = append(sets, fmt.Sprintf("last_seen = $%d", argPos))
		args = append(args, *patch.LastSeen)
		argPos++
	}
	if patch.IsStaff != nil {
		sets = append(sets, fmt.Sprintf("is_staff = $%d", argPos))
		args = append(args, *patch.IsStaff)
		argPos++
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d`,
		strings.Join(sets, ", "),
		argPos,
	)
	args = append(args, patch.ID)

	_, err := mainDB.Exec(query, args...)
	return err
}

func GetRole(id string) (*Role, error) {
	var role Role
	err := mainDB.QueryRow(`
		SELECT id, name, color, assigned_by_default
		FROM roles
		WHERE id = $1
	`, id).Scan(&role.ID, &role.Name, &role.Color, &role.AssignedByDefault)

	if err != nil {
		return nil, err
	}
	return &role, nil
}
