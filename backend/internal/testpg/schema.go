// Package testpg gives real-Postgres tests a throwaway schema of their own (#547).
//
// pg_trgm is a database-wide extension, and CREATE EXTENSION IF NOT EXISTS checks only its name. Before this package,
// each test package installed pg_trgm into its own schema: a package that ran second skipped the install and failed on
// gin_trgm_ops, and a DROP SCHEMA ... CASCADE dropped the extension under the packages still running. NewSchema installs
// pg_trgm into public once, under an advisory lock, and ends every test's search_path with public. No test drops public.
package testpg

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// trgmLockKey is the advisory lock that serialises the one-time pg_trgm install across test packages (#547).
const trgmLockKey = 547

// NewSchema creates the throwaway schema prefix_<nanos> and returns a connection whose search_path is that schema,
// then public. base is TEST_DATABASE_URL. The schema is dropped when the test ends.
func NewSchema(t testing.TB, base, prefix string) *sqlx.DB {
	t.Helper()
	ctx := context.Background()
	schema := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())

	admin, err := sqlx.Open("postgres", base)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })
	installTrgm(t, admin)

	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = admin.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })

	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	db, err := sqlx.Open("postgres", base+sep+"search_path="+schema+",public")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// installTrgm makes pg_trgm available in public. The advisory lock stops two packages from creating it at the same time,
// which would otherwise fail on the pg_extension name index.
func installTrgm(t testing.TB, admin *sqlx.DB) {
	t.Helper()
	ctx := context.Background()
	conn, err := admin.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", trgmLockKey)
	require.NoError(t, err)
	defer func() { _, _ = conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", trgmLockKey) }()

	_, err = conn.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public")
	require.NoError(t, err)
}
