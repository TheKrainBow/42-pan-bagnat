package api

import (
	"backend/core"
)

func RoleToAPIRole(role core.Role) Role {
	return Role{
		ID:         role.ID,
		Name:       role.Name,
		Color:      role.Color,
		UsersCount: role.UsersCount,
		Users:      UsersToAPIUsers(role.Users),
		Modules:    ModulesToAPIModules(role.Modules),
	}
}

func RolesToAPIRoles(roles []core.Role) (dest []Role) {
	for _, role := range roles {
		dest = append(dest, RoleToAPIRole(role))
	}
	if len(dest) == 0 {
		return (make([]Role, 0))
	}
	return dest
}

func UserToAPIUser(user core.User) User {
	return User{
		ID:        user.ID,
		FtID:      user.FtID,
		FtLogin:   user.FtLogin,
		FtIsStaff: user.FtIsStaff,
		IsStaff:   user.IsStaff,
		PhotoURL:  user.PhotoURL,
		LastSeen:  user.LastSeen,
		Roles:     RolesToAPIRoles(user.Roles),
	}
}

func UsersToAPIUsers(users []core.User) (dest []User) {
	for _, user := range users {
		dest = append(dest, UserToAPIUser(user))
	}
	if len(dest) == 0 {
		return (make([]User, 0))
	}
	return dest
}

func ModuleToAPIModule(module core.Module) Module {
	return Module{
		ID:            module.ID,
		SSHPublicKey:  module.SSHPublicKey,
		Name:          module.Name,
		Version:       module.Version,
		LatestVersion: module.LatestVersion,
		LateCommits:   module.LateCommits,
		LastUpdate:    module.LastUpdate,
		URL:           module.URL,
		IconURL:       module.IconURL,
		Status:        ModuleStatus(module.Status),
		Roles:         RolesToAPIRoles(module.Roles),
	}
}

func ModulesToAPIModules(modules []core.Module) (dest []Module) {
	for _, module := range modules {
		dest = append(dest, ModuleToAPIModule(module))
	}
	if len(dest) == 0 {
		return (make([]Module, 0))
	}
	return dest
}
