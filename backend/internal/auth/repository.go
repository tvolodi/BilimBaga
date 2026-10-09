package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound is returned by repository methods when a record does not exist.
var ErrNotFound = errors.New("not found")

// Repository is the data-access interface for the auth domain.
type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, userID string) (*User, error)
	UpdateFailedAttempts(ctx context.Context, userID string, attempts int) error
	LockAccount(ctx context.Context, userID string, until time.Time) error
	ResetFailedAttempts(ctx context.Context, userID string) error
	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error

	// Password recovery (FR-BB115) — see recovery_repository.go.
	RecoveryRepository
}

type pgRepository struct {
	db *sqlx.DB
}

// NewRepository creates a PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

// GetUserByEmail fetches a user record joined with the role name. The caller passes the
// normalised (trim+lowercase) address; the comparison lowercases the stored column so legacy
// mixed-case rows stay reachable (ISS-164). A functional index on lower(email) is a follow-up.
func (r *pgRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT u.id, u.email, u.password_hash, u.full_name,
		       u.department_id, u.role_id, ro.name AS role_name,
		       u.status, u.force_password_change, u.failed_attempts,
		       u.locked_until, u.created_at, u.updated_at
		FROM   users u
		JOIN   roles ro ON ro.id = u.role_id
		WHERE  lower(u.email) = $1`
	var user User
	if err := r.db.GetContext(ctx, &user, q, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth.GetUserByEmail: %w", err)
	}
	return &user, nil
}

// GetUserByID fetches a user record joined with the role name.
func (r *pgRepository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	const q = `
		SELECT u.id, u.email, u.password_hash, u.full_name,
		       u.department_id, u.role_id, ro.name AS role_name,
		       u.status, u.force_password_change, u.failed_attempts,
		       u.locked_until, u.created_at, u.updated_at
		FROM   users u
		JOIN   roles ro ON ro.id = u.role_id
		WHERE  u.id = $1`
	var user User
	if err := r.db.GetContext(ctx, &user, q, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth.GetUserByID: %w", err)
	}
	return &user, nil
}

// UpdateFailedAttempts sets the failed_attempts counter for a user.
func (r *pgRepository) UpdateFailedAttempts(ctx context.Context, userID string, attempts int) error {
	const q = `UPDATE users SET failed_attempts = $1, updated_at = now() WHERE id = $2`
	if _, err := r.db.ExecContext(ctx, q, attempts, userID); err != nil {
		return fmt.Errorf("auth.UpdateFailedAttempts: %w", err)
	}
	return nil
}

// LockAccount sets locked_until for the given user.
func (r *pgRepository) LockAccount(ctx context.Context, userID string, until time.Time) error {
	const q = `UPDATE users SET locked_until = $1, updated_at = now() WHERE id = $2`
	if _, err := r.db.ExecContext(ctx, q, until, userID); err != nil {
		return fmt.Errorf("auth.LockAccount: %w", err)
	}
	return nil
}

// ResetFailedAttempts clears failed_attempts and locked_until for a user.
func (r *pgRepository) ResetFailedAttempts(ctx context.Context, userID string) error {
	const q = `UPDATE users SET failed_attempts = 0, locked_until = NULL, updated_at = now() WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, userID); err != nil {
		return fmt.Errorf("auth.ResetFailedAttempts: %w", err)
	}
	return nil
}

// CreateRefreshToken inserts a new refresh token record; the DB generates the id.
func (r *pgRepository) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := r.db.ExecContext(ctx, q, token.UserID, token.TokenHash, token.ExpiresAt); err != nil {
		return fmt.Errorf("auth.CreateRefreshToken: %w", err)
	}
	return nil
}

// GetRefreshTokenByHash fetches a refresh token by its SHA-256 hash.
func (r *pgRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM   refresh_tokens
		WHERE  token_hash = $1`
	var token RefreshToken
	if err := r.db.GetContext(ctx, &token, q, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth.GetRefreshTokenByHash: %w", err)
	}
	return &token, nil
}

// RevokeRefreshToken marks a single token as revoked.
func (r *pgRepository) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, tokenID); err != nil {
		return fmt.Errorf("auth.RevokeRefreshToken: %w", err)
	}
	return nil
}

// RevokeAllUserRefreshTokens marks all active refresh tokens for a user as revoked.
func (r *pgRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, userID); err != nil {
		return fmt.Errorf("auth.RevokeAllUserRefreshTokens: %w", err)
	}
	return nil
}

// UpdatePassword replaces a user's password hash and clears force_password_change.
func (r *pgRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	const q = `
		UPDATE users
		SET    password_hash = $1, force_password_change = false, updated_at = now()
		WHERE  id = $2`
	if _, err := r.db.ExecContext(ctx, q, passwordHash, userID); err != nil {
		return fmt.Errorf("auth.UpdatePassword: %w", err)
	}
	return nil
}


