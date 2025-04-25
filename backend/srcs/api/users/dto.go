package users

import (
	"backend/api/roles"
	"backend/core"
)

func UserToAPIUser(user core.User) User {
	return User{
		ID:        user.ID,
		FtID:      user.FtID,
		FtLogin:   user.FtLogin,
		FtIsStaff: user.FtIsStaff,
		IsStaff:   user.IsStaff,
		PhotoURL:  user.PhotoURL,
		LastSeen:  user.LastSeen,
		Roles:     roles.RolesToAPIRoles(user.Roles),
	}
}

func UsersToAPIUsers(users []core.User) (dest []User) {
	for _, user := range users {
		dest = append(dest, UserToAPIUser(user))
	}
	return dest
}
