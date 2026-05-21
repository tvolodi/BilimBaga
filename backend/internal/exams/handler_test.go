package exams

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

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock service ─────────────────────────────────────────────────────────────

type mockSvc struct {
	createExamFn         func(ctx context.Context, input CreateExamInput) (*Exam, error)
	getExamFn            func(ctx context.Context, id string) (*ExamDetail, error)
	listExamsFn          func(ctx context.Context, filter ExamFilter) ([]*ExamListItem, int, error)
	updateExamFn         func(ctx context.Context, id string, input UpdateExamInput) (*Exam, error)
	transitionStatusFn   func(ctx context.Context, id, newStatus string) (*Exam, error)
	publishFn            func(ctx context.Context, id string) (*Exam, error)
	archiveFn            func(ctx context.Context, id string) (*Exam, error)
	unpublishExamFn      func(ctx context.Context, id string) (*Exam, error)
	deleteExamFn         func(ctx context.Context, id string) error
	createSectionFn      func(ctx context.Context, examID string, input SectionInput) (*ExamSection, error)
	updateSectionFn      func(ctx context.Context, examID, sectionID string, input SectionInput) (*ExamSection, error)
	deleteSectionFn      func(ctx context.Context, examID, sectionID string) error
	createRuleFn         func(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error)
	updateRuleFn         func(ctx context.Context, examID, ruleID string, input QuestionRuleInput) (*ExamQuestionRule, error)
	deleteRuleFn         func(ctx context.Context, examID, ruleID string) error
	setManualQuestionsFn func(ctx context.Context, ruleID string, questions []ManualQuestionInput) error
	createAssignmentFn   func(ctx context.Context, input CreateAssignmentInput) (*ExamAssignment, error)
	deleteAssignmentFn   func(ctx context.Context, examID, assignmentID, callerRole, callerDeptID string) error
	listAssignmentsFn    func(ctx context.Context, examID string) ([]*AssignmentDetail, error)
	getEligibleCountsFn  func(ctx context.Context, examID string) ([]RuleEligibleCount, error)
}

