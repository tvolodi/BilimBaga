package users

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-184 / FR-BB117: GET /users is scoped to the caller's OWN department only
// (exact match, no descendants), deliberately unlike reports (deptscope subtree).
// Every user mutation and GET /users/{id} use the same exact rule, so the list
// never shows a user the caller could not open or manage.

func TestListUsers_ISS184_ExactDepartment_ExcludesDescendantAndSiblingDepartments(t *testing.T) {
	repo := newMockRepo()
	repo.users["own"] = makeUser("own", "dept-parent", "role-emp", "employee")
	repo.users["child"] = makeUser("child", "dept-child", "role-emp", "employee") // descendant of dept-parent
	repo.users["other"] = makeUser("other", "dept-other", "role-emp", "employee")

	for _, role := range []string{"department_admin", customRole} {
		res, err := NewService(repo).ListUsers(context.Background(), role, "dept-parent", ListFilters{Page: 1, PerPage: 20})
		require.NoError(t, err)
		require.Len(t, res.Items, 1, role)
		assert.Equal(t, "own", res.Items[0].ID, role)
		assert.Equal(t, 1, res.Meta.Total, role)
	}

	res, err := NewService(repo).ListUsers(context.Background(), "super_admin", "", ListFilters{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 3, res.Meta.Total, "super_admin stays org-wide")
}

func TestListUsers_ISS184_DepartmentFilterCannotWidenScope(t *testing.T) {
	repo := newMockRepo()
	repo.users["own"] = makeUser("own", "dept-1", "role-emp", "employee")
	repo.users["other"] = makeUser("other", "dept-2", "role-emp", "employee")

	other := "dept-2"
	res, err := NewService(repo).ListUsers(context.Background(), "department_admin", "dept-1",
		ListFilters{Page: 1, PerPage: 20, DepartmentID: &other})
	require.NoError(t, err)
	assert.Empty(t, res.Items, "department_id filter is ANDed with the caller scope")
	assert.Equal(t, 0, res.Meta.Total)
}

func TestGetUser_ISS184_ChildDepartmentUserIsNotFound(t *testing.T) {
	repo := newMockRepo()
	repo.users["child"] = makeUser("child", "dept-child", "role-emp", "employee")
	_, err := NewService(repo).GetUser(context.Background(), "child", "department_admin", "caller", "dept-parent")
	assert.ErrorIs(t, err, ErrNotFound, "detail uses the same exact-department rule as the list")
}
