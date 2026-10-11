package api

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #530: the sweep found no open-work markers in backend code. Unfinished work is an issue, not a comment, so this test
// keeps the count at zero. The words are split in the pattern so this file does not match itself.
var openWorkMarker = regexp.MustCompile(`\b(T` + `ODO|FIX` + `ME)\b`)

func TestMarkerGuard_BackendHasNoOpenWorkMarkers(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..")) // backend/
	require.NoError(t, err)
	var hits []string
	scanned := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		rel, _ := filepath.Rel(root, path)
		for i, line := range strings.Split(string(raw), "\n") {
			if openWorkMarker.MatchString(line) {
				hits = append(hits, fmt.Sprintf("%s:%d", filepath.ToSlash(rel), i+1))
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotZero(t, scanned, "the guard must scan real sources")
	assert.Empty(t, hits, "remove the marker and file an issue for the work it names")
}
