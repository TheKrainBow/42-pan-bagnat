package modules

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetModules)          // GET /api/v1/modules
	r.Post("/", PostModule)         // POST /api/v1/modules
	r.Get("/{moduleID}", GetModule) // GET /api/v1/modules/{moduleID}
	r.Delete("/{moduleID}", DeleteModule)

	r.Get("/{moduleID}/logs", GetModuleLogs)

	r.Post("/{moduleID}/git/clone", GitClone)
	r.Post("/{moduleID}/git/pull", GitPull)
	r.Post("/{moduleID}/git/update-remote", GitUpdateRemote)

	r.Get("/{moduleID}/config", GetModuleConfig)

	r.Post("/{moduleID}/deploy", DeployConfig)

	r.Get("/pages", GetPages)
	r.Route("/{moduleID}/pages", func(r chi.Router) {
		r.Get("/", GetModulePages)              // GET    /modules/{moduleID}/pages
		r.Post("/", PostModulePage)             // POST   /modules/{moduleID}/pages
		r.Patch("/{pageID}", PatchModulePage)   // PATCH  /modules/{moduleID}/pages/{pageID}
		r.Delete("/{pageID}", DeleteModulePage) // DELETE /modules/{moduleID}/pages/{pageID}
	})
}
