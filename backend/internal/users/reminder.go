package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/email"
	"github.com/go-chi/chi/v5"
)

// Overdue-reminder errors (FR-BB510).
var (
	ErrExamNotFound         = errors.New("exam not found")
	ErrNotOverdue           = errors.New("user is not an overdue target for this exam")
	ErrUserInactive         = errors.New("user is not active")
	ErrEmailSend            = errors.New("reminder email delivery failed")
	ErrReminderNotAvailable = errors.New("reminder delivery is not configured")
)

// ReminderRateLimit is the minimum gap between two reminders for the same (user, exam).
const ReminderRateLimit = 24 * time.Hour

// ReminderRateLimitedError is returned when a reminder for the same (user, exam) was
// already sent within the rate-limit window.
type ReminderRateLimitedError struct {
	RetryAfter time.Duration
}

func (e *ReminderRateLimitedError) Error() string {
	return fmt.Sprintf("reminder rate limited; retry after %s", e.RetryAfter.Round(time.Second))
}

// RemindRequest is the POST /admin/users/{userId}/remind body.
type RemindRequest struct {
	ExamID string `json:"exam_id"`
}

// RemindResult is the outcome of a successful reminder.
type RemindResult struct {
	SentAt    time.Time `json:"sent_at"`
	ExamTitle string    `json:"-"`
}

// ReminderResult is what a Reminder returns after delivering the email.
type ReminderResult struct {
	SentAt    time.Time
	ExamTitle string
}

// Reminder delivers the overdue reminder email synchronously (satisfied by an adapter
// over email.EmailService, keeping internal/email free of domain imports).
type Reminder interface {
	SendOverdueReminder(ctx context.Context, userID, examID string) (*ReminderResult, error)
}

// emailReminder adapts *email.EmailService to the Reminder interface.
type emailReminder struct{ svc *email.EmailService }

func (e emailReminder) SendOverdueReminder(ctx context.Context, userID, examID string) (*ReminderResult, error) {
	res, err := e.svc.SendOverdueReminder(ctx, userID, examID)
	if err != nil {
		return nil, err
	}
	return &ReminderResult{SentAt: res.SentAt, ExamTitle: res.ExamTitle}, nil
}

// RemindEmployee validates that (user, exam) is a genuine reminder target, enforces the
// 24h rate limit, sends the email synchronously and records the reminder (FR-BB510).
// A reminder row is inserted only after a successful send. Auditing is done by the handler.
func (s *service) RemindEmployee(ctx context.Context, actorID, userID, examID string) (*RemindResult, error) {
	if s.reminder == nil {
		return nil, ErrReminderNotAvailable
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("users.RemindEmployee: get user: %w", err)
	}
	exists, err := s.repo.ExamExists(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("users.RemindEmployee: %w", err)
	}
	if !exists {
		return nil, ErrExamNotFound
	}
	if u.Status != "active" {
		return nil, ErrUserInactive
	}
	target, err := s.repo.IsOverdueTarget(ctx, userID, examID)
	if err != nil {
		return nil, fmt.Errorf("users.RemindEmployee: %w", err)
	}
	if !target {
		return nil, ErrNotOverdue
	}
	last, err := s.repo.LastReminderAt(ctx, userID, examID)
	if err != nil {
		return nil, fmt.Errorf("users.RemindEmployee: %w", err)
	}
	if last != nil {
		if wait := ReminderRateLimit - s.clock().Sub(*last); wait > 0 {
			return nil, &ReminderRateLimitedError{RetryAfter: wait}
		}
	}
	res, err := s.reminder.SendOverdueReminder(ctx, userID, examID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmailSend, err)
	}
	if err := s.repo.InsertReminder(ctx, userID, examID, actorID); err != nil {
		return nil, fmt.Errorf("users.RemindEmployee: record reminder: %w", err)
	}
	return &RemindResult{SentAt: res.SentAt, ExamTitle: res.ExamTitle}, nil
}

