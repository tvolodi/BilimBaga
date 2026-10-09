package deptscope

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeKey_ConstantsAndStability(t *testing.T) {
	assert.Equal(t, ScopeKeyAll, ScopeKey(Scope{}, []string{"a", "b"}), "unrestricted ignores ids")
	assert.Equal(t, ScopeKeyNone, ScopeKey(Scope{Restricted: true}, nil))
	assert.Equal(t, ScopeKeyNone, ScopeKey(Scope{Restricted: true}, []string{}))

	r := Scope{Restricted: true}
	k1 := ScopeKey(r, []string{"b", "a", "c"})
	k2 := ScopeKey(r, []string{"c", "a", "b", "a"})
	assert.Equal(t, k1, k2, "order and duplicates do not matter")
	assert.Len(t, k1, 64)
	assert.NotEqual(t, ScopeKeyAll, k1)
	assert.NotEqual(t, ScopeKeyNone, k1)
	assert.NotEqual(t, k1, ScopeKey(r, []string{"a", "b"}), "different sets differ")
	assert.NotEqual(t, ScopeKey(r, []string{"ab", "c"}), ScopeKey(r, []string{"a", "bc"}), "no join ambiguity")
}

func TestScopeKey_DoesNotMutateInput(t *testing.T) {
	ids := []string{"b", "a"}
	ScopeKey(Scope{Restricted: true}, ids)
	assert.Equal(t, []string{"b", "a"}, ids)
}

func TestSubtreeIDs(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id"}, [][]driver.Value{{deptA}, {target}})
	ids, err := SubtreeIDs(context.Background(), db, FromContext(principal("department_admin", deptA)))
	require.NoError(t, err)
	assert.Equal(t, []string{deptA, target}, ids)
	assert.Contains(t, f.queries[0], "WITH RECURSIVE sc_t")
	assert.Equal(t, []driver.Value{deptA}, f.args[0])

	// No department: sentinel root, matches nothing.
	db, f = newFakeDB(t)
	ids, err = SubtreeIDs(context.Background(), db, FromContext(principal("examiner", "")))
	require.NoError(t, err)
	assert.Empty(t, ids)
	assert.Equal(t, []driver.Value{noDepartment}, f.args[0])

	// Unrestricted: no query at all.
	db, f = newFakeDB(t)
	ids, err = SubtreeIDs(context.Background(), db, Scope{})
	require.NoError(t, err)
	assert.Nil(t, ids)
	assert.Empty(t, f.queries)

	db, f = newFakeDB(t)
	f.qErr = errors.New("boom")
	_, err = SubtreeIDs(context.Background(), db, Scope{Restricted: true, DepartmentID: deptA})
	require.Error(t, err)
}
