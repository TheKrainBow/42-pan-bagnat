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

	r.Get("/{moduleID}/logs", GetModuleLogs)

	r.Post("/{moduleID}/git/clone", GitClone)
	r.Post("/{moduleID}/git/pull", GitPull)
	r.Post("/{moduleID}/git/update-remote", GitUpdateRemote)

	r.Get("/{moduleID}/config", GetModuleConfig)
	r.Post("/{moduleID}/compose", ComposeModule)

	r.Post("/{moduleID}/deploy", DeployConfig)

	r.Get("/{moduleID}/pages", GetModulePages)
	r.Get("/pages", GetPages)
	r.Post("/{moduleID}/pages", PostModulePage)
	r.Delete("/{moduleID}/pages/{pageName}", DeleteModulePage)
}
