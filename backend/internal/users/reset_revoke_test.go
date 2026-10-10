package users

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #439: an admin password reset must revoke the user's earlier access tokens, including one issued
// in the same second as the reset, and every refresh token the user holds. #455: the stamp is
// auth.NextPasswordStamp over the application clock and the previous stamp, read under a row lock.

func TestAdminReset_StampsFromTheClockUnderALockAndRevokesRefreshTokens(t *testing.T) {
	f := &recDB{}
	db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })

	appNow := time.Date(2026, 10, 10, 12, 0, 0, 700_000_000, time.UTC)
	require.NoError(t, NewRepository(db).UpdatePassword(context.Background(), uuidA, "new-hash", appNow))

	lock, update, refreshRevoke := -1, -1, -1
	for i, q := range f.queries {
		switch {
		case strings.Contains(q, "FOR UPDATE"):
			lock = i
		case strings.Contains(q, "UPDATE users"):
			update = i
		case strings.Contains(q, "UPDATE refresh_tokens"):
			refreshRevoke = i
		}
	}
	require.GreaterOrEqual(t, lock, 0, "the previous stamp is read under a row lock")
	require.GreaterOrEqual(t, update, 0)
	assert.Less(t, lock, update, "the lock is taken before the stamp is written")
	assert.Contains(t, f.queries[update], "password_changed_at = $3", "the stamp is a parameter, not the database clock")
	assert.NotContains(t, f.queries[update], "password_changed_at = now()", "the stamp is not taken from the database clock")
	assert.NotContains(t, f.queries[update], "date_trunc", "the stamp is not computed in SQL")

	want := time.Date(2026, 10, 10, 12, 0, 1, 0, time.UTC) // the start of the next second after appNow
	assert.Contains(t, f.args[update], driver.Value(want), "the stamp is the next second after the application clock")

	require.GreaterOrEqual(t, refreshRevoke, 0, "every refresh token of the user is revoked")
	assert.Contains(t, f.queries[refreshRevoke], "user_id = $1")
	assert.Contains(t, f.queries[refreshRevoke], "revoked_at IS NULL")
}
