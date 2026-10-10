package auth

import (
	"context"
	"encoding/json"
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

// --- Mock Service ---

type mockService struct {
	loginFn          func(ctx context.Context, req *LoginRequest, ipAddr string) (*LoginResponse, *http.Cookie, error)
	refreshFn        func(ctx context.Context, rawToken, ipAddr string) (*RefreshResponse, *http.Cookie, error)
	logoutFn         func(ctx context.Context, rawToken, ipAddr string) (*http.Cookie, error)
	changePasswordFn func(ctx context.Context, userID string, req *ChangePasswordRequest, ipAddr string) (*ChangePasswordResponse, *http.Cookie, error)
	parseTokenFn     func(tokenString string) (*Claims, error)
	forgotFn         func(ctx context.Context, req *ForgotPasswordRequest, ipAddr string) (string, error)
	resetFn          func(ctx context.Context, req *ResetPasswordRequest, ipAddr string) (string, error)
}

func (m *mockService) Login(ctx context.Context, req *LoginRequest, ipAddr string) (*LoginResponse, *http.Cookie, error) {
	return m.loginFn(ctx, req, ipAddr)
}

func (m *mockService) Refresh(ctx context.Context, rawToken, ipAddr string) (*RefreshResponse, *http.Cookie, error) {
	return m.refreshFn(ctx, rawToken, ipAddr)
}

func (m *mockService) Logout(ctx context.Context, rawToken, ipAddr string) (*http.Cookie, error) {
	return m.logoutFn(ctx, rawToken, ipAddr)
}

func (m *mockService) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest, ipAddr string) (*ChangePasswordResponse, *http.Cookie, error) {
	return m.changePasswordFn(ctx, userID, req, ipAddr)
}

func (m *mockService) ParseAccessToken(tokenString string) (*Claims, error) {
	return m.parseTokenFn(tokenString)
}

func (m *mockService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest, ipAddr string) (string, error) {
	return m.forgotFn(ctx, req, ipAddr)
}

func (m *mockService) ResetPassword(ctx context.Context, req *ResetPasswordRequest, ipAddr string) (string, error) {
	return m.resetFn(ctx, req, ipAddr)
}

// --- Helpers ---

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) (data json.RawMessage, apiErr *apiError) {
	t.Helper()
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error *apiError       `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	return env.Data, env.Error
}

func makeValidClaims() *Claims {
	return &Claims{
		Email: "alice@example.com",
		Role:  "employee",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-uuid-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
}

func refreshCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		HttpOnly: true,
		Path:     "/api/v1/auth",
	}
}

// --- Login Tests ---

func TestLogin_ValidCredentials_Returns200WithTokenAndCookie(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, req *LoginRequest, _ string) (*LoginResponse, *http.Cookie, error) {
			return &LoginResponse{
				AccessToken: "access.token.here",
				TokenType:   "Bearer",
				ExpiresIn:   900,
				User:        UserInfo{ID: "user-1", FullName: "Alice", Email: req.Email, Role: "employee"},
			}, refreshCookie("raw-refresh-token"), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"email":"alice@example.com","password":"Secret123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	data, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)

	var loginResp LoginResponse
	require.NoError(t, json.Unmarshal(data, &loginResp))
	assert.Equal(t, "access.token.here", loginResp.AccessToken)
	assert.Equal(t, "Bearer", loginResp.TokenType)
	assert.Equal(t, 900, loginResp.ExpiresIn)

	cookies := resp.Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "refresh_token", cookies[0].Name)
	assert.Equal(t, "raw-refresh-token", cookies[0].Value)
}

func TestLogin_InvalidCredentials_Returns401(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, _ *LoginRequest, _ string) (*LoginResponse, *http.Cookie, error) {
			return nil, nil, &ServiceError{Code: "INVALID_CREDENTIALS", Message: "invalid email or password", HTTPStatus: http.StatusUnauthorized}
		},
	}
	h := NewHandler(svc, nil)

	body := `{"email":"alice@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_CREDENTIALS", apiErr.Code)
}