func (m *mockSvc) CreateExam(ctx context.Context, input CreateExamInput) (*Exam, error) {
	if m.createExamFn != nil {
		return m.createExamFn(ctx, input)
	}
	return &Exam{ID: "exam-1", Title: input.Title, Status: "draft"}, nil
}
func (m *mockSvc) GetExam(ctx context.Context, id string) (*ExamDetail, error) {
	if m.getExamFn != nil {
		return m.getExamFn(ctx, id)
	}
	return &ExamDetail{ID: id, Title: "Test", Status: "draft", Sections: []SectionDetail{}, Rules: []QuestionRuleDetail{}}, nil
}
func (m *mockSvc) ListExams(ctx context.Context, filter ExamFilter) ([]*ExamListItem, int, error) {
	if m.listExamsFn != nil {
		return m.listExamsFn(ctx, filter)
	}
	return []*ExamListItem{}, 0, nil
}
func (m *mockSvc) UpdateExam(ctx context.Context, id string, input UpdateExamInput) (*Exam, error) {
	if m.updateExamFn != nil {
		return m.updateExamFn(ctx, id, input)
	}
	return &Exam{ID: id, Title: input.Title, Status: "draft"}, nil
}
func (m *mockSvc) TransitionStatus(ctx context.Context, id, newStatus string) (*Exam, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, id, newStatus)
	}
	return &Exam{ID: id, Status: newStatus}, nil
}
func (m *mockSvc) Publish(ctx context.Context, id string) (*Exam, error) {
	if m.publishFn != nil {
		return m.publishFn(ctx, id)
	}
	return &Exam{ID: id, Status: "active"}, nil
}
func (m *mockSvc) Archive(ctx context.Context, id string) (*Exam, error) {
	if m.archiveFn != nil {
		return m.archiveFn(ctx, id)
	}
	return &Exam{ID: id, Status: "archived"}, nil
}
func (m *mockSvc) UnpublishExam(ctx context.Context, id string) (*Exam, error) {
	if m.unpublishExamFn != nil {
		return m.unpublishExamFn(ctx, id)
	}
	return &Exam{ID: id, Status: "draft"}, nil
}
func (m *mockSvc) DeleteExam(ctx context.Context, id string) error {
	if m.deleteExamFn != nil {
		return m.deleteExamFn(ctx, id)
	}
	return nil
}
func (m *mockSvc) CreateSection(ctx context.Context, examID string, input SectionInput) (*ExamSection, error) {
	if m.createSectionFn != nil {
		return m.createSectionFn(ctx, examID, input)
	}
	return &ExamSection{ID: "sec-1", ExamID: examID, Title: input.Title, SortOrder: input.SortOrder}, nil
}
func (m *mockSvc) UpdateSection(ctx context.Context, examID, sectionID string, input SectionInput) (*ExamSection, error) {
	if m.updateSectionFn != nil {
		return m.updateSectionFn(ctx, examID, sectionID, input)
	}
	return &ExamSection{ID: sectionID, ExamID: examID, Title: input.Title, SortOrder: input.SortOrder}, nil
}
func (m *mockSvc) DeleteSection(ctx context.Context, examID, sectionID string) error {
	if m.deleteSectionFn != nil {
		return m.deleteSectionFn(ctx, examID, sectionID)
	}
	return nil
}
func (m *mockSvc) CreateRule(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	if m.createRuleFn != nil {
		return m.createRuleFn(ctx, examID, input)
	}
	return &ExamQuestionRule{ID: "rule-1", ExamID: examID, Mode: input.Mode, Count: input.Count}, nil
}
func (m *mockSvc) UpdateRule(ctx context.Context, examID, ruleID string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	if m.updateRuleFn != nil {
		return m.updateRuleFn(ctx, examID, ruleID, input)
	}
	return &ExamQuestionRule{ID: ruleID, ExamID: examID, Mode: input.Mode, Count: input.Count}, nil
}
func (m *mockSvc) DeleteRule(ctx context.Context, examID, ruleID string) error {
	if m.deleteRuleFn != nil {
		return m.deleteRuleFn(ctx, examID, ruleID)
	}
	return nil
}
func (m *mockSvc) SetManualQuestions(ctx context.Context, ruleID string, questions []ManualQuestionInput) error {
	if m.setManualQuestionsFn != nil {
		return m.setManualQuestionsFn(ctx, ruleID, questions)
	}
	return nil
}
func (m *mockSvc) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*ExamAssignment, error) {
	if m.createAssignmentFn != nil {
		return m.createAssignmentFn(ctx, input)
	}
	return &ExamAssignment{
		ID:           "assign-1",
		ExamID:       input.ExamID,
		AssigneeType: input.AssigneeType,
		AssigneeID:   input.AssigneeID,
		Deadline:     input.Deadline,
		AssignedBy:   input.AssignedBy,
		AssignedAt:   time.Now(),
	}, nil
}
func (m *mockSvc) DeleteAssignment(ctx context.Context, examID, assignmentID, callerRole, callerDeptID string) error {
	if m.deleteAssignmentFn != nil {
		return m.deleteAssignmentFn(ctx, examID, assignmentID, callerRole, callerDeptID)
	}
	return nil
}
func (m *mockSvc) ListAssignments(ctx context.Context, examID string) ([]*AssignmentDetail, error) {
	if m.listAssignmentsFn != nil {
		return m.listAssignmentsFn(ctx, examID)
	}
	return []*AssignmentDetail{}, nil
}
func (m *mockSvc) GetEligibleCounts(ctx context.Context, examID string) ([]RuleEligibleCount, error) {
	if m.getEligibleCountsFn != nil {
		return m.getEligibleCountsFn(ctx, examID)
	}
	return []RuleEligibleCount{}, nil
}

