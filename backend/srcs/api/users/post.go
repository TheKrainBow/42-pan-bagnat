package users

import (
	api "backend/api/dto"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

// @Summary      Post User List
// @Description  Download a new user for your campus
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        input body UserPostInput true "User input"
// @Success      200 {object} User
// @Router       /users [post]
func PostUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	dest := api.User{
		ID: id.String(),
	}

	// Marshal the dest struct into JSON
	destJSON, err := json.Marshal(dest)
	if err != nil {
		http.Error(w, "Failed to convert struct to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(destJSON))
}
