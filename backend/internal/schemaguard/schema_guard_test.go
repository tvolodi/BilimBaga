// Package schemaguard holds a test-only schema guard (ISS-82). Repository SQL
// is only exercised against fake drivers in unit tests, so a reference to a
// column that was never migrated (e.g. exams.tenant_id, ISS-75/ISS-82) only
// fails at runtime with a 500. This test replays backend/migrations/*.up.sql
// into an in-memory table->columns map and statically checks every SQL string
// literal found in backend/internal against it.
//
// Scope (ISS-88, retro-002 decision 1): the walk covers every package under
// backend/internal (and backend/cmd), not a single module. A coverage guard
// (TestEveryDBCallingFileIsScanned) fails when a file issues DB calls but none
// of its SQL could be collected, so a new package cannot silently escape.
//
// Known limits (static analysis, no DB): SQL assembled from non-constant
// pieces (variables, fmt.Sprintf arguments, dynamic ORDER BY) is only seen for
// its constant parts; unqualified columns in multi-table queries are checked
// only against "exists in at least one referenced table"; column types,
// constraints and function/operator validity are not checked. A real-Postgres
// test remains the authority for those (see swarm/roles/uat.md).
package schemaguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type schema map[string]map[string]bool // table -> column set

var (
	createRe  = regexp.MustCompile(`(?is)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\((.*)\)\s*$`)
	alterRe   = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(\w+)\s+(.*)$`)
	dropTblRe = regexp.MustCompile(`(?is)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(\w+)`)
	addColRe  = regexp.MustCompile(`(?is)^ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`)
	dropColRe = regexp.MustCompile(`(?is)^DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\w+)`)
	renColRe  = regexp.MustCompile(`(?is)^RENAME\s+COLUMN\s+(\w+)\s+TO\s+(\w+)`)
)

func stripComments(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if i := strings.Index(l, "--"); i >= 0 {
			l = l[:i]
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

// splitTop splits on sep at parenthesis depth 0.
func splitTop(s string, sep rune) []string {
	var parts []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

func loadSchema(t *testing.T) schema {
	t.Helper()
	files, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("cannot locate migrations: %v", err)
	}
	sort.Strings(files)
	sc := schema{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, stmt := range splitTop(stripComments(string(b)), ';') {
			stmt = strings.TrimSpace(stmt)
			low := strings.ToLower(stmt)
			switch {
			case strings.HasPrefix(low, "create table"):
				m := createRe.FindStringSubmatch(stmt)
				if m == nil {
					continue
				}
				cols := map[string]bool{}
				for _, item := range splitTop(m[2], ',') {
					w := strings.Fields(strings.TrimSpace(item))
					if len(w) == 0 {
						continue
					}
					switch strings.ToLower(w[0]) {
					case "constraint", "primary", "unique", "foreign", "check", "exclude":
						continue
					}
					cols[strings.ToLower(w[0])] = true
				}
				sc[strings.ToLower(m[1])] = cols
			case strings.HasPrefix(low, "drop table"):
				if m := dropTblRe.FindStringSubmatch(stmt); m != nil {
					delete(sc, strings.ToLower(m[1]))
				}
			case strings.HasPrefix(low, "alter table"):
				m := alterRe.FindStringSubmatch(stmt)
				if m == nil {
					continue
				}
				tbl := strings.ToLower(m[1])
				for _, act := range splitTop(m[2], ',') {
					act = strings.TrimSpace(act)
					if x := addColRe.FindStringSubmatch(act); x != nil && sc[tbl] != nil {
						sc[tbl][strings.ToLower(x[1])] = true
					} else if x := dropColRe.FindStringSubmatch(act); x != nil && sc[tbl] != nil {
						delete(sc[tbl], strings.ToLower(x[1]))
					} else if x := renColRe.FindStringSubmatch(act); x != nil && sc[tbl] != nil {
						delete(sc[tbl], strings.ToLower(x[1]))
						sc[tbl][strings.ToLower(x[2])] = true
					}
				}
			}
		}
	}
	return sc
}

type literal struct {
	file string
	line int
	sql  string
}

// collectSQL returns every string literal (concatenations folded) that looks
// like a complete SQL statement.
func collectSQL(t *testing.T, roots ...string) []literal {
	t.Helper()
	var queries []literal
	fset := token.NewFileSet()
	for _, root := range roots {
		queries = append(queries, collectSQLFrom(t, fset, root)...)
	}
	return queries
}

func collectSQLFrom(t *testing.T, fset *token.FileSet, root string) []literal {
	t.Helper()
	var queries []literal
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", p, perr)
		}
		var fold func(e ast.Expr) (string, bool)
		fold = func(e ast.Expr) (string, bool) {
			switch v := e.(type) {
			case *ast.BasicLit:
				if v.Kind == token.STRING {
					s, err := strconv.Unquote(v.Value)
					return s, err == nil
				}
			case *ast.BinaryExpr:
				if v.Op == token.ADD {
					l, ok1 := fold(v.X)
					r, ok2 := fold(v.Y)
					if ok1 && ok2 {
						return l + r, true
					}
				}
			case *ast.ParenExpr:
				return fold(v.X)
			}
			return "", false
		}
		ast.Inspect(f, func(n ast.Node) bool {
			e, ok := n.(ast.Expr)
			if !ok {
				return true
			}
			s, ok := fold(e)
			if !ok {
				return true
			}
			if sqlStartRe.MatchString(s) {
				queries = append(queries, literal{file: filepath.ToSlash(p), line: fset.Position(e.Pos()).Line, sql: s})
			}
			return false
		})
		return nil
	})
	return queries
}

var (
	sqlStartRe = regexp.MustCompile(`(?is)^\s*(SELECT\s.+\sFROM\s|INSERT\s+INTO\s|UPDATE\s+\w+\s+SET\s|DELETE\s+FROM\s|WITH\s+(?:RECURSIVE\s+)?\w+\s*(?:\([^)]*\))?\s*AS\s)`)
	tableRefRe = regexp.MustCompile(`(?i)\b(?:FROM|JOIN|UPDATE|INTO)\s+(\w+)(?:\s+(?:AS\s+)?(\w+))?`)
	cteRe      = regexp.MustCompile(`(?i)\b(\w+)\s*(?:\([^)]*\))?\s*AS\s*\(`)
	qualRe     = regexp.MustCompile(`\b([a-z_]\w*)\.([a-z_]\w*)\b`)
	insertRe   = regexp.MustCompile(`(?is)INSERT\s+INTO\s+(\w+)\s*\(([^)]*)\)`)
	setRe      = regexp.MustCompile(`(?is)\bSET\b(.*?)(?:\bWHERE\b|\bFROM\b|\bRETURNING\b|$)`)
	whereColRe = regexp.MustCompile(`(?i)\b(?:WHERE|AND|OR)\s+(\w+)\s*(?:=|<>|!=|<=|>=|<|>|\s+IN\b|\s+IS\b|\s+LIKE\b|\s+ILIKE\b)`)
	selListRe  = regexp.MustCompile(`(?is)^\s*SELECT\s+(.*?)\s+FROM\s`)
	identRe    = regexp.MustCompile(`^[a-z_]\w*$`)
	strLitRe   = regexp.MustCompile(`'[^']*'`)
)

