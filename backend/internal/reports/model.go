// Package reports implements the dashboard metrics API (FR-BB51) and
// per-exam analytics API (FR-BB52).
package reports

import (
	"errors"
	"time"
)

// ErrNotFound is returned by repository methods when no matching row exists.
var ErrNotFound = errors.New("not found")

// DashboardMetrics is the top-level response returned by GET /api/v1/admin/dashboard.
type DashboardMetrics struct {
	CompletionRateByExam []*ExamCompletionRate `json:"completion_rate_by_exam"`
	OverdueEmployees     []*OverdueEmployee    `json:"overdue_employees"`
	RecentActivity       []*RecentActivity     `json:"recent_activity"`
	AvgScoreByTrack      TrackScores           `json:"avg_score_by_track"`
}

// ExamCompletionRate holds completion and pass-rate counts for one active exam.
type ExamCompletionRate struct {
	ExamID         string `json:"exam_id"`
	Title          string `json:"title"`
	AssignedCount  int    `json:"assigned_count"`
	CompletedCount int    `json:"completed_count"`
	PassedCount    int    `json:"passed_count"`
}

// OverdueEmployee represents an employee who has passed the assignment deadline
// without a passing session for the relevant exam.
type OverdueEmployee struct {
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	ExamTitle string    `json:"exam_title"`
	Deadline  time.Time `json:"deadline"`
}

// RecentActivity is one recently submitted or grading-pending session.
type RecentActivity struct {
	SessionID    string    `json:"session_id"`
	EmployeeName string    `json:"employee_name"`
	ExamTitle    string    `json:"exam_title"`
	ScorePct     *float64  `json:"score_pct"`
	Passed       bool      `json:"passed"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

// TrackScores maps the three compliance tracks to their average score (or null).
// A nil pointer serialises as JSON null (AC-5 / note in spec).
type TrackScores struct {
	Security *float64 `json:"security"`
	Safety   *float64 `json:"safety"`
	Loyalty  *float64 `json:"loyalty"`
}

// ── Per-Exam Analytics (FR-BB52) ─────────────────────────────────────────────

// ExamAnalyticsResponse is the top-level response for GET /api/v1/admin/exams/{id}/analytics.
type ExamAnalyticsResponse struct {
	ExamID             string          `json:"exam_id"`
	ExamTitle          string          `json:"exam_title"`
	ScoreDistribution  []BucketCount   `json:"score_distribution"`
	PassRate           float64         `json:"pass_rate"`
	AvgScore           *float64        `json:"avg_score"`
	MedianScore        *float64        `json:"median_score"`
	TotalAttempts      int             `json:"total_attempts"`
	UniqueParticipants int             `json:"unique_participants"`
	PerQuestionStats   []QuestionStat  `json:"per_question_stats"`
}

// BucketCount holds the session count for one score-distribution bucket.
type BucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// QuestionStat holds per-question analytics data.
type QuestionStat struct {
	QuestionID          string              `json:"question_id"`
	StemPreview         string              `json:"stem_preview"`
	CorrectRate         *float64            `json:"correct_rate"`
	AvgTimeSeconds      *float64            `json:"avg_time_seconds"`
	AnswerDistribution  []AnswerOptionCount  `json:"answer_distribution"`
}

// AnswerOptionCount holds the selection count for one answer option.
type AnswerOptionCount struct {
	OptionID   string `json:"option_id"`
	OptionText string `json:"option_text"`
	SelectCount int   `json:"select_count"`
}
