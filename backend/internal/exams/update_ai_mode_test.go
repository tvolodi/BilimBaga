package exams

// FR-BB73 AC-9 (#566): ai_grading_mode may change in every exam status. Any other field of a non-draft exam keeps the
// draft rule (EXAM_NOT_DRAFT), and a refused request changes nothing.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modeOnlyRequest repeats every field of the stored exam and changes only its AI grading mode.
func modeOnlyRequest(e *Exam, mode string) UpdateExamInput {
	return UpdateExamInput{
		Title:              e.Title,
		Description:        e.Description,
		TimeLimitMinutes:   e.TimeLimitMinutes,
		PassingScorePct:    e.PassingScorePct,
		MaxAttempts:        e.MaxAttempts,
		AvailableFrom:      e.AvailableFrom,
		AvailableUntil:     e.AvailableUntil,
		ShuffleQuestions:   e.ShuffleQuestions,
		ShuffleOptions:     e.ShuffleOptions,
		ShowAnswers:        e.ShowAnswers,
		OnTabSwitch:        e.OnTabSwitch,
		AIGradingMode:      mode,
		CertificateEnabled: e.CertificateEnabled,
		Adaptive:           e.Adaptive,
	}
}

func seedExamWithMode(repo *mockRepo, id, status, mode string) {
	seedExam(repo, id, status)
	repo.exams[id].AIGradingMode = mode
}

func TestUpdateExam_PublishedAndArchivedModeChangeIsAccepted(t *testing.T) {
	for _, status := range []string{"active", "archived"} {
		t.Run(status, func(t *testing.T) {
			repo := newMockRepo()
			seedExamWithMode(repo, "exam-1", status, "auto")
			repo.updateFn = func(context.Context, string, UpdateExamInput) (*Exam, error) {
				t.Fatal("the full update must not run for a mode-only change")
				return nil, nil
			}

			res, err := NewService(repo).UpdateExam(context.Background(), "exam-1", modeOnlyRequest(repo.exams["exam-1"], "off"))

			require.NoError(t, err)
			assert.Equal(t, "off", res.Exam.AIGradingMode)
			assert.Equal(t, "auto", res.PreviousAIGradingMode, "the audit row needs the mode before the change")
			assert.Equal(t, "off", repo.exams["exam-1"].AIGradingMode)
		})
	}
}

// Every other field, changed on its own together with a mode change, keeps the draft rule and changes nothing.
func TestUpdateExam_PublishedOtherFieldsKeepTheDraftRule(t *testing.T) {
	later := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	cases := map[string]func(in *UpdateExamInput){
		"title":             func(in *UpdateExamInput) { in.Title = "Renamed" },
		"time limit":        func(in *UpdateExamInput) { in.TimeLimitMinutes = 45 },
		"passing score":     func(in *UpdateExamInput) { in.PassingScorePct = 80 },
		"max attempts":      func(in *UpdateExamInput) { in.MaxAttempts = 2 },
		"available from":    func(in *UpdateExamInput) { in.AvailableFrom = &later },
		"available until":   func(in *UpdateExamInput) { in.AvailableUntil = &later },
		"shuffle questions": func(in *UpdateExamInput) { in.ShuffleQuestions = true },
		"shuffle options":   func(in *UpdateExamInput) { in.ShuffleOptions = true },
		"show answers":      func(in *UpdateExamInput) { in.ShowAnswers = "after_completion" },
		"on tab switch":     func(in *UpdateExamInput) { in.OnTabSwitch = "warn" },
		"certificate":       func(in *UpdateExamInput) { in.CertificateEnabled = true },
		"adaptive":          func(in *UpdateExamInput) { in.Adaptive = true },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			repo := newMockRepo()
			seedExamWithMode(repo, "exam-1", "active", "suggest")
			in := modeOnlyRequest(repo.exams["exam-1"], "auto")
			change(&in)

			_, err := NewService(repo).UpdateExam(context.Background(), "exam-1", in)

			assert.ErrorIs(t, err, ErrNotDraft)
			assert.Equal(t, "suggest", repo.exams["exam-1"].AIGradingMode, "a refused request changes no field")
			assert.Equal(t, "Test Exam", repo.exams["exam-1"].Title)
		})
	}
}

// An empty mode in the request keeps the stored one (AC-22), so a request that only repeats the exam changes nothing.
func TestUpdateExam_PublishedEmptyModeKeepsTheStoredMode(t *testing.T) {
	repo := newMockRepo()
	seedExamWithMode(repo, "exam-1", "active", "suggest")

	res, err := NewService(repo).UpdateExam(context.Background(), "exam-1", modeOnlyRequest(repo.exams["exam-1"], ""))

	require.NoError(t, err)
	assert.Equal(t, "suggest", res.Exam.AIGradingMode)
	assert.Equal(t, "suggest", res.PreviousAIGradingMode)
}

// The same instant written in another zone is the same value, so it is not a change to a published exam.
func TestUpdateExam_PublishedSameInstantInAnotherZoneIsNotAChange(t *testing.T) {
	repo := newMockRepo()
	seedExamWithMode(repo, "exam-1", "active", "suggest")
	until := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	repo.exams["exam-1"].AvailableUntil = &until
	in := modeOnlyRequest(repo.exams["exam-1"], "auto")
	sameInstant := until.In(time.FixedZone("UTC+5", 5*60*60))
	in.AvailableUntil = &sameInstant

	res, err := NewService(repo).UpdateExam(context.Background(), "exam-1", in)

	require.NoError(t, err)
	assert.Equal(t, "auto", res.Exam.AIGradingMode)
}