var notAlias = map[string]bool{"where": true, "set": true, "on": true, "left": true, "right": true, "inner": true,
	"outer": true, "join": true, "group": true, "order": true, "values": true, "using": true, "limit": true,
	"offset": true, "returning": true, "select": true, "cross": true, "full": true, "union": true, "having": true,
	"for": true, "do": true, "and": true, "or": true, "lateral": true, "not": true, "null": true,
	"true": true, "false": true, "skip": true}

type finding struct {
	file       string
	line       int
	table, col string
	why        string
}

// checkQuery returns problems for one SQL statement.
func checkQuery(sc schema, q literal, globalCTEs map[string]bool) []finding {
	var out []finding
	sql := strLitRe.ReplaceAllString(q.sql, "''")
	ctes := map[string]bool{}
	for k := range globalCTEs {
		ctes[k] = true
	}
	for _, m := range cteRe.FindAllStringSubmatch(sql, -1) {
		ctes[strings.ToLower(m[1])] = true
	}
	aliases := map[string]string{} // alias or table name -> table
	var real []string
	for _, m := range tableRefRe.FindAllStringSubmatch(sql, -1) {
		tbl := strings.ToLower(m[1])
		if ctes[tbl] || notAlias[tbl] || tbl == "unnest" || tbl == "generate_series" || strings.HasPrefix(tbl, "pg_") {
			continue
		}
		if sc[tbl] == nil {
			out = append(out, finding{q.file, q.line, tbl, "", "table does not exist in migrated schema"})
			continue
		}
		real = append(real, tbl)
		aliases[tbl] = tbl
		if a := strings.ToLower(m[2]); a != "" && !notAlias[a] {
			aliases[a] = tbl
		}
	}
	check := func(tbl, col string) {
		if sc[tbl] != nil && !sc[tbl][col] {
			out = append(out, finding{q.file, q.line, tbl, col, "column does not exist"})
		}
	}
	for _, m := range qualRe.FindAllStringSubmatch(sql, -1) {
		if tbl, ok := aliases[strings.ToLower(m[1])]; ok {
			check(tbl, strings.ToLower(m[2]))
		}
	}
	if m := insertRe.FindStringSubmatch(sql); m != nil {
		for _, c := range strings.Split(m[2], ",") {
			check(strings.ToLower(m[1]), strings.ToLower(strings.TrimSpace(c)))
		}
	}
	uniq := map[string]bool{}
	for _, r := range real {
		uniq[r] = true
	}
	if len(uniq) > 1 {
		// Unqualified WHERE/AND/OR column in a multi-table query: it must exist
		// in at least one referenced table (or be a select alias / CTE column).
		for _, m := range whereColRe.FindAllStringSubmatch(sql, -1) {
			c := strings.ToLower(m[1])
			if notAlias[c] || aliasDefined(sql, c) {
				continue
			}
			found := false
			for tbl := range uniq {
				if sc[tbl][c] {
					found = true
				}
			}
			if !found {
				out = append(out, finding{q.file, q.line, "(any of " + strings.Join(sortedKeys(uniq), ",") + ")", c, "column does not exist in any referenced table"})
			}
		}
	}
	if len(uniq) == 1 {
		tbl := real[0]
		upper := strings.ToUpper(strings.TrimSpace(sql))
		for _, m := range whereColRe.FindAllStringSubmatch(sql, -1) {
			if c := strings.ToLower(m[1]); !notAlias[c] {
				check(tbl, c)
			}
		}
		if m := setRe.FindStringSubmatch(sql); m != nil && strings.HasPrefix(upper, "UPDATE") {
			for _, a := range splitTop(m[1], ',') {
				if kv := strings.SplitN(a, "=", 2); len(kv) == 2 {
					if c := strings.ToLower(strings.TrimSpace(kv[0])); identRe.MatchString(c) {
						check(tbl, c)
					}
				}
			}
		}
		if m := selListRe.FindStringSubmatch(sql); m != nil && strings.HasPrefix(upper, "SELECT") {
			for _, item := range splitTop(m[1], ',') {
				w := strings.Fields(strings.ToLower(item))
				if len(w) >= 1 && identRe.MatchString(w[0]) && (len(w) == 1 || (len(w) == 3 && w[1] == "as")) {
					check(tbl, w[0])
				}
			}
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

// aliasDefined reports whether name is introduced as "AS name" in sql
// (select-list alias, CTE or derived-table column), so it is not a table column.
func aliasDefined(sql, name string) bool {
	return regexp.MustCompile(`(?i)\bAS\s+` + regexp.QuoteMeta(name) + `\b`).MatchString(sql)
}

var dbCallRe = regexp.MustCompile(`\.(?:QueryContext|QueryRowContext|ExecContext|GetContext|SelectContext|NamedExecContext|QueryxContext|QueryRowxContext|Query|QueryRow|Exec|Get|Select)\(`)

// TestEveryDBCallingFileIsScanned is the coverage guard: any non-test Go file
// that calls a DB method with a SQL-looking string must have had at least one
// statement collected. Files whose DB calls only take non-literal SQL are
// listed in nonLiteralAllowed with a reason.
func TestEveryDBCallingFileIsScanned(t *testing.T) {
	scanned := map[string]bool{}
	for _, q := range collectSQL(t, "..", "../../cmd") {
		scanned[q.file] = true
	}
	sqlWordRe := regexp.MustCompile(`\b(SELECT [^"]|INSERT INTO |DELETE FROM |UPDATE \w+ SET )`)
	for _, root := range []string{"..", "../../cmd"} {
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			b, _ := os.ReadFile(p)
			src := string(b)
			if dbCallRe.MatchString(src) && sqlWordRe.MatchString(src) && !scanned[filepath.ToSlash(p)] {
				if reason, ok := nonLiteralAllowed[filepath.ToSlash(p)]; ok {
					t.Logf("allowlisted (%s): %s", reason, p)
					return nil
				}
				t.Errorf("%s issues DB calls with SQL text but no statement was collected; fix the collector or allowlist it with a reason", p)
			}
			return nil
		})
	}
}

// nonLiteralAllowed lists files whose SQL cannot be statically collected.
var nonLiteralAllowed = map[string]string{
	// SQL is assembled from fragments at runtime; covered by TestDynamicSQLFragments.
	"../audit/repository.go": "dynamic WHERE built with fmt.Sprintf",
}

// dynamicFragmentFiles maps files with runtime-assembled SQL to the table
// aliases their fragments use. Every "alias.column" token found in the file's
// string literals must exist in the migrated schema.
var dynamicFragmentFiles = map[string]map[string]string{
	"../audit/repository.go": {"al": "audit_log", "u": "users"},
}

func TestDynamicSQLFragments(t *testing.T) {
	sc := loadSchema(t)
	fset := token.NewFileSet()
	for file, aliases := range dynamicFragmentFiles {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		seen := 0
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			for _, m := range qualRe.FindAllStringSubmatch(strings.ToLower(s), -1) {
				tbl, ok := aliases[m[1]]
				if !ok {
					continue
				}
				seen++
				if !sc[tbl][m[2]] {
					t.Errorf("%s:%d %s.%s: column does not exist in %s", file, fset.Position(lit.Pos()).Line, m[1], m[2], tbl)
				}
			}
			return true
		})
		if seen == 0 {
			t.Errorf("%s: no alias.column fragments found; update dynamicFragmentFiles", file)
		}
	}
}

