package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/rs/zerolog"
)

func TestRequestLogger_ProducesLogLine(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	// Apply RequestID first so the log line carries a real request_id.
	appmw.RequestID(appmw.RequestLogger(logger)(inner)).ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Fatal("expected a log line to be written, got nothing")
	}

	var entry map[string]interface{}
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v — raw: %s", err, buf.String())
	}

	requiredFields := []string{"method", "path", "status_code", "latency_ms", "user_id", "ip", "request_id"}
	for _, f := range requiredFields {
		if _, ok := entry[f]; !ok {
			t.Errorf("missing required log field %q in: %v", f, entry)
		}
	}
	if entry["method"] != http.MethodGet {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
	if entry["path"] != "/api/v1/test" {
		t.Errorf("expected path /api/v1/test, got %v", entry["path"])
	}
	if entry["status_code"] != float64(http.StatusOK) {
		t.Errorf("expected status_code 200, got %v", entry["status_code"])
	}
	// Anonymous request — user_id must be "-".
	if entry["user_id"] != "-" {
		t.Errorf("expected user_id '-' for unauthenticated request, got %v", entry["user_id"])
	}
}

func TestRequestLogger_RequestIDPropagated(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/some/path", nil)
	rec := httptest.NewRecorder()

	appmw.RequestID(appmw.RequestLogger(logger)(inner)).ServeHTTP(rec, req)

	headerID := rec.Header().Get("X-Request-ID")

	var entry map[string]interface{}
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v", err)
	}
	if entry["request_id"] != headerID {
		t.Errorf("log request_id %v != header X-Request-ID %v", entry["request_id"], headerID)
	}
}

func TestLoggerFromContext_FallsBackToGlobal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Must not panic when no logger is stored in context.
	_ = appmw.LoggerFromContext(req.Context())
}

// TestRequestLogger_AuthenticatedUserIDLogged verifies that when a downstream
// middleware (simulating auth.Authenticate) calls SetUserIDInHolder, the logged
// user_id field reflects the authenticated user rather than "-".
func TestRequestLogger_AuthenticatedUserIDLogged(t *testing.T) {
	const wantUserID = "user-abc-123"

	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Simulate auth middleware: reads the holder from context and sets the user ID.
	authSimulator := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			appmw.SetUserIDInHolder(r.Context(), wantUserID)
			next.ServeHTTP(w, r)
		})
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()

	// Chain: RequestID → RequestLogger → authSimulator → inner
	appmw.RequestID(appmw.RequestLogger(logger)(authSimulator(inner))).ServeHTTP(rec, req)

	var entry map[string]interface{}
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v — raw: %s", err, buf.String())
	}

	if entry["user_id"] != wantUserID {
		t.Errorf("expected user_id %q for authenticated request, got %v", wantUserID, entry["user_id"])
	}
}
