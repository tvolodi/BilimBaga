package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-160: the backend, not just the SPA, enforces users.force_password_change.

func forceToken(t *testing.T) string {
	t.Helper()
	return makeToken(t, testSecret, jwt.MapClaims{
		"sub": "u1", "role": "employee",
		"iat": time.Now().Unix(), "exp": time.Now().Add(15 * time.Minute).Unix(),
	})
}

func runForce(t *testing.T, lookup AccountStateLookup, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	h := Authenticate(testSecret, WithAccountState(lookup))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+forceToken(t))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func forced(forceFlag bool) AccountStateLookup {
	return func(context.Context, string) (AccountState, error) {
		return AccountState{ForcePasswordChange: forceFlag}, nil
	}
}

func TestAuthenticate_ForcePasswordChange_BlocksEverythingElse(t *testing.T) {
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/exams"},
		{http.MethodPost, "/api/v1/sessions"},
		{http.MethodGet, "/api/v1/portal/exams"},
		{http.MethodGet, "/api/v1/users/me/"},
		{http.MethodPut, "/api/v1/users/me"}, // only GET /users/me is allowed
		{http.MethodGet, "/api/v1/auth/change-password/x"},
	} {
		rec := runForce(t, forced(true), tc.method, tc.path)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s", tc.method, tc.path)
		assert.Equal(t, "PASSWORD_CHANGE_REQUIRED", decodeErrorEnvelope(t, rec), "%s %s", tc.method, tc.path)
	}
}

func TestAuthenticate_ForcePasswordChange_AllowlistPasses(t *testing.T) {
	assert.Equal(t, http.StatusOK, runForce(t, forced(true), http.MethodPost, "/api/v1/auth/change-password").Code)
	assert.Equal(t, http.StatusOK, runForce(t, forced(true), http.MethodGet, "/api/v1/users/me").Code)
}

func TestAuthenticate_ForcePasswordChange_FlagFalsePasses(t *testing.T) {
	assert.Equal(t, http.StatusOK, runForce(t, forced(false), http.MethodGet, "/api/v1/users").Code)
}

func TestAuthenticate_ForcePasswordChange_LookupFailuresFailClosed(t *testing.T) {
	gone := func(context.Context, string) (AccountState, error) { return AccountState{}, ErrNotFound }
	assert.Equal(t, http.StatusUnauthorized, runForce(t, gone, http.MethodGet, "/api/v1/users").Code, "unknown user")

	broken := func(context.Context, string) (AccountState, error) { return AccountState{}, errors.New("db down") }
	for _, path := range []string{"/api/v1/users", "/api/v1/auth/change-password", "/api/v1/users/me"} {
		rec := runForce(t, broken, http.MethodGet, path)
		assert.Equal(t, http.StatusInternalServerError, rec.Code, path)
		assert.Equal(t, "INTERNAL_ERROR", decodeErrorEnvelope(t, rec), path)
	}
}

func TestAccountStateCache_InvalidateForcesReread(t *testing.T) {
	var mu sync.Mutex
	force := true
	fetches := 0
	clock := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	c := NewAccountStateCache(func(context.Context, string) (AccountState, error) {
		mu.Lock()
		defer mu.Unlock()
		fetches++
		return AccountState{ForcePasswordChange: force}, nil
	}, func() time.Time { return clock })

	st, err := c.Lookup(context.Background(), "u1")
	require.NoError(t, err)
	assert.True(t, st.ForcePasswordChange)

	mu.Lock()
	force = false // the DB row is updated by change-password
	mu.Unlock()

	st, _ = c.Lookup(context.Background(), "u1")
	assert.True(t, st.ForcePasswordChange, "still cached (stale) within the TTL")

	c.Invalidate("u1")
	st, _ = c.Lookup(context.Background(), "u1")
	assert.False(t, st.ForcePasswordChange, "invalidate makes the change visible immediately")
	assert.Equal(t, 2, fetches)

	c.Invalidate("never-cached") // must not panic
}

