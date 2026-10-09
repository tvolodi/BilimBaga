package db_test

// Real-Postgres test of migration 035 (ISS-181). Skipped unless TEST_DATABASE_URL is set.
//
// UAT command (against the live-stack database; it only creates and drops a throwaway
// schema named bb_iss181_<random>, never touching public.users):
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:5432/DB?sslmode=disable' \
//	    go test ./internal/db -run TestUniqueEmailIndex -v

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
)

const migration035 = "../../migrations/035_users_email_lower_unique"

type pgSchema struct {
	conn    *sql.Conn
	schema  string
	notices []string
}

func newPGSchema(t *testing.T) *pgSchema {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping real-Postgres migration 035 test (see file header for the UAT command)")
	}
	ps := &pgSchema{schema: fmt.Sprintf("bb_iss181_%d_%d", time.Now().UnixNano()%1000000, rand.Intn(1000000))}

	base, err := pq.NewConnector(url)
	if err != nil {
		t.Fatalf("connector: %v", err)
	}
	connector := pq.ConnectorWithNoticeHandler(base, func(n *pq.Error) { ps.notices = append(ps.notices, n.Message) })
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1) // one pinned session so search_path sticks
	conn, err := db.Conn(context.Background())
	if err != nil {
		db.Close()
		t.Fatalf("connect: %v", err)
	}
	ps.conn = conn
	t.Cleanup(func() {
		_, _ = conn.ExecContext(context.Background(), "SET search_path TO public")
		if _, err := conn.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+ps.schema+" CASCADE"); err != nil {
			t.Errorf("cleanup drop schema %s: %v", ps.schema, err)
		}
		conn.Close()
		db.Close()
	})
	ps.exec(t, "CREATE SCHEMA "+ps.schema)
	ps.exec(t, "SET search_path TO "+ps.schema)
	// Mirrors the production constraint: case-sensitive UNIQUE(email).
	ps.exec(t, "CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL UNIQUE)")
	return ps
}

func (p *pgSchema) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := p.conn.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func (p *pgSchema) scalar(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := p.conn.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	return n
}

func (p *pgSchema) indexCount(t *testing.T) int {
	return p.scalar(t, "SELECT count(*) FROM pg_indexes WHERE schemaname = $1 AND indexname = 'idx_users_email_lower_unique'", p.schema)
}

func readSQL(t *testing.T, ext string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(migration035 + ext))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUniqueEmailIndex_NoDuplicates_CreatesIndex(t *testing.T) {
	p := newPGSchema(t)
	p.exec(t, "INSERT INTO users (email) VALUES ('a@x.com'), ('b@x.com')")

	p.exec(t, readSQL(t, ".up.sql"))
	if p.indexCount(t) != 1 {
		t.Fatal("index should exist when there are no duplicates")
	}
	// A case-variant duplicate must now be rejected by the database.
	_, err := p.conn.ExecContext(context.Background(), "INSERT INTO users (email) VALUES ('A@X.com')")
	var pqe *pq.Error
	if !errors.As(err, &pqe) || pqe.Code != "23505" {
		t.Fatalf("expected unique violation 23505, got %v", err)
	}
	// Re-running is idempotent.
	p.exec(t, readSQL(t, ".up.sql"))
	// Down drops it, and is itself idempotent.
	p.exec(t, readSQL(t, ".down.sql"))
	p.exec(t, readSQL(t, ".down.sql"))
	if p.indexCount(t) != 0 {
		t.Fatal("down migration should drop the index")
	}
}

func TestUniqueEmailIndex_WithDuplicates_SkipsWithoutError(t *testing.T) {
	p := newPGSchema(t)
	p.exec(t, "INSERT INTO users (email) VALUES ('John@X.com'), ('john@x.com'), ('solo@x.com')")

	p.exec(t, readSQL(t, ".up.sql")) // must NOT fail (would take the API down at startup)
	if p.indexCount(t) != 0 {
		t.Fatal("index must not be created while duplicates exist")
	}
	if n := p.scalar(t, "SELECT count(*) FROM users WHERE email IN ('John@X.com','john@x.com','solo@x.com')"); n != 3 {
		t.Fatalf("rows must be untouched, matched %d of 3", n)
	}
	found := false
	for _, n := range p.notices {
		if strings.Contains(n, "NOT created") && strings.Contains(n, "1 group") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the skip NOTICE, got %v", p.notices)
	}
	// Down on a skipped migration is a harmless no-op.
	p.exec(t, readSQL(t, ".down.sql"))
}
