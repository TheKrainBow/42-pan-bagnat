package core

import (
	"backend/database"
	"net/http"
	"time"
)

const defaultStatsRangeDays = 30

// ParseStatsRange reads "from"/"to" (RFC3339 or YYYY-MM-DD) query params, defaulting
// to the last 30 days when absent.
func ParseStatsRange(r *http.Request) (time.Time, time.Time, error) {
	to := time.Now().UTC()
	from := to.AddDate(0, 0, -defaultStatsRangeDays)

	if v := r.URL.Query().Get("to"); v != "" {
		parsed, err := parseStatsDate(v)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = parsed
	}
	if v := r.URL.Query().Get("from"); v != "" {
		parsed, err := parseStatsDate(v)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = parsed
	}
	return from, to, nil
}

func parseStatsDate(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

type GlobalStats struct {
	Summary database.GlobalStatsSummary   `json:"summary"`
	Modules []database.ModuleStatsSummary `json:"modules"`
	Daily   []database.DailyActivityPoint `json:"daily"`
	Heatmap []database.HeatmapCell        `json:"heatmap"`
}

// GetGlobalStats aggregates cross-module activity for the admin Stats overview tab.
func GetGlobalStats(from, to time.Time) (GlobalStats, error) {
	summary, err := database.GetGlobalStatsSummary(from, to)
	if err != nil {
		return GlobalStats{}, err
	}
	modules, err := database.GetModuleStatsSummaries(from, to)
	if err != nil {
		return GlobalStats{}, err
	}
	daily, err := database.GetDailyActivity(from, to, "")
	if err != nil {
		return GlobalStats{}, err
	}
	heatmap, err := database.GetActivityHeatmap(from, to, "")
	if err != nil {
		return GlobalStats{}, err
	}
	return GlobalStats{
		Summary: summary,
		Modules: modules,
		Daily:   daily,
		Heatmap: heatmap,
	}, nil
}

type ModuleStats struct {
	Module  ModuleSummary                 `json:"module"`
	Daily   []database.DailyActivityPoint `json:"daily"`
	Heatmap []database.HeatmapCell        `json:"heatmap"`
	Users   []database.ModuleUserActivity `json:"users"`
}

// GetModuleStats aggregates activity for a single module for the admin Stats module-detail tab.
func GetModuleStats(moduleID string, from, to time.Time) (ModuleStats, error) {
	module, err := GetModule(moduleID)
	if err != nil {
		return ModuleStats{}, err
	}
	daily, err := database.GetDailyActivity(from, to, moduleID)
	if err != nil {
		return ModuleStats{}, err
	}
	heatmap, err := database.GetActivityHeatmap(from, to, moduleID)
	if err != nil {
		return ModuleStats{}, err
	}
	users, err := database.GetModuleUserActivity(from, to, moduleID)
	if err != nil {
		return ModuleStats{}, err
	}
	return ModuleStats{
		Module: ModuleSummary{
			ID:      module.ID,
			Name:    module.Name,
			Slug:    module.Slug,
			IconURL: module.IconURL,
		},
		Daily:   daily,
		Heatmap: heatmap,
		Users:   users,
	}, nil
}
