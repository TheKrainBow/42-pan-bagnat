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
		Version:       dbModule.Version,
		Status:        ModuleStatus(dbModule.Status),
		GitURL:        dbModule.GitURL,
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