// A draft exam keeps the full update: the mode change goes through the same write as every other field.
func TestUpdateExam_DraftModeChangeUsesTheFullUpdate(t *testing.T) {
	repo := newMockRepo()
	seedExamWithMode(repo, "exam-1", "draft", "suggest")
	fullUpdate := false
	repo.updateFn = func(_ context.Context, id string, in UpdateExamInput) (*Exam, error) {
		fullUpdate = true
		e := repo.exams[id]
		e.AIGradingMode = in.AIGradingMode
		return e, nil
	}

	res, err := NewService(repo).UpdateExam(context.Background(), "exam-1", modeOnlyRequest(repo.exams["exam-1"], "auto"))

	require.NoError(t, err)
	assert.True(t, fullUpdate)
	assert.Equal(t, "auto", res.Exam.AIGradingMode)
	assert.Equal(t, "suggest", res.PreviousAIGradingMode)
	assert.True(t, res.Changed, "a draft update is always a change for the audit log, as before #566")
}

// #570: a description sent with a mode change on a published exam is compared like every other field. A changed one is
// refused; the same text, or no text on both sides, is not a change.
func TestUpdateExam_PublishedDescriptionIsCompared(t *testing.T) {
	old, same, other := "Old text", "Old text", "New text"
	cases := []struct {
		name    string
		stored  *string
		sent    *string
		refused bool
	}{
		{"changed text", &old, &other, true},
		{"same text", &old, &same, false},
		{"text added", nil, &other, true},
		{"text removed", &old, nil, true},
		{"empty on both sides", nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepo()
			seedExamWithMode(repo, "exam-1", "active", "suggest")
			repo.exams["exam-1"].Description = tc.stored
			in := modeOnlyRequest(repo.exams["exam-1"], "auto")
			in.Description = tc.sent

			_, err := NewService(repo).UpdateExam(context.Background(), "exam-1", in)

			if tc.refused {
				assert.ErrorIs(t, err, ErrNotDraft)
				assert.Equal(t, "suggest", repo.exams["exam-1"].AIGradingMode, "a refused request changes no field")
			} else {
				require.NoError(t, err)
				assert.Equal(t, "auto", repo.exams["exam-1"].AIGradingMode)
			}
		})
	}
}

// #570: the comparison covers every UpdateExamInput field except the mode. A field added later and left out of
// onlyAIGradingModeChanged would let a published exam change silently, so this walks the struct and fails for it.
func TestUpdateExam_EveryFieldButTheModeIsCompared(t *testing.T) {
	typ := reflect.TypeOf(UpdateExamInput{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if name == "AIGradingMode" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			repo := newMockRepo()
			seedExamWithMode(repo, "exam-1", "active", "suggest")
			in := modeOnlyRequest(repo.exams["exam-1"], "auto")
			changeField(t, reflect.ValueOf(&in).Elem().Field(i))

			_, err := NewService(repo).UpdateExam(context.Background(), "exam-1", in)

			assert.ErrorIs(t, err, ErrNotDraft, "%s changed on a published exam must keep the draft rule", name)
			assert.Equal(t, "suggest", repo.exams["exam-1"].AIGradingMode)
		})
	}
}

// changeField sets a field to a value different from any stored exam's, by kind.
func changeField(t *testing.T, f reflect.Value) {
	t.Helper()
	switch f.Kind() {
	case reflect.String:
		f.SetString(f.String() + "-changed")
	case reflect.Bool:
		f.SetBool(!f.Bool())
	case reflect.Int:
		f.SetInt(f.Int() + 1)
	case reflect.Float64:
		f.SetFloat(f.Float() + 1)
	case reflect.Ptr:
		switch f.Type().Elem() {
		case reflect.TypeOf(""):
			s := "changed"
			f.Set(reflect.ValueOf(&s))
		case reflect.TypeOf(time.Time{}):
			later := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
			f.Set(reflect.ValueOf(&later))
		default:
			t.Fatalf("no change rule for pointer to %s: teach changeField the new kind", f.Type().Elem())
		}
	default:
		t.Fatalf("no change rule for kind %s: teach changeField the new kind", f.Kind())
	}
}

// #570: a request that repeats a published exam changes nothing, so the result is not a change and writes no audit row.
// A mode change is a change.
func TestUpdateExam_PublishedChangedFlag(t *testing.T) {
	repo := newMockRepo()
	seedExamWithMode(repo, "exam-1", "active", "suggest")

	same, err := NewService(repo).UpdateExam(context.Background(), "exam-1", modeOnlyRequest(repo.exams["exam-1"], "suggest"))
	require.NoError(t, err)
	assert.False(t, same.Changed, "a no-op on a published exam is not a change")

	moved, err := NewService(repo).UpdateExam(context.Background(), "exam-1", modeOnlyRequest(repo.exams["exam-1"], "auto"))
	require.NoError(t, err)
	assert.True(t, moved.Changed)
}

func TestUpdate_PublishedModeChangeAnswers200(t *testing.T) {
	svc := &mockSvc{updateExamFn: func(_ context.Context, _ string, in UpdateExamInput) (*Exam, error) {
		return &Exam{ID: "exam-1", Title: in.Title, Status: "active", AIGradingMode: in.AIGradingMode}, nil
	}}
	body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"ai_grading_mode":"off"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "exam-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	newHandler(svc).Update(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ai_grading_mode":"off"`)
}

// A value outside off, suggest and auto is a validation error, and the service is never reached.
func TestUpdate_UnknownAIGradingModeIs422BeforeTheService(t *testing.T) {
	svc := &mockSvc{updateExamFn: func(context.Context, string, UpdateExamInput) (*Exam, error) {
		t.Fatal("the service must not be reached for an invalid mode")
		return nil, nil
	}}
	body := `{"title":"Fire","time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"ai_grading_mode":"always"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/exams/exam-1", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "exam-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	newHandler(svc).Update(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
