package version

import (
	"fmt"
	"net/http"
)

// Define the model for the API version response
// @Description API version response model
type VersionResponse struct {
	Version string `json:"version" example:"1.1"`
}

// @Summary      Get API version
// @Description  Returns the current version of the API
// @Tags         version
// @Accept       json
// @Produce      json
// @Success      200 {object} VersionResponse
// @Router       /version [get]
func GetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"version": "1.1"}`)
}
