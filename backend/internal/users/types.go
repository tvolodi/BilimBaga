package users

import (
	"encoding/json"
	"errors"
	"time"
)

// Sentinel errors for the users domain.
var (
	ErrNotFound       = errors.New("user not found")
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrForbidden      = errors.New("forbidden")
	// ErrAmbiguousName is returned when a name matches more than one row (e.g. two
	// departments with the same name under different parents).
	ErrAmbiguousName = errors.New("ambiguous name")
)

// User is the public-facing user record returned by the API.
type User struct {
	ID                  string  `db:"id"                   json:"id"`
	Email               string  `db:"email"                json:"email"`
	FullName            string  `db:"full_name"            json:"full_name"`
	DepartmentID        *string `db:"department_id"        json:"department_id"`
	DepartmentName      *string `db:"department_name"      json:"department_name"`
	RoleID              string  `db:"role_id"              json:"role_id"`
	RoleName            string  `db:"role_name"            json:"role_name"`
	Status              string  `db:"status"               json:"status"`
	ForcePasswordChange bool    `db:"force_password_change" json:"force_password_change"`
	IsLocked            bool    `db:"is_locked"            json:"is_locked"`
	// PreferredLocale is the user's persisted UI/email language (FR-BB116); nil when unset.
	PreferredLocale *string   `db:"preferred_locale"     json:"preferred_locale"`
	CreatedAt       time.Time `db:"created_at"           json:"created_at"`
}

// LocaleUpdate is the outcome of a successful preferred-locale change: the persisted record
// and the value it replaced (used for the audit entry).
type LocaleUpdate struct {
	User     *User
	Previous *string
}

// UpdateMeRequest is the JSON body for PATCH /api/v1/users/me. Only preferred_locale is
// accepted (FR-BB116 AC-2/AC-3). It is kept raw so an absent key can be told apart from an
// explicit null, which clears the preference.
type UpdateMeRequest struct {
	PreferredLocale json.RawMessage `json:"preferred_locale"`
}

// RoleRow is a minimal role record returned by the list-roles endpoint.
type RoleRow struct {
	ID          string `db:"id"          json:"id"`
	Name        string `db:"name"        json:"name"`
	Description string `db:"description" json:"description"`
	IsSystem    bool   `db:"is_system"   json:"is_system"`
	// Assignable reports whether the current caller may assign this role (computed per
	// request by the service; never stored).
	Assignable bool `db:"-" json:"assignable"`
}

// ListFilters holds the query parameters for the paginated list endpoint.
type ListFilters struct {
	DepartmentID *string
	RoleID       *string
	Status       *string
	Page         int
	PerPage      int
}

// ListResult is the paginated response body.
type ListResult struct {
	Items []User `json:"items"`
	Meta  Meta   `json:"meta"`
}

// Meta describes the current pagination state.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// CreateRequest is the JSON body for POST /api/v1/users.
type CreateRequest struct {
	Email        string  `json:"email"`
	FullName     string  `json:"full_name"`
	DepartmentID *string `json:"department_id"`
	RoleID       string  `json:"role_id"`
}

// CreateResponse embeds the created user and the one-time temporary password.
type CreateResponse struct {
	User
	TemporaryPassword string `json:"temporary_password"`
}

// UpdateRequest is the JSON body for PUT /api/v1/users/:id.
type UpdateRequest struct {
	FullName     string  `json:"full_name"`
	DepartmentID *string `json:"department_id"`
	RoleID       string  `json:"role_id"`
}

// ResetPasswordResponse holds the one-time temporary password returned after a reset.
type ResetPasswordResponse struct {
	TemporaryPassword string `json:"temporary_password"`
}

// CSVRow is a parsed, unvalidated row from the bulk import file.
type CSVRow struct {
	RowNum         int
	Email          string
	FullName       string
	DepartmentName string
	RoleName       string
}

// ImportRowResult describes the outcome of a single import row.
type ImportRowResult struct {
	RowNum         int     `json:"row_num"`
	Email          string  `json:"email"`
	FullName       string  `json:"full_name"`
	DepartmentName string  `json:"department_name"`
	RoleName       string  `json:"role_name"`
	Error          *string `json:"error,omitempty"`
}

// ImportPreview is the response for import without ?commit=true — no DB writes.
type ImportPreview struct {
	Valid  []ImportRowResult `json:"valid"`
	Errors []ImportRowResult `json:"errors"`
}
