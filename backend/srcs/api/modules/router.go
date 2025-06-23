package modules

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetModules)          // GET /api/v1/modules
	r.Post("/", PostModule)         // POST /api/v1/modules
	r.Get("/{moduleID}", GetModule) // GET /api/v1/modules/{moduleID}
	r.Patch("/{moduleID}", PatchModule)
	r.Delete("/{moduleID}", DeleteModule)

	r.Post("/{moduleID}/git/clone", GitClone)
	r.Post("/{moduleID}/git/pull", GitPull)
	r.Post("/{moduleID}/git/update-remote", GitUpdateRemote)
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
