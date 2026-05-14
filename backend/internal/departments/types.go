package departments

import (
	"errors"
	"time"
)

// Sentinel errors for the departments domain.
var (
	ErrNotFound              = errors.New("not found")
	ErrDuplicateName         = errors.New("duplicate name")
	ErrDepartmentNotEmpty    = errors.New("department not empty")
	ErrDepartmentHasChildren = errors.New("department has children")
)

// Department is the flat DB row shape.
type Department struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	ParentID  *string   `db:"parent_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// DepartmentNode is the nested JSON representation returned by the API.
type DepartmentNode struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	ParentID  *string           `json:"parent_id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Children  []*DepartmentNode `json:"children"`
}

// CreateRequest is the JSON body for POST /api/v1/departments.
type CreateRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

// UpdateRequest is the JSON body for PUT /api/v1/departments/:id.
type UpdateRequest struct {
	Name string `json:"name"`
}
