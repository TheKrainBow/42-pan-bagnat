package dashboard

import (
	"backend/api/auth"
	"backend/core"
	"encoding/json"
	"log"
	"net/http"
)

type PutDashboardInput struct {
	CanvasJSON json.RawMessage `json:"canvas_json"`
}

// PutDashboard replaces the dashboard's canvas content.
// @Security     SessionAuth
// @Summary      Save Dashboard
// @Description  Replaces the dashboard canvas content shown to every user.
// @Tags         Dashboard
// @Accept       json
// @Produce      json
// @Param        input  body      PutDashboardInput  true  "New canvas content"
// @Success      200    {object}  core.Dashboard
// @Failure      400    {string}  string  "Invalid JSON body"
// @Failure      500    {string}  string  "Internal server error"
// @Router       /admin/dashboard [put]
func PutDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input PutDashboardInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	userID := ""
	if u, ok := r.Context().Value(auth.UserCtxKey).(*core.User); ok && u != nil {
		userID = u.ID
	}

	d, err := core.SaveDashboard(input.CanvasJSON, userID)
	if err != nil {
		log.Printf("error while saving dashboard: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(d); err != nil {
		log.Printf("error encoding dashboard: %s\n", err.Error())
	}
}
