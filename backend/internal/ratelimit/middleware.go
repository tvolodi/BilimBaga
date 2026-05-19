// Package ratelimit provides HTTP middleware for rate limiting.
// Limits are applied per-IP globally and per-session for sensitive answer-save routes.
package ratelimit

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

const retryAfterSeconds = 60

// disabled returns true when the DISABLE_RATE_LIMIT env variable is set to "true" or "1".
// This is intended for E2E test environments only.
func disabled() bool {
	v := os.Getenv("DISABLE_RATE_LIMIT")
	return v == "true" || v == "1"
}

// noopMiddleware passes the request straight through without limiting.
func noopMiddleware(next http.Handler) http.Handler { return next }

// rateLimitedResponse is the standard error envelope returned on 429 responses.
type rateLimitedResponse struct {
	Data  any               `json:"data"`
	Error rateLimitedError  `json:"error"`
}

type rateLimitedError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// limitHandler is the custom 429 response writer shared by all rate limiters.
func limitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(rateLimitedResponse{
		Data: nil,
		Error: rateLimitedError{
			Code:    "RATE_LIMITED",
			Message: "Too many requests",
		},
	})
}

// AuthLimiter returns a middleware that allows 10 requests per minute per IP.
// It is intended for authentication endpoints.
func AuthLimiter() func(http.Handler) http.Handler {
	if disabled() {
		return func(next http.Handler) http.Handler { return noopMiddleware(next) }
	}
	return httprate.Limit(
		10,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(limitHandler),
	)
}

// GlobalLimiter returns a middleware that allows 300 requests per minute per IP.
// It is intended for all general API endpoints.
func GlobalLimiter() func(http.Handler) http.Handler {
	if disabled() {
		return func(next http.Handler) http.Handler { return noopMiddleware(next) }
	}
	return httprate.Limit(
		300,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(limitHandler),
	)
}

// AnswerSaveLimiter returns a middleware that allows 60 requests per minute,
// keyed by the session ID extracted from the URL parameter "id".
// It is intended for the answer-save endpoint: PUT /portal/sessions/{id}/answers/{questionId}.
func AnswerSaveLimiter() func(http.Handler) http.Handler {
	return httprate.Limit(
		60,
		time.Minute,
		httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
			sessionID := chi.URLParam(r, "id")
			if sessionID == "" {
				// Fall back to IP if session ID is not in the URL.
				return httprate.KeyByIP(r)
			}
			return "session:" + sessionID, nil
		}),
		httprate.WithLimitHandler(limitHandler),
	)
}
