package sessions

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (follow-up #306) for the query-only session paths that were
// at 0%: adaptive state, session-result reads and section scores. No transaction
// is involved, so the in-process fake driver is enough.

// ── adaptive state ───────────────────────────────────────────────────────────

func TestGetAdaptiveState_NullRawReturnsDefault(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive_state"}, [][]driver.Value{{nil}})
	st, err := NewRepository(db, nil).GetAdaptiveState(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "medium", st.CurrentDifficulty)
	assert.NotNil(t, st.ServedQuestionIDs)
	assert.NotNil(t, st.RecentResults)
	assert.Nil(t, st.CurrentQuestionID)
}

func TestGetAdaptiveState_ParsesStoredJSON(t *testing.T) {
	db, f := newFakeDB(t)
	raw := []byte(`{"current_difficulty":"hard","served_question_ids":["q1"],"recent_results":[true,false],"current_question_id":"q2"}`)
	f.queue([]string{"adaptive_state"}, [][]driver.Value{{raw}})
	st, err := NewRepository(db, nil).GetAdaptiveState(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "hard", st.CurrentDifficulty)
	assert.Equal(t, []string{"q1"}, st.ServedQuestionIDs)
	assert.Equal(t, []bool{true, false}, st.RecentResults)
	require.NotNil(t, st.CurrentQuestionID)
	assert.Equal(t, "q2", *st.CurrentQuestionID)
}

func TestGetAdaptiveState_JSONNullReturnsDefaultWithEmptySlices(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive_state"}, [][]driver.Value{{[]byte("null")}})
	st, err := NewRepository(db, nil).GetAdaptiveState(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "medium", st.CurrentDifficulty)
	assert.NotNil(t, st.ServedQuestionIDs)
}

func TestGetAdaptiveState_StoredNullSlicesBecomeEmpty(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive_state"}, [][]driver.Value{{[]byte(`{"current_difficulty":"easy"}`)}})
	st, err := NewRepository(db, nil).GetAdaptiveState(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "easy", st.CurrentDifficulty)
	assert.NotNil(t, st.ServedQuestionIDs)
	assert.NotNil(t, st.RecentResults)
}

func TestGetAdaptiveState_NotFound(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive_state"}, nil)
	_, err := NewRepository(db, nil).GetAdaptiveState(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestGetAdaptiveState_BadJSONAndQueryError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive_state"}, [][]driver.Value{{[]byte("{")}})
	_, err := NewRepository(db, nil).GetAdaptiveState(context.Background(), "sess-1")
	assert.ErrorContains(t, err, "unmarshal")

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetAdaptiveState(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetAdaptiveState")
}

func TestUpdateAdaptiveState_OKAndError(t *testing.T) {
	db, f := newFakeDB(t)
	st := &AdaptiveState{CurrentDifficulty: "hard", ServedQuestionIDs: []string{"q1"}, RecentResults: []bool{true}}
	require.NoError(t, NewRepository(db, nil).UpdateAdaptiveState(context.Background(), "sess-1", st))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "UPDATE exam_sessions SET adaptive_state")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2, nil).UpdateAdaptiveState(context.Background(), "sess-1", st)
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "UpdateAdaptiveState")
}

func TestInitAdaptiveState_DelegatesToUpdate(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db, nil).InitAdaptiveState(context.Background(), "sess-1", &AdaptiveState{CurrentDifficulty: "medium"}))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "UPDATE exam_sessions SET adaptive_state")
}

// ── GetSessionAdaptive: two queries (adaptive flag, then owner) ─────────────

func TestGetSessionAdaptive_OwnerSeesFlag(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive"}, [][]driver.Value{{true}})
	f.queue([]string{"user_id"}, [][]driver.Value{{"user-1"}})
	ok, err := NewRepository(db, nil).GetSessionAdaptive(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestGetSessionAdaptive_OtherUserForbidden(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive"}, [][]driver.Value{{true}})
	f.queue([]string{"user_id"}, [][]driver.Value{{"user-1"}})
	_, err := NewRepository(db, nil).GetSessionAdaptive(context.Background(), "sess-1", "user-2")
	assert.ErrorIs(t, err, ErrSessionForbidden)
}

func TestGetSessionAdaptive_NotFound(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"adaptive"}, nil)
	_, err := NewRepository(db, nil).GetSessionAdaptive(context.Background(), "missing", "user-1")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestGetSessionAdaptive_QueryErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, err := NewRepository(db, nil).GetSessionAdaptive(context.Background(), "sess-1", "user-1")
	assert.ErrorIs(t, err, f.qErr)
	assert.ErrorContains(t, err, "GetSessionAdaptive")
}

// ── session results ──────────────────────────────────────────────────────────

var sessionResultCols = []string{
	"session_id", "exam_id", "exam_title", "show_answers_mode", "user_id", "status",
	"score_pct", "passed", "time_taken_seconds", "submitted_at", "attempt_number",
}

func sessionResultRowFor(owner string) []driver.Value {
	return []driver.Value{
		"sess-1", "exam-1", "Fire Safety", "after_submit", owner, "submitted",
		float64(80), true, int64(600), time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), int64(1),
	}
}

func TestGetSessionResult_OwnerGetsRow(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(sessionResultCols, [][]driver.Value{sessionResultRowFor("user-1")})
	row, err := NewRepository(db, nil).GetSessionResult(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "Fire Safety", row.ExamTitle)
	assert.True(t, row.Passed)
	require.NotNil(t, row.ScorePct)
	assert.Equal(t, 80.0, *row.ScorePct)
	assert.Equal(t, 1, row.AttemptNumber)
}

func TestGetSessionResult_OtherUserForbidden(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(sessionResultCols, [][]driver.Value{sessionResultRowFor("user-1")})
	_, err := NewRepository(db, nil).GetSessionResult(context.Background(), "sess-1", "user-2")
	assert.ErrorIs(t, err, ErrSessionForbidden)
}

func TestGetSessionResult_NotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(sessionResultCols, nil)
	_, err := NewRepository(db, nil).GetSessionResult(context.Background(), "missing", "user-1")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetSessionResult(context.Background(), "sess-1", "user-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetSessionResult")
}

func TestGetAdminSessionResult_AnyOwnerAndNotFound(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(sessionResultCols, [][]driver.Value{sessionResultRowFor("user-9")})
	row, err := NewRepository(db, nil).GetAdminSessionResult(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "user-9", row.UserID)

	db2, f2 := newFakeDB(t)
	f2.queue(sessionResultCols, nil)
	_, err = NewRepository(db2, nil).GetAdminSessionResult(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// ── section scores ───────────────────────────────────────────────────────────

func TestGetSectionScores_RowsAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"section_id", "title", "score_pct"}, [][]driver.Value{{"sec-1", "Safety", float64(75.5)}})
	scores, err := NewRepository(db, nil).GetSectionScores(context.Background(), "sess-1")
	require.NoError(t, err)
	require.Len(t, scores, 1)
	assert.Equal(t, SectionScore{SectionID: "sec-1", Title: "Safety", ScorePct: 75.5}, scores[0])

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetSectionScores(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetSectionScores")
}
