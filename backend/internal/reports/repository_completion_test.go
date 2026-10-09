package reports

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var completionCols = []string{"exam_id", "title", "assigned_count", "completed_count", "passed_count"}

var innerJoinAssignments = regexp.MustCompile(`(?m)^\s*JOIN resolved_assignments`)

// Regression for issue #38 (FR-BB51 AC-2): an active exam without assignments
// must be returned with zero counts rather than dropped.
func TestGetCompletionRateByExam_ExamWithoutAssignmentsReturnedWithZeros(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(completionCols, [][]driver.Value{
		{"exam-assigned", "Assigned Exam", int64(10), int64(5), int64(4)},
		{"exam-empty", "Unassigned Exam", int64(0), int64(0), int64(0)},
	})

	got, err := NewRepository(db).GetCompletionRateByExam(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "exam-empty", got[1].ExamID)
	assert.Equal(t, 0, got[1].AssignedCount)
	assert.Equal(t, 0, got[1].CompletedCount)
	assert.Equal(t, 0, got[1].PassedCount)

	require.Len(t, f.queries, 1)
	q := f.queries[0]
	assert.Contains(t, q, "LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id")
	assert.False(t, innerJoinAssignments.MatchString(q), "resolved_assignments must not be inner-joined")
	// Active filter stays on the base table (exams), never on the joined side.
	assert.Contains(t, q, "WHERE e.status = 'active'")
}

func TestGetDashboardCompletionRatesForRange_ExamWithoutAssignmentsReturnedWithZeros(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(completionCols, [][]driver.Value{
		{"exam-empty", "Unassigned Exam", int64(0), int64(0), int64(0)},
	})

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := NewRepository(db).GetDashboardCompletionRatesForRange(context.Background(), "tenant-1", from, from.AddDate(0, 1, 0))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 0, got[0].AssignedCount)

	q := f.queries[0]
	assert.Contains(t, q, "LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id")
	assert.False(t, innerJoinAssignments.MatchString(q))
	// Tenant/active filters on exams (base table); date filter stays in the ON clause of the sessions join.
	where := q[strings.Index(q, "WHERE e.tenant_id"):]
	assert.Contains(t, where, "e.status = 'active'")
	assert.NotContains(t, where, "submitted_at")
}

func TestGetCompletionRateByExam_NoExamsReturnsEmptySlice(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(completionCols, nil)
	got, err := NewRepository(db).GetCompletionRateByExam(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// Service passes an unassigned exam through unchanged (no division, no filtering).
func TestGetDashboardMetrics_UnassignedExamPassesThrough(t *testing.T) {
	repo := &mockRepo{}
	repo.getCompletionFn = func(context.Context) ([]*ExamCompletionRate, error) {
		return []*ExamCompletionRate{{ExamID: "e1", Title: "Empty"}}, nil
	}
	m, err := NewService(repo).GetDashboardMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, m.CompletionRateByExam, 1)
	assert.Equal(t, 0, m.CompletionRateByExam[0].AssignedCount)
}

// Handler serialises zero counts explicitly (not omitted).
func TestGetDashboard_200_UnassignedExamZeroCounts(t *testing.T) {
	svc := &mockSvc{
		getDashboardFn: func(_ context.Context) (*DashboardMetrics, error) {
			m := buildFullMetrics()
			m.CompletionRateByExam = []*ExamCompletionRate{{ExamID: "e1", Title: "Empty"}}
			return m, nil
		},
	}
	w := httptest.NewRecorder()
	NewHandler(svc, nil).GetDashboard(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Data struct {
			C []map[string]any `json:"completion_rate_by_exam"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Len(t, env.Data.C, 1)
	for _, k := range []string{"assigned_count", "completed_count", "passed_count"} {
		assert.EqualValues(t, 0, env.Data.C[0][k], k)
	}
}
