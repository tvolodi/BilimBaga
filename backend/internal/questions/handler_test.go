package questions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── mock service ──────────────────────────────────────────────────────────────

type mockQService struct {
	createFullFn       func(ctx context.Context, input CreateQuestionFullInput) (*QuestionDetail, error)
	listFilteredFn     func(ctx context.Context, filter QuestionFilter) ([]*QuestionListItem, int, error)
	getWithDetailsFn   func(ctx context.Context, id string) (*QuestionDetail, error)
	getQuestionFn      func(ctx context.Context, id string) (*Question, error)
	updateFn           func(ctx context.Context, id string, input UpdateQuestionInput) (*Question, error)
	transitionStatusFn func(ctx context.Context, id, newStatus string) (*Question, error)
	deleteFn           func(ctx context.Context, id string) error
	listVersionsFn     func(ctx context.Context, id string) ([]*VersionEntry, error)
	addTagFn           func(ctx context.Context, questionID, tagID string) error
	untagFn            func(ctx context.Context, questionID, tagID string) error
	getTagsFn          func(ctx context.Context, questionID string) ([]string, error)
	// legacy methods — not exercised in handler tests but required by interface
	createQuestionFn    func(ctx context.Context, input CreateQuestionInput) (*Question, error)
	listQuestionsFn     func(ctx context.Context, categoryID string) ([]*Question, error)
	publishVersionFn    func(ctx context.Context, previousID string, input CreateQuestionInput) (*Question, error)
	addTranslationFn    func(ctx context.Context, questionID, locale, stem string, explanation *string) (*QuestionTranslation, error)
	addAnswerOptionFn   func(ctx context.Context, questionID string, sortOrder int, isCorrect bool, likertWeight *float64, likertPolarity *string) (*AnswerOption, error)
	addAnswerTranslFn   func(ctx context.Context, optionID, locale, text string) (*AnswerTranslation, error)
	validateAndImportFn func(ctx context.Context, rows []ImportRow, dryRun bool, createdBy string) (*DryRunReport, *CommitResult, error)
	streamExportFn      func(ctx context.Context, filter ExportFilter, fn func(*ExportRow) error) error
}

func (m *mockQService) CreateQuestionFull(ctx context.Context, input CreateQuestionFullInput) (*QuestionDetail, error) {
	return m.createFullFn(ctx, input)
}
func (m *mockQService) ListFiltered(ctx context.Context, filter QuestionFilter) ([]*QuestionListItem, int, error) {
	return m.listFilteredFn(ctx, filter)
}
func (m *mockQService) GetQuestionWithDetails(ctx context.Context, id string) (*QuestionDetail, error) {
	return m.getWithDetailsFn(ctx, id)
}
func (m *mockQService) GetQuestion(ctx context.Context, id string) (*Question, error) {
	return m.getQuestionFn(ctx, id)
}
func (m *mockQService) UpdateQuestion(ctx context.Context, id string, input UpdateQuestionInput) (*Question, error) {
	return m.updateFn(ctx, id, input)
}
func (m *mockQService) TransitionStatus(ctx context.Context, id, newStatus string) (*Question, error) {
	return m.transitionStatusFn(ctx, id, newStatus)
}
func (m *mockQService) DeleteQuestion(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}
func (m *mockQService) ListVersions(ctx context.Context, id string) ([]*VersionEntry, error) {
	return m.listVersionsFn(ctx, id)
}
func (m *mockQService) TagQuestion(ctx context.Context, qID, tagID string) error {
	return m.addTagFn(ctx, qID, tagID)
}
func (m *mockQService) UntagQuestion(ctx context.Context, qID, tagID string) error {
	return m.untagFn(ctx, qID, tagID)
}
func (m *mockQService) GetQuestionTags(ctx context.Context, qID string) ([]string, error) {
	return m.getTagsFn(ctx, qID)
}
func (m *mockQService) CreateQuestion(ctx context.Context, input CreateQuestionInput) (*Question, error) {
	if m.createQuestionFn != nil {
		return m.createQuestionFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}
func (m *mockQService) ListQuestions(ctx context.Context, categoryID string) ([]*Question, error) {
	if m.listQuestionsFn != nil {
		return m.listQuestionsFn(ctx, categoryID)
	}
	return nil, errors.New("not implemented")
}
func (m *mockQService) PublishNewVersion(ctx context.Context, previousID string, input CreateQuestionInput) (*Question, error) {
	if m.publishVersionFn != nil {
		return m.publishVersionFn(ctx, previousID, input)
	}
	return nil, errors.New("not implemented")
}
func (m *mockQService) AddTranslation(ctx context.Context, questionID, locale, stem string, explanation *string) (*QuestionTranslation, error) {
	if m.addTranslationFn != nil {
		return m.addTranslationFn(ctx, questionID, locale, stem, explanation)
	}
	return nil, errors.New("not implemented")
}
func (m *mockQService) AddAnswerOption(ctx context.Context, questionID string, sortOrder int, isCorrect bool, likertWeight *float64, likertPolarity *string) (*AnswerOption, error) {
	if m.addAnswerOptionFn != nil {
		return m.addAnswerOptionFn(ctx, questionID, sortOrder, isCorrect, likertWeight, likertPolarity)
	}
	return nil, errors.New("not implemented")
}
func (m *mockQService) AddAnswerTranslation(ctx context.Context, optionID, locale, text string) (*AnswerTranslation, error) {
	if m.addAnswerTranslFn != nil {
		return m.addAnswerTranslFn(ctx, optionID, locale, text)
	}
	return nil, errors.New("not implemented")
}
func (m *mockQService) ValidateAndImport(ctx context.Context, rows []ImportRow, dryRun bool, createdBy string) (*DryRunReport, *CommitResult, error) {
	if m.validateAndImportFn != nil {
		return m.validateAndImportFn(ctx, rows, dryRun, createdBy)
	}
	return nil, nil, errors.New("not implemented")
}
func (m *mockQService) StreamExport(ctx context.Context, filter ExportFilter, fn func(*ExportRow) error) error {
	if m.streamExportFn != nil {
		return m.streamExportFn(ctx, filter, fn)
	}
	return errors.New("not implemented")
}

// ── test helpers ──────────────────────────────────────────────────────────────

type qEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *qErrBody       `json:"error"`
}

type qErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeQEnvelope(t *testing.T, w *httptest.ResponseRecorder) (json.RawMessage, *qErrBody) {
	t.Helper()
	var env qEnvelope
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	return env.Data, env.Error
}

func withQAuthCtx(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, userID)
	return r.WithContext(ctx)
}

func withQChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleDetail(id string) *QuestionDetail {
	return &QuestionDetail{
		ID:            id,
		Type:          "single",
		Difficulty:    "easy",
		Status:        "draft",
		CategoryID:    "cat-1",
		DefaultLocale: "kk",
		Version:       1,
		Translations: map[string]TranslationDetail{
			"kk": {Stem: "Question stem"},
		},
		AnswerOptions: []AnswerOptionDetail{},
		TagIDs:        []string{},
		CreatedBy:     "actor-1",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func sampleQuestion(id string) *Question {
	return &Question{
		ID:            id,
		CategoryID:    "cat-1",
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "kk",
		Status:        "draft",
		Version:       1,
		CreatedBy:     "actor-1",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestQHandlerList_Returns200(t *testing.T) {
	svc := &mockQService{
		listFilteredFn: func(_ context.Context, _ QuestionFilter) ([]*QuestionListItem, int, error) {
			return []*QuestionListItem{}, 0, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeQEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestQHandlerList_ParsesLocaleAndVersionParams(t *testing.T) {
	var got QuestionFilter
	svc := &mockQService{
		listFilteredFn: func(_ context.Context, f QuestionFilter) ([]*QuestionListItem, int, error) {
			got = f
			return []*QuestionListItem{}, 0, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions?locale=kk&locale_missing=ru&include_versions=true", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, got.Locale) {
		assert.Equal(t, "kk", *got.Locale)
	}
	if assert.NotNil(t, got.LocaleMissing) {
		assert.Equal(t, "ru", *got.LocaleMissing)
	}
	assert.True(t, got.IncludeSuperseded)
}

func TestQHandlerList_DefaultHidesSupersededVersions(t *testing.T) {
	var got QuestionFilter
	svc := &mockQService{
		listFilteredFn: func(_ context.Context, f QuestionFilter) ([]*QuestionListItem, int, error) {
			got = f
			return []*QuestionListItem{}, 0, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions", nil)
	h.List(httptest.NewRecorder(), req)

	assert.Nil(t, got.Locale)
	assert.False(t, got.IncludeSuperseded)
}

func TestQHandlerList_ServiceError_Returns500(t *testing.T) {
	svc := &mockQService{
		listFilteredFn: func(_ context.Context, _ QuestionFilter) ([]*QuestionListItem, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestQHandlerCreate_Returns201(t *testing.T) {
	svc := &mockQService{
		createFullFn: func(_ context.Context, _ CreateQuestionFullInput) (*QuestionDetail, error) {
			return sampleDetail("q-1"), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{
		"category_id":"cat-1",
		"difficulty":"easy",
		"type":"single",
		"default_locale":"kk",
		"translations":{"kk":{"stem":"What is Go?"}},
		"answer_options":[
			{"sort_order":1,"is_correct":true,"translations":{"kk":{"text":"A language"}}},
			{"sort_order":2,"is_correct":false,"translations":{"kk":{"text":"A framework"}}}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(body))
	req = withQAuthCtx(req, "actor-1")
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	data, apiErr := decodeQEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestQHandlerCreate_InvalidJSON_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader("bad-json"))
	req = withQAuthCtx(req, "actor-1")
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_INVALID_BODY", apiErr.Code)
}

func TestQHandlerCreate_MissingFields_Returns422(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	// Missing category_id, bad type
	body := `{"category_id":"","difficulty":"easy","type":"bad","default_locale":"kk","translations":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(body))
	req = withQAuthCtx(req, "actor-1")
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestQHandlerGet_Returns200(t *testing.T) {
	svc := &mockQService{
		getWithDetailsFn: func(_ context.Context, id string) (*QuestionDetail, error) {
			return sampleDetail(id), nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/q-1", nil)
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ISS-017: ensure the GET detail response uses "tag_ids" (not "tags") so the
// frontend can populate the tag field after save.
func TestQHandlerGet_ResponseContainsTagIDs(t *testing.T) {
	tagID := "tag-uuid-1"
	svc := &mockQService{
		getWithDetailsFn: func(_ context.Context, id string) (*QuestionDetail, error) {
			d := sampleDetail(id)
			d.TagIDs = []string{tagID}
			return d, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/q-1", nil)
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	raw, _ := decodeQEnvelope(t, w)

	var detail map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &detail))

	// Must have "tag_ids" key, not "tags".
	tagIDsRaw, hasTagIDs := detail["tag_ids"]
	_, hasTagsKey := detail["tags"]
	assert.True(t, hasTagIDs, "response must contain 'tag_ids' field")
	assert.False(t, hasTagsKey, "response must NOT contain a 'tags' field on QuestionDetail")

	var ids []string
	require.NoError(t, json.Unmarshal(tagIDsRaw, &ids))
	assert.Equal(t, []string{tagID}, ids)
}

func TestQHandlerGet_NotFound_Returns404(t *testing.T) {
	svc := &mockQService{
		getWithDetailsFn: func(_ context.Context, _ string) (*QuestionDetail, error) {
			return nil, ErrQuestionNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/missing", nil)
	req = withQChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_NOT_FOUND", apiErr.Code)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestQHandlerUpdate_Returns200(t *testing.T) {
	svc := &mockQService{
		updateFn: func(_ context.Context, id string, _ UpdateQuestionInput) (*Question, error) {
			return sampleQuestion(id), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"category_id":"cat-1","difficulty":"medium","translations":{},"answer_options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1", strings.NewReader(body))
	req = withQAuthCtx(req, "actor-1")
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQHandlerUpdate_NotFound_Returns404(t *testing.T) {
	svc := &mockQService{
		updateFn: func(_ context.Context, _ string, _ UpdateQuestionInput) (*Question, error) {
			return nil, ErrQuestionNotFound
		},
	}
	h := NewHandler(svc, nil)

	body := `{"category_id":"cat-1","difficulty":"medium","translations":{},"answer_options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/missing", strings.NewReader(body))
	req = withQAuthCtx(req, "actor-1")
	req = withQChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestQHandlerUpdate_ArchivedQuestion_Returns422(t *testing.T) {
	svc := &mockQService{
		updateFn: func(_ context.Context, _ string, _ UpdateQuestionInput) (*Question, error) {
			return nil, fmt.Errorf("questions: UpdateQuestion: %w: question is archived", ErrInvalidInput)
		},
	}
	h := NewHandler(svc, nil)

	body := `{"category_id":"cat-1","difficulty":"medium","translations":{},"answer_options":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/archived-q", strings.NewReader(body))
	req = withQAuthCtx(req, "actor-1")
	req = withQChiParam(req, "id", "archived-q")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── TransitionStatus ──────────────────────────────────────────────────────────

func TestQHandlerTransitionStatus_Returns200(t *testing.T) {
	svc := &mockQService{
		transitionStatusFn: func(_ context.Context, id, _ string) (*Question, error) {
			q := sampleQuestion(id)
			q.Status = "published"
			return q, nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"status":"published"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/status", strings.NewReader(body))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.TransitionStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQHandlerTransitionStatus_InvalidTransition_Returns422(t *testing.T) {
	svc := &mockQService{
		transitionStatusFn: func(_ context.Context, _, _ string) (*Question, error) {
			return nil, ErrInvalidTransition
		},
	}
	h := NewHandler(svc, nil)

	body := `{"status":"archived"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/status", strings.NewReader(body))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.TransitionStatus(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestQHandlerTransitionStatus_MissingStatus_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	body := `{"status":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/status", strings.NewReader(body))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.TransitionStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ISS-012 regression: sending "target_status" (wrong key) must still return 400.
// The backend contract uses "status" — unknown JSON keys are silently ignored.
func TestQHandlerTransitionStatus_WrongFieldNameKey_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	body := `{"target_status":"review"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/status", strings.NewReader(body))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.TransitionStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestQHandlerDelete_Returns204(t *testing.T) {
	svc := &mockQService{
		deleteFn: func(_ context.Context, _ string) error { return nil },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/q-1", nil)
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestQHandlerDelete_NotFound_Returns404(t *testing.T) {
	svc := &mockQService{
		deleteFn: func(_ context.Context, _ string) error { return ErrQuestionNotFound },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/missing", nil)
	req = withQChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestQHandlerDelete_NotDraft_Returns409(t *testing.T) {
	svc := &mockQService{
		deleteFn: func(_ context.Context, _ string) error { return ErrNotDraft },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/q-1", nil)
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ── ListVersions ──────────────────────────────────────────────────────────────

func TestQHandlerListVersions_Returns200(t *testing.T) {
	svc := &mockQService{
		listVersionsFn: func(_ context.Context, _ string) ([]*VersionEntry, error) {
			return []*VersionEntry{
				{ID: "q-1", Version: 1, Status: "archived", CreatedAt: time.Now()},
			}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/q-1/versions", nil)
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.ListVersions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQHandlerListVersions_NotFound_Returns404(t *testing.T) {
	svc := &mockQService{
		listVersionsFn: func(_ context.Context, _ string) ([]*VersionEntry, error) {
			return nil, ErrQuestionNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/missing/versions", nil)
	req = withQChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.ListVersions(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── AddTag / RemoveTag ────────────────────────────────────────────────────────

func TestQHandlerAddTag_Returns200(t *testing.T) {
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, id string) (*Question, error) {
			return sampleQuestion(id), nil
		},
		addTagFn: func(_ context.Context, _, _ string) error { return nil },
		getTagsFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"tag-1"}, nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"tag_id":"tag-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/tags", strings.NewReader(body))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.AddTag(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQHandlerAddTag_QuestionNotFound_Returns404(t *testing.T) {
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, _ string) (*Question, error) {
			return nil, ErrQuestionNotFound
		},
	}
	h := NewHandler(svc, nil)

	body := `{"tag_id":"tag-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/missing/tags", strings.NewReader(body))
	req = withQChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.AddTag(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestQHandlerAddTag_MissingTagID_Returns400(t *testing.T) {
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, id string) (*Question, error) {
			return sampleQuestion(id), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"tag_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/tags", strings.NewReader(body))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.AddTag(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQHandlerRemoveTag_Returns204(t *testing.T) {
	svc := &mockQService{
		getQuestionFn: func(_ context.Context, id string) (*Question, error) {
			return sampleQuestion(id), nil
		},
		untagFn: func(_ context.Context, _, _ string) error { return nil },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/q-1/tags/tag-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "q-1")
	rctx.URLParams.Add("tagId", "tag-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.RemoveTag(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
