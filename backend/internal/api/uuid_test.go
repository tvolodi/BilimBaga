package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

const validUUID = "3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22"

func TestIsUUID(t *testing.T) {
	assert.True(t, api.IsUUID(validUUID))
	for _, bad := range []string{"", "abc", "super_admin", "u1", "urn:uuid:" + validUUID, "{" + validUUID + "}", "3f2b8c1e9d4a4b6e8a1f0c7d5e9a1b22"} {
		assert.False(t, api.IsUUID(bad), bad)
	}
}

func TestUUIDQuery(t *testing.T) {
	w := httptest.NewRecorder()
	v, ok := api.UUIDQuery(w, httptest.NewRequest(http.MethodGet, "/x", nil), "department_id")
	assert.True(t, ok)
	assert.Empty(t, v)

	w = httptest.NewRecorder()
	v, ok = api.UUIDQuery(w, httptest.NewRequest(http.MethodGet, "/x?department_id="+validUUID, nil), "department_id")
	assert.True(t, ok)
	assert.Equal(t, validUUID, v)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	_, ok = api.UUIDQuery(w, httptest.NewRequest(http.MethodGet, "/x?department_id=nope", nil), "department_id")
	assert.False(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
}

func TestValidateUUIDList(t *testing.T) {
	w := httptest.NewRecorder()
	assert.True(t, api.ValidateUUIDList(w, "tag_ids", []string{validUUID, validUUID}))
	w = httptest.NewRecorder()
	assert.False(t, api.ValidateUUIDList(w, "tag_ids", []string{validUUID, "x"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// The middleware must work as an inline r.Group middleware, i.e. after chi has
// resolved URL params, which is how the router mounts it.
func TestRequireUUIDPathParams_InGroup(t *testing.T) {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(api.RequireUUIDPathParams("id", "tagId"))
		ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
		r.Get("/things/{id}", ok)
		r.Get("/things/{id}/tags/{tagId}", ok)
		r.Get("/things/{id}/translations/{locale}", ok) // locale is not checked
		r.Get("/things/static", ok)
	})

	cases := []struct {
		path string
		want int
	}{
		{"/things/" + validUUID, http.StatusOK},
		{"/things/not-a-uuid", http.StatusNotFound},
		{"/things/" + validUUID + "/tags/" + validUUID, http.StatusOK},
		{"/things/" + validUUID + "/tags/zzz", http.StatusNotFound},
		{"/things/" + validUUID + "/translations/kk", http.StatusOK},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		assert.Equal(t, tc.want, w.Code, tc.path)
		if tc.want == http.StatusNotFound {
			assert.Contains(t, w.Body.String(), "NOT_FOUND")
		}
	}
}
