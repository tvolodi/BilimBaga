package sessions

// Real-Postgres tests of the FR-BB322 result experience: the mail once per transition, the masked employee reads, the
// attempt counts, and the one AI eligibility predicate. Skipped unless TEST_DATABASE_URL is set; the command is in
// aigradejob_pg_test.go.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedInProgressSession creates a user, an exam with questionCount short-text questions, and an in_progress session.
// expiresIn is a Postgres interval from now, such as "1 hour" or "-1 hour".
func seedInProgressSession(t *testing.T, db *sqlx.DB, questionCount int, expiresIn string) (sessionID, examID, userID string, questionIDs []string) {
	t.Helper()
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		require.NoError(t, err)
	}

	var roleID string
	must(db.GetContext(ctx, &roleID, `INSERT INTO roles (name) VALUES ('ai-grader') ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id::text`))
	must(db.GetContext(ctx, &userID, `INSERT INTO users (email, password_hash, full_name, role_id) VALUES (gen_random_uuid()::text || '@fr322.test', 'x', 'Employee', $1) RETURNING id::text`, roleID))
	var categoryID string
	must(db.GetContext(ctx, &categoryID, `INSERT INTO categories (name) VALUES ('fr322') RETURNING id::text`))
	must(db.GetContext(ctx, &examID, `INSERT INTO exams (title, time_limit_minutes, passing_score_pct, created_by, certificate_enabled) VALUES ('FR-BB322', 30, 50, $1, true) RETURNING id::text`, userID))
	must(db.GetContext(ctx, &sessionID, `
INSERT INTO exam_sessions (exam_id, user_id, status, seed, expires_at)
VALUES ($1, $2, 'in_progress', 42, NOW() + $3::interval)
RETURNING id::text`, examID, userID, expiresIn))

	for i := 0; i < questionCount; i++ {
		var qid string
		must(db.GetContext(ctx, &qid, `
INSERT INTO questions (category_id, difficulty, type, created_by, auto_grade, model_answer)
VALUES ($1, 'easy', 'shorttext', $2, true, 'Paris')
RETURNING id::text`, categoryID, userID))
		_, err := db.ExecContext(ctx, `INSERT INTO session_questions (session_id, question_id, sort_order) VALUES ($1, $2, $3)`, sessionID, qid, i)
		must(err)
		_, err = db.ExecContext(ctx, `INSERT INTO session_answers (session_id, question_id, text_answer) VALUES ($1, $2, 'Paris, France')`, sessionID, qid)
		must(err)
		questionIDs = append(questionIDs, qid)
	}
	return sessionID, examID, userID, questionIDs
}

// FR-BB322 AC-9: a manual submit sends exam_submitted once; a second submit is a replay and sends nothing.
func TestSubmitSession_MailOncePerSession(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, userID, _ := seedInProgressSession(t, db, 1, "1 hour")
	rec := &mailRecorder{}
	svc := NewService(NewRepository(db, NewGradingEngine()), WithResultMailer(rec))
	ctx := context.Background()

	first, err := svc.SubmitSession(ctx, sessionID, userID, "public", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "grading_pending", first.Status)
	assert.True(t, first.Transitioned)

	second, err := svc.SubmitSession(ctx, sessionID, userID, "public", "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, second.Transitioned)

	assert.Equal(t, []string{sessionID}, rec.submitted)
	assert.Empty(t, rec.results, "a waiting session gets no result mail (AC-10)")
}

// FR-BB322 AC-9: an auto-submit that arrives after a manual submit is a replay: no transition, no mail.
func TestAutoSubmitSession_AfterManualSubmitIsReplay(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, userID, _ := seedInProgressSession(t, db, 1, "1 hour")
	repo := NewRepository(db, NewGradingEngine())
	ctx := context.Background()

	_, err := NewService(repo).SubmitSession(ctx, sessionID, userID, "public", "127.0.0.1")
	require.NoError(t, err)

	res, err := repo.AutoSubmitSession(ctx, sessionID)
	require.NoError(t, err)
	assert.False(t, res.Transitioned)
	assert.Equal(t, "grading_pending", res.Status, "the replay leaves the status as it is")
}

// FR-BB322 AC-9: the expiry job sends exam_submitted once, for a session that moved to grading_pending.
func TestAutoSubmitJob_ExpiredShortTextSendsOneMail(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, examID, userID, _ := seedInProgressSession(t, db, 1, "-1 hour")
	rec := &mailRecorder{}
	sess := expiredSession{ID: sessionID, ExamID: examID, UserID: userID, ExpiresAt: time.Now().Add(-time.Hour)}
	ctx := context.Background()
	logger := slog.Default()
	engine := NewGradingEngine()

	require.NoError(t, processOneExpiredSession(ctx, db, logger, sess, engine, rec.TriggerExamSubmitted))
	require.NoError(t, processOneExpiredSession(ctx, db, logger, sess, engine, rec.TriggerExamSubmitted), "a second run finds nothing to do")

	assert.Equal(t, []string{sessionID}, rec.submitted)
	status, _, _ := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "grading_pending", status)
}

// FR-BB322 AC-9: an expired session without short-text answers is auto-submitted and sends no exam_submitted mail.
func TestAutoSubmitJob_ExpiredWithoutShortTextSendsNoMail(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, examID, userID, _ := seedInProgressSession(t, db, 0, "-1 hour")
	rec := &mailRecorder{}
	sess := expiredSession{ID: sessionID, ExamID: examID, UserID: userID, ExpiresAt: time.Now().Add(-time.Hour)}

	require.NoError(t, processOneExpiredSession(context.Background(), db, slog.Default(), sess, NewGradingEngine(), rec.TriggerExamSubmitted))

	assert.Empty(t, rec.submitted)
	status, _, _ := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "auto_submitted", status)
}