// ── Response helpers ─────────────────────────────────────────────────────────

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *errBody        `json:"error"`
}
type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decode(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	return env
}

func withChiParam(r *http.Request, params ...string) *http.Request {
	rctx := chi.NewRouteContext()
	for i := 0; i+1 < len(params); i += 2 {
		rctx.URLParams.Add(params[i], params[i+1])
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func newHandler(svc Service) *Handler {
	return NewHandler(svc, nil)
}

func sampleExam() *Exam {
	return &Exam{
		ID:               "exam-1",
		Title:            "Sample Exam",
		Status:           "draft",
		TimeLimitMinutes: 60,
		PassingScorePct:  70,
		MaxAttempts:      1,
		ShowAnswers:      "never",
		OnTabSwitch:      "log",
		CreatedBy:        "user-1",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestList_Returns200WithEmptyItems(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/v1/exams", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestList_ServiceError_Returns500(t *testing.T) {
	h := newHandler(&mockSvc{
		listExamsFn: func(_ context.Context, _ ExamFilter) ([]*ExamListItem, int, error) {
			return nil, 0, errors.New("db error")
		},
	})
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/v1/exams", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestCreate_InvalidBody_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader("{")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_INVALID_BODY", decode(t, w).Error.Code)
}

func TestCreate_MissingTitle_ReturnsValidationError(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "ERR_VALIDATION", decode(t, w).Error.Code)
}

func TestCreate_ZeroTimeLimitReturnsValidationError(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"title":"X","time_limit_minutes":0,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_InvalidShowAnswers_ReturnsValidationError(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"title":"X","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"show_answers":"bad"}`
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_InvalidOnTabSwitch_ReturnsValidationError(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"title":"X","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"on_tab_switch":"bad"}`
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_Success_Returns201(t *testing.T) {
	h := newHandler(&mockSvc{
		createExamFn: func(_ context.Context, _ CreateExamInput) (*Exam, error) {
			return sampleExam(), nil
		},
	})
	body := `{"title":"My Exam","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestCreate_ServiceError_Returns500(t *testing.T) {
	h := newHandler(&mockSvc{
		createExamFn: func(_ context.Context, _ CreateExamInput) (*Exam, error) {
			return nil, errors.New("db error")
		},
	})
	body := `{"title":"X","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Get ──────────────────────────────────────────────────────────────────────

func TestGet_Returns200(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/exam-1", nil), "id", "exam-1")
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestGet_NotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		getExamFn: func(_ context.Context, _ string) (*ExamDetail, error) {
			return nil, ErrNotFound
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/missing", nil), "id", "missing")
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "ERR_NOT_FOUND", decode(t, w).Error.Code)
}

// ── Update ───────────────────────────────────────────────────────────────────

func TestUpdate_InvalidBody_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader("{")), "id", "exam-1")
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_ArchivedConflict_Returns409(t *testing.T) {
	h := newHandler(&mockSvc{
		updateExamFn: func(_ context.Context, _ string, _ UpdateExamInput) (*Exam, error) {
			return nil, ErrNotDraft
		},
	})
	body := `{"title":"X","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body)), "id", "exam-1")
	h.Update(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdate_Success_Returns200(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"title":"Updated","time_limit_minutes":45,"passing_score_pct":80,"max_attempts":1}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body)), "id", "exam-1")
	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── TransitionStatus ─────────────────────────────────────────────────────────

func TestTransitionStatus_InvalidBody_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/status", strings.NewReader("{")), "id", "exam-1")
	h.TransitionStatus(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransitionStatus_InvalidTransition_Returns409(t *testing.T) {
	h := newHandler(&mockSvc{
		transitionStatusFn: func(_ context.Context, _ string, _ string) (*Exam, error) {
			return nil, ErrInvalidTransition
		},
	})
	body := `{"status":"archived"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/status", strings.NewReader(body)), "id", "exam-1")
	h.TransitionStatus(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_INVALID_TRANSITION", decode(t, w).Error.Code)
}

func TestTransitionStatus_Success_Returns200(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"status":"active"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/status", strings.NewReader(body)), "id", "exam-1")
	h.TransitionStatus(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Delete ───────────────────────────────────────────────────────────────────

func TestDelete_NotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		deleteExamFn: func(_ context.Context, _ string) error { return ErrNotFound },
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1", nil), "id", "exam-1")
	h.Delete(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_NotDraft_Returns409(t *testing.T) {
	h := newHandler(&mockSvc{
		deleteExamFn: func(_ context.Context, _ string) error { return ErrNotDraft },
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1", nil), "id", "exam-1")
	h.Delete(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_NOT_DRAFT", decode(t, w).Error.Code)
}

func TestDelete_Success_Returns204(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1", nil), "id", "exam-1")
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── CreateSection ─────────────────────────────────────────────────────────────

func TestCreateSection_Success_Returns201(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"title":"Part A","sort_order":0}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/sections", strings.NewReader(body)), "id", "exam-1")
	h.CreateSection(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestCreateSection_ExamNotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		createSectionFn: func(_ context.Context, _ string, _ SectionInput) (*ExamSection, error) {
			return nil, ErrNotFound
		},
	})
	body := `{"sort_order":0}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/missing/sections", strings.NewReader(body)), "id", "missing")
	h.CreateSection(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── DeleteSection ─────────────────────────────────────────────────────────────

func TestDeleteSection_Success_Returns204(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/sections/sec-1", nil),
		"id", "exam-1", "sectionId", "sec-1",
	)
	h.DeleteSection(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── CreateRule ────────────────────────────────────────────────────────────────

func TestCreateRule_InvalidBody_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/rules", strings.NewReader("{")), "id", "exam-1")
	h.CreateRule(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateRule_InvalidMode_ReturnsValidationError(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"mode":"invalid","count":5,"sort_order":0}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/rules", strings.NewReader(body)), "id", "exam-1")
	h.CreateRule(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateRule_ZeroCount_ReturnsValidationError(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"mode":"random","count":0,"sort_order":0}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/rules", strings.NewReader(body)), "id", "exam-1")
	h.CreateRule(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateRule_Success_Returns201(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"mode":"random","count":10,"sort_order":0}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/rules", strings.NewReader(body)), "id", "exam-1")
	h.CreateRule(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestCreateRule_InvalidTagIDs_Returns422(t *testing.T) {
	h := newHandler(&mockSvc{
		createRuleFn: func(_ context.Context, _ string, _ QuestionRuleInput) (*ExamQuestionRule, error) {
			return nil, ErrInvalidInput
		},
	})
	body := `{"mode":"random","count":5,"sort_order":0,"tag_ids":["not-a-uuid"]}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/rules", strings.NewReader(body)), "id", "exam-1")
	h.CreateRule(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── DeleteRule ────────────────────────────────────────────────────────────────

func TestDeleteRule_Success_Returns204(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/rules/rule-1", nil),
		"id", "exam-1", "ruleId", "rule-1",
	)
	h.DeleteRule(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteRule_NotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		deleteRuleFn: func(_ context.Context, _, _ string) error { return ErrNotFound },
	})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/rules/missing", nil),
		"id", "exam-1", "ruleId", "missing",
	)
	h.DeleteRule(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── SetManualQuestions ────────────────────────────────────────────────────────

func TestSetManualQuestions_InvalidBody_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1/rules/rule-1/questions", strings.NewReader("{")), "id", "exam-1", "ruleId", "rule-1")
	h.SetManualQuestions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetManualQuestions_Success_Returns200(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"questions":[{"question_id":"q-1","sort_order":0},{"question_id":"q-2","sort_order":1}]}`
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1/rules/rule-1/questions", strings.NewReader(body)),
		"id", "exam-1", "ruleId", "rule-1",
	)
	h.SetManualQuestions(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestSetManualQuestions_ServiceError_Returns500(t *testing.T) {
	h := newHandler(&mockSvc{
		setManualQuestionsFn: func(_ context.Context, _ string, _ []ManualQuestionInput) error {
			return errors.New("db error")
		},
	})
	body := `{"questions":[]}`
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1/rules/rule-1/questions", strings.NewReader(body)),
		"id", "exam-1", "ruleId", "rule-1",
	)
	h.SetManualQuestions(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetManualQuestions_RandomModeRule_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{
		setManualQuestionsFn: func(_ context.Context, _ string, _ []ManualQuestionInput) error {
			return ErrRulesModeConflict
		},
	})
	body := `{"questions":[]}`
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1/rules/rule-1/questions", strings.NewReader(body)),
		"id", "exam-1", "ruleId", "rule-1",
	)
	h.SetManualQuestions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "RULE_NOT_MANUAL", decode(t, w).Error.Code)
}

// ── Assign ────────────────────────────────────────────────────────────────────

func TestAssign_InvalidBody_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader("{")), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_INVALID_BODY", decode(t, w).Error.Code)
}

func TestAssign_InvalidAssigneeType_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"assignee_type":"bad"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAssign_MissingAssigneeID_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"assignee_type":"user"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAssign_ExamNotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		createAssignmentFn: func(_ context.Context, _ CreateAssignmentInput) (*ExamAssignment, error) {
			return nil, ErrNotFound
		},
	})
	body := `{"assignee_type":"all"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/missing/assign", strings.NewReader(body)), "id", "missing")
	h.Assign(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAssign_ExamNotActive_Returns422(t *testing.T) {
	h := newHandler(&mockSvc{
		createAssignmentFn: func(_ context.Context, _ CreateAssignmentInput) (*ExamAssignment, error) {
			return nil, ErrNotActive
		},
	})
	body := `{"assignee_type":"all"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "EXAM_NOT_ACTIVE", decode(t, w).Error.Code)
}

func TestAssign_Duplicate_Returns409(t *testing.T) {
	h := newHandler(&mockSvc{
		createAssignmentFn: func(_ context.Context, _ CreateAssignmentInput) (*ExamAssignment, error) {
			return nil, ErrAssignmentExists
		},
	})
	body := `{"assignee_type":"all"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ASSIGNMENT_ALREADY_EXISTS", decode(t, w).Error.Code)
}

func TestAssign_Forbidden_Returns403(t *testing.T) {
	h := newHandler(&mockSvc{
		createAssignmentFn: func(_ context.Context, _ CreateAssignmentInput) (*ExamAssignment, error) {
			return nil, ErrForbidden
		},
	})
	body := `{"assignee_type":"all"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAssign_DeadlineInPast_Returns400(t *testing.T) {
	h := newHandler(&mockSvc{
		createAssignmentFn: func(_ context.Context, _ CreateAssignmentInput) (*ExamAssignment, error) {
			return nil, ErrDeadlineInPast
		},
	})
	body := `{"assignee_type":"all"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_DEADLINE_IN_PAST", decode(t, w).Error.Code)
}

func TestAssign_Success_Returns201(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"assignee_type":"all"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestAssign_UserType_Success_Returns201(t *testing.T) {
	h := newHandler(&mockSvc{})
	body := `{"assignee_type":"user","assignee_id":"user-uuid-x"}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/assign", strings.NewReader(body)), "id", "exam-1")
	h.Assign(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

// ── Unassign ──────────────────────────────────────────────────────────────────

func TestUnassign_Success_Returns204(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/assign/assign-1", nil),
		"id", "exam-1", "assignmentId", "assign-1",
	)
	h.Unassign(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUnassign_ExamNotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		deleteAssignmentFn: func(_ context.Context, _, _, _, _ string) error { return ErrNotFound },
	})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/missing/assign/assign-1", nil),
		"id", "missing", "assignmentId", "assign-1",
	)
	h.Unassign(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUnassign_AssignmentNotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		deleteAssignmentFn: func(_ context.Context, _, _, _, _ string) error { return ErrAssignmentNotFound },
	})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/assign/missing", nil),
		"id", "exam-1", "assignmentId", "missing",
	)
	h.Unassign(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUnassign_Forbidden_Returns403(t *testing.T) {
	h := newHandler(&mockSvc{
		deleteAssignmentFn: func(_ context.Context, _, _, _, _ string) error { return ErrForbidden },
	})
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/assign/assign-1", nil),
		"id", "exam-1", "assignmentId", "assign-1",
	)
	h.Unassign(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── ListAssignments ────────────────────────────────────────────────────────────

func TestListAssignments_Returns200WithEmpty(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/exam-1/assignments", nil), "id", "exam-1")
	h.ListAssignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestListAssignments_ExamNotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		listAssignmentsFn: func(_ context.Context, _ string) ([]*AssignmentDetail, error) {
			return nil, ErrNotFound
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/missing/assignments", nil), "id", "missing")
	h.ListAssignments(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "ERR_NOT_FOUND", decode(t, w).Error.Code)
}

func TestListAssignments_ServiceError_Returns500(t *testing.T) {
	h := newHandler(&mockSvc{
		listAssignmentsFn: func(_ context.Context, _ string) ([]*AssignmentDetail, error) {
			return nil, errors.New("db error")
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/exam-1/assignments", nil), "id", "exam-1")
	h.ListAssignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Update — draft-only enforcement ──────────────────────────────────────────

func TestUpdate_ActiveExam_Returns409WithExamNotDraftCode(t *testing.T) {
	h := newHandler(&mockSvc{
		updateExamFn: func(_ context.Context, _ string, _ UpdateExamInput) (*Exam, error) {
			return nil, ErrNotDraft
		},
	})
	body := `{"title":"X","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body)), "id", "exam-1")
	h.Update(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "EXAM_NOT_DRAFT", decode(t, w).Error.Code)
}

// ── Publish ───────────────────────────────────────────────────────────────────

func TestPublish_Success_Returns200WithActiveStatus(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/publish", nil), "id", "exam-1")
	h.Publish(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestPublish_UnsatisfiedRules_Returns422(t *testing.T) {
	h := newHandler(&mockSvc{
		publishFn: func(_ context.Context, _ string) (*Exam, error) {
			return nil, &PublishValidationError{
				Details: []RuleUnsatisfiedDetail{
					{RuleID: "rule-1", Required: 10, Available: 3, Filter: map[string]any{"difficulty": "medium"}},
				},
			}
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/publish", nil), "id", "exam-1")
	h.Publish(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	env := decode(t, w)
	require.NotNil(t, env.Error)
	assert.Equal(t, "EXAM_RULES_UNSATISFIED", env.Error.Code)
}

func TestPublish_NotDraft_Returns409(t *testing.T) {
	h := newHandler(&mockSvc{
		publishFn: func(_ context.Context, _ string) (*Exam, error) { return nil, ErrNotDraft },
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/publish", nil), "id", "exam-1")
	h.Publish(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "EXAM_NOT_DRAFT", decode(t, w).Error.Code)
}

func TestPublish_NotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		publishFn: func(_ context.Context, _ string) (*Exam, error) { return nil, ErrNotFound },
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/publish", nil), "id", "exam-1")
	h.Publish(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Archive ───────────────────────────────────────────────────────────────────

func TestArchive_Success_Returns200WithArchivedStatus(t *testing.T) {
	h := newHandler(&mockSvc{})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/archive", nil), "id", "exam-1")
	h.Archive(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestArchive_NotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		archiveFn: func(_ context.Context, _ string) (*Exam, error) { return nil, ErrNotFound },
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/archive", nil), "id", "exam-1")
	h.Archive(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestArchive_AlreadyArchived_Returns409(t *testing.T) {
	h := newHandler(&mockSvc{
		archiveFn: func(_ context.Context, _ string) (*Exam, error) { return nil, ErrInvalidTransition },
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/archive", nil), "id", "exam-1")
	h.Archive(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_INVALID_TRANSITION", decode(t, w).Error.Code)
}

// ── GetEligibleCounts handler tests (FR-BB315) ────────────────────────────────

func TestGetEligibleCountsHandler_ReturnsCountsJSON(t *testing.T) {
	h := newHandler(&mockSvc{
		getEligibleCountsFn: func(_ context.Context, examID string) ([]RuleEligibleCount, error) {
			return []RuleEligibleCount{
				{RuleID: "rule-1", Eligible: 5},
				{RuleID: "rule-2", Eligible: 0},
			}, nil
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/exam-1/rules/eligible-counts", nil), "id", "exam-1")
	h.GetEligibleCounts(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	env := decode(t, w)
	require.Nil(t, env.Error)
	require.Contains(t, string(env.Data), "counts")
}

func TestGetEligibleCountsHandler_ExamNotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		getEligibleCountsFn: func(_ context.Context, _ string) ([]RuleEligibleCount, error) {
			return nil, ErrNotFound
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/nope/rules/eligible-counts", nil), "id", "nope")
	h.GetEligibleCounts(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "EXAM_NOT_FOUND", decode(t, w).Error.Code)
}

// ── Unpublish handler tests (FR-BB318) ────────────────────────────────────────

func TestUnpublish_Success_Returns200WithDraftStatus(t *testing.T) {
	h := newHandler(&mockSvc{
		unpublishExamFn: func(_ context.Context, id string) (*Exam, error) {
			return &Exam{ID: id, Status: "draft"}, nil
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/unpublish", nil), "id", "exam-1")
	h.Unpublish(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	env := decode(t, w)
	assert.Nil(t, env.Error)
	var data map[string]string
	require.NoError(t, json.Unmarshal(env.Data, &data))
	assert.Equal(t, "draft", data["status"])
	assert.Equal(t, "exam-1", data["id"])
}

func TestUnpublish_NotFound_Returns404(t *testing.T) {
	h := newHandler(&mockSvc{
		unpublishExamFn: func(_ context.Context, _ string) (*Exam, error) {
			return nil, ErrNotFound
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/nope/unpublish", nil), "id", "nope")
	h.Unpublish(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "ERR_NOT_FOUND", decode(t, w).Error.Code)
}

func TestUnpublish_NotActive_Returns409WithERRInvalidTransition(t *testing.T) {
	h := newHandler(&mockSvc{
		unpublishExamFn: func(_ context.Context, _ string) (*Exam, error) {
			return nil, errors.New("wrapped: " + ErrNotActive.Error())
		},
	})
	// Use real ErrNotActive so errors.Is works
	h2 := newHandler(&mockSvc{
		unpublishExamFn: func(_ context.Context, _ string) (*Exam, error) {
			return nil, fmt.Errorf("ctx: %w", ErrNotActive)
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/unpublish", nil), "id", "exam-1")
	h2.Unpublish(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_INVALID_TRANSITION", decode(t, w).Error.Code)
	_ = h // suppress unused warning
}

func TestUnpublish_ActiveSessions_Returns409WithERRActiveSessions(t *testing.T) {
	h := newHandler(&mockSvc{
		unpublishExamFn: func(_ context.Context, _ string) (*Exam, error) {
			return nil, ErrActiveSessionsExist
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/api/v1/exams/exam-1/unpublish", nil), "id", "exam-1")
	h.Unpublish(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_ACTIVE_SESSIONS", decode(t, w).Error.Code)
}
