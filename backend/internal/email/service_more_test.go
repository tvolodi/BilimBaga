package email

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// stubRepo is a configurable, concurrency-safe Repository for goroutine-based
// trigger methods. logged is signalled on every LogAttempt.
type stubRepo struct {
	mu sync.Mutex

	tenantLocale    string
	tenantLocaleErr error
	user            UserEmailData
	userErr         error
	deptUsers       []UserEmailData
	deptErr         error
	allUsers        []UserEmailData
	allErr          error
	session         *SessionEmailData
	sessionErr      error
	targets         []ReminderRow
	targetsErr      error
	logErr          error

	logs   []loggedAttempt
	logged chan struct{}
}

type loggedAttempt struct {
	to, tmpl string
	err      error
}

func newStubRepo() *stubRepo { return &stubRepo{logged: make(chan struct{}, 32), tenantLocale: "en"} }

func (r *stubRepo) LogAttempt(_ context.Context, to, tmpl string, sendErr error) error {
	r.mu.Lock()
	r.logs = append(r.logs, loggedAttempt{to, tmpl, sendErr})
	r.mu.Unlock()
	r.logged <- struct{}{}
	return r.logErr
}
func (r *stubRepo) GetTenantDefaultLocale(context.Context) (string, error) {
	return r.tenantLocale, r.tenantLocaleErr
}
func (r *stubRepo) FetchDeadlineReminderTargets(context.Context, time.Time, time.Time) ([]ReminderRow, error) {
	return r.targets, r.targetsErr
}
func (r *stubRepo) GetUserForEmail(context.Context, string) (UserEmailData, error) {
	return r.user, r.userErr
}
func (r *stubRepo) GetDepartmentUsersForEmail(context.Context, string) ([]UserEmailData, error) {
	return r.deptUsers, r.deptErr
}
func (r *stubRepo) GetAllActiveUsersForEmail(context.Context) ([]UserEmailData, error) {
	return r.allUsers, r.allErr
}
func (r *stubRepo) GetSessionEmailData(context.Context, string) (*SessionEmailData, error) {
	return r.session, r.sessionErr
}

func (r *stubRepo) waitLogs(t *testing.T, n int) []loggedAttempt {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-r.logged:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for log attempt %d/%d", i+1, n)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]loggedAttempt(nil), r.logs...)
}

func svcFor(repo Repository, smtpSrv *fakeSMTP) *EmailService {
	cfg := Config{From: "BilimBaga <noreply@example.com>", APIBaseURL: "https://app.example"}
	if smtpSrv != nil {
		cfg.Host, cfg.Port = smtpSrv.Host, smtpSrv.Port
	} else {
		cfg.Host, cfg.Port = "127.0.0.1", 1 // refused
	}
	return &EmailService{cfg: cfg, repo: repo, logger: slog.Default()}
}

// ── pure helpers ──────────────────────────────────────────────────────────────

func TestExtractEmailAddress(t *testing.T) {
	got, err := extractEmailAddress("BilimBaga <noreply@example.com>")
	if err != nil || got != "noreply@example.com" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := extractEmailAddress("not an address"); err == nil || !strings.Contains(err.Error(), "parse from address") {
		t.Fatalf("expected wrapped parse error, got %v", err)
	}
}

func TestBuildMIMEMessage(t *testing.T) {
	msg := string(buildMIMEMessage("a@x.io", "b@x.io", "Hello", "plain é", "<p>html</p>"))
	for _, want := range []string{
		"MIME-Version: 1.0\r\n", "From: a@x.io\r\n", "To: b@x.io\r\n", "Subject: Hello\r\n",
		"multipart/alternative", "text/plain; charset=UTF-8", "text/html; charset=UTF-8",
		"quoted-printable", "plain =C3=A9", "<p>html</p>",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q", want)
		}
	}
	if !strings.HasSuffix(msg, "--\r\n") {
		t.Error("message must end with closing boundary")
	}
}

// ── GetUserEmail / send ───────────────────────────────────────────────────────

