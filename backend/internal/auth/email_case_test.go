package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Regression tests for ISS-164: user creation stores emails lowercased, so login and
// password recovery must normalise (trim + lowercase) what the user typed.

func strictLowercaseRepo(t *testing.T, stored string, hash []byte) *mockRepository {
	t.Helper()
	return &mockRepository{
		getUserByEmailFn: func(_ context.Context, email string) (*User, error) {
			if strings.ToLower(stored) != email { // mirrors the SQL "WHERE lower(u.email) = $1"
				return nil, ErrNotFound
			}
			return &User{ID: "u1", Email: stored, PasswordHash: string(hash), RoleName: "employee", Status: "active"}, nil
		},
	}
}

func TestService_Login_MixedCaseAndWhitespaceEmail_Succeeds(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("ValidPass123!"), 4)
	require.NoError(t, err)
	svc := NewService(ServiceConfig{
		JWTSecret: "a-secret-that-is-at-least-32-chars!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 7, BcryptCost: 4,
	}, strictLowercaseRepo(t, "john.doe@corp.com", hash))

	for _, typed := range []string{"John.Doe@Corp.com", "  john.doe@corp.com\t", "JOHN.DOE@CORP.COM", "john.doe@corp.com"} {
		resp, _, err := svc.Login(context.Background(), &LoginRequest{Email: typed, Password: "ValidPass123!"}, "127.0.0.1")
		require.NoError(t, err, typed)
		assert.Equal(t, "john.doe@corp.com", resp.User.Email)
	}
}

func TestService_Login_WrongEmailStillUnauthorized(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("ValidPass123!"), 4)
	svc := NewService(ServiceConfig{JWTSecret: "a-secret-that-is-at-least-32-chars!!", BcryptCost: 4},
		strictLowercaseRepo(t, "john.doe@corp.com", hash))
	_, _, err := svc.Login(context.Background(), &LoginRequest{Email: "Other@Corp.com", Password: "ValidPass123!"}, "127.0.0.1")
	assert.Equal(t, "INVALID_CREDENTIALS", serviceErrCode(t, err))
}

func TestLoginHandler_MixedCaseEmail_Returns200(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("ValidPass123!"), 4)
	svc := NewService(ServiceConfig{
		JWTSecret: "a-secret-that-is-at-least-32-chars!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 7, BcryptCost: 4,
	}, strictLowercaseRepo(t, "john.doe@corp.com", hash))
	h, _ := newRecoveryHandler(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"John.Doe@Corp.com","password":"ValidPass123!"}`))
	h.Login(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestForgotPassword_MixedCaseAndWhitespaceEmail_IssuesTokenAndMails(t *testing.T) {
	for _, typed := range []string{"Alice@Example.COM", "  alice@example.com ", "ALICE@example.com"} {
		f := newRecoveryFixture(t)
		id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: typed}, "1.2.3.4")
		require.NoError(t, err, typed)
		assert.Equal(t, "u1", id, typed)
		require.Len(t, f.mailer.calls, 1, typed)
		assert.Len(t, f.repo.resets, 1, typed)
	}
}

func TestForgotPassword_MixedCaseUnknownEmail_StaysNeutral(t *testing.T) {
	f := newRecoveryFixture(t)
	id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "Nobody@Example.com"}, "1.2.3.4")
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Empty(t, f.mailer.calls)
	assert.Empty(t, f.repo.resets)
}

// Cycle 2: legacy rows whose stored email is mixed case must stay reachable.

func TestService_Login_LegacyMixedCaseStoredRow_Succeeds(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("ValidPass123!"), 4)
	require.NoError(t, err)
	svc := NewService(ServiceConfig{
		JWTSecret: "a-secret-that-is-at-least-32-chars!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 7, BcryptCost: 4,
	}, strictLowercaseRepo(t, "John.Doe@Corp.com", hash))

	for _, typed := range []string{"john.doe@corp.com", "JOHN.DOE@CORP.COM", " John.Doe@Corp.com "} {
		resp, _, err := svc.Login(context.Background(), &LoginRequest{Email: typed, Password: "ValidPass123!"}, "127.0.0.1")
		require.NoError(t, err, typed)
		assert.Equal(t, "John.Doe@Corp.com", resp.User.Email)
	}
}

func TestForgotPassword_LegacyMixedCaseStoredRow_IssuesTokenAndMails(t *testing.T) {
	f := newRecoveryFixture(t)
	f.repo.addUser(&User{ID: "u9", Email: "John.Doe@Corp.com", Status: "active", RoleName: "employee"})
	id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "john.doe@corp.com"}, "1.2.3.4")
	require.NoError(t, err)
	assert.Equal(t, "u9", id)
	assert.Len(t, f.mailer.calls, 1)
}
