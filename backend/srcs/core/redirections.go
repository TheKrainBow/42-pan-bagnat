package core

import (
	"backend/database"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
)

type Redirection struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	TargetURL      string `json:"target_url"`
	IconURL        string `json:"icon_url"`
	NeedAuth       bool   `json:"need_auth"`
	IsVisible      bool   `json:"is_visible"`
	Roles          []Role `json:"roles,omitempty"`
	ForbiddenRoles []Role `json:"forbidden_roles,omitempty"`
}

func DatabaseRedirectionToRedirection(dbRedirection database.Redirection) Redirection {
	return Redirection{
		ID:        dbRedirection.ID,
		Name:      dbRedirection.Name,
		Slug:      dbRedirection.Slug,
		TargetURL: dbRedirection.TargetURL,
		IconURL:   dbRedirection.IconURL,
		NeedAuth:  dbRedirection.NeedAuth,
		IsVisible: dbRedirection.IsVisible,
	}
}

func GenerateRedirectionSlug(name string) string {
	slug := normalizePageSlug(name)
	if slug == "" {
		slug = "redirection"
	}

	attempt := 1
	for {
		candidate := slug
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", slug, attempt)
		}
		isTaken, err := database.IsRedirectionSlugTaken(candidate)
		if err != nil {
			log.Printf("error while generating redirection slug `%s`: %s\n", candidate, err)
			return ""
		}
		if !isTaken {
			return candidate
		}
		attempt++
	}
}

func ensureRedirectionSlugAvailable(value string, excludeRedirectionID string) (string, error) {
	slug := normalizePageSlug(value)
	if slug == "" {
		return "", fmt.Errorf("invalid redirection slug")
	}

	existing, err := database.GetRedirection(slug)
	if err != nil {
		return "", fmt.Errorf("couldn't validate redirection slug: %w", err)
	}
	if existing != nil && existing.ID != excludeRedirectionID {
		return "", fmt.Errorf("redirection slug already exists")
	}

	return slug, nil
}

func validateTargetURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("target url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("target url must be a valid http(s) url")
	}
	return trimmed, nil
}

func CreateRedirection(name string, slug *string, targetURL string, needAuth, isVisible bool) (Redirection, error) {
	redirectionID, err := GenerateULID(RedirectionKind)
	if err != nil {
		return Redirection{}, fmt.Errorf("failed to generate redirection ID: %w", err)
	}

	sanitizedTargetURL, err := validateTargetURL(targetURL)
	if err != nil {
		return Redirection{}, err
	}

	redirectionSlug := ""
	if slug != nil && strings.TrimSpace(*slug) != "" {
		redirectionSlug, err = ensureRedirectionSlugAvailable(*slug, "")
		if err != nil {
			return Redirection{}, err
		}
	} else {
		redirectionSlug = GenerateRedirectionSlug(name)
	}

	dest := Redirection{
		ID:        redirectionID,
		Name:      name,
		Slug:      redirectionSlug,
		TargetURL: sanitizedTargetURL,
		NeedAuth:  needAuth,
		IsVisible: isVisible,
	}

	if err := database.InsertRedirection(database.Redirection{
		ID:        dest.ID,
		Name:      dest.Name,
		Slug:      dest.Slug,
		TargetURL: dest.TargetURL,
		NeedAuth:  dest.NeedAuth,
		IsVisible: dest.IsVisible,
	}); err != nil {
		return Redirection{}, fmt.Errorf("couldn't insert redirection in db: %w", err)
	}

	return dest, nil
}

func GetRedirectionBySlug(slug string) (Redirection, error) {
	dbRedirection, err := database.GetRedirection(slug)
	if err != nil {
		return Redirection{}, fmt.Errorf("couldn't get redirection in db: %w", err)
	}
	if dbRedirection == nil {
		return Redirection{}, sql.ErrNoRows
	}
	return hydrateRedirectionRoles(DatabaseRedirectionToRedirection(*dbRedirection))
}

