package sessions

// Real-Postgres tests of the AI grading job (#528). Skipped unless TEST_DATABASE_URL is set. Each test runs the full
// migration set in a throwaway schema and drops it afterwards, so public tables are never touched.
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' \
//	    go test ./internal/sessions -run TestAIGrade -v

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAIGradePGDB opens a connection to a fresh schema with every migration applied.
func newAIGradePGDB(t *testing.T) *sqlx.DB {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping real-Postgres AI grading tests")
	}
	schema := fmt.Sprintf("bb_528_%d", time.Now().UnixNano())
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}

	admin, err := sqlx.Open("postgres", base)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })
	_, err = admin.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`) })

	db, err := sqlx.Open("postgres", base+sep+"search_path="+schema)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, dbpkg.RunMigrations(db, "../../migrations"))
	return db
}

// seedAIGradeSession creates a submitted session whose questionCount short-text answers are pending_manual.
// Each question has max_score 2, the model answer "Paris" and auto_grade on. The exam passes at 50%.
func seedAIGradeSession(t *testing.T, db *sqlx.DB, questionCount int) (sessionID, userID string, questionIDs []string) {
	t.Helper()
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		require.NoError(t, err)
	}

	var roleID string
	must(db.GetContext(ctx, &roleID, `INSERT INTO roles (name) VALUES ('ai-grader') ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id::text`))
	must(db.GetContext(ctx, &userID, `INSERT INTO users (email, password_hash, full_name, role_id) VALUES (gen_random_uuid()::text || '@528.test', 'x', 'Employee', $1) RETURNING id::text`, roleID))

	var categoryID string
	must(db.GetContext(ctx, &categoryID, `INSERT INTO categories (name) VALUES ('ai-grade') RETURNING id::text`))

	var examID string
	must(db.GetContext(ctx, &examID, `INSERT INTO exams (title, time_limit_minutes, passing_score_pct, created_by) VALUES ('AI grading', 30, 50, $1) RETURNING id::text`, userID))

	must(db.GetContext(ctx, &sessionID, `
INSERT INTO exam_sessions (exam_id, user_id, status, seed, expires_at, submitted_at)
VALUES ($1, $2, 'grading_pending', 42, NOW() + INTERVAL '1 hour', NOW())
RETURNING id::text`, examID, userID))

	for i := 0; i < questionCount; i++ {
		var qid string
		must(db.GetContext(ctx, &qid, `
INSERT INTO questions (category_id, difficulty, type, created_by, auto_grade, model_answer)
VALUES ($1, 'easy', 'shorttext', $2, true, 'Paris')
RETURNING id::text`, categoryID, userID))
		_, err := db.ExecContext(ctx, `INSERT INTO session_answers (session_id, question_id, text_answer) VALUES ($1, $2, 'Paris, France')`, sessionID, qid)
		must(err)
		_, err = db.ExecContext(ctx, `INSERT INTO session_question_scores (session_id, question_id, max_score, grading_status) VALUES ($1, $2, 2, 'pending_manual')`, sessionID, qid)
		must(err)
		questionIDs = append(questionIDs, qid)
	}
	return sessionID, userID, questionIDs
}

type aiGradeScoreRow struct {
	Score    float64        `db:"score"`
	Status   string         `db:"grading_status"`
	Reason   sql.NullString `db:"ai_reasoning"`
	Attempts int            `db:"ai_grade_attempts"`
}

func readAIGradeScore(t *testing.T, db *sqlx.DB, sessionID, questionID string) aiGradeScoreRow {
	t.Helper()
	var row aiGradeScoreRow
	err := db.GetContext(context.Background(), &row, `
