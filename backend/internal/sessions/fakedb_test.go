package sessions

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
)

// fakeDB is a minimal in-process database/sql driver: queries return canned
// rows (consumed in order) and every call is recorded.
type fakeDB struct {
	mu      sync.Mutex
	cols    [][]string
	rows    [][][]driver.Value
	qErr    error
	xErr    error
	queries []string
	args    [][]driver.Value
}

type fakeConn struct{ f *fakeDB }
type fakeStmt struct {
	f *fakeDB
	q string
}
type fakeRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (c fakeConn) Prepare(q string) (driver.Stmt, error) { return fakeStmt{c.f, q}, nil }
func (c fakeConn) Close() error                          { return nil }
func (c fakeConn) Begin() (driver.Tx, error)             { return fakeTx{}, nil }

// fakeTx is a no-op transaction: statements run on the fake directly (#311).
type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }
func (s fakeStmt) Close() error                          { return nil }
func (s fakeStmt) NumInput() int                         { return -1 }
func (s fakeStmt) Exec(a []driver.Value) (driver.Result, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.q)
	s.f.args = append(s.f.args, a)
	if s.f.xErr != nil {
		return nil, s.f.xErr
	}
	return driver.RowsAffected(1), nil
}
func (s fakeStmt) Query(a []driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.q)
	s.f.args = append(s.f.args, a)
	if s.f.qErr != nil {
		return nil, s.f.qErr
	}
	r := &fakeRows{}
	if len(s.f.cols) > 0 {
		r.cols, s.f.cols = s.f.cols[0], s.f.cols[1:]
		r.data, s.f.rows = s.f.rows[0], s.f.rows[1:]
	}
	return r, nil
}
func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

type fakeConnector struct{ f *fakeDB }

func (c fakeConnector) Connect(_ context.Context) (driver.Conn, error) { return fakeConn(c), nil }
func (c fakeConnector) Driver() driver.Driver                          { return nil }

func newFakeDB(t *testing.T) (*sqlx.DB, *fakeDB) {
	t.Helper()
	f := &fakeDB{}
	db := sqlx.NewDb(sql.OpenDB(fakeConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	return db, f
}

func (f *fakeDB) queue(cols []string, rows [][]driver.Value) {
	f.cols = append(f.cols, cols)
	f.rows = append(f.rows, rows)
}
