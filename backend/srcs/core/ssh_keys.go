package core

import (
	"backend/database"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SSHKeyOwner struct {
	Type   string         `json:"type"`
	User   *UserSummary   `json:"user,omitempty"`
	Module *ModuleSummary `json:"module,omitempty"`
}

type SSHKey struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	PublicKey  string       `json:"public_key"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	LastUsedAt time.Time    `json:"last_used_at"`
	PrivateKey string       `json:"-"`
	UsageCount int          `json:"usage_count"`
	CreatedBy  *SSHKeyOwner `json:"created_by,omitempty"`
}

type SSHKeyEvent struct {
	ID          int64          `json:"id"`
	Message     string         `json:"message"`
	CreatedAt   time.Time      `json:"created_at"`
	ActorUser   *UserSummary   `json:"actor_user,omitempty"`
	ActorModule *ModuleSummary `json:"actor_module,omitempty"`
}

func dbSSHKeyToSSHKey(k database.SSHKey) SSHKey {
	return SSHKey{
		ID:         k.ID,
		Name:       k.Name,
		PublicKey:  k.PublicKey,
		CreatedAt:  k.CreatedAt,
		UpdatedAt:  k.UpdatedAt,
		LastUsedAt: k.LastUsedAt,
		PrivateKey: k.PrivateKey,
	}
}

func ownerFromRow(k database.SSHKeyWithUsage) *SSHKeyOwner {
	if k.CreatedByUserID.Valid {
		return &SSHKeyOwner{
			Type: "user",
			User: &UserSummary{
				ID:       k.CreatedByUserID.String,
				Login:    k.CreatedByUserLogin.String,
				PhotoURL: k.CreatedByUserPhoto.String,
			},
		}
	}
	if k.CreatedByModuleID.Valid {
		return &SSHKeyOwner{
			Type:   "module",
			Module: &ModuleSummary{ID: k.CreatedByModuleID.String, Name: k.CreatedByModuleName.String, IconURL: k.CreatedByModuleIcon.String},
		}
	}
	return nil
}

func dbSSHKeyWithUsageToSSHKey(k database.SSHKeyWithUsage) SSHKey {
	res := dbSSHKeyToSSHKey(k.SSHKey)
	res.UsageCount = k.UsageCount
	res.CreatedBy = ownerFromRow(k)
	return res
}

func ListSSHKeys() ([]SSHKey, error) {
	ks, err := database.ListSSHKeysWithUsage()
	if err != nil {
		return nil, err
	}
	out := make([]SSHKey, 0, len(ks))
	for _, k := range ks {
		out = append(out, dbSSHKeyWithUsageToSSHKey(k))
	}
	return out, nil
}

func GetSSHKey(id string) (SSHKey, error) {
	k, err := database.GetSSHKey(id)
	if err != nil {
		return SSHKey{}, err
	}
	return dbSSHKeyToSSHKey(k), nil
}

func ensureSSHKeyName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("ssh key name is required")
	}
	return trimmed, nil
}

func CreateSSHKey(name string, providedPrivate string, ownerUserID *string, ownerModuleID *string) (SSHKey, error) {
	trimmedName, err := ensureSSHKeyName(name)
	if err != nil {
		return SSHKey{}, err
	}

	var pub, priv string
	trimmedPriv := strings.TrimSpace(providedPrivate)
	if trimmedPriv == "" {
		pub, priv, err = GenerateSSHKeys()
		if err != nil {
			return SSHKey{}, err
		}
	} else {
		pub, err = DeriveSSHPublicKey(trimmedPriv)
		if err != nil {
			return SSHKey{}, err
		}
		priv = trimmedPriv
	}

	id, err := GenerateULID(SSHKeyKind)
	if err != nil {
		return SSHKey{}, err
	}

	var ownerUser sql.NullString
	var ownerModule sql.NullString
	if ownerUserID != nil && *ownerUserID != "" {
		ownerUser = sql.NullString{String: *ownerUserID, Valid: true}
	}
	if ownerModuleID != nil && *ownerModuleID != "" {
		ownerModule = sql.NullString{String: *ownerModuleID, Valid: true}
	}

	inserted, err := database.InsertSSHKey(database.SSHKey{
		ID:                id,
		Name:              trimmedName,
		PublicKey:         pub,
		PrivateKey:        priv,
		CreatedByUserID:   ownerUser,
		CreatedByModuleID: ownerModule,
	})
	if err != nil {
		return SSHKey{}, err
	}
	return dbSSHKeyToSSHKey(inserted), nil
}

func RegenerateSSHKey(id string, providedPrivate string, ownerUserID *string) (SSHKey, error) {
	if strings.TrimSpace(id) == "" {
		return SSHKey{}, fmt.Errorf("missing ssh key id")
	}
	var pub, priv string
	var err error
	trimmed := strings.TrimSpace(providedPrivate)
	if trimmed == "" {
		pub, priv, err = GenerateSSHKeys()
		if err != nil {
			return SSHKey{}, err
		}
	} else {
		pub, err = DeriveSSHPublicKey(trimmed)
		if err != nil {
			return SSHKey{}, err
		}
		priv = trimmed
	}
	updated, err := database.UpdateSSHKeyMaterial(id, pub, priv, ownerUserID)
	if err != nil {
		return SSHKey{}, err
	}
	return dbSSHKeyToSSHKey(updated), nil
}

func DeleteSSHKey(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("missing ssh key id")
	}
	return database.DeleteSSHKey(id)
}

func AppendSSHKeyEvent(sshKeyID string, actorUser *User, actorModuleID *string, message string) error {
	if strings.TrimSpace(sshKeyID) == "" || strings.TrimSpace(message) == "" {
		return fmt.Errorf("invalid ssh key event payload")
	}
	var userID sql.NullString
	if actorUser != nil && actorUser.ID != "" {
		userID = sql.NullString{String: actorUser.ID, Valid: true}
	}
	var moduleID sql.NullString
	if actorModuleID != nil && *actorModuleID != "" {
		moduleID = sql.NullString{String: *actorModuleID, Valid: true}
	}
	_, err := database.InsertSSHKeyEvent(database.SSHKeyEvent{
		SSHKeyID:      sshKeyID,
		Message:       message,
		ActorUserID:   userID,
		ActorModuleID: moduleID,
	})
	return err
}

func ListSSHKeyEvents(sshKeyID string, limit int) ([]SSHKeyEvent, error) {
	rows, err := database.ListSSHKeyEvents(sshKeyID, limit)
	if err != nil {
		return nil, err
	}
	var out []SSHKeyEvent
	for _, ev := range rows {
		var user *UserSummary
		if ev.ActorUserID.Valid {
			user = &UserSummary{ID: ev.ActorUserID.String, Login: ev.ActorUserLogin.String, PhotoURL: ev.ActorUserPhoto.String}
		}
		var module *ModuleSummary
		if ev.ActorModuleID.Valid {
			module = &ModuleSummary{ID: ev.ActorModuleID.String, Name: ev.ActorModuleName.String, IconURL: ev.ActorModuleIcon.String}
		}
		out = append(out, SSHKeyEvent{
			ID:          ev.ID,
			Message:     ev.Message,
			CreatedAt:   ev.CreatedAt,
			ActorUser:   user,
			ActorModule: module,
		})
	}
	return out, nil
}
