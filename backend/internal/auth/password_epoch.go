package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// AccountState is the per-user session-gating state read on every authenticated request:
// the password epoch (ISS-105) and the forced-password-change flag (ISS-160).
type AccountState struct {
	// PasswordChangedAt is when the password was last reset (zero if never).
	PasswordChangedAt time.Time
	// ForcePasswordChange mirrors users.force_password_change.
	ForcePasswordChange bool
	// Status, RoleName and DepartmentID mirror users.status, roles.name and
	// users.department_id ("" when NULL). When Status is non-empty (the production lookup
	// always sets it) Authenticate rejects the token unless the account is active and the
	// token's role/department claims equal these values, so a demotion, move or
	// deactivation takes effect within the cache TTL instead of the JWT lifetime (ISS-240).
	Status       string
	RoleName     string
	DepartmentID string
}

// AccountStateLookup returns the user's AccountState. It returns ErrNotFound when the
// user no longer exists.
type AccountStateLookup func(ctx context.Context, userID string) (AccountState, error)

const (
	// accountStateTTL bounds how stale the per-user cache may be: after an admin reset,
	// access tokens issued earlier stop working within this window (ISS-105). Password
	// changes made through this process invalidate the entry explicitly (ISS-160).
	accountStateTTL = 10 * time.Second
	// accountStateMaxEntries caps the cache; it is cleared when exceeded.
	accountStateMaxEntries = 10000
)

type accountEntry struct {
	state     AccountState
	fetchedAt time.Time
}

// AccountStateCache is a short-lived per-user cache over an AccountStateLookup.
type AccountStateCache struct {
	fetch AccountStateLookup
	now   func() time.Time
	mu    sync.Mutex
	cache map[string]accountEntry
	gen   uint64 // bumped by Invalidate; a fetch that raced with it is not stored
}

// NewAccountStateCache builds a cache over fetch. now is injectable for tests.
func NewAccountStateCache(fetch AccountStateLookup, now func() time.Time) *AccountStateCache {
	return &AccountStateCache{fetch: fetch, now: now, cache: map[string]accountEntry{}}
}

// Lookup is an AccountStateLookup serving from the cache within the TTL. Errors are
// never cached.
func (c *AccountStateCache) Lookup(ctx context.Context, userID string) (AccountState, error) {
	t := c.now()
	c.mu.Lock()
	e, ok := c.cache[userID]
	gen := c.gen
	c.mu.Unlock()
	if ok && t.Sub(e.fetchedAt) < accountStateTTL {
		return e.state, nil
	}
	st, err := c.fetch(ctx, userID)
	if err != nil {
		return AccountState{}, err
	}
	c.mu.Lock()
	if gen == c.gen {
		if len(c.cache) >= accountStateMaxEntries {
			c.cache = map[string]accountEntry{}
		}
		c.cache[userID] = accountEntry{state: st, fetchedAt: t}
	}
	c.mu.Unlock()
	return st, nil
}

// Invalidate drops the user's cached entry so the next request re-reads the database.
// It is called right after a password change/reset so the user is not blocked (or let
// through) by a stale force_password_change value.
func (c *AccountStateCache) Invalidate(userID string) {
	c.mu.Lock()
	delete(c.cache, userID)
	c.gen++
	c.mu.Unlock()
}

// accountStateSQL reads the epoch column (migration 032) and the force flag in one query.
const accountStateSQL = `SELECT u.password_changed_at, u.force_password_change, u.status,
       ro.name AS role_name, u.department_id
FROM users u JOIN roles ro ON ro.id = u.role_id WHERE u.id = $1`

// NewDBAccountStateCache is the production cache over users.password_changed_at and
// users.force_password_change.
func NewDBAccountStateCache(db *sqlx.DB) *AccountStateCache {
	return NewAccountStateCache(func(ctx context.Context, userID string) (AccountState, error) {
		var row struct {
			ChangedAt sql.NullTime   `db:"password_changed_at"`
			Force     bool           `db:"force_password_change"`
			Status    string         `db:"status"`
			RoleName  string         `db:"role_name"`
			DeptID    sql.NullString `db:"department_id"`
		}
		if err := db.GetContext(ctx, &row, accountStateSQL, userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return AccountState{}, ErrNotFound
			}
			return AccountState{}, fmt.Errorf("auth.accountState: %w", err)
		}
		return AccountState{PasswordChangedAt: row.ChangedAt.Time, ForcePasswordChange: row.Force,
			Status: row.Status, RoleName: row.RoleName, DepartmentID: row.DeptID.String}, nil
	}, time.Now)
}

// tokenPredatesPasswordChange reports whether a token issued at iat (unix seconds) was
// issued strictly before the password change (compared at second resolution).
func tokenPredatesPasswordChange(iat int64, changedAt time.Time) bool {
	if changedAt.IsZero() {
		return false
	}
	return iat < changedAt.Unix()
}
