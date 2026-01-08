package core

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// EntityKind represents valid types of entities
type EntityKind string

const (
	UserKind   EntityKind = "user"
	RoleKind   EntityKind = "role"
	ModuleKind EntityKind = "module"
	PageKind   EntityKind = "page"
	SSHKeyKind EntityKind = "ssh-key"
)

func GenerateULID(kind EntityKind) (string, error) {
	switch kind {
	case UserKind, RoleKind, ModuleKind, PageKind, SSHKeyKind:
		// valid
	default:
		return "", fmt.Errorf("invalid entity kind: %s", kind)
	}

	t := time.Now()
	entropy := ulid.Monotonic(rand.Reader, 0)
	id, err := ulid.New(ulid.Timestamp(t), entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}

	return fmt.Sprintf("%s_%s", kind, id.String()), nil
}
