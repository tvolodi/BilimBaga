package users

// Real-Postgres check of the #253 subtree query (pgRepository.UserInDeptScope). Skipped
// unless TEST_DATABASE_URL is set. It creates and drops a throwaway schema with minimal
// departments/users tables, so public.users is never touched.
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' \
//	    go test ./internal/users -run TestUserInDeptScope_RealPostgres -v

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
	"github.com/stretchr/testify/require"
)

const (
	pgDeptRoot   = "aaaaaaaa-0000-4000-8000-000000000001" // D1: root
	pgDeptChild  = "aaaaaaaa-0000-4000-8000-000000000002" // D2: child of D1
	pgDeptSib    = "aaaaaaaa-0000-4000-8000-000000000003" // D3: second root
	pgDeptGrand  = "aaaaaaaa-0000-4000-8000-000000000004" // D4: child of D2
	pgUserRoot   = "bbbbbbbb-0000-4000-8000-000000000001" // in D1
	pgUserChild  = "bbbbbbbb-0000-4000-8000-000000000002" // in D2
	pgUserSib    = "bbbbbbbb-0000-4000-8000-000000000003" // in D3
	pgUserGrand  = "bbbbbbbb-0000-4000-8000-000000000004" // in D4
	pgUserNoDept = "bbbbbbbb-0000-4000-8000-000000000005" // no department
	pgUserNone   = "bbbbbbbb-0000-4000-8000-0000000000ff" // does not exist
)

func TestUserInDeptScope_RealPostgres(t *testing.T) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping real-Postgres department-scope test")
	}
	schema := fmt.Sprintf("bb_253_%d", time.Now().UnixNano())
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}

	admin, err := sqlx.Open("postgres", base)
	require.NoError(t, err)
	defer admin.Close()
	_, err = admin.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()

	db, err := sqlx.Open("postgres", base+sep+"search_path="+schema)
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE departments (id uuid PRIMARY KEY, parent_id uuid REFERENCES departments(id));
		CREATE TABLE users (id uuid PRIMARY KEY, department_id uuid REFERENCES departments(id));
		INSERT INTO departments (id, parent_id) VALUES
			('`+pgDeptRoot+`', NULL), ('`+pgDeptChild+`', '`+pgDeptRoot+`'),
			('`+pgDeptSib+`', NULL), ('`+pgDeptGrand+`', '`+pgDeptChild+`');
		INSERT INTO users (id, department_id) VALUES
			('`+pgUserRoot+`', '`+pgDeptRoot+`'), ('`+pgUserChild+`', '`+pgDeptChild+`'),
			('`+pgUserSib+`', '`+pgDeptSib+`'), ('`+pgUserGrand+`', '`+pgDeptGrand+`'),
			('`+pgUserNoDept+`', NULL);`)
	require.NoError(t, err)

	repo := NewRepository(db).(*pgRepository)
	scope := func(deptID string) deptscope.Scope {
		return deptscope.Scope{Restricted: true, DepartmentID: deptID}
	}
	cases := []struct {
		name   string
		sc     deptscope.Scope
		userID string
		want   bool
	}{
		{"root sees itself", scope(pgDeptRoot), pgUserRoot, true},
		{"root sees child department", scope(pgDeptRoot), pgUserChild, true},
		{"root sees grandchild department", scope(pgDeptRoot), pgUserGrand, true},
		{"root does not see second root", scope(pgDeptRoot), pgUserSib, false},
		{"root does not see user without department", scope(pgDeptRoot), pgUserNoDept, false},
		{"child does not see its parent", scope(pgDeptChild), pgUserRoot, false},
		{"child sees its own subtree", scope(pgDeptChild), pgUserGrand, true},
		{"caller without department sees nothing", deptscope.Scope{Restricted: true}, pgUserRoot, false},
		{"unknown user is out of scope", scope(pgDeptRoot), pgUserNone, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.UserInDeptScope(ctx, tc.sc, tc.userID)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
