package core

import (
	"backend/database"
	"encoding/json"
	"fmt"
	"time"
)

type Dashboard struct {
	CanvasJSON json.RawMessage `json:"canvas_json"`
	UpdatedAt  time.Time       `json:"updated_at"`
	UpdatedBy  *string         `json:"updated_by,omitempty"`
}

func databaseDashboardToDashboard(d database.Dashboard) Dashboard {
	return Dashboard{
		CanvasJSON: d.CanvasJSON,
		UpdatedAt:  d.UpdatedAt,
		UpdatedBy:  d.UpdatedBy,
	}
}

func GetDashboard() (Dashboard, error) {
	d, err := database.GetDashboard()
	if err != nil {
		return Dashboard{}, fmt.Errorf("couldn't get dashboard in db: %w", err)
	}
	return databaseDashboardToDashboard(d), nil
}

func SaveDashboard(canvasJSON json.RawMessage, updatedByUserID string) (Dashboard, error) {
	if !json.Valid(canvasJSON) {
		return Dashboard{}, fmt.Errorf("canvas_json is not valid JSON")
	}
	d, err := database.SaveDashboard(canvasJSON, updatedByUserID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("couldn't save dashboard in db: %w", err)
	}
	return databaseDashboardToDashboard(d), nil
}
