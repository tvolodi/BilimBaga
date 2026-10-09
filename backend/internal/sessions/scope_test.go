package sessions

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// ISS-165 / ISS-218: the manual-grading queue is limited to the caller's
// department subtree for every role except super_admin. The built-in examiner
// additionally sees sessions of exams it created (created_by = caller); no
// other role gets that carve-out.
func TestListGradingQueue_DepartmentScoped(t *testing.T) {
	const dept = "11111111-1111-1111-1111-111111111111"
	const caller = "22222222-2222-2222-2222-222222222222"
	zero := "00000000-0000-0000-0000-000000000000"
	for _, tc := range []struct {
		role      string
		dept      string
		wantScope driver.Value
		wantOwner driver.Value
	}{
		{"department_admin", dept, dept, nil},
		{"super_admin", dept, nil, nil},
		{"examiner", dept, dept, caller},
		{"custom_grader", dept, dept, nil},
		{"custom_grader", "", zero, nil},
		{"", dept, dept, nil},
	} {
		db, f := newFakeDB(t)
		f.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
		ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, tc.role)
		ctx = context.WithValue(ctx, ctxkeys.CtxUserID, caller)
		ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, tc.dept)

		if _, _, err := NewRepository(db, nil).ListGradingQueue(ctx, nil, nil, nil, 1, 20); err != nil {
			t.Fatalf("%s: %v", tc.role, err)
		}
		if len(f.queries) != 2 {
			t.Fatalf("%s: want count+rows queries, got %d", tc.role, len(f.queries))
		}
		if !strings.Contains(f.queries[0], "es.user_id IN (") || !strings.Contains(f.queries[0], "$4::uuid IS NULL") ||
			!strings.Contains(f.queries[0], "e.created_by = $5::uuid") {
			t.Errorf("%s: count query lacks scope filter: %s", tc.role, f.queries[0])
		}
		if !strings.Contains(f.queries[1], "es.user_id IN (") || !strings.Contains(f.queries[1], "$6::uuid IS NULL") ||
			!strings.Contains(f.queries[1], "e.created_by = $7::uuid") {
			t.Errorf("%s: rows query lacks scope filter: %s", tc.role, f.queries[1])
		}
		if got := f.args[0][3]; got != tc.wantScope {
			t.Errorf("%s: count scope arg = %v, want %v", tc.role, got, tc.wantScope)
		}
		if got := f.args[0][4]; got != tc.wantOwner {
			t.Errorf("%s: count owner arg = %v, want %v", tc.role, got, tc.wantOwner)
		}
		if got := f.args[1][5]; got != tc.wantScope {
			t.Errorf("%s: rows scope arg = %v, want %v", tc.role, got, tc.wantScope)
		}
		if got := f.args[1][6]; got != tc.wantOwner {
			t.Errorf("%s: rows owner arg = %v, want %v", tc.role, got, tc.wantOwner)
		}
	}
}