func TestAccountStateCache_ErrorsAreNotCached(t *testing.T) {
	clock := time.Now()
	calls := 0
	c := NewAccountStateCache(func(context.Context, string) (AccountState, error) {
		calls++
		if calls == 1 {
			return AccountState{}, errors.New("blip")
		}
		return AccountState{}, nil
	}, func() time.Time { return clock })
	_, err := c.Lookup(context.Background(), "u1")
	require.Error(t, err)
	_, err = c.Lookup(context.Background(), "u1")
	require.NoError(t, err)
}

// changePassword then immediate access: the handler hook invalidates the cache so the very
// next request is not blocked by a stale force_password_change=true.
func TestChangePasswordHandler_ClearsFlagThenImmediateAccess(t *testing.T) {
	var mu sync.Mutex
	force := true // simulated users.force_password_change
	clock := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cache := NewAccountStateCache(func(context.Context, string) (AccountState, error) {
		mu.Lock()
		defer mu.Unlock()
		return AccountState{ForcePasswordChange: force}, nil
	}, func() time.Time { return clock })

	svc := &mockService{
		parseTokenFn: func(string) (*Claims, error) { return makeValidClaims(), nil },
		changePasswordFn: func(_ context.Context, _ string, _ *ChangePasswordRequest, _ string) error {
			mu.Lock()
			force = false
			mu.Unlock()
			return nil
		},
	}
	h := NewHandler(svc, nil)
	h.SetPasswordChangedHook(cache.Invalidate)

	tok := makeToken(t, testSecret, jwt.MapClaims{
		"sub": "user-uuid-1", "role": "employee",
		"iat": clock.Unix(), "exp": clock.Add(time.Hour).Unix(),
	})
	call := func(method, path string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		mw := Authenticate(testSecret, WithAccountState(cache.Lookup))(handler)
		req := httptest.NewRequest(method, path, strings.NewReader(`{"current_password":"a","new_password":"b"}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		return rec
	}
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	// Blocked before the change (and the blocked state is now cached).
	assert.Equal(t, http.StatusForbidden, call(http.MethodGet, "/api/v1/users", ok).Code)

	// Change password through the real handler behind the middleware.
	rec := call(http.MethodPost, "/api/v1/auth/change-password", h.ChangePassword)
	require.Equal(t, http.StatusOK, rec.Code)

	// Same token, same clock (inside the TTL): no longer blocked.
	assert.Equal(t, http.StatusOK, call(http.MethodGet, "/api/v1/users", ok).Code)
}

func TestResetPasswordHandler_InvalidatesCache(t *testing.T) {
	var got string
	svc := &mockService{resetFn: func(context.Context, *ResetPasswordRequest, string) (string, error) { return "u9", nil }}
	h := NewHandler(svc, nil)
	h.SetPasswordChangedHook(func(id string) { got = id })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(`{"token":"t","new_password":"x"}`))
	rec := httptest.NewRecorder()
	h.ResetPassword(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "u9", got)
}

func TestChangePasswordHandler_FailureDoesNotInvalidate(t *testing.T) {
	called := false
	svc := &mockService{
		parseTokenFn:     func(string) (*Claims, error) { return makeValidClaims(), nil },
		changePasswordFn: func(context.Context, string, *ChangePasswordRequest, string) error { return errors.New("boom") },
	}
	h := NewHandler(svc, nil)
	h.SetPasswordChangedHook(func(string) { called = true })
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"current_password":"a","new_password":"b"}`))
	req.Header.Set("Authorization", "Bearer t")
	h.ChangePassword(httptest.NewRecorder(), req)
	assert.False(t, called)
}

func TestAccountStateCache_InvalidateDuringFetchDoesNotStoreStale(t *testing.T) {
	clock := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	var c *AccountStateCache
	force := true
	c = NewAccountStateCache(func(context.Context, string) (AccountState, error) {
		st := AccountState{ForcePasswordChange: force}
		force = false  // DB updated after the read...
		c.Invalidate("u1") // ...and invalidated while the read was in flight
		return st, nil
	}, func() time.Time { return clock })
	_, _ = c.Lookup(context.Background(), "u1")
	st, _ := c.Lookup(context.Background(), "u1")
	assert.False(t, st.ForcePasswordChange, "stale in-flight result must not be cached")
}
