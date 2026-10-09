package db_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/jmoiron/sqlx"
)

// migFake is an in-process database/sql driver that answers the handful of
// queries golang-migrate's postgres driver issues. No real database involved.
type migFake struct {
	mu         sync.Mutex
	failDBName bool // fail the CURRENT_DATABASE() probe
	hasVersion bool
	version    int64
	dirty      bool
	execs      []string
}

type migConn struct{ f *migFake }
type migStmt struct {
	f *migFake
	q string
}
type migTx struct{}
type migRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (c migConn) Prepare(q string) (driver.Stmt, error) { return migStmt{c.f, q}, nil }
func (c migConn) Close() error                          { return nil }
func (c migConn) Begin() (driver.Tx, error)             { return migTx{}, nil }
func (migTx) Commit() error                             { return nil }
func (migTx) Rollback() error                           { return nil }
func (s migStmt) Close() error                          { return nil }
func (s migStmt) NumInput() int                         { return -1 }

func (s migStmt) Exec(_ []driver.Value) (driver.Result, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.execs = append(s.f.execs, s.q)
	if strings.Contains(s.q, "BOOM") {
		return nil, errors.New("boom: syntax error")
	}
	return driver.RowsAffected(0), nil
}

func (s migStmt) Query(_ []driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	switch {
	case strings.Contains(s.q, "CURRENT_DATABASE"):
		if s.f.failDBName {
			return nil, errors.New("connection refused")
		}
		return &migRows{cols: []string{"db"}, data: [][]driver.Value{{"testdb"}}}, nil
	case strings.Contains(s.q, "CURRENT_SCHEMA"):
		return &migRows{cols: []string{"schema"}, data: [][]driver.Value{{"public"}}}, nil
	case strings.Contains(s.q, "information_schema.tables"):
		return &migRows{cols: []string{"count"}, data: [][]driver.Value{{int64(1)}}}, nil
	case strings.Contains(s.q, "version, dirty"):
		if !s.f.hasVersion {
			return &migRows{cols: []string{"version", "dirty"}}, nil
		}
		return &migRows{cols: []string{"version", "dirty"}, data: [][]driver.Value{{s.f.version, s.f.dirty}}}, nil
	}
	return &migRows{cols: []string{"x"}}, nil
}

func (r *migRows) Columns() []string { return r.cols }
func (r *migRows) Close() error      { return nil }
func (r *migRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

type migConnector struct{ f *migFake }

func (c migConnector) Connect(context.Context) (driver.Conn, error) { return migConn(c), nil }
func (c migConnector) Driver() driver.Driver                        { return nil }

func newMigDB(t *testing.T, f *migFake) *sqlx.DB {
	t.Helper()
	db := sqlx.NewDb(sql.OpenDB(migConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func writeMigrations(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.ToSlash(dir) // forward slashes keep the file:// URL valid on Windows
}

func TestRunMigrations_AppliesPending(t *testing.T) {
	f := &migFake{}
	dir := writeMigrations(t, map[string]string{
		"1_init.up.sql":   "CREATE TABLE a (id int);",
		"1_init.down.sql": "DROP TABLE a;",
	})

	if err := dbpkg.RunMigrations(newMigDB(t, f), dir); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if !strings.Contains(strings.Join(f.execs, "\n"), "CREATE TABLE a") {
		t.Errorf("migration body was not executed; execs = %v", f.execs)
	}
}

func TestRunMigrations_NoChangeIsSuccess(t *testing.T) {
	f := &migFake{hasVersion: true, version: 1}
	dir := writeMigrations(t, map[string]string{"1_init.up.sql": "SELECT 1;"})

	if err := dbpkg.RunMigrations(newMigDB(t, f), dir); err != nil {
		t.Fatalf("ErrNoChange must map to nil, got %v", err)
	}
	for _, q := range f.execs {
		if strings.Contains(q, "SELECT 1;") {
			t.Errorf("already-applied migration was re-run")
		}
	}
}

func TestRunMigrations_DirtyDatabaseReturnsVersion(t *testing.T) {
	f := &migFake{hasVersion: true, version: 7, dirty: true}
	dir := writeMigrations(t, map[string]string{"7_x.up.sql": "SELECT 1;"})

	err := dbpkg.RunMigrations(newMigDB(t, f), dir)
	if err == nil {
		t.Fatal("expected error for dirty database")
	}
	if !strings.Contains(err.Error(), "dirty at version 7") {
		t.Errorf("error should name the dirty version, got %q", err)
	}
}

func TestRunMigrations_FailingStatementIsWrapped(t *testing.T) {
	f := &migFake{}
	dir := writeMigrations(t, map[string]string{"1_bad.up.sql": "BOOM;"})

	err := dbpkg.RunMigrations(newMigDB(t, f), dir)
	if err == nil || !strings.HasPrefix(err.Error(), "migrate: up: ") {
		t.Fatalf("want 'migrate: up:' wrapped error, got %v", err)
	}
}

func TestRunMigrations_DriverCreationFailure(t *testing.T) {
	f := &migFake{failDBName: true}
	err := dbpkg.RunMigrations(newMigDB(t, f), writeMigrations(t, nil))
	if err == nil || !strings.HasPrefix(err.Error(), "migrate: create driver: ") {
		t.Fatalf("want 'migrate: create driver:' error, got %v", err)
	}
}

func TestRunMigrations_MissingSourceDirectory(t *testing.T) {
	f := &migFake{}
	missing := filepath.ToSlash(filepath.Join(t.TempDir(), "does-not-exist"))
	err := dbpkg.RunMigrations(newMigDB(t, f), missing)
	if err == nil || !strings.HasPrefix(err.Error(), "migrate: init: ") {
		t.Fatalf("want 'migrate: init:' error, got %v", err)
	}
}
