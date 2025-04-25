package database

func GetRoleModules(roleID string) ([]Module, error) {
	rows, err := mainDB.Query(`
		SELECT mod.id, mod.name, mod.version, mod.status, mod.url, mod.latest_version, mod.late_commits, mod.last_update
		FROM modules mod
		JOIN module_roles ur ON ur.module_id = mod.id
		WHERE ur.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []Module
	for rows.Next() {
		var module Module
		if err := rows.Scan(
			&module.ID,
			&module.Name,
			&module.Version,
			&module.Status,
			&module.URL,
			&module.LatestVersion,
			&module.LateCommits,
			&module.LastUpdate,
		); err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}
