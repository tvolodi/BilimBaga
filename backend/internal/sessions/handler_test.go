package sessions

import (
	"context"
	"encoding/json"
	"errors"
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

// ── Mock service ─────────────────────────────────────────────────────────────

type mockSvc struct {
	createSessionFn      func(ctx context.Context, examID, userID, deptID string) (*CreateSessionResponse, error)
	saveAnswerFn         func(ctx context.Context, sessionID, questionID, userID string, input SaveAnswerInput) (*SaveAnswerResponse, error)
	getSessionStateFn    func(ctx context.Context, sessionID, userID string) (*ResumeSessionResponse, error)
	reportEventFn        func(ctx context.Context, sessionID, userID string, input ReportEventInput) (*ReportEventResponse, error)
	submitSessionFn      func(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error)
	getSessionResultFn   func(ctx context.Context, sessionID, userID string) (*SessionResultResponse, error)
	getAdminResultFn     func(ctx context.Context, sessionID string) (*SessionResultResponse, error)
	getExamHistoryFn     func(ctx context.Context, examID, userID string, page, perPage int) (*ExamHistoryResponse, error)
}

func (m *mockSvc) CreateSession(ctx context.Context, examID, userID, deptID string) (*CreateSessionResponse, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, examID, userID, deptID)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) SaveAnswer(ctx context.Context, sessionID, questionID, userID string, input SaveAnswerInput) (*SaveAnswerResponse, error) {
	if m.saveAnswerFn != nil {
		return m.saveAnswerFn(ctx, sessionID, questionID, userID, input)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) GetSessionState(ctx context.Context, sessionID, userID string) (*ResumeSessionResponse, error) {
	if m.getSessionStateFn != nil {
		return m.getSessionStateFn(ctx, sessionID, userID)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) ReportEvent(ctx context.Context, sessionID, userID string, input ReportEventInput) (*ReportEventResponse, error) {
	if m.reportEventFn != nil {
		return m.reportEventFn(ctx, sessionID, userID, input)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) SubmitSession(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error) {
	if m.submitSessionFn != nil {
		return m.submitSessionFn(ctx, sessionID, userID, tenantID, actorIP)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) GetSessionResult(ctx context.Context, sessionID, userID string) (*SessionResultResponse, error) {
	if m.getSessionResultFn != nil {
		return m.getSessionResultFn(ctx, sessionID, userID)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) GetAdminSessionResult(ctx context.Context, sessionID string) (*SessionResultResponse, error) {
	if m.getAdminResultFn != nil {
		return m.getAdminResultFn(ctx, sessionID)
	}
	return nil, errors.New("not configured")
}
func (m *mockSvc) GetExamHistory(ctx context.Context, examID, userID string, page, perPage int) (*ExamHistoryResponse, error) {
	if m.getExamHistoryFn != nil {
		return m.getExamHistoryFn(ctx, examID, userID, page, perPage)
	}
	return nil, errors.New("not configured")
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withUserCtx(r *http.Request, userID, deptID string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, userID)
	ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, deptID)
	return r.WithContext(ctx)
}

func decodeBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

func successResponse() *CreateSessionResponse {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	return &CreateSessionResponse{
		SessionID:        "sess-1",
		ExamID:           "exam-1",
		StartedAt:        now,
		ExpiresAt:        now.Add(90 * time.Minute),
		RemainingSeconds: 5400,
		Questions:        []SessionQuestionResponse{},
	}
}

// ── 201 success ──────────────────────────────────────────────────────────────

func TestCreateSession_Handler_201(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, examID, userID, deptID string) (*CreateSessionResponse, error) {
			assert.Equal(t, "exam-1", examID)
			assert.Equal(t, "user-1", userID)
			return successResponse(), nil
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sess-1", data["session_id"])
}

// ── AC-1: 403 not assigned ───────────────────────────────────────────────────

func TestCreateSession_Handler_403_NotAssigned(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, ErrNotAssigned
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "EXAM_NOT_ASSIGNED", errObj["code"])
}

// ── AC-2: 422 exam not active ────────────────────────────────────────────────

func TestCreateSession_Handler_422_ExamNotActive(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, ErrExamNotActive
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "EXAM_NOT_ACTIVE", errObj["code"])
}

// ── AC-3: 422 outside window ─────────────────────────────────────────────────

func TestCreateSession_Handler_422_OutsideWindow(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, ErrExamOutsideWindow
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "EXAM_OUTSIDE_WINDOW", errObj["code"])
}

// ── AC-4: 422 attempts exhausted ─────────────────────────────────────────────

func TestCreateSession_Handler_422_AttemptsExhausted(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, ErrAttemptsExhausted
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "ATTEMPTS_EXHAUSTED", errObj["code"])
}

