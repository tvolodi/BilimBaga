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

// decodeResponse unmarshals the JSON response body into the standard envelope.
func decodeResponse(t *testing.T, body *strings.Reader) (data json.RawMessage, apiErr *apiError) {
	t.Helper()
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error *apiError       `json:"error"`
	}
	require.NoError(t, json.NewDecoder(body).Decode(&env))
	return env.Data, env.Error
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
