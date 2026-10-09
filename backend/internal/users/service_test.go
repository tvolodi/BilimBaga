package users

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- mock repository --------------------------------------------------------

type mockRepo struct {
	users      map[string]*User
	depts      map[string]string // name → id
	roles      map[string]string // name → id
	roleByID   map[string]string // id → name
	createFn   func(ctx context.Context, email, fullName, hash string, deptID *string, roleID string) (*User, error)
	deactivate map[string]bool
	revoked    []string
	unlocked   []string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:    make(map[string]*User),
		depts:    map[string]string{"Engineering": "dept-1", "Sales": "dept-2"},
		roles:    map[string]string{"employee": "role-emp", "department_admin": "role-da", "super_admin": "role-sa", "examiner": "role-ex"},
		roleByID: map[string]string{"role-emp": "employee", "role-da": "department_admin", "role-sa": "super_admin", "role-ex": "examiner"},
	}
}

func (m *mockRepo) List(_ context.Context, f ListFilters, deptScope *string) ([]User, int, error) {
	var out []User
	for _, u := range m.users {
		if deptScope != nil && (u.DepartmentID == nil || *u.DepartmentID != *deptScope) {
			continue
		}
		if f.DepartmentID != nil && (u.DepartmentID == nil || *u.DepartmentID != *f.DepartmentID) {
			continue
		}
		if f.RoleID != nil && u.RoleID != *f.RoleID {
			continue
		}
		if f.Status != nil && u.Status != *f.Status {
			continue
		}
		out = append(out, *u)
	}
	total := len(out)
	start := (f.Page - 1) * f.PerPage
	if start >= total {
		return []User{}, total, nil
	}
	end := start + f.PerPage
	if end > total {
		end = total
	}
	return out[start:end], total, nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *mockRepo) Create(ctx context.Context, email, fullName, hash string, deptID *string, roleID string) (*User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, email, fullName, hash, deptID, roleID)
	}
	for _, u := range m.users {
		if u.Email == email {
			return nil, ErrDuplicateEmail
		}
	}
	id := fmt.Sprintf("user-%d", len(m.users)+1)
	deptName := ""
	for n, did := range m.depts {
		if deptID != nil && did == *deptID {
			deptName = n
		}
	}
	roleName := m.roleByID[roleID]
	u := &User{
		ID:                  id,
		Email:               email,
		FullName:            fullName,
		DepartmentID:        deptID,
		DepartmentName:      &deptName,
		RoleID:              roleID,
		RoleName:            roleName,
		Status:              "active",
		ForcePasswordChange: true,
		CreatedAt:           time.Now(),
	}
	m.users[id] = u
	return u, nil
}

func (m *mockRepo) Update(_ context.Context, id, fullName string, deptID *string, roleID string) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	u.FullName = fullName
	u.DepartmentID = deptID
	u.RoleID = roleID
	u.RoleName = m.roleByID[roleID]
	return u, nil
}

func (m *mockRepo) Deactivate(_ context.Context, id string) error {
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	u.Status = "inactive"
	if m.deactivate == nil {
		m.deactivate = make(map[string]bool)
	}
	m.deactivate[id] = true
	return nil
}

func (m *mockRepo) RevokeAllTokens(_ context.Context, userID string) error {
	m.revoked = append(m.revoked, userID)
	return nil
}

func (m *mockRepo) UpdatePassword(_ context.Context, id, hash string) error {
	if _, ok := m.users[id]; !ok {
		return ErrNotFound
	}
	m.users[id].ForcePasswordChange = true
	return nil
}

func (m *mockRepo) Unlock(_ context.Context, id string) error {
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	u.IsLocked = false
	m.unlocked = append(m.unlocked, id)
	return nil
}

func (m *mockRepo) GetDepartmentIDByName(_ context.Context, name string) (string, error) {
	id, ok := m.depts[name]
	if !ok {
		return "", ErrNotFound
	}
	return id, nil
}

func (m *mockRepo) GetRoleIDByName(_ context.Context, name string) (string, error) {
	id, ok := m.roles[name]
	if !ok {
		return "", ErrNotFound
	}
	return id, nil
}

func (m *mockRepo) GetRoleNameByID(_ context.Context, roleID string) (string, error) {
	name, ok := m.roleByID[roleID]
	if !ok {
		return "", ErrNotFound
	}
	return name, nil
}

func (m *mockRepo) ListRoles(_ context.Context) ([]RoleRow, error) {
	rows := make([]RoleRow, 0, len(m.roleByID))
	for id, name := range m.roleByID {
		rows = append(rows, RoleRow{ID: id, Name: name})
	}
	return rows, nil
}

// helpers
func strPtr(s string) *string { return &s }

func makeUser(id, deptID, roleID, roleName string) *User {
	return &User{
		ID:             id,
		Email:          id + "@example.com",
		FullName:       "Test " + id,
		DepartmentID:   &deptID,
		DepartmentName: strPtr("Engineering"),
		RoleID:         roleID,
		RoleName:       roleName,
		Status:         "active",
		CreatedAt:      time.Now(),
	}
}

