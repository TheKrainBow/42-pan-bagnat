package users

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

// @Summary      Delete User
// @Description  Delete a user for your campus (All user datas will be lost!)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        input body UserPatchInput true "User input"
// @Success      200
// @Router       /users [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := User{
		ID:            id.String(),
		Name:          "Test",
		Version:       "1.2",
		Status:        Enabled,
		URL:           "https://github.com/some-user/some-repo",
		LatestVersion: "1.7",
		LastUpdate:    time.Date(2025, 02, 18, 15, 0, 0, 0, time.UTC),
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
