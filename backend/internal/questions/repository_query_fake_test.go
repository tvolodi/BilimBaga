package questions

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (follow-up #343) for the query-only questions repository paths
// left at 0% after #331: translations, answer options and answer translations,
// version chain and question delete. None of these open a transaction.

var translationCols = []string{"question_id", "locale", "stem", "explanation", "updated_at"}
var answerOptionCols = []string{"id", "question_id", "sort_order", "is_correct", "likert_weight", "likert_polarity", "created_at"}

func TestQuestionTranslation_CreateAndGet(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	db, f := newFakeDB(t)
	f.queue(translationCols, [][]driver.Value{{"q-1", "en", "What?", nil, at}})
	tr := &QuestionTranslation{QuestionID: "q-1", Locale: "en", Stem: "What?"}
	require.NoError(t, NewRepository(db).CreateTranslation(context.Background(), tr))
	assert.Equal(t, "What?", tr.Stem)

	db2, f2 := newFakeDB(t)
	f2.queue(translationCols, [][]driver.Value{{"q-1", "en", "What?", nil, at}})
	got, err := NewRepository(db2).GetTranslation(context.Background(), "q-1", "en")
	require.NoError(t, err)
	assert.Equal(t, "en", got.Locale)

	db3, f3 := newFakeDB(t)
	f3.queue(translationCols, nil)
	_, err = NewRepository(db3).GetTranslation(context.Background(), "q-1", "kk")
	assert.ErrorIs(t, err, ErrNotFound)

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).GetTranslation(context.Background(), "q-1", "en")
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "questions.GetTranslation")

	db5, f5 := newFakeDB(t)
	f5.qErr = errors.New("boom")
	err = NewRepository(db5).CreateTranslation(context.Background(), &QuestionTranslation{QuestionID: "q-1"})
	assert.ErrorIs(t, err, f5.qErr)
}

func TestAnswerOption_CreateAndList(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	db, f := newFakeDB(t)
	f.queue(answerOptionCols, [][]driver.Value{{"o-1", "q-1", int64(1), true, nil, nil, at}})
	opt := &AnswerOption{ID: "o-1", QuestionID: "q-1", SortOrder: 1, IsCorrect: true}
	require.NoError(t, NewRepository(db).CreateAnswerOption(context.Background(), opt))
	assert.True(t, opt.IsCorrect)

	db2, f2 := newFakeDB(t)
	f2.queue(answerOptionCols, [][]driver.Value{{"o-1", "q-1", int64(1), true, nil, nil, at}, {"o-2", "q-1", int64(2), false, nil, nil, at}})
	opts, err := NewRepository(db2).GetAnswerOptions(context.Background(), "q-1")
	require.NoError(t, err)
	require.Len(t, opts, 2)
	assert.Equal(t, "o-2", opts[1].ID)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetAnswerOptions(context.Background(), "q-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetAnswerOptions")

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	err = NewRepository(db4).CreateAnswerOption(context.Background(), &AnswerOption{ID: "o-1"})
	assert.ErrorIs(t, err, f4.qErr)
}

func TestAnswerTranslation_Create(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"option_id", "locale", "text"}, [][]driver.Value{{"o-1", "en", "Yes"}})
	at := &AnswerTranslation{OptionID: "o-1", Locale: "en", Text: "Yes"}
	require.NoError(t, NewRepository(db).CreateAnswerTranslation(context.Background(), at))
	assert.Equal(t, "Yes", at.Text)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	err := NewRepository(db2).CreateAnswerTranslation(context.Background(), &AnswerTranslation{OptionID: "o-1"})
	assert.ErrorIs(t, err, f2.qErr)
}

func TestQuestionDeleteByID_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).DeleteByID(context.Background(), "q-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM questions")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).DeleteByID(context.Background(), "q-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "questions.DeleteByID")
}

func TestQuestionGetVersionChain_RowsAndError(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue([]string{"id", "version", "status", "created_at", "created_by"},
		[][]driver.Value{{"q-1", int64(1), "archived", at, "user-1"}, {"q-2", int64(2), "active", at, "user-1"}})
	chain, err := NewRepository(db).GetVersionChain(context.Background(), "q-2")
	require.NoError(t, err)
	require.Len(t, chain, 2)
	assert.Equal(t, 1, chain[0].Version)
	assert.Equal(t, "active", chain[1].Status)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).GetVersionChain(context.Background(), "q-2")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "GetVersionChain")
}
