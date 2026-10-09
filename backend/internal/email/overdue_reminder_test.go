package email

import (
	"database/sql/driver"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// Satisfy Repository for the pre-existing test doubles (FR-BB510).
func (r *stubRepo) GetOverdueReminderData(context.Context, string, string) (*OverdueReminderData, error) {
	return nil, errors.New("not configured")
}
func (m *mockRepo) GetOverdueReminderData(context.Context, string, string) (*OverdueReminderData, error) {
	return nil, errors.New("not configured")
}

// reminderRepo is a stubRepo with a canned overdue-reminder context.
type reminderRepo struct {
	*stubRepo
	data *OverdueReminderData
	err  error
}

func (r *reminderRepo) GetOverdueReminderData(context.Context, string, string) (*OverdueReminderData, error) {
	return r.data, r.err
}

func TestOverdueReminderTemplate_AllLocales(t *testing.T) {
	data := map[string]any{
		"ExamTitle":  "Fire Safety",
		"Deadline":   "2026-01-02 08:00 UTC",
		"PortalLink": "https://app.example/portal/exams/abc",
	}
	for _, locale := range []string{"en", "ru", "kk"} {
		r, err := renderTemplate("overdue_reminder", data, locale)
		if err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
		for _, part := range []string{r.Subject, r.TextBody, r.HTMLBody} {
			if part == "" || strings.Contains(part, "{{") {
				t.Errorf("%s: empty or unresolved part %q", locale, part)
			}
		}
		if !strings.Contains(r.Subject, "Fire Safety") ||
			!strings.Contains(r.TextBody, "2026-01-02 08:00 UTC") ||
			!strings.Contains(r.TextBody, "https://app.example/portal/exams/abc") ||
			!strings.Contains(r.HTMLBody, `href="https://app.example/portal/exams/abc"`) {
			t.Errorf("%s: missing title/deadline/link", locale)
		}
	}
}

func TestSendOverdueReminder_Success(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	dl := time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)
	repo := &reminderRepo{stubRepo: newStubRepo(), data: &OverdueReminderData{
		Email: "emp@x.io", Locale: "ru", ExamTitle: "Fire Safety", Deadline: &dl}}
	svc := svcFor(repo, srv)
	svc.cfg.PublicAppURL = "https://spa.example/"

	res, err := svc.SendOverdueReminder(context.Background(), "u1", "e1")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if res.ExamTitle != "Fire Safety" || res.SentAt.IsZero() {
		t.Fatalf("result = %+v", res)
	}
	msgs, rcpts, _ := srv.snapshot()
	if len(msgs) != 1 || !strings.Contains(rcpts[0], "emp@x.io") {
		t.Fatalf("msgs=%d rcpts=%v", len(msgs), rcpts)
	}
	want, _ := renderTemplate("overdue_reminder", map[string]any{
		"ExamTitle": "Fire Safety", "Deadline": "2026-01-02 08:00 UTC", "PortalLink": "x"}, "ru")
	if !strings.Contains(msgs[0], "Subject: "+want.Subject) {
		t.Errorf("subject not localized to ru:\n%s", msgs[0])
	}
	logs := repo.waitLogs(t, 1)
	if logs[0].tmpl != "overdue_reminder" || logs[0].err != nil || logs[0].to != "emp@x.io" {
		t.Errorf("log = %+v", logs[0])
	}
}

func TestSendOverdueReminder_SMTPFailureIsLoggedAndReturned(t *testing.T) {
	repo := &reminderRepo{stubRepo: newStubRepo(), data: &OverdueReminderData{
		Email: "emp@x.io", ExamTitle: "T"}}
	svc := svcFor(repo, nil) // connection refused
	if _, err := svc.SendOverdueReminder(context.Background(), "u1", "e1"); err == nil {
		t.Fatal("expected error")
	}
	logs := repo.waitLogs(t, 1)
	if logs[0].err == nil {
		t.Error("failure must be recorded in email_log")
	}
}

func TestSendOverdueReminder_ContextErrors(t *testing.T) {
	svc := &EmailService{}
	if _, err := svc.SendOverdueReminder(context.Background(), "u", "e"); err == nil {
		t.Error("nil repo must error")
	}
	repo := &reminderRepo{stubRepo: newStubRepo(), err: errors.New("db")}
	if _, err := svcFor(repo, nil).SendOverdueReminder(context.Background(), "u", "e"); err == nil {
		t.Error("repo error must propagate")
	}
}

func TestRepo_GetOverdueReminderData(t *testing.T) {
	cols := []string{"email", "preferred_locale", "title", "deadline"}
	dl := time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)

	r, f := newRepoWithFake(t)
	f.queue(cols, [][]driver.Value{{"a@x.io", "kk", "Safety", dl}})
	got, err := r.GetOverdueReminderData(context.Background(), "u1", "e1")
	if err != nil || got.Email != "a@x.io" || got.Locale != "kk" || got.ExamTitle != "Safety" ||
		got.Deadline == nil || !got.Deadline.Equal(dl) {
		t.Fatalf("got %+v, %v", got, err)
	}

	r, f = newRepoWithFake(t)
	f.queue(cols, [][]driver.Value{{"a@x.io", "", "Safety", nil}})
	got, err = r.GetOverdueReminderData(context.Background(), "u1", "e1")
	if err != nil || got.Deadline != nil {
		t.Fatalf("nil deadline: got %+v, %v", got, err)
	}

	r, f = newRepoWithFake(t)
	f.qErr = errors.New("boom")
	if _, err = r.GetOverdueReminderData(context.Background(), "u1", "e1"); err == nil {
		t.Fatal("expected error")
	}
}