// ── AC-5: 409 session already open ───────────────────────────────────────────

func TestCreateSession_Handler_409_SessionAlreadyOpen(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, ErrSessionAlreadyOpen
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_ALREADY_OPEN", errObj["code"])
}

// ── Insufficient questions ───────────────────────────────────────────────────

func TestCreateSession_Handler_422_InsufficientQuestions(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, ErrInsufficientQuestions
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "INSUFFICIENT_QUESTIONS", errObj["code"])
}

// ── 500 internal error ───────────────────────────────────────────────────────

func TestCreateSession_Handler_500_InternalError(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Wrapped errors still match ───────────────────────────────────────────────

func TestCreateSession_Handler_WrappedNotAssigned_Returns403(t *testing.T) {
	svc := &mockSvc{
		createSessionFn: func(_ context.Context, _, _, _ string) (*CreateSessionResponse, error) {
			return nil, errors.Join(errors.New("sessions: CreateSession"), ErrNotAssigned)
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/portal/exams/exam-1/sessions", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()

	h.CreateSession(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── SaveAnswer handler tests ──────────────────────────────────────────────────

func saveAnswerRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/portal/sessions/sess-1/answers/q-1",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParam(req, "id", "sess-1")
	req = withChiParam(req, "questionId", "q-1")
	return withUserCtx(req, "user-1", "dept-1")
}

func saveAnswerSuccess() *SaveAnswerResponse {
	return &SaveAnswerResponse{
		QuestionID:       "q-1",
		SavedAt:          time.Date(2026, 5, 1, 10, 1, 0, 0, time.UTC),
		RemainingSeconds: 5166,
	}
}

func TestSaveAnswer_Handler_200(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return saveAnswerSuccess(), nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":["opt-b"],"time_spent_seconds":45}`))

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "q-1", data["question_id"])
}

func TestSaveAnswer_Handler_400_InvalidBody(t *testing.T) {
	h := NewHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPut, "/portal/sessions/sess-1/answers/q-1",
		strings.NewReader("not-json"))
	req = withChiParam(req, "id", "sess-1")
	req = withChiParam(req, "questionId", "q-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()
	h.SaveAnswer(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaveAnswer_Handler_400_NegativeTimeSpent(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrNegativeTimeSpent
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":[],"time_spent_seconds":-1}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "INVALID_TIME_SPENT", errObj["code"])
}

func TestSaveAnswer_Handler_400_InvalidOption(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrInvalidOption
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":["bad-opt"],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "INVALID_OPTION", errObj["code"])
}

func TestSaveAnswer_Handler_400_InvalidAnswerFormat(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrInvalidAnswerFormat
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":["a","b"],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "INVALID_ANSWER_FORMAT", errObj["code"])
}

func TestSaveAnswer_Handler_404_QuestionNotInSession(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrQuestionNotInSession
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":[],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusNotFound, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "QUESTION_NOT_IN_SESSION", errObj["code"])
}

func TestSaveAnswer_Handler_422_SessionExpired(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrSessionExpired
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":[],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_EXPIRED", errObj["code"])
}

func TestSaveAnswer_Handler_422_SessionNotActive(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrSessionNotActive
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":[],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_NOT_ACTIVE", errObj["code"])
}

func TestSaveAnswer_Handler_403_Forbidden(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, ErrSessionForbidden
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":[],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveAnswer_Handler_500(t *testing.T) {
	svc := &mockSvc{
		saveAnswerFn: func(_ context.Context, _, _, _ string, _ SaveAnswerInput) (*SaveAnswerResponse, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SaveAnswer(w, saveAnswerRequest(`{"selected_option_ids":[],"time_spent_seconds":0}`))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetSessionState handler tests ─────────────────────────────────────────────

func resumeResponse() *ResumeSessionResponse {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	return &ResumeSessionResponse{
		SessionID:        "sess-1",
		ExamID:           "exam-1",
		Status:           "in_progress",
		StartedAt:        now,
		ExpiresAt:        now.Add(90 * time.Minute),
		RemainingSeconds: 5166,
		Questions:        []SessionQuestionResponse{},
		Answers:          map[string]SavedAnswer{},
	}
}

func TestGetSessionState_Handler_200(t *testing.T) {
	svc := &mockSvc{
		getSessionStateFn: func(_ context.Context, sessionID, _ string) (*ResumeSessionResponse, error) {
			assert.Equal(t, "sess-1", sessionID)
			return resumeResponse(), nil
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1", nil)
	req = withChiParam(req, "id", "sess-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()
	h.GetSessionState(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sess-1", data["session_id"])
}

func TestGetSessionState_Handler_403_Forbidden(t *testing.T) {
	svc := &mockSvc{
		getSessionStateFn: func(_ context.Context, _, _ string) (*ResumeSessionResponse, error) {
			return nil, ErrSessionForbidden
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1", nil)
	req = withChiParam(req, "id", "sess-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()
	h.GetSessionState(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_FORBIDDEN", errObj["code"])
}

func TestGetSessionState_Handler_404_NotFound(t *testing.T) {
	svc := &mockSvc{
		getSessionStateFn: func(_ context.Context, _, _ string) (*ResumeSessionResponse, error) {
			return nil, ErrSessionNotFound
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1", nil)
	req = withChiParam(req, "id", "sess-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()
	h.GetSessionState(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_NOT_FOUND", errObj["code"])
}

func TestGetSessionState_Handler_500(t *testing.T) {
	svc := &mockSvc{
		getSessionStateFn: func(_ context.Context, _, _ string) (*ResumeSessionResponse, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1", nil)
	req = withChiParam(req, "id", "sess-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()
	h.GetSessionState(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── ReportEvent handler tests (FR-BB38) ──────────────────────────────────────

func reportEventRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/portal/sessions/sess-1/events",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParam(req, "id", "sess-1")
	return withUserCtx(req, "user-1", "dept-1")
}

// AC-4: on_tab_switch='log' → 200 with warn=false.
func TestReportEvent_Handler_200_Log(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return &ReportEventResponse{Warn: false, EventCount: 1}, nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"tab_switch"}`))
	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, data["warn"])
}

// AC-3: on_tab_switch='warn' → 200 with warn=true + event_count.
func TestReportEvent_Handler_200_Warn(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return &ReportEventResponse{Warn: true, EventCount: 3}, nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"blur"}`))
	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, data["warn"])
	assert.Equal(t, float64(3), data["event_count"])
}

// AC-2: on_tab_switch='submit' → 200 with session_id + status=auto_submitted.
func TestReportEvent_Handler_200_Submit(t *testing.T) {
	sid := "sess-1"
	status := "auto_submitted"
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return &ReportEventResponse{
				Warn:      false,
				SessionID: &sid,
				Status:    &status,
			}, nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"fullscreen_exit"}`))
	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sess-1", data["session_id"])
	assert.Equal(t, "auto_submitted", data["status"])
}

// AC-8: invalid event type → 400 INVALID_EVENT_TYPE.
func TestReportEvent_Handler_400_InvalidEventType(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return nil, ErrInvalidEventType
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"keyboard"}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "INVALID_EVENT_TYPE", errObj["code"])
}

// bad JSON → 400 INVALID_BODY.
func TestReportEvent_Handler_400_InvalidBody(t *testing.T) {
	h := NewHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/portal/sessions/sess-1/events",
		strings.NewReader("not-json"))
	req = withChiParam(req, "id", "sess-1")
	req = withUserCtx(req, "user-1", "dept-1")
	w := httptest.NewRecorder()
	h.ReportEvent(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "INVALID_BODY", errObj["code"])
}

// AC-7: wrong user → 403 SESSION_FORBIDDEN.
func TestReportEvent_Handler_403_Forbidden(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return nil, ErrSessionForbidden
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"tab_switch"}`))
	assert.Equal(t, http.StatusForbidden, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_FORBIDDEN", errObj["code"])
}

// AC-5: session not in_progress → 422 SESSION_NOT_ACTIVE.
func TestReportEvent_Handler_422_SessionNotActive(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return nil, ErrSessionNotActive
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"tab_switch"}`))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_NOT_ACTIVE", errObj["code"])
}

// AC-6: session expired → 422 SESSION_EXPIRED.
func TestReportEvent_Handler_422_SessionExpired(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return nil, ErrSessionExpired
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"tab_switch"}`))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_EXPIRED", errObj["code"])
}

// 500 internal error.
func TestReportEvent_Handler_500(t *testing.T) {
	svc := &mockSvc{
		reportEventFn: func(_ context.Context, _, _ string, _ ReportEventInput) (*ReportEventResponse, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.ReportEvent(w, reportEventRequest(`{"type":"tab_switch"}`))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── FR-BB39: SubmitSession handler ───────────────────────────────────────────

func submitRequest(sessionID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/portal/sessions/"+sessionID+"/submit", strings.NewReader("{}"))
	req = withChiParam(req, "id", sessionID)
	req = withUserCtx(req, "user-1", "dept-1")
	return req
}

// AC-5: auto-graded → 200 with score_pct and passed.
func TestSubmitSession_Handler_200_AutoGraded(t *testing.T) {
	score := 82.5
	passed := true
	submittedAt := time.Date(2026, 5, 14, 11, 15, 0, 0, time.UTC)
	svc := &mockSvc{
		submitSessionFn: func(_ context.Context, sessionID, userID, _, _ string) (*SubmitSessionResponse, error) {
			assert.Equal(t, "sess-42", sessionID)
			assert.Equal(t, "user-1", userID)
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "submitted",
				SubmittedAt: submittedAt,
				ScorePct:    &score,
				Passed:      &passed,
			}, nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-42"))

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sess-42", data["session_id"])
	assert.Equal(t, "submitted", data["status"])
	assert.NotNil(t, data["score_pct"])
	assert.NotNil(t, data["passed"])
}

// AC-4: grading_pending → 200 with null score_pct and null passed.
func TestSubmitSession_Handler_200_GradingPending(t *testing.T) {
	submittedAt := time.Date(2026, 5, 14, 11, 15, 0, 0, time.UTC)
	svc := &mockSvc{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "grading_pending",
				SubmittedAt: submittedAt,
				ScorePct:    nil,
				Passed:      nil,
			}, nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-43"))

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "grading_pending", data["status"])
	assert.Nil(t, data["score_pct"])
	assert.Nil(t, data["passed"])
}

// AC-8: already submitted → 200 idempotent.
func TestSubmitSession_Handler_200_Idempotent(t *testing.T) {
	score := 90.0
	passed := true
	submittedAt := time.Date(2026, 5, 14, 11, 10, 0, 0, time.UTC)
	svc := &mockSvc{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "submitted",
				SubmittedAt: submittedAt,
				ScorePct:    &score,
				Passed:      &passed,
			}, nil
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-42"))

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	data, _ := body["data"].(map[string]any)
	assert.Equal(t, "submitted", data["status"])
}

// AC-1: wrong user → 403.
func TestSubmitSession_Handler_403_Forbidden(t *testing.T) {
	svc := &mockSvc{
		submitSessionFn: func(_ context.Context, _, _, _, _ string) (*SubmitSessionResponse, error) {
			return nil, ErrSessionForbidden
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-1"))

	assert.Equal(t, http.StatusForbidden, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "FORBIDDEN", errObj["code"])
}

// Session not found → 404.
func TestSubmitSession_Handler_404_NotFound(t *testing.T) {
	svc := &mockSvc{
		submitSessionFn: func(_ context.Context, _, _, _, _ string) (*SubmitSessionResponse, error) {
			return nil, ErrSessionNotFound
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-missing"))

	assert.Equal(t, http.StatusNotFound, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "SESSION_NOT_FOUND", errObj["code"])
}

// Internal error → 500.
func TestSubmitSession_Handler_500(t *testing.T) {
	svc := &mockSvc{
		submitSessionFn: func(_ context.Context, _, _, _, _ string) (*SubmitSessionResponse, error) {
			return nil, errors.New("db down")
		},
	}
	h := NewHandler(svc)
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-1"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