SELECT score, grading_status, ai_reasoning, ai_grade_attempts
FROM session_question_scores WHERE session_id = $1 AND question_id = $2`, sessionID, questionID)
	require.NoError(t, err)
	return row
}

func readAIGradeSession(t *testing.T, db *sqlx.DB, sessionID string) (status string, scorePct sql.NullFloat64, passed bool) {
	t.Helper()
	err := db.QueryRowContext(context.Background(),
		`SELECT status::text, score_pct, passed FROM exam_sessions WHERE id = $1`, sessionID).Scan(&status, &scorePct, &passed)
	require.NoError(t, err)
	return status, scorePct, passed
}

func countAIGradeUsage(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.GetContext(context.Background(), &n,
		`SELECT COUNT(*) FROM ai_usage_log WHERE feature = $1`, aiGradeFeature))
	return n
}

// gradeOne claims and grades the next candidate once, as one tick would.
func gradeOne(t *testing.T, db *sqlx.DB, cfg AIGradeConfig) {
	t.Helper()
	cands, err := selectAIGradeCandidates(context.Background(), db, 1)
	require.NoError(t, err)
	require.Len(t, cands, 1)
	require.NoError(t, gradeAIAnswer(context.Background(), db, slog.Default(), cfg, cands[0]))
}

func TestAIGrade_TickGradesAndFinalisesTheLastAnswer(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 1)

	client := &gradeStubClient{response: `{"score_pct": 50, "reasoning": "half"}`, tokens: 30}
	var finalised []string
	cfg := AIGradeConfig{
		Client:      client,
		Model:       "test-model",
		DailyLimit:  10,
		OnFinalised: func(id string) { finalised = append(finalised, id) },
	}
	runAIGradeTick(context.Background(), db, slog.Default(), cfg)

	require.Equal(t, 1, client.calls)
	row := readAIGradeScore(t, db, sessionID, qs[0])
	assert.Equal(t, "ai_graded", row.Status)
	assert.InDelta(t, 1.0, row.Score, 1e-9, "50% of max_score 2")
	assert.Equal(t, "half", row.Reason.String)
	assert.Equal(t, 1, row.Attempts)

	status, scorePct, passed := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "submitted", status)
	require.True(t, scorePct.Valid)
	assert.InDelta(t, 50.0, scorePct.Float64, 1e-9)
	assert.True(t, passed, "50% meets the 50% pass mark")
	assert.Equal(t, []string{sessionID}, finalised, "the result notification fires once, when the session finalises")
	assert.Equal(t, 1, countAIGradeUsage(t, db))
}

func TestAIGrade_SessionStaysPendingUntilTheLastAnswer(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, _ := seedAIGradeSession(t, db, 2)

	client := &gradeStubClient{response: `{"score_pct": 100, "reasoning": "right"}`, tokens: 5}
	var finalised []string
	cfg := AIGradeConfig{Client: client, DailyLimit: 10, OnFinalised: func(id string) { finalised = append(finalised, id) }}

	gradeOne(t, db, cfg)
	status, _, _ := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "grading_pending", status, "one answer is still pending")
	assert.Empty(t, finalised)

	gradeOne(t, db, cfg)
	status, scorePct, _ := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "submitted", status)
	require.True(t, scorePct.Valid)
	assert.InDelta(t, 100.0, scorePct.Float64, 1e-9)
	assert.Equal(t, []string{sessionID}, finalised)
}

// A person grades the answer while the provider call is in flight. The AI result is discarded, the person's score
// stays, and the notification does not fire twice.
func TestAIGrade_HumanGradeDuringTheCallWins(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, userID, qs := seedAIGradeSession(t, db, 1)
	repo := NewRepository(db, NewGradingEngine())

	client := &gradeStubClient{response: `{"score_pct": 0, "reasoning": "wrong"}`, tokens: 5}
	client.onCall = func() {
		_, err := repo.GradeAnswer(context.Background(), sessionID, qs[0], userID, "public", "127.0.0.1", 100, "human")
		require.NoError(t, err)
	}
	var finalised []string
	cfg := AIGradeConfig{Client: client, DailyLimit: 10, OnFinalised: func(id string) { finalised = append(finalised, id) }}

	gradeOne(t, db, cfg)

	row := readAIGradeScore(t, db, sessionID, qs[0])
	assert.Equal(t, "graded", row.Status, "the person's grade stays")
	assert.InDelta(t, 2.0, row.Score, 1e-9, "the person gave 100% of max_score 2")
	status, _, _ := readAIGradeSession(t, db, sessionID)
	assert.Equal(t, "submitted", status)
	assert.Empty(t, finalised, "the person's own finalise path notifies, not the AI job")
}

// Two workers claim the same answer. Only one claim succeeds.
func TestAIGrade_ClaimIsExclusive(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 1)
	cands, err := selectAIGradeCandidates(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, cands, 1)

	first, err := claimAIGrade(context.Background(), db, cands[0])
	require.NoError(t, err)
	second, err := claimAIGrade(context.Background(), db, cands[0])
	require.NoError(t, err)
	assert.True(t, first)
	assert.False(t, second, "the stale read must not claim the row again")
	assert.Equal(t, 1, readAIGradeScore(t, db, sessionID, qs[0]).Attempts)
}

// After two attempts the answer is left for a person to grade.
func TestAIGrade_AttemptCapStopsRetries(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 1)
	_, err := db.ExecContext(context.Background(), `UPDATE session_question_scores SET ai_grade_attempts = 2 WHERE session_id = $1`, sessionID)
	require.NoError(t, err)

	cands, err := selectAIGradeCandidates(context.Background(), db, 10)
	require.NoError(t, err)
	assert.Empty(t, cands)
	assert.Equal(t, "pending_manual", readAIGradeScore(t, db, sessionID, qs[0]).Status)
}

// A provider error leaves the answer pending with its attempt counted. The next tick grades it, because the state
// lives in the database and a restart loses nothing.
func TestAIGrade_ProviderErrorIsRetriedFromTheDatabase(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 1)

	failing := &gradeStubClient{err: fmt.Errorf("timeout")}
	runAIGradeTick(context.Background(), db, slog.Default(), AIGradeConfig{Client: failing, DailyLimit: 10})
	row := readAIGradeScore(t, db, sessionID, qs[0])
	assert.Equal(t, "pending_manual", row.Status)
	assert.Equal(t, 1, row.Attempts)

	working := &gradeStubClient{response: `{"score_pct": 100, "reasoning": "right"}`, tokens: 5}
	runAIGradeTick(context.Background(), db, slog.Default(), AIGradeConfig{Client: working, DailyLimit: 10})
	row = readAIGradeScore(t, db, sessionID, qs[0])
	assert.Equal(t, "ai_graded", row.Status)
	assert.Equal(t, 2, row.Attempts)
}

// A reply without JSON leaves the answer pending.
func TestAIGrade_UnusableReplyLeavesTheAnswerPending(t *testing.T) {
	db := newAIGradePGDB(t)
	sessionID, _, qs := seedAIGradeSession(t, db, 1)

	client := &gradeStubClient{response: "I cannot grade this", tokens: 5}
	runAIGradeTick(context.Background(), db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 10})
	assert.Equal(t, "pending_manual", readAIGradeScore(t, db, sessionID, qs[0]).Status)
}

// The daily cap counts provider calls in the last 24 hours and caps this tick's batch to what is left.
func TestAIGrade_DailyCapLimitsTheBatch(t *testing.T) {
	db := newAIGradePGDB(t)
	_, _, _ = seedAIGradeSession(t, db, 2)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		_, err := db.ExecContext(ctx, `INSERT INTO ai_usage_log (user_id, feature, tokens_used, model) VALUES (NULL, $1, 10, 'm')`, aiGradeFeature)
		require.NoError(t, err)
	}

	client := &gradeStubClient{response: `{"score_pct": 100, "reasoning": "right"}`, tokens: 5}
	runAIGradeTick(ctx, db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 3})
	assert.Equal(t, 1, client.calls, "3 allowed, 2 already used: one more call")
	assert.Equal(t, 3, countAIGradeUsage(t, db))

	runAIGradeTick(ctx, db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 3})
	assert.Equal(t, 1, client.calls, "at the cap: no call")
}

// Only short-text answers with auto_grade, a model answer and a submission in the last 24 hours are candidates.
func TestAIGrade_CandidateFilters(t *testing.T) {
	db := newAIGradePGDB(t)
	ctx := context.Background()
	eligible, _, _ := seedAIGradeSession(t, db, 1)
	oldSession, _, _ := seedAIGradeSession(t, db, 1)
	noAutoGrade, _, noAutoQ := seedAIGradeSession(t, db, 1)

	_, err := db.ExecContext(ctx, `UPDATE exam_sessions SET submitted_at = NOW() - INTERVAL '2 days' WHERE id = $1`, oldSession)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE questions SET auto_grade = false WHERE id = $1`, noAutoQ[0])
	require.NoError(t, err)

	cands, err := selectAIGradeCandidates(ctx, db, 10)
	require.NoError(t, err)
	require.Len(t, cands, 1)
	assert.Equal(t, eligible, cands[0].SessionID)
	assert.NotEqual(t, noAutoGrade, cands[0].SessionID)
}
