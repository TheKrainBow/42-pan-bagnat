package integrations

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router) {
	r.Get("/42/users/{login}", GetUser42)
}
