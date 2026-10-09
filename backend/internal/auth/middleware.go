package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// AuthOption configures Authenticate.
type AuthOption func(*authOptions)

type authOptions struct{ state AccountStateLookup }

// WithAccountState makes Authenticate consult the user's AccountState on every request:
//   - access tokens issued before the user's last password reset are rejected
//     (users.password_changed_at; tokens without iat are rejected) — ISS-105;
//   - while users.force_password_change is true, every route except
//     forcePasswordChangeAllowedPaths answers 403 PASSWORD_CHANGE_REQUIRED — ISS-160.
//
// Lookup failures fail closed.
func WithAccountState(l AccountStateLookup) AuthOption {
	return func(c *authOptions) { c.state = l }
}

// forcePasswordChangeAllowedPaths are the only authenticated routes reachable while
// force_password_change is set: changing the password, and reading the own profile
// (the SPA needs it to render the change-password page). /auth/login, /auth/refresh and
// /auth/logout are public (cookie-based) routes and never pass through Authenticate.
var forcePasswordChangeAllowedPaths = map[string]bool{
	"/api/v1/auth/change-password": true,
	"/api/v1/users/me":             true,
}

// forceChangeAllowed reports whether r may proceed for a user who must change password.
// GET /users/me only; change-password for any method (the route itself is POST-only).
func forceChangeAllowed(r *http.Request) bool {
	if !forcePasswordChangeAllowedPaths[r.URL.Path] {
		return false
	}
	return r.URL.Path != "/api/v1/users/me" || r.Method == http.MethodGet
}

// Authenticate validates a Bearer JWT and injects user claims into the context.
// It does NOT inject tenant context — that is handled by TenantContext().
func Authenticate(jwtSecret string, opts ...AuthOption) func(http.Handler) http.Handler {
	var cfg authOptions
	for _, o := range opts {
		o(&cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header required")
				return
			}
			if !strings.HasPrefix(header, "Bearer ") {
				api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "malformed authorization header")
				return
			}
			raw := strings.TrimPrefix(header, "Bearer ")
			claims, err := parseJWT(raw, jwtSecret)
			if err != nil {
				if isExpired(err) {
					api.WriteError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired")
				} else {
					api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
				}
				return
			}

			// Safe claim extraction — return INVALID_TOKEN instead of panicking.
			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
				return
			}
			role, ok := claims["role"].(string)
			if !ok {
				api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
				return
			}
			deptID, _ := claims["department_id"].(string) // optional; defaults to ""

			if cfg.state != nil {
				iat, ok := claims["iat"].(float64)
				if !ok {
					api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
					return
				}
				st, err := cfg.state(r.Context(), userID)
				if err != nil {
					if errors.Is(err, ErrNotFound) {
						api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
					} else {
						// Fail closed: without the state we cannot prove the token is current
						// or that the user is not required to change their password.
						api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not verify session")
					}
					return
				}
				if tokenPredatesPasswordChange(int64(iat), st.PasswordChangedAt) {
					api.WriteError(w, http.StatusUnauthorized, "TOKEN_REVOKED", "password was changed; please sign in again")
					return
				}
				if st.ForcePasswordChange && !forceChangeAllowed(r) {
					api.WriteError(w, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "you must change your password before continuing")
					return
				}
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxkeys.CtxUserID, userID)
			ctx = context.WithValue(ctx, ctxkeys.CtxRole, role)
			ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, deptID)

			// Populate the logger middleware's mutable holder so that the request
			// log line records the correct user_id even though the logger wraps this
			// middleware from the outside and cannot see values set on child contexts.
			appmw.SetUserIDInHolder(ctx, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseJWT(raw, secret string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	}, jwt.WithLeeway(5*time.Second))
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func isExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}
