package users

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	rUser = "11111111-1111-4111-8111-111111111111"
	rExam = "22222222-2222-4222-8222-222222222222"
)

// ── mockRepo / mockUserService FR-BB510 methods ──────────────────────────────

func (m *mockRepo) ExamExists(_ context.Context, id string) (bool, error) { return m.exams[id], nil }
func (m *mockRepo) IsOverdueTarget(_ context.Context, u, e string) (bool, error) {
	return m.overdue[u+"|"+e], nil
}
func (m *mockRepo) LastReminderAt(context.Context, string, string) (*time.Time, error) {
	return m.lastRemind, nil
}
func (m *mockRepo) InsertReminder(_ context.Context, u, e, by string) error {
	if m.insertErr != nil {
		return m.insertErr
	}
	m.reminders = append(m.reminders, u+"|"+e+"|"+by)
	return nil
}

func (m *mockUserService) RemindEmployee(ctx context.Context, actorID, userID, examID string) (*RemindResult, error) {
	return m.remindFn(ctx, actorID, userID, examID)
}

type fakeReminder struct {
	calls int
	err   error
}

func (f *fakeReminder) SendOverdueReminder(context.Context, string, string) (*ReminderResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &ReminderResult{SentAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), ExamTitle: "Fire Safety"}, nil
}

func remindSvc(t *testing.T) (*service, *mockRepo, *fakeReminder) {
	t.Helper()
	repo := newMockRepo()
	repo.users[rUser] = sampleUser(rUser)
	repo.exams = map[string]bool{rExam: true}
	repo.overdue = map[string]bool{rUser + "|" + rExam: true}
	fr := &fakeReminder{}
	s := NewService(repo).(*service)
	s.reminder = fr
	s.now = func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }
	return s, repo, fr
}

// ── service ──────────────────────────────────────────────────────────────────

func TestRemindEmployee_Success_SendsAndRecords(t *testing.T) {
	s, repo, fr := remindSvc(t)
	res, err := s.RemindEmployee(context.Background(), "admin-1", rUser, rExam)
	require.NoError(t, err)
	assert.Equal(t, "Fire Safety", res.ExamTitle)
	assert.Equal(t, 1, fr.calls)
	assert.Equal(t, []string{rUser + "|" + rExam + "|admin-1"}, repo.reminders)
}

func TestRemindEmployee_Rejections_SendNothing(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*mockRepo)
		user   string
		exam   string
		want   error
	}{
		{"unknown user", nil, "nope", rExam, ErrNotFound},
		{"unknown exam", nil, rUser, "nope", ErrExamNotFound},
		{"inactive user", func(r *mockRepo) { r.users[rUser].Status = "inactive" }, rUser, rExam, ErrUserInactive},
		{"not overdue (not assigned / already passed)", func(r *mockRepo) { r.overdue = map[string]bool{} }, rUser, rExam, ErrNotOverdue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, repo, fr := remindSvc(t)
			if tc.mutate != nil {
				tc.mutate(repo)
			}
			_, err := s.RemindEmployee(context.Background(), "admin-1", tc.user, tc.exam)
			assert.ErrorIs(t, err, tc.want)
			assert.Equal(t, 0, fr.calls)
			assert.Empty(t, repo.reminders)
		})
	}
}

func TestRemindEmployee_RateLimited(t *testing.T) {
	s, repo, fr := remindSvc(t)
	last := s.now().Add(-2 * time.Hour)
	repo.lastRemind = &last
	_, err := s.RemindEmployee(context.Background(), "admin-1", rUser, rExam)
	var rl *ReminderRateLimitedError
	require.ErrorAs(t, err, &rl)
	assert.Equal(t, 22*time.Hour, rl.RetryAfter)
	assert.Equal(t, 0, fr.calls)
	assert.Empty(t, repo.reminders)

	// Outside the window the reminder goes through.
	old := s.now().Add(-25 * time.Hour)
	repo.lastRemind = &old
	_, err = s.RemindEmployee(context.Background(), "admin-1", rUser, rExam)
	assert.NoError(t, err)
}

func TestRemindEmployee_SendFailure_NoRowInserted(t *testing.T) {
	s, repo, fr := remindSvc(t)
	fr.err = errors.New("smtp down")
	_, err := s.RemindEmployee(context.Background(), "admin-1", rUser, rExam)
	assert.ErrorIs(t, err, ErrEmailSend)
	assert.Empty(t, repo.reminders)
}

func TestRemindEmployee_NoReminderConfigured(t *testing.T) {
	s, _, _ := remindSvc(t)
	s.reminder = nil
	_, err := s.RemindEmployee(context.Background(), "a", rUser, rExam)
	assert.ErrorIs(t, err, ErrReminderNotAvailable)
}

func TestRemindEmployee_InsertFailureSurfaces(t *testing.T) {
	s, repo, _ := remindSvc(t)
	repo.insertErr = errors.New("db")
	_, err := s.RemindEmployee(context.Background(), "a", rUser, rExam)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrEmailSend)
}

