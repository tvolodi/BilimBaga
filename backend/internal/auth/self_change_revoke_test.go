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
	"golang.org/x/crypto/bcrypt"
)

// ISS-171: a self password change stamps password_changed_at, revokes every earlier access
// token and every refresh token, and hands the caller a fresh session.

const selfChangeSecret = "a-secret-that-is-at-least-32-chars!!"

// selfChangeEnv wires the real service to an in-memory account (the stand-in for the users /
// refresh_tokens rows) and exposes the real Authenticate middleware over the same state.
type selfChangeEnv struct {
	t          *testing.T
	svc        *service
	clock      time.Time
	changedAt  time.Time // users.password_changed_at
	force      bool      // users.force_password_change
	stamps     []time.Time
	events     []string // ordered repo writes
	revoked    bool     // all pre-existing refresh tokens revoked
	newRefresh int
}

func newSelfChangeEnv(t *testing.T, now time.Time, force bool) *selfChangeEnv {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("OldPass123"), 4)
	require.NoError(t, err)
	e := &selfChangeEnv{t: t, clock: now, force: force}
	repo := &mockRepository{
		getUserByIDFn: func(context.Context, string) (*User, error) {
			return &User{ID: "u1", Email: "a@example.com", RoleName: "employee", PasswordHash: string(hash), ForcePasswordChange: e.force}, nil
		},
		updatePasswordFn: func(_ context.Context, _ string, _ string, at time.Time) error {
			e.events = append(e.events, "update+revoke")
			e.changedAt, e.force, e.revoked = at, false, true
			e.stamps = append(e.stamps, at)
			return nil
		},
		createRefreshTokenFn: func(context.Context, *RefreshToken) error {
			e.events = append(e.events, "create-refresh")
			e.newRefresh++
			return nil
		},
	}
	e.svc = NewService(ServiceConfig{JWTSecret: selfChangeSecret, JWTAccessTTLMin: 15, JWTRefreshTTLDays: 7, BcryptCost: 4}, repo).(*service)
	e.svc.now = func() time.Time { return e.clock }
	return e
}

func (e *selfChangeEnv) tokenAt(iat time.Time) string {
	e.t.Helper()
	return makeToken(e.t, selfChangeSecret, jwt.MapClaims{
		"sub": "u1", "role": "employee", "iat": iat.Unix(), "exp": iat.Add(time.Hour).Unix(),
	})
}

// get calls a protected endpoint through the real Authenticate middleware.
func (e *selfChangeEnv) get(tok string) *httptest.ResponseRecorder {
	lookup := func(context.Context, string) (AccountState, error) {
		return AccountState{PasswordChangedAt: e.changedAt, ForcePasswordChange: e.force}, nil
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

func (e *selfChangeEnv) change(current string) (*ChangePasswordResponse, *http.Cookie, error) {
	return e.svc.ChangePassword(context.Background(), "u1", &ChangePasswordRequest{CurrentPassword: current, NewPassword: "NewPass456"}, "127.0.0.1")
}

func TestSelfChange_OldTokenRevoked_NewTokenWorksImmediately(t *testing.T) {
	now := testClock(0, 500_000_000)
	e := newSelfChangeEnv(t, now, false)
	old := e.tokenAt(now.Add(-10 * time.Minute))
	require.Equal(t, http.StatusOK, e.get(old).Code, "sanity: valid before the change")

	resp, cookie, err := e.change("OldPass123")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, cookie)
	assert.Equal(t, "password changed", resp.Message)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, 900, resp.ExpiresIn)

	rec := e.get(old)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "TOKEN_REVOKED")

	// The returned token works at once (same second as the stamp).
	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code)
}

func TestSelfChange_BoundarySameSecond(t *testing.T) {
	// The clock sits 700ms into the second: the stamp is truncated to the second and the new
	// token's iat is that same second, so iat == floor(stamp) and the new token passes.
	now := testClock(0, 700_000_000)
	e := newSelfChangeEnv(t, now, false)
	resp, _, err := e.change("OldPass123")
	require.NoError(t, err)

	require.Len(t, e.stamps, 1)
	assert.Equal(t, now.Truncate(time.Second), e.stamps[0])
	claims, err := e.svc.ParseAccessToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, e.stamps[0].Unix(), claims.IssuedAt.Unix(), "new token iat equals the stamp second")
	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code)

	// iat one second before the stamp is revoked; the same second is accepted (documented
	// second-granularity caveat: the epoch is inclusive).
	assert.Equal(t, http.StatusUnauthorized, e.get(e.tokenAt(e.stamps[0].Add(-time.Second))).Code)
	assert.Equal(t, http.StatusOK, e.get(e.tokenAt(e.stamps[0])).Code)
}

