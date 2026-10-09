package exams

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (follow-up #315) for small exams repository queries and the
// unique-violation helper. Each query gets its success, not-found or wrapped-error
// path where the fake can express it.

func TestIsUniqueViolation_OnlyCode23505(t *testing.T) {
	assert.True(t, isUniqueViolation(&pq.Error{Code: "23505"}))
	assert.True(t, isUniqueViolation(errors.Join(errors.New("ctx"), &pq.Error{Code: "23505"})))
	assert.False(t, isUniqueViolation(&pq.Error{Code: "23503"}))
	assert.False(t, isUniqueViolation(errors.New("plain")))
	assert.False(t, isUniqueViolation(nil))
}

func TestGetByID_FoundNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id", "title"}, [][]driver.Value{{"exam-1", "Fire Safety"}})
	e, err := NewRepository(db).GetByID(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "exam-1", e.ID)
	assert.Equal(t, "Fire Safety", e.Title)

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"id", "title"}, nil)
	_, err = NewRepository(db2).GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetByID(context.Background(), "exam-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetByID")
}

func TestUpdateStatus_UpdatesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).UpdateStatus(context.Background(), "exam-1", "active"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "UPDATE exams SET status")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).UpdateStatus(context.Background(), "exam-1", "active")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "UpdateStatus")
}

func TestDeleteByID_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).DeleteByID(context.Background(), "exam-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM exams")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).DeleteByID(context.Background(), "exam-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "DeleteByID")
}

func TestGetRuleByID_FoundNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id", "exam_id", "section_id", "mode", "category_id", "tag_ids", "difficulty", "count", "sort_order"},
		[][]driver.Value{{"rule-1", "exam-1", nil, "random", nil, []byte("[]"), nil, int64(5), int64(1)}})
	rule, err := NewRepository(db).GetRuleByID(context.Background(), "rule-1")
	require.NoError(t, err)
	assert.Equal(t, "random", rule.Mode)
	assert.Equal(t, 5, rule.Count)

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"id"}, nil)
	_, err = NewRepository(db2).GetRuleByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetRuleByID(context.Background(), "rule-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetRuleByID")
}

func TestCountActiveSessionsForExam_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(4)}})
	n, err := NewRepository(db).CountActiveSessionsForExam(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).CountActiveSessionsForExam(context.Background(), "exam-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CountActiveSessionsForExam")
}

func TestUserDepartmentID_ValueNullUnknownAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"department_id"}, [][]driver.Value{{"dept-1"}})
	dept, err := NewRepository(db).UserDepartmentID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "dept-1", dept)

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"department_id"}, [][]driver.Value{{nil}})
	dept, err = NewRepository(db2).UserDepartmentID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "", dept)

	db3, f3 := newFakeDB(t)
	f3.queue([]string{"department_id"}, nil)
	dept, err = NewRepository(db3).UserDepartmentID(context.Background(), "ghost")
	require.NoError(t, err)
	assert.Equal(t, "", dept)

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).UserDepartmentID(context.Background(), "user-1")
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "UserDepartmentID")
}
