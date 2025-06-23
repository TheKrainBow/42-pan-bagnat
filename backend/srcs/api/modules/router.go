package modules

import (
	"backend/api/modules/git"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetModules)
	r.Post("/", PostModule)
	r.Get("/{moduleID}", GetModule)
	r.Patch("/{moduleID}", PatchModule)
	r.Delete("/{moduleID}", DeleteModule)

	// Git operations
	r.Route("/{moduleID}/git", func(r chi.Router) {
		r.Post("/clone", git.GitClone)
		r.Post("/pull", git.GitPull)
		r.Post("/update-remote", git.GitUpdateRemote)
		// r.Get("/status", git.GitStatus) // Optional
	})

	// Docker operations
	// r.Route("/{moduleID}/docker", func(r chi.Router) {
	// 	r.Post("/start", DockerStart)
	// 	r.Post("/stop", DockerStop)
	// 	r.Post("/restart", DockerRestart)
	// 	r.Get("/logs", DockerLogs)
	// })

	// // Config operations
	// r.Route("/{moduleID}/config", func(r chi.Router) {
	// 	r.Get("/", GetModuleConfig)
	// 	r.Patch("/", PatchModuleConfig)
	// })
}
