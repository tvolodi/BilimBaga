package reports

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeSessionRows(n int) [][]driver.Value {
	rows := make([][]driver.Value, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, []driver.Value{
			fmt.Sprintf("s%d", i), "Emp", "Dept", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 50.0, true, int64(60),
		})
	}
	return rows
}

func capRepo(t *testing.T, n int) *mockRepo {
	t.Helper()
	db, f := newFakeDB(t)
	f.queue([]string{"session_id", "employee_name", "department", "started_at", "submitted_at", "score_pct", "passed", "time_taken_seconds"}, fakeSessionRows(n))
	return &mockRepo{
		streamExamResultSessionsFn: func(ctx context.Context, _, _ string) (*sqlx.Rows, error) {
			return db.QueryxContext(ctx, "SELECT 1")
		},
	}
}

// ISS-210: more sessions than the cap -> ErrExportTooLarge and nothing written.
func TestStreamExamResultsCSV_RowCapExceeded(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewService(capRepo(t, 6), WithMaxExportRows(5)).StreamExamResultsCSV(context.Background(), w, "e", "t")
	require.ErrorIs(t, err, ErrExportTooLarge)
}

// Exactly at the cap is allowed.
func TestStreamExamResultsCSV_RowCapBoundaryOK(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewService(capRepo(t, 5), WithMaxExportRows(5)).StreamExamResultsCSV(context.Background(), w, "e", "t")
	require.NoError(t, err)
	assert.Len(t, parseCSV(t, w.Body.String()), 6) // header + 5
}

func TestNewService_DefaultExportCap(t *testing.T) {
	s := NewService(&mockRepo{}).(*service)
	assert.Equal(t, DefaultMaxExportRows, s.maxExportRows)
	s = NewService(&mockRepo{}, WithMaxExportRows(0)).(*service)
	assert.Equal(t, DefaultMaxExportRows, s.maxExportRows)
}

// ISS-210: handler maps ErrExportTooLarge to 422 EXPORT_TOO_LARGE, no CSV headers.
func TestExamResultsCSV_422_TooLarge(t *testing.T) {
	h := NewHandler(&mockSvc{
		streamExamResultsCSVFn: func(context.Context, http.ResponseWriter, string, string) error {
			return fmt.Errorf("wrapped: %w", ErrExportTooLarge)
		},
	}, nil)
	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/exam-uuid/results/export", "exam-uuid")
	w := httptest.NewRecorder()
	h.ExamResultsCSV(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"EXPORT_TOO_LARGE"`)
	assert.Contains(t, w.Body.String(), "row limit")
	assert.NotContains(t, w.Header().Get("Content-Type"), "text/csv")
}
