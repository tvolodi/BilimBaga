package email

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestCalcNextFireDuration_Exactly0800RollsToTomorrow(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	if got := calcNextFireDuration(now, time.UTC); got != 24*time.Hour {
		t.Fatalf("got %v; want 24h", got)
	}
}

func TestCalcNextFireDuration_BeforeAndAfter(t *testing.T) {
	loc := time.FixedZone("UTC+5", 5*3600)
	// 07:30 local on a UTC+5 clock: 30 minutes to go.
	now := time.Date(2026, 5, 1, 2, 30, 0, 0, time.UTC)
	if got := calcNextFireDuration(now, loc); got != 30*time.Minute {
		t.Errorf("before 08:00: got %v; want 30m", got)
	}
	// 20:00 local: 12h to go.
	now = time.Date(2026, 5, 1, 15, 0, 0, 0, time.UTC)
	if got := calcNextFireDuration(now, loc); got != 12*time.Hour {
		t.Errorf("after 08:00: got %v; want 12h", got)
	}
}

func TestStartDeadlineReminderScheduler_InvalidTimezoneFallsBackToUTC(t *testing.T) {
	var buf bytes.Buffer
	svc := &EmailService{logger: slog.New(slog.NewTextHandler(&buf, nil))}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the timer would fire hours from now; cancelling keeps the callback inert

	StartDeadlineReminderScheduler(ctx, svc, "Not/AZone")

	if !strings.Contains(buf.String(), "invalid TENANT_TIMEZONE") {
		t.Errorf("expected fallback error log, got %q", buf.String())
	}
}

func TestStartDeadlineReminderScheduler_ValidTimezoneIsSilent(t *testing.T) {
	var buf bytes.Buffer
	svc := &EmailService{logger: slog.New(slog.NewTextHandler(&buf, nil))}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	StartDeadlineReminderScheduler(ctx, svc, "UTC")
	if buf.Len() != 0 {
		t.Errorf("unexpected log output %q", buf.String())
	}
}

func TestRunDeadlineReminders_NilRepoIsNoOp(t *testing.T) {
	runDeadlineReminders(context.Background(), &EmailService{logger: slog.Default()})
}

func TestRunDeadlineReminders_FetchErrorLogged(t *testing.T) {
	var buf bytes.Buffer
	repo := newStubRepo()
	repo.targetsErr = errors.New("db down")
	svc := svcFor(repo, nil)
	svc.logger = slog.New(slog.NewTextHandler(&buf, nil))

	runDeadlineReminders(context.Background(), svc)

	if !strings.Contains(buf.String(), "fetch targets") {
		t.Errorf("log = %q", buf.String())
	}
	assertNoLog(t, repo)
}

func TestRunDeadlineReminders_SendsToEachTarget(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	repo := newStubRepo()
	dl := time.Now().UTC().Add(30 * time.Hour)
	repo.targets = []ReminderRow{
		{UserID: "u1", Email: "a@x.io", Locale: "ru", Title: "Safety", Deadline: dl},
		{UserID: "u2", Email: "b@x.io", Locale: "", Title: "Fire", Deadline: dl},
	}
	runDeadlineReminders(context.Background(), svcFor(repo, srv))

	logs := repo.waitLogs(t, 2)
	for _, l := range logs {
		if l.tmpl != "deadline_reminder" || l.err != nil {
			t.Errorf("log = %+v", l)
		}
	}
	if msgs, _, _ := srv.snapshot(); len(msgs) != 2 {
		t.Fatalf("messages = %d; want 2", len(msgs))
	}
}

func TestRunDeadlineReminders_SendFailureRecorded(t *testing.T) {
	repo := newStubRepo()
	repo.targets = []ReminderRow{{Email: "a@x.io", Title: "Safety", Deadline: time.Now()}}
	runDeadlineReminders(context.Background(), svcFor(repo, nil))
	if logs := repo.waitLogs(t, 1); logs[0].err == nil {
		t.Fatal("expected recorded send error")
	}
}
