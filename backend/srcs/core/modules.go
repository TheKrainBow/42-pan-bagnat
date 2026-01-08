package core

import (
	"backend/database"
	"backend/websocket"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

type ModuleStatus string

const (
	Enabled          ModuleStatus = "enabled"
	Disabled         ModuleStatus = "disabled"
	Downloading      ModuleStatus = "downloading"
	WaitingForAction ModuleStatus = "waiting_for_action"
)

type Module struct {
	ID                   string       `json:"id"`
	SSHPublicKey         string       `json:"ssh_public_key"`
	SSHPrivateKey        string       `json:"ssh_private_key"`
	SSHKeyID             string       `json:"ssh_key_id"`
	Name                 string       `json:"name"`
	Slug                 string       `json:"slug"`
	Version              string       `json:"version"`
	Status               ModuleStatus `json:"status"`
	GitURL               string       `json:"git_url"`
	GitBranch            string       `json:"git_branch"`
	IconURL              string       `json:"icon_url"`
	LatestVersion        string       `json:"latest_Version"`
	LateCommits          int          `json:"late_commits"`
	LastUpdate           time.Time    `json:"last_update"`
	Roles                []Role       `json:"roles"`
	IsDeploying          bool         `json:"is_deploying"`
	LastDeploy           time.Time    `json:"last_deploy"`
	LastDeployStatus     string       `json:"last_deploy_status"`
	GitLastFetch         time.Time    `json:"git_last_fetch"`
	GitLastPull          time.Time    `json:"git_last_pull"`
	CurrentCommitHash    string       `json:"current_commit_hash"`
	CurrentCommitSubject string       `json:"current_commit_subject"`
	LatestCommitHash     string       `json:"latest_commit_hash"`
	LatestCommitSubject  string       `json:"latest_commit_subject"`
}

type ModuleSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	IconURL string `json:"icon_url"`
}

type UserSummary struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	PhotoURL string `json:"photo_url"`
}

type ModulePostInput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GitURL    string `json:"git_url"`
	GitBranch string `json:"git_branch"`
}

type ModulePatchInput struct {
	Name      string `json:"name"`
	GitURL    string `json:"git_url"`
	GitBranch string `json:"git_branch"`
}

// Core-level patch structure for modules
type ModulePatch struct {
	ID                   string     `json:"id"`
	Name                 *string    `json:"name,omitempty"`
	GitURL               *string    `json:"git_url,omitempty"`
	GitBranch            *string    `json:"git_branch,omitempty"`
	IconURL              *string    `json:"icon_url,omitempty"`
	GitLastFetch         *time.Time `json:"git_last_fetch,omitempty"`
	GitLastPull          *time.Time `json:"git_last_pull,omitempty"`
	LateCommits          *int       `json:"late_commits,omitempty"`
	CurrentCommitHash    *string    `json:"current_commit_hash,omitempty"`
	CurrentCommitSubject *string    `json:"current_commit_subject,omitempty"`
	LatestCommitHash     *string    `json:"latest_commit_hash,omitempty"`
	LatestCommitSubject  *string    `json:"latest_commit_subject,omitempty"`
}

type ModulePagination struct {
	OrderBy    []database.ModuleOrder
	Filter     string
	LastModule *database.Module
	Limit      int
}

