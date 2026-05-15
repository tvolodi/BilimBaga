package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Manual mock repository ────────────────────────────────────────────────────

type mockRepo struct {
	getCompletionFn         func(ctx context.Context) ([]*ExamCompletionRate, error)
	getOverdueFn            func(ctx context.Context) ([]*OverdueEmployee, error)
	getRecentFn             func(ctx context.Context) ([]*RecentActivity, error)
	getAvgScoreByTrack      func(ctx context.Context) (map[string]*float64, error)
	getExamTitleFn          func(ctx context.Context, examID string) (string, error)
	getScoreDistFn          func(ctx context.Context, examID string) ([]BucketCount, error)
	getSummaryStatsFn       func(ctx context.Context, examID string) (*examSummaryRow, error)
	getPerQuestionStatsFn   func(ctx context.Context, examID string) ([]questionStatRow, error)
	getAnswerDistFn         func(ctx context.Context, examID string) ([]answerDistRow, error)
	getUserInfoFn           func(ctx context.Context, userID string) (*userInfoRow, error)
	getUserSessionHistoryFn func(ctx context.Context, userID string, limit, offset int) ([]SessionRecord, error)
	getUserSessionCountFn   func(ctx context.Context, userID string) (int, error)
	getUserTrackActivityFn  func(ctx context.Context, userID string) ([]TrackActivity, error)
	getUserRequiredExamsFn  func(ctx context.Context, userID string) ([]ExamProgress, error)
}

func (m *mockRepo) GetCompletionRateByExam(ctx context.Context) ([]*ExamCompletionRate, error) {
	if m.getCompletionFn != nil {
		return m.getCompletionFn(ctx)
	}
	return []*ExamCompletionRate{}, nil
}

func (m *mockRepo) GetOverdueEmployees(ctx context.Context) ([]*OverdueEmployee, error) {
	if m.getOverdueFn != nil {
		return m.getOverdueFn(ctx)
	}
	return []*OverdueEmployee{}, nil
}

func (m *mockRepo) GetRecentActivity(ctx context.Context) ([]*RecentActivity, error) {
	if m.getRecentFn != nil {
		return m.getRecentFn(ctx)
	}
	return []*RecentActivity{}, nil
}

func (m *mockRepo) GetAvgScoreByTrack(ctx context.Context) (map[string]*float64, error) {
	if m.getAvgScoreByTrack != nil {
		return m.getAvgScoreByTrack(ctx)
	}
	return map[string]*float64{}, nil
}

func (m *mockRepo) GetExamTitle(ctx context.Context, examID string) (string, error) {
	if m.getExamTitleFn != nil {
		return m.getExamTitleFn(ctx, examID)
	}
	return "Test Exam", nil
}

func (m *mockRepo) GetExamScoreDistribution(ctx context.Context, examID string) ([]BucketCount, error) {
	if m.getScoreDistFn != nil {
		return m.getScoreDistFn(ctx, examID)
	}
	return []BucketCount{}, nil
}

func (m *mockRepo) GetExamSummaryStats(ctx context.Context, examID string) (*examSummaryRow, error) {
	if m.getSummaryStatsFn != nil {
		return m.getSummaryStatsFn(ctx, examID)
	}
	return &examSummaryRow{}, nil
}

func (m *mockRepo) GetPerQuestionStats(ctx context.Context, examID string) ([]questionStatRow, error) {
	if m.getPerQuestionStatsFn != nil {
		return m.getPerQuestionStatsFn(ctx, examID)
	}
	return []questionStatRow{}, nil
}

func (m *mockRepo) GetAnswerDistribution(ctx context.Context, examID string) ([]answerDistRow, error) {
	if m.getAnswerDistFn != nil {
		return m.getAnswerDistFn(ctx, examID)
	}
	return []answerDistRow{}, nil
}

func (m *mockRepo) GetUserInfo(ctx context.Context, userID string) (*userInfoRow, error) {
	if m.getUserInfoFn != nil {
		return m.getUserInfoFn(ctx, userID)
	}
	dept := "Engineering"
	return &userInfoRow{ID: userID, FullName: "Test User", DepartmentName: &dept}, nil
}

