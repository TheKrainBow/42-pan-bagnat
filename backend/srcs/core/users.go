package core

import (
	"backend/database"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type UserPagination struct {
	OrderBy  []database.UserOrder
	Filter   string
	LastUser *database.User
	Limit    int
}

func GenerateOrderBy(order string) (dest []database.UserOrder) {
	if order == "" {
		return nil
	}
	args := strings.Split(order, ",")
	for _, arg := range args {
		var direction database.OrderDirection
		if arg[0] == '-' {
			direction = database.Desc
			arg = arg[1:]
		} else {
			direction = database.Asc
		}

		var field database.UserOrderField
		switch arg {
		case "id":
			field = database.UserID
		case "ft_login":
			field = database.UserFtLogin
		case "last_seen":
			field = database.UserLastSeen
		case "is_staff":
			field = database.UserIsStaff
		case "ft_is_staff":
			field = database.UserFtIsStaff
		case "ft_id":
			field = database.UserFtID
		default:
			continue
		}

		dest = append(dest, database.UserOrder{
			Field: field,
			Order: direction,
		})
	}
	return dest
}

func EncodeUserPaginationToken(token UserPagination) (string, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func DecodeUserPaginationToken(encoded string) (UserPagination, error) {
	var token UserPagination
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(data, &token)
	return token, err
}

func DatabaseRoleToRole(dbRoles database.Role) (dest Role) {
	return Role{
		ID:    dbRoles.ID,
		Name:  dbRoles.Name,
		Color: dbRoles.Color,
	}
}

func DatabaseRolesToRoles(dbRoles []database.Role) (dest []Role) {
	for _, role := range dbRoles {
		dest = append(dest, DatabaseRoleToRole(role))
	}
	return dest
}

func DatabaseUserToUser(dbUser database.User) User {
	return User{
		ID:        dbUser.ID,
		FtID:      dbUser.FtID,
		FtLogin:   dbUser.FtLogin,
		FtIsStaff: dbUser.FtIsStaff,
		IsStaff:   dbUser.IsStaff,
		LastSeen:  dbUser.LastSeen,
		PhotoURL:  dbUser.PhotoURL,
		Roles:     []Role{},
	}
}

func GetUsersFromToken(token string) ([]User, string, error) {
	dest := []User{}

	pagination, err := DecodeUserPaginationToken(token)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't decode token: %w", err)
	}

	realLimit := pagination.Limit + 1
	users, err := database.GetAllUsers(&pagination.OrderBy, pagination.Filter, pagination.LastUser, realLimit)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get users in db: %w", err)
	}

	hasMore := len(users) > pagination.Limit
	if hasMore {
		users = users[:pagination.Limit]
	}

	for _, user := range users {
		apiUser := DatabaseUserToUser(user)
		roles, err := database.GetUserRoles(apiUser.ID)
		if err != nil {
			fmt.Printf("couldn't get roles for user %s: %s\n", apiUser.ID, err.Error())
		} else {
			apiUser.Roles = DatabaseRolesToRoles(roles)
		}
		dest = append(dest, apiUser)
	}

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastUser = &users[len(users)-1]
	token, err = EncodeUserPaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}

func GetUsers(pagination UserPagination) ([]User, string, error) {
	dest := []User{}
	realLimit := pagination.Limit + 1

	users, err := database.GetAllUsers(&pagination.OrderBy, pagination.Filter, pagination.LastUser, realLimit)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get users in db: %w", err)
	}

	hasMore := len(users) > pagination.Limit
	if hasMore {
		users = users[:pagination.Limit]
	}

	for _, user := range users {
		apiUser := DatabaseUserToUser(user)
		roles, err := database.GetUserRoles(apiUser.ID)
		if err != nil {
			fmt.Printf("couldn't get roles for user %s: %s\n", apiUser.ID, err.Error())
		} else {
			apiUser.Roles = DatabaseRolesToRoles(roles)
		}
		dest = append(dest, apiUser)
	}

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastUser = &users[len(users)-1]
	token, err := EncodeUserPaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}
