package roles

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

// @Summary      Post Role List
// @Description  Download a new role for your campus
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        input body RolePatchInput true "Role input"
// @Success      200 {object} Role
// @Router       /roles [post]
func PostRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := Role{
		ID:    id.String(),
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
