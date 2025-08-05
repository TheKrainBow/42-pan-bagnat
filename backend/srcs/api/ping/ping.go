package ping

import (
	"fmt"
	"net/http"
)

// Define the model for the API version response
// @Description API version response model
type VersionResponse struct {
	Version string `json:"version" example:"1.1"`
}

// @Summary      Ping backend API
// @Description  Get a response from the API
// @Tags         Ping
// @Accept       json
// @Produce      json
// @Success      200 {object} string
// @Router       /ping [get]
func Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `Pong!`)
}