// ── handler ──────────────────────────────────────────────────────────────────

type spyWriter struct {
	action, entity string
	id             *string
	meta           any
	n              int
}

func (w *spyWriter) Write(_ context.Context, _ *http.Request, action, entityType string, entityID *string, metadata any) {
	w.n++
	w.action, w.entity, w.id, w.meta = action, entityType, entityID, metadata
}

func remindReq(userID, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+userID+"/remind", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", userID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return withAuthCtx(req, "admin-1", "super_admin", "")
}

func doRemind(h *Handler, userID, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.RemindEmployee(w, remindReq(userID, body))
	return w
}

func examBody() string { return "{\"exam_id\":\"" + rExam + "\"}" }

func TestHandlerRemind_ValidationErrors(t *testing.T) {
	called := false
	svc := &mockUserService{remindFn: func(context.Context, string, string, string) (*RemindResult, error) {
		called = true
		return nil, nil
	}}
	h := &Handler{svc: svc, writer: &spyWriter{}}
	for _, body := range []string{"", "{}", "{\"exam_id\":\"\"}", "{\"exam_id\":\"not-a-uuid\"}", "not json"} {
		w := doRemind(h, rUser, body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		_, apiErr := decodeHandlerEnvelope(t, w)
		require.NotNil(t, apiErr)
		assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	}
	assert.False(t, called)
}

func TestHandlerRemind_Success_AuditsWithoutEmail(t *testing.T) {
	spy := &spyWriter{}
	svc := &mockUserService{remindFn: func(_ context.Context, actor, uid, eid string) (*RemindResult, error) {
		assert.Equal(t, "admin-1", actor)
		assert.Equal(t, rUser, uid)
		assert.Equal(t, rExam, eid)
		return &RemindResult{SentAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), ExamTitle: "Fire Safety"}, nil
	}}
	h := &Handler{svc: svc, writer: spy}
	w := doRemind(h, rUser, examBody())
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"sent_at\":\"2026-03-01T10:00:00Z\"")
	assert.NotContains(t, w.Body.String(), "@")
	assert.Equal(t, 1, spy.n)
	assert.Equal(t, "users.remind", spy.action)
	assert.Equal(t, "user", spy.entity)
	assert.Equal(t, rUser, *spy.id)
	assert.Equal(t, map[string]any{"exam_id": rExam, "exam_title": "Fire Safety"}, spy.meta)
}

func TestHandlerRemind_ErrorMapping_NoAudit(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{ErrNotFound, 404, "NOT_FOUND"},
		{ErrExamNotFound, 404, "NOT_FOUND"},
		{ErrNotOverdue, 409, "NOT_OVERDUE"},
		{ErrUserInactive, 409, "USER_INACTIVE"},
		{&ReminderRateLimitedError{RetryAfter: 90 * time.Minute}, 429, "REMINDER_RATE_LIMITED"},
		{ErrEmailSend, 502, "EMAIL_SEND_FAILED"},
		{ErrReminderNotAvailable, 500, "INTERNAL_ERROR"},
		{errors.New("boom"), 500, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		spy := &spyWriter{}
		err := tc.err
		h := &Handler{svc: &mockUserService{remindFn: func(context.Context, string, string, string) (*RemindResult, error) {
			return nil, err
		}}, writer: spy}
		w := doRemind(h, rUser, examBody())
		assert.Equal(t, tc.status, w.Code, tc.code)
		assert.Contains(t, w.Body.String(), "\"code\":\""+tc.code+"\"")
		assert.Equal(t, 0, spy.n, "rejected requests must not be audited")
		if tc.status == 429 {
			assert.Contains(t, w.Body.String(), "\"retry_after_seconds\":5400")
			assert.Equal(t, "5400", w.Header().Get("Retry-After"))
		}
	}
}

func TestHandlerRemind_NonUUIDUser_404(t *testing.T) {
	h := &Handler{svc: &mockUserService{}, writer: &spyWriter{}}
	w := doRemind(h, "nope", examBody())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// AC-7: employees get 403 from the RBAC middleware that guards the route.
func TestRemindRoute_EmployeeForbidden(t *testing.T) {
	cache := rbac.NewCache()
	cache.LoadFromMap(map[string]rbac.PermissionSet{
		"employee":    {},
		"super_admin": {"reports:read": true},
	})
	called := false
	svc := &mockUserService{remindFn: func(context.Context, string, string, string) (*RemindResult, error) {
		called = true
		return &RemindResult{SentAt: time.Now(), ExamTitle: "x"}, nil
	}}
	h := &Handler{svc: svc, writer: &spyWriter{}}
	guarded := rbac.RequirePermission(cache, "reports", "read")(http.HandlerFunc(h.RemindEmployee))

	req := withAuthCtx(remindReq(rUser, examBody()), "emp-1", "employee", "")
	w := httptest.NewRecorder()
	guarded.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}
