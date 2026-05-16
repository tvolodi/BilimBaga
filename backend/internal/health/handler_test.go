package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/health"
)

// okPinger simulates a healthy database connection.
type okPinger struct{}

func (p *okPinger) PingContext(_ context.Context) error { return nil }

// failPinger simulates an unreachable database.
type failPinger struct{}

func (p *failPinger) PingContext(_ context.Context) error {
	return errors.New("connection refused")
}

const testVersion = "abc1234"

func TestHandler_DBOk_Returns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	health.Handler(&okPinger{}, testVersion)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'data' to be an object, got: %v", body["data"])
	}
	if data["status"] != "ok" {
		t.Errorf("expected data.status='ok', got %v", data["status"])
	}
	if data["db_ok"] != true {
		t.Errorf("expected data.db_ok=true, got %v", data["db_ok"])
	}
	if data["version"] != testVersion {
		t.Errorf("expected data.version=%q, got %v", testVersion, data["version"])
	}
	if body["error"] != nil {
		t.Errorf("expected error to be null, got %v", body["error"])
	}
}

func TestHandler_DBFail_Returns503(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	health.Handler(&failPinger{}, testVersion)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'data' to be an object, got: %v", body["data"])
	}
	if data["status"] != "degraded" {
		t.Errorf("expected data.status='degraded', got %v", data["status"])
	}
	if data["db_ok"] != false {
		t.Errorf("expected data.db_ok=false, got %v", data["db_ok"])
	}
	if data["version"] != testVersion {
		t.Errorf("expected data.version=%q, got %v", testVersion, data["version"])
	}
	if body["error"] != nil {
		t.Errorf("expected error to be null, got %v", body["error"])
	}
}

func TestHandler_ContentTypeJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	health.Handler(&okPinger{}, testVersion)(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}
