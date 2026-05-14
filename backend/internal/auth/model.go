package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a persisted user record joined with its role name.
type User struct {
	ID                  string     `db:"id"`
	Email               string     `db:"email"`
	PasswordHash        string     `db:"password_hash"`
	FullName            string     `db:"full_name"`
	DepartmentID        *string    `db:"department_id"`
	RoleID              string     `db:"role_id"`
	RoleName            string     `db:"role_name"`
	Status              string     `db:"status"`
	ForcePasswordChange bool       `db:"force_password_change"`
	FailedAttempts      int        `db:"failed_attempts"`
	LockedUntil         *time.Time `db:"locked_until"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

// RefreshToken represents a persisted refresh token record.
type RefreshToken struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned in the JSON body on a successful login.
type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	User        UserInfo `json:"user"`
}

// RefreshResponse is returned in the JSON body on a successful token rotation.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// UserInfo is the minimal user profile included in login responses.
type UserInfo struct {
	ID                  string `json:"id"`
	FullName            string `json:"full_name"`
	Email               string `json:"email"`
	Role                string `json:"role"`
	ForcePasswordChange bool   `json:"force_password_change"`
}

// ChangePasswordRequest is the JSON body for POST /api/v1/auth/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Claims holds the custom JWT payload used by BilimBaga.
type Claims struct {
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	DepartmentID *string `json:"department_id,omitempty"`
	jwt.RegisteredClaims
}

// ServiceError represents a domain-level error with an associated HTTP status code.
// Handlers use errors.As to detect these and write the appropriate HTTP response.
type ServiceError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
