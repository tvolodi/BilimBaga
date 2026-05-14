package categories

import (
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrParentNotFound = errors.New("parent not found")
	ErrCycle          = errors.New("cyclic parent reference")
	ErrCategoryInUse  = errors.New("category in use")
	ErrInvalidName    = errors.New("invalid name")
)

// Category is the flat DB row shape.
type Category struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	ParentID  *string   `db:"parent_id"`
	Track     *string   `db:"track"`
	SortOrder int       `db:"sort_order"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Node is the nested JSON representation returned by GET /categories.
type Node struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Track     *string   `json:"track"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Children  []*Node   `json:"children"`
}

type CreateRequest struct {
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	Track     *string `json:"track"`
	SortOrder int     `json:"sort_order"`
}

type UpdateRequest struct {
	Name      *string `json:"name"`
	ParentID  *string `json:"parent_id"`
	Track     *string `json:"track"`
	SortOrder *int    `json:"sort_order"`
	// ClearParent forces parent_id to NULL when true (since omitted vs null is ambiguous in JSON).
	ClearParent bool `json:"clear_parent"`
}
