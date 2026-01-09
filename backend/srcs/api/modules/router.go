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
	r.Get("/{moduleID}/networks", GetModuleNetworks)

	r.Post("/{moduleID}/git/clone", GitClone)
	r.Post("/{moduleID}/git/pull", GitPull)
	r.Post("/{moduleID}/git/update-remote", GitUpdateRemote)
	r.Post("/{moduleID}/git/ssh-key", GitSetSSHKey)
	r.Get("/{moduleID}/git/status", GitStatus)
	r.Post("/{moduleID}/git/fetch", GitFetch)
	r.Post("/{moduleID}/git/add", GitAdd)
	r.Post("/{moduleID}/git/merge/continue", GitMergeContinueHandler)
	r.Post("/{moduleID}/git/merge/abort", GitMergeAbortHandler)
	r.Get("/{moduleID}/git/commits", GitCommits)
	r.Get("/{moduleID}/git/branches", GitBranches)
	r.Get("/{moduleID}/git/behind", GitBehind)
	r.Post("/{moduleID}/git/checkout", GitCheckout)
	r.Post("/{moduleID}/git/branch", GitCreateBranch)
	r.Delete("/{moduleID}/git/branch", GitDeleteBranch)
	r.Get("/{moduleID}/git/commit/current", GitCurrentCommit)
	r.Get("/{moduleID}/git/commit/latest", GitLatestCommit)
	r.Post("/{moduleID}/git/file/checkout", GitFileCheckout)
	r.Post("/{moduleID}/git/file/resolve/ours", GitFileResolveOurs)
	r.Post("/{moduleID}/git/file/resolve/theirs", GitFileResolveTheirs)

	r.Get("/{moduleID}/pages", GetModulePages)
	r.Post("/{moduleID}/pages", PostModulePage)
	r.Patch("/{moduleID}/pages/{pageID}", PatchModulePage)
	r.Delete("/{moduleID}/pages/{pageID}", DeleteModulePage)

	r.Get("/{moduleID}/docker/config", GetModuleConfig)
	r.Post("/{moduleID}/docker/deploy", DeployConfig)

	r.Get("/{moduleID}/docker/ls", GetModuleContainers)
	r.Post("/{moduleID}/docker/compose/deploy", ComposeDeploy)
	r.Post("/{moduleID}/docker/compose/rebuild", ComposeRebuild)
	r.Post("/{moduleID}/docker/compose/down", ComposeDown)
	r.Get("/{moduleID}/docker/{containerName}/logs", GetContainerLogs)
	r.Post("/{moduleID}/docker/{containerName}/start", StartModuleContainer)
	r.Post("/{moduleID}/docker/{containerName}/stop", StopModuleContainer)
	r.Post("/{moduleID}/docker/{containerName}/restart", RestartModuleContainer)
	r.Delete("/{moduleID}/docker/{containerName}/delete", DeleteModuleContainer)

	// File system endpoints for module repo
	r.Get("/{moduleID}/fs/tree", GetFsTree)
	r.Get("/{moduleID}/fs/read", ReadFsFile)
	r.Get("/{moduleID}/fs/root", GetFsRoot)
	r.Post("/{moduleID}/fs/write", WriteFsFile)
	r.Post("/{moduleID}/fs/rename", RenameFsPath)
	r.Post("/{moduleID}/fs/delete", DeleteFsPath)
	r.Post("/{moduleID}/fs/mkdir", MkdirFsPath)

	// Icon management
	r.Post("/{moduleID}/icon/upload", SetModuleIconUpload)
	r.Post("/{moduleID}/icon/url", SetModuleIconFromURL)
	r.Post("/{moduleID}/icon/from-repo", SetModuleIconFromRepo)

	// Page icon management
	r.Post("/{moduleID}/pages/{pageID}/icon/upload", SetPageIconUpload)
	r.Post("/{moduleID}/pages/{pageID}/icon/url", SetPageIconFromURL)
	r.Post("/{moduleID}/pages/{pageID}/icon/from-repo", SetPageIconFromRepo)
	r.Post("/{moduleID}/pages/{pageID}/icon/clear", SetPageIconClear)
}
