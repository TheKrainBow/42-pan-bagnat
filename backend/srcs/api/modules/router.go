package modules

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetModules)
	r.Post("/", PostModule)

	r.Get("/{moduleID}", GetModule)
	r.Delete("/{moduleID}", DeleteModule)

	r.Post("/{moduleID}/roles/{roleID}", PostModuleRole)
	r.Delete("/{moduleID}/roles/{roleID}", DeleteModuleRole)

	r.Get("/{moduleID}/logs", GetModuleLogs)

	r.Post("/{moduleID}/git/clone", GitClone)
	r.Post("/{moduleID}/git/pull", GitPull)
	r.Post("/{moduleID}/git/update-remote", GitUpdateRemote)

	r.Get("/{moduleID}/pages", GetModulePages)
	r.Post("/{moduleID}/pages", PostModulePage)
	r.Patch("/{moduleID}/pages/{pageID}", PatchModulePage)
	r.Delete("/{moduleID}/pages/{pageID}", DeleteModulePage)

	r.Get("/{moduleID}/docker/config", GetModuleConfig)
	r.Post("/{moduleID}/docker/deploy", DeployConfig)

	r.Get("/{moduleID}/docker/ls", GetModuleContainers)
	r.Get("/{moduleID}/docker/{containerName}/logs", GetContainerLogs)
	r.Post("/{moduleID}/docker/{containerName}/start", StartModuleContainer)
	r.Post("/{moduleID}/docker/{containerName}/stop", StopModuleContainer)
	r.Post("/{moduleID}/docker/{containerName}/restart", RestartModuleContainer)
    r.Delete("/{moduleID}/docker/{containerName}/delete", DeleteModuleContainer)

    // File system endpoints for module repo
    r.Get("/{moduleID}/fs/tree", GetFsTree)
    r.Get("/{moduleID}/fs/read", ReadFsFile)
    r.Post("/{moduleID}/fs/write", WriteFsFile)
    r.Post("/{moduleID}/fs/rename", RenameFsPath)
    r.Post("/{moduleID}/fs/delete", DeleteFsPath)
    r.Post("/{moduleID}/fs/mkdir", MkdirFsPath)
}
