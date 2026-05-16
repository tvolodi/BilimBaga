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
	getDashboardFn         func(ctx context.Context) (*DashboardMetrics, error)
	getExamAnalyticsFn     func(ctx context.Context, examID string) (*ExamAnalyticsResponse, error)
	getUserRecordFn        func(ctx context.Context, userID string, page, perPage int) (*UserRecordResponse, int, error)
	getUserProgressFn      func(ctx context.Context, userID string) (*UserProgressResponse, error)
	streamExamResultsCSVFn func(ctx context.Context, w http.ResponseWriter, examID, tenantID string) error
	streamUserRecordCSVFn  func(ctx context.Context, w http.ResponseWriter, userID, tenantID string) error
	buildDashboardReportFn func(ctx context.Context, tenantID string, from, to time.Time, companyName, logoBase64 string) (*DashboardReportData, error)
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

func (m *mockSvc) GetUserRecord(ctx context.Context, userID string, page, perPage int) (*UserRecordResponse, int, error) {
	if m.getUserRecordFn != nil {
		return m.getUserRecordFn(ctx, userID, page, perPage)
	}
	return nil, 0, errors.New("not configured")
}

func (m *mockSvc) GetUserProgress(ctx context.Context, userID string) (*UserProgressResponse, error) {
	if m.getUserProgressFn != nil {
		return m.getUserProgressFn(ctx, userID)
	}
	return nil, errors.New("not configured")
}

func (m *mockSvc) StreamExamResultsCSV(ctx context.Context, w http.ResponseWriter, examID, tenantID string) error {
	if m.streamExamResultsCSVFn != nil {
		return m.streamExamResultsCSVFn(ctx, w, examID, tenantID)
	}
	return nil
}

func (m *mockSvc) StreamUserRecordCSV(ctx context.Context, w http.ResponseWriter, userID, tenantID string) error {
	if m.streamUserRecordCSVFn != nil {
		return m.streamUserRecordCSVFn(ctx, w, userID, tenantID)
	}
	return nil
}

func (m *mockSvc) BuildDashboardReport(ctx context.Context, tenantID string, from, to time.Time, companyName, logoBase64 string) (*DashboardReportData, error) {
	if m.buildDashboardReportFn != nil {
		return m.buildDashboardReportFn(ctx, tenantID, from, to, companyName, logoBase64)
	}
	return &DashboardReportData{
		From:            from.Format("02 Jan 2006"),
		To:              to.Format("02 Jan 2006"),
		CompanyName:     companyName,
		LogoBase64:      logoBase64,
		CompletionRates: []*ExamCompletionRate{},
		TopQuestions:    []QuestionStat{},
		BottomQuestions: []QuestionStat{},
	}, nil
}

// mockTenantCfg is a no-op TenantConfigProvider for tests.
type mockTenantCfg struct{}