func TestRepositorySQL_MatchesMigratedSchema(t *testing.T) {
	sc := loadSchema(t)

	// Sanity-check the replay itself against known facts.
	if sc["exams"] == nil || sc["exams"]["tenant_id"] || sc["exam_sessions"]["tenant_id"] {
		t.Fatal("schema replay: exams/exam_sessions must exist and have no tenant_id")
	}
	if !sc["audit_log"]["tenant_id"] || sc["audit_log"]["user_id"] {
		t.Fatal("schema replay: audit_log must be the migration-007 shape")
	}
	if sc["session_questions"]["id"] || !sc["session_questions"]["rule_id"] || !sc["exams"]["adaptive"] {
		t.Fatal("schema replay: ALTER TABLE ADD/DROP COLUMN not applied")
	}

	queries := collectSQL(t, "..", "../../cmd")
	if len(queries) < 50 {
		t.Fatalf("only %d SQL statements found; collector is broken", len(queries))
	}
	perFile := map[string]int{}
	for _, q := range queries {
		perFile[q.file]++
	}
	for f, n := range perFile {
		t.Logf("checked %3d statements in %s", n, f)
	}
	// CTEs are sometimes defined in one const and used in another query.
	ctes := map[string]bool{}
	for _, q := range queries {
		for _, m := range cteRe.FindAllStringSubmatch(q.sql, -1) {
			ctes[strings.ToLower(m[1])] = true
		}
	}
	var bad []string
	for _, q := range queries {
		for _, f := range checkQuery(sc, q, ctes) {
			bad = append(bad, fmt.Sprintf("%s:%d table=%s col=%s: %s", f.file, f.line, f.table, f.col, f.why))
		}
	}
	if len(bad) > 0 {
		t.Fatalf("SQL references schema objects missing from migrations:\n  %s", strings.Join(bad, "\n  "))
	}
}

