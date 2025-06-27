package database

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type ModuleOrderField string

const (
	ModuleID            ModuleOrderField = "id"
	ModuleName          ModuleOrderField = "name"
	ModuleSlug          ModuleOrderField = "slug"
	ModuleVersion       ModuleOrderField = "version"
	ModuleStatus        ModuleOrderField = "status"
	ModuleGitURL        ModuleOrderField = "git_url"
	ModuleGitBranch     ModuleOrderField = "git_branch"
	ModuleLatestVersion ModuleOrderField = "latest_version"
	ModuleLateCommits   ModuleOrderField = "late_commits"
	ModuleLastUpdate    ModuleOrderField = "last_update"
)

type ModuleLogsOrderField string

const (
	ModuleLogsCreatedAt ModuleLogsOrderField = "created_at"
	ModuleLogsID        ModuleLogsOrderField = "id"
)

type Module struct {
	ID            string    `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	SSHPublicKey  string    `json:"ssh_public_key" example:"ssh-rsa AAAA..."`
	SSHPrivateKey string    `json:"ssh_private_key" example:"-----BEGIN OPENSSH PRIVATE KEY-----..."`
	Name          string    `json:"name" example:"Captain Hook"`
	Slug          string    `json:"slug" example:"captain-hook-main"`
	Version       string    `json:"version" example:"1.2"`
	Status        string    `json:"status" example:"enabled"`
	GitURL        string    `json:"git_url" example:"https://github.com/some-user/some-repo"`
	GitBranch     string    `json:"git_branch" example:"main"`
	IconURL       string    `json:"icon_url" example:"https://someURL/image.png"`
	LatestVersion string    `json:"latest_version" example:"1.7"`
	LateCommits   int       `json:"late_commits" example:"2"`
	LastUpdate    time.Time `json:"last_update" example:"2025-02-18T15:00:00Z"`
}

type ModulePatch struct {
	ID            string     `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	SSHPublicKey  *string    `json:"ssh_public_key" example:"ssh-rsa AAAA..."`
	SSHPrivateKey *string    `json:"ssh_private_key" example:"-----BEGIN OPENSSH PRIVATE KEY-----..."`
	Name          *string    `json:"name" example:"captain-hook"`
	Version       *string    `json:"version" example:"1.2"`
	Status        *string    `json:"status" example:"enabled"`
	GitURL        *string    `json:"git_url" example:"https://github.com/some-user/some-repo"`
	IconURL       *string    `json:"icon_url" example:"https://someURL/image.png"`
	LatestVersion *string    `json:"latest_version" example:"1.7"`
	LateCommits   *int       `json:"late_commits" example:"2"`
	LastUpdate    *time.Time `json:"last_update" example:"2025-02-18T15:00:00Z"`
}

type ModuleLog struct {
	ID        int64                  `json:"id"`
	ModuleID  string                 `json:"module_id"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Meta      map[string]interface{} `json:"meta"`
	CreatedAt time.Time              `json:"created_at"`
}

type ModuleOrder struct {
	Field ModuleOrderField
	Order OrderDirection
}

type ModuleLogsOrder struct {
	Field ModuleLogsOrderField
	Order OrderDirection
}

type ModuleLogPagination struct {
	OrderBy  *[]ModuleLogsOrder
	ModuleID string
	LastLog  *ModuleLog
	Limit    int
	Filter   string
}

func GetModule(moduleID string) (*Module, error) {
	row := mainDB.QueryRow(`
		SELECT id, ssh_public_key, ssh_private_key, name, slug, version, status, git_url, git_branch, icon_url, latest_version, late_commits, last_update
		FROM modules
		WHERE id = $1
	`, moduleID)

	var module Module
	if err := row.Scan(
		&module.ID,
		&module.SSHPublicKey,
		&module.SSHPrivateKey,
		&module.Name,
		&module.Slug,
		&module.Version,
		&module.Status,
		&module.GitURL,
		&module.GitBranch,
		&module.IconURL,
		&module.LatestVersion,
		&module.LateCommits,
		&module.LastUpdate,
	); err != nil {
		return nil, err
	}
	return &module, nil
}

func IsSlugTaken(slug string) (bool, error) {
	var exists bool
	err := mainDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM modules WHERE slug = $1
		)
	`, slug).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func GetModuleRoles(moduleID string) ([]Role, error) {
	rows, err := mainDB.Query(`
		SELECT r.id, r.name, r.color
		FROM roles r
		JOIN module_roles ur ON ur.role_id = r.id
		WHERE ur.module_id = $1
	`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Color); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
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
			case ModuleSlug:
				args = append(args, lastModule.Slug)
			case ModuleVersion:
				args = append(args, lastModule.Version)
			case ModuleStatus:
				args = append(args, lastModule.Status)
			case ModuleGitURL:
				args = append(args, lastModule.GitURL)
			case ModuleGitBranch:
				args = append(args, lastModule.GitBranch)
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
		`SELECT id, ssh_private_key, ssh_public_key, name, slug, version, status, git_url, git_branch, icon_url, latest_version, late_commits, last_update
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
			&m.SSHPrivateKey,
			&m.SSHPublicKey,
			&m.Name,
			&m.Slug,
			&m.Version,
			&m.Status,
			&m.GitURL,
			&m.GitBranch,
			&m.IconURL,
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

func InsertModule(m Module) error {
	status := m.Status
	if m.LastUpdate.IsZero() {
		status = "waiting_for_action"
	}

	_, err := mainDB.Exec(`
		INSERT INTO modules (id, name, slug, git_url, git_branch, ssh_public_key, ssh_private_key, last_update, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, m.ID, m.Name, m.Slug, m.GitURL, m.GitBranch, m.SSHPublicKey, m.SSHPrivateKey, m.LastUpdate, status)
	return err
}

