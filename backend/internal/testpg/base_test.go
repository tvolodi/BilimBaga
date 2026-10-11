package testpg

// #576 (retro-012 decision 4): Base skips a real-Postgres test when TEST_DATABASE_URL is unset and fails it when
// REQUIRE_TEST_DB=1, and no Go file outside this package reads TEST_DATABASE_URL itself. The Base tests run the test binary
// as a child process, because Base's outcome is a test's skip or failure.

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseHelperEnv = "TESTPG_BASE_HELPER"

// TestBase_Helper is the child process of the tests below. In a normal run it skips.
func TestBase_Helper(t *testing.T) {
	if os.Getenv(baseHelperEnv) != "1" {
		t.Skip("child process of the Base tests only")
	}
	Base(t)
}

// runBaseHelper runs TestBase_Helper in a child process with the two variables Base reads set as given.
func runBaseHelper(t *testing.T, extra ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestBase_Helper$", "-test.v")
	cmd.Env = append(childEnv(), append(extra, baseHelperEnv+"=1")...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// childEnv is the environment without the two variables Base reads, so each child sets its own.
func childEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "TEST_DATABASE_URL=") || strings.HasPrefix(kv, "REQUIRE_TEST_DB=") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

func TestBase_SkipsWhenTheURLIsUnset(t *testing.T) {
	out, err := runBaseHelper(t)
	require.NoError(t, err, out)
	assert.Contains(t, out, "--- SKIP: TestBase_Helper")
}

func TestBase_FailsWhenRequiredAndUnset(t *testing.T) {
	out, err := runBaseHelper(t, "REQUIRE_TEST_DB=1")
	require.Error(t, err, out)
	assert.Contains(t, out, "--- FAIL: TestBase_Helper")
	assert.Contains(t, out, "REQUIRE_TEST_DB=1")
}

func TestBase_RunsWhenTheURLIsSet(t *testing.T) {
	out, err := runBaseHelper(t, "TEST_DATABASE_URL=postgres://ci:ci@localhost:1/none?sslmode=disable", "REQUIRE_TEST_DB=1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "--- PASS: TestBase_Helper")
}

// TestRealPostgresTestsTakeTheURLFromBase: no Go file outside this package reads TEST_DATABASE_URL itself. Each real-Postgres
// test takes the URL from Base, so it cannot skip silently under REQUIRE_TEST_DB=1.
func TestRealPostgresTestsTakeTheURLFromBase(t *testing.T) {
	const direct = `Getenv("TEST_DATABASE_URL")`
	var offenders []string
	err := filepath.WalkDir(filepath.Join("..", ".."), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testpg" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), direct) {
			offenders = append(offenders, filepath.ToSlash(p))
		}
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, offenders, "read the URL with testpg.Base(t), not os.Getenv")
}
