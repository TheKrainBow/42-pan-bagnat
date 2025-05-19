package roles

import (
	"backend/handlers/api"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

// @Summary      Post Role List
// @Description  Download a new module for your campus
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        input body RolePatchInput true "Role input"
// @Success      200 {object} Role
// @Router       /roles/{roleID} [patch]
func PatchRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "roleID")

	dest := api.Role{
		ID:    id,
		Name:  "Test",
		Color: "0xFF00FF",
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