func TestGetUserEmail(t *testing.T) {
	ctx := context.Background()
	if _, err := (&EmailService{}).GetUserEmail(ctx, "u"); err == nil || !strings.Contains(err.Error(), "repository not initialised") {
		t.Errorf("nil repo err = %v", err)
	}

	repo := newStubRepo()
	repo.userErr = errors.New("nope")
	_, err := svcFor(repo, nil).GetUserEmail(ctx, "u")
	if err == nil || !strings.Contains(err.Error(), "email: get user email") || !errors.Is(err, repo.userErr) {
		t.Errorf("wrapped err = %v", err)
	}

	repo = newStubRepo()
	repo.user = UserEmailData{Email: "u@x.io"}
	if got, err := svcFor(repo, nil).GetUserEmail(ctx, "u"); err != nil || got != "u@x.io" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestSend_LocaleFallbackAndDelivery(t *testing.T) {
	tests := []struct {
		name         string
		userLocale   string
		tenantLocale string
		tenantErr    error
		wantLocale   string
	}{
		{"user ru wins", "ru", "kk", nil, "ru"},
		{"user kk wins", "kk", "en", nil, "kk"},
		{"unknown user falls back to tenant", "fr", "ru", nil, "ru"},
		{"unknown both falls back to en", "fr", "de", nil, "en"},
		{"tenant lookup error falls back to en", "", "ru", errors.New("db"), "en"},
		{"empty tenant locale falls back to en", "", "", nil, "en"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startFakeSMTP(t, nil)
			repo := newStubRepo()
			repo.tenantLocale, repo.tenantLocaleErr = tt.tenantLocale, tt.tenantErr
			svc := svcFor(repo, srv)

			if err := svc.send("to@x.io", "password_reset", map[string]any{"TempPassword": "Tmp-123"}, tt.userLocale); err != nil {
				t.Fatalf("send: %v", err)
			}
			msgs, rcpts, _ := srv.snapshot()
			if len(msgs) != 1 || len(rcpts) != 1 || !strings.Contains(rcpts[0], "to@x.io") {
				t.Fatalf("msgs=%d rcpts=%v", len(msgs), rcpts)
			}
			want, err := renderTemplate("password_reset", map[string]any{"TempPassword": "Tmp-123"}, tt.wantLocale)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(msgs[0], "Subject: "+want.Subject) {
				t.Errorf("subject for locale %q not found in message:\n%s", tt.wantLocale, msgs[0])
			}
		})
	}
}

func TestSend_NilRepoUsesEnglish(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	svc := svcFor(nil, srv)
	svc.repo = nil
	if err := svc.send("to@x.io", "password_reset", map[string]any{"TempPassword": "x"}, ""); err != nil {
		t.Fatal(err)
	}
}

func TestSend_RenderErrorIsWrapped(t *testing.T) {
	err := svcFor(newStubRepo(), nil).send("to@x.io", "no_such_template", nil, "en")
	if err == nil || !strings.Contains(err.Error(), "email: render no_such_template") {
		t.Fatalf("err = %v", err)
	}
}

func TestSend_InvalidFromReturnsError(t *testing.T) {
	for _, from := range []string{"", "   ", "not an address"} {
		srv := startFakeSMTP(t, nil)
		svc := svcFor(newStubRepo(), srv)
		svc.cfg.From = from
		err := svc.send("to@x.io", "password_reset", map[string]any{"TempPassword": "x"}, "en")
		if err == nil || !strings.Contains(err.Error(), "email:") {
			t.Fatalf("from=%q: expected explicit error, got %v", from, err)
		}
		if err := svc.TestSend("to@x.io"); err == nil {
			t.Fatalf("from=%q: TestSend must fail with invalid From", from)
		}
	}
}

// ── smtpSend branches ─────────────────────────────────────────────────────────

