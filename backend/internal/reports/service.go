package reports

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/deptscope"
)

// Service defines the business logic for the reports domain.
type Service interface {
	// GetDashboardMetrics returns all four metric groups for the dashboard (FR-BB51).
	// The four underlying queries are executed concurrently via goroutines + WaitGroup.
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)

	// GetExamAnalytics returns deep analytics for a single exam (FR-BB52).
	GetExamAnalytics(ctx context.Context, examID string) (*ExamAnalyticsResponse, error)

	// GetUserRecord returns the paginated session history for one employee (FR-BB53).
	GetUserRecord(ctx context.Context, userID string, page, perPage int) (*UserRecordResponse, int, error)

	// GetUserProgress returns the three-track progress summary for one employee (FR-BB53).
	GetUserProgress(ctx context.Context, userID string) (*UserProgressResponse, error)

	// ── FR-BB54: Export API ───────────────────────────────────────────────────

	// StreamExamResultsCSV writes a CSV of all session results for one exam
	// directly to w using Go's encoding/csv (AC-2, AC-3, AC-4, AC-8, AC-10).
	StreamExamResultsCSV(ctx context.Context, w http.ResponseWriter, examID, tenantID string) error

	// StreamUserRecordCSV writes a CSV of all sessions for one user
	// directly to w using Go's encoding/csv (AC-2, AC-5, AC-8, AC-10).
	StreamUserRecordCSV(ctx context.Context, w http.ResponseWriter, userID, tenantID string) error

	// BuildDashboardReport assembles DashboardReportData from the repository
	// for a given date range and tenant, used by the PDF handler (AC-6, AC-7).
	BuildDashboardReport(ctx context.Context, tenantID string, from, to time.Time, companyName, logoBase64 string) (*DashboardReportData, error)
}

// DefaultMaxExportRows is the hard row cap for the exam results CSV export
// when no explicit limit is configured (EXPORT_MAX_ROWS, ISS-210).
const DefaultMaxExportRows = 200000

// ErrExportTooLarge is returned when an export would exceed the configured row cap.
var ErrExportTooLarge = errors.New("export exceeds the maximum number of rows")

type service struct {
	repo          Repository
	maxExportRows int
}

// Option customises a Service built by NewService.
type Option func(*service)

// WithMaxExportRows sets the hard row cap for the exam results CSV export.
// Values <= 0 keep the default (DefaultMaxExportRows).
func WithMaxExportRows(n int) Option {
	return func(s *service) {
		if n > 0 {
			s.maxExportRows = n
		}
	}
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository, opts ...Option) Service {
	s := &service{repo: repo, maxExportRows: DefaultMaxExportRows}
	for _, o := range opts {
		o(s)
	}
	return s
}

// GetDashboardMetrics fans out four concurrent DB queries, waits for all to
// complete, and assembles the DashboardMetrics response.  The first non-nil
// error encountered is returned (all goroutines are always awaited).
func (s *service) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	var (
		completion []*ExamCompletionRate
		overdue    []*OverdueEmployee
		recent     []*RecentActivity
		trackMap   map[string]*float64

		errCompletion error
		errOverdue    error
		errRecent     error
		errTrack      error
	)

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		completion, errCompletion = s.repo.GetCompletionRateByExam(ctx)
	}()

	go func() {
		defer wg.Done()
		overdue, errOverdue = s.repo.GetOverdueEmployees(ctx)
	}()

	go func() {
		defer wg.Done()
		recent, errRecent = s.repo.GetRecentActivity(ctx)
	}()

	go func() {
		defer wg.Done()
		trackMap, errTrack = s.repo.GetAvgScoreByTrack(ctx)
	}()

	wg.Wait()

	// Return the first error encountered; all goroutines have already finished.
	for _, e := range []error{errCompletion, errOverdue, errRecent, errTrack} {
		if e != nil {
			return nil, fmt.Errorf("GetDashboardMetrics: %w", e)
		}
	}

	// Build the TrackScores struct; tracks absent from the map remain nil (JSON null).
	trackScores := TrackScores{
		Security: trackMap["security"],
		Safety:   trackMap["safety"],
		Loyalty:  trackMap["loyalty"],
	}

	return &DashboardMetrics{
		CompletionRateByExam: completion,
		OverdueEmployees:     overdue,
		RecentActivity:       recent,
		AvgScoreByTrack:      trackScores,
	}, nil
}

// ── FR-BB52: Per-Exam Analytics ──────────────────────────────────────────────

// orderedBuckets defines the 10 fixed score-distribution buckets in ascending order.
var orderedBuckets = []string{
	"0-10", "10-20", "20-30", "30-40", "40-50",
	"50-60", "60-70", "70-80", "80-90", "90-100",
}

