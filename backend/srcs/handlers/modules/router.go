package modules

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetModules)
	r.Post("/", PostModule)
	r.Get("/{moduleID}", GetModule)
	r.Patch("/{moduleID}", PatchModule)
	r.Delete("/{moduleID}", DeleteModule)
}
