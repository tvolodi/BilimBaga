package tags

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (follow-up #317) for the tags repository: list, get, create,
// update, delete and the question-reference probes. Each path gets its success,
// not-found, duplicate or wrapped-error case where the fake can express it.

var tagCols = []string{"id", "name", "created_at", "usage_count"}

func TestGetAll_WithAndWithoutQuestionTags(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{true}})
	f.queue(tagCols, [][]driver.Value{{"t1", "Safety", at, int64(3)}})
	list, err := NewRepository(db).GetAll(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Safety", list[0].Name)
	assert.Equal(t, 3, list[0].UsageCount)
	assert.Contains(t, f.queries[1], "LEFT JOIN")

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"exists"}, [][]driver.Value{{false}})
	f2.queue(tagCols, [][]driver.Value{{"t2", "Fire", at, int64(0)}})
	list2, err := NewRepository(db2).GetAll(context.Background())
	require.NoError(t, err)
	require.Len(t, list2, 1)
	assert.Equal(t, 0, list2[0].UsageCount)
	assert.NotContains(t, f2.queries[1], "LEFT JOIN")
}

func TestGetAll_ProbeErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, err := NewRepository(db).GetAll(context.Background())
	assert.ErrorIs(t, err, f.qErr)
	assert.ErrorContains(t, err, "GetAll: probe")
}

func TestGetByID_FoundNotFoundAndError(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue(tagCols, [][]driver.Value{{"t1", "Safety", at, int64(0)}})
	tg, err := NewRepository(db).GetByID(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "Safety", tg.Name)

	db2, f2 := newFakeDB(t)
	f2.queue(tagCols, nil)
	_, err = NewRepository(db2).GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetByID(context.Background(), "t1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetByID")
}

func TestCreate_InsertsDuplicateAndError(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue(tagCols, [][]driver.Value{{"t1", "Safety", at, int64(0)}})
	tg, err := NewRepository(db).Create(context.Background(), "Safety")
	require.NoError(t, err)
	assert.Equal(t, "t1", tg.ID)

	db2, f2 := newFakeDB(t)
	f2.qErr = &pq.Error{Code: "23505"}
	_, err = NewRepository(db2).Create(context.Background(), "Safety")
	assert.ErrorIs(t, err, ErrDuplicate)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).Create(context.Background(), "Safety")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "tags.Create")
}

func TestUpdate_UpdatesNotFoundDuplicateAndError(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue(tagCols, [][]driver.Value{{"t1", "Fire", at, int64(0)}})
	tg, err := NewRepository(db).Update(context.Background(), "t1", "Fire")
	require.NoError(t, err)
	assert.Equal(t, "Fire", tg.Name)

	db2, f2 := newFakeDB(t)
	f2.queue(tagCols, nil)
	_, err = NewRepository(db2).Update(context.Background(), "missing", "x")
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = &pq.Error{Code: "23505"}
	_, err = NewRepository(db3).Update(context.Background(), "t1", "Fire")
	assert.ErrorIs(t, err, ErrDuplicate)

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).Update(context.Background(), "t1", "Fire")
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "tags.Update")
}

func TestDelete_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).Delete(context.Background(), "t1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM tags")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).Delete(context.Background(), "t1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "tags.Delete")
}

func TestHasReferences_ProbeAndCount(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{false}})
	ok, err := NewRepository(db).HasReferences(context.Background(), "t1")
	require.NoError(t, err)
	assert.False(t, ok, "no question_tags table means no references")

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"exists"}, [][]driver.Value{{true}})
	f2.queue([]string{"count"}, [][]driver.Value{{int64(2)}})
	ok, err = NewRepository(db2).HasReferences(context.Background(), "t1")
	require.NoError(t, err)
	assert.True(t, ok)

	db3, f3 := newFakeDB(t)
	f3.queue([]string{"exists"}, [][]driver.Value{{true}})
	f3.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
	ok, err = NewRepository(db3).HasReferences(context.Background(), "t1")
	require.NoError(t, err)
	assert.False(t, ok)

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).HasReferences(context.Background(), "t1")
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "HasReferences: probe")
}
