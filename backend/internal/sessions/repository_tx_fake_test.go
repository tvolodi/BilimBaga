package sessions

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (follow-up #311) for the transactional sessions paths:
// CreateSession, SubmitSession, AutoSubmitSession and GradeAnswer. The fake
// driver (fakedb_test.go) now begins no-op transactions. Queries inside a
// transaction consume queued results in the order the code runs them; Exec
// calls consume nothing. The grading engine is replaced by txStubEngine.

// txStubEngine stands in for the GradingEngine, so SubmitSession and
// AutoSubmitSession do not need the grading queries.
type txStubEngine struct {
	score  float64
	passed bool
	err    error
}

func (e txStubEngine) Grade(_ *sqlx.Tx, _ string) (float64, bool, error) {
	return e.score, e.passed, e.err
}

func countQueries(f *fakeDB, substr string) int {
	n := 0
	for _, q := range f.queries {
		if strings.Contains(q, substr) {
			n++
		}
	}
	return n
}

// ── CreateSession ────────────────────────────────────────────────────────────

func TestCreateSession_InsertsSessionAndQuestions(t *testing.T) {
	db, f := newFakeDB(t)
	start := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	exp := start.Add(time.Hour)
	f.queue([]string{"id", "started_at", "expires_at"}, [][]driver.Value{{"sess-1", start, exp}})
	id, gotStart, gotExp, err := NewRepository(db, nil).CreateSession(context.Background(), createSessionInput{
		ExamID: "exam-1", UserID: "user-1", Seed: 7, ExpiresAt: exp,
		Questions: []resolvedQuestion{
			{QuestionID: "q1", SortOrder: 1, OptionsOrder: []string{"o1", "o2"}},
			{QuestionID: "q2", SortOrder: 2},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "sess-1", id)
	assert.True(t, gotStart.Equal(start))
	assert.True(t, gotExp.Equal(exp))
	assert.Equal(t, 2, countQueries(f, "INSERT INTO session_questions"))
}

func TestCreateSession_InsertSessionErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, _, _, err := NewRepository(db, nil).CreateSession(context.Background(), createSessionInput{ExamID: "exam-1"})
	assert.ErrorIs(t, err, f.qErr)
	assert.ErrorContains(t, err, "insert session")
}

func TestCreateSession_InsertQuestionErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id", "started_at", "expires_at"}, [][]driver.Value{{"sess-1", time.Now(), time.Now()}})
	f.xErr = errors.New("boom")
	_, _, _, err := NewRepository(db, nil).CreateSession(context.Background(), createSessionInput{
		ExamID: "exam-1", Questions: []resolvedQuestion{{QuestionID: "q1", SortOrder: 1}},
	})
	assert.ErrorIs(t, err, f.xErr)
	assert.ErrorContains(t, err, "insert question")
}

// ── SubmitSession ────────────────────────────────────────────────────────────

var submitFetchCols = []string{"id", "user_id", "status", "submitted_at", "score_pct", "passed"}
var submitUpdateCols = []string{"id", "status", "submitted_at", "score_pct", "passed"}

func queueSubmit(f *fakeDB, owner, status string, hasShortText bool, updatedStatus string) {
	f.queue(submitFetchCols, [][]driver.Value{{"sess-1", owner, status, nil, nil, nil}})
	f.queue([]string{"exists"}, [][]driver.Value{{hasShortText}})
	f.queue(submitUpdateCols, [][]driver.Value{{"sess-1", updatedStatus, time.Now(), nil, nil}})
}

func TestSubmitSession_SubmitsAndGrades(t *testing.T) {
	db, f := newFakeDB(t)
	queueSubmit(f, "user-1", "in_progress", false, "submitted")
	repo := NewRepository(db, txStubEngine{score: 80, passed: true})
	res, err := repo.SubmitSession(context.Background(), "sess-1", "user-1", "tenant-1", "10.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "submitted", res.Status)
	require.NotNil(t, res.ScorePct)
	assert.Equal(t, 80.0, *res.ScorePct)
	require.NotNil(t, res.Passed)
	assert.True(t, *res.Passed)
	assert.Equal(t, 1, countQueries(f, "audit_log"))
}

func TestSubmitSession_ShortTextGoesPendingAndAuditsTwice(t *testing.T) {
	db, f := newFakeDB(t)
	queueSubmit(f, "user-1", "in_progress", true, "grading_pending")
	repo := NewRepository(db, txStubEngine{score: 0, passed: false})
	res, err := repo.SubmitSession(context.Background(), "sess-1", "user-1", "tenant-1", "")
	require.NoError(t, err)
	assert.Equal(t, "grading_pending", res.Status)
	assert.Equal(t, 2, countQueries(f, "audit_log"))
}

func TestSubmitSession_AlreadySubmittedIsIdempotent(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(submitFetchCols, [][]driver.Value{{"sess-1", "user-1", "submitted", nil, nil, nil}})
	res, err := NewRepository(db, txStubEngine{}).SubmitSession(context.Background(), "sess-1", "user-1", "tenant-1", "")
	require.NoError(t, err)
	assert.Equal(t, "submitted", res.Status)
	assert.Len(t, f.queries, 1, "no further statements once already submitted")
}

func TestSubmitSession_ForbiddenNotFoundAndFetchError(t *testing.T) {
	db, f := newFakeDB(t)
	queueSubmit(f, "user-1", "in_progress", false, "submitted")
	_, err := NewRepository(db, txStubEngine{}).SubmitSession(context.Background(), "sess-1", "user-2", "tenant-1", "")
	assert.ErrorIs(t, err, ErrSessionForbidden)

	db2, f2 := newFakeDB(t)
	f2.queue(submitFetchCols, nil)
	_, err = NewRepository(db2, txStubEngine{}).SubmitSession(context.Background(), "missing", "user-1", "tenant-1", "")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3, txStubEngine{}).SubmitSession(context.Background(), "sess-1", "user-1", "tenant-1", "")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "fetch")
}

