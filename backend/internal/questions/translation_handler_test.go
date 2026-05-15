package questions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── mock TranslationService ───────────────────────────────────────────────────

type mockTranslationService struct {
	getAllFn   func(ctx context.Context, questionID string) (*QuestionTranslations, error)
	upsertFn  func(ctx context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error)
	deleteFn  func(ctx context.Context, questionID, locale string) (*LocaleTranslation, error)
}

func (m *mockTranslationService) GetAll(ctx context.Context, questionID string) (*QuestionTranslations, error) {
	return m.getAllFn(ctx, questionID)
}
func (m *mockTranslationService) Upsert(ctx context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
	return m.upsertFn(ctx, questionID, locale, input)
}
func (m *mockTranslationService) Delete(ctx context.Context, questionID, locale string) (*LocaleTranslation, error) {
	return m.deleteFn(ctx, questionID, locale)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func withIDLocaleParam(r *http.Request, id, locale string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	rctx.URLParams.Add("locale", locale)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleLocaleTranslation(locale string) *LocaleTranslation {
	return &LocaleTranslation{
		Locale:    locale,
		Stem:      "Test stem",
		Options:   []AnswerTextTranslation{},
		UpdatedAt: time.Now(),
	}
}

func sampleQuestionTranslations(questionID string) *QuestionTranslations {
	return &QuestionTranslations{
		QuestionID:    questionID,
		DefaultLocale: "kk",
		LocaleCoverage: LocaleCoverage{
			Present: []string{"kk"},
			Missing: []string{"ru", "en"},
		},
		Translations: map[string]LocaleTranslation{
			"kk": *sampleLocaleTranslation("kk"),
		},
	}
}

// ── List translations ─────────────────────────────────────────────────────────

func TestTHList_Returns200(t *testing.T) {
	svc := &mockTranslationService{
		getAllFn: func(_ context.Context, qID string) (*QuestionTranslations, error) {
			return sampleQuestionTranslations(qID), nil
		},
	}
	h := NewTranslationHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/q-1/translations", nil)
	req = withIDLocaleParam(req, "q-1", "")
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeQEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestTHList_QuestionNotFound_Returns404(t *testing.T) {
	svc := &mockTranslationService{
		getAllFn: func(_ context.Context, _ string) (*QuestionTranslations, error) {
			return nil, ErrQuestionNotFound
		},
	}
	h := NewTranslationHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/missing/translations", nil)
	req = withIDLocaleParam(req, "missing", "")
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_QUESTION_NOT_FOUND", apiErr.Code)
}

// ── Upsert translation ────────────────────────────────────────────────────────

func TestTHUpsert_Returns200(t *testing.T) {
	svc := &mockTranslationService{
		upsertFn: func(_ context.Context, _, locale string, _ UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
			before := sampleLocaleTranslation(locale)
			after := sampleLocaleTranslation(locale)
			after.Stem = "Updated stem"
			return before, after, nil
		},
	}
	h := NewTranslationHandler(svc, nil)

	body := `{"stem":"Updated stem","options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1/translations/kk", strings.NewReader(body))
	req = withIDLocaleParam(req, "q-1", "kk")
	w := httptest.NewRecorder()
	h.Upsert(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTHUpsert_EmptyStem_Returns422(t *testing.T) {
	h := NewTranslationHandler(&mockTranslationService{}, nil)

	body := `{"stem":"","options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1/translations/kk", strings.NewReader(body))
	req = withIDLocaleParam(req, "q-1", "kk")
	w := httptest.NewRecorder()
	h.Upsert(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTHUpsert_InvalidJSON_Returns400(t *testing.T) {
	h := NewTranslationHandler(&mockTranslationService{}, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1/translations/kk", strings.NewReader("bad-json"))
	req = withIDLocaleParam(req, "q-1", "kk")
	w := httptest.NewRecorder()
	h.Upsert(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTHUpsert_QuestionNotFound_Returns404(t *testing.T) {
	svc := &mockTranslationService{
		upsertFn: func(_ context.Context, _, _ string, _ UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
			return nil, nil, ErrQuestionNotFound
		},
	}
	h := NewTranslationHandler(svc, nil)

	body := `{"stem":"A stem","options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/missing/translations/kk", strings.NewReader(body))
	req = withIDLocaleParam(req, "missing", "kk")
	w := httptest.NewRecorder()
	h.Upsert(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTHUpsert_UnsupportedLocale_Returns422(t *testing.T) {
	svc := &mockTranslationService{
		upsertFn: func(_ context.Context, _, _ string, _ UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
			return nil, nil, ErrUnsupportedLocale
		},
	}
	h := NewTranslationHandler(svc, nil)

	body := `{"stem":"A stem","options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1/translations/xx", strings.NewReader(body))
	req = withIDLocaleParam(req, "q-1", "xx")
	w := httptest.NewRecorder()
	h.Upsert(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTHUpsert_MissingOptions_Returns422(t *testing.T) {
	svc := &mockTranslationService{
		upsertFn: func(_ context.Context, _, _ string, _ UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
			return nil, nil, &MissingOptionsError{MissingIDs: []string{"opt-1", "opt-2"}}
		},
	}
	h := NewTranslationHandler(svc, nil)

	body := `{"stem":"A stem","options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1/translations/ru", strings.NewReader(body))
	req = withIDLocaleParam(req, "q-1", "ru")
	w := httptest.NewRecorder()
	h.Upsert(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── Delete translation ────────────────────────────────────────────────────────

func TestTHDelete_Returns204(t *testing.T) {
	svc := &mockTranslationService{
		deleteFn: func(_ context.Context, _, _ string) (*LocaleTranslation, error) {
			return sampleLocaleTranslation("ru"), nil
		},
	}
	h := NewTranslationHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/q-1/translations/ru", nil)
	req = withIDLocaleParam(req, "q-1", "ru")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestTHDelete_QuestionNotFound_Returns404(t *testing.T) {
	svc := &mockTranslationService{
		deleteFn: func(_ context.Context, _, _ string) (*LocaleTranslation, error) {
			return nil, ErrQuestionNotFound
		},
	}
	h := NewTranslationHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/missing/translations/ru", nil)
	req = withIDLocaleParam(req, "missing", "ru")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTHDelete_CannotDeleteDefault_Returns409(t *testing.T) {
	svc := &mockTranslationService{
		deleteFn: func(_ context.Context, _, _ string) (*LocaleTranslation, error) {
			return nil, ErrCannotDeleteDefaultLocale
		},
	}
	h := NewTranslationHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/q-1/translations/kk", nil)
	req = withIDLocaleParam(req, "q-1", "kk")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}