// ---- tests ------------------------------------------------------------------

func TestListUsers_Pagination(t *testing.T) {
	repo := newMockRepo()
	for i := 0; i < 5; i++ {
		u := makeUser(fmt.Sprintf("u%d", i), "dept-1", "role-emp", "employee")
		repo.users[u.ID] = u
	}
	svc := NewService(repo)

	result, err := svc.ListUsers(context.Background(), "super_admin", "", ListFilters{Page: 2, PerPage: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Meta.Total)
	assert.Equal(t, 2, result.Meta.Page)
	assert.Equal(t, 2, result.Meta.PerPage)
	assert.Len(t, result.Items, 2)
}

func TestListUsers_DefaultPerPage(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	result, err := svc.ListUsers(context.Background(), "super_admin", "", ListFilters{})
	require.NoError(t, err)
	assert.Equal(t, 20, result.Meta.PerPage)
	assert.Equal(t, 1, result.Meta.Page)
}

func TestListUsers_MaxPerPage(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	result, err := svc.ListUsers(context.Background(), "super_admin", "", ListFilters{PerPage: 999})
	require.NoError(t, err)
	assert.Equal(t, 100, result.Meta.PerPage)
}

func TestListUsers_DepartmentAdminScope(t *testing.T) {
	repo := newMockRepo()
	// Two users in dept-1, one in dept-2.
	u1 := makeUser("u1", "dept-1", "role-emp", "employee")
	u2 := makeUser("u2", "dept-1", "role-emp", "employee")
	u3 := makeUser("u3", "dept-2", "role-emp", "employee")
	repo.users["u1"] = u1
	repo.users["u2"] = u2
	repo.users["u3"] = u3

	svc := NewService(repo)
	result, err := svc.ListUsers(context.Background(), "department_admin", "dept-1", ListFilters{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Meta.Total)
	for _, u := range result.Items {
		assert.Equal(t, "dept-1", *u.DepartmentID)
	}
}

func TestCreateUser_TempPassword(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	resp, err := svc.CreateUser(context.Background(), CreateRequest{
		Email:        "new@example.com",
		FullName:     "New User",
		DepartmentID: strPtr("dept-1"),
		RoleID:       "role-emp",
	}, "super_admin", "", "caller-id", "127.0.0.1")

	require.NoError(t, err)
	assert.NotEmpty(t, resp.TemporaryPassword)
	assert.Len(t, resp.TemporaryPassword, 10)
	assert.True(t, resp.ForcePasswordChange)
	assert.Equal(t, "new@example.com", resp.Email)

}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	existing := makeUser("u1", "dept-1", "role-emp", "employee")
	existing.Email = "dup@example.com"
	repo.users["u1"] = existing

	svc := NewService(repo)
	_, err := svc.CreateUser(context.Background(), CreateRequest{
		Email:        "dup@example.com",
		FullName:     "Dup User",
		DepartmentID: strPtr("dept-1"),
		RoleID:       "role-emp",
	}, "super_admin", "", "caller-id", "127.0.0.1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateEmail))
}

func TestCreateUser_DeptAdminWrongDept(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	_, err := svc.CreateUser(context.Background(), CreateRequest{
		Email:        "x@example.com",
		FullName:     "X",
		DepartmentID: strPtr("dept-2"), // caller is dept-1
		RoleID:       "role-emp",
	}, "department_admin", "dept-1", "caller-id", "127.0.0.1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrForbidden))
}

func TestCreateUser_ValidationError(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	_, err := svc.CreateUser(context.Background(), CreateRequest{
		Email:    "not-an-email",
		FullName: "X",
		RoleID:   "role-emp",
	}, "super_admin", "", "caller-id", "127.0.0.1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrValidation))
}

func TestDeactivateUser_TokenRevocation(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-1", "role-emp", "employee")
	repo.users["u1"] = u

	svc := NewService(repo)
	err := svc.DeactivateUser(context.Background(), "u1", "super_admin", "", "caller-id", "127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, "inactive", repo.users["u1"].Status)
	assert.Contains(t, repo.revoked, "u1")

}

func TestDeactivateUser_DeptAdminWrongDept(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-2", "role-emp", "employee")
	repo.users["u1"] = u

	svc := NewService(repo)
	err := svc.DeactivateUser(context.Background(), "u1", "department_admin", "dept-1", "caller-id", "127.0.0.1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrForbidden))
}

func TestResetPassword(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-1", "role-emp", "employee")
	repo.users["u1"] = u

	svc := NewService(repo)
	resp, err := svc.ResetPassword(context.Background(), "u1", "super_admin", "", "caller-id", "127.0.0.1")

	require.NoError(t, err)
	assert.NotEmpty(t, resp.TemporaryPassword)
	assert.Len(t, resp.TemporaryPassword, 10)
	assert.True(t, repo.users["u1"].ForcePasswordChange)

}

func TestImportUsers_Preview(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	rows := []CSVRow{
		{RowNum: 2, Email: "a@example.com", FullName: "Alice", DepartmentName: "Engineering", RoleName: "employee"},
		{RowNum: 3, Email: "bad-email", FullName: "Bob", DepartmentName: "Engineering", RoleName: "employee"},
		{RowNum: 4, Email: "c@example.com", FullName: "Charlie", DepartmentName: "Unknown", RoleName: "employee"},
	}

	preview, err := svc.ImportUsers(context.Background(), rows, false, "super_admin", "", "caller-id", "127.0.0.1")
	require.NoError(t, err)

	assert.Len(t, preview.Valid, 1)
	assert.Len(t, preview.Errors, 2)
	// No users should have been created (preview only).
	assert.Empty(t, repo.users)
}

func TestImportUsers_Commit(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	rows := []CSVRow{
		{RowNum: 2, Email: "a@example.com", FullName: "Alice", DepartmentName: "Engineering", RoleName: "employee"},
		{RowNum: 3, Email: "bad@", FullName: "Bad", DepartmentName: "Engineering", RoleName: "employee"},
	}

	preview, err := svc.ImportUsers(context.Background(), rows, true, "super_admin", "", "caller-id", "127.0.0.1")
	require.NoError(t, err)

	assert.Len(t, preview.Valid, 1)
	assert.Len(t, preview.Errors, 1)
	assert.Len(t, repo.users, 1)
}

func TestImportUsers_DeptAdminScope(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	rows := []CSVRow{
		{RowNum: 2, Email: "a@example.com", FullName: "Alice", DepartmentName: "Engineering", RoleName: "employee"},
		{RowNum: 3, Email: "b@example.com", FullName: "Bob", DepartmentName: "Sales", RoleName: "employee"},
	}

	// dept-1 = Engineering; dept-2 = Sales
	preview, err := svc.ImportUsers(context.Background(), rows, false, "department_admin", "dept-1", "caller-id", "127.0.0.1")
	require.NoError(t, err)

	assert.Len(t, preview.Valid, 1)
	assert.Len(t, preview.Errors, 1)
	require.NotNil(t, preview.Errors[0].Error)
	assert.Contains(t, *preview.Errors[0].Error, "outside your scope")
}

func TestGenerateTempPassword(t *testing.T) {
	for i := 0; i < 10; i++ {
		pwd, err := generateTempPassword()
		require.NoError(t, err)
		assert.Len(t, pwd, 10)
		hasLower, hasUpper, hasDigit, hasSpecial := false, false, false, false
		for _, c := range pwd {
			switch {
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= '0' && c <= '9':
				hasDigit = true
			default:
				hasSpecial = true
			}
		}
		assert.True(t, hasLower, "password should contain lowercase letters")
		assert.True(t, hasUpper, "password should contain uppercase letters")
		assert.True(t, hasDigit, "password should contain digits")
		assert.True(t, hasSpecial, "password should contain special characters")
	}
}

// AC-5 (FR-BB64): every generated temporary password must satisfy the
// password complexity policy enforced on password change.
func TestGenerateTempPassword_PassesValidateComplexity(t *testing.T) {
	for i := 0; i < 2000; i++ {
		pwd, err := generateTempPassword()
		require.NoError(t, err)
		require.NoError(t, auth.ValidateComplexity(pwd), "generated password %q failed complexity", pwd)
	}
}

// ---- FR-BB115 AC-5: admin unlock --------------------------------------------

func TestUnlockUser_ClearsLockAndReturnsUpdatedUser(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-1", "role-emp", "employee")
	u.IsLocked = true
	repo.users["u1"] = u

	got, err := NewService(repo).UnlockUser(context.Background(), "u1", "super_admin", "", "admin-1", "127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, []string{"u1"}, repo.unlocked)
	assert.False(t, got.IsLocked)
}

func TestUnlockUser_UnknownID_ReturnsNotFound(t *testing.T) {
	_, err := NewService(newMockRepo()).UnlockUser(context.Background(), "missing", "super_admin", "", "admin-1", "")
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestUnlockUser_DeptAdminOtherDept_ReturnsForbidden(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-2", "role-emp", "employee")
	u.IsLocked = true
	repo.users["u1"] = u

	_, err := NewService(repo).UnlockUser(context.Background(), "u1", "department_admin", "dept-1", "da-1", "")

	assert.True(t, errors.Is(err, ErrForbidden))
	assert.Empty(t, repo.unlocked)
	assert.True(t, repo.users["u1"].IsLocked)
}

func TestUnlockUser_DeptAdminOwnDept_Succeeds(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-1", "role-emp", "employee")
	u.IsLocked = true
	repo.users["u1"] = u

	_, err := NewService(repo).UnlockUser(context.Background(), "u1", "department_admin", "dept-1", "da-1", "")

	require.NoError(t, err)
	assert.Equal(t, []string{"u1"}, repo.unlocked)
}