// fillBuckets ensures the returned slice always contains exactly 10 buckets
// labelled "0-10" through "90-100", zero-filling any that are absent from raw.
func fillBuckets(raw []BucketCount) []BucketCount {
	counts := make(map[string]int, len(raw))
	for _, b := range raw {
		counts[b.Bucket] = b.Count
	}
	out := make([]BucketCount, len(orderedBuckets))
	for i, label := range orderedBuckets {
		out[i] = BucketCount{Bucket: label, Count: counts[label]}
	}
	return out
}

// GetExamAnalytics returns full analytics for one exam (FR-BB52).
func (s *service) GetExamAnalytics(ctx context.Context, examID string) (*ExamAnalyticsResponse, error) {
	// Verify exam exists (returns 404-able ErrNotFound if absent).
	title, err := s.repo.GetExamTitle(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetExamAnalytics: %w", err)
	}

	// Fetch all four analytics data sets.
	rawBuckets, err := s.repo.GetExamScoreDistribution(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetExamAnalytics: %w", err)
	}

	summary, err := s.repo.GetExamSummaryStats(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetExamAnalytics: %w", err)
	}

	qStats, err := s.repo.GetPerQuestionStats(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetExamAnalytics: %w", err)
	}

	answerRows, err := s.repo.GetAnswerDistribution(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetExamAnalytics: %w", err)
	}

	// Coerce nil pass_rate to 0.0 (AC-3).
	passRate := 0.0
	if summary.PassRate != nil {
		passRate = *summary.PassRate
	}

	// Group answer distribution rows by question_id.
	optionsByQuestion := make(map[string][]AnswerOptionCount)
	for _, row := range answerRows {
		optionsByQuestion[row.QuestionID] = append(optionsByQuestion[row.QuestionID], AnswerOptionCount{
			OptionID:    row.OptionID,
			OptionText:  row.OptionText,
			SelectCount: row.SelectCount,
		})
	}

	// Build per-question stats.
	perQuestion := make([]QuestionStat, 0, len(qStats))
	for _, qs := range qStats {
		options := optionsByQuestion[qs.QuestionID]
		if options == nil {
			options = []AnswerOptionCount{}
		}
		perQuestion = append(perQuestion, QuestionStat{
			QuestionID:         qs.QuestionID,
			StemPreview:        qs.StemPreview,
			CorrectRate:        qs.CorrectRate,
			AvgTimeSeconds:     qs.AvgTimeSeconds,
			AnswerDistribution: options,
		})
	}

	return &ExamAnalyticsResponse{
		ExamID:             examID,
		ExamTitle:          title,
		ScoreDistribution:  fillBuckets(rawBuckets),
		PassRate:           passRate,
		AvgScore:           summary.AvgScore,
		MedianScore:        summary.MedianScore,
		TotalAttempts:      summary.TotalAttempts,
		UniqueParticipants: summary.UniqueParticipants,
		PerQuestionStats:   perQuestion,
	}, nil
}

// ── FR-BB53: Per-Employee Record & Progress ──────────────────────────────────

// allTracks is the fixed ordered set of compliance tracks (AC-6).
var allTracks = []string{"security", "safety", "loyalty"}

// buildTrackProgress merges per-track activity rows and required exam rows
// into a fixed three-element slice, one entry per compliance track.
// Tracks with no activity get zero-value QuestionsAnswered and nil LastActivity.
// Tracks with no required exams get an empty (non-nil) RequiredExams slice.
func buildTrackProgress(activity []TrackActivity, exams []ExamProgress) []TrackSummary {
	actMap := make(map[string]TrackActivity, len(activity))
	for _, a := range activity {
		actMap[a.Track] = a
	}

	examsByTrack := make(map[string][]ExamProgress)
	for _, ex := range exams {
		examsByTrack[ex.Track] = append(examsByTrack[ex.Track], ex)
	}

	result := make([]TrackSummary, len(allTracks))
	for i, t := range allTracks {
		a := actMap[t]
		required := examsByTrack[t]
		if required == nil {
			required = []ExamProgress{}
		}
		result[i] = TrackSummary{
			Track:             t,
			QuestionsAnswered: a.QuestionsAnswered,
			LastActivity:      a.LastActivity,
			RequiredExams:     required,
		}
	}
	return result
}

