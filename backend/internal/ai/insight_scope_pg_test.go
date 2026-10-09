package ai

// Real-Postgres test of migration 037 (#263) and the scope-keyed insight cache.
// Skipped unless TEST_DATABASE_URL is set. It creates and drops a throwaway schema
// with minimal exams/users tables and the migration-028 ai_insight_cache table, so
// public tables are never touched.
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' \
//	    go test ./internal/ai -run TestInsightCacheScope_RealPostgres -v

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/deptscope"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	pgScopeExam = "aaaaaaaa-0000-4000-8000-000000000001"
	pgScopeUser = "bbbbbbbb-0000-4000-8000-000000000001"
	pgKeyA      = "1111111111111111111111111111111111111111111111111111111111111111"
	pgKeyB      = "2222222222222222222222222222222222222222222222222222222222222222"
)

// migration 028 DDL, the table as it exists before migration 037.
const pgInsightCache028 = `
CREATE TABLE exams (id uuid PRIMARY KEY);
CREATE TABLE users (id uuid PRIMARY KEY);
CREATE TABLE ai_insight_cache (
  exam_id       UUID        PRIMARY KEY REFERENCES exams(id) ON DELETE CASCADE,
  insights      JSONB       NOT NULL,
  generated_at  TIMESTAMPTZ NOT NULL,
  generated_by  UUID        REFERENCES users(id) ON DELETE SET NULL
);`

func TestInsightCacheScope_RealPostgres(t *testing.T) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping real-Postgres insight scope test")
	}
	const up = "../../migrations/037_ai_insight_cache_scope.up.sql"
	const down = "../../migrations/037_ai_insight_cache_scope.down.sql"
	upSQL, err := os.ReadFile(up)
	require.NoError(t, err)
	downSQL, err := os.ReadFile(down)
	require.NoError(t, err)

	schema := fmt.Sprintf("bb_263_%d", time.Now().UnixNano())
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	ctx := context.Background()

	admin, err := sqlx.Open("postgres", base)
	require.NoError(t, err)
	defer admin.Close()
	_, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`) }()

	db, err := sqlx.Open("postgres", base+sep+"search_path="+schema)
	require.NoError(t, err)
	defer db.Close()

	// Pre-migration state: one row written under the old exam_id primary key.
	_, err = db.ExecContext(ctx, pgInsightCache028+`
		INSERT INTO exams (id) VALUES ('`+pgScopeExam+`');
		INSERT INTO users (id) VALUES ('`+pgScopeUser+`');
		INSERT INTO ai_insight_cache (exam_id, insights, generated_at, generated_by)
		VALUES ('`+pgScopeExam+`', '["org-wide"]', NOW(), '`+pgScopeUser+`');`)
	require.NoError(t, err)

	// Apply migration 037 up.
	_, err = db.ExecContext(ctx, string(upSQL))
	require.NoError(t, err, "migration 037 up must apply to a table with existing rows")

	repo := NewRepository(db)

	// The pre-existing row became scope_key 'all' and is still readable.
	all, err := repo.GetInsightCache(ctx, pgScopeExam, deptscope.ScopeKeyAll)
	require.NoError(t, err)
	require.NotNil(t, all, "existing row must be backfilled as scope_key 'all'")
	assert.Equal(t, []string{"org-wide"}, all.Insights)

	// Scoped rows coexist with the 'all' row for the same exam.
	require.NoError(t, repo.UpsertInsightCache(ctx, pgScopeExam, pgKeyA, pgScopeUser, []string{"scope-a"}))
	require.NoError(t, repo.UpsertInsightCache(ctx, pgScopeExam, pgKeyB, pgScopeUser, []string{"scope-b"}))

	a, err := repo.GetInsightCache(ctx, pgScopeExam, pgKeyA)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, []string{"scope-a"}, a.Insights)

	b, err := repo.GetInsightCache(ctx, pgScopeExam, pgKeyB)
	require.NoError(t, err)
	require.NotNil(t, b)
	assert.Equal(t, []string{"scope-b"}, b.Insights)

	all, err = repo.GetInsightCache(ctx, pgScopeExam, deptscope.ScopeKeyAll)
	require.NoError(t, err)
	require.NotNil(t, all)
	assert.Equal(t, []string{"org-wide"}, all.Insights, "scoped writes must not touch the 'all' row")

	// Upsert on the same (exam, scope) overwrites that row only.
	require.NoError(t, repo.UpsertInsightCache(ctx, pgScopeExam, pgKeyA, pgScopeUser, []string{"scope-a2"}))
	a, err = repo.GetInsightCache(ctx, pgScopeExam, pgKeyA)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, []string{"scope-a2"}, a.Insights)

	// An unseen scope is a miss, not someone else's row.
	miss, err := repo.GetInsightCache(ctx, pgScopeExam, "3333333333333333333333333333333333333333333333333333333333333333")
	require.NoError(t, err)
	assert.Nil(t, miss)

	var rows int
	require.NoError(t, db.GetContext(ctx, &rows, `SELECT COUNT(*) FROM ai_insight_cache WHERE exam_id = $1`, pgScopeExam))
	assert.Equal(t, 3, rows, "one row per scope: all, A, B")

	// Down migration: scoped rows are dropped, the 'all' row survives on the old key.
	_, err = db.ExecContext(ctx, string(downSQL))
	require.NoError(t, err, "migration 037 down must apply")
	require.NoError(t, db.GetContext(ctx, &rows, `SELECT COUNT(*) FROM ai_insight_cache WHERE exam_id = $1`, pgScopeExam))
	assert.Equal(t, 1, rows, "down keeps only the 'all' row")
}
