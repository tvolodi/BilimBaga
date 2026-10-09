package ai

import (
	"database/sql/driver"
	"strings"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ISS-232 / ISS-184: cost guard for GetInsights - daily per-user cap,
// singleflight for concurrent cold requests, durable scope-keyed cache.

// slowClient blocks every call until released and counts invocations.
type slowClient struct {
	calls   atomic.Int32
	release chan struct{}
}

func (c *slowClient) GenerateText(_ context.Context, _, _ string) (string, int, error) {
	c.calls.Add(1)
	<-c.release
	return validInsightJSON, 10, nil
}

func TestGetInsights_DailyCap_ExceededBlocksPaidCall(t *testing.T) {
	repo := &mockRepository{examInsightData: sampleExamData, usageDay: 3}
	cl := &countingClient{}
	svc := NewService(repo, cl, "m", newLogger(), WithInsightsDailyLimit(3))

	_, err := svc.GetInsights(context.Background(), "exam-1", "t", "u", false)
	if !errors.Is(err, ErrAIRateLimited) {
		t.Fatalf("want ErrAIRateLimited, got %v", err)
	}
	if cl.calls != 0 || repo.logCalled || repo.upsertCalled {
		t.Errorf("capped request must not pay, log or cache: calls=%d", cl.calls)
	}
}

func TestGetInsights_DailyCap_UnderLimitAndDisabled(t *testing.T) {
	repo := &mockRepository{examInsightData: sampleExamData, usageDay: 2}
	svc := NewService(repo, &countingClient{}, "m", newLogger(), WithInsightsDailyLimit(3))
	if _, err := svc.GetInsights(context.Background(), "exam-1", "t", "u", false); err != nil {
		t.Fatalf("under the cap must succeed: %v", err)
	}

	repo2 := &mockRepository{examInsightData: sampleExamData, usageDay: 1000}
	svc2 := NewService(repo2, &countingClient{}, "m", newLogger(), WithInsightsDailyLimit(0))
	if _, err := svc2.GetInsights(context.Background(), "exam-1", "t", "u", false); err != nil {
		t.Fatalf("cap 0 disables the limit: %v", err)
	}
}

func TestGetInsights_DefaultCapApplies(t *testing.T) {
	repo := &mockRepository{examInsightData: sampleExamData, usageDay: DefaultInsightsDailyLimit}
	svc := NewService(repo, &countingClient{}, "m", newLogger())
	if _, err := svc.GetInsights(context.Background(), "exam-1", "t", "u", false); !errors.Is(err, ErrAIRateLimited) {
		t.Fatalf("default cap must apply, got %v", err)
	}
}

// Cache hits are free: a capped user can still read cached insights, but a
// forced refresh (paid) is blocked.
func TestGetInsights_DailyCap_CacheHitsStillServed(t *testing.T) {
	repo := &mockRepository{
		usageDay:     99,
		insightCache: &InsightResult{Insights: []string{"c"}, GeneratedAt: time.Now().UTC().Add(-time.Hour), Cached: true},
	}
	svc := NewService(repo, &countingClient{}, "m", newLogger(), WithInsightsDailyLimit(1))
	if r, err := svc.GetInsights(ctxFor("super_admin", ""), "exam-1", "t", "u", false); err != nil || !r.Cached {
		t.Fatalf("cache hit must bypass the cap: %+v %v", r, err)
	}
	if _, err := svc.GetInsights(ctxFor("super_admin", ""), "exam-1", "t", "u", true); !errors.Is(err, ErrAIRateLimited) {
		t.Fatalf("refresh is a paid call and must be capped, got %v", err)
	}
}

func TestGetInsights_DailyCap_CountErrorPropagates(t *testing.T) {
	boom := errors.New("db down")
	repo := &mockRepository{examInsightData: sampleExamData, usageDayErr: boom}
	svc := NewService(repo, &countingClient{}, "m", newLogger())
	if _, err := svc.GetInsights(context.Background(), "exam-1", "t", "u", false); !errors.Is(err, boom) {
		t.Fatalf("want wrapped count error, got %v", err)
	}
}

func TestGetInsights_Singleflight_ConcurrentColdRequestsPayOnce(t *testing.T) {
	repo := &mockRepository{examInsightData: sampleExamData}
	cl := &slowClient{release: make(chan struct{})}
	svc := NewService(repo, cl, "m", newLogger())

	const n = 8
	var wg sync.WaitGroup
	results := make([]*InsightResult, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = svc.GetInsights(context.Background(), "exam-1", "t", "u", false)
		}(i)
	}
	// Wait until the leader is inside the paid call, give followers time to join.
	deadline := time.Now().Add(5 * time.Second)
	for cl.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	close(cl.release)
	wg.Wait()

	if got := cl.calls.Load(); got != 1 {
		t.Fatalf("want exactly 1 paid call for %d concurrent requests, got %d", n, got)
	}
	for i := range results {
		if errs[i] != nil || results[i] == nil || len(results[i].Insights) == 0 {
			t.Fatalf("request %d: %+v %v", i, results[i], errs[i])
		}
	}
	// Callers must not share a mutable slice.
	results[0].Insights[0] = "mutated"
	if results[1].Insights[0] == "mutated" {
		t.Error("flight result must be copied per caller")
	}
}

