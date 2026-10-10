package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ServiceConfig holds the configuration values needed by the auth service.
type ServiceConfig struct {
	JWTSecret         string
	JWTAccessTTLMin   int
	JWTRefreshTTLDays int
	BcryptCost        int
	CookieDomain      string
	CookieSecure      bool
}

// Service defines the business-logic interface for authentication.
type Service interface {
	Login(ctx context.Context, req *LoginRequest, ipAddr string) (*LoginResponse, *http.Cookie, error)
	Refresh(ctx context.Context, rawToken, ipAddr string) (*RefreshResponse, *http.Cookie, error)
	Logout(ctx context.Context, rawToken, ipAddr string) (*http.Cookie, error)
	ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest, ipAddr string) (*ChangePasswordResponse, *http.Cookie, error)
	ParseAccessToken(tokenString string) (*Claims, error)
	RecoveryService
}

type service struct {
	cfg    ServiceConfig
	repo   Repository
	mailer ResetMailer
	now    func() time.Time
	logger *slog.Logger
}

// NewService creates a new Service backed by the given Repository.
// An optional ResetMailer enables password recovery emails (FR-BB115); without it,
// recovery tokens are still issued but nothing is sent.
func NewService(cfg ServiceConfig, repo Repository, mailer ...ResetMailer) Service {
	s := &service{cfg: cfg, repo: repo, now: func() time.Time { return time.Now().UTC() }, logger: slog.Default()}
	if len(mailer) > 0 {
		s.mailer = mailer[0]
	}
	return s
}

// Login validates credentials and issues a JWT access token plus an httpOnly refresh cookie.
// AC-4: increments failed_attempts on bad password; locks after 5 failures.
// AC-5: locked accounts are rejected before password validation.
func (s *service) Login(ctx context.Context, req *LoginRequest, ipAddr string) (*LoginResponse, *http.Cookie, error) {
	user, err := s.repo.GetUserByEmail(ctx, api.NormalizeEmail(req.Email))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Run a dummy bcrypt comparison to prevent user-enumeration via timing.
			_ = bcrypt.CompareHashAndPassword(
				[]byte("$2a$12$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
				[]byte(req.Password),
			)
			return nil, nil, &ServiceError{
				Code:       "INVALID_CREDENTIALS",
				Message:    "invalid email or password",
				HTTPStatus: http.StatusUnauthorized,
			}
		}
		return nil, nil, fmt.Errorf("auth.service.Login: look up user: %w", err)
	}

	// AC-5: check lockout before verifying the password.
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now().UTC()) {
		return nil, nil, &ServiceError{
			Code:       "ACCOUNT_LOCKED",
			Message:    fmt.Sprintf("account locked, try again after %s", user.LockedUntil.Format(time.RFC3339)),
			HTTPStatus: http.StatusLocked,
		}
	}

	// Reject inactive accounts before password verification.
	if user.Status != "active" {
		return nil, nil, &ServiceError{
			Code:       "ACCOUNT_INACTIVE",
			Message:    "account is inactive",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	// Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// AC-4: increment failure counter; lock if threshold reached.
		newAttempts := user.FailedAttempts + 1
		if newAttempts >= 5 {
			lockUntil := time.Now().UTC().Add(30 * time.Minute)
			_ = s.repo.LockAccount(ctx, user.ID, lockUntil)
			_ = s.repo.UpdateFailedAttempts(ctx, user.ID, newAttempts)
			return nil, nil, &ServiceError{
				Code:       "ACCOUNT_LOCKED",
				Message:    fmt.Sprintf("account locked, try again after %s", lockUntil.Format(time.RFC3339)),
				HTTPStatus: http.StatusLocked,
			}
		}
		_ = s.repo.UpdateFailedAttempts(ctx, user.ID, newAttempts)
		return nil, nil, &ServiceError{
			Code:       "INVALID_CREDENTIALS",
			Message:    "invalid email or password",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	// Successful authentication.
	_ = s.repo.ResetFailedAttempts(ctx, user.ID)

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.Login: issue access token: %w", err)
	}

	cookie, err := s.issueRefreshCookie(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.Login: issue refresh cookie: %w", err)
	}

	return &LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   s.cfg.JWTAccessTTLMin * 60,
		User: UserInfo{
			ID:                  user.ID,
			FullName:            user.FullName,
			Email:               user.Email,
			Role:                user.RoleName,
			ForcePasswordChange: user.ForcePasswordChange,
		},
	}, cookie, nil
}

