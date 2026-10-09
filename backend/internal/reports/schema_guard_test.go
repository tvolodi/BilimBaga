package reports

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRepositorySQL_NoTenantIDColumn is a schema guard (ISS-75). The reports
// queries are only exercised against fake drivers in unit tests, so a reference
// to a column that was never migrated (exams.tenant_id / exam_sessions.tenant_id)
// slipped through and made every dashboard export fail with a 500 at runtime.
// The only migrated tenant_id column is audit_log.tenant_id (migration 007).
func TestRepositorySQL_NoTenantIDColumn(t *testing.T) {
	migrations, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(migrations) == 0 {
		t.Fatalf("cannot locate migrations: %v", err)
	}
	tenantTables := map[string]bool{}
	createRe := regexp.MustCompile(`(?is)CREATE TABLE\s+(?:IF NOT EXISTS\s+)?(\w+)\s*\((.*?)\n\);`)
	for _, m := range migrations {
		b, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		for _, mm := range createRe.FindAllStringSubmatch(string(b), -1) {
			if strings.Contains(strings.ToLower(mm[2]), "tenant_id") {
				tenantTables[strings.ToLower(mm[1])] = true
			}
		}
		if strings.Contains(strings.ToLower(string(b)), "add column") &&
			regexp.MustCompile(`(?i)ADD COLUMN\s+(IF NOT EXISTS\s+)?tenant_id`).Match(b) {
			t.Fatalf("%s adds a tenant_id column; update this guard and the reports queries", m)
		}
	}

	src, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "tenant_id") && !tenantTables["audit_log"] {
		t.Fatal("repository.go references tenant_id but no migration defines it")
	}
	for tbl := range tenantTables {
		if tbl != "audit_log" {
			t.Logf("table %s now has tenant_id; revisit reports queries", tbl)
		}
	}
	// reports queries never touch audit_log, so any tenant_id here is a bug.
	if strings.Contains(string(src), "tenant_id") {
		t.Fatal("repository.go SQL references tenant_id, which exists only on audit_log (migration 007); reports tables are single-tenant")
	}
}
