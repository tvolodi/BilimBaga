package users

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recDB is a minimal database/sql driver recording every statement and its args.
// existsResult is returned for the single-column EXISTS probe.
type recDB struct {
	mu           sync.Mutex
	queries      []string
	args         [][]driver.Value
	existsResult bool
	idRows       []string // when non-nil, queries return these ids in a single "id" column
}

type recConn struct{ f *recDB }
type recStmt struct {
	f *recDB
	q string
}
type recRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (c recConn) Prepare(q string) (driver.Stmt, error) { return recStmt{c.f, q}, nil }
func (c recConn) Close() error                          { return nil }
func (c recConn) Begin() (driver.Tx, error)             { return recTx{}, nil }

// recTx is a no-op transaction, so repository methods that run inside one can be recorded.
type recTx struct{}

func (recTx) Commit() error   { return nil }
func (recTx) Rollback() error { return nil }
func (s recStmt) Close() error                          { return nil }
func (s recStmt) NumInput() int                         { return -1 }
func (s recStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.q)
	s.f.args = append(s.f.args, append([]driver.Value(nil), args...))
	return driver.RowsAffected(1), nil
}
func (s recStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.q)
	s.f.args = append(s.f.args, append([]driver.Value(nil), args...))
	if s.f.idRows != nil {
		data := [][]driver.Value{}
		for _, id := range s.f.idRows {
			data = append(data, []driver.Value{id})
		}
		return &recRows{cols: []string{"id"}, data: data}, nil
	}
	return &recRows{cols: []string{"exists"}, data: [][]driver.Value{{s.f.existsResult}}}, nil
}
func (r *recRows) Columns() []string { return r.cols }
func (r *recRows) Close() error      { return nil }
func (r *recRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

type recConnector struct{ f *recDB }

func (c recConnector) Connect(context.Context) (driver.Conn, error) { return recConn(c), nil }
func (c recConnector) Driver() driver.Driver                        { return nil }

func TestRepositoryCreate_DuplicateProbeIsCaseInsensitive(t *testing.T) {
	f := &recDB{existsResult: true}
	db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), "john.doe@corp.com", "John", "hash", nil, "11111111-1111-4111-8111-111111111111")

	assert.ErrorIs(t, err, ErrDuplicateEmail)
	require.Len(t, f.queries, 1, "no INSERT may be issued when a case-insensitive twin exists")
	assert.Equal(t, "SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = lower($1))", strings.TrimSpace(f.queries[0]))
	assert.Equal(t, "john.doe@corp.com", f.args[0][0])
}

// FR-BB116 AC-1 / AC-2: the preferred_locale column is selected by every user read and written
// by SetPreferredLocale, with a nil locale bound as SQL NULL (clears the preference).
func TestRepositoryPreferredLocaleSQL(t *testing.T) {
	const uid = "11111111-1111-4111-8111-111111111111"

	t.Run("GetByID selects preferred_locale", func(t *testing.T) {
		f := &recDB{}
		db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
		t.Cleanup(func() { _ = db.Close() })

		_, _ = NewRepository(db).GetByID(context.Background(), uid) // the fake rows never scan; only the SQL matters

		require.Len(t, f.queries, 1)
		assert.Contains(t, f.queries[0], "u.preferred_locale")
	})

	t.Run("List selects preferred_locale", func(t *testing.T) {
		f := &recDB{idRows: []string{"1"}} // one row: the COUNT probe scans it as int, the list as a user id
		db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
		t.Cleanup(func() { _ = db.Close() })

		_, _, err := NewRepository(db).List(context.Background(), ListFilters{Page: 1, PerPage: 20}, nil)
		require.NoError(t, err)

		var listQ string
		for _, q := range f.queries {
			if strings.Contains(q, "ORDER BY u.created_at DESC") {
				listQ = q
			}
		}
		require.NotEmpty(t, listQ, "list query must be issued")
		assert.Contains(t, listQ, "u.preferred_locale")
	})

	t.Run("SetPreferredLocale writes the column; nil binds NULL", func(t *testing.T) {
		f := &recDB{}
		db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
		t.Cleanup(func() { _ = db.Close() })
		repo := NewRepository(db)

		ru := "ru"
		require.NoError(t, repo.SetPreferredLocale(context.Background(), uid, &ru))
		require.NoError(t, repo.SetPreferredLocale(context.Background(), uid, nil))

		require.Len(t, f.queries, 2)
		assert.Equal(t, "UPDATE users SET preferred_locale = $1, updated_at = now() WHERE id = $2", strings.TrimSpace(f.queries[0]))
		assert.Equal(t, []driver.Value{"ru", uid}, f.args[0])
		assert.Equal(t, []driver.Value{nil, uid}, f.args[1])
	})
}