func TestLogin_LockedAccount_Returns423(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, _ *LoginRequest, _ string) (*LoginResponse, *http.Cookie, error) {
			return nil, nil, &ServiceError{Code: "ACCOUNT_LOCKED", Message: "account locked", HTTPStatus: http.StatusLocked}
		},
	}
	h := NewHandler(svc, nil)

	body := `{"email":"alice@example.com","password":"Secret123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusLocked, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ACCOUNT_LOCKED", apiErr.Code)
}

func TestLogin_MalformedBody_Returns400(t *testing.T) {
	h := NewHandler(&mockService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not-json"))
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Refresh Tests ---

func TestRefresh_ValidCookie_Returns200WithNewToken(t *testing.T) {
	svc := &mockService{
		refreshFn: func(_ context.Context, rawToken, _ string) (*RefreshResponse, *http.Cookie, error) {
			return &RefreshResponse{AccessToken: "new.access.token", TokenType: "Bearer", ExpiresIn: 900},
				refreshCookie("new-refresh-token"), nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(refreshCookie("old-refresh-token"))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)

	var refreshResp RefreshResponse
	require.NoError(t, json.Unmarshal(data, &refreshResp))
	assert.Equal(t, "new.access.token", refreshResp.AccessToken)
}

func TestRefresh_NoCookie_Returns401(t *testing.T) {
	h := NewHandler(&mockService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_REFRESH_TOKEN", apiErr.Code)
}

func TestRefresh_RevokedToken_Returns401(t *testing.T) {
	svc := &mockService{
		refreshFn: func(_ context.Context, _ string, _ string) (*RefreshResponse, *http.Cookie, error) {
			return nil, nil, &ServiceError{Code: "INVALID_REFRESH_TOKEN", Message: "refresh token is invalid or expired", HTTPStatus: http.StatusUnauthorized}
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(refreshCookie("revoked-token"))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_REFRESH_TOKEN", apiErr.Code)
}

// --- Logout Tests ---

func TestLogout_WithCookie_Returns200AndClearsCookie(t *testing.T) {
	svc := &mockService{
		logoutFn: func(_ context.Context, rawToken, _ string) (*http.Cookie, error) {
			return &http.Cookie{Name: "refresh_token", Value: "", MaxAge: -1, Path: "/api/v1/auth"}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(refreshCookie("some-token"))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)

	// Cookie should be cleared
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "refresh_token", cookies[0].Name)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestLogout_WithoutCookie_Returns200(t *testing.T) {
	svc := &mockService{
		logoutFn: func(_ context.Context, _ string, _ string) (*http.Cookie, error) {
			return &http.Cookie{Name: "refresh_token", Value: "", MaxAge: -1, Path: "/api/v1/auth"}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	h.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- ChangePassword Tests ---

func TestChangePassword_Valid_Returns200(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(_ string) (*Claims, error) {
			return makeValidClaims(), nil
		},
		changePasswordFn: func(_ context.Context, userID string, _ *ChangePasswordRequest, _ string) (*ChangePasswordResponse, *http.Cookie, error) {
			assert.Equal(t, "user-uuid-1", userID)
			return &ChangePasswordResponse{Message: "password changed", AccessToken: "new.jwt.token", TokenType: "Bearer", ExpiresIn: 900},
				&http.Cookie{Name: "refresh_token", Value: "new-refresh"}, nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"current_password":"OldPass123!","new_password":"NewPass456!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid.jwt.token")
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)

	var msg map[string]any
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "password changed", msg["message"])
	// ISS-171: the response carries the caller's replacement session.
	assert.Equal(t, "new.jwt.token", msg["access_token"])
	assert.Equal(t, "Bearer", msg["token_type"])
	var refresh *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			refresh = c
		}
	}
	require.NotNil(t, refresh)
	assert.Equal(t, "new-refresh", refresh.Value)
}

func TestChangePassword_ShortNewPassword_Returns422(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(_ string) (*Claims, error) {
			return makeValidClaims(), nil
		},
		changePasswordFn: func(_ context.Context, _ string, _ *ChangePasswordRequest, _ string) (*ChangePasswordResponse, *http.Cookie, error) {
			return nil, nil, &ServiceError{Code: "VALIDATION_ERROR", Message: "new password must be at least 8 characters", HTTPStatus: http.StatusUnprocessableEntity}
		},
	}
	h := NewHandler(svc, nil)

	body := `{"current_password":"OldPass123!","new_password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid.jwt.token")
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
}