// authorizeUser enforces department scoping (ISS-165): a department_admin may
// only read data about users of its own department subtree (or itself); every
// other role is unrestricted. Returns ErrNotFound otherwise, so an out-of-scope user is
// indistinguishable from an unknown one (no existence leak).
func (s *service) authorizeUser(ctx context.Context, userID string) error {
	sc := deptscope.FromContext(ctx)
	if !sc.Restricted || sc.UserID == userID {
		return nil
	}
	ok, err := s.repo.UserInScope(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// GetUserRecord validates the target user exists, then returns their paginated
// session history together with the total session count (AC-2 through AC-5).
func (s *service) GetUserRecord(ctx context.Context, userID string, page, perPage int) (*UserRecordResponse, int, error) {
	info, err := s.repo.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("reports: GetUserRecord: %w", err)
	}
	if err := s.authorizeUser(ctx, userID); err != nil {
		return nil, 0, fmt.Errorf("reports: GetUserRecord: %w", err)
	}

	total, err := s.repo.GetUserSessionCount(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("reports: GetUserRecord: %w", err)
	}

	offset := (page - 1) * perPage
	sessions, err := s.repo.GetUserSessionHistory(ctx, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("reports: GetUserRecord: %w", err)
	}

	dept := ""
	if info.DepartmentName != nil {
		dept = *info.DepartmentName
	}

	return &UserRecordResponse{
		UserID:     info.ID,
		FullName:   info.FullName,
		Department: dept,
		Sessions:   sessions,
	}, total, nil
}

// GetUserProgress validates the target user exists, then returns a three-track
// progress summary (AC-6 through AC-9).
func (s *service) GetUserProgress(ctx context.Context, userID string) (*UserProgressResponse, error) {
	info, err := s.repo.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetUserProgress: %w", err)
	}
	if err := s.authorizeUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("reports: GetUserProgress: %w", err)
	}

	activity, err := s.repo.GetUserTrackActivity(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetUserProgress: %w", err)
	}

	requiredExams, err := s.repo.GetUserRequiredExams(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetUserProgress: %w", err)
	}

	return &UserProgressResponse{
		UserID:   info.ID,
		FullName: info.FullName,
		Tracks:   buildTrackProgress(activity, requiredExams),
	}, nil
}

// ── FR-BB54: Export API ──────────────────────────────────────────────────────

// StreamExamResultsCSV writes a CSV of all session results for one exam to w.
// The session rows are collected in memory (the handler also buffers the body so
// failures can still return a proper error status, ISS-163), so the export is
// bounded: more than maxExportRows sessions yields ErrExportTooLarge (AC-8, ISS-210).
// Headers and Content-Disposition are set by the handler.
func (s *service) StreamExamResultsCSV(ctx context.Context, w http.ResponseWriter, examID, tenantID string) error {
	// ISS-178: an unknown exam id must yield ErrNotFound (handler maps it to 404)
	// instead of a header-only CSV.
	if _, err := s.repo.GetExamTitle(ctx, examID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("reports: StreamExamResultsCSV: exam lookup: %w", err)
	}

	// Fetch ordered question list to build the dynamic header (AC-3).
	questions, err := s.repo.GetExamQuestions(ctx, examID, tenantID)
	if err != nil {
		return fmt.Errorf("reports: StreamExamResultsCSV: get questions: %w", err)
	}

	// Build CSV header row.
	header := []string{
		"employee_name", "department", "started_at", "submitted_at",
		"score_pct", "passed", "time_taken_seconds",
	}
	for i := range questions {
		header = append(header, fmt.Sprintf("question_%d_score", i+1))
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("reports: StreamExamResultsCSV: write header: %w", err)
	}

	// Stream session rows from the database cursor (AC-8).
	rows, err := s.repo.StreamExamResultSessions(ctx, examID, tenantID)
	if err != nil {
		return fmt.Errorf("reports: StreamExamResultsCSV: stream sessions: %w", err)
	}

	// Collect session IDs to batch-fetch question scores.
	type sessionMeta struct {
		row ExamResultSessionRow
	}
	var sessions []sessionMeta
	var sessionIDs []string

	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			if len(sessions) >= s.maxExportRows {
				return ErrExportTooLarge
			}
			var r ExamResultSessionRow
			if err := rows.StructScan(&r); err != nil {
				return fmt.Errorf("reports: StreamExamResultsCSV: scan: %w", err)
			}
			sessions = append(sessions, sessionMeta{row: r})
			sessionIDs = append(sessionIDs, r.SessionID)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("reports: StreamExamResultsCSV: rows: %w", err)
		}
	}

	// Fetch per-question scores for all sessions.
	qScores, err := s.repo.GetSessionQuestionScores(ctx, sessionIDs)
	if err != nil {
		return fmt.Errorf("reports: StreamExamResultsCSV: get scores: %w", err)
	}

	// Index scores by (sessionID, questionID) → score.
	type scoreKey struct{ sessionID, questionID string }
	scoreMap := make(map[scoreKey]*float64, len(qScores))
	for _, qs := range qScores {
		k := scoreKey{qs.SessionID, qs.QuestionID}
		v := qs.Score
		scoreMap[k] = v
	}

	// Write one row per session (AC-3, AC-4).
	for _, sm := range sessions {
		r := sm.row

		scorePctStr := ""
		if r.ScorePct != nil {
			scorePctStr = strconv.FormatFloat(*r.ScorePct, 'f', 2, 64)
		}
		passedStr := ""
		if r.Passed != nil {
			passedStr = strconv.FormatBool(*r.Passed)
		}
		timeTakenStr := ""
		if r.TimeTakenSeconds != nil {
			timeTakenStr = strconv.Itoa(*r.TimeTakenSeconds)
		}

		// ISS-191: text cells are guarded against formula injection; numeric
		// cells (score_pct, time_taken_seconds, question scores) stay raw.
		record := []string{
			api.CSVSafe(r.EmployeeName),
			api.CSVSafe(r.Department),
			r.StartedAt,
			r.SubmittedAt,
			scorePctStr,
			passedStr,
			timeTakenStr,
		}

		// Append one cell per question; empty if score not recorded (AC-4).
		for _, q := range questions {
			k := scoreKey{r.SessionID, q.QuestionID}
			if score, ok := scoreMap[k]; ok && score != nil {
				record = append(record, strconv.FormatFloat(*score, 'f', -1, 64))
			} else {
				record = append(record, "") // grading_pending → empty cell
			}
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("reports: StreamExamResultsCSV: write row: %w", err)
		}
	}

	return nil
}

