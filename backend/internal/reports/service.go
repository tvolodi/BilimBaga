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
