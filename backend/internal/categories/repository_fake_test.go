package categories

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

// Fake-DB coverage (follow-up #335) for the categories repository: list, get,
// create, update, delete, and the child and question reference checks. Each path
// gets its success, not-found, parent-missing or wrapped-error case where the
// fake can express it. The fake always reports one affected row, so the
// not-found branch of Delete is not reachable here.

var categoryCols = []string{"id", "name", "parent_id", "track", "sort_order", "created_at", "updated_at"}

func categoryRow(id, name string) []driver.Value {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return []driver.Value{id, name, nil, nil, int64(1), at, at}
}

func TestCategoryGetAll_RowsAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(categoryCols, [][]driver.Value{categoryRow("c-1", "Safety"), categoryRow("c-2", "Fire")})
	list, err := NewRepository(db).GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "Safety", list[0].Name)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).GetAll(context.Background())
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "categories.GetAll")
}

func TestCategoryGetByID_FoundNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(categoryCols, [][]driver.Value{categoryRow("c-1", "Safety")})
	c, err := NewRepository(db).GetByID(context.Background(), "c-1")
	require.NoError(t, err)
	assert.Equal(t, "c-1", c.ID)

	db2, f2 := newFakeDB(t)
	f2.queue(categoryCols, nil)
	_, err = NewRepository(db2).GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetByID(context.Background(), "c-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "categories.GetByID")
}

func TestCategoryCreate_ReturnsRowParentMissingAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(categoryCols, [][]driver.Value{categoryRow("c-1", "Safety")})
	c, err := NewRepository(db).Create(context.Background(), "Safety", nil, nil, 1)
	require.NoError(t, err)
	assert.Equal(t, "Safety", c.Name)

	db2, f2 := newFakeDB(t)
	f2.qErr = &pq.Error{Code: "23503"}
	_, err = NewRepository(db2).Create(context.Background(), "Child", strp("missing"), nil, 1)
	assert.ErrorIs(t, err, ErrParentNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).Create(context.Background(), "Safety", nil, nil, 1)
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "categories.Create")
}

func TestCategoryUpdate_UpdatesNotFoundParentMissingAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(categoryCols, [][]driver.Value{categoryRow("c-1", "Fire")})
	out, err := NewRepository(db).Update(context.Background(), Category{ID: "c-1", Name: "Fire", SortOrder: 1})
	require.NoError(t, err)
	assert.Equal(t, "Fire", out.Name)

	db2, f2 := newFakeDB(t)
	f2.queue(categoryCols, nil)
	_, err = NewRepository(db2).Update(context.Background(), Category{ID: "missing"})
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = &pq.Error{Code: "23503"}
	_, err = NewRepository(db3).Update(context.Background(), Category{ID: "c-1", ParentID: strp("missing")})
	assert.ErrorIs(t, err, ErrParentNotFound)

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).Update(context.Background(), Category{ID: "c-1"})
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "categories.Update")
}

func TestCategoryDelete_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).Delete(context.Background(), "c-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM categories")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).Delete(context.Background(), "c-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "categories.Delete")
}

func TestCategoryHasChildren_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(2)}})
	ok, err := NewRepository(db).HasChildren(context.Background(), "c-1")
	require.NoError(t, err)
	assert.True(t, ok)

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
	ok, err = NewRepository(db2).HasChildren(context.Background(), "c-1")
	require.NoError(t, err)
	assert.False(t, ok)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).HasChildren(context.Background(), "c-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "categories.HasChildren")
}

func TestCategoryHasQuestions_ProbeAndCount(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{false}})
	ok, err := NewRepository(db).HasQuestions(context.Background(), "c-1")
	require.NoError(t, err)
	assert.False(t, ok, "no questions table means no references")

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"exists"}, [][]driver.Value{{true}})
	f2.queue([]string{"count"}, [][]driver.Value{{int64(3)}})
	ok, err = NewRepository(db2).HasQuestions(context.Background(), "c-1")
	require.NoError(t, err)
	assert.True(t, ok)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).HasQuestions(context.Background(), "c-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "HasQuestions: probe")
}

func strp(s string) *string { return &s }