// StreamUserRecordCSV streams a CSV of all session history for one user
// to the http.ResponseWriter row by row (AC-5, AC-8).
func (s *service) StreamUserRecordCSV(ctx context.Context, w http.ResponseWriter, userID, tenantID string) error {
	if _, err := s.repo.GetUserInfo(ctx, userID); err != nil {
		return fmt.Errorf("reports: StreamUserRecordCSV: %w", err)
	}
	if err := s.authorizeUser(ctx, userID); err != nil {
		return fmt.Errorf("reports: StreamUserRecordCSV: %w", err)
	}
	header := []string{
		"exam_title", "started_at", "submitted_at",
		"score_pct", "passed", "time_taken_seconds", "status",
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("reports: StreamUserRecordCSV: write header: %w", err)
	}

	rows, err := s.repo.StreamUserRecordSessions(ctx, userID, tenantID)
	if err != nil {
		return fmt.Errorf("reports: StreamUserRecordCSV: stream sessions: %w", err)
	}
	if rows == nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var r UserRecordCSVRow
		if err := rows.StructScan(&r); err != nil {
			return fmt.Errorf("reports: StreamUserRecordCSV: scan: %w", err)
		}

		scorePctStr := ""
		if r.ScorePct != nil {
			scorePctStr = strconv.FormatFloat(*r.ScorePct, 'f', 2, 64)
		}
		passedStr := ""
		if r.Passed != nil {
			passedStr = strconv.FormatBool(*r.Passed)
		}
		timeTakenStr := ""
		if r.TimeTakenSeconds != nil {
			timeTakenStr = strconv.Itoa(*r.TimeTakenSeconds)
		}

		record := []string{
			api.CSVSafe(r.ExamTitle),
			r.StartedAt,
			r.SubmittedAt,
			scorePctStr,
			passedStr,
			timeTakenStr,
			api.CSVSafe(r.Status),
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("reports: StreamUserRecordCSV: write row: %w", err)
		}
	}

	return rows.Err()
}

// BuildDashboardReport assembles DashboardReportData for the PDF export (AC-6, AC-7).
func (s *service) BuildDashboardReport(ctx context.Context, tenantID string, from, to time.Time, companyName, logoBase64 string) (*DashboardReportData, error) {
	completionRates, err := s.repo.GetDashboardCompletionRatesForRange(ctx, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("reports: BuildDashboardReport: completion rates: %w", err)
	}

	top, bottom, err := s.repo.GetTopBottomQuestions(ctx, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("reports: BuildDashboardReport: top/bottom questions: %w", err)
	}

	return &DashboardReportData{
		From:            from.Format("02 Jan 2006"),
		To:              to.Format("02 Jan 2006"),
		CompanyName:     companyName,
		LogoBase64:      logoBase64,
		CompletionRates: completionRates,
		TopQuestions:    top,
		BottomQuestions: bottom,
	}, nil
}

