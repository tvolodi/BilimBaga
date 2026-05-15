package reports

import (
	"context"
	"fmt"
	"sync"
)

// Service defines the business logic for the reports domain.
type Service interface {
	// GetDashboardMetrics returns all four metric groups for the dashboard (FR-BB51).
	// The four underlying queries are executed concurrently via goroutines + WaitGroup.
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)

	// GetExamAnalytics returns deep analytics for a single exam (FR-BB52).
	GetExamAnalytics(ctx context.Context, examID string) (*ExamAnalyticsResponse, error)
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// queryResult holds the outcome of a single concurrent query.
type queryResult struct {
	completion []*ExamCompletionRate
	overdue    []*OverdueEmployee
	recent     []*RecentActivity
	trackMap   map[string]*float64
	err        error
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
