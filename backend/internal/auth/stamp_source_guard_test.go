package auth

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #457: every write of users.password_changed_at in non-test code must take its value from a
// parameter that NextPasswordStamp (or lockAndStamp, which calls it under the row lock) computed from
// the application clock. A literal such as now() puts the database clock, or a stamp that ignores the
// previous one, back into the epoch check (#455). This guard reads the sources, so a new writer that
// bypasses the rule fails the build.

// stampAssignment matches an assignment to password_changed_at in SQL and captures its right-hand side.
var stampAssignment = regexp.MustCompile(`password_changed_at\s*=\s*([^\s,]+)`)

// stampRuleCall is the Go call that every stamping file must use.
const stampRuleCall = "NextPasswordStamp("

// stampLockCall is the locked helper in package auth, which calls NextPasswordStamp.
const stampLockCall = "lockAndStamp("

// stampWriteViolations returns the reasons src breaks the stamp rule, or nil when it follows it.
func stampWriteViolations(name, src string) []string {
	var out []string
	writes := stampAssignment.FindAllStringSubmatch(src, -1)
	for _, w := range writes {
		if !strings.HasPrefix(w[1], "$") {
			out = append(out, name+": password_changed_at is written from "+w[1]+", not from a parameter computed by NextPasswordStamp")
		}
	}
	if len(writes) > 0 && !strings.Contains(src, stampRuleCall) && !strings.Contains(src, stampLockCall) {
		out = append(out, name+": writes password_changed_at without NextPasswordStamp (or lockAndStamp)")
	}
	return out
}

// sqlLiteral matches a Go raw string literal, where the SQL of this package lives.
var sqlLiteral = regexp.MustCompile("`([^`]*)`")

// forcedChangeViolations flags every SQL statement that sets force_password_change = true without also
// stamping password_changed_at (#466). A forced change is a session-ending event, so it must revoke
// earlier access tokens like every other password writer does.
func forcedChangeViolations(name, src string) []string {
	var out []string
	for _, m := range sqlLiteral.FindAllStringSubmatch(src, -1) {
		q := m[1]
		if strings.Contains(q, "force_password_change = true") && !regexp.MustCompile(`password_changed_at\s*=`).MatchString(q) {
			out = append(out, name+": sets force_password_change = true without stamping password_changed_at")
		}
	}
	return out
}

// TestStampGuard_FixtureForcedChangeWithoutStampIsFlagged shows the guard sees the #466 write.
func TestStampGuard_FixtureForcedChangeWithoutStampIsFlagged(t *testing.T) {
	fixture := "const q = `UPDATE users SET force_password_change = true, updated_at = now() WHERE id = $1`"
	assert.NotEmpty(t, forcedChangeViolations("fixture.go", fixture), "a forced change that does not stamp must be flagged")
}

// TestStampGuard_FixtureForcedChangeWithStampIsAccepted shows a stamped forced change passes.
func TestStampGuard_FixtureForcedChangeWithStampIsAccepted(t *testing.T) {
	fixture := "const q = `UPDATE users SET force_password_change = true, password_changed_at = $3 WHERE id = $1`"
	assert.Empty(t, forcedChangeViolations("fixture.go", fixture))
}

// TestStampGuard_FixtureWithDatabaseClockIsFlagged shows the guard catches the case it exists for.
func TestStampGuard_FixtureWithDatabaseClockIsFlagged(t *testing.T) {
	fixture := "const q = `UPDATE users SET password_changed_at = now() WHERE id = $1`"
	assert.NotEmpty(t, stampWriteViolations("fixture.go", fixture), "a now() stamp must be flagged")
}

// TestStampGuard_FixtureWithParameterAndRuleIsAccepted shows a conforming writer passes.
func TestStampGuard_FixtureWithParameterAndRuleIsAccepted(t *testing.T) {
	fixture := "stamp := NextPasswordStamp(now, prev)\nconst q = `UPDATE users SET password_changed_at = $3 WHERE id = $2`"
	assert.Empty(t, stampWriteViolations("fixture.go", fixture))
}

// TestStampGuard_NoNonTestWriterBypassesTheRule scans the non-test sources of internal/auth and internal/users.
func TestStampGuard_NoNonTestWriterBypassesTheRule(t *testing.T) {
	var violations []string
	scanned := 0
	for _, dir := range []string{".", filepath.Join("..", "users")} {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			scanned++
			violations = append(violations, stampWriteViolations(filepath.Join(dir, name), string(b))...)
			violations = append(violations, forcedChangeViolations(filepath.Join(dir, name), string(b))...)
		}
	}
	require.NotZero(t, scanned, "the guard must scan real sources")
	assert.Empty(t, violations, "every password_changed_at write must use NextPasswordStamp")
}
