package roles

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

// @Summary      Delete Role
// @Description  Delete a role for your campus (All role datas will be lost!)
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        input body RolePatchInput true "Role input"
// @Success      200
// @Router       /roles [delete]
func DeleteRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := Role{
		ID:   id.String(),
		Name: "Test",
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
