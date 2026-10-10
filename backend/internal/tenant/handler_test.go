package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHandler wires up a Handler with a mock repository pre-loaded with seed data.
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	repo := newMockRepository(defaultSeedData())
	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))
	return NewHandler(svc, nil)
}

// TestGetConfig_Returns200WithPublicKeys verifies the GET /tenant/config response.
func TestGetConfig_Returns200WithPublicKeys(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant/config", nil)
	w := httptest.NewRecorder()

	h.GetConfig(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var env struct {
		Data  map[string]json.RawMessage `json:"data"`
		Error *apiError                  `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	assert.Nil(t, env.Error)
	assert.Contains(t, env.Data, "app_name")
	assert.Contains(t, env.Data, "primary_color")
	assert.Contains(t, env.Data, "accent_color")
	assert.Contains(t, env.Data, "default_locale")
	assert.Contains(t, env.Data, "available_locales")
	assert.NotContains(t, env.Data, "logo")
}

// TestGetLogo_Returns204WhenNoLogo verifies that a null logo returns 204.
func TestGetLogo_Returns204WhenNoLogo(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant/logo", nil)
	w := httptest.NewRecorder()

	h.GetLogo(w, req)

	assert.Equal(t, http.StatusNoContent, w.Result().StatusCode)
}

// TestGetLogo_CacheControlHeader verifies that when a logo is present the response
// includes Cache-Control: public, max-age=3600 (AC-5, FR-BB65).
func TestGetLogo_CacheControlHeader(t *testing.T) {
	// Build a minimal valid data-URI PNG (1×1 PNG magic bytes, base64-encoded).
	// The upload validator checks the first bytes; use a real PNG header.
	// For the unit test we bypass the upload validator by setting the logo
	// directly in the mock repository rather than going through UpdateConfig.
	repo := newMockRepository(defaultSeedData())
	// Use a tiny valid-ish base64 blob — the handler only decodes, it does not
	// re-validate the MIME type.
	logoVal, _ := json.Marshal("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	repo.(*mockRepository).data["logo"] = logoVal

	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant/logo", nil)
	w := httptest.NewRecorder()
	h.GetLogo(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "public, max-age=3600", resp.Header.Get("Cache-Control"))
}

// TestUpdateConfig_AcceptsPartialUpdate verifies that PUT updates the supplied keys.
func TestUpdateConfig_AcceptsPartialUpdate(t *testing.T) {
	h := newTestHandler(t)

	body := `{"app_name": "Acme Corp", "primary_color": "#1d4ed8"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateConfig(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var env struct {
		Data  map[string]json.RawMessage `json:"data"`
		Error *apiError                  `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	assert.Nil(t, env.Error)

	var updatedKeys []string
	require.NoError(t, json.Unmarshal(env.Data["updated"], &updatedKeys))
	assert.ElementsMatch(t, []string{"app_name", "primary_color"}, updatedKeys)
}

// TestUpdateConfig_InvalidLocale verifies that a mismatched default_locale returns 400.
func TestUpdateConfig_InvalidLocale(t *testing.T) {
	h := newTestHandler(t)

	body := `{"default_locale": "fr"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateConfig(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var env struct {
		Data  json.RawMessage `json:"data"`
		Error *apiError       `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "INVALID_LOCALE", env.Error.Code)
}

// TestUpdateConfig_InvalidBody verifies that malformed JSON returns 400.
func TestUpdateConfig_InvalidBody(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant/config", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

// TestUpdateConfig_LowContrastPrimaryReturns422 verifies FR-BB320 AC-1 at the HTTP layer:
// a primary colour below 4.5:1 against white is refused with 422 VALIDATION_ERROR.
func TestUpdateConfig_LowContrastPrimaryReturns422(t *testing.T) {
	h := newTestHandler(t)

	body := `{"primary_color": "#0ea5e9"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateConfig(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var env struct {
		Data  json.RawMessage `json:"data"`
		Error *apiError       `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	assert.Equal(t, "null", string(env.Data))
	require.NotNil(t, env.Error)
	assert.Equal(t, "VALIDATION_ERROR", env.Error.Code)
	assert.Contains(t, env.Error.Message, "primary_color")
}

// TestUpdateConfig_PassingPrimaryReturns200 verifies that a colour reaching 4.5:1 is accepted.
func TestUpdateConfig_PassingPrimaryReturns200(t *testing.T) {
	h := newTestHandler(t)

	body := `{"primary_color": "#2E6DB4"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenant/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
