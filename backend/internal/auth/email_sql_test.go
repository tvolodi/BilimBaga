package auth

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capDB records the SQL text and bound args of every query; it returns no rows.
type capDB struct {
	mu      sync.Mutex
	queries []string
	args    [][]driver.Value
}
type capConn struct{ f *capDB }
type capStmt struct {
	f *capDB
	q string
}
type capRows struct{}

func (c capConn) Prepare(q string) (driver.Stmt, error) { return capStmt{c.f, q}, nil }
func (c capConn) Close() error                          { return nil }
func (c capConn) Begin() (driver.Tx, error)             { return nil, errors.New("no tx") }
func (s capStmt) Close() error                          { return nil }
func (s capStmt) NumInput() int                         { return -1 }
func (s capStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}
func (s capStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.q)
	s.f.args = append(s.f.args, append([]driver.Value(nil), args...))
	return capRows{}, nil
}
func (capRows) Columns() []string              { return []string{"id"} }
func (capRows) Close() error                   { return nil }
func (capRows) Next(dest []driver.Value) error { return io.EOF }

type capConnector struct{ f *capDB }

func (c capConnector) Connect(context.Context) (driver.Conn, error) { return capConn(c), nil }
func (c capConnector) Driver() driver.Driver                        { return nil }

func newCapDB(t *testing.T) (*sqlx.DB, *capDB) {
	t.Helper()
	f := &capDB{}
	db := sqlx.NewDb(sql.OpenDB(capConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	return db, f
}

func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func TestRepository_GetUserByEmail_ComparesLowerOfStoredEmail(t *testing.T) {
	db, f := newCapDB(t)
	_, err := NewRepository(db).GetUserByEmail(context.Background(), "john.doe@corp.com")
	assert.ErrorIs(t, err, ErrNotFound)
	require.Len(t, f.queries, 1)
	assert.Contains(t, squash(f.queries[0]), "WHERE lower(u.email) = $1")
	assert.NotContains(t, squash(f.queries[0]), "u.email = $1")
	assert.Equal(t, "john.doe@corp.com", f.args[0][0], "bound as a parameter, never concatenated")
}

func TestBootstrapStore_GetAdminCredentials_ComparesLowerOfStoredEmail(t *testing.T) {
	db, f := newCapDB(t)
	_, _, err := NewBootstrapStore(db).GetAdminCredentials(context.Background(), BootstrapAdminEmail)
	assert.ErrorIs(t, err, ErrNotFound)
	require.Len(t, f.queries, 1)
	assert.Equal(t, "SELECT id, password_hash FROM users WHERE lower(email) = lower($1)", squash(f.queries[0]))
}
