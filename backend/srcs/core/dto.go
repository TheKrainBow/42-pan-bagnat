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
