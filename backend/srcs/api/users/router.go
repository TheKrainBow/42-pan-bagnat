package users

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetUsers)
	r.Post("/", PostUser)
	r.Get("/{identifier}", GetUser)
	r.Get("/me", GetUserMe)
	r.Patch("/{identifier}", PatchUser)
	r.Delete("/{identifier}", DeleteUser)
	r.Post("/{identifier}/roles/{role_id}", PostUserRole)
	r.Delete("/{identifier}/roles/{role_id}", DeleteUserRole)
}
