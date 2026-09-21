package database

import "time"

// activitySessionsCTE sessionizes raw module_activity pings into "activities":
// consecutive pings from the same user on the same module are merged into one
// activity as long as the gap between them stays under 60 minutes. A gap of
// 60 minutes or more (or the first ping seen) starts a new activity.
const activitySessionsCTE = `
	WITH ordered AS (
		SELECT module_id, user_id, created_at,
		       created_at - LAG(created_at) OVER (
		           PARTITION BY module_id, user_id ORDER BY created_at
		       ) AS gap
		  FROM module_activity
		 WHERE created_at >= $1 AND created_at < $2
	),
	flagged AS (
		SELECT module_id, user_id, created_at,
		       CASE WHEN gap IS NULL OR gap >= INTERVAL '60 minutes' THEN 1 ELSE 0 END AS is_new
		  FROM ordered
	),
	grouped AS (
		SELECT module_id, user_id, created_at,
		       SUM(is_new) OVER (
		           PARTITION BY module_id, user_id ORDER BY created_at
		           ROWS UNBOUNDED PRECEDING
		       ) AS session_group
		  FROM flagged
	),
	sessions_cte AS (
		SELECT module_id, user_id, session_group,
		       MIN(created_at) AS started_at,
		       MAX(created_at) AS ended_at
		  FROM grouped
		 GROUP BY module_id, user_id, session_group
	)
`

type GlobalStatsSummary struct {
	ActivityCount int `json:"activity_count" db:"activity_count"`
	UniqueUsers   int `json:"unique_users" db:"unique_users"`
	ActiveModules int `json:"active_modules" db:"active_modules"`
}

type ModuleStatsSummary struct {
	ModuleID      string     `json:"module_id" db:"module_id"`
	ModuleName    string     `json:"module_name" db:"module_name"`
	ModuleSlug    string     `json:"module_slug" db:"module_slug"`
	IconURL       string     `json:"icon_url" db:"icon_url"`
	ActivityCount int        `json:"activity_count" db:"activity_count"`
	UniqueUsers   int        `json:"unique_users" db:"unique_users"`
	LastActiveAt  *time.Time `json:"last_active_at" db:"last_active_at"`
}

type DailyActivityPoint struct {
	Day           time.Time `json:"day" db:"day"`
	ActivityCount int       `json:"activity_count" db:"activity_count"`
	UniqueUsers   int       `json:"unique_users" db:"unique_users"`
}

type HeatmapCell struct {
	Weekday int `json:"weekday" db:"weekday"` // 0=Sunday .. 6=Saturday
	Hour    int `json:"hour" db:"hour"`
	Pings   int `json:"pings" db:"pings"`
}

type ModuleUserActivity struct {
	UserID        string    `json:"user_id" db:"user_id"`
	FtLogin       string    `json:"ft_login" db:"ft_login"`
	PhotoURL      string    `json:"photo_url" db:"photo_url"`
	ActivityCount int       `json:"activity_count" db:"activity_count"`
	LastActiveAt  time.Time `json:"last_active_at" db:"last_active_at"`
}

// GetGlobalStatsSummary returns activity totals across all modules for the given range.
func GetGlobalStatsSummary(from, to time.Time) (GlobalStatsSummary, error) {
	var s GlobalStatsSummary
	err := mainDB.Get(&s, `
		`+activitySessionsCTE+`
		SELECT COUNT(*) AS activity_count,
		       COUNT(DISTINCT user_id) AS unique_users,
		       COUNT(DISTINCT module_id) AS active_modules
		  FROM sessions_cte
	`, from, to)
	return s, err
}

// GetModuleStatsSummaries returns per-module activity totals for the given range, busiest first.
func GetModuleStatsSummaries(from, to time.Time) ([]ModuleStatsSummary, error) {
	rows, err := mainDB.Queryx(`
		`+activitySessionsCTE+`
		SELECT m.id AS module_id,
		       m.name AS module_name,
		       m.slug AS module_slug,
		       m.icon_url AS icon_url,
		       COUNT(sc.*) AS activity_count,
		       COUNT(DISTINCT sc.user_id) AS unique_users,
		       MAX(sc.ended_at) AS last_active_at
		  FROM sessions_cte sc
		  JOIN modules m ON m.id = sc.module_id
		 GROUP BY m.id, m.name, m.slug, m.icon_url
		 ORDER BY activity_count DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModuleStatsSummary
	for rows.Next() {
		var s ModuleStatsSummary
		if err := rows.StructScan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetDailyActivity returns one point per calendar day in the range, optionally filtered to a single module.
func GetDailyActivity(from, to time.Time, moduleID string) ([]DailyActivityPoint, error) {
	moduleFilter := ""
	args := []any{from, to}
	if moduleID != "" {
		moduleFilter = "WHERE module_id = $3"
		args = append(args, moduleID)
	}
	rows, err := mainDB.Queryx(`
		`+activitySessionsCTE+`
		SELECT date_trunc('day', started_at) AS day,
		       COUNT(*) AS activity_count,
		       COUNT(DISTINCT user_id) AS unique_users
		  FROM sessions_cte
		  `+moduleFilter+`
		 GROUP BY day
		 ORDER BY day ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DailyActivityPoint
	for rows.Next() {
		var p DailyActivityPoint
		if err := rows.StructScan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetActivityHeatmap buckets raw pings by weekday/hour to surface peak usage times,
// optionally filtered to a single module. Raw pings (not sessionized activities) are
// used here so a long activity contributes to every hour it actually spans.
func GetActivityHeatmap(from, to time.Time, moduleID string) ([]HeatmapCell, error) {
	moduleFilter := ""
	args := []any{from, to}
	if moduleID != "" {
		moduleFilter = "AND module_id = $3"
		args = append(args, moduleID)
	}
	rows, err := mainDB.Queryx(`
		SELECT EXTRACT(DOW FROM created_at)::int AS weekday,
		       EXTRACT(HOUR FROM created_at)::int AS hour,
		       COUNT(*) AS pings
		  FROM module_activity
		 WHERE created_at >= $1 AND created_at < $2
		 `+moduleFilter+`
		 GROUP BY weekday, hour
		 ORDER BY weekday, hour
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HeatmapCell
	for rows.Next() {
		var c HeatmapCell
		if err := rows.StructScan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetModuleUserActivity returns per-user activity counts for a single module, busiest first.
func GetModuleUserActivity(from, to time.Time, moduleID string) ([]ModuleUserActivity, error) {
	rows, err := mainDB.Queryx(`
		`+activitySessionsCTE+`
		SELECT u.id AS user_id,
		       u.ft_login AS ft_login,
		       u.photo_url AS photo_url,
		       COUNT(sc.*) AS activity_count,
		       MAX(sc.ended_at) AS last_active_at
		  FROM sessions_cte sc
		  JOIN users u ON u.id = sc.user_id
		 WHERE sc.module_id = $3
		 GROUP BY u.id, u.ft_login, u.photo_url
		 ORDER BY activity_count DESC
	`, from, to, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModuleUserActivity
	for rows.Next() {
		var a ModuleUserActivity
		if err := rows.StructScan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
