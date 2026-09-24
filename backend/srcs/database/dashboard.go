package database

import (
	"encoding/json"
	"time"
)

const dashboardID = "default"

type Dashboard struct {
	ID         string          `json:"id" db:"id"`
	CanvasJSON json.RawMessage `json:"canvas_json" db:"canvas_json"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
	UpdatedBy  *string         `json:"updated_by" db:"updated_by"`
}

func GetDashboard() (Dashboard, error) {
	var d Dashboard
	err := mainDB.Get(&d, `
		SELECT id, canvas_json, updated_at, updated_by
		FROM dashboard
		WHERE id = $1
	`, dashboardID)
	return d, err
}

func SaveDashboard(canvasJSON json.RawMessage, updatedByUserID string) (Dashboard, error) {
	var updatedBy *string
	if updatedByUserID != "" {
		updatedBy = &updatedByUserID
	}
	var d Dashboard
	err := mainDB.Get(&d, `
		UPDATE dashboard
		   SET canvas_json = $1,
		       updated_at  = NOW(),
		       updated_by  = $2
		 WHERE id = $3
	 RETURNING id, canvas_json, updated_at, updated_by
	`, canvasJSON, updatedBy, dashboardID)
	return d, err
}
