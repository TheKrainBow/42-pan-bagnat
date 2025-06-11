package modules

import (
	api "backend/api/dto"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

// @Summary      Post Module List
// @Description  Download a new module for your campus
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body ModuleGitInput true "Git URL and SSH key"
// @Success      200 {object} Module
// @Router       /modules [post]
func PostModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse input
	var input struct {
		GitURL string `json:"git_url"`
		SSHKey string `json:"ssh_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Temporary dummy output
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := api.Module{
		ID:            id.String(),
		Name:          "Test",
		Version:       "1.2",
		Status:        api.Enabled,
		URL:           input.GitURL,
		LatestVersion: "1.7",
		LastUpdate:    time.Date(2025, 2, 18, 15, 0, 0, 0, time.UTC),
	}

	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
