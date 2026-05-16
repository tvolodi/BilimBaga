package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/rs/zerolog"
)

// Recovery returns a middleware that catches any panic from downstream handlers,
// logs the full stack trace at error level (with the request_id correlation field),
// and writes a generic HTTP 500 response.
//
// The stack trace is NEVER included in the response body — only in the log output.
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rv := recover(); rv != nil {
					logger.Error().
						Str("request_id", GetRequestID(r.Context())).
						Interface("panic", rv).
						Str("stack", string(debug.Stack())).
						Msg("panic recovered")

					// Flush any partial response headers and return a clean 500.
					api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
