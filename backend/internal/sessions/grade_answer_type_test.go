package sessions

// FR-BB42 AC-11 (#565): manual grading is for short-text rows only. A row of another type answers 422
// ANSWER_NOT_GRADABLE, and the handler maps that error to its code.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestHandleGradeAnswer_422_AnswerNotGradable(t *testing.T) {
	svc := &mockSvc{
		gradeAnswerFn: func(_ context.Context, _, _, _, _, _ string, _ GradeAnswerRequest) (*GradeAnswerResponse, error) {
			return nil, ErrAnswerNotGradable
		},
	}
	h := NewHandler(svc)

	bodyStr := `{"score_pct": 50, "feedback": ""}`
	req := httptest.NewRequest(http.MethodPost, "/admin/grading/s1/answers/q1", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxkeys.CtxTenantID, "tenant-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionId", "s1")
	rctx.URLParams.Add("questionId", "q1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.HandleGradeAnswer(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "ANSWER_NOT_GRADABLE", errObj["code"])
}

// The refusal comes from the check alone: after the session lock and the row read, no statement writes.
func TestGradeAnswer_NotShortTextRefusedBeforeAnyWrite(t *testing.T) {
	db, f := newFakeDB(t)
	queueGradeLock(f, "owner-1", "grading_pending", false)
	queueGradeRowOfType(f, "graded", 2, nil, nil, "single")

	_, err := NewRepository(db, nil).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "t", "", 0, "")

	assert.ErrorIs(t, err, ErrAnswerNotGradable)
	assert.Len(t, f.queries, 2, "only the session lock and the row read run")
	// The lock is a SELECT ... FOR UPDATE, so the test looks at the statement's leading verb.
	for _, q := range f.queries {
		verb := strings.Fields(q)[0]
		assert.Equal(t, "SELECT", verb, "no write statement runs: %q", q)
	}
}
