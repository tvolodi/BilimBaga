package exams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FR-BB73 AC-22: the AI grading mode of an exam through create and update.

func TestCreate_AIGradingModeDefaultsToSuggest(t *testing.T) {
	var got string
	svc := &mockSvc{createExamFn: func(_ context.Context, in CreateExamInput) (*Exam, error) {
		got = in.AIGradingMode
		return &Exam{ID: "exam-1", Title: in.Title, Status: "draft"}, nil
	}}
	body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	w := httptest.NewRecorder()
	newHandler(svc).Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "suggest", got, "an exam created without a mode takes the default (AC-9)")
}

func TestCreate_AIGradingModeIsPassedThrough(t *testing.T) {
	for _, mode := range []string{"off", "suggest", "auto"} {
		t.Run(mode, func(t *testing.T) {
			var got string
			svc := &mockSvc{createExamFn: func(_ context.Context, in CreateExamInput) (*Exam, error) {
				got = in.AIGradingMode
				return &Exam{ID: "exam-1", Title: in.Title, Status: "draft"}, nil
			}}
			body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"ai_grading_mode":"` + mode + `"}`
			w := httptest.NewRecorder()
			newHandler(svc).Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
			require.Equal(t, http.StatusCreated, w.Code)
			assert.Equal(t, mode, got)
		})
	}
}

func TestCreate_UnknownAIGradingModeIs422(t *testing.T) {
	body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"ai_grading_mode":"always"}`
	w := httptest.NewRecorder()
	newHandler(&mockSvc{}).Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/exams", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "ERR_VALIDATION", decode(t, w).Error.Code)
}

// Omitted on update: the input carries an empty mode, which the repository reads as "unchanged".
func TestUpdate_AIGradingModeOmittedIsEmpty(t *testing.T) {
	var got string
	svc := &mockSvc{updateExamFn: func(_ context.Context, _ string, in UpdateExamInput) (*Exam, error) {
		got = in.AIGradingMode
		return &Exam{ID: "exam-1", Status: "draft"}, nil
	}}
	body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "exam-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	newHandler(svc).Update(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", got)
}

func TestUpdate_AIGradingModeIsPassedThroughAndValidated(t *testing.T) {
	var got string
	svc := &mockSvc{updateExamFn: func(_ context.Context, _ string, in UpdateExamInput) (*Exam, error) {
		got = in.AIGradingMode
		return &Exam{ID: "exam-1", Status: "draft"}, nil
	}}
	body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"ai_grading_mode":"auto"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "exam-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	newHandler(svc).Update(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "auto", got)

	bad := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"ai_grading_mode":"always"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(bad))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	newHandler(svc).Update(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