// Refresh validates the refresh cookie, rotates the token pair, and returns new credentials.
// AC-11: a revoked token triggers revocation of ALL user tokens.
func (s *service) Refresh(ctx context.Context, rawToken, ipAddr string) (*RefreshResponse, *http.Cookie, error) {
	tokenHash := hashToken(rawToken)

	record, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, &ServiceError{
				Code:       "INVALID_REFRESH_TOKEN",
				Message:    "refresh token is invalid or expired",
				HTTPStatus: http.StatusUnauthorized,
			}
		}
		return nil, nil, fmt.Errorf("auth.service.Refresh: look up token: %w", err)
	}

	// AC-11: detect reuse of a revoked token.
	if record.RevokedAt != nil {
		_ = s.repo.RevokeAllUserRefreshTokens(ctx, record.UserID)
		return nil, nil, &ServiceError{
			Code:       "INVALID_REFRESH_TOKEN",
			Message:    "refresh token is invalid or expired",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	if record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, nil, &ServiceError{
			Code:       "INVALID_REFRESH_TOKEN",
			Message:    "refresh token is invalid or expired",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	// Revoke the old token before issuing a new one.
	if err := s.repo.RevokeRefreshToken(ctx, record.ID); err != nil {
		return nil, nil, fmt.Errorf("auth.service.Refresh: revoke old token: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, record.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.Refresh: get user: %w", err)
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.Refresh: issue access token: %w", err)
	}

	newCookie, err := s.issueRefreshCookie(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.Refresh: issue refresh cookie: %w", err)
	}

	return &RefreshResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   s.cfg.JWTAccessTTLMin * 60,
	}, newCookie, nil
}

// Logout revokes the refresh token and returns a cookie that clears the browser value.
// AC-7: returns success even if no cookie was present.
func (s *service) Logout(ctx context.Context, rawToken, ipAddr string) (*http.Cookie, error) {
	clearCookie := s.clearRefreshCookie()

	if rawToken == "" {
		return clearCookie, nil
	}

	tokenHash := hashToken(rawToken)
	record, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return clearCookie, nil
		}
		return clearCookie, fmt.Errorf("auth.service.Logout: look up token: %w", err)
	}

	if record.RevokedAt == nil {
		_ = s.repo.RevokeRefreshToken(ctx, record.ID)
	}

	return clearCookie, nil
}

// ChangePassword validates the current password, enforces complexity constraints,
// hashes and stores the new password, and clears force_password_change.
//
// ISS-171: the change also stamps users.password_changed_at, which revokes every access
// token issued earlier (other devices, a stolen token) and every refresh token of the user.
// To keep the caller signed in it then issues a fresh access token and refresh cookie.
//
// Boundary semantics (see tokenPredatesPasswordChange): a token is rejected only when
// iat < floor(password_changed_at) at second resolution. The stamp is taken from the
// application clock (not the database's now()) and the new token's iat is that same
// instant, so iat == floor(stamp) always holds for the new token and it is accepted
// immediately, even when issued in the same second as the stamp. The flip side is that an
// OLD token issued within that same second also survives (known second-granularity caveat).
func (s *service) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest, ipAddr string) (*ChangePasswordResponse, *http.Cookie, error) {
	// AC-8 (FR-BB64): enforce password complexity.
	if err := ValidateComplexity(req.NewPassword); err != nil {
		return nil, nil, &ServiceError{
			Code:       "WEAK_PASSWORD",
			Message:    "password must be at least 8 characters and contain uppercase, lowercase, and a digit",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.ChangePassword: get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return nil, nil, &ServiceError{
			Code:       "INVALID_CREDENTIALS",
			Message:    "current password is incorrect",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	// AC-9: bcrypt cost 12.
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.cfg.BcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.ChangePassword: hash new password: %w", err)
	}

	// Stamp and the new token's iat share one instant (truncated to seconds by the JWT lib).
	changedAt := s.now().Truncate(time.Second)
	if err := s.repo.UpdatePassword(ctx, userID, string(newHash), changedAt); err != nil {
		return nil, nil, fmt.Errorf("auth.service.ChangePassword: update password: %w", err)
	}

	accessToken, err := s.issueAccessTokenAt(user, changedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.ChangePassword: issue access token: %w", err)
	}
	cookie, err := s.issueRefreshCookie(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.service.ChangePassword: issue refresh cookie: %w", err)
	}

	return &ChangePasswordResponse{
		Message:     "password changed",
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   s.cfg.JWTAccessTTLMin * 60,
	}, cookie, nil
}

// ParseAccessToken validates a JWT access token string and returns its claims.
func (s *service) ParseAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth.ParseAccessToken: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth.ParseAccessToken: invalid claims")
	}
	return claims, nil
}

// issueAccessToken generates a signed JWT for the given user.
func (s *service) issueAccessToken(user *User) (string, error) {
	return s.issueAccessTokenAt(user, issueTime(s.now().UTC(), user.PasswordChangedAt))
}

// issueTime is the iat of a token issued at now. It is never before the account's password stamp, so a
// sign-in or refresh in the second a reset stamped (the next second) is not revoked from birth (#439).
// A stamp at or before now leaves the clock's own second, which the epoch check accepts.
func issueTime(now time.Time, changedAt *time.Time) time.Time {
	if changedAt != nil && changedAt.After(now) {
		return *changedAt
	}
	return now
}

// issueAccessTokenAt is issueAccessToken with an explicit issue time (ISS-171).
func (s *service) issueAccessTokenAt(user *User, now time.Time) (string, error) {
	claims := Claims{
		Email:        user.Email,
		Role:         user.RoleName,
		DepartmentID: user.DepartmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWTAccessTTLMin) * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// issueRefreshCookie generates a 32-byte random token, stores its SHA-256 hash in the DB,
// and returns an httpOnly cookie carrying the raw base64url-encoded token.
func (s *service) issueRefreshCookie(ctx context.Context, userID string) (*http.Cookie, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token bytes: %w", err)
	}
	rawToken := base64.URLEncoding.EncodeToString(rawBytes)
	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().UTC().Add(time.Duration(s.cfg.JWTRefreshTTLDays) * 24 * time.Hour)

	if err := s.repo.CreateRefreshToken(ctx, &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    rawToken,
		Path:     "/api/v1/auth",
		MaxAge:   s.cfg.JWTRefreshTTLDays * 24 * 3600,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
	if s.cfg.CookieDomain != "" && s.cfg.CookieDomain != "localhost" {
		cookie.Domain = s.cfg.CookieDomain
	}
	return cookie, nil
}

// clearRefreshCookie returns a cookie that instructs the browser to delete the refresh_token.
func (s *service) clearRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
}

// hashToken returns the hex-encoded SHA-256 hash of a token string.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
