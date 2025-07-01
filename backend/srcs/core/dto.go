package core

import "backend/database"

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

func DatabaseUsersToUsers(dbUsers []database.User) (dest []User) {
	for _, user := range dbUsers {
		dest = append(dest, DatabaseUserToUser(user))
	}
	return dest
}

func DatabaseModuleToModule(dbModule database.Module) Module {
	return Module{
		ID:            dbModule.ID,
		SSHPublicKey:  dbModule.SSHPublicKey,
		SSHPrivateKey: dbModule.SSHPrivateKey,
		Name:          dbModule.Name,
		Slug:          dbModule.Slug,
		Version:       dbModule.Version,
		Status:        ModuleStatus(dbModule.Status),
		GitURL:        dbModule.GitURL,
		GitBranch:     dbModule.GitBranch,
		IconURL:       dbModule.IconURL,
		LatestVersion: dbModule.LatestVersion,
		LateCommits:   dbModule.LateCommits,
		LastUpdate:    dbModule.LastUpdate,
	}
}

func DatabaseModulesToModules(dbModules []database.Module) (dest []Module) {
	for _, module := range dbModules {
		dest = append(dest, DatabaseModuleToModule(module))
	}
	return dest
}

func DatabaseModuleLogToModuleLog(dbLog database.ModuleLog) ModuleLog {
	return ModuleLog{
		ID:        dbLog.ID,
		ModuleID:  dbLog.ModuleID,
		Level:     dbLog.Level,
		Message:   dbLog.Message,
		Meta:      dbLog.Meta,
		CreatedAt: dbLog.CreatedAt,
	}
}

func DatabaseModuleLogsToModuleLogs(dbLogs []database.ModuleLog) (dest []ModuleLog) {
	for _, log := range dbLogs {
		dest = append(dest, DatabaseModuleLogToModuleLog(log))
	}
	return dest
}

func DatabaseModulePageToModulePage(dbPage database.ModulePage) ModulePage {
	return ModulePage{
		ModuleID:    dbPage.ModuleID,
		Name:        dbPage.Name,
		DisplayName: dbPage.DisplayName,
		URL:         dbPage.URL,
		IsPublic:    dbPage.IsPublic,
	}
}

func DatabaseModulePagesToModulePages(dbPages []database.ModulePage) (dest []ModulePage) {
	for _, log := range dbPages {
		dest = append(dest, DatabaseModulePageToModulePage(log))
	}
	return dest
}
