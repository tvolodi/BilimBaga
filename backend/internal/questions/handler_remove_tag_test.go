package questions

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// #514: the RemoveTag status mapping. A missing question is 404 and the tag is not touched; a failure on either
// step is 500 with its own code; success is 204.

func removeTagRequest(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/q-1/tags/tag-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "q-1")
	rctx.URLParams.Add("tagId", "tag-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.RemoveTag(w, req)
	return w
}

func TestQHandlerRemoveTag_MissingQuestionIs404AndTagIsKept(t *testing.T) {
	untagged := false
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, _ string) (*Question, error) { return nil, ErrQuestionNotFound },
		untagFn: func(_ context.Context, _, _ string) error {
			untagged = true
			return nil
		},
	}

	w := removeTagRequest(t, NewHandler(svc, nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, eb := decodeQEnvelope(t, w)
	if assert.NotNil(t, eb) {
		assert.Equal(t, "ERR_NOT_FOUND", eb.Code)
	}
	assert.False(t, untagged, "a missing question must not reach the tag removal")
}

func TestQHandlerRemoveTag_LookupFailureIs500(t *testing.T) {
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, _ string) (*Question, error) { return nil, errors.New("db down") },
	}

	w := removeTagRequest(t, NewHandler(svc, nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	_, eb := decodeQEnvelope(t, w)
	if assert.NotNil(t, eb) {
		assert.Equal(t, "ERR_INTERNAL", eb.Code)
		assert.Equal(t, "failed to find question", eb.Message)
	}
}

func TestQHandlerRemoveTag_RemovalFailureIs500(t *testing.T) {
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, id string) (*Question, error) { return sampleQuestion(id), nil },
		untagFn:       func(_ context.Context, _, _ string) error { return errors.New("constraint") },
	}

	w := removeTagRequest(t, NewHandler(svc, nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	_, eb := decodeQEnvelope(t, w)
	if assert.NotNil(t, eb) {
		assert.Equal(t, "failed to remove tag", eb.Message)
	}
}
