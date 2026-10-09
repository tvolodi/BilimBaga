package portal

import (
	"errors"
	"time"
)

var (
	ErrNotAssigned = errors.New("exam not assigned to user")
)

// UserStatus is the computed per-user state for an exam.
type UserStatus string

const (
	UserStatusNotStarted UserStatus = "not_started"
	UserStatusInProgress UserStatus = "in_progress"
	UserStatusPassed     UserStatus = "passed"
	UserStatusFailed     UserStatus = "failed"
	UserStatusExpired    UserStatus = "expired"
)

// PortalExamItem is the summary row returned by the list endpoint.
type PortalExamItem struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description"`
	TimeLimitMinutes   int        `json:"time_limit_minutes"`
	PassingScorePct    float64    `json:"passing_score_pct"`
	MaxAttempts        int        `json:"max_attempts"`
	AttemptsUsed       int        `json:"attempts_used"`
	Deadline           *time.Time `json:"deadline"`
	UserStatus         UserStatus `json:"user_status"`
	OpenSessionID      *string    `json:"open_session_id"`
	ShowAnswers        string     `json:"show_answers"`
	ShuffleQuestions   bool       `json:"shuffle_questions"`
	ShuffleOptions     bool       `json:"shuffle_options"`
	CertificateEnabled bool       `json:"certificate_enabled"`
	// ISS-132: start window so the UI can disable Start while the exam is closed.
	AvailableFrom  *time.Time `json:"available_from"`
	AvailableUntil *time.Time `json:"available_until"`
}

// AttemptHistory is a single historical session entry.
type AttemptHistory struct {
	SessionID   string     `json:"session_id"`
	StartedAt   time.Time  `json:"started_at"`
	SubmittedAt *time.Time `json:"submitted_at"`
	Status      string     `json:"status"`
	ScorePct    *float64   `json:"score_pct"`
	Passed      bool       `json:"passed"`
}

// PortalExamDetail is the full detail returned by the single-exam endpoint.
type PortalExamDetail struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	Description        *string          `json:"description"`
	TimeLimitMinutes   int              `json:"time_limit_minutes"`
	PassingScorePct    float64          `json:"passing_score_pct"`
	MaxAttempts        int              `json:"max_attempts"`
	ShuffleQuestions   bool             `json:"shuffle_questions"`
	ShuffleOptions     bool             `json:"shuffle_options"`
	ShowAnswers        string           `json:"show_answers"`
	CertificateEnabled bool             `json:"certificate_enabled"`
	AvailableFrom      *time.Time       `json:"available_from"`
	AvailableUntil     *time.Time       `json:"available_until"`
	Deadline           *time.Time       `json:"deadline"`
	UserStatus         UserStatus       `json:"user_status"`
	AttemptsUsed       int              `json:"attempts_used"`
	OpenSessionID      *string          `json:"open_session_id"`
	AttemptHistory     []AttemptHistory `json:"attempt_history"`
}

// sessionRow holds the raw session data fetched from the DB for status computation.
type sessionRow struct {
	SessionID   string
	Status      string
	ExpiresAt   *time.Time
	Passed      bool
	SubmittedAt *time.Time
	ScorePct    *float64
	StartedAt   time.Time
}
