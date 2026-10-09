package reports

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FR-BB510 AC-1 / ISS-061: overdue rows must expose exam_id so the dashboard can
// target a reminder at a specific exam.
func TestGetOverdueEmployees_ExposesExamID(t *testing.T) {
	db, f := newFakeDB(t)
	dl := time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)
	f.queue([]string{"user_id", "name", "exam_id", "exam_title", "deadline"}, [][]driver.Value{
		{"u1", "Alice", "e1", "Fire Safety", dl},
	})

	got, err := NewRepository(db).GetOverdueEmployees(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "e1", got[0].ExamID)
	assert.Contains(t, f.queries[0], "ra.exam_id           AS exam_id")

	raw, err := json.Marshal(got[0])
	require.NoError(t, err)
	assert.Contains(t, string(raw), "\"exam_id\":\"e1\"")
}
