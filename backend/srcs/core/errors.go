package core

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned when trying to create an entity that already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidInput is returned when provided input is invalid or malformed.
	ErrInvalidInput = errors.New("invalid input")

	// ErrConflict is returned when an operation conflicts with the current state.
	ErrConflict = errors.New("conflict")

	// ErrPaginationTokenInvalid is returned when a pagination token is malformed or expired.
	ErrPaginationTokenInvalid = errors.New("invalid pagination token")

	// ErrModuleAlreadyCloned is returned when attempting to clone a module that's already been cloned.
	ErrModuleAlreadyCloned = errors.New("module already cloned")

	// ErrModuleNotCloned is returned when attempting to pull or deploy a module that hasn't been cloned.
	ErrModuleNotCloned = errors.New("module not cloned")

	// ErrPageNotFound is returned when a requested page does not exist.
	ErrPageNotFound = errors.New("page not found")

	// ErrPageAlreadyExists is returned when attempting to create a page that already exists.
	ErrPageAlreadyExists = errors.New("page already exists")

	// ErrRoleNotFound is returned when a requested role does not exist.
	ErrRoleNotFound = errors.New("role not found")

	// ErrRoleAlreadyAssigned is returned when assigning a role that the user/module already has.
	ErrRoleAlreadyAssigned = errors.New("role already assigned")

	// ErrRoleNotAssigned is returned when removing a role that the user/module does not have.
	ErrRoleNotAssigned = errors.New("role not assigned")

	// ErrUserNotFound is returned when a requested user does not exist.
	ErrUserNotFound = errors.New("user not found")
)
