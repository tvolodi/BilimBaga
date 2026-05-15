// Package reports implements the dashboard metrics API (FR-BB51).
package reports

import "time"

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
