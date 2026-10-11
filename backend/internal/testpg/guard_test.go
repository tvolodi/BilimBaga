package testpg

// #549: guards that keep the schemaguard allow-list entry for this package narrow, and keep real-Postgres migration
// tests on NewSchema. They read source files only, so they need no TEST_DATABASE_URL.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const testpgImport = "github.com/bilimbaga/bilimbaga/internal/testpg"

// allowedSQL is every statement the non-test sources of this package may issue. A keyword-bearing string literal
// outside this set fails TestTestpgSourcesIssueOnlyAllowedSQL.
var allowedSQL = map[string]bool{
	"CREATE SCHEMA ":         true,
	"DROP SCHEMA IF EXISTS ": true,
	"CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public": true,
	"SELECT pg_advisory_lock($1)":                               true,
	"SELECT pg_advisory_unlock($1)":                             true,
}

var sqlKeywordRe = regexp.MustCompile(`\b(CREATE|DROP|SELECT|INSERT|UPDATE|DELETE|ALTER|GRANT|TRUNCATE)\b`)

// walkBackendGoFiles calls fn for every .go file under backend/ (this package's ../..), parsed with mode.
func walkBackendGoFiles(t *testing.T, mode parser.Mode, fn func(path string, f *ast.File)) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join("..", ".."), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, mode)
		if err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		fn(filepath.ToSlash(p), f)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestTestpgSourcesIssueOnlyAllowedSQL: the non-test sources of this package issue exactly the statements in
// allowedSQL. Fragments without a SQL keyword are not statements and are not compared.
func TestTestpgSourcesIssueOnlyAllowedSQL(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	found := map[string]bool{}
	for _, p := range files {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatalf("unquote in %s: %v", p, err)
			}
			if sqlKeywordRe.MatchString(s) {
				found[s] = true
			}
			return true
		})
	}
	for s := range found {
		if !allowedSQL[s] {
			t.Errorf("%q is a statement outside allowedSQL; review it against the schemaguard allow-list entry (#549)", s)
		}
	}
	for s := range allowedSQL {
		if !found[s] {
			t.Errorf("allowedSQL entry %q is no longer issued; remove it (#549)", s)
		}
	}
}

// TestNoProductionFileImportsTestpg: only _test.go files may import this package.
func TestNoProductionFileImportsTestpg(t *testing.T) {
	walkBackendGoFiles(t, parser.ImportsOnly, func(p string, f *ast.File) {
		if strings.HasSuffix(p, "_test.go") {
			return
		}
		for _, imp := range f.Imports {
			if path, err := strconv.Unquote(imp.Path.Value); err == nil && path == testpgImport {
				t.Errorf("%s imports %s; the helper is for tests only", p, testpgImport)
			}
		}
	})
}

// TestRealPostgresMigrationTestsUseNewSchema: a test that reads TEST_DATABASE_URL and runs the migrations must call
// testpg.NewSchema. Running the migrations means calling RunMigrations or applyUpMigrations, or selecting *.up.sql
// files by suffix. Tests that use a fake driver or parse the SQL text do not read TEST_DATABASE_URL and are not checked.
func TestRealPostgresMigrationTestsUseNewSchema(t *testing.T) {
	walkBackendGoFiles(t, parser.ParseComments, func(p string, f *ast.File) {
		if !strings.HasSuffix(p, "_test.go") {
			return
		}
		readsDB, runsMigrations, usesHelper := false, false, false
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				if x.Kind == token.STRING && x.Value == `"TEST_DATABASE_URL"` {
					readsDB = true
				}
			case *ast.CallExpr:
				switch fn := x.Fun.(type) {
				case *ast.Ident:
					if fn.Name == "applyUpMigrations" {
						runsMigrations = true
					}
				case *ast.SelectorExpr:
					if fn.Sel.Name == "RunMigrations" {
						runsMigrations = true
					}
					if fn.Sel.Name == "HasSuffix" && hasUpSQLArg(x) {
						runsMigrations = true
					}
					if id, ok := fn.X.(*ast.Ident); ok && id.Name == "testpg" && fn.Sel.Name == "NewSchema" {
						usesHelper = true
					}
				}
			}
			return true
		})
		if readsDB && runsMigrations && !usesHelper {
			t.Errorf("%s runs the migrations against TEST_DATABASE_URL but does not call testpg.NewSchema (#549)", p)
		}
	})
}

// hasUpSQLArg reports whether a call takes the ".up.sql" literal as an argument.
func hasUpSQLArg(call *ast.CallExpr) bool {
	for _, a := range call.Args {
		if lit, ok := a.(*ast.BasicLit); ok && lit.Kind == token.STRING && lit.Value == `".up.sql"` {
			return true
		}
	}
	return false
}
