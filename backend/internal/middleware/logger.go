package middleware

import (
	"context"
	"net"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// loggerContextKey is the context key under which the request-scoped logger is stored.
const loggerContextKey contextKey = "logger"

// userIDHolderKey is the context key under which the mutable userIDHolder is stored.
const userIDHolderKey contextKey = "user_id_holder"

// userIDHolder is a mutable slot that auth middleware writes the authenticated
// user ID into after validating the JWT. Because it is a pointer stored in the
// context before the handler chain runs, every downstream handler (including auth)
// can write to it and the logger middleware will see the updated value when the
// chain unwinds, even though Go contexts are copy-on-write.
type userIDHolder struct{ id string }

// WithUserIDHolder injects a new *userIDHolder into ctx and returns the enriched
// context together with a pointer to the holder. Call this once in the logger
// middleware, then pass ctx to downstream handlers.
func WithUserIDHolder(ctx context.Context) (context.Context, *userIDHolder) {
	h := &userIDHolder{id: "-"}
	return context.WithValue(ctx, userIDHolderKey, h), h
}

// SetUserIDInHolder writes uid into the userIDHolder that the logger middleware
// stored in ctx. Auth middleware calls this after successfully validating the JWT.
// If ctx carries no holder (e.g. in tests that skip the logger middleware) the
// call is a no-op.
func SetUserIDInHolder(ctx context.Context, uid string) {
	if h, ok := ctx.Value(userIDHolderKey).(*userIDHolder); ok {
		h.id = uid
	}
}

// RequestLogger returns a middleware that logs one structured JSON line per request.
// It captures the HTTP status code by wrapping the ResponseWriter, extracts the
// authenticated user ID (if any) from the context via a mutable holder that auth
// middleware populates, and stores a request-scoped logger (with request_id pre-set)
// back into the context so downstream handlers, services, and repositories can log
// with the same correlation ID automatically.
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			reqID := GetRequestID(r.Context())

			// Enrich the logger with request_id so every log call within this
			// request automatically carries the correlation field.
			reqLogger := logger.With().Str("request_id", reqID).Logger()

			// Inject both the request-scoped logger and a mutable userIDHolder into
			// the context. Auth middleware (which runs downstream) will call
			// SetUserIDInHolder once it extracts the JWT subject, allowing the logger
			// to record the correct user_id after the full handler chain completes.
			ctx := context.WithValue(r.Context(), loggerContextKey, reqLogger)
			ctx, holder := WithUserIDHolder(ctx)

			next.ServeHTTP(ww, r.WithContext(ctx))

			// holder.id was set by auth middleware (if the request was authenticated).
			reqLogger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status_code", ww.Status()).
				Int64("latency_ms", time.Since(start).Milliseconds()).
				Str("user_id", holder.id).
				Str("ip", realIP(r)).
				Msg("request")
		})
	}
}

// LoggerFromContext retrieves the request-scoped zerolog.Logger stored by
// RequestLogger middleware.  Falls back to the global zerolog.Logger when the
// context does not carry one (e.g. in unit tests that do not use the middleware).
func LoggerFromContext(ctx context.Context) zerolog.Logger {
	if l, ok := ctx.Value(loggerContextKey).(zerolog.Logger); ok {
		return l
	}
	return zerolog.Ctx(ctx).With().Logger()
}

// realIP returns the best-guess client IP address.  It prefers X-Forwarded-For
// (set by the Nginx reverse proxy) over RemoteAddr.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may be a comma-separated list; the first entry is the client.
		if i := len(xff); i > 0 {
			for j := 0; j < i; j++ {
				if xff[j] == ',' {
					return xff[:j]
				}
			}
			return xff
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
