package core

import (
	"backend/database"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type ModulePagination struct {
	OrderBy    []database.ModuleOrder
	Filter     string
	LastModule *database.Module
	Limit      int
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

func EncodeModulePaginationToken(token ModulePagination) (string, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
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
	token, err := EncodeModulePaginationToken(pagination)
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
	if module == nil {
		return Module{}, nil
	}
	dest = DatabaseModuleToModule(*module)
	roles, err := database.GetModuleRoles(moduleID)
	if err != nil {
		return dest, fmt.Errorf("couldn't get module's roles in db: %w", err)
	}
	dest.Roles = DatabaseRolesToRoles(roles)
	return dest, nil
}

func ImportModule(name string, gitURL string, gitBranch string) (Module, error) {
	var dest Module

	// Generate a ULID for the module
	moduleID, err := GenerateULID(ModuleKind)
	if err != nil {
		return Module{}, fmt.Errorf("failed to generate module ID: %w", err)
	}

	// Generate SSH keys
	pubKey, privKey, err := GenerateSSHKeys()
	if err != nil {
		return Module{}, fmt.Errorf("failed to generate SSH keys: %w", err)
	}

	// Prepare module struct
	dest = Module{
		ID:            moduleID,
		Name:          name,
		GitURL:        gitURL,
		GitBranch:     gitBranch,
		SSHPublicKey:  pubKey,
		SSHPrivateKey: privKey,
	}

	// Insert into DB
	if err := database.InsertModule(database.Module{
		ID:            dest.ID,
		Name:          dest.Name,
		GitURL:        dest.GitURL,
		GitBranch:     dest.GitBranch,
		SSHPublicKey:  dest.SSHPublicKey,
		SSHPrivateKey: dest.SSHPrivateKey,
		LastUpdate:    dest.LastUpdate,
	}); err != nil {
		return Module{}, fmt.Errorf("failed to insert module in DB: %w", err)
	}

	return dest, nil
}
