package auth

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router) {
	r.Get("/42/login", StartLogin)
	r.Get("/42/callback", Callback)
}
