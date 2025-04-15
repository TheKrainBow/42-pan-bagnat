package users

import (
	"net/http"
)

// // @Summary      Post User List
// // @Description  Download a new module for your campus
// // @Tags         users
// // @Accept       json
// // @Produce      json
// // @Param        input body UserPatchInput true "User input"
// // @Success      200 {object} User
// // @Router       /users/{userID} [patch]
func PatchUser(w http.ResponseWriter, r *http.Request) {
	// For now, User are not Patchable since it will fetch 42 API datas. Only Patch available will be Password for staff accounts
	w.Header().Set("Content-Type", "application/json")

	http.Error(w, "ID not found", http.StatusNotImplemented)
}
