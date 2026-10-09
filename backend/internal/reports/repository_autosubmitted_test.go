package reports

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-195 item 1: every "completed session" predicate in the dashboard,
// user-progress and completion-rate queries must count auto_submitted sessions,
// consistent with the results CSV / analytics queries (PR #194).
var (
	completedInList = regexp.MustCompile(`status IN \('submitted',\s*'auto_submitted',\s*'grading_pending'\)`)
	legacyCompleted = regexp.MustCompile(`status IN \('submitted',\s*'grading_pending'\)|status = 'submitted'`)
)

// Split rule (ISS-195): completion counts include grading_pending; score
// aggregates (averages, correct rates) exclude it because its scores are partial.
var scoreAggregateIn = regexp.MustCompile(`status IN \('submitted',\s*'auto_submitted'\)`)

func assertScoreAggregateExcludesGradingPending(t *testing.T, q string) {
	t.Helper()
	assert.Len(t, scoreAggregateIn.FindAllString(q, -1), 1)
	assert.NotContains(t, q, "grading_pending")
	assert.NotContains(t, q, "tenant_id")
	assert.NotContains(t, q, "@SCOPE@")
}

func assertCountsAutoSubmitted(t *testing.T, q string, wantOccurrences int) {
	t.Helper()
	assert.Len(t, completedInList.FindAllString(q, -1), wantOccurrences)
	assert.False(t, legacyCompleted.MatchString(q), "query still uses a completed-status list without auto_submitted")
	assert.NotContains(t, q, "'graded'", "there is no 'graded' session status")
	assert.NotContains(t, q, "tenant_id", "exams/sessions have no tenant column")
	assert.NotContains(t, q, "@SCOPE@", "scope placeholder must be expanded")
}

func TestAutoSubmittedCountedAsCompleted_RepositorySQL(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	empty := func(f *fakeDB) { f.queue([]string{"x"}, [][]driver.Value(nil)) }

	cases := []struct {
		name     string
		scoreAgg bool
		run      func(r Repository)
	}{
		{"GetCompletionRateByExam", false, func(r Repository) { _, _ = r.GetCompletionRateByExam(ctx) }},
		{"GetRecentActivity", false, func(r Repository) { _, _ = r.GetRecentActivity(ctx) }},
		{"GetAvgScoreByTrack", true, func(r Repository) { _, _ = r.GetAvgScoreByTrack(ctx) }},
		{"GetUserTrackActivity", false, func(r Repository) { _, _ = r.GetUserTrackActivity(ctx, "u1") }},
		{"GetUserRequiredExams", false, func(r Repository) { _, _ = r.GetUserRequiredExams(ctx, "u1") }},
		{"GetDashboardCompletionRatesForRange", false, func(r Repository) {
			_, _ = r.GetDashboardCompletionRatesForRange(ctx, "t", from, to)
		}},
		{"GetTopBottomQuestions", true, func(r Repository) { _, _, _ = r.GetTopBottomQuestions(ctx, "t", from, to) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, f := newFakeDB(t)
			empty(f)
			empty(f)
			tc.run(NewRepository(db))
			require.NotEmpty(t, f.queries)
			if tc.scoreAgg {
				assertScoreAggregateExcludesGradingPending(t, f.queries[0])
				return
			}
			assertCountsAutoSubmitted(t, f.queries[0], 1)
		})
	}
}

// Service/handler layer: completed_count produced by the repository (which now
// includes auto_submitted sessions) is surfaced unchanged on the dashboard.
func TestGetDashboardMetrics_AutoSubmittedCountedInCompletion(t *testing.T) {
	repo := &mockRepo{}
	repo.getCompletionFn = func(context.Context) ([]*ExamCompletionRate, error) {
		// 4 assigned: 2 submitted + 1 auto_submitted + 1 grading_pending => 4 completed.
		return []*ExamCompletionRate{{ExamID: "e1", Title: "E", AssignedCount: 4, CompletedCount: 4, PassedCount: 3}}, nil
	}
	m, err := NewService(repo).GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, m.CompletionRateByExam, 1)
	assert.Equal(t, 4, m.CompletionRateByExam[0].CompletedCount)
}
