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
	"strconv"
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
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Slug                    string  `json:"slug"`
	TargetContainer         *string `json:"target_container,omitempty"`
	TargetPort              *int    `json:"target_port,omitempty"`
	IframeOnly              bool    `json:"iframe_only"`
	PageOnly                bool    `json:"page_only"`
	NeedAuth                bool    `json:"need_auth"`
	IsVisible               bool    `json:"is_visible"`
	ModuleID                string  `json:"module_id"`
	IconURL                 string  `json:"icon_url"`
	NetworkName             string  `json:"network_name,omitempty"`
	MaxUploadBodySize       string  `json:"max_upload_body_size"`
	ProxyTimeoutSeconds     int     `json:"proxy_timeout_seconds"`
	RateLimitRPS            int     `json:"rate_limit_rps"`
	RateLimitBurst          int     `json:"rate_limit_burst"`
	DisableRequestBuffering bool    `json:"disable_request_buffering"`
	Roles                   []Role  `json:"roles,omitempty"`
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
	Ports  []ContainerPort `json:"ports,omitempty"`
}

type ContainerPort struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	Scope         string `json:"scope,omitempty"`
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
		case string(database.ModulePagesIframeOnly):
			field = database.ModulePagesIframeOnly
		case string(database.ModulePagesNeedAuth):
			field = database.ModulePagesNeedAuth
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
		dest = append(dest, DatabaseModuleToModule(module))
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
	return dest, nil
}

func GetModuleBySlug(slug string) (Module, error) {
	dbModule, err := database.GetModuleBySlug(slug)
	if err != nil {
		return Module{}, err
	}
	if dbModule.ID == "" {
		return Module{}, fmt.Errorf("module slug %s not found", slug)
	}
	module := DatabaseModuleToModule(dbModule)
	return module, nil
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
	for i := range dest {
		roles, err := database.GetPageRoles(dest[i].ID)
		if err != nil {
			return nil, fmt.Errorf("couldn't get page roles in db: %w", err)
		}
		dest[i].Roles = DatabaseRolesToRoles(roles)
	}
	return dest, nil
}

func UserCanAccessPage(userIdentifier, slug string) (bool, error) {
	if strings.TrimSpace(userIdentifier) == "" || strings.TrimSpace(slug) == "" {
		return false, nil
	}
	return database.UserCanAccessPage(userIdentifier, slug)
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
	roles, err := database.GetPageRoles(dest.ID)
	if err != nil {
		return dest, fmt.Errorf("couldn't get page roles in db: %w", err)
	}
	dest.Roles = DatabaseRolesToRoles(roles)
	return dest, nil
}

func GeneratePageSlug(name string) string {
	slug := normalizePageSlug(name)
	if slug == "" {
		slug = "page"
	}

	attempt := 1
	for {
		candidate := slug
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", slug, attempt)
		}
		isTaken, err := database.IsPageSlugTaken(candidate)
		if err != nil {
			log.Printf("error while generating slug `%s`: %s\n", candidate, err)
			return ""
		}
		if !isTaken {
			return candidate
		}
		attempt++
	}
}

func normalizePageSlug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func ensurePageSlugAvailable(value string, excludePageID string) (string, error) {
	slug := normalizePageSlug(value)
	if slug == "" {
		return "", fmt.Errorf("invalid page slug")
	}

	existing, err := database.GetPage(slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("couldn't validate page slug: %w", err)
	}
	if err == nil && existing != nil && existing.ID != excludePageID {
		return "", fmt.Errorf("page slug already exists")
	}

	return slug, nil
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
	attempt := 1
	for {
		candidate := slug
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", slug, attempt)
		}
		isTaken, err := database.IsModuleSlugTaken(candidate)
		if err != nil {
			log.Printf("error while generating slug `%s`: %s\n", candidate, err)
			return ""
		}
		if !isTaken {
			return candidate
		}
		attempt++
	}
}

