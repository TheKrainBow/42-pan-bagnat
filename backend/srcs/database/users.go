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

func GetAllUsers(orderBy *[]UserOrder, lastUser *User, limit int) ([]User, error) {
	orderSTR := ""
	pagination := ""
	orderList := []string{}
	paginationList := []string{}
	paginationArgsList := []string{}
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
				paginationArgsList = append(paginationArgsList, fmt.Sprintf("'%s'", (*lastUser).ID))
			case FtID:
				paginationArgsList = append(paginationArgsList, (*lastUser).FtID)
			case FtIsStaff:
				paginationArgsList = append(paginationArgsList, fmt.Sprintf("%t", (*lastUser).FtIsStaff))
			case IsStaff:
				paginationArgsList = append(paginationArgsList, fmt.Sprintf("%t", (*lastUser).IsStaff))
			case LastSeen:
				formattedTimestamp := (*lastUser).LastSeen.Format("2006-01-02 15:04:05-07:00")
				paginationArgsList = append(paginationArgsList, fmt.Sprintf("'%s'", formattedTimestamp))
			case FtLogin:
				paginationArgsList = append(paginationArgsList, fmt.Sprintf("'%s'", (*lastUser).FtLogin))
			}
		}
	}

	if lastUser != nil {
		if !hasID {
			paginationList = append(paginationList, "id")
			paginationArgsList = append(paginationArgsList, fmt.Sprintf("'%s'", (*lastUser).ID))
		}
		pagination = "WHERE (" + strings.Join(paginationList, ", ") + ") " + comparaisonDirection + " (" + strings.Join(paginationArgsList, ", ") + ")"
	}

	if !hasID {
		orderList = append(orderList, "id ASC")
	}
	orderSTR = "ORDER BY " + strings.Join(orderList, ", ")

	query := "SELECT id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff\n" +
		"FROM users"
	if lastUser != nil {
		query = fmt.Sprintf("%s\n%s", query, pagination)
	}
	query = fmt.Sprintf("%s\n%s", query, orderSTR)
	if limit > 0 {
		query = fmt.Sprintf("%s\n%s", query, fmt.Sprintf("LIMIT %d;", limit))
	}
	query = query + "\n"
	rows, err := db.Query(query)
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
