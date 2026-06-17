package database

import (
	"backend/utils"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	OIDCClientTypeConfidential = "confidential"
	OIDCClientTypePublic       = "public"
)

type OIDCClient struct {
	ID                  string         `json:"id" db:"id"`
	ModuleID            string         `json:"module_id" db:"module_id"`
	Name                string         `json:"name" db:"name"`
	ClientID            string         `json:"client_id" db:"client_id"`
	ClientSecretHash    sql.NullString `json:"client_secret_hash" db:"client_secret_hash"`
	ClientType          string         `json:"client_type" db:"client_type"`
	AllowedRedirectURIs []string       `json:"allowed_redirect_uris" db:"allowed_redirect_uris"`
	AllowedScopes       []string       `json:"allowed_scopes" db:"allowed_scopes"`
	Enabled             bool           `json:"enabled" db:"enabled"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
	LastSecretRotatedAt sql.NullTime   `json:"last_secret_rotated_at" db:"last_secret_rotated_at"`
}

type OIDCAuthorizationCode struct {
	ID          string         `json:"id" db:"id"`
	CodeHash    string         `json:"code_hash" db:"code_hash"`
	ClientID    string         `json:"client_id" db:"client_id"`
	ModuleID    string         `json:"module_id" db:"module_id"`
	UserID      string         `json:"user_id" db:"user_id"`
	RedirectURI string         `json:"redirect_uri" db:"redirect_uri"`
	Scopes      []string       `json:"scopes" db:"scopes"`
	Nonce       sql.NullString `json:"nonce" db:"nonce"`
	ExpiresAt   time.Time      `json:"expires_at" db:"expires_at"`
	ConsumedAt  sql.NullTime   `json:"consumed_at" db:"consumed_at"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

type OIDCAccessToken struct {
	ID        string       `json:"id" db:"id"`
	TokenHash string       `json:"token_hash" db:"token_hash"`
	ClientID  string       `json:"client_id" db:"client_id"`
	ModuleID  string       `json:"module_id" db:"module_id"`
	UserID    string       `json:"user_id" db:"user_id"`
	Scopes    []string     `json:"scopes" db:"scopes"`
	ExpiresAt time.Time    `json:"expires_at" db:"expires_at"`
	RevokedAt sql.NullTime `json:"revoked_at" db:"revoked_at"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
}

type OIDCClientPatch struct {
	ID                  string
	Name                *string
	ClientType          *string
	AllowedRedirectURIs *[]string
	AllowedScopes       *[]string
	Enabled             *bool
}

func oidcID(prefix string) string {
	return utils.GenerateULID(utils.Type(prefix))
}

func normalizeOIDCList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func GetOIDCClientByModuleID(moduleID string) (*OIDCClient, error) {
	row := mainDB.QueryRow(`
		SELECT id, module_id, name, client_id, client_secret_hash, client_type, allowed_redirect_uris, allowed_scopes, enabled, created_at, updated_at, last_secret_rotated_at
		FROM oidc_clients
		WHERE module_id = $1
	`, moduleID)

	var client OIDCClient
	if err := row.Scan(
		&client.ID,
		&client.ModuleID,
		&client.Name,
		&client.ClientID,
		&client.ClientSecretHash,
		&client.ClientType,
		pq.Array(&client.AllowedRedirectURIs),
		pq.Array(&client.AllowedScopes),
		&client.Enabled,
		&client.CreatedAt,
		&client.UpdatedAt,
		&client.LastSecretRotatedAt,
	); err != nil {
		return nil, err
	}
	return &client, nil
}

func GetOIDCClientByClientID(clientID string) (*OIDCClient, error) {
	row := mainDB.QueryRow(`
		SELECT id, module_id, name, client_id, client_secret_hash, client_type, allowed_redirect_uris, allowed_scopes, enabled, created_at, updated_at, last_secret_rotated_at
		FROM oidc_clients
		WHERE client_id = $1
	`, clientID)

	var client OIDCClient
	if err := row.Scan(
		&client.ID,
		&client.ModuleID,
		&client.Name,
		&client.ClientID,
		&client.ClientSecretHash,
		&client.ClientType,
		pq.Array(&client.AllowedRedirectURIs),
		pq.Array(&client.AllowedScopes),
		&client.Enabled,
		&client.CreatedAt,
		&client.UpdatedAt,
		&client.LastSecretRotatedAt,
	); err != nil {
		return nil, err
	}
	return &client, nil
}

func ListOIDCClients() ([]OIDCClient, error) {
	rows, err := mainDB.Query(`
		SELECT id, module_id, name, client_id, client_secret_hash, client_type, allowed_redirect_uris, allowed_scopes, enabled, created_at, updated_at, last_secret_rotated_at
		FROM oidc_clients
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OIDCClient
	for rows.Next() {
		var client OIDCClient
		if err := rows.Scan(
			&client.ID,
			&client.ModuleID,
			&client.Name,
			&client.ClientID,
			&client.ClientSecretHash,
			&client.ClientType,
			pq.Array(&client.AllowedRedirectURIs),
			pq.Array(&client.AllowedScopes),
			&client.Enabled,
			&client.CreatedAt,
			&client.UpdatedAt,
			&client.LastSecretRotatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, client)
	}
	return out, rows.Err()
}

func InsertOIDCClient(client OIDCClient) (*OIDCClient, error) {
	if client.ID == "" {
		client.ID = oidcID("oidc-client")
	}
	if client.ClientType == "" {
		client.ClientType = OIDCClientTypeConfidential
	}
	_, err := mainDB.Exec(`
		INSERT INTO oidc_clients (
			id, module_id, name, client_id, client_secret_hash, client_type,
			allowed_redirect_uris, allowed_scopes, enabled, created_at, updated_at, last_secret_rotated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,DEFAULT,DEFAULT,$10)
		ON CONFLICT (module_id) DO NOTHING
	`, client.ID, client.ModuleID, client.Name, client.ClientID, client.ClientSecretHash, client.ClientType, pq.Array(client.AllowedRedirectURIs), pq.Array(client.AllowedScopes), client.Enabled, client.LastSecretRotatedAt)
	if err != nil {
		return nil, err
	}
	return GetOIDCClientByModuleID(client.ModuleID)
}

func UpdateOIDCClient(p OIDCClientPatch) (*OIDCClient, error) {
	if p.ID == "" {
		return nil, errors.New("missing oidc client id")
	}
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	pos := 1
	if p.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", pos))
		args = append(args, *p.Name)
		pos++
	}
	if p.ClientType != nil {
		sets = append(sets, fmt.Sprintf("client_type = $%d", pos))
		args = append(args, *p.ClientType)
		pos++
	}
	if p.AllowedRedirectURIs != nil {
		sets = append(sets, fmt.Sprintf("allowed_redirect_uris = $%d", pos))
		args = append(args, pq.Array(normalizeOIDCList(*p.AllowedRedirectURIs)))
		pos++
	}
	if p.AllowedScopes != nil {
		sets = append(sets, fmt.Sprintf("allowed_scopes = $%d", pos))
		args = append(args, pq.Array(normalizeOIDCList(*p.AllowedScopes)))
		pos++
	}
	if p.Enabled != nil {
		sets = append(sets, fmt.Sprintf("enabled = $%d", pos))
		args = append(args, *p.Enabled)
		pos++
	}
	args = append(args, p.ID)
	_, err := mainDB.Exec(fmt.Sprintf(`UPDATE oidc_clients SET %s WHERE id = $%d`, strings.Join(sets, ", "), pos), args...)
	if err != nil {
		return nil, err
	}
	return getOIDCClientByID(p.ID)
}