// TestCheckQuery_FlagsKnownBadQuery proves the checker catches the ISS-82 bug.
func TestCheckQuery_FlagsKnownBadQuery(t *testing.T) {
	sc := loadSchema(t)
	bad := literal{file: "x.go", line: 1, sql: `SELECT title, passing_score_pct, tenant_id FROM exams WHERE id = $1`}
	if got := checkQuery(sc, bad, nil); len(got) != 1 || got[0].col != "tenant_id" {
		t.Fatalf("expected one tenant_id finding, got %+v", got)
	}
	bad2 := literal{file: "x.go", line: 2, sql: `SELECT e.id FROM exams e JOIN exam_sessions s ON s.exam_id = e.id WHERE s.tenant_id = $1`}
	if got := checkQuery(sc, bad2, nil); len(got) != 1 || got[0].table != "exam_sessions" {
		t.Fatalf("expected exam_sessions.tenant_id finding, got %+v", got)
	}
	ok := literal{file: "x.go", line: 3, sql: `INSERT INTO audit_log (tenant_id, actor_id, action) VALUES ($1,$2,$3)`}
	if got := checkQuery(sc, ok, nil); len(got) != 0 {
		t.Fatalf("audit_log insert wrongly flagged: %+v", got)
	}
}

// TestOnlyAuditLogHasTenantID guards the single-tenant assumption: if a
// migration ever adds tenant_id elsewhere, revisit the queries fixed in
// ISS-75/ISS-82.
func TestOnlyAuditLogHasTenantID(t *testing.T) {
	for tbl, cols := range loadSchema(t) {
		if cols["tenant_id"] && tbl != "audit_log" {
			t.Errorf("table %s now has tenant_id; revisit single-tenant assumptions (ISS-75/ISS-82)", tbl)
		}
	}
}