func ListModuleNetworks(module Module) ([]string, error) {
	_, networks, err := CollectModuleNetworkAliases(module)
	if err != nil {
		return nil, err
	}
	return networks, nil
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

	if _, err := ensureOIDCClientForModule(dest); err != nil {
		log.Printf("failed to ensure OIDC client for module %s: %v", dest.ID, err)
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
	for i := range dest {
		roles, err := database.GetPageRoles(dest[i].ID)
		if err != nil {
			return nil, "", fmt.Errorf("couldn't get page roles in db: %w", err)
		}
		dest[i].Roles = DatabaseRolesToRoles(roles)
	}

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

func ImportModulePage(moduleID, name string, slug *string, targetContainer *string, targetPort *int, iframeOnly, pageOnly, needAuth, isVisible bool, network string, advanced GatewayAdvancedSettings) (ModulePage, error) {
	pageID, err := GenerateULID(PageKind)
	if err != nil {
		return ModulePage{}, fmt.Errorf("failed to generate page ID: %w", err)
	}

	if err := validatePageMode(iframeOnly, pageOnly); err != nil {
		return ModulePage{}, err
	}

	sanitizedMaxUploadBodySize, err := sanitizeMaxUploadBodySize(advanced.MaxUploadBodySize)
	if err != nil {
		return ModulePage{}, err
	}

	sanitizedProxyTimeout, err := sanitizeProxyTimeoutSeconds(advanced.ProxyTimeoutSeconds)
	if err != nil {
		return ModulePage{}, err
	}

	sanitizedRateLimitRPS, sanitizedRateLimitBurst, err := sanitizeRateLimit(advanced.RateLimitRPS, advanced.RateLimitBurst)
	if err != nil {
		return ModulePage{}, err
	}

	var sanitizedTarget *string
	if targetContainer != nil {
		trimmed := strings.TrimSpace(*targetContainer)
		if trimmed != "" {
			sanitizedTarget = &trimmed
		}
	}

	var sanitizedPort *int
	if targetPort != nil {
		if *targetPort <= 0 || *targetPort > 65535 {
			return ModulePage{}, fmt.Errorf("target port must be between 1 and 65535")
		}
		val := *targetPort
		sanitizedPort = &val
	}

	if (sanitizedTarget == nil) != (sanitizedPort == nil) {
		return ModulePage{}, fmt.Errorf("target_container and target_port must be defined together")
	}

	pageSlug := ""
	if slug != nil && strings.TrimSpace(*slug) != "" {
		pageSlug, err = ensurePageSlugAvailable(*slug, "")
		if err != nil {
			return ModulePage{}, err
		}
	} else {
		pageSlug = GeneratePageSlug(name)
	}

	// Prepare module struct
	dest := ModulePage{
		ID:                      pageID,
		ModuleID:                moduleID,
		Name:                    name,
		Slug:                    pageSlug,
		TargetContainer:         sanitizedTarget,
		TargetPort:              sanitizedPort,
		IframeOnly:              iframeOnly,
		PageOnly:                pageOnly,
		NeedAuth:                needAuth,
		IsVisible:               isVisible,
		NetworkName:             strings.TrimSpace(network),
		MaxUploadBodySize:       sanitizedMaxUploadBodySize,
		ProxyTimeoutSeconds:     sanitizedProxyTimeout,
		RateLimitRPS:            sanitizedRateLimitRPS,
		RateLimitBurst:          sanitizedRateLimitBurst,
		DisableRequestBuffering: advanced.DisableRequestBuffering,
	}

	// Insert into DB
	if err := database.InsertModulePage(database.ModulePage{
		ID:                      dest.ID,
		ModuleID:                dest.ModuleID,
		Name:                    dest.Name,
		Slug:                    dest.Slug,
		TargetContainer:         toNullString(dest.TargetContainer),
		TargetPort:              toNullInt(dest.TargetPort),
		IframeOnly:              dest.IframeOnly,
		PageOnly:                dest.PageOnly,
		NeedAuth:                dest.NeedAuth,
		IsVisible:               dest.IsVisible,
		NetworkName:             dest.NetworkName,
		MaxUploadBodySize:       dest.MaxUploadBodySize,
		ProxyTimeoutSeconds:     dest.ProxyTimeoutSeconds,
		RateLimitRPS:            dest.RateLimitRPS,
		RateLimitBurst:          dest.RateLimitBurst,
		DisableRequestBuffering: dest.DisableRequestBuffering,
	}); err != nil {
		return ModulePage{}, fmt.Errorf("failed to insert module in DB: %w", err)
	}

	if err := database.AssignRoleToPage(RoleIDAdmin, dest.ID); err != nil {
		return ModulePage{}, fmt.Errorf("failed to assign default role to page: %w", err)
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

func UpdateModulePage(pageID string, name *string,
	slug *string, slugSet bool,
	targetContainer *string, targetContainerSet bool,
	targetPort *int, targetPortSet bool,
	iframeOnly *bool,
	pageOnly *bool,
	needAuth *bool,
	isVisible *bool,
	network *string, networkSet bool,
	advanced GatewayAdvancedPatch,
) (ModulePage, error) {
	var sanitizedContainer *string
	if targetContainerSet {
		if targetContainer != nil {
			trimmed := strings.TrimSpace(*targetContainer)
			if trimmed != "" {
				value := trimmed
				sanitizedContainer = &value
			}
		}
	}

	var sanitizedPort *int
	if targetPortSet {
		if targetPort != nil {
			if *targetPort <= 0 || *targetPort > 65535 {
				return ModulePage{}, fmt.Errorf("target port must be between 1 and 65535")
			}
			value := *targetPort
			sanitizedPort = &value
		}
	}

	if targetContainerSet != targetPortSet {
		return ModulePage{}, fmt.Errorf("target_container and target_port must be defined together")
	}

	if targetContainerSet && targetPortSet {
		if (sanitizedContainer == nil) != (sanitizedPort == nil) {
			return ModulePage{}, fmt.Errorf("target_container and target_port must be defined together")
		}
	}

	var sanitizedNetwork *string
	if networkSet {
		if network != nil {
			trimmed := strings.TrimSpace(*network)
			if trimmed != "" {
				value := trimmed
				sanitizedNetwork = &value
			}
		}
	}

	if slugSet {
		sanitizedSlug, err := ensurePageSlugAvailable(strings.TrimSpace(ptrValue(slug)), pageID)
		if err != nil {
			return ModulePage{}, err
		}
		slug = &sanitizedSlug
	}

	if iframeOnly != nil && pageOnly != nil && *iframeOnly && *pageOnly {
		return ModulePage{}, fmt.Errorf("iframe_only and page_only cannot both be true")
	}
	if iframeOnly != nil && *iframeOnly && pageOnly == nil {
		off := false
		pageOnly = &off
	}
	if pageOnly != nil && *pageOnly && iframeOnly == nil {
		off := false
		iframeOnly = &off
	}

	var sanitizedMaxUploadBodySize *string
	if advanced.MaxUploadBodySizeSet {
		value, err := sanitizeMaxUploadBodySize(ptrValue(advanced.MaxUploadBodySize))
		if err != nil {
			return ModulePage{}, err
		}
		sanitizedMaxUploadBodySize = &value
	}

	var sanitizedProxyTimeout *int
	if advanced.ProxyTimeoutSeconds != nil {
		value, err := sanitizeProxyTimeoutSeconds(*advanced.ProxyTimeoutSeconds)
		if err != nil {
			return ModulePage{}, err
		}
		sanitizedProxyTimeout = &value
	}

	var sanitizedRateLimitRPS, sanitizedRateLimitBurst *int
	if advanced.RateLimitRPS != nil || advanced.RateLimitBurst != nil {
		rps := 0
		if advanced.RateLimitRPS != nil {
			rps = *advanced.RateLimitRPS
		}
		burst := 0
		if advanced.RateLimitBurst != nil {
			burst = *advanced.RateLimitBurst
		}
		sanitizedRPS, sanitizedBurst, err := sanitizeRateLimit(rps, burst)
		if err != nil {
			return ModulePage{}, err
		}
		sanitizedRateLimitRPS = &sanitizedRPS
		sanitizedRateLimitBurst = &sanitizedBurst
	}

	patch := database.ModulePagePatch{
		ID:                      pageID,
		Name:                    name,
		Slug:                    slug,
		TargetContainer:         sanitizedContainer,
		TargetContainerSet:      targetContainerSet,
		TargetPort:              sanitizedPort,
		TargetPortSet:           targetPortSet,
		IframeOnly:              iframeOnly,
		PageOnly:                pageOnly,
		NeedAuth:                needAuth,
		IsVisible:               isVisible,
		Network:                 sanitizedNetwork,
		NetworkSet:              networkSet,
		MaxUploadBodySize:       sanitizedMaxUploadBodySize,
		MaxUploadBodySizeSet:    advanced.MaxUploadBodySizeSet,
		ProxyTimeoutSeconds:     sanitizedProxyTimeout,
		RateLimitRPS:            sanitizedRateLimitRPS,
		RateLimitBurst:          sanitizedRateLimitBurst,
		DisableRequestBuffering: advanced.DisableRequestBuffering,
	}

	dbPage, err := database.PatchModulePage(patch)
	if err != nil {
		return ModulePage{}, err
	}

	page := DatabaseModulePageToModulePage(dbPage)
	return page, nil
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func validatePageMode(iframeOnly, pageOnly bool) error {
	if iframeOnly && pageOnly {
		return fmt.Errorf("iframe_only and page_only cannot both be true")
	}
	return nil
}

const defaultMaxUploadBodySize = "1m"

// nginx client_max_body_size accepts a plain byte count or a number
// suffixed with k/m/g (case-insensitive). "0" disables the limit, which
// we intentionally reject here since this setting exists to bound upload
// size and prevent modules from being used as a DoS vector.
var maxUploadBodySizeRe = regexp.MustCompile(`(?i)^[1-9][0-9]*[kmg]?$`)

// sanitizeMaxUploadBodySize validates and normalizes a page's configured
// max upload body size. An empty value falls back to the safe 1m default.
// maxUploadBodySizeCapBytes is the hard ceiling any page's gateway can be
// configured to accept. The edge gateway (nginx/snippets/modules-proxy.conf)
// is deliberately unlimited and delegates all enforcement to the per-page
// gateway, so this cap is the actual, single source of truth for "how big
// can an upload to a module ever be".
const maxUploadBodySizeCapBytes int64 = 2 * 1024 * 1024 * 1024 // 2 GiB

// maxUploadBodySizeToBytes converts an already-validated nginx-style size
// (e.g. "512m") to bytes, using nginx's own base-1024 units.
func maxUploadBodySizeToBytes(value string) int64 {
	unit := value[len(value)-1]
	multiplier := int64(1)
	numPart := value
	switch unit {
	case 'k', 'm', 'g':
		numPart = value[:len(value)-1]
		switch unit {
		case 'k':
			multiplier = 1024
		case 'm':
			multiplier = 1024 * 1024
		case 'g':
			multiplier = 1024 * 1024 * 1024
		}
	}
	n, err := strconv.ParseInt(numPart, 10, 64)
	if err != nil {
		return 0
	}
	return n * multiplier
}

func sanitizeMaxUploadBodySize(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return defaultMaxUploadBodySize, nil
	}
	if !maxUploadBodySizeRe.MatchString(trimmed) {
		return "", fmt.Errorf("max_upload_body_size must be a positive number optionally suffixed with k, m or g (e.g. \"10m\")")
	}
	if maxUploadBodySizeToBytes(trimmed) > maxUploadBodySizeCapBytes {
		return "", fmt.Errorf("max_upload_body_size must not exceed 2g")
	}
	return trimmed, nil
}

const (
	defaultProxyTimeoutSeconds = 60
	minProxyTimeoutSeconds     = 1
	maxProxyTimeoutSeconds     = 600
	maxRateLimitRPS            = 1000
	maxRateLimitBurst          = 10000
)

// GatewayAdvancedSettings groups the tunable nginx gateway settings for a
// module page. It is used when creating a page, where every field always
// has a concrete (possibly default) value.
type GatewayAdvancedSettings struct {
	MaxUploadBodySize       string
	ProxyTimeoutSeconds     int
	RateLimitRPS            int
	RateLimitBurst          int
	DisableRequestBuffering bool
}

// GatewayAdvancedPatch mirrors GatewayAdvancedSettings for partial updates:
// a nil field means "leave unchanged".
type GatewayAdvancedPatch struct {
	MaxUploadBodySize       *string
	MaxUploadBodySizeSet    bool
	ProxyTimeoutSeconds     *int
	RateLimitRPS            *int
	RateLimitBurst          *int
	DisableRequestBuffering *bool
}

// sanitizeProxyTimeoutSeconds validates the read/send timeout applied by a
// page's gateway. A value <= 0 falls back to the nginx-like 60s default.
func sanitizeProxyTimeoutSeconds(raw int) (int, error) {
	if raw <= 0 {
		return defaultProxyTimeoutSeconds, nil
	}
	if raw < minProxyTimeoutSeconds || raw > maxProxyTimeoutSeconds {
		return 0, fmt.Errorf("proxy_timeout_seconds must be between %d and %d", minProxyTimeoutSeconds, maxProxyTimeoutSeconds)
	}
	return raw, nil
}

// sanitizeRateLimit validates the per-client request rate limit for a page's
// gateway. rps <= 0 disables rate limiting entirely (burst is then forced to 0).
func sanitizeRateLimit(rps, burst int) (int, int, error) {
	if rps < 0 {
		return 0, 0, fmt.Errorf("rate_limit_rps must be zero or a positive number")
	}
	if burst < 0 {
		return 0, 0, fmt.Errorf("rate_limit_burst must be zero or a positive number")
	}
	if rps > maxRateLimitRPS {
		return 0, 0, fmt.Errorf("rate_limit_rps must not exceed %d", maxRateLimitRPS)
	}
	if burst > maxRateLimitBurst {
		return 0, 0, fmt.Errorf("rate_limit_burst must not exceed %d", maxRateLimitBurst)
	}
	if rps == 0 {
		return 0, 0, nil
	}
	return rps, burst, nil
}

func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func toNullInt(value *int) sql.NullInt32 {
	if value == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*value), Valid: true}
}

// AnnotateModulePagesWithModuleChecks currently returns pages unchanged.
// Placeholder to preserve previous API surface until health checks are reimplemented.
func AnnotateModulePagesWithModuleChecks(_ Module, pages []ModulePage) []ModulePage {
	return pages
}
