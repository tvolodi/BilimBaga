package sessions

// Real-Postgres tests of the examiner's rules of FR-BB73 revision 2 (AC-17 to AC-21): self-grading, the final-session
// refusal, the audited override, the certificate refusal, the accepted comparison at two decimals, and the issuance
// against override under the session lock. Skipped unless TEST_DATABASE_URL is set; the names carry the TestAIGrade_
// prefix, so the verbose CI step shows PASS lines.

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/certificates"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noTenantConfig is the tenant configuration the certificate snapshot reads. The tests need none of it.
type noTenantConfig struct{}

func (noTenantConfig) GetAllConfig() map[string]json.RawMessage { return map[string]json.RawMessage{} }

// finalWithAIGrade builds a final session whose second answer carries an applied AI grade: the first answer is graded by
// a person at 100%, and the pass mark is 50%, so the AI score cannot decide pass or fail (AC-15, already passed). It
// returns the session, the employee, a grader distinct from the employee, and the questions.
func finalWithAIGrade(t *testing.T, db *sqlx.DB) (sessionID, employeeID, graderID string, qs []string) {
	t.Helper()
	sessionID, employeeID, qs = seedAIGradeSession(t, db, 2)
	graderID = seedGrader(t, db)
	repo := NewRepository(db, NewGradingEngine())
	ctx := context.Background()
	_, err := repo.GradeAnswer(ctx, sessionID, qs[0], graderID, "public", "127.0.0.1", 100, "")
	require.NoError(t, err)
	client := &gradeStubClient{response: `{"score_pct": 0, "reasoning": "wrong"}`, tokens: 5}
	runAIGradeTick(ctx, db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 10})
	status, _, passed := readAIGradeSession(t, db, sessionID)
	require.Equal(t, "submitted", status)
	require.True(t, passed, "2 of 4 points is the 50% pass mark")
	return sessionID, employeeID, graderID, qs
}

// AC-21: the employee cannot grade their own session, for every role. Nothing is written.
func TestAIGrade_SelfGradingIsRefused(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, employeeID, qs := seedAIGradeSession(t, db, 1)
	repo := NewRepository(db, NewGradingEngine())

	_, err := repo.GradeAnswer(context.Background(), sessionID, qs[0], employeeID, "public", "127.0.0.1", 100, "")
	assert.ErrorIs(t, err, ErrCannotGradeOwnSession)
	assert.Equal(t, "pending_manual", readAIGradeScore(t, db, sessionID, qs[0]).Status)
}

// AC-19: a final session's graded answer is not re-graded silently.
func TestAIGrade_FinalSessionRefusesAGradedAnswer(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, graderID, qs := finalWithAIGrade(t, db)
	repo := NewRepository(db, NewGradingEngine())

	_, err := repo.GradeAnswer(context.Background(), sessionID, qs[0], graderID, "public", "127.0.0.1", 0, "")
	assert.ErrorIs(t, err, ErrSessionFinalised)
	assert.Equal(t, "graded", readAIGradeScore(t, db, sessionID, qs[0]).Status)
}

// AC-19: an override of an ai_graded answer while the session still waits is audited as an override and sends no mail.
// Three answers, pass mark 70%: the first is graded by a person at 0%, the second is applied by AI (the remaining open
// points cannot reach the pass mark, so the score cannot decide it), and the third stays pending, so the session waits.
func TestAIGrade_OverrideWhileWaitingIsAuditedAndSilent(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 3)
	_, err := db.ExecContext(context.Background(), `UPDATE exams SET passing_score_pct = 70 WHERE id = (SELECT exam_id FROM exam_sessions WHERE id = $1)`, sessionID)
	require.NoError(t, err)
	grader := seedGrader(t, db)
	ctx := context.Background()
	rec := &mailRecorder{}
	svc := NewService(NewRepository(db, NewGradingEngine()), WithResultMailer(rec))

	_, err = svc.GradeAnswer(ctx, sessionID, qs[0], grader, "public", "127.0.0.1", GradeAnswerRequest{ScorePct: 0})
	require.NoError(t, err)
	// The third answer is out of the job's candidate set, so only the second is asked about.
	_, err = db.ExecContext(ctx, `UPDATE session_question_scores SET ai_grade_attempts = 2 WHERE session_id = $1 AND question_id = $2`, sessionID, qs[2])
	require.NoError(t, err)
	client := &gradeStubClient{response: `{"score_pct": 100, "reasoning": "right"}`, tokens: 5}
	runAIGradeTick(ctx, db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 10})
	require.Equal(t, "ai_graded", readAIGradeScore(t, db, sessionID, qs[1]).Status, "the AI grade is applied, and the session waits")
	status, _, _ := readAIGradeSession(t, db, sessionID)
	require.Equal(t, "grading_pending", status)

	_, err = svc.GradeAnswer(ctx, sessionID, qs[1], grader, "public", "127.0.0.1", GradeAnswerRequest{ScorePct: 50})
	require.NoError(t, err)
	assert.Equal(t, "graded", readAIGradeScore(t, db, sessionID, qs[1]).Status)
	assert.Equal(t, 1, countAudit(t, db, "answer.ai_override"))
	assert.Empty(t, rec.corrected)
	assert.Empty(t, rec.results, "the session is not final, so no result mail")
}