func (m *mockRepo) GetUserSessionHistory(ctx context.Context, userID string, limit, offset int) ([]SessionRecord, error) {
	if m.getUserSessionHistoryFn != nil {
		return m.getUserSessionHistoryFn(ctx, userID, limit, offset)
	}
	return []SessionRecord{}, nil
}

func (m *mockRepo) GetUserSessionCount(ctx context.Context, userID string) (int, error) {
	if m.getUserSessionCountFn != nil {
		return m.getUserSessionCountFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockRepo) GetUserTrackActivity(ctx context.Context, userID string) ([]TrackActivity, error) {
	if m.getUserTrackActivityFn != nil {
		return m.getUserTrackActivityFn(ctx, userID)
	}
	return []TrackActivity{}, nil
}

func (m *mockRepo) GetUserRequiredExams(ctx context.Context, userID string) ([]ExamProgress, error) {
	if m.getUserRequiredExamsFn != nil {
		return m.getUserRequiredExamsFn(ctx, userID)
	}
	return []ExamProgress{}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func ptr(f float64) *float64 { return &f }

// ── Tests ─────────────────────────────────────────────────────────────────────

// AC-7: endpoint returns HTTP 200 with all four keys present even when individual
// arrays are empty or track averages are null.
func TestGetDashboardMetrics_HappyPath_EmptyData(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// All four keys must be present.
	assert.NotNil(t, metrics.CompletionRateByExam)
	assert.NotNil(t, metrics.OverdueEmployees)
	assert.NotNil(t, metrics.RecentActivity)
	// TrackScores is a struct — always present. All tracks nil when no data.
	assert.Nil(t, metrics.AvgScoreByTrack.Security)
	assert.Nil(t, metrics.AvgScoreByTrack.Safety)
	assert.Nil(t, metrics.AvgScoreByTrack.Loyalty)
}

// AC-2: completion_rate_by_exam data is passed through correctly.
func TestGetDashboardMetrics_CompletionRatePopulated(t *testing.T) {
	expected := []*ExamCompletionRate{
		{ExamID: "exam-1", Title: "Fire Safety", AssignedCount: 150, CompletedCount: 112, PassedCount: 98},
	}
	repo := &mockRepo{
		getCompletionFn: func(_ context.Context) ([]*ExamCompletionRate, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, metrics.CompletionRateByExam, 1)
	assert.Equal(t, "exam-1", metrics.CompletionRateByExam[0].ExamID)
	assert.Equal(t, 150, metrics.CompletionRateByExam[0].AssignedCount)
	assert.Equal(t, 112, metrics.CompletionRateByExam[0].CompletedCount)
	assert.Equal(t, 98, metrics.CompletionRateByExam[0].PassedCount)
}

// AC-3: overdue_employees list is passed through correctly.
func TestGetDashboardMetrics_OverdueEmployeesPopulated(t *testing.T) {
	deadline := time.Now().Add(-24 * time.Hour)
	expected := []*OverdueEmployee{
		{UserID: "user-1", Name: "Aibek Seitkali", ExamTitle: "Security Awareness", Deadline: deadline},
	}
	repo := &mockRepo{
		getOverdueFn: func(_ context.Context) ([]*OverdueEmployee, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, metrics.OverdueEmployees, 1)
	assert.Equal(t, "user-1", metrics.OverdueEmployees[0].UserID)
	assert.Equal(t, "Aibek Seitkali", metrics.OverdueEmployees[0].Name)
}

// AC-4: recent_activity list is passed through correctly.
func TestGetDashboardMetrics_RecentActivityPopulated(t *testing.T) {
	submittedAt := time.Now().Add(-1 * time.Hour)
	scorePct := 84.5
	expected := []*RecentActivity{
		{
			SessionID:    "sess-1",
			EmployeeName: "Aibek Seitkali",
			ExamTitle:    "Fire Safety",
			ScorePct:     &scorePct,
			Passed:       true,
			SubmittedAt:  submittedAt,
		},
	}
	repo := &mockRepo{
		getRecentFn: func(_ context.Context) ([]*RecentActivity, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, metrics.RecentActivity, 1)
	assert.Equal(t, "sess-1", metrics.RecentActivity[0].SessionID)
	assert.Equal(t, 84.5, *metrics.RecentActivity[0].ScorePct)
	assert.True(t, metrics.RecentActivity[0].Passed)
}

// AC-5/AC-6: avg_score_by_track is populated when data exists; absent tracks remain null.
func TestGetDashboardMetrics_TrackScoresPartial(t *testing.T) {
	repo := &mockRepo{
		getAvgScoreByTrack: func(_ context.Context) (map[string]*float64, error) {
			return map[string]*float64{
				"security": ptr(71.4),
				"safety":   ptr(83.2),
				// loyalty absent → null
			}, nil
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, metrics.AvgScoreByTrack.Security)
	assert.Equal(t, 71.4, *metrics.AvgScoreByTrack.Security)
	require.NotNil(t, metrics.AvgScoreByTrack.Safety)
	assert.Equal(t, 83.2, *metrics.AvgScoreByTrack.Safety)
	// AC-5: loyalty must be JSON null (nil pointer), not 0.
	assert.Nil(t, metrics.AvgScoreByTrack.Loyalty)
}

// All three tracks present.
func TestGetDashboardMetrics_AllTracksPresent(t *testing.T) {
	repo := &mockRepo{
		getAvgScoreByTrack: func(_ context.Context) (map[string]*float64, error) {
			return map[string]*float64{
				"security": ptr(71.4),
				"safety":   ptr(83.2),
				"loyalty":  ptr(55.0),
			}, nil
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 71.4, *metrics.AvgScoreByTrack.Security)
	assert.Equal(t, 83.2, *metrics.AvgScoreByTrack.Safety)
	assert.Equal(t, 55.0, *metrics.AvgScoreByTrack.Loyalty)
}

// Error branch: completion rate query fails → service returns error.
func TestGetDashboardMetrics_CompletionError(t *testing.T) {
	repo := &mockRepo{
		getCompletionFn: func(_ context.Context) ([]*ExamCompletionRate, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	assert.Nil(t, metrics)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// Error branch: overdue query fails → service returns error.
func TestGetDashboardMetrics_OverdueError(t *testing.T) {
	repo := &mockRepo{
		getOverdueFn: func(_ context.Context) ([]*OverdueEmployee, error) {
			return nil, errors.New("overdue query failed")
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	assert.Nil(t, metrics)
	require.Error(t, err)
}

// Error branch: recent activity query fails → service returns error.
func TestGetDashboardMetrics_RecentActivityError(t *testing.T) {
	repo := &mockRepo{
		getRecentFn: func(_ context.Context) ([]*RecentActivity, error) {
			return nil, errors.New("recent activity query failed")
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	assert.Nil(t, metrics)
	require.Error(t, err)
}

// Error branch: track score query fails → service returns error.
func TestGetDashboardMetrics_TrackScoreError(t *testing.T) {
	repo := &mockRepo{
		getAvgScoreByTrack: func(_ context.Context) (map[string]*float64, error) {
			return nil, errors.New("track score query failed")
		},
	}
	svc := NewService(repo)

	metrics, err := svc.GetDashboardMetrics(context.Background())
	assert.Nil(t, metrics)
	require.Error(t, err)
}

// ── FR-BB52: Per-Exam Analytics ──────────────────────────────────────────────

// AC-2: fillBuckets always produces exactly 10 ordered buckets even when input is sparse.
func TestFillBuckets_AlwaysTenBuckets(t *testing.T) {
	raw := []BucketCount{
		{Bucket: "70-80", Count: 5},
		{Bucket: "90-100", Count: 3},
	}
	result := fillBuckets(raw)
	require.Len(t, result, 10)
	assert.Equal(t, "0-10", result[0].Bucket)
	assert.Equal(t, 0, result[0].Count)
	assert.Equal(t, "70-80", result[7].Bucket)
	assert.Equal(t, 5, result[7].Count)
	assert.Equal(t, "90-100", result[9].Bucket)
	assert.Equal(t, 3, result[9].Count)
}

// AC-2: fillBuckets with empty input returns 10 zero-count buckets.
func TestFillBuckets_EmptyInput(t *testing.T) {
	result := fillBuckets([]BucketCount{})
	require.Len(t, result, 10)
	for _, b := range result {
		assert.Equal(t, 0, b.Count)
	}
}

// AC-3: pass_rate nil → 0.0 coercion.
func TestGetExamAnalytics_PassRateNilCoercion(t *testing.T) {
	repo := &mockRepo{
		getSummaryStatsFn: func(_ context.Context, _ string) (*examSummaryRow, error) {
			return &examSummaryRow{PassRate: nil}, nil // nil → 0.0
		},
	}
	svc := NewService(repo)

	result, err := svc.GetExamAnalytics(context.Background(), "exam-uuid")
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.PassRate)
}

// AC-3: pass_rate non-nil → preserved.
func TestGetExamAnalytics_PassRatePreserved(t *testing.T) {
	pr := 0.72
	repo := &mockRepo{
		getSummaryStatsFn: func(_ context.Context, _ string) (*examSummaryRow, error) {
			return &examSummaryRow{PassRate: &pr}, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.GetExamAnalytics(context.Background(), "exam-uuid")
	require.NoError(t, err)
	assert.Equal(t, 0.72, result.PassRate)
}

// AC-1: ErrNotFound returned when exam title query fails with ErrNotFound.
func TestGetExamAnalytics_ExamNotFound(t *testing.T) {
	repo := &mockRepo{
		getExamTitleFn: func(_ context.Context, _ string) (string, error) {
			return "", ErrNotFound
		},
	}
	svc := NewService(repo)

	result, err := svc.GetExamAnalytics(context.Background(), "nonexistent")
	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

// AC-2: score_distribution has exactly 10 buckets even with no sessions.
func TestGetExamAnalytics_ScoreDistributionAlwaysTenBuckets(t *testing.T) {
	repo := &mockRepo{} // all defaults → empty buckets
	svc := NewService(repo)

	result, err := svc.GetExamAnalytics(context.Background(), "exam-uuid")
	require.NoError(t, err)
	assert.Len(t, result.ScoreDistribution, 10)
	assert.Equal(t, "0-10", result.ScoreDistribution[0].Bucket)
	assert.Equal(t, "90-100", result.ScoreDistribution[9].Bucket)
}

// AC-8: answer_distribution includes options even with zero selections.
func TestGetExamAnalytics_AnswerDistributionGroupedByQuestion(t *testing.T) {
	qID := "q-1"
	repo := &mockRepo{
		getPerQuestionStatsFn: func(_ context.Context, _ string) ([]questionStatRow, error) {
			return []questionStatRow{{QuestionID: qID, StemPreview: "What?"}}, nil
		},
		getAnswerDistFn: func(_ context.Context, _ string) ([]answerDistRow, error) {
			return []answerDistRow{
				{QuestionID: qID, OptionID: "opt-a", OptionText: "Yes", SelectCount: 5},
				{QuestionID: qID, OptionID: "opt-b", OptionText: "No", SelectCount: 0},
			}, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.GetExamAnalytics(context.Background(), "exam-uuid")
	require.NoError(t, err)
	require.Len(t, result.PerQuestionStats, 1)
	qs := result.PerQuestionStats[0]
	assert.Equal(t, qID, qs.QuestionID)
	require.Len(t, qs.AnswerDistribution, 2)
	assert.Equal(t, 5, qs.AnswerDistribution[0].SelectCount)
	assert.Equal(t, 0, qs.AnswerDistribution[1].SelectCount)
}

// ── FR-BB53: buildTrackProgress unit tests ───────────────────────────────────

// AC-6: always returns exactly three tracks in the fixed order.
func TestBuildTrackProgress_AlwaysThreeTracks(t *testing.T) {
	result := buildTrackProgress(nil, nil)
	require.Len(t, result, 3)
	assert.Equal(t, "security", result[0].Track)
	assert.Equal(t, "safety", result[1].Track)
	assert.Equal(t, "loyalty", result[2].Track)
}

// AC-7: questions_answered is zero for tracks with no activity.
func TestBuildTrackProgress_ZeroQuestionsForMissingTracks(t *testing.T) {
	activity := []TrackActivity{
		{Track: "security", QuestionsAnswered: 42},
	}
	result := buildTrackProgress(activity, nil)
	assert.Equal(t, 42, result[0].QuestionsAnswered)
	assert.Equal(t, 0, result[1].QuestionsAnswered) // safety
	assert.Equal(t, 0, result[2].QuestionsAnswered) // loyalty
}

// AC-8: last_activity is nil for tracks with no submitted sessions.
func TestBuildTrackProgress_NilLastActivityForMissingTracks(t *testing.T) {
	ts := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	activity := []TrackActivity{
		{Track: "loyalty", QuestionsAnswered: 10, LastActivity: &ts},
	}
	result := buildTrackProgress(activity, nil)
	assert.Nil(t, result[0].LastActivity) // security
	assert.Nil(t, result[1].LastActivity) // safety
	require.NotNil(t, result[2].LastActivity)
	assert.Equal(t, ts, *result[2].LastActivity)
}

// AC-9: required_exams is empty (non-nil) for tracks with no assigned exams.
func TestBuildTrackProgress_RequiredExamsEmptySliceNotNil(t *testing.T) {
	result := buildTrackProgress(nil, nil)
	for _, ts := range result {
		assert.NotNil(t, ts.RequiredExams, "RequiredExams must be non-nil for track %s", ts.Track)
		assert.Empty(t, ts.RequiredExams)
	}
}

// AC-9: required_exams are correctly routed to the matching track.
func TestBuildTrackProgress_ExamsRoutedToCorrectTrack(t *testing.T) {
	passed := true
	exams := []ExamProgress{
		{ExamID: "e1", Title: "Security 101", Track: "security", Passed: &passed, Attempts: 1},
		{ExamID: "e2", Title: "Fire Safety", Track: "safety", Passed: nil, Attempts: 0},
	}
	result := buildTrackProgress(nil, exams)

	require.Len(t, result[0].RequiredExams, 1) // security
	assert.Equal(t, "e1", result[0].RequiredExams[0].ExamID)

	require.Len(t, result[1].RequiredExams, 1) // safety
	assert.Equal(t, "e2", result[1].RequiredExams[0].ExamID)

	assert.Empty(t, result[2].RequiredExams) // loyalty
}

// ── FR-BB53: GetUserRecord service tests ─────────────────────────────────────

// AC-2: ErrNotFound propagated when user does not exist.
func TestGetUserRecord_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		getUserInfoFn: func(_ context.Context, _ string) (*userInfoRow, error) {
			return nil, ErrNotFound
		},
	}
	svc := NewService(repo)

	_, _, err := svc.GetUserRecord(context.Background(), "missing-id", 1, 20)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

// AC-3/AC-5: happy path returns correct record and total count.
func TestGetUserRecord_HappyPath(t *testing.T) {
	dept := "Operations"
	submittedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	passed := true
	timeSec := 1800
	certID := "cert-uuid"
	repo := &mockRepo{
		getUserInfoFn: func(_ context.Context, userID string) (*userInfoRow, error) {
			return &userInfoRow{ID: userID, FullName: "Aibek Seitkali", DepartmentName: &dept}, nil
		},
		getUserSessionCountFn: func(_ context.Context, _ string) (int, error) {
			return 7, nil
		},
		getUserSessionHistoryFn: func(_ context.Context, _ string, limit, offset int) ([]SessionRecord, error) {
			assert.Equal(t, 20, limit)
			assert.Equal(t, 0, offset)
			return []SessionRecord{
				{
					SessionID:        "sess-uuid",
					ExamID:           "exam-uuid",
					ExamTitle:        "Fire Safety Fundamentals",
					StartedAt:        time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
					SubmittedAt:      &submittedAt,
					ScorePct:         ptr(84.50),
					Passed:           &passed,
					TimeTakenSeconds: &timeSec,
					Status:           "submitted",
					CertificateID:    &certID,
				},
			}, nil
		},
	}
	svc := NewService(repo)

	record, total, err := svc.GetUserRecord(context.Background(), "user-uuid", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 7, total)
	assert.Equal(t, "user-uuid", record.UserID)
	assert.Equal(t, "Aibek Seitkali", record.FullName)
	assert.Equal(t, "Operations", record.Department)
	require.Len(t, record.Sessions, 1)
	assert.Equal(t, "sess-uuid", record.Sessions[0].SessionID)
	assert.Equal(t, &certID, record.Sessions[0].CertificateID)
}

// AC-5: page 2 passes correct offset to repository.
func TestGetUserRecord_PaginationOffset(t *testing.T) {
	var capturedOffset int
	repo := &mockRepo{
		getUserSessionHistoryFn: func(_ context.Context, _ string, limit, offset int) ([]SessionRecord, error) {
			capturedOffset = offset
			return []SessionRecord{}, nil
		},
	}
	svc := NewService(repo)

	_, _, err := svc.GetUserRecord(context.Background(), "user-uuid", 3, 10)
	require.NoError(t, err)
	assert.Equal(t, 20, capturedOffset) // (3-1)*10
}

// ── FR-BB53: GetUserProgress service tests ───────────────────────────────────

// AC-2: ErrNotFound propagated when user does not exist.
func TestGetUserProgress_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		getUserInfoFn: func(_ context.Context, _ string) (*userInfoRow, error) {
			return nil, ErrNotFound
		},
	}
	svc := NewService(repo)

	_, err := svc.GetUserProgress(context.Background(), "missing-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

// AC-6: response always contains exactly three tracks.
func TestGetUserProgress_AlwaysThreeTracks(t *testing.T) {
	repo := &mockRepo{} // all defaults → empty activity and exams
	svc := NewService(repo)

	result, err := svc.GetUserProgress(context.Background(), "user-uuid")
	require.NoError(t, err)
	require.Len(t, result.Tracks, 3)
}

// AC-7/AC-8/AC-9: full happy path with activity and exams.
func TestGetUserProgress_HappyPath(t *testing.T) {
	ts := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	passed := true
	dept := "IT"
	repo := &mockRepo{
		getUserInfoFn: func(_ context.Context, userID string) (*userInfoRow, error) {
			return &userInfoRow{ID: userID, FullName: "Aibek Seitkali", DepartmentName: &dept}, nil
		},
		getUserTrackActivityFn: func(_ context.Context, _ string) ([]TrackActivity, error) {
			return []TrackActivity{
				{Track: "security", QuestionsAnswered: 120, LastActivity: &ts},
			}, nil
		},
		getUserRequiredExamsFn: func(_ context.Context, _ string) ([]ExamProgress, error) {
			return []ExamProgress{
				{ExamID: "e1", Title: "Security Awareness", Track: "security", Passed: &passed, Attempts: 2},
			}, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.GetUserProgress(context.Background(), "user-uuid")
	require.NoError(t, err)
	assert.Equal(t, "user-uuid", result.UserID)
	assert.Equal(t, "Aibek Seitkali", result.FullName)
	require.Len(t, result.Tracks, 3)

	sec := result.Tracks[0]
	assert.Equal(t, "security", sec.Track)
	assert.Equal(t, 120, sec.QuestionsAnswered)
	require.NotNil(t, sec.LastActivity)
	require.Len(t, sec.RequiredExams, 1)
	assert.Equal(t, "e1", sec.RequiredExams[0].ExamID)

	safety := result.Tracks[1]
	assert.Equal(t, 0, safety.QuestionsAnswered)
	assert.Nil(t, safety.LastActivity)
	assert.Empty(t, safety.RequiredExams)
}
