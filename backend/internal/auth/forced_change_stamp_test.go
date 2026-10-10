package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #466: the forced-change branch of the admin bootstrap (the admin is still on the default password and
// no new password is supplied) sets force_password_change, so it must also stamp password_changed_at.
// Otherwise an access token issued before the branch runs (for example before a restart) keeps working.
// The stamp is read back from the statement's arguments in the transaction fake, and the real
// Authenticate middleware checks the old token against that account state. No Postgres is needed.

// forcedChangeStamp returns the password_changed_at argument of the statement that sets
// force_password_change = true, and whether that statement stamps password_changed_at at all.
func forcedChangeStamp(t *testing.T, f *txFakeDB) (time.Time, bool) {
	t.Helper()
	for i, q := range f.stmts {
		if !strings.Contains(q, "force_password_change = true") {
			continue
		}
		if !strings.Contains(q, "password_changed_at") {
			return time.Time{}, false
		}
		for _, a := range f.args[i] {
			if at, ok := a.(time.Time); ok {
				return at, true
			}
		}
		return time.Time{}, false
	}
	t.Fatal("no statement sets force_password_change = true")
	return time.Time{}, false
}

// callWithAccountState runs tok through the real Authenticate middleware over the given account state.
func callWithAccountState(tok string, st AccountState) *httptest.ResponseRecorder {
	lookup := func(context.Context, string) (AccountState, error) { return st, nil }
	mw := Authenticate(selfChangeSecret, WithAccountState(lookup))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	return rec
}

func TestForcedChange_StampsPasswordChangedAt_RevokesTokenIssuedBeforeIt(t *testing.T) {
	clock := testNow(700_000_000) // the old token and the branch share the same second
	repo, f := newTxFake(t)
	store := &pgBootstrapStore{db: repo.(*pgRepository).db, now: func() time.Time { return clock }}

	old := makeToken(t, selfChangeSecret, jwt.MapClaims{
		"sub": "u1", "role": "super_admin", "iat": clock.Truncate(time.Second).Unix(), "exp": clock.Add(time.Hour).Unix(),
	})
	require.Equal(t, http.StatusOK, callWithAccountState(old, AccountState{}).Code, "sanity: valid before the branch runs")

	ok, err := store.RequireAdminPasswordChange(context.Background(), "u1", "default-hash")
	require.NoError(t, err)
	require.True(t, ok)

	stamp, stamped := forcedChangeStamp(t, f)
	require.True(t, stamped, "the forced-change branch must stamp password_changed_at (#466)")
	assert.Equal(t, clock.Truncate(time.Second).Add(time.Second), stamp, "the stamp is the next second after the application clock")

	rec := callWithAccountState(old, AccountState{PasswordChangedAt: stamp})
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a token issued before the forced change must be rejected")
	assert.Contains(t, rec.Body.String(), "TOKEN_REVOKED")
}
