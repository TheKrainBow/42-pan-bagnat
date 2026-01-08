package database

import (
	"database/sql"
	"fmt"
	"time"
)

type SSHKey struct {
	ID                string         `json:"id" db:"id"`
	Name              string         `json:"name" db:"name"`
	PublicKey         string         `json:"public_key" db:"public_key"`
	PrivateKey        string         `json:"private_key" db:"private_key"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
	LastUsedAt        time.Time      `json:"last_used_at" db:"last_used_at"`
	CreatedByUserID   sql.NullString `json:"created_by_user_id" db:"created_by_user_id"`
	CreatedByModuleID sql.NullString `json:"created_by_module_id" db:"created_by_module_id"`
}

type SSHKeyWithUsage struct {
	SSHKey
	UsageCount          int
	CreatedByUserLogin  sql.NullString
	CreatedByUserPhoto  sql.NullString
	CreatedByModuleName sql.NullString
	CreatedByModuleIcon sql.NullString
}

type SSHKeyEvent struct {
	ID              int64  `json:"id" db:"id"`
	SSHKeyID        string `json:"ssh_key_id" db:"ssh_key_id"`
	Message         string `json:"message" db:"message"`
	ActorUserID     sql.NullString
	ActorUserLogin  sql.NullString
	ActorUserPhoto  sql.NullString
	ActorModuleID   sql.NullString
	ActorModuleName sql.NullString
	ActorModuleIcon sql.NullString
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

func ListSSHKeysWithUsage() ([]SSHKeyWithUsage, error) {
	rows, err := mainDB.Query(`
        SELECT k.id,
               k.name,
               k.public_key,
               k.private_key,
               k.created_at,
               k.updated_at,
               k.last_used_at,
               k.created_by_user_id,
               k.created_by_module_id,
               COUNT(m.id) AS usage_count,
               u.ft_login AS created_by_user_login,
               u.photo_url AS created_by_user_photo,
               cm.name AS created_by_module_name,
               cm.icon_url AS created_by_module_icon
          FROM ssh_keys k
          LEFT JOIN modules m ON m.ssh_key_id = k.id
          LEFT JOIN users u ON u.id = k.created_by_user_id
          LEFT JOIN modules cm ON cm.id = k.created_by_module_id
      GROUP BY k.id, k.name, k.public_key, k.private_key, k.created_at, k.updated_at,
               k.created_by_user_id, k.created_by_module_id,
               u.ft_login, u.photo_url, cm.name, cm.icon_url
      ORDER BY k.name ASC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SSHKeyWithUsage
	for rows.Next() {
		var k SSHKeyWithUsage
		if err := rows.Scan(
			&k.ID,
			&k.Name,
			&k.PublicKey,
			&k.PrivateKey,
			&k.CreatedAt,
			&k.UpdatedAt,
			&k.LastUsedAt,
			&k.CreatedByUserID,
			&k.CreatedByModuleID,
			&k.UsageCount,
			&k.CreatedByUserLogin,
			&k.CreatedByUserPhoto,
			&k.CreatedByModuleName,
			&k.CreatedByModuleIcon,
		); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, nil
}

func GetSSHKey(id string) (SSHKey, error) {
	var k SSHKey
	err := mainDB.Get(&k, `
        SELECT id, name, public_key, private_key, created_at, updated_at, last_used_at,
               created_by_user_id, created_by_module_id
          FROM ssh_keys
         WHERE id = $1
    `, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return SSHKey{}, fmt.Errorf("ssh key %s not found", id)
		}
		return SSHKey{}, err
	}
	return k, nil
}

func GetSSHKeyByName(name string) (SSHKey, error) {
	var k SSHKey
	err := mainDB.Get(&k, `
        SELECT id, name, public_key, private_key, created_at, updated_at, last_used_at,
               created_by_user_id, created_by_module_id
          FROM ssh_keys
         WHERE name = $1
    `, name)
	if err != nil {
		return SSHKey{}, err
	}
	return k, nil
}