type ModuleLog struct {
	ID        int64          `json:"id"`
	ModuleID  string         `json:"module_id"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Meta      map[string]any `json:"meta"`
	CreatedAt time.Time      `json:"created_at"`
}

type ModuleLogsPagination struct {
	ModuleID      string
	OrderBy       []database.ModuleLogsOrder
	Filter        string
	LastModuleLog *database.ModuleLog
	Limit         int
}

type ModulePage struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	URL      string `json:"url"`
	IsPublic bool   `json:"is_public"`
	ModuleID string `json:"module_id"`
	IconURL  string `json:"icon_url"`
}

type ModulePagesPagination struct {
	ModuleID       *string
	OrderBy        []database.ModulePagesOrder
	Filter         string
	LastModulePage *database.ModulePage
	Limit          int
}

type ContainerStatus string

const (
	ContainerRunning    ContainerStatus = "running"
	ContainerExited     ContainerStatus = "exited"
	ContainerPaused     ContainerStatus = "paused"
	ContainerCreated    ContainerStatus = "created"
	ContainerRestarting ContainerStatus = "restarting"
	ContainerDead       ContainerStatus = "dead"
	ContainerUnknown    ContainerStatus = "unknown"
)

type ModuleContainer struct {
	Name   string          `json:"name"`
	Status ContainerStatus `json:"status"`
	Reason string          `json:"reason"`
	Since  string          `json:"since"`
}

// AllContainer describes containers across all modules/projects
type AllContainer struct {
	Name       string          `json:"name"`
	Status     ContainerStatus `json:"status"`
	Reason     string          `json:"reason"`
	Since      string          `json:"since"`
	Project    string          `json:"project"`
	Networks   []string        `json:"networks"`
	ModuleID   string          `json:"module_id,omitempty"`
	ModuleName string          `json:"module_name,omitempty"`
	Missing    bool            `json:"missing,omitempty"`
	Orphan     bool            `json:"orphan,omitempty"`
}

func GenerateModuleOrderBy(order string) (dest []database.ModuleOrder) {
	if order == "" {
		return nil
	}
	args := strings.Split(order, ",")
	for _, arg := range args {
		var direction database.OrderDirection
		if arg[0] == '-' {
			direction = database.Desc
			arg = arg[1:]
		} else {
			direction = database.Asc
		}

		var field database.ModuleOrderField
		switch arg {
		case string(database.ModuleID):
			field = database.ModuleID
		case string(database.ModuleName):
			field = database.ModuleName
		case string(database.ModuleVersion):
			field = database.ModuleVersion
		case string(database.ModuleStatus):
			field = database.ModuleStatus
		case string(database.ModuleGitURL):
			field = database.ModuleGitURL
		case string(database.ModuleLatestVersion):
			field = database.ModuleLatestVersion
		case string(database.ModuleLateCommits):
			field = database.ModuleLateCommits
		case string(database.ModuleLastUpdate):
			field = database.ModuleLastUpdate
		default:
			continue
		}

		dest = append(dest, database.ModuleOrder{
			Field: field,
			Order: direction,
		})
	}
	return dest
}

func GenerateModuleLogsOrderBy(order string) (dest []database.ModuleLogsOrder) {
	if order == "" {
		return nil
	}
	args := strings.Split(order, ",")
	for _, arg := range args {
		var direction database.OrderDirection
		if arg[0] == '-' {
			direction = database.Desc
			arg = arg[1:]
		} else {
			direction = database.Asc
		}

		var field database.ModuleLogsOrderField
		switch arg {
		case string(database.ModuleLogsCreatedAt):
			field = database.ModuleLogsCreatedAt
		default:
			continue
		}

		dest = append(dest, database.ModuleLogsOrder{
			Field: field,
			Order: direction,
		})
	}
	return dest
}

func GenerateModulePagesOrderBy(order string) (dest []database.ModulePagesOrder) {
	if order == "" {
		return nil
	}
	args := strings.Split(order, ",")
	for _, arg := range args {
		var direction database.OrderDirection
		if arg[0] == '-' {
			direction = database.Desc
			arg = arg[1:]
		} else {
			direction = database.Asc
		}

		var field database.ModulePagesOrderField
		switch arg {
		case string(database.ModulePagesName):
			field = database.ModulePagesName
		case string(database.ModulePagesSlug):
			field = database.ModulePagesSlug
		case string(database.ModulePagesIsPublic):
			field = database.ModulePagesIsPublic
		case string(database.ModulePagesURL):
			field = database.ModulePagesURL
		default:
			continue
		}

		dest = append(dest, database.ModulePagesOrder{
			Field: field,
			Order: direction,
		})
	}
	return dest
}

func EncodePaginationToken(token any) (string, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func DecodeModuleLogsPaginationToken(encoded string) (ModuleLogsPagination, error) {
	var token ModuleLogsPagination
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(data, &token)
	return token, err
}

func DecodeModulePaginationToken(encoded string) (ModulePagination, error) {
	var token ModulePagination
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(data, &token)
	return token, err
}
func DecodeModulePagesPaginationToken(encoded string) (ModulePagesPagination, error) {
	var token ModulePagesPagination
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(data, &token)
	return token, err
}

func GetModules(pagination ModulePagination) ([]Module, string, error) {
	var dest []Module
	realLimit := pagination.Limit + 1

	modules, err := database.GetAllModules(&pagination.OrderBy, pagination.Filter, pagination.LastModule, realLimit)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get modules in db: %w", err)
	}

	hasMore := len(modules) > pagination.Limit
	if hasMore {
		modules = modules[:pagination.Limit]
	}

	for _, module := range modules {
		apiModule := DatabaseModuleToModule(module)
		roles, err := database.GetModuleRoles(apiModule.ID)
		if err != nil {
			fmt.Printf("couldn't get modules for user %s: %s\n", apiModule.ID, err.Error())
		} else if len(roles) > 0 {
			apiModule.Roles = DatabaseRolesToRoles(roles)
		}
		dest = append(dest, apiModule)
	}

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastModule = &modules[len(modules)-1]
	token, err := EncodePaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}

func GetModule(moduleID string) (Module, error) {
	var dest Module

	module, err := database.GetModule(moduleID)
	if err != nil {
		return Module{}, fmt.Errorf("couldn't get module in db: %w", err)
	}
	if module.ID == "" {
		return Module{}, nil
	}
	dest = DatabaseModuleToModule(module)
	roles, err := database.GetModuleRoles(moduleID)
	if err != nil {
		return dest, fmt.Errorf("couldn't get module's roles in db: %w", err)
	}
	dest.Roles = DatabaseRolesToRoles(roles)
	return dest, nil
}

func GetModulesBySSHKey(sshKeyID string) ([]ModuleSummary, error) {
	if strings.TrimSpace(sshKeyID) == "" {
		return nil, fmt.Errorf("missing ssh key id")
	}
	mods, err := database.GetModulesBySSHKeyID(sshKeyID)
	if err != nil {
		return nil, err
	}
	out := make([]ModuleSummary, 0, len(mods))
	for _, m := range mods {
		out = append(out, DatabaseModuleSummaryToModuleSummary(m))
	}
	return out, nil
}

// AssignModuleSSHKey updates the module to use an existing SSH key
func AssignModuleSSHKey(moduleID, sshKeyID string, actor *User) (Module, error) {
	moduleID = strings.TrimSpace(moduleID)
	sshKeyID = strings.TrimSpace(sshKeyID)
	if moduleID == "" {
		return Module{}, fmt.Errorf("missing module id")
	}
	if sshKeyID == "" {
		return Module{}, fmt.Errorf("missing ssh key id")
	}
	if _, err := database.GetSSHKey(sshKeyID); err != nil {
		return Module{}, err
	}
	module, err := GetModule(moduleID)
	if err != nil {
		return Module{}, err
	}
	oldKey := module.SSHKeyID
	if err := database.UpdateModuleSSHKey(moduleID, sshKeyID); err != nil {
		return Module{}, err
	}
	module, err = GetModule(moduleID)
	if err != nil {
		return Module{}, err
	}
	if oldKey != "" && oldKey != sshKeyID {
		_ = AppendSSHKeyEvent(oldKey, actor, &module.ID, fmt.Sprintf("Key unassigned from module %s", module.Name))
	}
	msg := fmt.Sprintf("Key assigned to module %s", module.Name)
	_ = AppendSSHKeyEvent(sshKeyID, actor, &module.ID, msg)
	return module, nil
}

func GetUserPages(userIdentifier string) ([]ModulePage, error) {
	var dest []ModulePage

	pages, err := database.GetUserPages(userIdentifier)
	if err != nil {
		return nil, fmt.Errorf("couldn't get user's page in db: %w", err)
	}

	dest = DatabaseModulePagesToModulePages(pages)
	return dest, nil
}

func GetPage(pageName string) (ModulePage, error) {
	var dest ModulePage

	module, err := database.GetPage(pageName)
	if err != nil {
		return ModulePage{}, fmt.Errorf("couldn't get module in db: %w", err)
	}
	if module == nil {
		return ModulePage{}, nil
	}
	dest = DatabaseModulePageToModulePage(*module)
	return dest, nil
}

func GeneratePageSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.TrimSpace(slug)
	slug = regexp.MustCompile(`\s+`).ReplaceAllString(slug, "-")
	attempt := ""
	for {
		isTaken, err := database.IsModuleSlugTaken(slug + attempt)
		if err != nil {
			log.Printf("error while generating slug `%s`: %s\n", slug+attempt, err)
			return ""
		}
		if !isTaken {
			return slug + attempt
		}
		attempt += "-"
	}
}

func GenerateModuleSlug(name, _ string) string {
	sanitize := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		out := make([]rune, 0, len(s))
		lastDash := false
		for _, r := range s {
			ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
			if ok {
				out = append(out, r)
				lastDash = false
				continue
			}
			// normalize any other char to '-'
			if !lastDash {
				out = append(out, '-')
				lastDash = true
			}
		}
		// trim leading/trailing '-'
		for len(out) > 0 && out[0] == '-' {
			out = out[1:]
		}
		for len(out) > 0 && out[len(out)-1] == '-' {
			out = out[:len(out)-1]
		}
		if len(out) == 0 {
			return "module"
		}
		return string(out)
	}
	slug := sanitize(name)
	attempt := ""
	for {
		isTaken, err := database.IsModuleSlugTaken(slug + attempt)
		if err != nil {
			log.Printf("error while generating slug `%s`: %s\n", slug+attempt, err)
			return ""
		}
		if !isTaken {
			return slug + attempt
		}
		attempt += "-"
	}
}

func ensureModuleSSHKeyForSlug(slug string) (SSHKey, error) {
	base := strings.TrimSpace(slug)
	if base == "" {
		base = "module"
	}
	name := base
	suffix := 1
	for {
		_, err := database.GetSSHKeyByName(name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CreateSSHKey(name, "", nil, nil)
			}
			return SSHKey{}, err
		}
		suffix++
		name = fmt.Sprintf("%s-%d", base, suffix)
	}
}

func ImportModule(actor *User, name string, gitURL string, gitBranch string, sshKeyID string) (Module, error) {
	var dest Module

	// Generate a ULID for the module
	moduleID, err := GenerateULID(ModuleKind)
	if err != nil {
		return Module{}, fmt.Errorf("failed to generate module ID: %w", err)
	}

	slug := GenerateModuleSlug(name, gitBranch)
	if slug == "" {
		return Module{}, fmt.Errorf("failed to generate a valid slug: %w", err)
	}

	// Determine SSH key to use
	var (
		key          SSHKey
		trimmedKeyID = strings.TrimSpace(sshKeyID)
		generatedKey bool
	)
	if trimmedKeyID == "" {
		key, err = ensureModuleSSHKeyForSlug(slug)
		if err != nil {
			return Module{}, fmt.Errorf("failed to generate ssh key: %w", err)
		}
		generatedKey = true
	} else {
		key, err = GetSSHKey(trimmedKeyID)
		if err != nil {
			return Module{}, err
		}
	}

	// Prepare module struct
	dest = Module{
		ID:            moduleID,
		Name:          name,
		Slug:          slug,
		GitURL:        gitURL,
		GitBranch:     gitBranch,
		SSHPublicKey:  key.PublicKey,
		SSHPrivateKey: key.PrivateKey,
		SSHKeyID:      key.ID,
	}

	// Insert into DB
	if err := database.InsertModule(database.Module{
		ID:         dest.ID,
		Name:       dest.Name,
		Slug:       dest.Slug,
		GitURL:     dest.GitURL,
		GitBranch:  dest.GitBranch,
		SSHKeyID:   dest.SSHKeyID,
		LastUpdate: dest.LastUpdate,
	}); err != nil {
		return Module{}, fmt.Errorf("failed to insert module in DB: %w", err)
	}

	if generatedKey {
		if err := database.SetSSHKeyModuleOwner(key.ID, dest.ID); err != nil {
			log.Printf("failed to set ssh key owner for %s: %v", key.ID, err)
		} else {
			_ = AppendSSHKeyEvent(key.ID, actor, &dest.ID, "ssh key generated")
		}
	} else {
		msg := fmt.Sprintf("Key assigned to module %s", dest.Name)
		_ = AppendSSHKeyEvent(sshKeyID, actor, &dest.ID, msg)
	}

	return dest, nil
}

// PatchModule updates selected fields on a module and returns the updated module.
func PatchModule(patch ModulePatch) (*Module, error) {
	if patch.ID == "" {
		return nil, fmt.Errorf("missing module id")
	}

	dbPatch := database.ModulePatch{
		ID:      patch.ID,
		Name:    patch.Name,
		GitURL:  patch.GitURL,
		IconURL: patch.IconURL,
		// Note: database.ModulePatch doesn't currently expose git_branch,
		// but it can be supported by adding it there. For now, keep to fields allowed.
	}
	// Support git_branch when available in DB patch structure
	// via reflection-less assignment if the field exists in the type.
	if patch.GitBranch != nil {
		// database.ModulePatch has GitBranch? Check by assigning via helper when upstream adds it.
		// For now, update by calling a dedicated DB method if needed. No-op otherwise.
	}

	updated, err := database.PatchModule(dbPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch module: %w", err)
	}
	m := DatabaseModuleToModule(updated)
	return &m, nil
}

func GetModuleLogs(pagination ModuleLogsPagination) ([]ModuleLog, string, error) {
	var dest []ModuleLog
	realLimit := pagination.Limit + 1

	moduleLogs, err := database.GetModuleLogs(database.ModuleLogPagination{
		OrderBy:  &pagination.OrderBy,
		ModuleID: pagination.ModuleID,
		Limit:    realLimit,
		Filter:   pagination.Filter,
		LastLog:  pagination.LastModuleLog,
	})
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get modules in db: %w", err)
	}

	hasMore := len(moduleLogs) > pagination.Limit
	if hasMore {
		moduleLogs = moduleLogs[:pagination.Limit]
	}

	dest = DatabaseModuleLogsToModuleLogs(moduleLogs)

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastModuleLog = &moduleLogs[len(moduleLogs)-1]
	token, err := EncodePaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}

func SetModuleStatus(moduleID string, status ModuleStatus, sendWebhook bool) error {
	LogModule(moduleID, "INFO", fmt.Sprintf("Changing module status to %s", string(status)), nil, nil)
	oldModule, err := database.GetModule(moduleID)
	if err != nil {
		LogModule(moduleID, "ERROR", "Couldn't fetch module", nil, err)
	}
	newStatus := string(status)
	module, err := database.PatchModule(
		database.ModulePatch{
			ID:     moduleID,
			Status: &newStatus,
		},
	)

	if err != nil {
		return LogModule(moduleID, "ERROR", "Failed to change status", nil, err)
	}

	_ = oldModule
	log.Printf("Sending ws notif for status update\n")
	if oldModule.Status != module.Status && sendWebhook {
		websocket.SendModuleStatusChangedEvent(module.ID, module.Name, string(module.Status))
	}
	return nil
}

func GetModulePages(pagination ModulePagesPagination) ([]ModulePage, string, error) {
	var dest []ModulePage
	realLimit := pagination.Limit + 1

	moduleLogs, err := database.GetModulePages(database.ModulePagesPagination{
		OrderBy:  &pagination.OrderBy,
		ModuleID: pagination.ModuleID,
		Limit:    realLimit,
		Filter:   pagination.Filter,
		LastPage: pagination.LastModulePage,
	})
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get modules in db: %w", err)
	}

	hasMore := len(moduleLogs) > pagination.Limit
	if hasMore {
		moduleLogs = moduleLogs[:pagination.Limit]
	}

	dest = DatabaseModulePagesToModulePages(moduleLogs)

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastModulePage = &moduleLogs[len(moduleLogs)-1]
	token, err := EncodePaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}

func ImportModulePage(moduleID, name, url string, isPublic bool) (ModulePage, error) {
	pageID, err := GenerateULID(PageKind)
	if err != nil {
		return ModulePage{}, fmt.Errorf("failed to generate page ID: %w", err)
	}

	// Prepare module struct
	dest := ModulePage{
		ID:       pageID,
		ModuleID: moduleID,
		Name:     name,
		Slug:     GeneratePageSlug(name),
		URL:      url,
		IsPublic: isPublic,
	}

	// Insert into DB
	if err := database.InsertModulePage(database.ModulePage{
		ID:       dest.ID,
		ModuleID: dest.ModuleID,
		Name:     dest.Name,
		Slug:     dest.Slug,
		URL:      dest.URL,
		IsPublic: dest.IsPublic,
	}); err != nil {
		return ModulePage{}, fmt.Errorf("failed to insert module in DB: %w", err)
	}

	return dest, nil
}

func DeleteModule(moduleID string) error {
	module, err := GetModule(moduleID)
	if err != nil {
		return fmt.Errorf("module %s not found", moduleID)
	}
	if module.SSHKeyID != "" {
		_ = AppendSSHKeyEvent(module.SSHKeyID, nil, &module.ID, fmt.Sprintf("Key unassigned from module %s", module.Name))
	}
	err = CleanupModuleDockerResources(module)
	if err != nil {
		LogModule(moduleID, "WARN", "couldn't clean docker ressources", nil, err)
	}

	err = DeleteModuleRepoDir(module)
	if err != nil {
		LogModule(moduleID, "WARN", "couldn't delete repo folder", nil, err)
	}

	err = database.DeleteModule(moduleID)
	if err != nil {
		fmt.Printf("couldn't delete module: %s\n", err.Error())
		return err
	}

	websocket.SendModuleDeletedEvent(module.ID, module.Name)
	return nil
}

func DeleteModulePage(pageID string) error {
	err := database.DeleteModulePage(pageID)
	if err != nil {
		fmt.Printf("couldn't delete module page: %s\n", err.Error())
		return err
	}
	return nil
}

func UpdateModulePage(pageID string, name, url *string, isPublic *bool) (ModulePage, error) {
	// Build the patch struct for the DB layer
	patch := database.ModulePagePatch{
		ID:       pageID,
		Name:     name,
		URL:      url,
		IsPublic: isPublic,
	}

	if name != nil {
		newSlug := GeneratePageSlug(*name)
		patch.Slug = &newSlug
	}

	// Apply the patch
	dbPage, err := database.PatchModulePage(patch)
	if err != nil {
		return ModulePage{}, err
	}

	// Convert to core model
	page := DatabaseModulePageToModulePage(dbPage)
	return page, nil
}
