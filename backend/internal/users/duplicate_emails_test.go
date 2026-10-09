package users

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDupStore struct {
	indexExists bool
	indexErr    error
	total       int
	groups      [][]string
	groupsErr   error
	auditErr    error
	panicOn     string

	groupCalls int
	limit      int
	audits     []map[string]any
}

func (f *fakeDupStore) EmailUniqueIndexExists(context.Context) (bool, error) {
	if f.panicOn == "index" {
		panic("boom")
	}
	return f.indexExists, f.indexErr
}

func (f *fakeDupStore) DuplicateEmailGroups(_ context.Context, limit int) (int, [][]string, error) {
	f.groupCalls++
	f.limit = limit
	if f.groupsErr != nil {
		return 0, nil, f.groupsErr
	}
	g := f.groups
	if len(g) > limit {
		g = g[:limit]
	}
	return f.total, g, nil
}

func (f *fakeDupStore) RecordDuplicateEmails(_ context.Context, m map[string]any) error {
	f.audits = append(f.audits, m)
	return f.auditErr
}

func capture() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

func TestCheckDuplicateEmails_IndexPresent_Silent(t *testing.T) {
	st := &fakeDupStore{indexExists: true, total: 5}
	lg, buf := capture()
	CheckDuplicateEmails(context.Background(), st, lg)
	assert.Empty(t, buf.String())
	assert.Zero(t, st.groupCalls, "no duplicate query when the index exists")
	assert.Empty(t, st.audits)
}

func TestCheckDuplicateEmails_NoDuplicates_Silent(t *testing.T) {
	st := &fakeDupStore{}
	lg, buf := capture()
	CheckDuplicateEmails(context.Background(), st, lg)
	assert.Empty(t, buf.String())
	assert.Empty(t, st.audits)
}

func TestCheckDuplicateEmails_Duplicates_OneWarnOneAudit(t *testing.T) {
	st := &fakeDupStore{total: 2, groups: [][]string{{"John@X.com", "john@x.com"}, {"A@b.kz", "a@b.kz"}}}
	lg, buf := capture()
	CheckDuplicateEmails(context.Background(), st, lg)

	out := buf.String()
	assert.Equal(t, 1, strings.Count(out, "level=WARN"), out)
	assert.Contains(t, out, "John@X.com")
	assert.Contains(t, out, "duplicate_groups=2")
	assert.Contains(t, out, "more_groups_not_listed=0")
	require.Len(t, st.audits, 1)
	assert.Equal(t, 2, st.audits[0]["duplicate_groups"])
	assert.Equal(t, maxReportedGroups, st.limit)
}

func TestCheckDuplicateEmails_CapsListAndAuditSample(t *testing.T) {
	var groups [][]string
	for i := 0; i < 80; i++ {
		groups = append(groups, []string{fmt.Sprintf("U%d@x.com", i), fmt.Sprintf("u%d@x.com", i)})
	}
	st := &fakeDupStore{total: 80, groups: groups}
	lg, buf := capture()
	CheckDuplicateEmails(context.Background(), st, lg)

	out := buf.String()
	assert.Contains(t, out, "listed_groups=50")
	assert.Contains(t, out, "more_groups_not_listed=30")
	assert.NotContains(t, out, "U60@x.com")
	require.Len(t, st.audits, 1)
	sample, ok := st.audits[0]["sample"].([][]string)
	require.True(t, ok)
	assert.Len(t, sample, maxAuditSampleGroups)
	assert.Equal(t, 80, st.audits[0]["duplicate_groups"])
}

func TestCheckDuplicateEmails_ErrorsNeverAbort(t *testing.T) {
	cases := map[string]*fakeDupStore{
		"index error": {indexErr: errors.New("db down")},
		"query error": {groupsErr: errors.New("syntax")},
		"audit error": {total: 1, groups: [][]string{{"A@x.com", "a@x.com"}}, auditErr: errors.New("insert failed")},
		"panic":       {panicOn: "index"},
	}
	for name, st := range cases {
		t.Run(name, func(t *testing.T) {
			lg, buf := capture()
			assert.NotPanics(t, func() { CheckDuplicateEmails(context.Background(), st, lg) })
			assert.Contains(t, buf.String(), "continuing startup")
		})
	}
}

func TestCheckDuplicateEmails_NilLogger(t *testing.T) {
	assert.NotPanics(t, func() { CheckDuplicateEmails(context.Background(), &fakeDupStore{}, nil) })
}
