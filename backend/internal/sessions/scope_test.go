package sessions

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// ISS-165: the manual-grading queue only lists sessions of employees inside a
// department_admin's department subtree.
func TestListGradingQueue_DepartmentScoped(t *testing.T) {
	const dept = "11111111-1111-1111-1111-111111111111"
	for _, tc := range []struct {
		role string
		want driver.Value
	}{
		{"department_admin", dept},
		{"super_admin", nil},
		{"examiner", nil},
	} {
		db, f := newFakeDB(t)
		f.queue([]string{"count"}, [][]driver.Value{{int64(0)}})
		ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, tc.role)
		ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, dept)

		if _, _, err := NewRepository(db, nil).ListGradingQueue(ctx, nil, nil, nil, 1, 20); err != nil {
			t.Fatalf("%s: %v", tc.role, err)
		}
		if len(f.queries) != 2 {
			t.Fatalf("%s: want count+rows queries, got %d", tc.role, len(f.queries))
		}
		if !strings.Contains(f.queries[0], "es.user_id IN (") || !strings.Contains(f.queries[0], "$4::uuid IS NULL") {
			t.Errorf("%s: count query lacks scope filter: %s", tc.role, f.queries[0])
		}
		if !strings.Contains(f.queries[1], "es.user_id IN (") || !strings.Contains(f.queries[1], "$6::uuid IS NULL") {
			t.Errorf("%s: rows query lacks scope filter: %s", tc.role, f.queries[1])
		}
		if got := f.args[0][3]; got != tc.want {
			t.Errorf("%s: count scope arg = %v, want %v", tc.role, got, tc.want)
		}
		if got := f.args[1][5]; got != tc.want {
			t.Errorf("%s: rows scope arg = %v, want %v", tc.role, got, tc.want)
		}
	}
}
