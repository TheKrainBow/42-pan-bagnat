package api

import (
	"backend/core"
)

func RoleToAPIRole(role core.Role) Role {
	return Role{
		ID:         role.ID,
		Name:       role.Name,
		Color:      role.Color,
		IsDefault:  role.IsDefault,
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
		ID:               module.ID,
		SSHPublicKey:     module.SSHPublicKey,
		SSHKeyID:         module.SSHKeyID,
		Name:             module.Name,
		Slug:             module.Slug,
		Version:          module.Version,
		LatestVersion:    module.LatestVersion,
		LateCommits:      module.LateCommits,
		LastUpdate:       module.LastUpdate,
		GitURL:           module.GitURL,
		GitBranch:        module.GitBranch,
		IconURL:          module.IconURL,
		Status:           ModuleStatus(module.Status),
		Roles:            RolesToAPIRoles(module.Roles),
		IsDeploying:      module.IsDeploying,
		LastDeploy:       module.LastDeploy,
		LastDeployStatus: module.LastDeployStatus,
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

func ModuleLogToAPIModuleLog(moduleLog core.ModuleLog) ModuleLog {
	return ModuleLog{
		ID:        moduleLog.ID,
		ModuleID:  moduleLog.ModuleID,
		Level:     moduleLog.Level,
		Message:   moduleLog.Message,
		Meta:      moduleLog.Meta,
		CreatedAt: moduleLog.CreatedAt,
	}
}

func ModuleLogsToAPIModuleLogs(logs []core.ModuleLog) (dest []ModuleLog) {
	for _, module := range logs {
		dest = append(dest, ModuleLogToAPIModuleLog(module))
	}
	if len(dest) == 0 {
		return (make([]ModuleLog, 0))
	}
	return dest
}

func ModulePageToAPIModulePage(modulePage core.ModulePage) ModulePage {
	return ModulePage{
		ID:              modulePage.ID,
		ModuleID:        modulePage.ModuleID,
		Name:            modulePage.Name,
		Slug:            modulePage.Slug,
		TargetContainer: modulePage.TargetContainer,
		TargetPort:      modulePage.TargetPort,
		IframeOnly:      modulePage.IframeOnly,
		NeedAuth:        modulePage.NeedAuth,
		IconURL:         modulePage.IconURL,
		NetworkName:     modulePage.NetworkName,
	}
}

func ModulePagesToAPIModulePages(pages []core.ModulePage) (dest []ModulePage) {
	for _, module := range pages {
		dest = append(dest, ModulePageToAPIModulePage(module))
	}
	if len(dest) == 0 {
		return (make([]ModulePage, 0))
	}
	return dest
}
