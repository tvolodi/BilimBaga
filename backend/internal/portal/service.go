package portal

import (
	"context"
	"fmt"
	"time"
)

// Service defines the business logic for the employee exam portal.
type Service interface {
	// ListMyExams returns all active exams assigned to the given user with computed status.
	ListMyExams(ctx context.Context, userID, deptID string) ([]*PortalExamItem, error)

	// GetMyExam returns the detail of one exam assigned to the given user.
	// Returns ErrNotAssigned if the exam exists but is not assigned.
	GetMyExam(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error)
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// now is a variable so tests can override it.
var now = func() time.Time { return time.Now().UTC() }

func (s *service) ListMyExams(ctx context.Context, userID, deptID string) ([]*PortalExamItem, error) {
	rows, err := s.repo.ListAssignedExams(ctx, userID, deptID)
	if err != nil {
		return nil, fmt.Errorf("portal: ListMyExams: %w", err)
	}

	result := make([]*PortalExamItem, 0, len(rows))
	for _, row := range rows {
		sessions, err := s.repo.ListUserSessions(ctx, row.ID, userID)
		if err != nil {
			return nil, fmt.Errorf("portal: ListMyExams: sessions for %s: %w", row.ID, err)
		}

		status, openSessionID, attemptsUsed := computeStatus(row.MaxAttempts, row.Deadline, sessions)
		result = append(result, &PortalExamItem{
			ID:                 row.ID,
			Title:              row.Title,
			Description:        row.Description,
			TimeLimitMinutes:   row.TimeLimitMinutes,
			PassingScorePct:    row.PassingScorePct,
			MaxAttempts:        row.MaxAttempts,
			AttemptsUsed:       attemptsUsed,
			Deadline:           row.Deadline,
			UserStatus:         status,
			OpenSessionID:      openSessionID,
			ShowAnswers:        row.ShowAnswers,
			ShuffleQuestions:   row.ShuffleQuestions,
			ShuffleOptions:     row.ShuffleOptions,
			CertificateEnabled: row.CertificateEnabled,
			AvailableFrom:      row.AvailableFrom,
			AvailableUntil:     row.AvailableUntil,
		})
	}
	return result, nil
}

func (s *service) GetMyExam(ctx context.Context, examID, userID, deptID string) (*PortalExamDetail, error) {
	row, err := s.repo.GetAssignedExam(ctx, examID, userID, deptID)
	if err != nil {
		return nil, fmt.Errorf("portal: GetMyExam: %w", err)
	}

	sessions, err := s.repo.ListUserSessions(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("portal: GetMyExam: sessions: %w", err)
	}

	status, openSessionID, attemptsUsed := computeStatus(row.MaxAttempts, row.Deadline, sessions)

	history := make([]AttemptHistory, 0)
	for _, sess := range sessions {
		if sess.Status == "submitted" || sess.Status == "auto_submitted" {
			history = append(history, AttemptHistory{
				SessionID:   sess.SessionID,
				StartedAt:   sess.StartedAt,
				SubmittedAt: sess.SubmittedAt,
				Status:      sess.Status,
				ScorePct:    sess.ScorePct,
				Passed:      sess.Passed,
			})
		}
	}

	return &PortalExamDetail{
		ID:                 row.ID,
		Title:              row.Title,
		Description:        row.Description,
		TimeLimitMinutes:   row.TimeLimitMinutes,
		PassingScorePct:    row.PassingScorePct,
		MaxAttempts:        row.MaxAttempts,
		ShuffleQuestions:   row.ShuffleQuestions,
		ShuffleOptions:     row.ShuffleOptions,
		ShowAnswers:        row.ShowAnswers,
		CertificateEnabled: row.CertificateEnabled,
		AvailableFrom:      row.AvailableFrom,
		AvailableUntil:     row.AvailableUntil,
		Deadline:           row.Deadline,
		UserStatus:         status,
		AttemptsUsed:       attemptsUsed,
		OpenSessionID:      openSessionID,
		AttemptHistory:     history,
	}, nil
}

// computeStatus implements the status computation logic from the spec (AC-2 through AC-6).
//
// Status priority:
//  1. passed  — if any session has passed=true (AC-3)
//  2. in_progress — if there is an open session not yet expired (AC-4)
//  3. expired — deadline passed with no passed session (AC-2, takes priority over failed)
//  4. failed  — all attempts used, none passed, deadline not reached (AC-5)
//  5. not_started — all other cases (AC-6)
//
// AC-8: attempts_used = count of sessions with status IN ('submitted','auto_submitted').
func computeStatus(maxAttempts int, deadline *time.Time, sessions []sessionRow) (UserStatus, *string, int) {
	currentTime := now()

	var openSessionID *string
	var passedSessionID *string
	attemptsUsed := 0

	for i := range sessions {
		sess := &sessions[i]
		switch sess.Status {
		case "submitted", "auto_submitted":
			attemptsUsed++
			if sess.Passed {
				passedSessionID = &sess.SessionID
			}
		case "in_progress":
			if sess.ExpiresAt != nil && sess.ExpiresAt.After(currentTime) {
				id := sess.SessionID
				openSessionID = &id
			}
		}
	}

	if passedSessionID != nil {
		return UserStatusPassed, openSessionID, attemptsUsed
	}
	if openSessionID != nil {
		return UserStatusInProgress, openSessionID, attemptsUsed
	}
	if deadline != nil && !deadline.After(currentTime) {
		return UserStatusExpired, nil, attemptsUsed
	}
	if attemptsUsed >= maxAttempts {
		return UserStatusFailed, nil, attemptsUsed
	}
	return UserStatusNotStarted, nil, attemptsUsed
}
