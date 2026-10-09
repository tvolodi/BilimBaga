package email

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"
)

func newRepoWithFake(t *testing.T) (Repository, *fakeDB) {
	t.Helper()
	db, f := newFakeDB(t)
	return newPostgresRepository(db), f
}

func TestRepo_LogAttempt(t *testing.T) {
	ctx := context.Background()

	t.Run("success records recipient, template and nil error", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		if err := r.LogAttempt(ctx, "a@x.io", "password_reset", nil); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(f.queries[0], "INSERT INTO email_log") {
			t.Errorf("unexpected query %q", f.queries[0])
		}
		if f.args[0][0] != "a@x.io" || f.args[0][1] != "password_reset" || f.args[0][2] != nil {
			t.Errorf("args = %v", f.args[0])
		}
	})

	t.Run("send error text is persisted", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		if err := r.LogAttempt(ctx, "a@x.io", "t", errors.New("smtp down")); err != nil {
			t.Fatal(err)
		}
		if f.args[0][2] != "smtp down" {
			t.Errorf("error arg = %v; want 'smtp down'", f.args[0][2])
		}
	})

	t.Run("exec failure is wrapped", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.xErr = errors.New("db down")
		err := r.LogAttempt(ctx, "a@x.io", "t", nil)
		if err == nil || !strings.Contains(err.Error(), "email: log attempt") || !errors.Is(err, f.xErr) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestRepo_GetTenantDefaultLocale(t *testing.T) {
	ctx := context.Background()

	t.Run("returns configured locale", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.queue([]string{"value"}, [][]driver.Value{{"kk"}})
		got, err := r.GetTenantDefaultLocale(ctx)
		if err != nil || got != "kk" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	t.Run("no rows falls back to en", func(t *testing.T) {
		r, _ := newRepoWithFake(t)
		got, err := r.GetTenantDefaultLocale(ctx)
		if err != nil || got != "en" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	t.Run("query failure is wrapped", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.qErr = errors.New("boom")
		_, err := r.GetTenantDefaultLocale(ctx)
		if err == nil || !strings.Contains(err.Error(), "email: get tenant locale") || !errors.Is(err, f.qErr) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestRepo_FetchDeadlineReminderTargets(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	cols := []string{"user_id", "deadline", "email", "preferred_locale", "title"}

	t.Run("maps rows", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.queue(cols, [][]driver.Value{
			{"u1", to, "u1@x.io", "ru", "Safety"},
			{"u2", to, "u2@x.io", "", "Fire"},
		})
		got, err := r.FetchDeadlineReminderTargets(ctx, from, to)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].UserID != "u1" || got[0].Locale != "ru" || got[0].Title != "Safety" ||
			got[1].Email != "u2@x.io" || !got[1].Deadline.Equal(to) {
			t.Fatalf("rows = %+v", got)
		}
		if len(f.args[0]) != 2 {
			t.Errorf("expected 2 bind args, got %v", f.args[0])
		}
	})

	t.Run("empty result is non-nil empty slice", func(t *testing.T) {
		r, _ := newRepoWithFake(t)
		got, err := r.FetchDeadlineReminderTargets(ctx, from, to)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})

	t.Run("query failure is wrapped", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.qErr = errors.New("boom")
		_, err := r.FetchDeadlineReminderTargets(ctx, from, to)
		if err == nil || !strings.Contains(err.Error(), "email: fetch deadline reminder targets") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestRepo_GetUserForEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.queue([]string{"email", "preferred_locale"}, [][]driver.Value{{"u@x.io", "kk"}})
		got, err := r.GetUserForEmail(ctx, "u-1")
		if err != nil || got.Email != "u@x.io" || got.Locale != "kk" {
			t.Fatalf("got %+v, %v", got, err)
		}
		if f.args[0][0] != "u-1" {
			t.Errorf("args = %v", f.args[0])
		}
	})

	t.Run("not found wraps with user id", func(t *testing.T) {
		r, _ := newRepoWithFake(t)
		_, err := r.GetUserForEmail(ctx, "missing")
		if err == nil || !strings.Contains(err.Error(), "email: get user missing") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestRepo_UserListQueries(t *testing.T) {
	ctx := context.Background()
	cols := []string{"email", "preferred_locale"}
	data := [][]driver.Value{{"a@x.io", "en"}, {"b@x.io", "ru"}}

	t.Run("department users", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.queue(cols, data)
		got, err := r.GetDepartmentUsersForEmail(ctx, "d-1")
		if err != nil || len(got) != 2 || got[1].Email != "b@x.io" || got[1].Locale != "ru" {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
	t.Run("department users error", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.qErr = errors.New("boom")
		_, err := r.GetDepartmentUsersForEmail(ctx, "d-1")
		if err == nil || !strings.Contains(err.Error(), "email: get department users d-1") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("all active users", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.queue(cols, data)
		got, err := r.GetAllActiveUsersForEmail(ctx)
		if err != nil || len(got) != 2 || got[0].Email != "a@x.io" {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
	t.Run("all active users error", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.qErr = errors.New("boom")
		_, err := r.GetAllActiveUsersForEmail(ctx)
		if err == nil || !strings.Contains(err.Error(), "email: get all active users") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestRepo_GetSessionEmailData(t *testing.T) {
	ctx := context.Background()
	cols := []string{"user_email", "user_locale", "exam_title", "score_pct", "passing_score_pct", "max_attempts", "passed", "session_id", "attempt_number"}

	t.Run("maps all fields", func(t *testing.T) {
		r, f := newRepoWithFake(t)
		f.queue(cols, [][]driver.Value{{"u@x.io", "ru", "Exam", 81.5, 70.0, int64(3), true, "s-1", int64(2)}})
		got, err := r.GetSessionEmailData(ctx, "s-1")
		if err != nil {
			t.Fatal(err)
		}
		want := SessionEmailData{UserEmail: "u@x.io", UserLocale: "ru", ExamTitle: "Exam", ScorePct: 81.5,
			PassingScorePct: 70, MaxAttempts: 3, AttemptNumber: 2, Passed: true, SessionID: "s-1"}
		if *got != want {
			t.Fatalf("got %+v; want %+v", *got, want)
		}
	})

	t.Run("not found is wrapped", func(t *testing.T) {
		r, _ := newRepoWithFake(t)
		got, err := r.GetSessionEmailData(ctx, "s-x")
		if got != nil || err == nil || !strings.Contains(err.Error(), "email: get session data s-x") {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}

func TestNewEmailService_RepoWiring(t *testing.T) {
	svc := NewEmailService(Config{}, nil, nil)
	if svc.repo != nil {
		t.Error("nil db must leave repo nil")
	}
	if svc.logger == nil {
		t.Error("nil logger must default to slog.Default")
	}
	db, _ := newFakeDB(t)
	if NewEmailService(Config{}, db, nil).repo == nil {
		t.Error("non-nil db must create a postgres repository")
	}
}
