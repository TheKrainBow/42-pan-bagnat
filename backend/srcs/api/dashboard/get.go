package dashboard

import (
	"backend/core"
	"encoding/json"
	"log"
	"net/http"
)

// GetDashboard returns the current global dashboard (canvas content shown to
// every user instead of auto-opening their first module).
// @Security     SessionAuth
// @Summary      Get Dashboard
// @Description  Returns the current dashboard canvas content.
// @Tags         Dashboard
// @Produce      json
// @Success      200  {object}  core.Dashboard
// @Failure      500  {string}  string  "Internal server error"
// @Router       /dashboard [get]
func GetDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	d, err := core.GetDashboard()
	if err != nil {
		log.Printf("error while getting dashboard: %s\n", err.Error())
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(d); err != nil {
		log.Printf("error encoding dashboard: %s\n", err.Error())
	}
}
