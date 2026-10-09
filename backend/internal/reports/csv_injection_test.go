package reports

import (
	"context"
	"database/sql/driver"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseCSV(t *testing.T, body string) [][]string {
	t.Helper()
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return recs
}

// ISS-191: results CSV neutralises formula injection in text cells and keeps
// numeric cells (including a negative question score) untouched.
func TestStreamExamResultsCSV_FormulaInjectionGuard(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(
		[]string{"session_id", "employee_name", "department", "started_at", "submitted_at", "score_pct", "passed", "time_taken_seconds"},
		[][]driver.Value{
			{"s1", `=HYPERLINK("http://evil","x")`, "+1", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 84.5, true, int64(1800)},
			{"s2", "\t=1", "\r=1", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 10.0, false, int64(60)},
			{"s3", "Жанар Әлиева", "", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 50.0, true, int64(60)},
		},
	)
	neg := -1.0
	repo := &mockRepo{
		getExamQuestionsFn: func(_ context.Context, _, _ string) ([]ExamQuestion, error) {
			return []ExamQuestion{{QuestionID: "q1", Position: 1}}, nil
		},
		streamExamResultSessionsFn: func(ctx context.Context, _, _ string) (*sqlx.Rows, error) {
			return db.QueryxContext(ctx, "SELECT 1")
		},
		getSessionQuestionScoresFn: func(_ context.Context, _ []string) ([]QuestionScore, error) {
			return []QuestionScore{{SessionID: "s1", QuestionID: "q1", Score: &neg}}, nil
		},
	}
	w := httptest.NewRecorder()
	require.NoError(t, NewService(repo).StreamExamResultsCSV(context.Background(), w, "e", "t"))

	recs := parseCSV(t, w.Body.String())
	require.Len(t, recs, 4)
	assert.Equal(t, `'=HYPERLINK("http://evil","x")`, recs[1][0])
	assert.Equal(t, "'+1", recs[1][1])
	assert.Equal(t, "'\t=1", recs[2][0])
	assert.Equal(t, "'\r=1", recs[2][1])
	assert.Equal(t, "Жанар Әлиева", recs[3][0])
	assert.Equal(t, "", recs[3][1])
	// numeric cells untouched
	assert.Equal(t, "84.50", recs[1][4])
	assert.Equal(t, "1800", recs[1][6])
	assert.Equal(t, "-1", recs[1][7])
}

// ISS-191: user-record CSV guards exam_title and status.
func TestStreamUserRecordCSV_FormulaInjectionGuard(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(
		[]string{"exam_title", "started_at", "submitted_at", "score_pct", "passed", "time_taken_seconds", "status"},
		[][]driver.Value{
			{"-2+3", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 70.0, true, int64(5), "submitted"},
			{"@SUM(A1)", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 70.0, true, int64(5), "=cmd"},
			{"Safety 101", "2026-05-14T10:00:00Z", "2026-05-14T10:30:00Z", 70.0, true, int64(5), "auto_submitted"},
		},
	)
	repo := &mockRepo{
		streamUserRecordSessionsFn: func(ctx context.Context, _, _ string) (*sqlx.Rows, error) {
			return db.QueryxContext(ctx, "SELECT 1")
		},
	}
	w := httptest.NewRecorder()
	require.NoError(t, NewService(repo).StreamUserRecordCSV(context.Background(), w, "u", "t"))

	recs := parseCSV(t, w.Body.String())
	require.Len(t, recs, 4)
	assert.Equal(t, "'-2+3", recs[1][0])
	assert.Equal(t, "'@SUM(A1)", recs[2][0])
	assert.Equal(t, "'=cmd", recs[2][6])
	assert.Equal(t, "Safety 101", recs[3][0])
	assert.Equal(t, "70.00", recs[1][3])
}

// ISS-178: unknown exam id -> ErrNotFound, nothing written.
func TestStreamExamResultsCSV_UnknownExam_ErrNotFound(t *testing.T) {
	repo := &mockRepo{
		getExamTitleFn: func(_ context.Context, _ string) (string, error) { return "", ErrNotFound },
	}
	w := httptest.NewRecorder()
	err := NewService(repo).StreamExamResultsCSV(context.Background(), w, "missing", "t")
	require.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, w.Body.String())
}

// ISS-178: handler maps ErrNotFound to 404 NOT_FOUND with no CSV headers.
func TestExamResultsCSV_UnknownExam_404(t *testing.T) {
	svc := &mockSvc{
		streamExamResultsCSVFn: func(_ context.Context, _ http.ResponseWriter, _, _ string) error { return ErrNotFound },
	}
	h := NewHandler(svc, nil)
	req := newRequestWithID(http.MethodGet, "/api/v1/admin/exams/x/results/export", "x")
	w := httptest.NewRecorder()
	h.ExamResultsCSV(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"NOT_FOUND"`)
	assert.NotContains(t, w.Header().Get("Content-Type"), "text/csv")
}

// ISS-178: results export statuses match the analytics set
// (submitted, auto_submitted, grading_pending).
func TestResultsExportQueries_IncludeAutoSubmitted(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"question_id", "position"}, nil)
	f.queue([]string{"employee_name"}, nil)
	repo := NewRepository(db)

	_, err := repo.GetExamQuestions(context.Background(), "e", "t")
	require.NoError(t, err)
	rows, err := repo.StreamExamResultSessions(context.Background(), "e", "t")
	require.NoError(t, err)
	rows.Close()

	require.Len(t, f.queries, 2)
	for _, q := range f.queries {
		assert.Contains(t, q, "'submitted', 'auto_submitted', 'grading_pending'")
	}
}
