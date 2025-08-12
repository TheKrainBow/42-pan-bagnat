package database

import "errors"

type OrderDirection string

const (
	Desc OrderDirection = "DESC"
	Asc  OrderDirection = "ASC"
)

var (
	// Public sentinel used by handlers.
	ErrNotFound = errors.New("not found")
)
