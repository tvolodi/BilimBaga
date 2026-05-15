package reports

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Manual mock service ───────────────────────────────────────────────────────

type mockSvc struct {
	getDashboardFn    func(ctx context.Context) (*DashboardMetrics, error)
	getExamAnalyticsFn func(ctx context.Context, examID string) (*ExamAnalyticsResponse, error)
}

func (m *mockSvc) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	if m.getDashboardFn != nil {
		return m.getDashboardFn(ctx)
	}
	return nil, errors.New("not configured")
}

func (m *mockSvc) GetExamAnalytics(ctx context.Context, examID string) (*ExamAnalyticsResponse, error) {
	if m.getExamAnalyticsFn != nil {
		return m.getExamAnalyticsFn(ctx, examID)
	}
	return nil, errors.New("not configured")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildFullMetrics() *DashboardMetrics {
	deadline := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
	submittedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	score := 84.5
	security := 71.4
	safety := 83.2
	return &DashboardMetrics{
		CompletionRateByExam: []*ExamCompletionRate{
			{ExamID: "exam-uuid", Title: "Fire Safety Fundamentals", AssignedCount: 150, CompletedCount: 112, PassedCount: 98},
		},
		OverdueEmployees: []*OverdueEmployee{
			{UserID: "user-uuid", Name: "Aibek Seitkali", ExamTitle: "Security Awareness", Deadline: deadline},
		},
		RecentActivity: []*RecentActivity{
			{SessionID: "sess-uuid", EmployeeName: "Aibek Seitkali", ExamTitle: "Fire Safety Fundamentals", ScorePct: &score, Passed: true, SubmittedAt: submittedAt},
		},
		AvgScoreByTrack: TrackScores{
			Security: &security,
			Safety:   &safety,
			Loyalty:  nil,
		},
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// AC-7: HTTP 200 with all four top-level keys in the response.
func TestGetDashboard_200_AllKeysPresent(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			return buildFullMetrics(), nil
		},
	}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	// Top-level envelope keys.
	_, hasData := body["data"]
	_, hasError := body["error"]
	assert.True(t, hasData, "response must contain 'data' key")
	assert.True(t, hasError, "response must contain 'error' key")

	// error field must be null.
	assert.Equal(t, "null", string(body["error"]))

	// data must contain all four dashboard keys.
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body["data"], &data))
	assert.Contains(t, data, "completion_rate_by_exam")
	assert.Contains(t, data, "overdue_employees")
	assert.Contains(t, data, "recent_activity")
	assert.Contains(t, data, "avg_score_by_track")
}

// AC-5: absent loyalty track must serialise as JSON null, not 0.
func TestGetDashboard_200_LoyaltyTrackIsNull(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			return buildFullMetrics(), nil
		},
	}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data struct {
			AvgScoreByTrack struct {
				Security *float64 `json:"security"`
				Safety   *float64 `json:"safety"`
				Loyalty  *float64 `json:"loyalty"`
			} `json:"avg_score_by_track"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	require.NotNil(t, envelope.Data.AvgScoreByTrack.Security)
	assert.Equal(t, 71.4, *envelope.Data.AvgScoreByTrack.Security)
	require.NotNil(t, envelope.Data.AvgScoreByTrack.Safety)
	assert.Equal(t, 83.2, *envelope.Data.AvgScoreByTrack.Safety)
	assert.Nil(t, envelope.Data.AvgScoreByTrack.Loyalty)
}

// AC-7: empty arrays are still present and valid JSON.
func TestGetDashboard_200_EmptyArraysPresent(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			return &DashboardMetrics{
				CompletionRateByExam: []*ExamCompletionRate{},
				OverdueEmployees:     []*OverdueEmployee{},
				RecentActivity:       []*RecentActivity{},
				AvgScoreByTrack:      TrackScores{},
			}, nil
		},
	}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data struct {
			CompletionRateByExam []json.RawMessage `json:"completion_rate_by_exam"`
			OverdueEmployees     []json.RawMessage `json:"overdue_employees"`
			RecentActivity       []json.RawMessage `json:"recent_activity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	assert.Len(t, envelope.Data.CompletionRateByExam, 0)
	assert.Len(t, envelope.Data.OverdueEmployees, 0)
	assert.Len(t, envelope.Data.RecentActivity, 0)
}

// 500: service returns an error → HTTP 500 with error envelope.
func TestGetDashboard_500_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			return nil, errors.New("database unavailable")
		},
	}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "null", string(body["data"]))

	var errObj map[string]string
	require.NoError(t, json.Unmarshal(body["error"], &errObj))
	assert.Equal(t, "ERR_INTERNAL", errObj["code"])
}

