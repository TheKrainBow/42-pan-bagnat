package core

import (
	"backend/database"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type User struct {
	ID        string    `json:"id"`
	FtLogin   string    `json:"ftLogin"`
	FtID      int       `json:"ft_id"`
	FtIsStaff bool      `json:"ft_is_staff"`
	PhotoURL  string    `json:"photo_url"`
	LastSeen  time.Time `json:"last_update"`
	IsStaff   bool      `json:"is_staff"`
	Roles     []Role    `json:"roles"`
}

type UserPatch struct {
	ID        string     `json:"id"`
	FtLogin   *string    `json:"ftLogin"`
	FtID      *int       `json:"ft_id"`
	FtIsStaff *bool      `json:"ft_is_staff"`
	PhotoURL  *string    `json:"photo_url"`
	LastSeen  *time.Time `json:"last_update"`
	IsStaff   *bool      `json:"is_staff"`
	Roles     []string   `json:"roles,omitempty"`
}

type UserPagination struct {
	OrderBy  []database.UserOrder
	Filter   string
	LastUser *database.User
	Limit    int
}

func GenerateUserOrderBy(order string) (dest []database.UserOrder) {
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

		var field database.UserOrderField
		switch arg {
		case "id":
			field = database.UserID
		case "ft_login":
			field = database.UserFtLogin
		case "last_seen":
			field = database.UserLastSeen
		case "ft_is_staff":
			field = database.UserFtIsStaff
		case "ft_id":
			field = database.UserFtID
		default:
			continue
		}

		dest = append(dest, database.UserOrder{
			Field: field,
			Order: direction,
		})
	}
	return dest
}

func EncodeUserPaginationToken(token UserPagination) (string, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func DecodeUserPaginationToken(encoded string) (UserPagination, error) {
	var token UserPagination
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(data, &token)
	return token, err
}

func GetUser(identifier string) (User, error) {
	dbUser, err := database.GetUser(identifier)
	if err != nil {
		return User{}, ErrNotFound
	}

	apiUser := DatabaseUserToUser(*dbUser)

	roles, err := database.GetUserRoles(apiUser.ID)
	if err == nil {
		apiUser.Roles = DatabaseRolesToRoles(roles)
	}
	isAdmin, _ := database.UserHasRoleByID(context.Background(), apiUser.ID, "roles_admin")
	apiUser.IsStaff = isAdmin
	return apiUser, nil
}

func GetUsers(pagination UserPagination) ([]User, string, error) {
	var dest []User
	var realLimit int
	if pagination.Limit > 0 {
		realLimit = pagination.Limit + 1
	} else {
		realLimit = 0
	}

	users, err := database.GetAllUsers(&pagination.OrderBy, pagination.Filter, pagination.LastUser, realLimit)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't get users in db: %w", err)
	}

	hasMore := len(users) > pagination.Limit && pagination.Limit > 0
	if hasMore {
		users = users[:pagination.Limit]
	}

	for _, user := range users {
		apiUser := DatabaseUserToUser(user)
		roles, err := database.GetUserRoles(apiUser.ID)
		if err != nil {
			fmt.Printf("couldn't get roles for user %s: %s\n", apiUser.ID, err.Error())
		} else {
			apiUser.Roles = DatabaseRolesToRoles(roles)
		}
		isAdmin, _ := database.UserHasRoleByID(context.Background(), apiUser.ID, "roles_admin")
		apiUser.IsStaff = isAdmin
		dest = append(dest, apiUser)
	}

	if !hasMore {
		return dest, "", nil
	}

	pagination.LastUser = &users[len(users)-1]

	token, err := EncodeUserPaginationToken(pagination)
	if err != nil {
		return dest, "", fmt.Errorf("couldn't generate next token: %w", err)
	}
	return dest, token, nil
}

// --- small helper: struct/anything → map[string]any for eval ---
func toEvalMap(v any) (map[string]any, error) {
	if v == nil {
		return map[string]any{}, nil
	}
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func ApplyRoleRulesForNewUser(userID string, evalPayload any) error {
	payload, err := toEvalMap(evalPayload)
	if err != nil {
		return fmt.Errorf("normalize eval payload: %w", err)
	}

	roles, err := database.ListRolesWithRules()
	if err != nil {
		return fmt.Errorf("list roles with rules: %w", err)
	}

	var addIDs []string
	for _, r := range roles {
		if r.IsDefault {
			continue // defaults handled separately
		}
		if len(r.Rules) == 0 || string(r.Rules) == "null" {
			continue
		}
		var node any
		if err := json.Unmarshal(r.Rules, &node); err != nil {
			continue // skip invalid rules
		}
		if evalRuleNode(node, payload) {
			addIDs = append(addIDs, r.ID)
		}
	}

	if len(addIDs) == 0 {
		return nil
	}
	// Bulk insert; no deletions for new users.
	return database.BulkAddUserRoles(userID, addIDs)
}

func HandleUser42Connection(ctx context.Context, token *oauth2.Token, meta DeviceMeta) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.intra.42.fr/v2/me", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch user")
	}
	defer resp.Body.Close()

	var intra User42
	if err := json.NewDecoder(resp.Body).Decode(&intra); err != nil {
		return "", fmt.Errorf("couldn't decode user")
	}

	user, err := database.GetUserByLogin(intra.Login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			user = &database.User{
				FtLogin:   intra.Login,
				FtID:      intra.ID,
				FtIsStaff: intra.Staff,
				PhotoURL:  intra.Image.Link,
				LastSeen:  time.Now(),
			}
			if err := database.AddUser(user); err != nil {
				return "", fmt.Errorf("failed to create user: %w", err)
			}
			if err := ApplyRoleRulesForNewUser(user.ID, intra); err != nil {
				fmt.Printf("failed to link default roles: %s\n", err.Error())
			}
			if err := database.LinkDefaultRolesToUser(user.ID); err != nil {
				fmt.Printf("failed to link default roles: %s\n", err.Error())
			}
		} else {
			return "", fmt.Errorf("failed to get user: %w", err)
		}
	}

	user.LastSeen = time.Now()
	_ = database.UpdateUserLastSeen(user.FtLogin, user.LastSeen)

	sessionID, err := EnsureDeviceSession(ctx, user.FtLogin, meta)
	if err != nil {
		return "", fmt.Errorf("failed to ensure session: %w", err)
	}

	return sessionID, nil
}

func ResolveUserIdentifier(identifier string) (string, error) {
	user, err := database.GetUser(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user identifier %q: %w", identifier, err)
	}
	return user.ID, nil
}

func PatchUser(patch UserPatch) (*User, error) {
	if patch.ID == "" {
		return nil, fmt.Errorf("missing user identifier")
	}

	userID, err := ResolveUserIdentifier(patch.ID)
	if err != nil {
		return nil, err
	}

	dbPatch := database.UserPatch{
		ID:        userID,
		FtLogin:   patch.FtLogin,
		FtID:      patch.FtID,
		FtIsStaff: patch.FtIsStaff,
		PhotoURL:  patch.PhotoURL,
		LastSeen:  patch.LastSeen,
	}

	err = database.PatchUser(dbPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch user: %w", err)
	}

	dbUser, err := database.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to patch user: %w", err)
	}

	user := DatabaseUserToUser(*dbUser)
	return &user, nil
}

func TouchUserLastSeen(userID string) {
	now := time.Now().UTC()
	_ = database.UpdateUserLastSeen(userID, now) // ignore error
}