func TestSubmitSession_AuditAndGradeErrorsWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	queueSubmit(f, "user-1", "in_progress", false, "submitted")
	f.xErr = errors.New("boom")
	_, err := NewRepository(db, txStubEngine{}).SubmitSession(context.Background(), "sess-1", "user-1", "tenant-1", "")
	assert.ErrorIs(t, err, f.xErr)
	assert.ErrorContains(t, err, "audit session.submit")

	db2, f2 := newFakeDB(t)
	queueSubmit(f2, "user-1", "in_progress", false, "submitted")
	gradeErr := errors.New("grade failed")
	_, err = NewRepository(db2, txStubEngine{err: gradeErr}).SubmitSession(context.Background(), "sess-1", "user-1", "tenant-1", "")
	assert.ErrorIs(t, err, gradeErr)
	assert.ErrorContains(t, err, "grade")
}

// ── AutoSubmitSession ────────────────────────────────────────────────────────

var autoUpdateCols = []string{"id", "status", "score_pct", "passed"}

func TestAutoSubmitSession_SubmitsAndGrades(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{false}})
	f.queue(autoUpdateCols, [][]driver.Value{{"sess-1", "auto_submitted", nil, nil}})
	res, err := NewRepository(db, txStubEngine{score: 70, passed: true}).AutoSubmitSession(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "auto_submitted", res.Status)
	require.NotNil(t, res.ScorePct)
	assert.Equal(t, 70.0, *res.ScorePct)
	require.NotNil(t, res.Passed)
	assert.True(t, *res.Passed)
}

func TestAutoSubmitSession_ShortTextGoesPending(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{true}})
	f.queue(autoUpdateCols, [][]driver.Value{{"sess-1", "grading_pending", nil, nil}})
	res, err := NewRepository(db, txStubEngine{}).AutoSubmitSession(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "grading_pending", res.Status)
}

func TestAutoSubmitSession_ErrorsWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, err := NewRepository(db, txStubEngine{}).AutoSubmitSession(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f.qErr)
	assert.ErrorContains(t, err, "check shorttext")

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"exists"}, [][]driver.Value{{false}})
	f2.queue(autoUpdateCols, [][]driver.Value{{"sess-1", "auto_submitted", nil, nil}})
	gradeErr := errors.New("grade failed")
	_, err = NewRepository(db2, txStubEngine{err: gradeErr}).AutoSubmitSession(context.Background(), "sess-1")
	assert.ErrorIs(t, err, gradeErr)
	assert.ErrorContains(t, err, "grade")
}

// ── GradeAnswer ──────────────────────────────────────────────────────────────

// queueGradeLock queues the session as read under the lock: its owner, status and stored pass.
func queueGradeLock(f *fakeDB, owner, status string, passed bool) {
	f.queue([]string{"user_id", "status", "passed"}, [][]driver.Value{{owner, status, passed}})
}

// queueGradeRow queues the answer being graded: its status, maximum points, points, suggestion and flags.
func queueGradeRow(f *fakeDB, gradingStatus string, score float64, suggested any, flags any) {
	f.queue([]string{"grading_status", "max_score", "score", "ai_suggested_score_pct", "ai_flags"},
		[][]driver.Value{{gradingStatus, float64(10), score, suggested, flags}})
}

