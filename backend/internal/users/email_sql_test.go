package users

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
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
func (c recConn) Begin() (driver.Tx, error)             { return nil, errors.New("no tx") }
func (s recStmt) Close() error                          { return nil }
func (s recStmt) NumInput() int                         { return -1 }
func (s recStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}
func (s recStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.q)
	s.f.args = append(s.f.args, append([]driver.Value(nil), args...))
	return &recRows{cols: []string{"exists"}, data: [][]driver.Value{{s.f.existsResult}}}, nil
}
func (r *recRows) Columns() []string { return r.cols }
func (r *recRows) Close() error      { return nil }
func (r *recRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return errors.New("EOF")
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

	_, err := repo.Create(context.Background(), "john.doe@corp.com", "John", "hash", nil, "role")

	assert.ErrorIs(t, err, ErrDuplicateEmail)
	require.Len(t, f.queries, 1, "no INSERT may be issued when a case-insensitive twin exists")
	assert.Equal(t, "SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = lower($1))", strings.TrimSpace(f.queries[0]))
	assert.Equal(t, "john.doe@corp.com", f.args[0][0])
}