func getOIDCClientByID(id string) (*OIDCClient, error) {
	row := mainDB.QueryRow(`
		SELECT id, module_id, name, client_id, client_secret_hash, client_type, allowed_redirect_uris, allowed_scopes, enabled, created_at, updated_at, last_secret_rotated_at
		FROM oidc_clients
		WHERE id = $1
	`, id)
	var client OIDCClient
	if err := row.Scan(
		&client.ID,
		&client.ModuleID,
		&client.Name,
		&client.ClientID,
		&client.ClientSecretHash,
		&client.ClientType,
		pq.Array(&client.AllowedRedirectURIs),
		pq.Array(&client.AllowedScopes),
		&client.Enabled,
		&client.CreatedAt,
		&client.UpdatedAt,
		&client.LastSecretRotatedAt,
	); err != nil {
		return nil, err
	}
	return &client, nil
}

func SetOIDCClientSecretHash(clientID string, hash *string, rotatedAt time.Time) error {
	_, err := mainDB.Exec(`
		UPDATE oidc_clients
		SET client_secret_hash = $1,
		    last_secret_rotated_at = $2,
		    updated_at = NOW()
		WHERE client_id = $3
	`, hash, rotatedAt, clientID)
	return err
}

func AddOIDCClientRedirectURI(clientID, redirectURI string) error {
	_, err := mainDB.Exec(`
		UPDATE oidc_clients
		SET allowed_redirect_uris = (
			SELECT ARRAY(
				SELECT DISTINCT unnest(allowed_redirect_uris || ARRAY[$1])
			)
		),
		updated_at = NOW()
		WHERE client_id = $2
	`, strings.TrimSpace(redirectURI), clientID)
	return err
}

