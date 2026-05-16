package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
)

func TestRequestID_AssignsUUID(t *testing.T) {
	var capturedID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = appmw.GetRequestID(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	appmw.RequestID(inner).ServeHTTP(rec, req)

	if capturedID == "" || capturedID == "-" {
		t.Fatalf("expected a non-empty UUID in context, got %q", capturedID)
	}
}

func TestRequestID_SetsHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	appmw.RequestID(inner).ServeHTTP(rec, req)

	hdr := rec.Header().Get("X-Request-ID")
	if hdr == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
}

func TestRequestID_HeaderMatchesContext(t *testing.T) {
	var contextID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = appmw.GetRequestID(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	appmw.RequestID(inner).ServeHTTP(rec, req)

	headerID := rec.Header().Get("X-Request-ID")
	if contextID != headerID {
		t.Errorf("context ID %q != header ID %q", contextID, headerID)
	}
}

func TestGetRequestID_FallbackWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := appmw.GetRequestID(req.Context())
	if id != "-" {
		t.Errorf("expected fallback '-', got %q", id)
	}
}
