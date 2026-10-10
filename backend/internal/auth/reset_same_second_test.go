package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// #439 (architect, PR 453): a password reset stamps password_changed_at at the start of the next
// second, so a token issued before the reset in the same second is revoked. A sign-in or refresh in
// the reset's own second must not be revoked from birth: its iat is max(now, password_changed_at).

// resetEnv is one account whose password was reset at stamp. The real service signs in and refreshes
// it, and the real Authenticate middleware reads the same account state through a fake lookup.
type resetEnv struct {
	t     *testing.T
	svc   *service
	clock time.Time
	stamp *time.Time // users.password_changed_at; nil when the password was never reset
}

func newResetEnv(t *testing.T, clock time.Time, stamp *time.Time) *resetEnv {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("OldPass123"), 4)
	require.NoError(t, err)
	e := &resetEnv{t: t, clock: clock, stamp: stamp}
	account := func() *User {
		return &User{
			ID: "u1", Email: "a@example.com", RoleName: "employee", Status: "active",
			PasswordHash: string(hash), PasswordChangedAt: e.stamp,
		}
	}
	repo := &mockRepository{
		getUserByEmailFn: func(context.Context, string) (*User, error) { return account(), nil },
		getUserByIDFn:    func(context.Context, string) (*User, error) { return account(), nil },
		getRefreshTokenByHashFn: func(context.Context, string) (*RefreshToken, error) {
			return &RefreshToken{ID: "rt1", UserID: "u1", TokenHash: "h", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
		},
		updatePasswordFn: func(_ context.Context, _ string, _ string, at time.Time) error {
			e.stamp = &at // the change stamps the account, as the users row would
			return nil
		},
	}
	e.svc = NewService(ServiceConfig{JWTSecret: selfChangeSecret, JWTAccessTTLMin: 15, JWTRefreshTTLDays: 7, BcryptCost: 4}, repo).(*service)
	e.svc.now = func() time.Time { return e.clock }
	return e
}

// resetStamp is the password_changed_at a reset at now writes: the start of the next second.
func resetStamp(now time.Time) time.Time { return now.Truncate(time.Second).Add(time.Second) }

func (e *resetEnv) tokenAt(iat time.Time) string {
	e.t.Helper()
	return makeToken(e.t, selfChangeSecret, jwt.MapClaims{
		"sub": "u1", "role": "employee", "iat": iat.Unix(), "exp": iat.Add(time.Hour).Unix(),
	})
}

// get calls a protected endpoint through the real Authenticate middleware over the same account state.
func (e *resetEnv) get(tok string) *httptest.ResponseRecorder {
	lookup := func(context.Context, string) (AccountState, error) {
		st := AccountState{}
		if e.stamp != nil {
			st.PasswordChangedAt = *e.stamp
		}
		return st, nil
	}
	mw := Authenticate(selfChangeSecret, WithAccountState(lookup))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	return rec
}

func TestSignIn_InTheResetsOwnSecond_IsNotRevokedFromBirth(t *testing.T) {
	reset := testNow(700_000_000) // 700 ms into the second; the reset stamps the next second
	stamp := resetStamp(reset)
	e := newResetEnv(t, reset, &stamp)

	resp, _, err := e.svc.Login(context.Background(), &LoginRequest{Email: "a@example.com", Password: "OldPass123"}, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code, "a sign-in in the reset's own second must get a usable token")
}

func TestRefresh_InTheResetsOwnSecond_IsNotRevokedFromBirth(t *testing.T) {
	reset := testNow(700_000_000)
	stamp := resetStamp(reset)
	e := newResetEnv(t, reset, &stamp)

	resp, _, err := e.svc.Refresh(context.Background(), "raw-refresh-token", "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code, "a refresh in the reset's own second must get a usable token")
}

func TestToken_IssuedBeforeTheReset_InTheSameSecond_IsRevoked(t *testing.T) {
	reset := testNow(700_000_000)
	stamp := resetStamp(reset)
	e := newResetEnv(t, reset, &stamp)

	old := e.tokenAt(reset.Add(-300 * time.Millisecond)) // issued in the same second, before the reset
	rec := e.get(old)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "TOKEN_REVOKED")
}

func TestSignIn_AfterTheStamp_UsesTheClock(t *testing.T) {
	reset := testNow(700_000_000)
	stamp := resetStamp(reset)
	later := stamp.Add(500 * time.Millisecond)
	e := newResetEnv(t, later, &stamp)

	resp, _, err := e.svc.Login(context.Background(), &LoginRequest{Email: "a@example.com", Password: "OldPass123"}, "127.0.0.1")
	require.NoError(t, err)
	claims, err := e.svc.ParseAccessToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, later.Unix(), claims.IssuedAt.Unix(), "after the stamp the token carries the clock's second")
	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code)
}

// A sign-in in the reset's own second followed at once by a password change: the change must revoke
// the sign-in's token, because that token's iat can be the next second (#439).
func TestChangePassword_RightAfterSignInInTheResetsSecond_RevokesThatToken(t *testing.T) {
	reset := testNow(700_000_000)
	stamp := resetStamp(reset)
	e := newResetEnv(t, reset, &stamp)

	signIn, _, err := e.svc.Login(context.Background(), &LoginRequest{Email: "a@example.com", Password: "OldPass123"}, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, e.get(signIn.AccessToken).Code, "sanity: the sign-in token works before the change")

	changed, _, err := e.svc.ChangePassword(context.Background(), "u1", &ChangePasswordRequest{CurrentPassword: "OldPass123", NewPassword: "NewPass456"}, "127.0.0.1")
	require.NoError(t, err)
	rec := e.get(signIn.AccessToken)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "the sign-in token issued before the change must be revoked")
	assert.Contains(t, rec.Body.String(), "TOKEN_REVOKED")
	assert.Equal(t, http.StatusOK, e.get(changed.AccessToken).Code, "the token from the change works at once")
}

func TestSignIn_BeforeAnyReset_IsUnaffected(t *testing.T) {
	now := testNow(200_000_000)
	e := newResetEnv(t, now, nil)

	resp, _, err := e.svc.Login(context.Background(), &LoginRequest{Email: "a@example.com", Password: "OldPass123"}, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code)
}
