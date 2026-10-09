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

// Fake-DB coverage (follow-up #331) for the questions repository's query-only
// paths: create, get, list, update, and the question-tag operations. Each path
// gets its success, not-found or wrapped-error case.

var questionCols = []string{
	"id", "category_id", "difficulty", "type", "default_locale", "status", "created_by",
	"version", "parent_id", "auto_grade", "model_answer", "created_at", "updated_at",
}

func questionRow(id string) []driver.Value {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return []driver.Value{id, "cat-1", "easy", "single", "en", "active", "user-1",
		int64(1), nil, false, nil, at, at}
}

func TestQuestionCreate_ReturnsRowAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(questionCols, [][]driver.Value{questionRow("q-1")})
	q := &Question{CategoryID: "cat-1", Difficulty: "easy", Type: "single", DefaultLocale: "en", Status: "draft", CreatedBy: "user-1", Version: 1}
	require.NoError(t, NewRepository(db).Create(context.Background(), q))
	assert.Equal(t, "q-1", q.ID)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	err := NewRepository(db2).Create(context.Background(), &Question{})
	assert.ErrorIs(t, err, f2.qErr)
}

func TestQuestionGetByID_FoundNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(questionCols, [][]driver.Value{questionRow("q-1")})
	q, err := NewRepository(db).GetByID(context.Background(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, "single", q.Type)

	db2, f2 := newFakeDB(t)
	f2.queue(questionCols, nil)
	_, err = NewRepository(db2).GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrQuestionNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetByID(context.Background(), "q-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "questions.GetByID")
}

func TestQuestionListByCategory_RowsAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(questionCols, [][]driver.Value{questionRow("q-1"), questionRow("q-2")})
	list, err := NewRepository(db).ListByCategory(context.Background(), "cat-1")
	require.NoError(t, err)
	assert.Len(t, list, 2)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).ListByCategory(context.Background(), "cat-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "ListByCategory")
}

func TestQuestionUpdate_UpdatesNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(questionCols, [][]driver.Value{questionRow("q-1")})
	require.NoError(t, NewRepository(db).Update(context.Background(), &Question{ID: "q-1", CategoryID: "cat-1", Status: "active"}))

	db2, f2 := newFakeDB(t)
	f2.queue(questionCols, nil)
	err := NewRepository(db2).Update(context.Background(), &Question{ID: "missing"})
	assert.ErrorIs(t, err, ErrQuestionNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	err = NewRepository(db3).Update(context.Background(), &Question{ID: "q-1"})
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "questions.Update")
}

func TestQuestionTags_AddRemoveListAndExists(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).AddTag(context.Background(), "q-1", "tag-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "INSERT INTO question_tags")

	db2, f2 := newFakeDB(t)
	require.NoError(t, NewRepository(db2).RemoveTag(context.Background(), "q-1", "tag-1"))
	assert.Contains(t, f2.queries[0], "DELETE FROM question_tags")

	db3, f3 := newFakeDB(t)
	f3.queue([]string{"tag_id"}, [][]driver.Value{{"tag-1"}, {"tag-2"}})
	ids, err := NewRepository(db3).GetTags(context.Background(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"tag-1", "tag-2"}, ids)

	db4, f4 := newFakeDB(t)
	f4.queue([]string{"exists"}, [][]driver.Value{{true}})
	ok, err := NewRepository(db4).TagExists(context.Background(), "tag-1")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestQuestionTags_ErrorsWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.xErr = errors.New("boom")
	err := NewRepository(db).AddTag(context.Background(), "q-1", "tag-1")
	assert.ErrorIs(t, err, f.xErr)
	assert.ErrorContains(t, err, "AddTag")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err = NewRepository(db2).RemoveTag(context.Background(), "q-1", "tag-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "RemoveTag")

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetTags(context.Background(), "q-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetTags")

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).TagExists(context.Background(), "tag-1")
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "TagExists")
}
