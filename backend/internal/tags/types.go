package tags

import (
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrDuplicate   = errors.New("duplicate name")
	ErrTagInUse    = errors.New("tag in use")
	ErrInvalidName = errors.New("invalid name")
)

const MaxTagNameLength = 64

// Tag is the DB row shape, also serialised directly as JSON.
// UsageCount is populated by GetAll (LEFT JOIN question_tags) and is zero for
// rows fetched by GetByID / Create / Update since callers there don't need it.
type Tag struct {
	ID         string    `db:"id"          json:"id"`
	Name       string    `db:"name"        json:"name"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UsageCount int       `db:"usage_count" json:"usage_count"`
}

type CreateRequest struct {
	Name string `json:"name"`
}

type UpdateRequest struct {
	Name string `json:"name"`
}
