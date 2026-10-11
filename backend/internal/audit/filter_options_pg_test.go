package audit_test

// #557: real-Postgres test of the audit filter options. The SQL (DISTINCT, the tenant condition, the empty and NULL
// filter and the C collation) runs against the production migrations. Skipped unless TEST_DATABASE_URL is set.
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' \
//	    go test ./internal/audit -run FilterOptions_RealPostgres -v

import (
	"context"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/bilimbaga/bilimbaga/internal/testpg"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuditPGDB(t *testing.T) *sqlx.DB {
	t.Helper()
	base := testpg.Base(t)
	db := testpg.NewSchema(t, base, "bb_557")
	require.NoError(t, dbpkg.RunMigrations(db, "../../migrations"))
	return db
}

func seedAuditRow(t *testing.T, db *sqlx.DB, tenant, action string, entityType *string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO audit_log (tenant_id, action, entity_type) VALUES ($1, $2, $3)`, tenant, action, entityType)
	require.NoError(t, err)
}

func textPtr(s string) *string { return &s }

func TestFilterOptions_RealPostgres_EmptyLogIsEmptyArrays(t *testing.T) {
	db := newAuditPGDB(t)

	opts, err := audit.ListFilterOptions(context.Background(), db, "tenant-a")

	require.NoError(t, err)
	assert.NotNil(t, opts.Actions)
	assert.Empty(t, opts.Actions)
	assert.NotNil(t, opts.EntityTypes)
	assert.Empty(t, opts.EntityTypes)
}

func TestFilterOptions_RealPostgres_TenantScopeDistinctSortedAndNoEmptyValues(t *testing.T) {
	db := newAuditPGDB(t)
	ctx := context.Background()
	seedAuditRow(t, db, "tenant-a", "user.login", textPtr("user"))
	seedAuditRow(t, db, "tenant-a", "exam.create", textPtr("exam"))
	seedAuditRow(t, db, "tenant-a", "exam.create", textPtr("exam")) // a repeat is listed once
	seedAuditRow(t, db, "tenant-a", "auth.login.success", nil)      // NULL entity type is left out
	seedAuditRow(t, db, "tenant-a", "", textPtr(""))                // empty action and empty entity type are left out
	seedAuditRow(t, db, "tenant-b", "zz.other", textPtr("zzz"))     // another tenant's rows never appear

	opts, err := audit.ListFilterOptions(ctx, db, "tenant-a")
	require.NoError(t, err)
	assert.Equal(t, []string{"auth.login.success", "exam.create", "user.login"}, opts.Actions)
	assert.Equal(t, []string{"exam", "user"}, opts.EntityTypes)

	other, err := audit.ListFilterOptions(ctx, db, "tenant-b")
	require.NoError(t, err)
	assert.Equal(t, []string{"zz.other"}, other.Actions)
	assert.Equal(t, []string{"zzz"}, other.EntityTypes)
}
