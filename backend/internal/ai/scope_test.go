package ai

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/bilimbaga/bilimbaga/internal/deptscope"
)

const scopeDept = "11111111-1111-1111-1111-111111111111"

func scopedCtx(role string) context.Context {
	ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, role)
	return context.WithValue(ctx, ctxkeys.CtxDepartmentID, scopeDept)
}

// ISS-165: a department_admin gets insights computed over its own department
// subtree, so they must neither read nor populate the exam-wide shared cache.
func TestGetInsights_DepartmentAdmin_BypassesSharedCache(t *testing.T) {
	freshCache := &InsightResult{
		Insights:    []string{"All-department insight."},
		GeneratedAt: time.Now().UTC().Add(-time.Hour),
		Cached:      true,
	}
	repo := &mockRepository{insightCache: freshCache, examInsightData: sampleExamData}
	client := &mockClient{text: validInsightJSON, tokens: 10}
	svc := NewService(repo, client, "model", newLogger())

	res, err := svc.GetInsights(scopedCtx("department_admin"), "exam-1", "tenant-1", "user-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Cached || len(res.Insights) == 0 || res.Insights[0] == "All-department insight." {
		t.Errorf("department_admin must get freshly generated scoped insights, got %+v", res)
	}
	for _, sk := range repo.upsertScopes {
		if sk == deptscope.ScopeKeyAll {
			t.Error("department-scoped insights must not be written to the shared 'all' cache row")
		}
	}
	if !repo.logCalled {
		t.Error("usage must still be logged")
	}
}

func TestGetInsights_SuperAdmin_StillUsesSharedCache(t *testing.T) {
	freshCache := &InsightResult{Insights: []string{"cached"}, GeneratedAt: time.Now().UTC().Add(-time.Hour), Cached: true}
	repo := &mockRepository{insightCache: freshCache}
	svc := NewService(repo, &mockClient{}, "model", newLogger())
	res, err := svc.GetInsights(scopedCtx("super_admin"), "exam-1", "tenant-1", "user-1", false)
	if err != nil || !res.Cached {
		t.Fatalf("super_admin must still hit the shared cache: %+v %v", res, err)
	}
}

// ISS-165: insight aggregates are filtered to the department subtree for a
// department_admin; the parameter is NULL (unrestricted) for everyone else.
func TestGetExamInsightData_DepartmentScoped(t *testing.T) {
	for _, tc := range []struct {
		role string
		want driver.Value
	}{
		{"department_admin", scopeDept},
		{"examiner", scopeDept},
		{"custom_insights_reader", scopeDept},
		{"", scopeDept},
		{"super_admin", nil},
	} {
		db, f := newFakeDB(t)
		f.queue([]string{"title", "passing_score_pct"}, [][]driver.Value{{"Safety 101", int64(70)}})
		f.queue([]string{"total_attempts", "pass_rate", "avg_score_pct", "avg_completion_secs"},
			[][]driver.Value{{int64(0), nil, nil, nil}})
		f.queue([]string{"order_num", "stem", "correct_rate", "avg_time_secs"}, nil)
		if _, err := NewRepository(db).GetExamInsightData(scopedCtx(tc.role), "exam-1", "public"); err != nil {
			t.Fatalf("%s: %v", tc.role, err)
		}
		if len(f.queries) != 3 {
			t.Fatalf("%s: want 3 queries, got %d", tc.role, len(f.queries))
		}
		for i := 1; i < 3; i++ { // summary + per-question
			if !strings.Contains(f.queries[i], "WITH RECURSIVE sc_t") || strings.Contains(f.queries[i], "@SCOPE@") {
				t.Errorf("%s: query %d lacks department filter: %s", tc.role, i, f.queries[i])
			}
			a := f.args[i]
			if a[len(a)-1] != tc.want {
				t.Errorf("%s: query %d scope arg = %v, want %v", tc.role, i, a[len(a)-1], tc.want)
			}
		}
	}
}

// ISS-218: any non-super_admin role (examiner, custom role, empty role) gets
// scoped insights, so none may read or populate the exam-wide shared cache.
func TestGetInsights_AllNonSuperAdminRoles_BypassSharedCache(t *testing.T) {
	for _, role := range []string{"examiner", "custom_insights_reader", ""} {
		freshCache := &InsightResult{
			Insights:    []string{"All-department insight."},
			GeneratedAt: time.Now().UTC().Add(-time.Hour),
			Cached:      true,
		}
		repo := &mockRepository{insightCache: freshCache, examInsightData: sampleExamData}
		svc := NewService(repo, &mockClient{text: validInsightJSON, tokens: 10}, "model", newLogger())
		res, err := svc.GetInsights(scopedCtx(role), "exam-1", "tenant-1", "user-1", false)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", role, err)
		}
		if res.Cached || (len(res.Insights) > 0 && res.Insights[0] == "All-department insight.") {
			t.Errorf("%q must get freshly generated scoped insights, got %+v", role, res)
		}
		for _, sk := range repo.upsertScopes {
			if sk == deptscope.ScopeKeyAll {
				t.Errorf("%q: scoped insights must not be written to the shared 'all' cache row", role)
			}
		}
	}
}

// ISS-218: loyalty narratives are gated by the department check for every role
// but super_admin (examiner and custom roles included); empty role fails closed.
func TestGetLoyaltyNarrative_NonSuperAdminRolesDepartmentChecked(t *testing.T) {
	for _, role := range []string{"examiner", "custom_insights_reader", ""} {
		repo := &mockRepository{sessionTrack: "loyalty", sessionEmployeeID: "emp-1", inDepartment: false}
		svc := NewService(repo, &mockClient{}, "model", newLogger())
		if _, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-1", role); !errors.Is(err, ErrForbidden) {
			t.Errorf("%q: expected ErrForbidden, got %v", role, err)
		}
	}
}
