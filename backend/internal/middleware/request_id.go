// Package middleware provides HTTP middleware for structured logging,
// request correlation, and panic recovery.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

// RequestIDKey is the context key under which the request UUID is stored.
const RequestIDKey contextKey = "request_id"

// RequestID is a middleware that generates a UUID v4 for every incoming request,
// stores it in the request context under RequestIDKey, and echoes it back in the
// X-Request-ID response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID stored in ctx by RequestID middleware.
// Returns "-" when no ID is found (e.g. in tests that do not use the middleware).
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "-"
}
