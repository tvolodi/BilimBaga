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

type authOptions struct{ epoch PasswordChangedLookup }

// WithPasswordEpoch makes Authenticate reject access tokens issued before the user's
// password was last reset (users.password_changed_at). Tokens without iat are rejected.
func WithPasswordEpoch(l PasswordChangedLookup) AuthOption {
	return func(c *authOptions) { c.epoch = l }
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

			if cfg.epoch != nil {
				iat, ok := claims["iat"].(float64)
				if !ok {
					api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
					return
				}
				changedAt, err := cfg.epoch(r.Context(), userID)
				if err != nil {
					if errors.Is(err, ErrNotFound) {
						api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
					} else {
						// Fail closed: without the epoch we cannot prove the token is current.
						api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not verify session")
					}
					return
				}
				if tokenPredatesPasswordChange(int64(iat), changedAt) {
					api.WriteError(w, http.StatusUnauthorized, "TOKEN_REVOKED", "password was changed; please sign in again")
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
