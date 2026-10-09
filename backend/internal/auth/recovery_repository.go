package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// RecoveryRepository is the data-access contract for password recovery (FR-BB115).
type RecoveryRepository interface {
	// CountRecentResetTokens returns how many reset tokens were created for the user since the cutoff.
	CountRecentResetTokens(ctx context.Context, userID string, since time.Time) (int, error)
	// CreateResetToken invalidates earlier unused tokens of the user and stores the new token hash.
	CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	// PurgeExpiredResetTokens deletes tokens that expired before the cutoff.
	PurgeExpiredResetTokens(ctx context.Context, cutoff time.Time) error
	// CompleteReset atomically consumes the token, stores the new password hash, clears the
	// lockout and revokes all refresh tokens. Returns ErrNotFound for an unknown, used or
	// expired token, or one that belongs to an inactive user.
	CompleteReset(ctx context.Context, tokenHash, passwordHash string, now time.Time) (string, error)
}

// CountRecentResetTokens counts reset tokens issued for a user since the given time.
func (r *pgRepository) CountRecentResetTokens(ctx context.Context, userID string, since time.Time) (int, error) {
	const q = `SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = $1 AND created_at > $2`
	var n int
	if err := r.db.GetContext(ctx, &n, q, userID, since); err != nil {
		return 0, fmt.Errorf("auth.CountRecentResetTokens: %w", err)
	}
	return n, nil
}

// CreateResetToken marks earlier unused tokens as used and inserts the new token hash.
func (r *pgRepository) CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("auth.CreateResetToken: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const invalidate = `
		UPDATE password_reset_tokens SET used_at = now()
		WHERE  user_id = $1 AND used_at IS NULL`
	if _, err := tx.ExecContext(ctx, invalidate, userID); err != nil {
		return fmt.Errorf("auth.CreateResetToken: invalidate previous: %w", err)
	}
	const insert = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := tx.ExecContext(ctx, insert, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("auth.CreateResetToken: insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("auth.CreateResetToken: commit: %w", err)
	}
	return nil
}

// PurgeExpiredResetTokens deletes reset tokens that expired before the cutoff.
func (r *pgRepository) PurgeExpiredResetTokens(ctx context.Context, cutoff time.Time) error {
	const q = `DELETE FROM password_reset_tokens WHERE expires_at < $1`
	if _, err := r.db.ExecContext(ctx, q, cutoff); err != nil {
		return fmt.Errorf("auth.PurgeExpiredResetTokens: %w", err)
	}
	return nil
}

// CompleteReset runs the whole password reset in one transaction so a token can be used once.
func (r *pgRepository) CompleteReset(ctx context.Context, tokenHash, passwordHash string, now time.Time) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("auth.CompleteReset: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const claim = `
		UPDATE password_reset_tokens
		SET    used_at = $2
		WHERE  token_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING user_id`
	var userID string
	if err := tx.GetContext(ctx, &userID, claim, tokenHash, now); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("auth.CompleteReset: claim token: %w", err)
	}

	const setPassword = `
		UPDATE users
		SET    password_hash = $1, force_password_change = false,
		       failed_attempts = 0, locked_until = NULL, updated_at = now()
		WHERE  id = $2 AND status = 'active'`
	res, err := tx.ExecContext(ctx, setPassword, passwordHash, userID)
	if err != nil {
		return "", fmt.Errorf("auth.CompleteReset: update user: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", ErrNotFound
	}

	const revokeRefresh = `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := tx.ExecContext(ctx, revokeRefresh, userID); err != nil {
		return "", fmt.Errorf("auth.CompleteReset: revoke refresh tokens: %w", err)
	}

	const invalidateOthers = `UPDATE password_reset_tokens SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`
	if _, err := tx.ExecContext(ctx, invalidateOthers, userID, now); err != nil {
		return "", fmt.Errorf("auth.CompleteReset: invalidate other tokens: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("auth.CompleteReset: commit: %w", err)
	}
	return userID, nil
}
