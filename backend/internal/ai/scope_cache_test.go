package ai

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/bilimbaga/bilimbaga/internal/deptscope"
)

// ISS-218 / #263: scoped insights are cached per exam + scope hash in
// ai_insight_cache, primary key (exam_id, scope_key); super_admin uses the "all"
// key. Custom roles with grading:* get NO examiner carve-out anywhere in AI
// insights (subtree-only scope), see TestGetInsights_AllNonSuperAdminRoles_*.

const (
	deptX = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	deptY = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	deptZ = "cccccccc-cccc-cccc-cccc-cccccccccccc"
)

type countingClient struct{ calls int }

func (c *countingClient) GenerateText(_ context.Context, _, _ string) (string, int, error) {
	c.calls++
	return validInsightJSON, 10, nil
}

func ctxFor(role, dept string) context.Context {
	ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, role)
	if dept != "" {
		ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, dept)
	}
	return ctx
}

func scopeRepo() *mockRepository {
	return &mockRepository{
		examInsightData: sampleExamData,
		scopeIDs: map[string][]string{
			deptX: {deptX, deptZ},
			deptY: {deptZ, deptX}, // same set as deptX, different order/root
			deptZ: {deptZ},
		},
	}
}

func TestScopedInsights_EqualScopesShareEntry(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	svc := NewService(repo, cl, "m", newLogger())

	r1, err := svc.GetInsights(ctxFor("department_admin", deptX), "exam-1", "t", "u1", false)
	if err != nil || r1.Cached {
		t.Fatalf("first call must be a paid miss: %+v %v", r1, err)
	}
	// Different role and root department but an equal department-id set.
	r2, err := svc.GetInsights(ctxFor("custom_reader", deptY), "exam-1", "t", "u2", false)
	if err != nil || !r2.Cached {
		t.Fatalf("equal scope must hit the cache: %+v %v", r2, err)
	}
	if cl.calls != 1 {
		t.Errorf("want exactly 1 paid call, got %d", cl.calls)
	}
	// The scoped entry is durable: one row under the scope key, never under "all".
	if len(repo.upsertScopes) != 1 || repo.upsertScopes[0] == deptscope.ScopeKeyAll {
		t.Errorf("scoped insight must be written once under its scope key, got %v", repo.upsertScopes)
	}
}

func TestScopedInsights_DifferentScopesSeparate(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	svc := NewService(repo, cl, "m", newLogger())
	for _, d := range []string{deptX, deptZ} {
		r, err := svc.GetInsights(ctxFor("department_admin", d), "exam-1", "t", "u", false)
		if err != nil || r.Cached {
			t.Fatalf("%s: different scope must be a fresh paid call: %+v %v", d, r, err)
		}
	}
	if cl.calls != 2 {
		t.Errorf("want 2 paid calls, got %d", cl.calls)
	}
	for _, d := range []string{deptX, deptZ} {
		if r, _ := svc.GetInsights(ctxFor("department_admin", d), "exam-1", "t", "u", false); !r.Cached {
			t.Errorf("%s: expected own cached entry", d)
		}
	}
	// Different exam, same scope: separate.
	if r, _ := svc.GetInsights(ctxFor("department_admin", deptX), "exam-2", "t", "u", false); r.Cached {
		t.Error("different exam must not share an entry")
	}
	if cl.calls != 3 {
		t.Errorf("want 3 paid calls, got %d", cl.calls)
	}
}

// Security: unrestricted ("all") and scoped entries never cross.
func TestScopedInsights_SuperAdminAndScopedNeverShare(t *testing.T) {
	shared := &InsightResult{Insights: []string{"ORG-WIDE SECRET"}, GeneratedAt: time.Now().UTC(), Cached: true}
	repo, cl := scopeRepo(), &countingClient{}
	repo.insightCache = shared
	svc := NewService(repo, cl, "m", newLogger())

	r, err := svc.GetInsights(ctxFor("department_admin", deptX), "exam-1", "t", "u", false)
	if err != nil || r.Cached || r.Insights[0] == "ORG-WIDE SECRET" {
		t.Fatalf("scoped caller must not be served the 'all' entry: %+v %v", r, err)
	}

	// Conversely a scoped entry is never served to super_admin: super_admin
	// reads only the DB row (absent here), so it pays for its own call.
	repo2, cl2 := scopeRepo(), &countingClient{}
	svc2 := NewService(repo2, cl2, "m", newLogger())
	if _, err := svc2.GetInsights(ctxFor("department_admin", deptX), "exam-1", "t", "u", false); err != nil {
		t.Fatal(err)
	}
	sa, err := svc2.GetInsights(ctxFor("super_admin", ""), "exam-1", "t", "root", false)
	if err != nil || sa.Cached {
		t.Fatalf("super_admin must not be served a scoped entry: %+v %v", sa, err)
	}
	if cl2.calls != 2 || !repo2.upsertCalled {
		t.Errorf("super_admin must generate and write the shared row; calls=%d upsert=%v", cl2.calls, repo2.upsertCalled)
	}
}