func TestChangePassword_WrongCurrentPassword_Returns400(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(_ string) (*Claims, error) {
			return makeValidClaims(), nil
		},
		changePasswordFn: func(_ context.Context, _ string, _ *ChangePasswordRequest, _ string) (*ChangePasswordResponse, *http.Cookie, error) {
			return nil, nil, &ServiceError{Code: "INVALID_CREDENTIALS", Message: "current password is incorrect", HTTPStatus: http.StatusBadRequest}
		},
	}
	h := NewHandler(svc, nil)

	body := `{"current_password":"WrongPass!","new_password":"NewPass456!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid.jwt.token")
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_CREDENTIALS", apiErr.Code)
}

func TestChangePassword_NoAuthHeader_Returns401(t *testing.T) {
	h := NewHandler(&mockService{}, nil)

	body := `{"current_password":"OldPass123!","new_password":"NewPass456!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "UNAUTHORIZED", apiErr.Code)
}

func TestChangePassword_InvalidToken_Returns401(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(_ string) (*Claims, error) {
			return nil, &ServiceError{Code: "UNAUTHORIZED", Message: "invalid token", HTTPStatus: http.StatusUnauthorized}
		},
	}
	h := NewHandler(svc, nil)

	body := `{"current_password":"OldPass123!","new_password":"NewPass456!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer expired.token")
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Service Unit Tests ---

func TestHashToken_Deterministic(t *testing.T) {
	h1 := hashToken("my-secret-token")
	h2 := hashToken("my-secret-token")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64) // SHA-256 hex = 64 chars
}

func TestHashToken_DifferentInputsDifferentHashes(t *testing.T) {
	h1 := hashToken("token-a")
	h2 := hashToken("token-b")
	assert.NotEqual(t, h1, h2)
}

// #478: X-Forwarded-For is client-controlled, so the audit IP is the RemoteAddr the router resolved, never the header.
func TestClientIP_XForwardedForIsIgnored(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:4567"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.1")
	assert.Equal(t, "192.0.2.1", clientIP(req))
}

func TestClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:4567"
	req.Header.Del("X-Forwarded-For")
	assert.Equal(t, "192.0.2.1", clientIP(req))
}

// --- Service Logic Tests (using mock repository) ---

func TestService_Login_LockedAccount_Returns423BeforePasswordCheck(t *testing.T) {
	lockTime := time.Now().UTC().Add(10 * time.Minute)
	repo := &mockRepository{
		getUserByEmailFn: func(_ context.Context, email string) (*User, error) {
			return &User{
				ID:           "user-1",
				Email:        email,
				PasswordHash: "$2a$12$invalid",
				RoleName:     "employee",
				Status:       "active",
				LockedUntil:  &lockTime,
			}, nil
		},
	}

	svc := NewService(ServiceConfig{
		JWTSecret:         "a-secret-that-is-at-least-32-chars!!",
		JWTAccessTTLMin:   15,
		JWTRefreshTTLDays: 7,
		BcryptCost:        4, // low cost for tests
	}, repo)

	_, _, err := svc.Login(context.Background(), &LoginRequest{Email: "alice@example.com", Password: "anypassword"}, "127.0.0.1")

	require.Error(t, err)
	var svcErr *ServiceError
	require.ErrorAs(t, err, &svcErr)
	assert.Equal(t, "ACCOUNT_LOCKED", svcErr.Code)
	assert.Equal(t, http.StatusLocked, svcErr.HTTPStatus)
}

// TestService_Login_InactiveUser_Returns401 is the regression test for ISS-030.
// An inactive user must never receive a JWT, regardless of password correctness.
func TestService_Login_InactiveUser_Returns401(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("ValidPass123!"), 4)
	require.NoError(t, err)

	repo := &mockRepository{
		getUserByEmailFn: func(_ context.Context, email string) (*User, error) {
			return &User{
				ID:           "user-inactive-1",
				Email:        email,
				PasswordHash: string(passwordHash),
				RoleName:     "employee",
				Status:       "inactive",
			}, nil
		},
	}

	svc := NewService(ServiceConfig{
		JWTSecret:         "a-secret-that-is-at-least-32-chars!!",
		JWTAccessTTLMin:   15,
		JWTRefreshTTLDays: 7,
		BcryptCost:        4,
	}, repo)

	resp, cookie, loginErr := svc.Login(context.Background(), &LoginRequest{
		Email:    "inactive@example.com",
		Password: "ValidPass123!",
	}, "127.0.0.1")

	require.Error(t, loginErr)
	assert.Nil(t, resp)
	assert.Nil(t, cookie)

	var svcErr *ServiceError
	require.ErrorAs(t, loginErr, &svcErr)
	assert.Equal(t, "ACCOUNT_INACTIVE", svcErr.Code)
	assert.Equal(t, http.StatusUnauthorized, svcErr.HTTPStatus)
}

func TestService_Login_FifthFailureLocks(t *testing.T) {
	var lockedUntil time.Time
	var storedAttempts int

	repo := &mockRepository{
		getUserByEmailFn: func(_ context.Context, _ string) (*User, error) {
			return &User{
				ID:             "user-1",
				Email:          "alice@example.com",
				PasswordHash:   "$2a$04$notavalidhashXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
				RoleName:       "employee",
				Status:         "active",
				FailedAttempts: 4, // one more failure should lock
			}, nil
		},
		lockAccountFn: func(_ context.Context, _ string, until time.Time) error {
			lockedUntil = until
			return nil
		},
		updateFailedAttemptsFn: func(_ context.Context, _ string, attempts int) error {
			storedAttempts = attempts
			return nil
		},
	}

	svc := NewService(ServiceConfig{
		JWTSecret:       "a-secret-that-is-at-least-32-chars!!",
		JWTAccessTTLMin: 15, BcryptCost: 4,
	}, repo)

	_, _, err := svc.Login(context.Background(), &LoginRequest{Email: "alice@example.com", Password: "wrong"}, "127.0.0.1")

	require.Error(t, err)
	var svcErr *ServiceError
	require.ErrorAs(t, err, &svcErr)
	assert.Equal(t, "ACCOUNT_LOCKED", svcErr.Code)
	assert.Equal(t, 5, storedAttempts)
	assert.True(t, lockedUntil.After(time.Now().UTC()))
}

func TestService_ChangePassword_ShortPassword_Returns422(t *testing.T) {
	svc := NewService(ServiceConfig{
		JWTSecret:  "a-secret-that-is-at-least-32-chars!!",
		BcryptCost: 4,
	}, &mockRepository{})

	_, _, err := svc.ChangePassword(context.Background(), "user-1", &ChangePasswordRequest{
		CurrentPassword: "OldPass123!",
		NewPassword:     "short",
	}, "127.0.0.1")

	require.Error(t, err)
	var svcErr *ServiceError
	require.ErrorAs(t, err, &svcErr)
	// AC-8 (FR-BB64): short/weak password returns WEAK_PASSWORD 400.
	assert.Equal(t, "WEAK_PASSWORD", svcErr.Code)
	assert.Equal(t, http.StatusBadRequest, svcErr.HTTPStatus)
}

func TestService_ParseAccessToken_ValidToken(t *testing.T) {
	secret := "a-secret-that-is-at-least-32-chars!!"
	svc := NewService(ServiceConfig{
		JWTSecret:       secret,
		JWTAccessTTLMin: 15,
		BcryptCost:      4,
	}, &mockRepository{})

	// Issue a real token through the service's internal method.
	s := svc.(*service)
	user := &User{ID: "user-uuid-1", Email: "alice@example.com", RoleName: "employee"}
	token, err := s.issueAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ParseAccessToken(token)
	require.NoError(t, err)

	sub, err := claims.GetSubject()
	require.NoError(t, err)
	assert.Equal(t, "user-uuid-1", sub)
	assert.Equal(t, "alice@example.com", claims.Email)
}

func TestService_ParseAccessToken_ExpiredToken_ReturnsError(t *testing.T) {
	secret := "a-secret-that-is-at-least-32-chars!!"
	svc := NewService(ServiceConfig{JWTSecret: secret, BcryptCost: 4}, &mockRepository{})

	// Build an already-expired token manually.
	claims := Claims{
		Email: "alice@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = svc.ParseAccessToken(signed)
	assert.Error(t, err)
}

// --- Mock Repository ---

type mockRepository struct {
	RecoveryRepository // unused by these tests; recovery has its own fake (recovery_service_test.go)
	getUserByEmailFn        func(ctx context.Context, email string) (*User, error)
	getUserByIDFn           func(ctx context.Context, userID string) (*User, error)
	updateFailedAttemptsFn  func(ctx context.Context, userID string, attempts int) error
	lockAccountFn           func(ctx context.Context, userID string, until time.Time) error
	resetFailedAttemptsFn   func(ctx context.Context, userID string) error
	createRefreshTokenFn    func(ctx context.Context, token *RefreshToken) error
	getRefreshTokenByHashFn func(ctx context.Context, tokenHash string) (*RefreshToken, error)
	revokeRefreshTokenFn    func(ctx context.Context, tokenID string) error
	revokeAllUserTokensFn   func(ctx context.Context, userID string) error
	updatePasswordFn        func(ctx context.Context, userID, passwordHash string, changedAt time.Time) error
	// passwordStamp reports the account's current password_changed_at (what the repository reads under
	// its row lock); nil means never changed. Optional: a nil func means the account has no stamp.
	passwordStamp func() *time.Time
}

func (m *mockRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return nil, ErrNotFound
}

func (m *mockRepository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, userID)
	}
	return nil, ErrNotFound
}

func (m *mockRepository) UpdateFailedAttempts(ctx context.Context, userID string, attempts int) error {
	if m.updateFailedAttemptsFn != nil {
		return m.updateFailedAttemptsFn(ctx, userID, attempts)
	}
	return nil
}

func (m *mockRepository) LockAccount(ctx context.Context, userID string, until time.Time) error {
	if m.lockAccountFn != nil {
		return m.lockAccountFn(ctx, userID, until)
	}
	return nil
}

func (m *mockRepository) ResetFailedAttempts(ctx context.Context, userID string) error {
	if m.resetFailedAttemptsFn != nil {
		return m.resetFailedAttemptsFn(ctx, userID)
	}
	return nil
}

func (m *mockRepository) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	if m.createRefreshTokenFn != nil {
		return m.createRefreshTokenFn(ctx, token)
	}
	return nil
}

func (m *mockRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	if m.getRefreshTokenByHashFn != nil {
		return m.getRefreshTokenByHashFn(ctx, tokenHash)
	}
	return nil, ErrNotFound
}

func (m *mockRepository) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	if m.revokeRefreshTokenFn != nil {
		return m.revokeRefreshTokenFn(ctx, tokenID)
	}
	return nil
}

func (m *mockRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	if m.revokeAllUserTokensFn != nil {
		return m.revokeAllUserTokensFn(ctx, userID)
	}
	return nil
}

// UpdatePassword stamps the next password_changed_at from the account's current stamp, as the real
// repository does under its row lock, and reports the stamp it wrote.
func (m *mockRepository) UpdatePassword(ctx context.Context, userID, passwordHash string, appNow time.Time) (time.Time, error) {
	var previous *time.Time
	if m.passwordStamp != nil {
		previous = m.passwordStamp()
	}
	stamp := NextPasswordStamp(appNow, previous)
	if m.updatePasswordFn != nil {
		return stamp, m.updatePasswordFn(ctx, userID, passwordHash, stamp)
	}
	return stamp, nil
}
