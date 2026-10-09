package exams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/bilimbaga/bilimbaga/internal/deptscope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopedCtx(role, dept string) context.Context {
	ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, role)
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, "caller-1")
	return context.WithValue(ctx, ctxkeys.CtxDepartmentID, dept)
}

// ISS-183: ListAssignments must hand the repository the caller's department
// scope; only super_admin is unrestricted, every other role is limited.
func TestListAssignments_PassesCallerScopeToRepo(t *testing.T) {
	cases := []struct {
		role, dept string
		restricted bool
		arg        any
	}{
		{"super_admin", "", false, nil},
		{"department_admin", "dept-1", true, "dept-1"},
		{"examiner", "dept-1", true, "dept-1"},
		{"qa_lead", "dept-1", true, "dept-1"},
		{"department_admin", "", true, "00000000-0000-0000-0000-000000000000"},
		{"", "", true, "00000000-0000-0000-0000-000000000000"},
	}
	for _, c := range cases {
		repo := newMockRepo()
		seedExam(repo, "exam-1", "active")
		_, err := NewService(repo).ListAssignments(scopedCtx(c.role, c.dept), "exam-1")
		require.NoError(t, err, c.role)
		assert.Equal(t, c.restricted, repo.lastScope.Restricted, c.role)
		assert.Equal(t, c.arg, repo.lastScope.Arg(), c.role)
	}
}

// Out-of-scope assignees (and their counts) reach the caller only if the repo
// returns them; the service must return exactly what the scoped repo returned.
func TestListAssignments_ReturnsOnlyScopedRepoRows(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	own := "Own Dept"
	repo.listAssignmentsWithStatsFn = func(ctx context.Context, _ string) ([]*AssignmentDetail, error) {
		all := []*AssignmentDetail{
			{ID: "a-own", AssigneeType: "department", AssigneeName: &own, Stats: AssignmentStats{TotalUsers: 3}},
			{ID: "a-all", AssigneeType: "all", Stats: AssignmentStats{TotalUsers: 3}},
		}
		if !repo.lastScope.Restricted {
			other := "Other Dept"
			all = append(all, &AssignmentDetail{ID: "a-other", AssigneeType: "department", AssigneeName: &other, Stats: AssignmentStats{TotalUsers: 40}})
		}
		return all, nil
	}
	got, err := NewService(repo).ListAssignments(scopedCtx("department_admin", "dept-1"), "exam-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, d := range got {
		assert.NotEqual(t, "a-other", d.ID)
		assert.LessOrEqual(t, d.Stats.TotalUsers, 3)
	}
	got, err = NewService(repo).ListAssignments(scopedCtx("super_admin", ""), "exam-1")
	require.NoError(t, err)
	assert.Len(t, got, 3)
}

// The exam existence check stays unscoped: exam configuration is visible.
func TestListAssignments_ExamStillVisibleToScopedCaller(t *testing.T) {
	repo := newMockRepo()
	_, err := NewService(repo).ListAssignments(scopedCtx("department_admin", "dept-1"), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

// SQL guard: the generated query must bind the scope param, restrict assignees
// and scope the "all" counts. Real-Postgres coverage: see ISS-183 report.
func TestScopeAssignmentsQuery_ExpandsAllMarkers(t *testing.T) {
	in := "@SCOPE_ASSIGNEE@|@SCOPE_USERS@|@SCOPE_SESSIONS@"
	out := scopeAssignmentsQuery(in)
	assert.NotContains(t, out, "@SCOPE_")
	assert.Contains(t, out, "ea.assignee_id IN (SELECT id FROM sc_t)")
	assert.Contains(t, out, "ea.assignee_type = 'all'")
	assert.Contains(t, out, deptscope.Predicate("ea.assignee_id", "$2"))
	assert.Contains(t, out, deptscope.Predicate("users.id", "$2"))
	assert.Contains(t, out, deptscope.Predicate("es.user_id", "$2"))
	assert.True(t, strings.Contains(out, "$2::uuid IS NULL"))
}

func TestListAssignments_Handler_PropagatesRequestContext(t *testing.T) {
	var seenRole string
	h := newHandler(&mockSvc{
		listAssignmentsFn: func(ctx context.Context, _ string) ([]*AssignmentDetail, error) {
			seenRole = ctxkeys.RoleFromCtx(ctx)
			return []*AssignmentDetail{}, nil
		},
	})
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodGet, "/api/v1/exams/exam-1/assignments", nil), "id", "exam-1")
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.CtxRole, "department_admin"))
	h.ListAssignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "department_admin", seenRole)
}
