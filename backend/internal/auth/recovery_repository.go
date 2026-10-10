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
	// IssueResetToken atomically enforces the per-user throttle and stores a new token.
	// In one transaction it locks the active user's row (FOR UPDATE), counts the tokens
	// created since windowStart and, only when fewer than maxPerWindow exist, invalidates
	// earlier unused tokens and inserts the new hash. Concurrent calls for one user are
	// serialised by the row lock, so the limit cannot be exceeded by a race. Returns false
	// (nil error) when the user is missing/inactive or the throttle is hit.
	IssueResetToken(ctx context.Context, userID, tokenHash string, expiresAt, now, windowStart time.Time, maxPerWindow int) (bool, error)
	// PurgeExpiredResetTokens deletes tokens that expired before the cutoff.
	PurgeExpiredResetTokens(ctx context.Context, cutoff time.Time) error
	// CompleteReset atomically consumes the token, stores the new password hash, clears the
	// lockout, stamps users.password_changed_at (invalidating earlier access tokens) and
	// revokes all refresh tokens. Returns ErrNotFound for an unknown, used or expired
	// token, or one that belongs to an inactive user.
	CompleteReset(ctx context.Context, tokenHash, passwordHash string, now time.Time) (string, error)
}

// SQL for IssueResetToken, kept as constants so tests can assert the locking contract.
const (
	issueLockUserSQL = `SELECT id FROM users WHERE id = $1 AND status = 'active' FOR UPDATE`
	issueCountSQL    = `SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = $1 AND created_at > $2`
	issueInvalidSQL  = `UPDATE password_reset_tokens SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`
	issueInsertSQL   = `INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4)`
)

// IssueResetToken implements the atomic throttle-and-insert (see RecoveryRepository).
// The user row lock is taken before the count so a concurrent request waits, then counts
// with a fresh statement snapshot that includes the first request's committed insert.
func (r *pgRepository) IssueResetToken(ctx context.Context, userID, tokenHash string, expiresAt, now, windowStart time.Time, maxPerWindow int) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false, fmt.Errorf("auth.IssueResetToken: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var locked string
	if err := tx.GetContext(ctx, &locked, issueLockUserSQL, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("auth.IssueResetToken: lock user: %w", err)
	}
	var n int
	if err := tx.GetContext(ctx, &n, issueCountSQL, userID, windowStart); err != nil {
		return false, fmt.Errorf("auth.IssueResetToken: count recent: %w", err)
	}
	if n >= maxPerWindow {
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, issueInvalidSQL, userID, now); err != nil {
		return false, fmt.Errorf("auth.IssueResetToken: invalidate previous: %w", err)
	}
	if _, err := tx.ExecContext(ctx, issueInsertSQL, userID, tokenHash, expiresAt, now); err != nil {
		return false, fmt.Errorf("auth.IssueResetToken: insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("auth.IssueResetToken: commit: %w", err)
	}
	return true, nil
}

// PurgeExpiredResetTokens deletes reset tokens that expired before the cutoff.
func (r *pgRepository) PurgeExpiredResetTokens(ctx context.Context, cutoff time.Time) error {
	const q = `DELETE FROM password_reset_tokens WHERE expires_at < $1`
	if _, err := r.db.ExecContext(ctx, q, cutoff); err != nil {
		return fmt.Errorf("auth.PurgeExpiredResetTokens: %w", err)
	}
	return nil
}

// completeResetSetPasswordSQL stamps password_changed_at so access tokens issued before
// the reset are rejected by Authenticate (ISS-105). The stamp is the start of the next second, so
// a token issued in the reset's own second is rejected too (#439).
const completeResetSetPasswordSQL = `
		UPDATE users
		SET    password_hash = $1, force_password_change = false,
		       failed_attempts = 0, locked_until = NULL, updated_at = now(),
		       password_changed_at = date_trunc('second', now()) + interval '1 second'
		WHERE  id = $2 AND status = 'active'`

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

	res, err := tx.ExecContext(ctx, completeResetSetPasswordSQL, passwordHash, userID)
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
