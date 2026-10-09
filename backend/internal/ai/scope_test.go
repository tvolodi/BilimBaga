package ai

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
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
	if repo.upsertCalled {
		t.Error("department-scoped insights must not be written to the shared cache")
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
