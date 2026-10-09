package email

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// ── Repository mock ───────────────────────────────────────────────────────────

type mockRepo struct {
	mu               sync.Mutex // guards the logged* fields (written from async send goroutines)
	logAttemptCalled bool
	loggedTo         string
	loggedTmpl       string
	loggedErr        error
}

func (m *mockRepo) LogAttempt(_ context.Context, to, tmpl string, sendErr error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logAttemptCalled = true
	m.loggedTo = to
	m.loggedTmpl = tmpl
	m.loggedErr = sendErr
	return nil
}

// logged returns a race-free snapshot of the last LogAttempt call.
func (m *mockRepo) logged() (called bool, tmpl string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logAttemptCalled, m.loggedTmpl, m.loggedErr
}

func (m *mockRepo) GetTenantDefaultLocale(_ context.Context) (string, error) {
	return "en", nil
}

func (m *mockRepo) FetchDeadlineReminderTargets(_ context.Context, _, _ time.Time) ([]ReminderRow, error) {
	return nil, nil
}

func (m *mockRepo) GetUserForEmail(_ context.Context, _ string) (UserEmailData, error) {
	return UserEmailData{Email: "test@example.com", Locale: "en"}, nil
}

func (m *mockRepo) GetDepartmentUsersForEmail(_ context.Context, _ string) ([]UserEmailData, error) {
	return nil, nil
}

func (m *mockRepo) GetAllActiveUsersForEmail(_ context.Context) ([]UserEmailData, error) {
	return nil, nil
}

func (m *mockRepo) GetSessionEmailData(_ context.Context, sessionID string) (*SessionEmailData, error) {
	return &SessionEmailData{
		UserEmail:       "test@example.com",
		UserLocale:      "en",
		ExamTitle:       "Test Exam",
		ScorePct:        85.0,
		PassingScorePct: 70.0,
		MaxAttempts:     3,
		AttemptNumber:   1,
		Passed:          true,
		SessionID:       sessionID,
	}, nil
}

// TestSanitiseHeader verifies that CR and LF are stripped (AC-9, CWE-93).
func TestSanitiseHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"normal subject", "normal subject"},
		{"inject\r\nBcc: attacker@evil.com", "inject  Bcc: attacker@evil.com"},
		{"only\nnewline", "only newline"},
		{"only\rcarriage", "only carriage"},
		{"multi\r\nline\r\nvalue", "multi  line  value"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := sanitiseHeader(tt.input)
			if got != tt.want {
				t.Errorf("sanitiseHeader(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestLogAttempt verifies that logAttempt writes to the repository (AC-5).
func TestLogAttempt(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{}
	svc := &EmailService{
		cfg:    Config{},
		repo:   repo,
		logger: slog.Default(),
	}

	svc.logAttempt("dest@example.com", "password_reset", nil)

	if !repo.logAttemptCalled {
		t.Fatal("expected LogAttempt to be called on the repository")
	}
	if repo.loggedTo != "dest@example.com" {
		t.Errorf("loggedTo = %q; want %q", repo.loggedTo, "dest@example.com")
	}
	if repo.loggedTmpl != "password_reset" {
		t.Errorf("loggedTmpl = %q; want %q", repo.loggedTmpl, "password_reset")
	}
	if repo.loggedErr != nil {
		t.Errorf("loggedErr = %v; want nil", repo.loggedErr)
	}
}

// TestHandleTestNotification_SMTPFail verifies that a bad SMTP config causes
// the handler to return 503 EMAIL_UNAVAILABLE (AC-6).
func TestHandleTestNotification_SMTPFail(t *testing.T) {
	t.Parallel()

	svc := &EmailService{
		cfg: Config{
			Host: "127.0.0.1",
			Port: 19999,
			From: "noreply@test.local",
		},
		repo:   &mockRepo{},
		logger: slog.Default(),
	}

	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/notifications/test", nil)
	ctx := context.WithValue(req.Context(), ctxkeys.CtxUserID, "test-user-id")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleTestNotification(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d; want %d (EMAIL_UNAVAILABLE)", w.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "EMAIL_UNAVAILABLE" {
		t.Errorf("error.code = %q; want %q", body.Error.Code, "EMAIL_UNAVAILABLE")
	}
}

// TestSchedulerNextFireDuration verifies calcNextFireDuration produces the
// correct duration to the next 08:00 in the given timezone.
func TestSchedulerNextFireDuration(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Skip("timezone data not available:", err)
	}

	tests := []struct {
		desc        string
		localHour   int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{"hour=7", 7, 59 * time.Minute, 61 * time.Minute},
		{"hour=9", 9, 22*time.Hour + 59*time.Minute, 23*time.Hour + 1*time.Minute},
		{"hour=8", 8, 23*time.Hour + 59*time.Minute, 24*time.Hour + 1*time.Minute},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			now := time.Date(2026, 1, 15, tt.localHour, 0, 0, 0, loc)
			dur := calcNextFireDuration(now, loc)
			if dur < tt.minExpected || dur > tt.maxExpected {
				t.Errorf("calcNextFireDuration(hour=%d) = %v; want between %v and %v",
					tt.localHour, dur, tt.minExpected, tt.maxExpected)
			}
		})
	}
}