func TestSelfChange_Boundary_FractionalSecondStamp(t *testing.T) {
	// If a stamp carried a fraction (e.g. a database now()), floor semantics still hold.
	changedAt := testClock(5, 900_000_000)
	assert.False(t, tokenPredatesPasswordChange(changedAt.Unix(), changedAt))
	assert.True(t, tokenPredatesPasswordChange(changedAt.Unix()-1, changedAt))
	assert.False(t, tokenPredatesPasswordChange(changedAt.Unix()+1, changedAt))
	assert.False(t, tokenPredatesPasswordChange(0, time.Time{}), "never changed: nothing predates it")
}

func TestSelfChange_RefreshTokensRevokedThenCallerGetsNewOne(t *testing.T) {
	now := testClock(0, 0)
	e := newSelfChangeEnv(t, now, false)
	_, cookie, err := e.change("OldPass123")
	require.NoError(t, err)

	// The revoke-all happens inside the update, strictly before the caller's new refresh
	// token is created, so the new one survives and every other session's one is revoked.
	assert.Equal(t, []string{"update+revoke", "create-refresh"}, e.events)
	assert.True(t, e.revoked)
	assert.Equal(t, 1, e.newRefresh)
	assert.Equal(t, "refresh_token", cookie.Name)
	assert.NotEmpty(t, cookie.Value)
	assert.True(t, cookie.HttpOnly)
}

func TestSelfChange_ForcedChangeFlowStillWorksWithReturnedToken(t *testing.T) {
	now := testClock(0, 0)
	e := newSelfChangeEnv(t, now, true)
	old := e.tokenAt(now.Add(-time.Minute))
	assert.Equal(t, http.StatusForbidden, e.get(old).Code, "blocked before the change")

	resp, _, err := e.change("OldPass123")
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, e.get(resp.AccessToken).Code, "returned token passes, flag cleared")
	assert.Equal(t, http.StatusUnauthorized, e.get(old).Code, "pre-change token is revoked")
}

func TestSelfChange_WrongCurrentPasswordDoesNotStamp(t *testing.T) {
	now := testClock(0, 0)
	e := newSelfChangeEnv(t, now, false)
	old := e.tokenAt(now.Add(-time.Minute))

	resp, cookie, err := e.change("WrongPass1")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, cookie)
	assert.Empty(t, e.stamps)
	assert.Empty(t, e.events)
	assert.True(t, e.changedAt.IsZero())
	assert.Equal(t, http.StatusOK, e.get(old).Code, "existing token is untouched")
}

func TestSelfChange_WeakPasswordDoesNotStamp(t *testing.T) {
	now := testClock(0, 0)
	e := newSelfChangeEnv(t, now, false)
	_, _, err := e.svc.ChangePassword(context.Background(), "u1", &ChangePasswordRequest{CurrentPassword: "OldPass123", NewPassword: "weak"}, "")
	require.Error(t, err)
	assert.Empty(t, e.stamps)
}

// Through the HTTP handler: the body carries the token, the cookie is set, the hook runs.
func TestSelfChange_HandlerReturnsSessionAndCookie(t *testing.T) {
	now := testClock(0, 0)
	e := newSelfChangeEnv(t, now, false)
	h := NewHandler(e.svc, nil)
	var hooked string
	h.SetPasswordChangedHook(func(id string) { hooked = id })
	h.writer = nopAudit{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password",
		strings.NewReader(`{"current_password":"OldPass123","new_password":"NewPass456"}`))
	req.Header.Set("Authorization", "Bearer "+e.tokenAt(now.Add(-time.Minute)))
	w := httptest.NewRecorder()
	h.ChangePassword(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "u1", hooked)
	assert.Contains(t, w.Body.String(), `"access_token"`)
	var got *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			got = c
		}
	}
	require.NotNil(t, got)
}

type nopAudit struct{}

func (nopAudit) Write(context.Context, *http.Request, string, string, *string, any) {}

// testClock returns a deterministic sub-minute offset from the start of the current UTC
// minute. The JWT library validates exp against the real clock, so fixed calendar dates
// make these tests rot once the date passes.
func testClock(sec, nsec int) time.Time {
	return time.Now().UTC().Truncate(time.Minute).Add(time.Duration(sec)*time.Second + time.Duration(nsec))
}
