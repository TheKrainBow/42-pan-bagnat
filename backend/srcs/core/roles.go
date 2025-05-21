package core

import (
	"backend/database"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type RolePagination struct {
	OrderBy  []database.RoleOrder
	Filter   string
	LastRole *database.Role
	Limit    int
}

func GenerateRoleOrderBy(order string) (dest []database.RoleOrder) {
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

		var field database.RoleOrderField
		switch arg {
		case string(database.RoleID):
			field = database.RoleID
		case string(database.RoleName):
			field = database.RoleName
		case string(database.RoleColor):
			field = database.RoleColor
		default:
			continue
		}

		dest = append(dest, database.RoleOrder{
			Field: field,
			Order: direction,
		})
	}
	return dest
}

func EncodeRolePaginationToken(token RolePagination) (string, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func DecodeRolePaginationToken(encoded string) (RolePagination, error) {
	var token RolePagination
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(data, &token)
	return token, err
}

func GetRoles(pagination RolePagination) ([]Role, string, error) {
	var dest []Role
	realLimit := pagination.Limit + 1

	roles, err := database.GetAllRoles(&pagination.OrderBy, pagination.Filter, pagination.LastRole, realLimit)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get roles in db: %w", err)
	}

	hasMore := len(roles) > pagination.Limit && pagination.Limit > 0
	if hasMore {
		roles = roles[:pagination.Limit]
	}

	for _, role := range roles {
		apiRole := DatabaseRoleToRole(role)
		users, err := database.GetRoleUsers(apiRole.ID)
		if err != nil {
			fmt.Printf("couldn't get roles for user %s: %s\n", apiRole.ID, err.Error())
		} else if len(users) > 0 {
			apiRole.Users = DatabaseUsersToUsers(users)
		}
		modules, err := database.GetRoleModules(apiRole.ID)
		if err != nil {
			fmt.Printf("couldn't get roles for user %s: %s\n", apiRole.ID, err.Error())
		} else {
			apiRole.Modules = DatabaseModulesToModules(modules)
		}
		dest = append(dest, apiRole)
	}

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastRole = &roles[len(roles)-1]

	token, err := EncodeRolePaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}