func (s *service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// RemindEmployee handles POST /api/v1/admin/users/:userId/remind (FR-BB510).
func (h *Handler) RemindEmployee(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	var req RemindRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil || !uuidRe.MatchString(req.ExamID) {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id must be a valid UUID")
		return
	}
	if !uuidRe.MatchString(userID) {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	actorID := auth.UserIDFromCtx(r.Context())
	res, err := h.svc.RemindEmployee(r.Context(), actorID, userID, req.ExamID)
	if err != nil {
		var rl *ReminderRateLimitedError
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrExamNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "exam not found")
		case errors.Is(err, ErrNotOverdue):
			api.WriteError(w, http.StatusConflict, "NOT_OVERDUE", "user is not an overdue target for this exam")
		case errors.Is(err, ErrUserInactive):
			api.WriteError(w, http.StatusConflict, "USER_INACTIVE", "user is not active")
		case errors.As(err, &rl):
			secs := int(rl.RetryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			api.WriteJSON(w, http.StatusTooManyRequests, map[string]any{
				"data": nil,
				"error": map[string]any{
					"code":    "REMINDER_RATE_LIMITED",
					"message": "a reminder was already sent recently",
					"details": map[string]any{"retry_after_seconds": secs},
				},
			})
		case errors.Is(err, ErrEmailSend):
			api.WriteError(w, http.StatusBadGateway, "EMAIL_SEND_FAILED", "failed to deliver reminder email")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send reminder")
		}
		return
	}
	h.writer.Write(r.Context(), r, "users.remind", "user", &userID, map[string]any{
		"exam_id":    req.ExamID,
		"exam_title": res.ExamTitle,
	})
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  map[string]any{"sent_at": res.SentAt.UTC().Format(time.RFC3339)},
		"error": nil,
	})
}

// ── repository ────────────────────────────────────────────────────────────────

// ExamExists reports whether an exam with the given id exists.
func (r *pgRepository) ExamExists(ctx context.Context, examID string) (bool, error) {
	var ok bool
	const q = `SELECT EXISTS (SELECT 1 FROM exams WHERE id = $1)`
	if err := r.db.GetContext(ctx, &ok, q, examID); err != nil {
		return false, fmt.Errorf("users.ExamExists: %w", err)
	}
	return ok, nil
}

// IsOverdueTarget reports whether the exam has an assignment resolving to the user
// (user, department-tree or 'all') and the user has no passing session for it.
// Same resolution as reports.GetOverdueEmployees, implemented locally to avoid an import.
func (r *pgRepository) IsOverdueTarget(ctx context.Context, userID, examID string) (bool, error) {
	const q = `
		WITH RECURSIVE dept_tree(id, root_id) AS (
			SELECT d.id, d.id AS root_id FROM departments d
			UNION ALL
			SELECT d.id, dt.root_id FROM departments d JOIN dept_tree dt ON d.parent_id = dt.id
		)
		SELECT EXISTS (
			SELECT 1
			FROM users u
			JOIN exam_assignments ea ON ea.exam_id = $2
			WHERE u.id = $1
			  AND (
				(ea.assignee_type = 'user' AND ea.assignee_id = u.id)
				OR (ea.assignee_type = 'department' AND EXISTS (
					SELECT 1 FROM dept_tree dt
					WHERE dt.root_id = ea.assignee_id AND dt.id = u.department_id))
				OR ea.assignee_type = 'all'
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM exam_sessions es
				WHERE es.exam_id = ea.exam_id AND es.user_id = u.id AND es.passed = TRUE
			  )
		)`
	var ok bool
	if err := r.db.GetContext(ctx, &ok, q, userID, examID); err != nil {
		return false, fmt.Errorf("users.IsOverdueTarget: %w", err)
	}
	return ok, nil
}

// LastReminderAt returns the time of the latest reminder for (user, exam), or nil.
func (r *pgRepository) LastReminderAt(ctx context.Context, userID, examID string) (*time.Time, error) {
	const q = `SELECT MAX(sent_at) FROM exam_reminders WHERE user_id = $1 AND exam_id = $2`
	var t sql.NullTime
	if err := r.db.GetContext(ctx, &t, q, userID, examID); err != nil {
		return nil, fmt.Errorf("users.LastReminderAt: %w", err)
	}
	if !t.Valid {
		return nil, nil
	}
	v := t.Time.UTC()
	return &v, nil
}

// InsertReminder records a successfully sent reminder.
func (r *pgRepository) InsertReminder(ctx context.Context, userID, examID, sentBy string) error {
	const q = `INSERT INTO exam_reminders (user_id, exam_id, sent_by) VALUES ($1, $2, $3)`
	if _, err := r.db.ExecContext(ctx, q, userID, examID, sentBy); err != nil {
		return fmt.Errorf("users.InsertReminder: %w", err)
	}
	return nil
}
