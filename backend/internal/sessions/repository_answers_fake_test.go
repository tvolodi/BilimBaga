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

// Fake-DB coverage (follow-up #311) for sessions paths that do not open a
// transaction: grading detail, answer upsert and reads, option and question
// lookups. The transactional paths are covered separately.

var gradingHeaderCols = []string{"session_id", "employee_name", "exam_title", "submitted_at"}
var gradingQuestionCols = []string{"question_id", "stem", "text_answer", "grading_status", "current_score_pct", "manual_feedback"}

func TestGetGradingDetail_HeaderAndShortTextQuestions(t *testing.T) {
	db, f := newFakeDB(t)
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	f.queue(gradingHeaderCols, [][]driver.Value{{"sess-1", "Ann", "Fire Safety", at}})
	f.queue(gradingQuestionCols, [][]driver.Value{{"q1", "Explain", "my answer", "pending_manual", nil, nil}})
	res, err := NewRepository(db, nil).GetGradingDetail(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "sess-1", res.SessionID)
	assert.Equal(t, "Ann", res.EmployeeName)
	assert.Equal(t, "Fire Safety", res.ExamTitle)
	require.NotNil(t, res.SubmittedAt)
	require.Len(t, res.Questions, 1)
	assert.Equal(t, "q1", res.Questions[0].QuestionID)
	assert.Equal(t, "pending_manual", res.Questions[0].GradingStatus)
	assert.Nil(t, res.Questions[0].CurrentScorePct)
}

func TestGetGradingDetail_NoShortTextQuestionsIsEmptyNonNil(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(gradingHeaderCols, [][]driver.Value{{"sess-1", "Ann", "Fire Safety", nil}})
	f.queue(gradingQuestionCols, nil)
	res, err := NewRepository(db, nil).GetGradingDetail(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.NotNil(t, res.Questions)
	assert.Empty(t, res.Questions)
}

func TestGetGradingDetail_NotFoundAndHeaderError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(gradingHeaderCols, nil)
	_, err := NewRepository(db, nil).GetGradingDetail(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrSessionNotFound)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetGradingDetail(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetGradingDetail")
}

func TestUpsertAnswer_ReturnsSavedAtAndError(t *testing.T) {
	db, f := newFakeDB(t)
	at := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	f.queue([]string{"saved_at"}, [][]driver.Value{{at}})
	got, err := NewRepository(db, nil).UpsertAnswer(context.Background(), upsertAnswerInput{
		SessionID: "sess-1", QuestionID: "q1", SelectedOptionIDs: []string{"o1"}, TimeSpentSeconds: 12,
	})
	require.NoError(t, err)
	assert.True(t, got.Equal(at))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "ON CONFLICT (session_id, question_id)")

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).UpsertAnswer(context.Background(), upsertAnswerInput{SessionID: "sess-1", QuestionID: "q1"})
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "UpsertAnswer")
}

func TestGetSessionAnswers_MapsByQuestion(t *testing.T) {
	db, f := newFakeDB(t)
	at := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	f.queue([]string{"question_id", "selected_option_ids", "text_answer", "time_spent_seconds", "saved_at"},
		[][]driver.Value{{"q1", []byte(`["o1"]`), "txt", int64(12), at}})
	m, err := NewRepository(db, nil).GetSessionAnswers(context.Background(), "sess-1")
	require.NoError(t, err)
	require.Contains(t, m, "q1")
	assert.Equal(t, 12, m["q1"].TimeSpentSeconds)
	require.NotNil(t, m["q1"].TextAnswer)
	assert.Equal(t, "txt", *m["q1"].TextAnswer)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetSessionAnswers(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetSessionAnswers")
}

func TestGetValidOptionIDs_SetAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id"}, [][]driver.Value{{"o1"}, {"o2"}})
	set, err := NewRepository(db, nil).GetValidOptionIDs(context.Background(), "q1")
	require.NoError(t, err)
	assert.Len(t, set, 2)
	assert.Contains(t, set, "o2")

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetValidOptionIDs(context.Background(), "q1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetValidOptionIDs")
}

func TestGetOptionTexts_EmptyInputNoQuery(t *testing.T) {
	db, f := newFakeDB(t)
	m, err := NewRepository(db, nil).GetOptionTexts(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, m)
	assert.Empty(t, f.queries)
}

func TestGetOptionTexts_MapsTextAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"option_id", "text"}, [][]driver.Value{{"o1", "Yes"}, {"o2", "No"}})
	m, err := NewRepository(db, nil).GetOptionTexts(context.Background(), []string{"o1", "o2"})
	require.NoError(t, err)
	assert.Equal(t, "Yes", m["o1"])
	assert.Equal(t, "No", m["o2"])

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).GetOptionTexts(context.Background(), []string{"o1"})
	assert.ErrorIs(t, err, f2.qErr)
}

func TestGetQuestionDetails_EmptyInputNoQueryAndMapping(t *testing.T) {
	db, f := newFakeDB(t)
	m, err := NewRepository(db, nil).GetQuestionDetails(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, m)
	assert.Empty(t, f.queries)

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"id", "stem", "type"}, [][]driver.Value{{"q1", "Explain", "shorttext"}})
	m2, err := NewRepository(db2, nil).GetQuestionDetails(context.Background(), []string{"q1"})
	require.NoError(t, err)
	require.Contains(t, m2, "q1")
	assert.Equal(t, "shorttext", m2["q1"].Type)
}
