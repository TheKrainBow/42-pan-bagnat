package modules

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetModules)
	r.Post("/", PostModule)
	r.Get("/pages", GetPages)

	r.Get("/{moduleID}", GetModule)
	r.Delete("/{moduleID}", DeleteModule)

	r.Get("/{moduleID}/logs", GetModuleLogs)
	r.Get("/{moduleID}/config", GetModuleConfig)
	r.Post("/{moduleID}/deploy", DeployConfig)

	r.Post("/{moduleID}/git/clone", GitClone)
	r.Post("/{moduleID}/git/pull", GitPull)
	r.Post("/{moduleID}/git/update-remote", GitUpdateRemote)

	r.Get("/{moduleID}/pages", GetModulePages)
	r.Post("/{moduleID}/pages", PostModulePage)
	r.Patch("/{moduleID}/pages/{pageID}", PatchModulePage)
	r.Delete("/{moduleID}/pages/{pageID}", DeleteModulePage)

	r.Get("/{moduleID}/containers", GetModuleContainers)
	r.Get("/{moduleID}/containers/{containerName}/logs", GetContainerLogs)
	r.Post("/{moduleID}/containers/{containerName}/start", StartModuleContainer)
	r.Post("/{moduleID}/containers/{containerName}/stop", StopModuleContainer)
	r.Post("/{moduleID}/containers/{containerName}/restart", RestartModuleContainer)
	r.Delete("/{moduleID}/containers/{containerName}/delete", DeleteModuleContainer)
}
