package exams

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (task #354) for exam create and update, rule listing, the
// manual-rule availability count, and manual-question writes. Each path gets
// success, not-found or wrapped-error cases where the code and fake allow.

var examUpdateCols = []string{
	"id", "title", "description", "status", "time_limit_minutes", "passing_score_pct",
	"max_attempts", "available_from", "available_until", "shuffle_questions", "shuffle_options",
	"show_answers", "on_tab_switch", "certificate_enabled", "adaptive", "created_by", "created_at", "updated_at",
}

func examUpdateRow(id string) []driver.Value {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return []driver.Value{id, "Fire Safety", nil, "draft", int64(30), float64(70),
		int64(2), nil, nil, true, false, "after_submit", "warn", false, false, "user-1", at, at}
}

func TestExamCreate_SetsIDAndTimestamps(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue([]string{"id", "created_at", "updated_at"}, [][]driver.Value{{"exam-9", at, at}})
	e := &Exam{Title: "Fire Safety", Status: "draft", CreatedBy: "user-1", ShowAnswers: "after_submit", OnTabSwitch: "warn"}
	require.NoError(t, NewRepository(db).Create(context.Background(), e))
	assert.Equal(t, "exam-9", e.ID)
	assert.True(t, e.CreatedAt.Equal(at))

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	err := NewRepository(db2).Create(context.Background(), &Exam{Title: "x"})
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "exams: Create")
}

func TestExamUpdate_ReturnsRowNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(examUpdateCols, [][]driver.Value{examUpdateRow("exam-1")})
	e, err := NewRepository(db).Update(context.Background(), "exam-1", UpdateExamInput{Title: "Fire Safety", TimeLimitMinutes: 30, PassingScorePct: 70})
	require.NoError(t, err)
	assert.Equal(t, "Fire Safety", e.Title)
	assert.Equal(t, 2, e.MaxAttempts)

	db2, f2 := newFakeDB(t)
	f2.queue(examUpdateCols, nil)
	_, err = NewRepository(db2).Update(context.Background(), "missing", UpdateExamInput{})
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).Update(context.Background(), "exam-1", UpdateExamInput{})
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "exams: Update")
}

func TestExamRulesForExam_ListsAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id", "exam_id", "section_id", "mode", "category_id", "tag_ids", "difficulty", "count", "sort_order"},
		[][]driver.Value{
			{"rule-1", "exam-1", nil, "random", nil, []byte("[]"), nil, int64(3), int64(1)},
			{"rule-2", "exam-1", nil, "manual", nil, []byte("[]"), nil, int64(2), int64(2)},
		})
	rules, err := NewRepository(db).ListRulesForExam(context.Background(), "exam-1")
	require.NoError(t, err)
	require.Len(t, rules, 2)
	assert.Equal(t, "manual", rules[1].Mode)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).ListRulesForExam(context.Background(), "exam-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "ListRulesForExam")
}

func TestExamCountAvailableForManualRule_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(4)}})
	n, err := NewRepository(db).CountAvailableForManualRule(context.Background(), "rule-2")
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).CountAvailableForManualRule(context.Background(), "rule-2")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CountAvailableForManualRule")
}

func TestExamSetManualQuestions_ReplacesListAndWrapsDeleteError(t *testing.T) {
	db, f := newFakeDB(t)
	err := NewRepository(db).SetManualQuestions(context.Background(), "rule-2", []ManualQuestionInput{
		{QuestionID: "q-1", SortOrder: 1}, {QuestionID: "q-2", SortOrder: 2},
	})
	require.NoError(t, err)
	// One DELETE for the old list, then one INSERT per question.
	require.Len(t, f.queries, 3)
	assert.Contains(t, f.queries[0], "DELETE FROM exam_manual_questions")
	assert.Contains(t, f.queries[1], "INSERT INTO exam_manual_questions")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err = NewRepository(db2).SetManualQuestions(context.Background(), "rule-2", []ManualQuestionInput{{QuestionID: "q-1"}})
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "SetManualQuestions: delete")
}