// FR-BB322 AC-13: a waiting session returns no score and no pass to the employee, from the real rows, while the
// examiner keeps the stored values.
func TestEmployeeReads_MaskAWaitingSession(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, userID, _ := seedAIGradeSession(t, db, 1)
	_, err := db.ExecContext(context.Background(), `UPDATE exam_sessions SET score_pct = 40, passed = true WHERE id = $1`, sessionID)
	require.NoError(t, err)
	var examID string
	require.NoError(t, db.GetContext(context.Background(), &examID, `SELECT exam_id::text FROM exam_sessions WHERE id = $1`, sessionID))
	repo := NewRepository(db, NewGradingEngine())
	ctx := context.Background()

	history, total, err := repo.GetExamHistory(ctx, examID, userID, 1, 100)
	require.NoError(t, err)
	require.Equal(t, 1, total, "a waiting session is in the history (AC-1)")
	assert.Nil(t, history[0].ScorePct)
	assert.False(t, history[0].Passed)

	results, total, err := repo.GetMyResults(ctx, userID, 1, 20, "date", "desc")
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Nil(t, results[0].ScorePct)
	assert.False(t, results[0].Passed)
	assert.False(t, results[0].CertificateAvailable)

	employee, err := NewService(repo).GetSessionResult(ctx, sessionID, userID)
	require.NoError(t, err)
	assert.Nil(t, employee.ScorePct)
	assert.False(t, employee.Passed)
	assert.Empty(t, employee.PerSectionScores)
	assert.Equal(t, 1, employee.PendingQuestionCount)
	assert.Equal(t, 1, employee.TotalGradableCount)
	require.NotNil(t, employee.GradingExpectedBy)
	assert.Equal(t, "examiner", *employee.GradingExpectedBy)

	examiner, err := NewService(repo).GetAdminSessionResult(ctx, sessionID)
	require.NoError(t, err)
	if assert.NotNil(t, examiner.ScorePct) {
		assert.InDelta(t, 40.0, *examiner.ScorePct, 0.01)
	}
	assert.True(t, examiner.Passed)
}

// FR-BB322 AC-1: the start check, the history and the my-results count all see a waiting session.
func TestCountingPlaces_SeeAWaitingSession(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, userID, _ := seedAIGradeSession(t, db, 1)
	repo := NewRepository(db, NewGradingEngine())
	ctx := context.Background()

	var examID string
	require.NoError(t, db.GetContext(ctx, &examID, `SELECT exam_id::text FROM exam_sessions WHERE id = $1`, sessionID))

	waiting, err := repo.HasGradingPendingSession(ctx, examID, userID)
	require.NoError(t, err)
	assert.True(t, waiting)

	finished, err := repo.CountFinishedSessions(ctx, examID, userID)
	require.NoError(t, err)
	assert.Equal(t, 1, finished)

	_, historyTotal, err := repo.GetExamHistory(ctx, examID, userID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, historyTotal)

	_, myTotal, err := repo.GetMyResults(ctx, userID, 1, 20, "date", "desc")
	require.NoError(t, err)
	assert.Equal(t, 1, myTotal)
}

// FR-BB322 AC-4: the number the result service uses to name the grader is the job's own selection. When an answer
// drops out of the job's predicate, both drop it.
func TestGradingExpectedBy_UsesTheJobsPredicate(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, userID, qs := seedAIGradeSession(t, db, 2)
	repo := NewRepository(db, NewGradingEngine())
	ctx := context.Background()
	svc := NewService(repo, WithAIGrading(true, 10))

	eligible, err := repo.AICandidateCount(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, 2, eligible)
	assert.Len(t, candidatesFor(t, db, sessionID), 2, "the job selects the same two answers")

	resp, err := svc.GetSessionResult(ctx, sessionID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp.GradingExpectedBy)
	assert.Equal(t, "ai", *resp.GradingExpectedBy)

	// Turn off auto-grading for one question: it leaves the job's predicate.
	_, err = db.ExecContext(ctx, `UPDATE questions SET auto_grade = false WHERE id = $1`, qs[0])
	require.NoError(t, err)
	eligible, err = repo.AICandidateCount(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, 1, eligible)
	assert.Len(t, candidatesFor(t, db, sessionID), 1)

	// The other answer has used its two attempts: nothing is eligible, so an examiner grades.
	_, err = db.ExecContext(ctx, `UPDATE session_question_scores SET ai_grade_attempts = 2 WHERE session_id = $1 AND question_id = $2`, sessionID, qs[1])
	require.NoError(t, err)
	resp, err = svc.GetSessionResult(ctx, sessionID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp.GradingExpectedBy)
	assert.Equal(t, "examiner", *resp.GradingExpectedBy)
}

// candidatesFor returns the job's batch selection, restricted to one session.
func candidatesFor(t *testing.T, db *sqlx.DB, sessionID string) []aiGradeCandidate {
	t.Helper()
	rows, err := selectAIGradeCandidates(context.Background(), db, 100)
	require.NoError(t, err)
	var mine []aiGradeCandidate
	for _, c := range rows {
		if c.SessionID == sessionID {
			mine = append(mine, c)
		}
	}
	return mine
}
