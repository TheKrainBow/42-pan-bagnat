package integrations

import (
	"backend/core"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// GetUser42 returns details about 42 related datas for a specific user by login.
// @Summary      Get User 42
// @Description  Retrieves a user’s 42 details given their 42 login.
// @Tags         integrations, 42
// @Accept       json
// @Produce      json
// @Param        login  path      string  true  "User 42 login"
// @Success      200    {object}  core.User42
// @Failure      400    {object}  ErrorResponse "Identifier is required"
// @Failure      404    {object}  ErrorResponse "User not found"
// @Failure      500    {object}  ErrorResponse "Internal server error"
// @Router       /admin/integrations/42/users/{login} [get]
func GetUser42(w http.ResponseWriter, r *http.Request) {
	login := chi.URLParam(r, "login")
	if strings.TrimSpace(login) == "" {
		writeJSONError(w, http.StatusBadRequest, "Identifier is required")
		return
	}

	user42, err := core.GetUser42(login) // or core.GetUser42(r.Context(), login) if you add ctx
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "User not found")
			return
		}
		log.Printf("error fetching user %q from 42: %v", login, err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// 200 by default; set explicitly if you prefer.
	if err := json.NewEncoder(w).Encode(user42); err != nil {
		// At this point headers may be partially sent; best-effort.
		log.Printf("encode response for login %q failed: %v", login, err)
	}
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Message: msg,
	})
}