// Completion rate fields map correctly.
func TestGetDashboard_200_CompletionRateFields(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			return buildFullMetrics(), nil
		},
	}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data struct {
			CompletionRateByExam []struct {
				ExamID         string `json:"exam_id"`
				Title          string `json:"title"`
				AssignedCount  int    `json:"assigned_count"`
				CompletedCount int    `json:"completed_count"`
				PassedCount    int    `json:"passed_count"`
			} `json:"completion_rate_by_exam"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Len(t, envelope.Data.CompletionRateByExam, 1)

	cr := envelope.Data.CompletionRateByExam[0]
	assert.Equal(t, "exam-uuid", cr.ExamID)
	assert.Equal(t, "Fire Safety Fundamentals", cr.Title)
	assert.Equal(t, 150, cr.AssignedCount)
	assert.Equal(t, 112, cr.CompletedCount)
	assert.Equal(t, 98, cr.PassedCount)
}

// Recent activity fields map correctly.
func TestGetDashboard_200_RecentActivityFields(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			return buildFullMetrics(), nil
		},
	}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data struct {
			RecentActivity []struct {
				SessionID    string   `json:"session_id"`
				EmployeeName string   `json:"employee_name"`
				ExamTitle    string   `json:"exam_title"`
				ScorePct     *float64 `json:"score_pct"`
				Passed       bool     `json:"passed"`
			} `json:"recent_activity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Len(t, envelope.Data.RecentActivity, 1)

	ra := envelope.Data.RecentActivity[0]
	assert.Equal(t, "sess-uuid", ra.SessionID)
	assert.Equal(t, "Aibek Seitkali", ra.EmployeeName)
	assert.True(t, ra.Passed)
	require.NotNil(t, ra.ScorePct)
	assert.Equal(t, 84.5, *ra.ScorePct)
}

// ── FR-BB52: GetExamAnalytics handler tests ──────────────────────────────────

// newRequestWithID builds an httptest.Request with chi URL param "id" set.
func newRequestWithID(method, url, id string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// AC-1: HTTP 200 with correct envelope structure.
func TestGetExamAnalytics_200_HappyPath(t *testing.T) {
	pr := 0.72
	avg := 74.3
	median := 76.0
	svc := &mockSvc{
		getExamAnalyticsFn: func(_ context.Context, _ string) (*ExamAnalyticsResponse, error) {
			return &ExamAnalyticsResponse{
				ExamID:             "exam-uuid",
				ExamTitle:          "Fire Safety",
				ScoreDistribution:  fillBuckets(nil),
				PassRate:           pr,
				AvgScore:           &avg,
				MedianScore:        &median,
				TotalAttempts:      120,
				UniqueParticipants: 98,
				PerQuestionStats:   []QuestionStat{},
			}, nil
		},
	}
	h := NewHandler(svc)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/exam-uuid/analytics", "exam-uuid")
	w := httptest.NewRecorder()
	h.GetExamAnalytics(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "null", string(body["error"]))

	var data ExamAnalyticsResponse
	require.NoError(t, json.Unmarshal(body["data"], &data))
	assert.Equal(t, "exam-uuid", data.ExamID)
	assert.Equal(t, "Fire Safety", data.ExamTitle)
	assert.Len(t, data.ScoreDistribution, 10)
	assert.Equal(t, 120, data.TotalAttempts)
	assert.Equal(t, 98, data.UniqueParticipants)
}

// AC-1: HTTP 404 when exam is not found.
func TestGetExamAnalytics_404_ExamNotFound(t *testing.T) {
	svc := &mockSvc{
		getExamAnalyticsFn: func(_ context.Context, _ string) (*ExamAnalyticsResponse, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/nonexistent/analytics", "nonexistent")
	w := httptest.NewRecorder()
	h.GetExamAnalytics(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	var errObj map[string]string
	require.NoError(t, json.Unmarshal(body["error"], &errObj))
	assert.Equal(t, "EXAM_NOT_FOUND", errObj["code"])
}

// HTTP 500 when service returns unexpected error.
func TestGetExamAnalytics_500_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getExamAnalyticsFn: func(_ context.Context, _ string) (*ExamAnalyticsResponse, error) {
			return nil, errors.New("database connection lost")
		},
	}
	h := NewHandler(svc)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/exam-uuid/analytics", "exam-uuid")
	w := httptest.NewRecorder()
	h.GetExamAnalytics(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	var errObj map[string]string
	require.NoError(t, json.Unmarshal(body["error"], &errObj))
	assert.Equal(t, "INTERNAL_ERROR", errObj["code"])
}
