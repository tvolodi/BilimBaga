package roles

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (task #347) for the roles repository: list with permission
// attachment, get, permission lookups, create and update (transactional), delete
// with the in-use mapping, and the user count. Each path gets success and
// error cases; the fake always reports one affected row, so the not-found
// branches of Update and Delete are not reachable here.

var roleCols = []string{"id", "name", "description", "is_system", "created_at", "user_count"}
var rolePermCols = []string{"role_id", "resource", "action"}

func roleRow(id, name string, system bool) []driver.Value {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return []driver.Value{id, name, "desc", system, at, int64(2)}
}

func TestRolesList_AttachesPermissions(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(roleCols, [][]driver.Value{roleRow("r-1", "admin", true), roleRow("r-2", "custom", false)})
	f.queue(rolePermCols, [][]driver.Value{{"r-1", "users", "read"}, {"r-1", "exams", "write"}})
	list, err := NewRepository(db).List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, []string{"users:read", "exams:write"}, list[0].Permissions)
	assert.Equal(t, []string{}, list[1].Permissions, "a role without permissions gets an empty, non-nil list")

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).List(context.Background())
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "roles.List")
}

func TestRolesGetByID_FoundNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(roleCols, [][]driver.Value{roleRow("r-1", "admin", true)})
	f.queue(rolePermCols, [][]driver.Value{{"r-1", "users", "read"}})
	role, err := NewRepository(db).GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
	assert.Equal(t, []string{"users:read"}, role.Permissions)

	db2, f2 := newFakeDB(t)
	f2.queue(roleCols, nil)
	_, err = NewRepository(db2).GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetByID(context.Background(), "r-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "roles.GetByID")
}

func TestRolesPermissions_ListAndByIDs(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id", "resource", "action"}, [][]driver.Value{{"p-1", "users", "read"}, {"p-2", "exams", "write"}})
	all, err := NewRepository(db).ListPermissions(context.Background())
	require.NoError(t, err)
	assert.Len(t, all, 2)

	db2, f2 := newFakeDB(t)
	empty, err := NewRepository(db2).PermissionsByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.Empty(t, f2.queries, "no ids means no query")

	db3, f3 := newFakeDB(t)
	f3.queue([]string{"id", "resource", "action"}, [][]driver.Value{{"p-1", "users", "read"}})
	some, err := NewRepository(db3).PermissionsByIDs(context.Background(), []string{"p-1"})
	require.NoError(t, err)
	assert.Len(t, some, 1)

	db4, f4 := newFakeDB(t)
	f4.qErr = errors.New("boom")
	_, err = NewRepository(db4).ListPermissions(context.Background())
	assert.ErrorIs(t, err, f4.qErr)
	assert.ErrorContains(t, err, "roles.ListPermissions")

	db5, f5 := newFakeDB(t)
	f5.qErr = errors.New("boom")
	_, err = NewRepository(db5).PermissionsByIDs(context.Background(), []string{"p-1"})
	assert.ErrorIs(t, err, f5.qErr)
	assert.ErrorContains(t, err, "roles.PermissionsByIDs")
}

func TestRolesCreate_InsertsPermissionsAndMapsNameTaken(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id"}, [][]driver.Value{{"r-9"}})
	id, err := NewRepository(db).Create(context.Background(), "auditor", "reads", []string{"p-1"})
	require.NoError(t, err)
	assert.Equal(t, "r-9", id)
	assert.Equal(t, 1, countRoleQueries(f, "DELETE FROM role_permissions"))
	assert.Equal(t, 1, countRoleQueries(f, "INSERT INTO role_permissions"))

	db2, f2 := newFakeDB(t)
	f2.qErr = &pq.Error{Code: "23505"}
	_, err = NewRepository(db2).Create(context.Background(), "auditor", "", nil)
	assert.ErrorIs(t, err, ErrNameTaken)

	db3, f3 := newFakeDB(t)
	f3.queue([]string{"id"}, [][]driver.Value{{"r-9"}})
	f3.xErr = errors.New("boom")
	_, err = NewRepository(db3).Create(context.Background(), "auditor", "", []string{"p-1"})
	assert.ErrorIs(t, err, f3.xErr)
	assert.ErrorContains(t, err, "clear permissions")
}

func TestRolesUpdate_RewritesPermissionsAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).Update(context.Background(), "r-9", "new desc", []string{"p-1", "p-2"}))
	assert.Equal(t, 1, countRoleQueries(f, "UPDATE roles SET description"))
	assert.Equal(t, 1, countRoleQueries(f, "INSERT INTO role_permissions"))

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).Update(context.Background(), "r-9", "x", nil)
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "roles.Update")
}

func TestRolesDelete_DeletesInUseAndError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).Delete(context.Background(), "r-9"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM roles")

	db2, f2 := newFakeDB(t)
	f2.xErr = &pq.Error{Code: "23503"}
	err := NewRepository(db2).Delete(context.Background(), "r-9")
	var inUse *InUseError
	assert.True(t, errors.As(err, &inUse), "a foreign-key violation maps to InUseError")

	db3, f3 := newFakeDB(t)
	f3.xErr = errors.New("boom")
	err = NewRepository(db3).Delete(context.Background(), "r-9")
	assert.ErrorIs(t, err, f3.xErr)
	assert.ErrorContains(t, err, "roles.Delete")
}

func TestRolesCountUsers_CountAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(4)}})
	n, err := NewRepository(db).CountUsers(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).CountUsers(context.Background(), "r-1")
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "roles.CountUsers")
}

func countRoleQueries(f *fakeDB, substr string) int {
	n := 0
	for _, q := range f.queries {
		if strings.Contains(q, substr) {
			n++
		}
	}
	return n
}

