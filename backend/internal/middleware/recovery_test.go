package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/rs/zerolog"
)

func TestRecovery_CatchesPanic_Returns500(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	appmw.RequestID(appmw.Recovery(logger)(panicking)).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestRecovery_ResponseBodyContainsNoStack(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("secret internal error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	appmw.RequestID(appmw.Recovery(logger)(panicking)).ServeHTTP(rec, req)

	body := rec.Body.String()
	// Stack traces contain "goroutine" keyword — must not appear in the response.
	if strings.Contains(body, "goroutine") {
		t.Errorf("response body must not contain stack trace, got: %s", body)
	}
	// The panic value must not be echoed in the response body.
	if strings.Contains(body, "secret internal error") {
		t.Errorf("response body must not contain the panic value, got: %s", body)
	}
}

func TestRecovery_ResponseBodyIsStandardError(t *testing.T) {
	logger := zerolog.New(bytes.NewBuffer(nil))

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	appmw.RequestID(appmw.Recovery(logger)(panicking)).ServeHTTP(rec, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	if resp["data"] != nil {
		t.Errorf("expected data=null, got %v", resp["data"])
	}
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object, got %v", resp["error"])
	}
	if errObj["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected code=INTERNAL_ERROR, got %v", errObj["code"])
	}
}

func TestRecovery_LogsStackTrace(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("log me")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	appmw.RequestID(appmw.Recovery(logger)(panicking)).ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Fatal("expected a log line, got nothing")
	}
	var entry map[string]interface{}
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log line is not JSON: %v", err)
	}
	if entry["level"] != "error" {
		t.Errorf("expected level=error, got %v", entry["level"])
	}
	if _, ok := entry["stack"]; !ok {
		t.Error("expected 'stack' field in log entry")
	}
	if _, ok := entry["request_id"]; !ok {
		t.Error("expected 'request_id' field in log entry")
	}
}

func TestRecovery_NoPanicPassesThrough(t *testing.T) {
	logger := zerolog.New(bytes.NewBuffer(nil))

	normal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	appmw.Recovery(logger)(normal).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-panicking handler, got %d", rec.Code)
	}
}
