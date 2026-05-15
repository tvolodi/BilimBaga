package portal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock service ──────────────────────────────────────────────────────────────

type mockSvc struct {
	listMyExamsFn func(ctx context.Context, userID, deptID string) ([]*PortalExamItem, error)
	getMyExamFn   func(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error)
}

func (m *mockSvc) ListMyExams(ctx context.Context, userID, deptID string) ([]*PortalExamItem, error) {
	if m.listMyExamsFn != nil {
		return m.listMyExamsFn(ctx, userID, deptID)
	}
	return []*PortalExamItem{}, nil
}

func (m *mockSvc) GetMyExam(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error) {
	if m.getMyExamFn != nil {
		return m.getMyExamFn(ctx, examID, userID, deptID)
	}
	return nil, ErrNotAssigned
}

// ── Test helpers ─────────────────────────────────────────────────────────────

// withChiParam injects a Chi URL parameter into the request context.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withUserCtx injects user ID and department ID into the request context
// using the same keys the auth middleware writes.
func withUserCtx(r *http.Request, userID, deptID string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, userID)
	ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, deptID)
	return r.WithContext(ctx)
}

func decode(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

// ── ListMyExams handler tests ─────────────────────────────────────────────────

func TestListMyExams_OK_Empty(t *testing.T) {
	h := NewHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/portal/exams", nil)
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.ListMyExams(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decode(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].([]any)
	require.True(t, ok)
	assert.Empty(t, data)
}

func TestListMyExams_OK_WithItems(t *testing.T) {
	deadline := time.Date(2026, 7, 1, 23, 59, 59, 0, time.UTC)
	openID := "session-uuid-42"
	items := []*PortalExamItem{
		{
			ID:               "exam-1",
			Title:            "Go Developer Certification",
			TimeLimitMinutes: 90,
			PassingScorePct:  70.0,
			MaxAttempts:      2,
			AttemptsUsed:     1,
			Deadline:         &deadline,
			UserStatus:       UserStatusInProgress,
			OpenSessionID:    &openID,
		},
		{
			ID:               "exam-2",
			Title:            "Safety Induction",
			TimeLimitMinutes: 30,
			PassingScorePct:  80.0,
			MaxAttempts:      3,
			UserStatus:       UserStatusNotStarted,
		},
	}
	svc := &mockSvc{
		listMyExamsFn: func(ctx context.Context, userID, deptID string) ([]*PortalExamItem, error) {
			return items, nil
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/exams", nil)
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.ListMyExams(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decode(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 2)
}

func TestListMyExams_InternalError(t *testing.T) {
	svc := &mockSvc{
		listMyExamsFn: func(ctx context.Context, userID, deptID string) ([]*PortalExamItem, error) {
			return nil, errors.New("db exploded")
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/exams", nil)
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.ListMyExams(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetMyExam handler tests ───────────────────────────────────────────────────

func TestGetMyExam_OK(t *testing.T) {
	detail := &PortalExamDetail{
		ID:             "exam-1",
		Title:          "Go Developer Certification",
		UserStatus:     UserStatusNotStarted,
		AttemptHistory: []AttemptHistory{},
	}
	svc := &mockSvc{
		getMyExamFn: func(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error) {
			assert.Equal(t, "exam-1", examID)
			return detail, nil
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/exams/exam-1", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.GetMyExam(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decode(t, w.Body.Bytes())
	assert.Nil(t, body["error"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "exam-1", data["id"])
}

func TestGetMyExam_NotAssigned_Returns403(t *testing.T) {
	// AC-7: 403 when exam not assigned to user.
	svc := &mockSvc{
		getMyExamFn: func(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error) {
			return nil, ErrNotAssigned
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/exams/exam-99", nil)
	req = withChiParam(req, "id", "exam-99")
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.GetMyExam(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	body := decode(t, w.Body.Bytes())
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "EXAM_NOT_ASSIGNED", errObj["code"])
}

func TestGetMyExam_NotAssigned_WrappedError_Returns403(t *testing.T) {
	// ErrNotAssigned wrapped inside another error also produces 403.
	svc := &mockSvc{
		getMyExamFn: func(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error) {
			return nil, fmt.Errorf("portal: GetMyExam: %w", ErrNotAssigned)
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/exams/exam-1", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.GetMyExam(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetMyExam_InternalError(t *testing.T) {
	svc := &mockSvc{
		getMyExamFn: func(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/portal/exams/exam-1", nil)
	req = withChiParam(req, "id", "exam-1")
	req = withUserCtx(req, "user1", "dept1")
	w := httptest.NewRecorder()

	h.GetMyExam(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
