package database

import (
	"fmt"
	"strings"
)

type ModuleOrderField string

const (
	ModuleID            ModuleOrderField = "id"
	ModuleName          ModuleOrderField = "name"
	ModuleVersion       ModuleOrderField = "version"
	ModuleStatus        ModuleOrderField = "status"
	ModuleURL           ModuleOrderField = "url"
	ModuleLatestVersion ModuleOrderField = "latest_version"
	ModuleLateCommits   ModuleOrderField = "late_commits"
	ModuleLastUpdate    ModuleOrderField = "last_update"
)

type ModuleOrder struct {
	Field ModuleOrderField
	Order OrderDirection
}

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

func GetAllModules(
	orderBy *[]ModuleOrder,
	filter string,
	lastModule *Module,
	limit int,
) ([]Module, error) {
	// 1) Default to ordering by ID ASC if none provided
	if orderBy == nil || len(*orderBy) == 0 {
		tmp := []ModuleOrder{{Field: ModuleID, Order: Asc}}
		orderBy = &tmp
	}

	// 2) Build ORDER BY clauses
	hasID := false
	var orderClauses []string
	for _, ord := range *orderBy {
		orderClauses = append(orderClauses,
			fmt.Sprintf("%s %s", ord.Field, ord.Order),
		)
		if ord.Field == ModuleID {
			hasID = true
		}
	}
	// Always append ID as tie-breaker in same direction as first field
	firstOrder := (*orderBy)[0].Order
	if !hasID {
		orderClauses = append(orderClauses,
			fmt.Sprintf("id %s", firstOrder),
		)
	}

	// 3) Build WHERE conditions and collect args
	var whereConds []string
	var args []any
	argPos := 1

	// Cursor pagination: tuple comparison
	if lastModule != nil {
		var cols, placeholders []string
		for _, ord := range *orderBy {
			cols = append(cols, string(ord.Field))
			placeholders = append(placeholders,
				fmt.Sprintf("$%d", argPos),
			)
			switch ord.Field {
			case ModuleID:
				args = append(args, lastModule.ID)
			case ModuleName:
				args = append(args, lastModule.Name)
			case ModuleVersion:
				args = append(args, lastModule.Version)
			case ModuleStatus:
				args = append(args, lastModule.Status)
			case ModuleURL:
				args = append(args, lastModule.URL)
			case ModuleLatestVersion:
				args = append(args, lastModule.LatestVersion)
			case ModuleLateCommits:
				args = append(args, lastModule.LateCommits)
			case ModuleLastUpdate:
				args = append(args, lastModule.LastUpdate)
			}
			argPos++
		}
		if !hasID {
			cols = append(cols, "id")
			placeholders = append(placeholders,
				fmt.Sprintf("$%d", argPos),
			)
			args = append(args, lastModule.ID)
			argPos++
		}

		// Asc vs Desc on the first order field
		dir := ">"
		if firstOrder == Desc {
			dir = "<"
		}

		whereConds = append(whereConds,
			fmt.Sprintf("(%s) %s (%s)",
				strings.Join(cols, ", "),
				dir,
				strings.Join(placeholders, ", "),
			),
		)
	}

	// Text‐filter only on Name
	if filter != "" {
		whereConds = append(whereConds,
			fmt.Sprintf("name ILIKE '%%' || $%d || '%%'", argPos),
		)
		args = append(args, filter)
		argPos++
	}

	// 4) Assemble SQL
	var sb strings.Builder
	sb.WriteString(
		`SELECT id, name, version, status, url, latest_version, late_commits, last_update
FROM modules`,
	)
	if len(whereConds) > 0 {
		sb.WriteString("\nWHERE ")
		sb.WriteString(strings.Join(whereConds, " AND "))
	}
	sb.WriteString("\nORDER BY ")
	sb.WriteString(strings.Join(orderClauses, ", "))
	if limit > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d", limit))
	}
	sb.WriteString(";")

	query := sb.String()

	// 5) Execute and scan
	rows, err := mainDB.Query(query, args...)
	if err != nil {
		return []Module{}, err
	}
	defer rows.Close()

	out := []Module{}
	for rows.Next() {
		var m Module
		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.Version,
			&m.Status,
			&m.URL,
			&m.LatestVersion,
			&m.LateCommits,
			&m.LastUpdate,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}
