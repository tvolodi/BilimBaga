package sessions

// #565: real-Postgres tests of FR-BB42 AC-11 and the check order of FR-BB73 AC-19: a row whose question is not short-text
// answers 422 ANSWER_NOT_GRADABLE and writes nothing; the self-grading rule (403), the row read (404) and the type check
// (422) come before the row rules (409). Skipped unless TEST_DATABASE_URL is set; the names carry the TestAIGrade_ prefix
// so the verbose CI step shows PASS lines.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedGradedMultipleChoice adds a single-choice question to the session, auto-graded at its full 2 points, and returns
// its question id.
func seedGradedMultipleChoice(t *testing.T, db *sqlx.DB, sessionID, userID string) string {
	t.Helper()
	ctx := context.Background()
	var categoryID, questionID string
	require.NoError(t, db.GetContext(ctx, &categoryID, `INSERT INTO categories (name) VALUES ($1) RETURNING id::text`,
		fmt.Sprintf("mc-%d", time.Now().UnixNano())))
	require.NoError(t, db.GetContext(ctx, &questionID, `
INSERT INTO questions (category_id, difficulty, type, created_by, auto_grade)
VALUES ($1, 'easy', 'single', $2, true) RETURNING id::text`, categoryID, userID))
	_, err := db.ExecContext(ctx, `
INSERT INTO session_question_scores (session_id, question_id, max_score, score, grading_status)
VALUES ($1, $2, 2, 2, 'graded')`, sessionID, questionID)
	require.NoError(t, err)
	return questionID
}

func countAuditRows(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.GetContext(context.Background(), &n, `SELECT COUNT(*) FROM audit_log`))
	return n
}

// AC-11: the examiner cannot override an auto-graded row. The refusal writes nothing and the session stays pending.
func TestAIGrade_RealPostgres_MultipleChoiceRowIsNotGradableAndNothingIsWritten(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, employeeID, _ := seedAIGradeSession(t, db, 1)
	mc := seedGradedMultipleChoice(t, db, sessionID, employeeID)
	graderID := seedGrader(t, db)
	repo := NewRepository(db, NewGradingEngine())
	before := readAIGradeScore(t, db, sessionID, mc)
	auditBefore := countAuditRows(t, db)

	_, err := repo.GradeAnswer(context.Background(), sessionID, mc, graderID, "public", "127.0.0.1", 0, "override")

	assert.ErrorIs(t, err, ErrAnswerNotGradable)
	assert.Equal(t, before, readAIGradeScore(t, db, sessionID, mc), "score and grading_status are unchanged")
	assert.Equal(t, auditBefore, countAuditRows(t, db), "a refused grade writes no audit row")
	status, _, _ := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "grading_pending", status)
}

// The short-text rows of the same session are still graded.
func TestAIGrade_RealPostgres_ShortTextRowIsStillGraded(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 1)
	graderID := seedGrader(t, db)
	repo := NewRepository(db, NewGradingEngine())

	_, err := repo.GradeAnswer(context.Background(), sessionID, qs[0], graderID, "public", "127.0.0.1", 100, "")

	require.NoError(t, err)
	assert.Equal(t, "graded", readAIGradeScore(t, db, sessionID, qs[0]).Status)
}

// AC-19 order: the self-grading rule (403) comes before the type check, so the employee gets 403, not 422.
func TestAIGrade_RealPostgres_OwnerIsRefusedBeforeTheTypeCheck(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, employeeID, _ := seedAIGradeSession(t, db, 0)
	mc := seedGradedMultipleChoice(t, db, sessionID, employeeID)
	repo := NewRepository(db, NewGradingEngine())

	_, err := repo.GradeAnswer(context.Background(), sessionID, mc, employeeID, "public", "127.0.0.1", 0, "")

	assert.ErrorIs(t, err, ErrCannotGradeOwnSession)
}

// AC-19 order: an unknown row is 404 before the type check.
func TestAIGrade_RealPostgres_UnknownRowIsNotFoundBeforeTheTypeCheck(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, _ := seedAIGradeSession(t, db, 0)
	graderID := seedGrader(t, db)
	repo := NewRepository(db, NewGradingEngine())

	_, err := repo.GradeAnswer(context.Background(), sessionID, "00000000-0000-4000-8000-00000000dead", graderID, "public", "127.0.0.1", 0, "")

	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// AC-19 order: the type check (422) comes before the final-session rule (409). A multiple-choice row of a final session
// is not gradable, whatever the session state.
func TestAIGrade_RealPostgres_TypeIsCheckedBeforeTheFinalRules(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, employeeID, _ := seedAIGradeSession(t, db, 0)
	mc := seedGradedMultipleChoice(t, db, sessionID, employeeID)
	_, err := db.ExecContext(context.Background(), `UPDATE exam_sessions SET status = 'submitted', passed = true WHERE id = $1`, sessionID)
	require.NoError(t, err)
	graderID := seedGrader(t, db)
	repo := NewRepository(db, NewGradingEngine())

	_, err = repo.GradeAnswer(context.Background(), sessionID, mc, graderID, "public", "127.0.0.1", 0, "")

	assert.ErrorIs(t, err, ErrAnswerNotGradable)
	assert.False(t, errors.Is(err, ErrSessionFinalised), "the type check is decided before the final-session rule")
}
