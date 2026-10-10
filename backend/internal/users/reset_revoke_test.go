package users

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #439: an admin password reset must revoke the user's earlier access tokens, including one issued
// in the same second as the reset, and every refresh token the user holds. The access-token epoch
// compares iat with floor(password_changed_at), so the reset stamps the start of the next second.

func TestAdminReset_StampsNextSecondAndRevokesRefreshTokensInOneTransaction(t *testing.T) {
	f := &recDB{}
	db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, NewRepository(db).UpdatePassword(context.Background(), uuidA, "new-hash"))

	var userUpdate, refreshRevoke string
	for _, q := range f.queries {
		switch {
		case strings.Contains(q, "UPDATE users"):
			userUpdate = q
		case strings.Contains(q, "UPDATE refresh_tokens"):
			refreshRevoke = q
		}
	}
	assert.Contains(t, userUpdate, "password_changed_at = date_trunc('second', now()) + interval '1 second'",
		"the stamp must be the start of the next second, so a token issued in the reset's own second is rejected")
	assert.Contains(t, refreshRevoke, "revoked_at = now()", "every refresh token of the user must be revoked")
	assert.Contains(t, refreshRevoke, "user_id = $1")
	assert.Contains(t, refreshRevoke, "revoked_at IS NULL")
}