func (m *mockTenantCfg) GetAllConfig() map[string]json.RawMessage {
	return map[string]json.RawMessage{}
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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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
	h := NewHandler(svc, nil)

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

// ── FR-BB53: GetUserRecord handler tests ─────────────────────────────────────

// AC-1: HTTP 200 with data + meta + error:null on happy path.
func TestGetUserRecord_200_HappyPath(t *testing.T) {
	submittedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	passed := true
	timeSec := 1800
	svc := &mockSvc{
		getUserRecordFn: func(_ context.Context, userID string, page, perPage int) (*UserRecordResponse, int, error) {
			return &UserRecordResponse{
				UserID:     userID,
				FullName:   "Aibek Seitkali",
				Department: "Operations",
				Sessions: []SessionRecord{
					{
						SessionID:        "sess-uuid",
						ExamID:           "exam-uuid",
						ExamTitle:        "Fire Safety Fundamentals",
						StartedAt:        time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
						SubmittedAt:      &submittedAt,
						ScorePct:         func() *float64 { v := 84.50; return &v }(),
						Passed:           &passed,
						TimeTakenSeconds: &timeSec,
						Status:           "submitted",
						CertificateID:    func() *string { s := "cert-uuid"; return &s }(),
					},
				},
			}, 7, nil
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/user-uuid/record?page=1&per_page=20", "user-uuid")
	w := httptest.NewRecorder()
	h.GetUserRecord(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data  *UserRecordResponse    `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Error *string                `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Nil(t, envelope.Error)
	require.NotNil(t, envelope.Data)
	assert.Equal(t, "user-uuid", envelope.Data.UserID)
	assert.Equal(t, "Aibek Seitkali", envelope.Data.FullName)
	assert.Equal(t, "Operations", envelope.Data.Department)
	require.Len(t, envelope.Data.Sessions, 1)

	require.NotNil(t, envelope.Meta)
	assert.Equal(t, float64(1), envelope.Meta["page"])
	assert.Equal(t, float64(20), envelope.Meta["per_page"])
	assert.Equal(t, float64(7), envelope.Meta["total"])
}

// AC-2: HTTP 404 when user does not exist.
func TestGetUserRecord_404_UserNotFound(t *testing.T) {
	svc := &mockSvc{
		getUserRecordFn: func(_ context.Context, _ string, _, _ int) (*UserRecordResponse, int, error) {
			return nil, 0, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/missing/record", "missing")
	w := httptest.NewRecorder()
	h.GetUserRecord(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	var errObj map[string]string
	require.NoError(t, json.Unmarshal(body["error"], &errObj))
	assert.Equal(t, "USER_NOT_FOUND", errObj["code"])
}

// HTTP 500 on unexpected service error.
func TestGetUserRecord_500_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getUserRecordFn: func(_ context.Context, _ string, _, _ int) (*UserRecordResponse, int, error) {
			return nil, 0, errors.New("db failure")
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/user-uuid/record", "user-uuid")
	w := httptest.NewRecorder()
	h.GetUserRecord(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// AC-5: per_page capped at 100.
func TestGetUserRecord_PerPageCappedAt100(t *testing.T) {
	var capturedPerPage int
	svc := &mockSvc{
		getUserRecordFn: func(_ context.Context, _ string, page, perPage int) (*UserRecordResponse, int, error) {
			capturedPerPage = perPage
			return &UserRecordResponse{Sessions: []SessionRecord{}}, 0, nil
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/user-uuid/record?per_page=500", "user-uuid")
	w := httptest.NewRecorder()
	h.GetUserRecord(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 100, capturedPerPage)
}

// ── FR-BB53: GetUserProgress handler tests ───────────────────────────────────

// AC-1: HTTP 200 with three tracks and error:null.
func TestGetUserProgress_200_HappyPath(t *testing.T) {
	ts := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	svc := &mockSvc{
		getUserProgressFn: func(_ context.Context, userID string) (*UserProgressResponse, error) {
			return &UserProgressResponse{
				UserID:   userID,
				FullName: "Aibek Seitkali",
				Tracks: []TrackSummary{
					{Track: "security", QuestionsAnswered: 120, LastActivity: &ts, RequiredExams: []ExamProgress{}},
					{Track: "safety", QuestionsAnswered: 45, LastActivity: nil, RequiredExams: []ExamProgress{}},
					{Track: "loyalty", QuestionsAnswered: 0, LastActivity: nil, RequiredExams: []ExamProgress{}},
				},
			}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/user-uuid/progress", "user-uuid")
	w := httptest.NewRecorder()
	h.GetUserProgress(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data  *UserProgressResponse `json:"data"`
		Error *string               `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Nil(t, envelope.Error)
	require.NotNil(t, envelope.Data)
	assert.Equal(t, "user-uuid", envelope.Data.UserID)
	require.Len(t, envelope.Data.Tracks, 3)
	assert.Equal(t, "security", envelope.Data.Tracks[0].Track)
	assert.Equal(t, 120, envelope.Data.Tracks[0].QuestionsAnswered)
}

// AC-2: HTTP 404 when user does not exist.
func TestGetUserProgress_404_UserNotFound(t *testing.T) {
	svc := &mockSvc{
		getUserProgressFn: func(_ context.Context, _ string) (*UserProgressResponse, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/missing/progress", "missing")
	w := httptest.NewRecorder()
	h.GetUserProgress(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	var errObj map[string]string
	require.NoError(t, json.Unmarshal(body["error"], &errObj))
	assert.Equal(t, "USER_NOT_FOUND", errObj["code"])
}

// HTTP 500 on unexpected service error.
func TestGetUserProgress_500_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getUserProgressFn: func(_ context.Context, _ string) (*UserProgressResponse, error) {
			return nil, errors.New("db failure")
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/user-uuid/progress", "user-uuid")
	w := httptest.NewRecorder()
	h.GetUserProgress(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── FR-BB54: ExamResultsCSV handler tests ───────────────────────────────────

// AC-2,3: Content-Type and Content-Disposition headers set; CSV header row present.
func TestExamResultsCSV_200_HeadersAndContentType(t *testing.T) {
	svc := &mockSvc{
		streamExamResultsCSVFn: func(_ context.Context, w http.ResponseWriter, _, _ string) error {
			_, _ = w.Write([]byte("employee_name,department,started_at,submitted_at,score_pct,passed,time_taken_seconds\n"))
			_, _ = w.Write([]byte("Aibek Seitkali,Operations,2026-05-14T10:00:00Z,2026-05-14T10:30:00Z,84.50,true,1800\n"))
			return nil
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/exam-uuid/results/export", "exam-uuid")
	w := httptest.NewRecorder()
	h.ExamResultsCSV(w, req)

	assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "exam-uuid")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".csv")
	body := w.Body.String()
	assert.Contains(t, body, "employee_name")
	assert.Contains(t, body, "Aibek Seitkali")
}

// AC-10: Filename contains exam ID and a date (YYYYMMDD format).
func TestExamResultsCSV_200_FilenameContainsExamID(t *testing.T) {
	svc := &mockSvc{}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/my-exam-id/results/export", "my-exam-id")
	w := httptest.NewRecorder()
	h.ExamResultsCSV(w, req)

	disp := w.Header().Get("Content-Disposition")
	assert.Contains(t, disp, "results-my-exam-id-")
}

// Missing exam id → 400.
func TestExamResultsCSV_400_MissingID(t *testing.T) {
	svc := &mockSvc{}
	h := NewHandler(svc, nil)

	// Build request with empty "id" param.
	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams//results/export", "")
	w := httptest.NewRecorder()
	h.ExamResultsCSV(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── FR-BB54: UserRecordCSV handler tests ────────────────────────────────────

// AC-2,5: Content-Type and Content-Disposition set; CSV header row present.
func TestUserRecordCSV_200_HeadersAndContentType(t *testing.T) {
	svc := &mockSvc{
		streamUserRecordCSVFn: func(_ context.Context, w http.ResponseWriter, _, _ string) error {
			_, _ = w.Write([]byte("exam_title,started_at,submitted_at,score_pct,passed,time_taken_seconds,status\n"))
			_, _ = w.Write([]byte("Fire Safety,2026-05-14T10:00:00Z,2026-05-14T10:30:00Z,84.50,true,1800,submitted\n"))
			return nil
		},
	}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/user-uuid/record/export", "user-uuid")
	w := httptest.NewRecorder()
	h.UserRecordCSV(w, req)

	assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "user-uuid")
	body := w.Body.String()
	assert.Contains(t, body, "exam_title")
	assert.Contains(t, body, "Fire Safety")
}

// AC-10: Filename contains user ID.
func TestUserRecordCSV_200_FilenameContainsUserID(t *testing.T) {
	svc := &mockSvc{}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users/my-user-id/record/export", "my-user-id")
	w := httptest.NewRecorder()
	h.UserRecordCSV(w, req)

	disp := w.Header().Get("Content-Disposition")
	assert.Contains(t, disp, "record-my-user-id-")
}

// Missing user id → 400.
func TestUserRecordCSV_400_MissingID(t *testing.T) {
	svc := &mockSvc{}
	h := NewHandler(svc, nil)

	req := newRequestWithID(http.MethodGet, "/api/v1/admin/users//record/export", "")
	w := httptest.NewRecorder()
	h.UserRecordCSV(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── FR-BB54: DashboardExportPDF handler tests ────────────────────────────────

// AC-9: Content-Type is application/pdf on success.
func TestDashboardExportPDF_200_ContentTypePDF(t *testing.T) {
	svc := &mockSvc{} // default BuildDashboardReport returns empty data
	h := NewHandler(svc, &mockTenantCfg{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/export", nil)
	w := httptest.NewRecorder()
	h.DashboardExportPDF(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".pdf")
	// PDF magic bytes: %PDF
	assert.True(t, len(w.Body.Bytes()) > 4)
}

// AC-10: Filename contains from/to dates when supplied.
func TestDashboardExportPDF_200_FilenameContainsDates(t *testing.T) {
	svc := &mockSvc{}
	h := NewHandler(svc, &mockTenantCfg{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/export?from=2026-04-01&to=2026-04-30", nil)
	w := httptest.NewRecorder()
	h.DashboardExportPDF(w, req)

	disp := w.Header().Get("Content-Disposition")
	assert.Contains(t, disp, "dashboard-report-20260401-20260430")
}

// AC-6: when from/to omitted the handler defaults to last 30 days (just verifying 200).
func TestDashboardExportPDF_200_DefaultDateRange(t *testing.T) {
	svc := &mockSvc{}
	h := NewHandler(svc, &mockTenantCfg{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/export", nil)
	w := httptest.NewRecorder()
	h.DashboardExportPDF(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Service error → 500.
func TestDashboardExportPDF_500_ServiceError(t *testing.T) {
	svc := &mockSvc{
		buildDashboardReportFn: func(_ context.Context, _ string, _, _ time.Time, _, _ string) (*DashboardReportData, error) {
			return nil, errors.New("db failure")
		},
	}
	h := NewHandler(svc, &mockTenantCfg{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/export", nil)
	w := httptest.NewRecorder()
	h.DashboardExportPDF(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
