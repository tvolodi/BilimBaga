package sessions

// Real-Postgres test of the examiner's grading detail AI fields (FR-BB73 AC-18): the suggestion of a waiting answer, and
// every ai_graded answer with its AI marker, score and reasoning. Skipped unless TEST_DATABASE_URL is set; the name
// carries the TestAIGrade_ prefix, so the verbose CI step shows its PASS line.

import (
	"context"
	"log/slog"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// linkSessionQuestions adds the session's question rows, which the grading detail joins. A real session always has them;
// the AI fixture does not, so the test adds them after the job has run.
func linkSessionQuestions(t *testing.T, db *sqlx.DB, sessionID string, questionIDs []string) {
	t.Helper()
	for i, qid := range questionIDs {
		_, err := db.ExecContext(context.Background(), `INSERT INTO session_questions (session_id, question_id, sort_order) VALUES ($1, $2, $3)`, sessionID, qid, i)
		require.NoError(t, err)
	}
}

func TestAIGrade_GradingDetailShowsSuggestionAndAIGrades(t *testing.T) {
	db := newAIGradePGDB(t)
	ctx := context.Background()
	repo := NewRepository(db, NewGradingEngine())

	// A finished session with one AI-graded answer: the person's grade at 100% already passes the 50% mark, so the AI
	// score of 0 is applied and the session stays final (AC-15).
	finalSession, _, _, qs := finalWithAIGrade(t, db)
	linkSessionQuestions(t, db, finalSession, qs)
	detail, err := repo.GetGradingDetail(ctx, finalSession)
	require.NoError(t, err)
	require.Len(t, detail.Questions, 2)
	assert.Equal(t, "submitted", detail.SessionStatus, "the session is final, so its graded answers are read-only in the grader")
	byID := map[string]GradingQuestionItem{}
	for _, q := range detail.Questions {
		byID[q.QuestionID] = q
	}

	// A person's grade carries no AI fields.
	human := byID[qs[0]]
	assert.Equal(t, "graded", human.GradingStatus)
	require.NotNil(t, human.CurrentScorePct)
	assert.InDelta(t, 100, *human.CurrentScorePct, 1e-9)
	assert.Nil(t, human.AISuggestedScorePct)
	assert.Nil(t, human.AIReasoning)

	// Every ai_graded answer shows its marker, its score, its suggestion and its reasoning.
	ai := byID[qs[1]]
	assert.Equal(t, "ai_graded", ai.GradingStatus)
	require.NotNil(t, ai.CurrentScorePct)
	assert.InDelta(t, 0, *ai.CurrentScorePct, 1e-9)
	require.NotNil(t, ai.AISuggestedScorePct)
	assert.InDelta(t, 0, *ai.AISuggestedScorePct, 1e-9)
	require.NotNil(t, ai.AISuggestedAt)
	require.NotNil(t, ai.AIReasoning)
	assert.Equal(t, "wrong", *ai.AIReasoning)
	assert.Empty(t, ai.AIFlags, "an applied grade carries no flags")

	// A waiting answer shows the suggestion, its reasoning and its flags, and no score yet. One question, so the AI score
	// of 50 is decisive and stays a suggestion (FR-BB73 AC-15).
	pendingSession, _, pendingQS := seedAIGradeSession(t, db, 1)
	client := &gradeStubClient{response: `{"score_pct": 50, "reasoning": "half"}`, tokens: 5}
	runAIGradeTick(ctx, db, slog.Default(), AIGradeConfig{Client: client, DailyLimit: 10})
	require.Equal(t, "pending_manual", readAIGradeScore(t, db, pendingSession, pendingQS[0]).Status)
	linkSessionQuestions(t, db, pendingSession, pendingQS)

	detail, err = repo.GetGradingDetail(ctx, pendingSession)
	require.NoError(t, err)
	require.Len(t, detail.Questions, 1)
	assert.Equal(t, "grading_pending", detail.SessionStatus)
	waiting := detail.Questions[0]
	assert.Equal(t, "pending_manual", waiting.GradingStatus)
	assert.Nil(t, waiting.CurrentScorePct)
	require.NotNil(t, waiting.AISuggestedScorePct)
	assert.InDelta(t, 50, *waiting.AISuggestedScorePct, 1e-9)
	require.NotNil(t, waiting.AISuggestedAt)
	require.NotNil(t, waiting.AIReasoning)
	assert.Equal(t, "half", *waiting.AIReasoning)
	assert.Contains(t, waiting.AIFlags, aiFlagDecisivePass)

	// The queue shows the exam's mode (FR-BB73 AC-22).
	// The queue is scoped by the caller's role; a super admin sees every department.
	superCtx := context.WithValue(ctx, ctxkeys.CtxRole, "super_admin")
	items, _, err := repo.ListGradingQueue(superCtx, nil, nil, nil, 1, 20)
	require.NoError(t, err)
	var queued *GradingQueueItem
	for i := range items {
		if items[i].SessionID == pendingSession {
			queued = &items[i]
		}
	}
	require.NotNil(t, queued, "the waiting session is in the queue")
	assert.Equal(t, "auto", queued.AIGradingMode)
}
