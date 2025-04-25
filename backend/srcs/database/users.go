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
	ID        UserOrderField = "id"
	FtLogin   UserOrderField = "ft_login"
	LastSeen  UserOrderField = "last_seen"
	IsStaff   UserOrderField = "is_staff"
	FtIsStaff UserOrderField = "ft_is_staff"
	FtID      UserOrderField = "ft_id"
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

func GetAllUsers(orderBy *[]UserOrder, filter string, lastUser *User, limit int) ([]User, error) {
	orderSTR := ""
	where := ""
	orderList := []string{}
	paginationList := []string{}
	paginationArgsList := []string{}
	args := []any{}
	argsCount := 1
	hasID := orderBy == nil
	comparaisonDirection := ""

	if orderBy == nil {
		tmp := []UserOrder{{Field: ID, Order: Asc}}
		orderBy = &tmp
	}

	for _, order := range *orderBy {
		if comparaisonDirection == "" {
			if order.Order == Asc {
				comparaisonDirection = ">"
			} else {
				comparaisonDirection = "<"
			}
		}
		if order.Field == ID {
			hasID = true
		}
		orderList = append(orderList, string(order.Field)+" "+string(order.Order))
		if lastUser != nil {
			paginationList = append(paginationList, string(order.Field))
			switch order.Field {
			case ID:
				args = append(args, (*lastUser).ID)
			case FtID:
				args = append(args, (*lastUser).FtID)
			case FtIsStaff:
				args = append(args, (*lastUser).FtIsStaff)
			case IsStaff:
				args = append(args, (*lastUser).IsStaff)
			case LastSeen:
				formattedTimestamp := (*lastUser).LastSeen.Format("2006-01-02 15:04:05-07:00")
				args = append(args, formattedTimestamp)
			case FtLogin:
				args = append(args, (*lastUser).FtLogin)
			}
			argsCount++
		}
	}

	whereCondition := []string{}

	if lastUser != nil {
		if !hasID {
			paginationList = append(paginationList, "id")
			args = append(args, (*lastUser).ID)
			argsCount++
		}
		for i := 1; i < argsCount; i++ {
			paginationArgsList = append(paginationArgsList, fmt.Sprintf("$%d", i))
		}
		whereCondition = append(whereCondition, "("+strings.Join(paginationList, ", ")+") "+comparaisonDirection+" ("+strings.Join(paginationArgsList, ", ")+")")
	}

	if filter != "" {
		whereCondition = append(whereCondition, fmt.Sprintf("(ft_login ILIKE '%%' || $%d || '%%' OR ft_id::text ILIKE '%%' || $%d || '%%')", argsCount, argsCount))
		args = append(args, filter)
		argsCount++
	}
	if len(whereCondition) != 0 {
		where = "WHERE " + strings.Join(whereCondition, " AND ")
	}
	if !hasID {
		if comparaisonDirection == ">" {
			orderList = append(orderList, "id ASC")
		} else {
			orderList = append(orderList, "id DESC")
		}
	}
	orderSTR = "ORDER BY " + strings.Join(orderList, ", ")

	query := "SELECT id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff\n" +
		"FROM users"
	if where != "" {
		query = fmt.Sprintf("%s\n%s", query, where)
	}
	query = fmt.Sprintf("%s\n%s", query, orderSTR)
	if limit > 0 {
		query = fmt.Sprintf("%s\n%s", query, fmt.Sprintf("LIMIT %d;", limit))
	}
	query = query + "\n"
	rows, err := mainDB.Query(query, args...)
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
