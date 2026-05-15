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
	getCompletionFn       func(ctx context.Context) ([]*ExamCompletionRate, error)
	getOverdueFn          func(ctx context.Context) ([]*OverdueEmployee, error)
	getRecentFn           func(ctx context.Context) ([]*RecentActivity, error)
	getAvgScoreByTrack    func(ctx context.Context) (map[string]*float64, error)
	getExamTitleFn        func(ctx context.Context, examID string) (string, error)
	getScoreDistFn        func(ctx context.Context, examID string) ([]BucketCount, error)
	getSummaryStatsFn     func(ctx context.Context, examID string) (*examSummaryRow, error)
	getPerQuestionStatsFn func(ctx context.Context, examID string) ([]questionStatRow, error)
	getAnswerDistFn       func(ctx context.Context, examID string) ([]answerDistRow, error)
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
