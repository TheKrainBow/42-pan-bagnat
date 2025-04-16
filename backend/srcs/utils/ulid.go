package utils

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid"
)

type Type string

const (
	User   Type = "user"
	Module Type = "module"
	Role   Type = "role"
)

func GenerateULID(objectType Type) string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return string(objectType) + "_" + id.String()
}

func GetULIDTime(prefixedUlid string) (time.Time, error) {
	parts := strings.Split(prefixedUlid, "_")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid prefixed ULID format")
	}
	id, err := ulid.Parse(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid ulid")
	}
	return time.UnixMilli(int64(id.Time())), nil
}