// AC-20: an override of a final session with an issued certificate changes nothing.
func TestAIGrade_OverrideRefusedWhenCertificateIssued(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, graderID, qs := finalWithAIGrade(t, db)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `UPDATE exams SET certificate_enabled = true WHERE id = (SELECT exam_id FROM exam_sessions WHERE id = $1)`, sessionID)
	require.NoError(t, err)
	_, _, err = certificates.NewService(certificates.NewRepository(db), noTenantConfig{}).GetOrCreate(ctx, sessionID, "", true)
	require.NoError(t, err)

	repo := NewRepository(db, NewGradingEngine())
	_, err = repo.GradeAnswer(ctx, sessionID, qs[1], graderID, "public", "127.0.0.1", 0, "")
	assert.ErrorIs(t, err, ErrCertificateIssued)
	assert.Equal(t, "ai_graded", readAIGradeScore(t, db, sessionID, qs[1]).Status, "the AI grade is unchanged")
}

// AC-19: an override of a final session without a certificate recomputes the result, and sends no mail when pass or
// fail does not change.
func TestAIGrade_OverrideWithoutFlipSendsNoMail(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, graderID, qs := finalWithAIGrade(t, db)
	rec := &mailRecorder{}
	svc := NewService(NewRepository(db, NewGradingEngine()), WithResultMailer(rec))

	_, err := svc.GradeAnswer(context.Background(), sessionID, qs[1], graderID, "public", "127.0.0.1", GradeAnswerRequest{ScorePct: 100})
	require.NoError(t, err)
	assert.Equal(t, 1, countAudit(t, db, "answer.ai_override"))
	_, scorePct, passed := readAIGradeSession(t, db, sessionID)
	require.True(t, scorePct.Valid)
	assert.InDelta(t, 100.0, scorePct.Float64, 1e-9, "the override recomputes the result: 4 of 4 points")
	assert.True(t, passed)
	assert.Empty(t, rec.corrected, "no pass or fail flip, so no corrected mail")
	assert.Empty(t, rec.results, "the result mail is not sent again")
}

// AC-17: accepted compares the posted score and the suggestion at two decimals, on the server.
func TestAIGrade_AcceptedComparesAtTwoDecimals(t *testing.T) {
	cases := []struct {
		name     string
		posted   float64
		accepted string
	}{
		{"same at two decimals", 50.004, "true"},
		{"a different score", 50.01, "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newAIGradePGDB(t)
			sessionID, _, qs := seedAIGradeSession(t, db, 1)
			ctx := context.Background()
			client := &gradeStubClient{response: `{"score_pct": 50, "reasoning": "half"}`, tokens: 5}
			runAIGradeTick(ctx, db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 10})
			require.True(t, readAISuggestion(t, db, sessionID, qs[0]).Pct.Valid)

			grader := seedGrader(t, db)
			_, err := NewRepository(db, NewGradingEngine()).GradeAnswer(ctx, sessionID, qs[0], grader, "public", "127.0.0.1", tc.posted, "")
			require.NoError(t, err)

			var accepted string
			require.NoError(t, db.GetContext(ctx, &accepted,
				`SELECT metadata->>'accepted' FROM audit_log WHERE action = 'answer.grade' AND entity_id = $1 ORDER BY created_at DESC LIMIT 1`, qs[0]))
			assert.Equal(t, tc.accepted, accepted)
		})
	}
}

// AC-20: an issuance and an override of the same final session run concurrently. Either the issuance goes first, the
// certificate exists and the override is refused with 409, or the override goes first and is audited, and the
// certificate is issued after it. The override never succeeds after a certificate exists.
func TestAIGrade_CertificateIssuanceAndOverrideAreSerialised(t *testing.T) {
	db := newAIGradePGDB(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		sessionID, _, graderID, qs := finalWithAIGrade(t, db)
		_, err := db.ExecContext(ctx, `UPDATE exams SET certificate_enabled = true WHERE id = (SELECT exam_id FROM exam_sessions WHERE id = $1)`, sessionID)
		require.NoError(t, err)

		certs := certificates.NewService(certificates.NewRepository(db), noTenantConfig{})
		repo := NewRepository(db, NewGradingEngine())
		var wg sync.WaitGroup
		var issueErr, overrideErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, issueErr = certs.GetOrCreate(ctx, sessionID, "", true)
		}()
		go func() {
			defer wg.Done()
			_, overrideErr = repo.GradeAnswer(ctx, sessionID, qs[1], graderID, "public", "127.0.0.1", 0, "")
		}()
		wg.Wait()

		require.NoError(t, issueErr)
		var count int
		require.NoError(t, db.GetContext(ctx, &count, `SELECT COUNT(*) FROM certificates WHERE session_id = $1`, sessionID))
		assert.Equal(t, 1, count, "the certificate exists after issuance")
		if overrideErr != nil {
			assert.True(t, errors.Is(overrideErr, ErrCertificateIssued), "issuance went first, so the override is refused: %v", overrideErr)
		} else {
			var audited int
			require.NoError(t, db.GetContext(ctx, &audited,
				`SELECT COUNT(*) FROM audit_log WHERE action = 'answer.ai_override' AND metadata->>'session_id' = $1`, sessionID))
			assert.Equal(t, 1, audited, "the override went first and was audited")
		}
	}
}
