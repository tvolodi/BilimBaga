package portal

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (task #386) for the portal repository reads: assigned exams
// (list and single), and a user's sessions for an exam, including the
// pre-migration fallback when exam_sessions does not exist yet.

var portalExamCols = []string{
	"id", "title", "description", "time_limit_minutes", "passing_score_pct", "max_attempts",
	"shuffle_questions", "shuffle_options", "show_answers", "certificate_enabled",
	"available_from", "available_until", "deadline",
}

func portalExamRowFor(id, title string, deadline time.Time) []driver.Value {
	return []driver.Value{id, title, nil, int64(30), float64(70), int64(2),
		true, false, "after_submit", false, nil, nil, deadline}
}

func TestPortalListAssignedExams_RowsEmptyAndError(t *testing.T) {
	deadline := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue(portalExamCols, [][]driver.Value{portalExamRowFor("e-1", "Fire Safety", deadline)})
	exams, err := NewRepository(db).ListAssignedExams(context.Background(), "user-1", "dept-1")
	require.NoError(t, err)
	require.Len(t, exams, 1)
	assert.Equal(t, "Fire Safety", exams[0].Title)
	require.NotNil(t, exams[0].Deadline)

	db2, f2 := newFakeDB(t)
	f2.queue(portalExamCols, nil)
	empty, err := NewRepository(db2).ListAssignedExams(context.Background(), "user-1", "")
	require.NoError(t, err)
	assert.NotNil(t, empty, "no assignments is an empty, non-nil list")
	assert.Empty(t, empty)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).ListAssignedExams(context.Background(), "user-1", "dept-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "ListAssignedExams")
}

func TestPortalGetAssignedExam_FoundNotAssignedAndError(t *testing.T) {
	deadline := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue(portalExamCols, [][]driver.Value{portalExamRowFor("e-1", "Fire Safety", deadline)})
	row, err := NewRepository(db).GetAssignedExam(context.Background(), "e-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.Equal(t, "e-1", row.ID)

	db2, f2 := newFakeDB(t)
	f2.queue(portalExamCols, nil)
	_, err = NewRepository(db2).GetAssignedExam(context.Background(), "e-9", "user-1", "dept-1")
	assert.ErrorIs(t, err, ErrNotAssigned)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetAssignedExam(context.Background(), "e-1", "user-1", "dept-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetAssignedExam")
}

var sessionListCols = []string{"session_id", "status", "expires_at", "passed", "submitted_at", "score_pct", "started_at"}

func TestPortalListUserSessions_RowsMissingTableAndError(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue(sessionListCols, [][]driver.Value{{"s-1", "submitted", nil, true, at, float64(80), at}})
	sessions, err := NewRepository(db).ListUserSessions(context.Background(), "e-1", "user-1")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "s-1", sessions[0].SessionID)
	assert.True(t, sessions[0].Passed)

	// Before migration FR-BB35 the table does not exist yet: an empty list, not an error.
	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New(`pq: relation "exam_sessions" does not exist`)
	none, err := NewRepository(db2).ListUserSessions(context.Background(), "e-1", "user-1")
	require.NoError(t, err)
	assert.NotNil(t, none)
	assert.Empty(t, none)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).ListUserSessions(context.Background(), "e-1", "user-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "ListUserSessions")
}
