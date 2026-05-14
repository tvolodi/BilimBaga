package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantContext(t *testing.T) {
	t.Run("AC-5: injects tenant_id=public for every request", func(t *testing.T) {
		var captured *http.Request
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured = r
			w.WriteHeader(http.StatusOK)
		})

		handler := TenantContext()(next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotNil(t, captured)
		assert.Equal(t, "public", TenantIDFromCtx(captured.Context()))
	})

	t.Run("AC-5: TenantIDFromCtx returns public even for unauthenticated requests", func(t *testing.T) {
		var tenantID string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID = TenantIDFromCtx(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := TenantContext()(next)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, "public", tenantID)
	})
}