func GetRedirectionByID(redirectionID string) (Redirection, error) {
	dbRedirection, err := database.GetRedirectionByID(redirectionID)
	if err != nil {
		return Redirection{}, fmt.Errorf("couldn't get redirection in db: %w", err)
	}
	if dbRedirection == nil {
		return Redirection{}, sql.ErrNoRows
	}
	return hydrateRedirectionRoles(DatabaseRedirectionToRedirection(*dbRedirection))
}

func ListRedirections() ([]Redirection, error) {
	dbRedirections, err := database.GetAllRedirections()
	if err != nil {
		return nil, fmt.Errorf("couldn't list redirections in db: %w", err)
	}

	dest := make([]Redirection, 0, len(dbRedirections))
	for _, dbRedirection := range dbRedirections {
		redirection, err := hydrateRedirectionRoles(DatabaseRedirectionToRedirection(dbRedirection))
		if err != nil {
			return nil, err
		}
		dest = append(dest, redirection)
	}
	return dest, nil
}

// GetUserRedirections returns all redirections accessible to a specific user,
// mirroring GetUserPages for module pages.
func GetUserRedirections(userIdentifier string) ([]Redirection, error) {
	dbRedirections, err := database.GetUserRedirections(userIdentifier)
	if err != nil {
		return nil, fmt.Errorf("couldn't get user's redirections in db: %w", err)
	}

	dest := make([]Redirection, 0, len(dbRedirections))
	for _, dbRedirection := range dbRedirections {
		redirection, err := hydrateRedirectionRoles(DatabaseRedirectionToRedirection(dbRedirection))
		if err != nil {
			return nil, err
		}
		dest = append(dest, redirection)
	}
	return dest, nil
}

func hydrateRedirectionRoles(dest Redirection) (Redirection, error) {
	roles, err := database.GetRedirectionRoles(dest.ID)
	if err != nil {
		return dest, fmt.Errorf("couldn't get redirection roles in db: %w", err)
	}
	dest.Roles = DatabaseRolesToRoles(roles)

	forbiddenRoles, err := database.GetRedirectionForbiddenRoles(dest.ID)
	if err != nil {
		return dest, fmt.Errorf("couldn't get redirection forbidden roles in db: %w", err)
	}
	dest.ForbiddenRoles = DatabaseRolesToRoles(forbiddenRoles)
	return dest, nil
}

func UpdateRedirection(redirectionID string, name, slug, targetURL *string, needAuth, isVisible *bool) (Redirection, error) {
	patch := database.RedirectionPatch{
		ID:        redirectionID,
		Name:      name,
		NeedAuth:  needAuth,
		IsVisible: isVisible,
	}

	if slug != nil {
		sanitizedSlug, err := ensureRedirectionSlugAvailable(*slug, redirectionID)
		if err != nil {
			return Redirection{}, err
		}
		patch.Slug = &sanitizedSlug
	}

	if targetURL != nil {
		sanitizedTargetURL, err := validateTargetURL(*targetURL)
		if err != nil {
			return Redirection{}, err
		}
		patch.TargetURL = &sanitizedTargetURL
	}

	dbRedirection, err := database.PatchRedirection(patch)
	if err != nil {
		return Redirection{}, err
	}
	return hydrateRedirectionRoles(DatabaseRedirectionToRedirection(dbRedirection))
}

func DeleteRedirectionByID(redirectionID string) error {
	if _, err := database.GetRedirectionByID(redirectionID); err != nil {
		return err
	}
	return database.DeleteRedirection(redirectionID)
}

func AssignRoleToRedirection(redirectionID, roleID string) error {
	return database.AssignRoleToRedirection(roleID, redirectionID)
}

func RemoveRoleFromRedirection(redirectionID, roleID string) error {
	return database.RemoveRoleFromRedirection(roleID, redirectionID)
}

func AssignForbiddenRoleToRedirection(redirectionID, roleID string) error {
	return database.AssignForbiddenRoleToRedirection(roleID, redirectionID)
}

func RemoveForbiddenRoleFromRedirection(redirectionID, roleID string) error {
	return database.RemoveForbiddenRoleFromRedirection(roleID, redirectionID)
}
