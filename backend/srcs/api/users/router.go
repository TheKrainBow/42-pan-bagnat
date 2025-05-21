package users

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetUsers)
	r.Post("/", PostUser)
	r.Get("/{userID}", GetUser)
	r.Patch("/{userID}", PatchUser)
	r.Delete("/{userID}", DeleteUser)
}
