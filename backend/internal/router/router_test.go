package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/rs/zerolog"
)

// newRouter builds the full route tree with nil handlers. Handlers are only
// dereferenced when a route executes, so routes that are rejected by middleware
// (auth, heartbeat) can be exercised without a database.
func newRouter() http.Handler {
	return router.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"test-secret", nil, nil, "test", zerolog.Nop())
}

func TestRouter_HeartbeatPing(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/ping status = %d; want 200", rec.Code)
	}
}

func TestRouter_ProtectedRoutesRequireBearerToken(t *testing.T) {
	h := newRouter()
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/change-password"},
		{http.MethodPut, "/api/v1/tenant/config"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d; want 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestRouter_UnknownRouteIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404", rec.Code)
	}
}
