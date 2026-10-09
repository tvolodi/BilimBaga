// Package roles implements admin-managed custom roles and the permission
// matrix (FR-BB117). The four built-in roles are system roles and immutable.
package roles

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for the roles domain.
var (
	ErrNotFound        = errors.New("role not found")
	ErrNameTaken       = errors.New("role name already taken")
	ErrValidation      = errors.New("validation error")
	ErrSystemImmutable = errors.New("system role is immutable")
	ErrInUse           = errors.New("role is in use")
	ErrCacheReload     = errors.New("rbac cache reload failed")
)

// InUseError carries the number of users still holding the role.
type InUseError struct{ Count int }

func (e *InUseError) Error() string {
	return fmt.Sprintf("role is assigned to %d user(s)", e.Count)
}

// Is makes errors.Is(err, ErrInUse) true for *InUseError.
func (e *InUseError) Is(target error) bool { return target == ErrInUse }

// Role is the API representation of a role.
type Role struct {
	ID          string    `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Description string    `db:"description" json:"description"`
	IsSystem    bool      `db:"is_system"   json:"is_system"`
	UserCount   int       `db:"user_count"  json:"user_count"`
	Permissions []string  `db:"-"           json:"permissions"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// Permission is one entry of the permission catalogue.
type Permission struct {
	ID       string `db:"id"       json:"id"`
	Resource string `db:"resource" json:"resource"`
	Action   string `db:"action"   json:"action"`
}

// Key returns the "resource:action" form of the permission.
func (p Permission) Key() string { return p.Resource + ":" + p.Action }

// RolePermission links a role to a permission (list helper row).
type RolePermission struct {
	RoleID   string `db:"role_id"`
	Resource string `db:"resource"`
	Action   string `db:"action"`
}

// CreateRequest is the body of POST /roles.
type CreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRequest is the body of PUT /roles/{id}. Name is immutable; if present
// it must equal the current name.
type UpdateRequest struct {
	Name        *string  `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateResult is returned by Service.Update so the handler can audit the diff.
type UpdateResult struct {
	Role               *Role
	PermissionsAdded   []string
	PermissionsRemoved []string
}
