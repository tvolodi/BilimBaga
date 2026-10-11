package sessions

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireCountsGradingPending asserts that every attempt-status filter on exam_sessions names grading_pending
// (FR-BB322 AC-1, AC-13).
func requireCountsGradingPending(t *testing.T, f *fakeDB) {
	t.Helper()
	require.NotEmpty(t, f.queries)
	checked := 0
	for _, q := range f.queries {
		if strings.Contains(q, "exam_sessions") && strings.Contains(q, "status IN") {
			checked++
			assert.Contains(t, q, "'grading_pending'", "attempt filter must count a waiting session: %s", q)
		}
	}
	require.Positive(t, checked, "no attempt filter was run")
}

// FR-BB322 AC-1: the start-attempt count and the history queries count a waiting session as an attempt.
func TestAttemptCountingQueries_CountGradingPending(t *testing.T) {
	ctx := context.Background()

	t.Run("CountFinishedSessions", func(t *testing.T) {
		db, f := newFakeDB(t)
		f.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
		_, err := NewRepository(db, nil).CountFinishedSessions(ctx, "exam-1", "user-1")
		require.NoError(t, err)
		requireCountsGradingPending(t, f)
	})

	t.Run("GetExamHistory count and rows", func(t *testing.T) {
		db, f := newFakeDB(t)
		f.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
		f.queue([]string{"session_id"}, nil)
		_, _, err := NewRepository(db, nil).GetExamHistory(ctx, "exam-1", "user-1", 1, 20)
		require.NoError(t, err)
		requireCountsGradingPending(t, f)
		// AC-13: the history row of a waiting session carries no stored score or pass.
		assert.Equal(t, 1, countQueries(f, "CASE WHEN status = 'grading_pending' THEN NULL ELSE score_pct END"), "the rows query masks the score")
	})

	t.Run("GetMyResults count and rows", func(t *testing.T) {
		db, f := newFakeDB(t)
		f.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
		f.queue([]string{"session_id"}, nil)
		_, _, err := NewRepository(db, nil).GetMyResults(ctx, "user-1", 1, 20, "date", "desc")
		require.NoError(t, err)
		requireCountsGradingPending(t, f)
		// AC-13: no COALESCE(score_pct, 0) for a waiting row, and no certificate for it.
		assert.Equal(t, 1, countQueries(f, "CASE WHEN es.status = 'grading_pending' THEN NULL ELSE es.score_pct END"))
		assert.Equal(t, 1, countQueries(f, "es.status <> 'grading_pending' AND COALESCE(es.passed, false) AND e.certificate_enabled"))
	})
}

// FR-BB322 AC-9: a replayed auto-submit is not a transition. It changes nothing, grades nothing, and reports
// Transitioned false with the stored state.
func TestAutoSubmitSession_ReplayChangesNothing(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{false}})
	f.queue(autoUpdateCols, nil)                                                 // the guarded UPDATE matches no in_progress row
	f.queue(autoUpdateCols, [][]driver.Value{{"sess-1", "submitted", nil, nil}}) // the replay read

	res, err := NewRepository(db, txStubEngine{}).AutoSubmitSession(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.False(t, res.Transitioned)
	assert.Equal(t, "submitted", res.Status)
	assert.Equal(t, 1, countQueries(f, "AND status = 'in_progress'"), "the update is guarded by status")
}

// FR-BB322 AC-9: the call that moves a session out of in_progress is the transition.
func TestAutoSubmitSession_FirstCallTransitions(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{true}})
	f.queue(autoUpdateCols, [][]driver.Value{{"sess-1", "grading_pending", nil, nil}})

	res, err := NewRepository(db, txStubEngine{}).AutoSubmitSession(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.True(t, res.Transitioned)
	assert.Equal(t, "grading_pending", res.Status)
}