func InsertSSHKey(k SSHKey) (SSHKey, error) {
	row := mainDB.QueryRow(`
        INSERT INTO ssh_keys (id, name, public_key, private_key, created_by_user_id, created_by_module_id)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, name, public_key, private_key, created_at, updated_at, last_used_at,
                  created_by_user_id, created_by_module_id
	`, k.ID, k.Name, k.PublicKey, k.PrivateKey, k.CreatedByUserID, k.CreatedByModuleID)
	if err := row.Scan(&k.ID, &k.Name, &k.PublicKey, &k.PrivateKey, &k.CreatedAt, &k.UpdatedAt, &k.LastUsedAt, &k.CreatedByUserID, &k.CreatedByModuleID); err != nil {
		return SSHKey{}, err
	}
	return k, nil
}

func UpdateSSHKeyMaterial(id, publicKey, privateKey string, newOwnerUserID *string) (SSHKey, error) {
	var ownerUser sql.NullString
	if newOwnerUserID != nil && *newOwnerUserID != "" {
		ownerUser = sql.NullString{String: *newOwnerUserID, Valid: true}
	}
	// when user takes ownership, module owner should be cleared
	row := mainDB.QueryRow(`
        UPDATE ssh_keys
           SET public_key = $2,
               private_key = $3,
               created_by_user_id = $4,
               created_by_module_id = NULL,
               updated_at = NOW(),
               last_used_at = NOW()
         WHERE id = $1
     RETURNING id, name, public_key, private_key, created_at, updated_at, last_used_at,
               created_by_user_id, created_by_module_id
    `, id, publicKey, privateKey, ownerUser)
	var k SSHKey
	if err := row.Scan(&k.ID, &k.Name, &k.PublicKey, &k.PrivateKey, &k.CreatedAt, &k.UpdatedAt, &k.LastUsedAt, &k.CreatedByUserID, &k.CreatedByModuleID); err != nil {
		return SSHKey{}, err
	}
	return k, nil
}

func DeleteSSHKey(id string) error {
	_, err := mainDB.Exec(`DELETE FROM ssh_keys WHERE id = $1`, id)
	return err
}

func SetSSHKeyModuleOwner(keyID, moduleID string) error {
	_, err := mainDB.Exec(`
		UPDATE ssh_keys
		   SET created_by_module_id = $2,
		       created_by_user_id = NULL
		 WHERE id = $1
	`, keyID, moduleID)
	return err
}

func InsertSSHKeyEvent(event SSHKeyEvent) (SSHKeyEvent, error) {
	row := mainDB.QueryRow(`
        INSERT INTO ssh_key_events (ssh_key_id, message, actor_user_id, actor_module_id)
        VALUES ($1, $2, $3, $4)
        RETURNING id, ssh_key_id, message, actor_user_id, actor_module_id, created_at
    `, event.SSHKeyID, event.Message, event.ActorUserID, event.ActorModuleID)
	if err := row.Scan(&event.ID, &event.SSHKeyID, &event.Message, &event.ActorUserID, &event.ActorModuleID, &event.CreatedAt); err != nil {
		return SSHKeyEvent{}, err
	}
	_, _ = mainDB.Exec(`UPDATE ssh_keys SET last_used_at = NOW() WHERE id = $1`, event.SSHKeyID)
	return event, nil
}

func ListSSHKeyEvents(sshKeyID string, limit int) ([]SSHKeyEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := mainDB.Query(`
        SELECT e.id, e.ssh_key_id, e.message,
               e.actor_user_id, u.ft_login, u.photo_url,
               e.actor_module_id, m.name, m.icon_url,
               e.created_at
          FROM ssh_key_events e
          LEFT JOIN users u ON u.id = e.actor_user_id
          LEFT JOIN modules m ON m.id = e.actor_module_id
         WHERE e.ssh_key_id = $1
         ORDER BY e.created_at DESC
         LIMIT $2
    `, sshKeyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SSHKeyEvent
	for rows.Next() {
		var ev SSHKeyEvent
		if err := rows.Scan(
			&ev.ID,
			&ev.SSHKeyID,
			&ev.Message,
			&ev.ActorUserID,
			&ev.ActorUserLogin,
			&ev.ActorUserPhoto,
			&ev.ActorModuleID,
			&ev.ActorModuleName,
			&ev.ActorModuleIcon,
			&ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}