func TestSMTPSend_Branches(t *testing.T) {
	msg := []byte("Subject: x\r\n\r\nbody\r\n")
	tests := []struct {
		name    string
		tls     bool
		user    string
		mutate  func(*fakeSMTP)
		wantErr string
	}{
		{"plain ok", false, "", nil, ""},
		{"plain ok with auth", false, "user", nil, ""},
		{"plain failure wrapped", false, "", func(s *fakeSMTP) { s.failRcpt = true }, "email: smtp sendmail"},
		{"tls-mode ok (no STARTTLS offered)", true, "", nil, ""},
		{"tls-mode ok with auth", true, "user", nil, ""},
		{"tls-mode auth failure", true, "user", func(s *fakeSMTP) { s.failAuth = true }, "email: smtp auth"},
		{"tls-mode MAIL FROM failure", true, "", func(s *fakeSMTP) { s.failMail = true }, "email: smtp MAIL FROM"},
		{"tls-mode RCPT TO failure", true, "", func(s *fakeSMTP) { s.failRcpt = true }, "email: smtp RCPT TO"},
		{"tls-mode DATA failure", true, "", func(s *fakeSMTP) { s.failData = true }, "email: smtp DATA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startFakeSMTP(t, tt.mutate)
			svc := svcFor(newStubRepo(), srv)
			svc.cfg.TLS, svc.cfg.User, svc.cfg.Pass = tt.tls, tt.user, "pw"

			err := svc.smtpSend("from@x.io", "to@x.io", msg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if msgs, _, auth := srv.snapshot(); len(msgs) != 1 || auth != (tt.user != "") {
					t.Errorf("msgs=%d auth=%v", len(msgs), auth)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v; want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestSMTPSend_TLSDialFailure(t *testing.T) {
	svc := svcFor(newStubRepo(), nil)
	svc.cfg.TLS = true
	err := svc.smtpSend("a@x.io", "b@x.io", []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "email: smtp dial") {
		t.Fatalf("err = %v", err)
	}
}

func TestTestSend(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	svc := svcFor(newStubRepo(), srv)
	if err := svc.TestSend("admin@x.io"); err != nil {
		t.Fatal(err)
	}
	msgs, _, _ := srv.snapshot()
	if len(msgs) != 1 || !strings.Contains(msgs[0], "Test Notification") {
		t.Fatalf("msgs = %v", msgs)
	}

	// Unparseable From falls back to raw string and header injection is stripped.
	svc.cfg.From = "raw-from"
	if err := svc.TestSend("a@x.io\r\nBcc: evil@x.io"); err == nil {
		t.Log("server accepted; header injection stripped from message headers")
	}
	msgs, _, _ = srv.snapshot()
	for _, m := range msgs {
		head := strings.SplitN(m, "\r\n\r\n", 2)[0]
		if strings.Contains(head, "\r\nBcc:") {
			t.Errorf("header injection not neutralised: %q", head)
		}
	}
}

// ── logAttempt ────────────────────────────────────────────────────────────────

func TestLogAttempt_NilRepoAndRepoError(t *testing.T) {
	(&EmailService{logger: slog.Default()}).logAttempt("a", "t", nil) // must not panic

	repo := newStubRepo()
	repo.logErr = errors.New("db down")
	svcFor(repo, nil).logAttempt("a", "t", errors.New("send failed")) // error swallowed, logged
	if got := repo.waitLogs(t, 1); got[0].err == nil {
		t.Error("send error should be forwarded to LogAttempt")
	}
}

// ── Trigger* (fire-and-forget goroutines) ────────────────────────────────────

func TestTriggerSessionResult_Passed(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	repo := newStubRepo()
	repo.session = &SessionEmailData{UserEmail: "u@x.io", UserLocale: "kk", ExamTitle: "Safety", ScorePct: 91.25, Passed: true}
	svcFor(repo, srv).TriggerSessionResult("s-1")

	logs := repo.waitLogs(t, 1)
	if logs[0].tmpl != "exam_passed" || logs[0].to != "u@x.io" || logs[0].err != nil {
		t.Fatalf("log = %+v", logs[0])
	}
	msgs, _, _ := srv.snapshot()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}
}

func TestTriggerSessionResult_FailedAttemptsRemaining(t *testing.T) {
	cases := []struct {
		name        string
		max, attemp int
	}{
		{"attempts left", 3, 1},
		{"over the limit clamps to zero", 2, 5},
		{"unlimited attempts", 0, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := startFakeSMTP(t, nil)
			repo := newStubRepo()
			repo.session = &SessionEmailData{UserEmail: "u@x.io", ExamTitle: "E", ScorePct: 10, PassingScorePct: 70,
				MaxAttempts: c.max, AttemptNumber: c.attemp}
			svcFor(repo, srv).TriggerSessionResult("s-1")
			if logs := repo.waitLogs(t, 1); logs[0].tmpl != "exam_failed" || logs[0].err != nil {
				t.Fatalf("log = %+v", logs[0])
			}
		})
	}
}

func TestTriggerSessionResult_SendFailureIsLogged(t *testing.T) {
	repo := newStubRepo()
	repo.session = &SessionEmailData{UserEmail: "u@x.io", ExamTitle: "E", Passed: true}
	svcFor(repo, nil).TriggerSessionResult("s-1") // SMTP refused
	if logs := repo.waitLogs(t, 1); logs[0].err == nil {
		t.Fatal("send error should be recorded in the email log")
	}
}

func TestTriggerSessionResult_NoOpPaths(t *testing.T) {
	// nil repo
	(&EmailService{logger: slog.Default()}).TriggerSessionResult("s")
	// repo error: nothing is logged
	repo := newStubRepo()
	repo.sessionErr = errors.New("gone")
	svcFor(repo, nil).TriggerSessionResult("s")
	assertNoLog(t, repo)
}

func assertNoLog(t *testing.T, repo *stubRepo) {
	t.Helper()
	select {
	case <-repo.logged:
		t.Fatal("unexpected LogAttempt call")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestTriggerAssignment(t *testing.T) {
	deadline := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	uid := "u-1"

	t.Run("single user", func(t *testing.T) {
		srv := startFakeSMTP(t, nil)
		repo := newStubRepo()
		repo.user = UserEmailData{Email: "u@x.io", Locale: "ru"}
		svcFor(repo, srv).TriggerAssignment("e-1", "Safety", &deadline, "user", &uid)
		if logs := repo.waitLogs(t, 1); logs[0].tmpl != "new_exam_assigned" || logs[0].to != "u@x.io" {
			t.Fatalf("log = %+v", logs[0])
		}
		msgs, _, _ := srv.snapshot()
		if len(msgs) != 1 {
			t.Fatalf("messages = %d", len(msgs))
		}
	})

	t.Run("department with nil deadline", func(t *testing.T) {
		srv := startFakeSMTP(t, nil)
		repo := newStubRepo()
		repo.deptUsers = []UserEmailData{{Email: "a@x.io", Locale: "en"}, {Email: "b@x.io", Locale: "kk"}}
		svcFor(repo, srv).TriggerAssignment("e-1", "Safety", nil, "department", &uid)
		repo.waitLogs(t, 2)
		if msgs, _, _ := srv.snapshot(); len(msgs) != 2 {
			t.Fatalf("messages = %d", len(msgs))
		}
	})

	t.Run("all users, send failures logged", func(t *testing.T) {
		repo := newStubRepo()
		repo.allUsers = []UserEmailData{{Email: "a@x.io"}}
		svcFor(repo, nil).TriggerAssignment("e-1", "Safety", &deadline, "all", nil)
		if logs := repo.waitLogs(t, 1); logs[0].err == nil {
			t.Fatal("expected recorded send error")
		}
	})

	t.Run("silent no-op paths", func(t *testing.T) {
		repo := newStubRepo()
		repo.userErr, repo.deptErr, repo.allErr = errors.New("x"), errors.New("x"), errors.New("x")
		svc := svcFor(repo, nil) // all lookups fail; fields are never mutated after goroutines start
		svc.TriggerAssignment("e", "t", nil, "user", nil)       // missing assignee id
		svc.TriggerAssignment("e", "t", nil, "department", nil) // missing assignee id
		svc.TriggerAssignment("e", "t", nil, "bogus", &uid)     // unknown type
		svc.TriggerAssignment("e", "t", nil, "user", &uid)      // user lookup error
		svc.TriggerAssignment("e", "t", nil, "department", &uid)
		svc.TriggerAssignment("e", "t", nil, "all", nil)
		(&EmailService{logger: slog.Default()}).TriggerAssignment("e", "t", nil, "all", nil)
		assertNoLog(t, repo)
	})
}

func TestTriggerPasswordReset(t *testing.T) {
	t.Run("sends and logs", func(t *testing.T) {
		srv := startFakeSMTP(t, nil)
		repo := newStubRepo()
		repo.user = UserEmailData{Email: "u@x.io", Locale: "kk"}
		svcFor(repo, srv).TriggerPasswordReset("u-1", "Tmp-9")
		if logs := repo.waitLogs(t, 1); logs[0].tmpl != "password_reset" || logs[0].err != nil {
			t.Fatalf("log = %+v", logs[0])
		}
		msgs, _, _ := srv.snapshot()
		if len(msgs) != 1 {
			t.Fatalf("messages = %d", len(msgs))
		}
	})
	t.Run("send failure logged", func(t *testing.T) {
		repo := newStubRepo()
		repo.user = UserEmailData{Email: "u@x.io"}
		svcFor(repo, nil).TriggerPasswordReset("u-1", "Tmp-9")
		if logs := repo.waitLogs(t, 1); logs[0].err == nil {
			t.Fatal("expected recorded send error")
		}
	})
	t.Run("no-op paths", func(t *testing.T) {
		(&EmailService{logger: slog.Default()}).TriggerPasswordReset("u", "p")
		repo := newStubRepo()
		repo.userErr = errors.New("gone")
		svcFor(repo, nil).TriggerPasswordReset("u", "p")
		assertNoLog(t, repo)
	})
}

// ── Handler ───────────────────────────────────────────────────────────────────

func handlerReq(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/notifications/test", nil)
	if userID != "" {
		req = req.WithContext(context.WithValue(req.Context(), ctxkeys.CtxUserID, userID))
	}
	return req
}

func TestHandleTestNotification_Unauthenticated(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(svcFor(newStubRepo(), nil)).HandleTestNotification(rec, handlerReq(""))
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "UNAUTHORIZED") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleTestNotification_UserLookupFails(t *testing.T) {
	repo := newStubRepo()
	repo.userErr = errors.New("gone")
	rec := httptest.NewRecorder()
	NewHandler(svcFor(repo, nil)).HandleTestNotification(rec, handlerReq("u-1"))
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "USER_NOT_FOUND") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleTestNotification_Success(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	repo := newStubRepo()
	repo.user = UserEmailData{Email: "admin@x.io"}
	rec := httptest.NewRecorder()
	NewHandler(svcFor(repo, srv)).HandleTestNotification(rec, handlerReq("u-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"recipient":"admin@x.io"`, `"status":"sent"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body missing %s: %s", want, rec.Body.String())
		}
	}
}
