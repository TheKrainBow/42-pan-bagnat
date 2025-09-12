package core

import (
	"backend/database"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Role struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color"`
	IsDefault  bool     `json:"is_default"`
	Users      []User   `json:"users"`
	UsersCount int      `json:"usersCount"`
	Modules    []Module `json:"modules"`
}

type RolePagination struct {
	OrderBy  []database.RoleOrder
	Filter   string
	LastRole *database.Role
	Limit    int
}

type RolePatch struct {
	ID        string  `json:"id"`
	Name      *string `json:"name"`
	Color     *string `json:"color"`
	IsDefault *bool   `json:"is_default"`
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

func GetRole(roleID string) (Role, error) {
	dbRole, err := database.GetRole(roleID)
	if err != nil {
		return Role{}, fmt.Errorf("could not find role '%s': %w", roleID, err)
	}

	dto := DatabaseRoleToRole(*dbRole)

	modules, err := database.GetRoleModules(roleID)
	if err == nil {
		dto.Modules = DatabaseModulesToModules(modules)
	}

	users, err := database.GetRoleUsers(roleID)
	if err == nil {
		dto.Users = DatabaseUsersToUsers(users)
		dto.UsersCount = len(dto.Users)
	}

	return dto, nil
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
			apiRole.UsersCount = len(users)
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

func ImportRole(name string, color string, isDefault bool, moduleIDs []string) (Role, error) {
	var dest Role

	// Generate a ULID for the role
	roleID, err := GenerateULID(RoleKind)
	if err != nil {
		return Role{}, fmt.Errorf("failed to generate module ID: %w", err)
	}

	// Prepare module struct
	dest = Role{
		ID:        roleID,
		Name:      name,
		Color:     color,
		IsDefault: isDefault,
	}

	// Insert into DB
	if err := database.AddRole(database.Role{
		ID:        dest.ID,
		Name:      dest.Name,
		Color:     dest.Color,
		IsDefault: dest.IsDefault,
	}); err != nil {
		return Role{}, fmt.Errorf("failed to insert module in DB: %w", err)
	}

	for _, id := range moduleIDs {
		_ = database.AssignRoleToModule(roleID, id)
	}

	modules, err := database.GetRoleModules(roleID)
	if err != nil {
		fmt.Printf("Fail to fetch role's module [%s]\n", roleID)
	} else {
		dest.Modules = DatabaseModulesToModules(modules)
	}

	return dest, nil
}

const (
	RoleIDBlacklist = "roles_blacklist"
	RoleIDAdmin     = "roles_admin"
)

// Returned when an action would leave the system with zero admins.
var ErrWouldBlacklistLastAdmin = errors.New("cannot blacklist the last admin user")
var ErrWouldRemoveLastAdmin = errors.New("cannot remove the last admin user")

func AddRoleToUser(roleID, userIdentifier string) error {
	if roleID == RoleIDBlacklist {
		// Resolve to user ID
		userID := userIdentifier
		if !strings.HasPrefix(userIdentifier, "users_") {
			u, err := database.GetUserByLogin(userIdentifier)
			if err != nil {
				return fmt.Errorf("resolve user by login: %w", err)
			}
			userID = u.ID
		}

		ctx := context.Background()

		isAdmin, err := database.UserHasRoleByID(ctx, userID, RoleIDAdmin)
		if err != nil {
			return fmt.Errorf("check admin role: %w", err)
		}

		if isAdmin {
			adminCount, err := database.CountActiveUsersWithRole(ctx, RoleIDAdmin, RoleIDBlacklist)
			if err != nil {
				return fmt.Errorf("count admins: %w", err)
			}
			if adminCount <= 1 {
				return ErrWouldBlacklistLastAdmin
			}
		}
	}
	return database.AssignRoleToUser(roleID, userIdentifier)
}

func DeleteRoleFromUser(roleID, userIdentifier string) error {
	if roleID == RoleIDAdmin {
		userID := userIdentifier
		if !strings.HasPrefix(userIdentifier, "users_") {
			u, err := database.GetUserByLogin(userIdentifier)
			if err != nil {
				return fmt.Errorf("resolve user by login: %w", err)
			}
			userID = u.ID
		}
		ctx := context.Background()
		isAdmin, err := database.UserHasRoleByID(ctx, userID, RoleIDAdmin)
		if err != nil {
			return fmt.Errorf("check admin role: %w", err)
		}

		isBlacklisted, err := database.UserHasRoleByID(ctx, userID, RoleIDBlacklist)
		if err != nil {
			return fmt.Errorf("check blacklist role: %w", err)
		}

		if isAdmin && !isBlacklisted {
			adminCount, err := database.CountActiveUsersWithRole(ctx, RoleIDAdmin, RoleIDBlacklist)
			if err != nil {
				return fmt.Errorf("count admins: %w", err)
			}
			if adminCount <= 1 {
				return ErrWouldRemoveLastAdmin
			}
		}
	}
	return database.RemoveRoleFromUser(roleID, userIdentifier)
}

func AddRoleToModule(roleID, moduleID string) error {
	return database.AssignRoleToModule(roleID, moduleID)
}

func DeleteRoleFromModule(roleID, moduleID string) error {
	return database.RemoveRoleFromModule(roleID, moduleID)
}

func DeleteRole(roleID string) error {
	err := database.DeleteRole(roleID)
	if err != nil {
		fmt.Printf("couldn't delete module: %s\n", err.Error())
		return err
	}
	return nil
}

func PatchRole(patch RolePatch) (*Role, error) {
	if patch.ID == "" {
		return nil, fmt.Errorf("missing role id")
	}

	dbPatch := database.RolePatch{
		ID:        patch.ID,
		Name:      patch.Name,
		Color:     patch.Color,
		IsDefault: patch.IsDefault,
	}

	err := database.PatchRole(dbPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch role: %w", err)
	}

	dbRole, err := database.GetRole(patch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to patch role: %w", err)
	}

	role := DatabaseRoleToRole(*dbRole)
	return &role, nil
}