func TestScopedInsights_NoDepartmentCallersShareEmptySetKey(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	svc := NewService(repo, cl, "m", newLogger())
	if _, err := svc.GetInsights(ctxFor("examiner", ""), "exam-1", "t", "u1", false); err != nil {
		t.Fatal(err)
	}
	r, err := svc.GetInsights(ctxFor("custom_reader", ""), "exam-1", "t", "u2", false)
	if err != nil || !r.Cached {
		t.Fatalf("no-department callers share the empty-set key: %+v %v", r, err)
	}
	if r, _ := svc.GetInsights(ctxFor("department_admin", deptX), "exam-1", "t", "u3", false); r.Cached {
		t.Error("empty-set entry must not serve a caller with departments")
	}
	if cl.calls != 2 {
		t.Errorf("want 2 paid calls, got %d", cl.calls)
	}
}

func TestScopedInsights_ForceRefreshBypassesAndOverwrites(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	svc := NewService(repo, cl, "m", newLogger())
	ctx := ctxFor("department_admin", deptX)
	_, _ = svc.GetInsights(ctx, "exam-1", "t", "u", false)
	r, err := svc.GetInsights(ctx, "exam-1", "t", "u", true)
	if err != nil || r.Cached || cl.calls != 2 {
		t.Fatalf("refresh must pay again: %+v %v calls=%d", r, err, cl.calls)
	}
	if r, _ := svc.GetInsights(ctx, "exam-1", "t", "u", false); !r.Cached {
		t.Error("refreshed entry should be served afterwards")
	}
}

func TestScopedInsights_ScopeLookupFailureSkipsCache(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	repo.scopeIDsErr = errors.New("db down")
	svc := NewService(repo, cl, "m", newLogger())
	ctx := ctxFor("department_admin", deptX)
	for i := 0; i < 2; i++ {
		r, err := svc.GetInsights(ctx, "exam-1", "t", "u", false)
		if err != nil || r.Cached {
			t.Fatalf("lookup failure must degrade to uncached generation: %+v %v", r, err)
		}
	}
	if cl.calls != 2 {
		t.Errorf("want 2 paid calls (no cache), got %d", cl.calls)
	}
}

// Usage logging, which the ISS-232 daily cap counts, is written per paid
// call and not on cache hits.
func TestScopedInsights_UsageLoggedPerPaidCallOnly(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	svc := NewService(repo, cl, "m", newLogger())
	ctx := ctxFor("department_admin", deptX)
	_, _ = svc.GetInsights(ctx, "exam-1", "t", "u", false)
	if !repo.logCalled || repo.lastLog.Feature != featureExamInsights {
		t.Fatalf("paid call must be logged: %+v", repo.lastLog)
	}
	repo.logCalled = false
	_, _ = svc.GetInsights(ctx, "exam-1", "t", "u", false)
	if repo.logCalled {
		t.Error("cache hit must not log usage")
	}
}

func TestGenerateQuestions_RateLimitUnchanged(t *testing.T) {
	repo := &mockRepository{countReturns: rateLimit, categoryName: "Safety"}
	svc := NewService(repo, &mockClient{text: validJSONResponse}, "m", newLogger())
	if _, err := svc.GenerateQuestions(context.Background(), "u", validRequest()); !errors.Is(err, ErrAIRateLimited) {
		t.Fatalf("want ErrAIRateLimited, got %v", err)
	}
}

// A stored scoped row older than the TTL is not served: the caller pays again.
func TestScopedInsights_StaleRowIsRegenerated(t *testing.T) {
	repo, cl := scopeRepo(), &countingClient{}
	key := deptscope.ScopeKey(deptscope.Scope{Restricted: true, DepartmentID: deptX}, []string{deptX, deptZ})
	repo.scopedRows = map[string]*InsightResult{
		"exam-1|" + key: {Insights: []string{"stale"}, GeneratedAt: time.Now().UTC().Add(-insightCacheTTL - time.Hour)},
	}
	svc := NewService(repo, cl, "m", newLogger())
	r, err := svc.GetInsights(ctxFor("department_admin", deptX), "exam-1", "t", "u", false)
	if err != nil || r.Cached || r.Insights[0] == "stale" {
		t.Fatalf("stale scoped row must be regenerated: %+v %v", r, err)
	}
	if cl.calls != 1 {
		t.Errorf("want 1 paid call, got %d", cl.calls)
	}
}

func TestRepo_GetScopeDepartmentIDs(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"id"}, [][]driver.Value{{deptX}, {deptZ}})
	ids, err := NewRepository(db).GetScopeDepartmentIDs(context.Background(), deptscope.Scope{Restricted: true, DepartmentID: deptX})
	if err != nil || len(ids) != 2 || ids[0] != deptX {
		t.Fatalf("%v %v", ids, err)
	}

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	if _, err := NewRepository(db2).GetScopeDepartmentIDs(context.Background(), deptscope.Scope{Restricted: true}); !errors.Is(err, f2.qErr) {
		t.Errorf("error must be wrapped: %v", err)
	}
}
