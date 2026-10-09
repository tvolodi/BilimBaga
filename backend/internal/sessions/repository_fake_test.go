package sessions

import (
	"context"
	"database/sql/driver"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (tech debt #299) for small read and write queries of the
// sessions repository. Each query gets its success, not-found or wrapped-error path.

func TestGetExamTitleByID_Found(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title"}, [][]driver.Value{{"Fire Safety"}})
	title, err := NewRepository(db, nil).GetExamTitleByID(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "Fire Safety", title)
}

func TestGetExamTitleByID_NotFound(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title"}, nil)
	_, err := NewRepository(db, nil).GetExamTitleByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrExamNotFound)
}

func TestGetExamTitleByID_QueryErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, err := NewRepository(db, nil).GetExamTitleByID(context.Background(), "exam-1")
	assert.ErrorIs(t, err, f.qErr)
	assert.ErrorContains(t, err, "GetExamTitleByID")
}

func TestHasOpenSession_TrueAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{true}})
	ok, err := NewRepository(db, nil).HasOpenSession(context.Background(), "exam-1", "user-1")
	require.NoError(t, err)
	assert.True(t, ok)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).HasOpenSession(context.Background(), "exam-1", "user-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "HasOpenSession")
}

func TestCountFinishedSessions_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(3)}})
	n, err := NewRepository(db, nil).CountFinishedSessions(context.Background(), "exam-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).CountFinishedSessions(context.Background(), "exam-1", "user-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CountFinishedSessions")
}

func TestIsAssigned_TrueAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"exists"}, [][]driver.Value{{true}})
	ok, err := NewRepository(db, nil).IsAssigned(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.True(t, ok)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).IsAssigned(context.Background(), "exam-1", "user-1", "")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "IsAssigned")
}

func TestInsertTabSwitchEvent_OKAndError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db, nil).InsertTabSwitchEvent(context.Background(), "sess-1", "tab_switch", "warned"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "INSERT INTO tab_switch_events")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2, nil).InsertTabSwitchEvent(context.Background(), "sess-1", "tab_switch", "warned")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "InsertTabSwitchEvent")
}

func TestCountTabSwitchEvents_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(2)}})
	n, err := NewRepository(db, nil).CountTabSwitchEvents(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).CountTabSwitchEvents(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CountTabSwitchEvents")
}

func TestCountAnsweredForSession_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(5)}})
	n, err := NewRepository(db, nil).CountAnsweredForSession(context.Background(), "sess-1")
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2, nil).CountAnsweredForSession(context.Background(), "sess-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CountAnsweredForSession")
}

// Grading engine constructors (tech debt #299): both return a usable engine.
func TestNewGradingEngines_NonNil(t *testing.T) {
	assert.NotNil(t, NewGradingEngine())
	assert.NotNil(t, NewAIGradingEngine(nil, "model", nil, slog.Default()))
}