// TestCheckQuery_FlagsUnqualifiedColumnInJoin proves the multi-table check
// catches a made-up unqualified column and tolerates select-list aliases.
func TestCheckQuery_FlagsUnqualifiedColumnInJoin(t *testing.T) {
	sc := loadSchema(t)
	bad := literal{file: "x.go", line: 1, sql: `SELECT e.id FROM exams e JOIN exam_sessions s ON s.exam_id = e.id WHERE no_such_col = $1`}
	if got := checkQuery(sc, bad, nil); len(got) != 1 || got[0].col != "no_such_col" {
		t.Fatalf("expected one no_such_col finding, got %+v", got)
	}
	ok := literal{file: "x.go", line: 2, sql: `SELECT COUNT(*) AS n FROM exams e JOIN exam_sessions s ON s.exam_id = e.id WHERE n > 1 AND status = 'x'`}
	if got := checkQuery(sc, ok, nil); len(got) != 0 {
		t.Fatalf("alias/real column wrongly flagged: %+v", got)
	}
}

// TestMigration035TargetsExistingColumn (ISS-181): the DO-block migration is not DDL the
// replay above parses, so assert directly that the table/column it indexes exist in the
// replayed schema and that it never touches rows (no DELETE/UPDATE/INSERT).
func TestMigration035TargetsExistingColumn(t *testing.T) {
	sc := loadSchema(t)
	if !sc["users"]["email"] {
		t.Fatal("users.email missing from replayed schema")
	}
	b, err := os.ReadFile("../../migrations/035_users_email_lower_unique.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(stripComments(string(b)))
	for _, bad := range []string{"delete ", "update ", "insert ", "drop ", "truncate"} {
		if strings.Contains(sql, bad) {
			t.Errorf("migration 035 must not contain %q", bad)
		}
	}
	for _, want := range []string{"from users", "on users (lower(email))", "create unique index if not exists", "raise notice"} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration 035 missing %q", want)
		}
	}
}
