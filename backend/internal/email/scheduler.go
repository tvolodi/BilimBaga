package email

import (
	"context"
	"log/slog"
	"time"
)

// StartDeadlineReminderScheduler fires at 08:00 in the given timezone every day.
// It sends deadline_reminder emails for exams whose deadline is 24–48 h away.
// The function is non-blocking; it schedules the first run via time.AfterFunc and
// recurses inside the callback.  Stops when ctx is cancelled.
func StartDeadlineReminderScheduler(ctx context.Context, emailSvc *EmailService, tz string) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		emailSvc.logger.Error("email: invalid TENANT_TIMEZONE, falling back to UTC", "tz", tz, "error", err)
		loc = time.UTC
	}
	scheduleNext(ctx, emailSvc, loc)
}

// calcNextFireDuration returns the duration until the next 08:00 in loc.
// Exported only for unit tests.
func calcNextFireDuration(now time.Time, loc *time.Location) time.Duration {
	localNow := now.In(loc)
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 8, 0, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func scheduleNext(ctx context.Context, emailSvc *EmailService, loc *time.Location) {
	dur := calcNextFireDuration(time.Now(), loc)
	time.AfterFunc(dur, func() {
		if ctx.Err() != nil {
			return // context cancelled, stop recursion
		}
		runDeadlineReminders(ctx, emailSvc)
		scheduleNext(ctx, emailSvc, loc) // schedule next day
	})
}

func runDeadlineReminders(ctx context.Context, emailSvc *EmailService) {
	if emailSvc.repo == nil {
		return
	}
	from := time.Now().UTC().Add(24 * time.Hour)
	to := time.Now().UTC().Add(48 * time.Hour)

	rows, err := emailSvc.repo.FetchDeadlineReminderTargets(ctx, from, to)
	if err != nil {
		emailSvc.logger.Error("email: deadline reminder: fetch targets", "error", err)
		return
	}

	slog.Info("email: sending deadline reminders", "count", len(rows))
	for _, row := range rows {
		tmplData := map[string]any{
			"ExamTitle": row.Title,
			"Deadline":  row.Deadline.UTC().Format("2006-01-02 15:04 UTC"),
		}
		sendErr := emailSvc.send(row.Email, "deadline_reminder", tmplData, row.Locale)
		emailSvc.logAttempt(row.Email, "deadline_reminder", sendErr)
		if sendErr != nil {
			emailSvc.logger.Error("email send failed", "template", "deadline_reminder", "to", row.Email, "error", sendErr)
		}
	}
}