func RemoveOIDCClientRedirectURI(clientID, redirectURI string) error {
	_, err := mainDB.Exec(`
		UPDATE oidc_clients
		SET allowed_redirect_uris = ARRAY(
			SELECT unnest(allowed_redirect_uris)
			EXCEPT
			SELECT $1
		),
		updated_at = NOW()
		WHERE client_id = $2
	`, strings.TrimSpace(redirectURI), clientID)
	return err
}

func CreateOIDCAuthorizationCode(code OIDCAuthorizationCode) error {
	if code.ID == "" {
		code.ID = oidcID("oidc-code")
	}
	_, err := mainDB.Exec(`
		INSERT INTO oidc_authorization_codes (
			id, code_hash, client_id, module_id, user_id, redirect_uri, scopes, nonce, expires_at, consumed_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,DEFAULT)
	`, code.ID, code.CodeHash, code.ClientID, code.ModuleID, code.UserID, code.RedirectURI, pq.Array(code.Scopes), code.Nonce, code.ExpiresAt, code.ConsumedAt)
	return err
}

func GetOIDCAuthorizationCodeByHash(codeHash string) (*OIDCAuthorizationCode, error) {
	row := mainDB.QueryRow(`
		SELECT id, code_hash, client_id, module_id, user_id, redirect_uri, scopes, nonce, expires_at, consumed_at, created_at
		FROM oidc_authorization_codes
		WHERE code_hash = $1
	`, codeHash)
	var code OIDCAuthorizationCode
	if err := row.Scan(
		&code.ID,
		&code.CodeHash,
		&code.ClientID,
		&code.ModuleID,
		&code.UserID,
		&code.RedirectURI,
		pq.Array(&code.Scopes),
		&code.Nonce,
		&code.ExpiresAt,
		&code.ConsumedAt,
		&code.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &code, nil
}

func ConsumeOIDCAuthorizationCode(codeHash string) (*OIDCAuthorizationCode, error) {
	row := mainDB.QueryRow(`
		UPDATE oidc_authorization_codes
		SET consumed_at = NOW()
		WHERE code_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, code_hash, client_id, module_id, user_id, redirect_uri, scopes, nonce, expires_at, consumed_at, created_at
	`, codeHash)
	var code OIDCAuthorizationCode
	if err := row.Scan(
		&code.ID,
		&code.CodeHash,
		&code.ClientID,
		&code.ModuleID,
		&code.UserID,
		&code.RedirectURI,
		pq.Array(&code.Scopes),
		&code.Nonce,
		&code.ExpiresAt,
		&code.ConsumedAt,
		&code.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &code, nil
}

func CreateOIDCAccessToken(token OIDCAccessToken) error {
	if token.ID == "" {
		token.ID = oidcID("oidc-token")
	}
	_, err := mainDB.Exec(`
		INSERT INTO oidc_access_tokens (
			id, token_hash, client_id, module_id, user_id, scopes, expires_at, revoked_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,DEFAULT)
	`, token.ID, token.TokenHash, token.ClientID, token.ModuleID, token.UserID, pq.Array(token.Scopes), token.ExpiresAt, token.RevokedAt)
	return err
}

func GetOIDCAccessTokenByHash(tokenHash string) (*OIDCAccessToken, error) {
	row := mainDB.QueryRow(`
		SELECT id, token_hash, client_id, module_id, user_id, scopes, expires_at, revoked_at, created_at
		FROM oidc_access_tokens
		WHERE token_hash = $1
	`, tokenHash)
	var token OIDCAccessToken
	if err := row.Scan(
		&token.ID,
		&token.TokenHash,
		&token.ClientID,
		&token.ModuleID,
		&token.UserID,
		pq.Array(&token.Scopes),
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &token, nil
}

func GetUserModuleRoles(userID, moduleID string) ([]Role, error) {
	rows, err := mainDB.Query(`
		SELECT DISTINCT r.id, r.name, r.color, r.is_default
		FROM roles r
		JOIN module_roles mr ON mr.role_id = r.id
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND mr.module_id = $2
		ORDER BY r.name ASC
	`, userID, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Color, &role.IsDefault); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func UserCanAccessModule(identifier, moduleID string) (bool, error) {
	var exists bool
	err := mainDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM users u
			JOIN user_roles ur ON ur.user_id = u.id
			JOIN module_roles mr ON mr.role_id = ur.role_id
			WHERE (u.id = $1 OR u.ft_login = $1)
			  AND mr.module_id = $2
		)
	`, identifier, moduleID).Scan(&exists)
	return exists, err
}

func HashOIDCSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
