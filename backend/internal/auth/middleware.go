package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/golang-jwt/jwt/v5"
)

// Authenticate validates a Bearer JWT and injects user claims into the context.
// It does NOT inject tenant context — that is handled by TenantContext().
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
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

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxUserID, userID)
			ctx = context.WithValue(ctx, ctxRole, role)
			ctx = context.WithValue(ctx, ctxDepartmentID, deptID)

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