func TestGradeAnswer_LastAnswerFinalisesSession(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "grading_pending", false)
	queueGradeRow(f, "pending_manual", 0, nil, nil)
	f.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
	f.queue([]string{"round"}, [][]driver.Value{{float64(80)}})
	f.queue([]string{"passing_score_pct"}, [][]driver.Value{{float64(60)}})
	res, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "tenant-1", "10.0.0.1", 80, "good")
	require.NoError(t, err)
	assert.True(t, res.allGraded)
	assert.Equal(t, "submitted", res.sessionStatus)
	assert.Equal(t, "grading_pending", res.previousStatus, "the mail decision sees the transition")
	assert.False(t, res.passedFlipped)
	require.NotNil(t, res.finalScorePct)
	assert.Equal(t, 80.0, *res.finalScorePct)
	require.NotNil(t, res.passed)
	assert.True(t, *res.passed)
	assert.Equal(t, 1, countQueries(f, "UPDATE exam_sessions SET score_pct"))
	assert.Equal(t, 1, countQueries(f, "'answer.grade'"))
	assert.Equal(t, 1, countQueries(f, "FOR UPDATE"), "the session row is locked before the score write (#528)")
}

func TestGradeAnswer_OtherAnswersPendingLeavesSessionOpen(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "grading_pending", false)
	queueGradeRow(f, "pending_manual", 0, nil, nil)
	f.queue([]string{"count"}, [][]driver.Value{{int64(2)}})
	res, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "tenant-1", "", 50, "")
	require.NoError(t, err)
	assert.False(t, res.allGraded)
	assert.Equal(t, "grading_pending", res.sessionStatus)
	assert.Nil(t, res.finalScorePct)
	assert.Equal(t, 0, countQueries(f, "UPDATE exam_sessions SET score_pct"))
}

func TestGradeAnswer_NotFoundAndUpdateErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"user_id", "status", "passed"}, nil)
	_, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "g", "t", "", 50, "")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	db2, f2 := newFakeDB(t)
	queueGradeLock(f2, "owner-1", "grading_pending", false)
	f2.queue([]string{"grading_status", "max_score", "score", "ai_suggested_score_pct", "ai_flags"}, nil)
	_, err = NewRepository(db2, nil).GradeAnswer(context.Background(), "sess-1", "q1", "g", "t", "", 50, "")
	assert.ErrorIs(t, err, ErrSessionNotFound, "an answer that is not in the session is not found")

	db3, f3 := newFakeDB(t)
	queueGradeLock(f3, "owner-1", "grading_pending", false)
	queueGradeRow(f3, "pending_manual", 0, nil, nil)
	f3.xErr = errors.New("boom")
	_, err = NewRepository(db3, nil).GradeAnswer(context.Background(), "sess-1", "q1", "g", "t", "", 50, "")
	assert.ErrorIs(t, err, f3.xErr)
	assert.ErrorContains(t, err, "update score")
}

// FR-BB73 AC-21: the employee cannot grade their own session, and nothing is written.
func TestGradeAnswer_CannotGradeOwnSession(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "grading_pending", false)
	_, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "owner-1", "t", "", 50, "")
	assert.ErrorIs(t, err, ErrCannotGradeOwnSession)
	assert.Equal(t, 0, countQueries(f, "UPDATE session_question_scores"))
}

// FR-BB73 AC-19: a final session's graded answer is not re-graded silently.
func TestGradeAnswer_FinalSessionRefusesGradedAnswer(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "submitted", true)
	queueGradeRow(f, "graded", 5, nil, nil)
	_, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "t", "", 40, "")
	assert.ErrorIs(t, err, ErrSessionFinalised)
	assert.Equal(t, 0, countQueries(f, "UPDATE session_question_scores"))
}

// FR-BB73 AC-20: an override of a final session with an issued certificate is refused, and nothing changes.
func TestGradeAnswer_OverrideRefusedWhenCertificateIssued(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "submitted", true)
	queueGradeRow(f, "ai_graded", 2, nil, nil)
	f.queue([]string{"exists"}, [][]driver.Value{{true}})
	_, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "t", "", 40, "")
	assert.ErrorIs(t, err, ErrCertificateIssued)
	assert.Equal(t, 0, countQueries(f, "UPDATE session_question_scores"))
}

// FR-BB73 AC-19: an override of an ai_graded answer while the session waits is allowed, audited as an override, and
// sends no mail (the session is not final).
func TestGradeAnswer_OverrideWhileWaitingIsAudited(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "grading_pending", false)
	queueGradeRow(f, "ai_graded", 2, nil, nil)
	f.queue([]string{"count"}, [][]driver.Value{{int64(1)}})
	res, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "t", "", 40, "")
	require.NoError(t, err)
	assert.False(t, res.allGraded)
	assert.Equal(t, 1, countQueries(f, "'answer.ai_override'"))
	assert.Equal(t, 0, countQueries(f, "'answer.grade'"))
}