// InsertModuleLog inserts a new entry into module_log.
// It marshals the Meta map into JSONB and relies on the DB default
// to set created_at = now().
func InsertModuleLog(l ModuleLog) error {
	metaJSON, err := json.Marshal(l.Meta)
	if err != nil {
		return fmt.Errorf("failed to marshal meta: %w", err)
	}

	_, err = mainDB.Exec(`
		INSERT INTO module_log (module_id, level, message, meta)
		VALUES ($1, $2, $3, $4)
	`, l.ModuleID, l.Level, l.Message, metaJSON)
	if err != nil {
		return fmt.Errorf("failed to insert module_log: %w", err)
	}

	return nil
}

func PatchModule(patch ModulePatch) error {
	if patch.ID == "" {
		return fmt.Errorf("missing module ID")
	}

	v := reflect.ValueOf(patch)
	t := reflect.TypeOf(patch)

	var (
		setClauses []string
		args       []interface{}
		argPos     = 1
	)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "ID" {
			continue
		}

		val := v.Field(i)
		if val.IsNil() {
			continue
		}

		dbField := field.Tag.Get("json")
		if dbField == "" {
			dbField = strings.ToLower(field.Name)
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", dbField, argPos))
		args = append(args, val.Elem().Interface())
		argPos++
	}

	if len(setClauses) == 0 {
		return nil // nothing to update
	}

	// Always update last_update to NOW()
	setClauses = append(setClauses, "last_update = NOW()")

	query := fmt.Sprintf(`
		UPDATE modules
		SET %s
		WHERE id = $%d
	`, strings.Join(setClauses, ", "), argPos)
	args = append(args, patch.ID)

	_, err := mainDB.Exec(query, args...)
	return err
}

// GetModuleLogs pages through module_log for one module,
// using the same pattern as GetAllModules.
func GetModuleLogs(p ModuleLogPagination) ([]ModuleLog, error) {
	// 1) Default ordering if none specified
	if p.OrderBy == nil || len(*p.OrderBy) == 0 {
		tmp := []ModuleLogsOrder{{Field: ModuleLogsCreatedAt, Order: Desc}}
		p.OrderBy = &tmp
	}

	// 2) Build ORDER BY clauses, track if we saw ID
	hasID := false
	var orderClauses []string
	for _, ord := range *p.OrderBy {
		orderClauses = append(orderClauses,
			fmt.Sprintf("%s %s", ord.Field, ord.Order),
		)
		if ord.Field == ModuleLogsID {
			hasID = true
		}
	}
	firstOrder := (*p.OrderBy)[0].Order
	if !hasID {
		orderClauses = append(orderClauses,
			fmt.Sprintf("%s %s", ModuleLogsID, firstOrder),
		)
	}

	// 3) Build WHERE conditions & collect args
	var whereConds []string
	var args []any
	argPos := 1

	// always filter by module_id
	whereConds = append(whereConds, fmt.Sprintf("module_id = $%d", argPos))
	args = append(args, p.ModuleID)
	argPos++

	// apply text‐filter on message
	if p.Filter != "" {
		whereConds = append(whereConds,
			fmt.Sprintf("message ILIKE '%%' || $%d || '%%'", argPos),
		)
		args = append(args, p.Filter)
		argPos++
	}

	// cursor pagination: tuple comparison on (created_at, id)
	if p.LastLog != nil {
		cmp := "<"
		if firstOrder == Asc {
			cmp = ">"
		}
		// build tuples
		var cols, holders []string
		for _, ord := range *p.OrderBy {
			cols = append(cols, string(ord.Field))
			holders = append(holders, fmt.Sprintf("$%d", argPos))
			switch ord.Field {
			case ModuleLogsCreatedAt:
				args = append(args, p.LastLog.CreatedAt)
			case ModuleLogsID:
				args = append(args, p.LastLog.ID)
			}
			argPos++
		}
		if !hasID {
			cols = append(cols, string(ModuleLogsID))
			holders = append(holders, fmt.Sprintf("$%d", argPos))
			args = append(args, p.LastLog.ID)
			argPos++
		}
		whereConds = append(whereConds,
			fmt.Sprintf("(%s) %s (%s)",
				strings.Join(cols, ", "),
				cmp,
				strings.Join(holders, ", "),
			),
		)
	}

	// 4) Assemble SQL
	var sb strings.Builder
	sb.WriteString(`
SELECT id, module_id, created_at, level, message, meta
  FROM module_log`)
	if len(whereConds) > 0 {
		sb.WriteString("\nWHERE ")
		sb.WriteString(strings.Join(whereConds, " AND "))
	}
	sb.WriteString("\nORDER BY ")
	sb.WriteString(strings.Join(orderClauses, ", "))
	if p.Limit > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d", p.Limit))
	}
	sb.WriteString(";")
	query := sb.String()

	// 5) Execute and scan
	rows, err := mainDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModuleLog
	for rows.Next() {
		var l ModuleLog
		var metaBytes []byte
		if err := rows.Scan(
			&l.ID,
			&l.ModuleID,
			&l.CreatedAt,
			&l.Level,
			&l.Message,
			&metaBytes,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metaBytes, &l.Meta); err != nil {
			return nil, err
		}
		out = append(out, l)
	}

	return out, nil
}
