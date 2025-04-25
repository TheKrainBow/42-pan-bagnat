package roles

import "backend/core"

func RoleToAPIRole(role core.Role) Role {
	return Role{
		ID:    role.ID,
		Name:  role.Name,
		Color: role.Color,
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