func TestGetInsights_Singleflight_DifferentScopesAndExamsNotCoalesced(t *testing.T) {
	repo := scopeRepo()
	cl := &slowClient{release: make(chan struct{})}
	close(cl.release) // do not block
	svc := NewService(repo, cl, "m", newLogger())

	var wg sync.WaitGroup
	for _, c := range []struct{ exam, dept string }{{"e1", deptX}, {"e1", deptZ}, {"e2", deptX}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.GetInsights(ctxFor("department_admin", c.dept), c.exam, "t", "u", false)
		}()
	}
	wg.Wait()
	if got := cl.calls.Load(); got != 3 {
		t.Fatalf("distinct exam/scope pairs must each pay, got %d calls", got)
	}
}

// A caller whose request is cancelled returns promptly; the paid call still
// completes and is cached for the next caller.
func TestGetInsights_Singleflight_CancelledCallerDoesNotKillSharedCall(t *testing.T) {
	repo := &mockRepository{examInsightData: sampleExamData}
	cl := &slowClient{release: make(chan struct{})}
	svc := NewService(repo, cl, "m", newLogger())

	ctx, cancel := context.WithCancel(ctxFor("super_admin", ""))
	done := make(chan error, 1)
	go func() {
		_, err := svc.GetInsights(ctx, "exam-1", "t", "u", false)
		done <- err
	}()
	for cl.calls.Load() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled caller must return ctx error, got %v", err)
	}
	close(cl.release)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		up := repo.upsertCalled
		repo.mu.Unlock()
		if up {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !repo.upsertCalled {
		t.Fatal("shared call must complete and write the cache")
	}
	if cl.calls.Load() != 1 {
		t.Fatalf("calls = %d", cl.calls.Load())
	}
}

func TestHandler_GetInsights_RateLimited_Envelope(t *testing.T) {
	svc := &fnService{insights: func(_, _, _ string, _ bool) (*InsightResult, error) { return nil, ErrAIRateLimited }}
	w := httptest.NewRecorder()
	NewHandler(svc).HandleGetInsights(w, paramRequest("/x", "examId", "e", "u1"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if string(env.Data) != "null" || env.Error == nil || env.Error.Code != "AI_RATE_LIMITED" || env.Error.Message == "" {
		t.Fatalf("envelope = %+v data=%s", env.Error, env.Data)
	}
}

func TestRepo_CountAIUsageLastDay(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(7)}})
	n, err := NewRepository(db).CountAIUsageLastDay(context.Background(), "u1", "exam_insights")
	if err != nil || n != 7 {
		t.Fatalf("got (%d,%v)", n, err)
	}
	since, ok := f.args[0][2].(time.Time)
	if !ok || since.Before(time.Now().UTC().Add(-25*time.Hour)) || since.After(time.Now().UTC().Add(-23*time.Hour)) {
		t.Fatalf("window arg = %v", f.args[0][2])
	}
	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	if _, err := NewRepository(db2).CountAIUsageLastDay(context.Background(), "u", "x"); !errors.Is(err, f2.qErr) || !strings.Contains(err.Error(), "CountAIUsageLastDay") {
		t.Fatalf("got %v", err)
	}
}
