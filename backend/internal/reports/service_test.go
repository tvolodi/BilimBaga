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
	getCompletionFn    func(ctx context.Context) ([]*ExamCompletionRate, error)
	getOverdueFn       func(ctx context.Context) ([]*OverdueEmployee, error)
	getRecentFn        func(ctx context.Context) ([]*RecentActivity, error)
	getAvgScoreByTrack func(ctx context.Context) (map[string]*float64, error)
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
