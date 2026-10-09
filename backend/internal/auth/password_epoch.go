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

// PasswordChangedLookup returns when the user's password was last reset (zero time if
// never). It returns ErrNotFound when the user no longer exists.
type PasswordChangedLookup func(ctx context.Context, userID string) (time.Time, error)

const (
	// passwordEpochTTL bounds how stale the per-user cache may be: after a reset, access
	// tokens issued earlier stop working within this window (ISS-105).
	passwordEpochTTL = 10 * time.Second
	// passwordEpochMaxEntries caps the cache; it is cleared when exceeded.
	passwordEpochMaxEntries = 10000
)

type epochEntry struct {
	changedAt time.Time
	fetchedAt time.Time
}

// NewPasswordEpochLookup builds a cached PasswordChangedLookup over fetch. now is
// injectable for tests.
func NewPasswordEpochLookup(fetch PasswordChangedLookup, now func() time.Time) PasswordChangedLookup {
	var mu sync.Mutex
	cache := map[string]epochEntry{}
	return func(ctx context.Context, userID string) (time.Time, error) {
		t := now()
		mu.Lock()
		e, ok := cache[userID]
		mu.Unlock()
		if ok && t.Sub(e.fetchedAt) < passwordEpochTTL {
			return e.changedAt, nil
		}
		changed, err := fetch(ctx, userID)
		if err != nil {
			return time.Time{}, err
		}
		mu.Lock()
		if len(cache) >= passwordEpochMaxEntries {
			cache = map[string]epochEntry{}
		}
		cache[userID] = epochEntry{changedAt: changed, fetchedAt: t}
		mu.Unlock()
		return changed, nil
	}
}

// passwordChangedAtSQL reads the epoch column (migration 032).
const passwordChangedAtSQL = `SELECT password_changed_at FROM users WHERE id = $1`

// NewDBPasswordEpochLookup is the production lookup: users.password_changed_at, cached.
func NewDBPasswordEpochLookup(db *sqlx.DB) PasswordChangedLookup {
	return NewPasswordEpochLookup(func(ctx context.Context, userID string) (time.Time, error) {
		var ts sql.NullTime
		if err := db.GetContext(ctx, &ts, passwordChangedAtSQL, userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return time.Time{}, ErrNotFound
			}
			return time.Time{}, fmt.Errorf("auth.passwordChangedAt: %w", err)
		}
		return ts.Time, nil // zero when NULL
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
