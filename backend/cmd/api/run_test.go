package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type fakeMigrator struct {
	calls int
	err   error
}

func (f *fakeMigrator) Migrate() error { f.calls++; return f.err }

func runWith(args []string, m *fakeMigrator) (code, served int, out, errOut string) {
	var o, e bytes.Buffer
	code = run(args, deps{serve: func() { served++ }, migrator: m, stdout: &o, stderr: &e})
	return code, served, o.String(), e.String()
}

func TestRun_NoArgsServes(t *testing.T) {
	m := &fakeMigrator{}
	code, served, _, _ := runWith(nil, m)
	if code != 0 || served != 1 || m.calls != 0 {
		t.Fatalf("code=%d served=%d migrate calls=%d", code, served, m.calls)
	}
}

func TestRun_MigrateDoesNotServe(t *testing.T) {
	m := &fakeMigrator{}
	code, served, out, _ := runWith([]string{"migrate"}, m)
	if code != 0 || served != 0 || m.calls != 1 {
		t.Fatalf("code=%d served=%d migrate calls=%d", code, served, m.calls)
	}
	if !strings.Contains(out, "migrations applied") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestRun_MigrateErrorNonZero(t *testing.T) {
	m := &fakeMigrator{err: errors.New("boom")}
	code, served, out, errOut := runWith([]string{"migrate"}, m)
	if code != 1 || served != 0 {
		t.Fatalf("code=%d served=%d", code, served)
	}
	if !strings.Contains(errOut, "boom") || strings.Contains(out, "migrations applied") {
		t.Fatalf("stdout=%q stderr=%q", out, errOut)
	}
}

func TestRun_BadArgsUsage(t *testing.T) {
	for _, args := range [][]string{{"bogus"}, {"migrate", "extra"}, {"--help"}} {
		m := &fakeMigrator{}
		code, served, _, errOut := runWith(args, m)
		if code != 2 || served != 0 || m.calls != 0 || !strings.Contains(errOut, "usage:") {
			t.Fatalf("args=%v code=%d served=%d calls=%d stderr=%q", args, code, served, m.calls, errOut)
		}
	}
}
