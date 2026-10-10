package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	// FR-BB510
	exams      map[string]bool
	overdue    map[string]bool // userID+"|"+examID
	lastRemind *time.Time
	reminders  []string
	outOfScope map[string]bool // #253: users outside the caller's department subtree
	insertErr  error
	ambiguous  map[string]bool // department names that resolve to more than one row
	// setLocaleErr, when set, is returned by SetPreferredLocale (FR-BB116 write-failure path).
	setLocaleErr error
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
		if strings.EqualFold(u.Email, email) { // mirrors the lower(email) duplicate probe
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

func (m *mockRepo) Reactivate(_ context.Context, id string) error {
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	u.Status = "active"
	return nil
}

func (m *mockRepo) RevokeAllTokens(_ context.Context, userID string) error {
	m.revoked = append(m.revoked, userID)
	return nil
}

func (m *mockRepo) UpdatePassword(_ context.Context, id, hash string, _ time.Time) error {
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
	if m.ambiguous[name] {
		return "", ErrAmbiguousName
	}
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
	assert.True(t, errors.Is(err, ErrNotFound))
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

	assert.True(t, errors.Is(err, ErrNotFound))
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

// ISS-133: the role filter returns only users holding that role id.
func TestListUsers_RoleIDFilter(t *testing.T) {
	repo := newMockRepo()
	repo.users["u1"] = makeUser("u1", "dept-1", "role-emp", "employee")
	repo.users["u2"] = makeUser("u2", "dept-1", "role-ex", "examiner")
	repo.users["u3"] = makeUser("u3", "dept-2", "role-ex", "examiner")
	svc := NewService(repo)

	roleID := "role-ex"
	result, err := svc.ListUsers(context.Background(), "super_admin", "", ListFilters{Page: 1, PerPage: 20, RoleID: &roleID})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Meta.Total)
	for _, u := range result.Items {
		assert.Equal(t, "role-ex", u.RoleID)
	}
}

// ISS-164: emails are normalised (trim + lowercase) on every write path.

func TestCreateUser_NormalisesMixedCaseAndWhitespaceEmail(t *testing.T) {
	repo := newMockRepo()
	var got string
	repo.createFn = func(_ context.Context, email, fullName, hash string, deptID *string, roleID string) (*User, error) {
		got = email
		return &User{ID: "new", Email: email, FullName: fullName}, nil
	}
	svc := NewService(repo)

	resp, err := svc.CreateUser(context.Background(), CreateRequest{
		Email: "  John.Doe@Corp.com ", FullName: "John", DepartmentID: strPtr("dept-1"), RoleID: "role-emp",
	}, "super_admin", "", "caller-id", "127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, "john.doe@corp.com", got)
	assert.Equal(t, "john.doe@corp.com", resp.Email)
}

func TestImportUsers_Commit_NormalisesEmail(t *testing.T) {
	repo := newMockRepo()
	var got []string
	repo.createFn = func(_ context.Context, email, fullName, hash string, deptID *string, roleID string) (*User, error) {
		got = append(got, email)
		return &User{ID: "new" + email, Email: email, FullName: fullName}, nil
	}
	svc := NewService(repo)

	rows := []CSVRow{
		{RowNum: 2, Email: "Alice@Example.COM", FullName: "Alice", DepartmentName: "Engineering", RoleName: "employee"},
		{RowNum: 3, Email: "  Bob@Example.com ", FullName: "Bob", DepartmentName: "Engineering", RoleName: "employee"},
	}
	preview, err := svc.ImportUsers(context.Background(), rows, true, "super_admin", "", "caller-id", "127.0.0.1")
	require.NoError(t, err)
	assert.Len(t, preview.Errors, 0)
	assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, got)
	assert.Equal(t, "alice@example.com", preview.Valid[0].Email)
}

// ISS-164 cycle 2: a legacy mixed-case row must still block its lowercase twin.
func TestCreateUser_LegacyMixedCaseRow_BlocksLowercaseTwin(t *testing.T) {
	repo := newMockRepo()
	repo.users["legacy"] = &User{ID: "legacy", Email: "John.Doe@Corp.com"}
	svc := NewService(repo)

	_, err := svc.CreateUser(context.Background(), CreateRequest{
		Email: "john.doe@corp.com", FullName: "John", DepartmentID: strPtr("dept-1"), RoleID: "role-emp",
	}, "super_admin", "", "caller-id", "127.0.0.1")

	assert.True(t, errors.Is(err, ErrDuplicateEmail))
}

func (m *mockRepo) SetPreferredLocale(_ context.Context, id string, locale *string) error {
	if m.setLocaleErr != nil {
		return m.setLocaleErr
	}
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	var v *string
	if locale != nil {
		c := *locale
		v = &c
	}
	u.PreferredLocale = v
	return nil
}

// ---- FR-BB116: self-service preferred locale ---------------------------------

// fixedLocales is a LocaleSource returning a fixed tenant available_locales list.
type fixedLocales []string

func (f fixedLocales) GetAvailableLocales() []string { return f }

// localeRepo holds two employees: u1 with no preference, u2 preferring kk.
func localeRepo() *mockRepo {
	repo := newMockRepo()
	repo.users["u1"] = &User{ID: "u1", Email: "u1@example.com", FullName: "U One", RoleID: "role-emp", RoleName: "employee", Status: "active"}
	repo.users["u2"] = &User{ID: "u2", Email: "u2@example.com", FullName: "U Two", RoleID: "role-emp", RoleName: "employee", Status: "active", PreferredLocale: strPtr("kk")}
	return repo
}

var tenantLocales = fixedLocales{"kk", "ru", "en"}

// AC-2: a code in available_locales is persisted and returned on the updated user.
func TestUpdateMyLocale_SetsAvailableLocale(t *testing.T) {
	repo := localeRepo()
	svc := WithLocaleSource(NewService(repo), tenantLocales)

	res, err := svc.UpdateMyLocale(context.Background(), "u1", strPtr("ru"))
	require.NoError(t, err)
	require.NotNil(t, res.User.PreferredLocale)
	assert.Equal(t, "ru", *res.User.PreferredLocale)
	assert.Nil(t, res.Previous, "u1 had no preference before")
	require.NotNil(t, repo.users["u1"].PreferredLocale)
	assert.Equal(t, "ru", *repo.users["u1"].PreferredLocale)
}

// AC-2: a code outside available_locales is a validation error and nothing is written.
func TestUpdateMyLocale_RejectsLocaleOutsideAvailable(t *testing.T) {
	repo := localeRepo()
	svc := WithLocaleSource(NewService(repo), tenantLocales)

	for _, bad := range []string{"de", "", "RU", "ru-RU"} {
		_, err := svc.UpdateMyLocale(context.Background(), "u1", strPtr(bad))
		require.Error(t, err, bad)
		assert.ErrorIs(t, err, ErrValidation, bad)
		// The handler relays this message as the 422 body, so it must name the field.
		assert.Contains(t, err.Error(), "preferred_locale", bad)
	}
	assert.Nil(t, repo.users["u1"].PreferredLocale, "rejected value must not be persisted")
}

// AC-2: null clears the preference; Previous reports the value that was cleared.
func TestUpdateMyLocale_NullClearsPreference(t *testing.T) {
	repo := localeRepo()
	svc := WithLocaleSource(NewService(repo), tenantLocales)

	res, err := svc.UpdateMyLocale(context.Background(), "u2", nil)
	require.NoError(t, err)
	assert.Nil(t, res.User.PreferredLocale)
	require.NotNil(t, res.Previous)
	assert.Equal(t, "kk", *res.Previous)
	assert.Nil(t, repo.users["u2"].PreferredLocale)
}

// Fail-closed: without a tenant locale source every non-null code is refused, while null
// (which needs no lookup) still clears.
func TestUpdateMyLocale_NoLocaleSourceFailsClosed(t *testing.T) {
	repo := localeRepo()
	svc := NewService(repo)

	_, err := svc.UpdateMyLocale(context.Background(), "u1", strPtr("en"))
	assert.ErrorIs(t, err, ErrValidation)

	_, err = svc.UpdateMyLocale(context.Background(), "u2", nil)
	require.NoError(t, err)
	assert.Nil(t, repo.users["u2"].PreferredLocale)
}

// AC-3 (service side): the write only ever addresses the given caller id; another user's
// preference is untouched.
func TestUpdateMyLocale_OnlyTouchesCallerRecord(t *testing.T) {
	repo := localeRepo()
	svc := WithLocaleSource(NewService(repo), tenantLocales)

	_, err := svc.UpdateMyLocale(context.Background(), "u1", strPtr("ru"))
	require.NoError(t, err)
	require.NotNil(t, repo.users["u2"].PreferredLocale)
	assert.Equal(t, "kk", *repo.users["u2"].PreferredLocale)
}

func TestUpdateMyLocale_UnknownUserIsNotFound(t *testing.T) {
	svc := WithLocaleSource(NewService(localeRepo()), tenantLocales)

	_, err := svc.UpdateMyLocale(context.Background(), "ghost", strPtr("ru"))
	assert.ErrorIs(t, err, ErrNotFound)
}

// Write failures are wrapped with context and are not reported as validation errors.
func TestUpdateMyLocale_RepoWriteErrorIsWrapped(t *testing.T) {
	repo := localeRepo()
	boom := errors.New("connection reset")
	repo.setLocaleErr = boom
	svc := WithLocaleSource(NewService(repo), tenantLocales)

	_, err := svc.UpdateMyLocale(context.Background(), "u1", strPtr("ru"))
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, ErrValidation)
	assert.Contains(t, err.Error(), "UpdateMyLocale")
}

// AC-1 (service side): the stored preference is exposed on the profile and the list/detail
// reads, for every caller that can see the record.
func TestPreferredLocale_ExposedOnReads(t *testing.T) {
	svc := NewService(localeRepo())
	ctx := context.Background()

	me, err := svc.GetMe(ctx, "u2")
	require.NoError(t, err)
	require.NotNil(t, me.PreferredLocale)
	assert.Equal(t, "kk", *me.PreferredLocale)

	detail, err := svc.GetUser(ctx, "u2", "super_admin", "admin", "")
	require.NoError(t, err)
	require.NotNil(t, detail.PreferredLocale)
	assert.Equal(t, "kk", *detail.PreferredLocale)

	list, err := svc.ListUsers(ctx, "super_admin", "", ListFilters{Page: 1, PerPage: 20})
	require.NoError(t, err)
	var found bool
	for _, u := range list.Items {
		if u.ID == "u2" {
			found = true
			require.NotNil(t, u.PreferredLocale)
			assert.Equal(t, "kk", *u.PreferredLocale)
		}
	}
	assert.True(t, found, "u2 must appear in the admin list")
}

// ── FR-BB18 AC-13: reactivate ────────────────────────────────────────────────

func TestReactivateUser_RestoresInactiveUserWithoutRevokingTokens(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-1", "role-emp", "employee")
	u.Status = "inactive"
	repo.users["u1"] = u

	svc := NewService(repo)
	err := svc.ReactivateUser(context.Background(), "u1", "super_admin", "", "caller-id", "127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, "active", repo.users["u1"].Status)
	assert.Empty(t, repo.revoked, "reactivation must not revoke refresh tokens")
}

func TestReactivateUser_AlreadyActiveIsConflictWithNoChange(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-1", "role-emp", "employee")
	u.Status = "active"
	repo.users["u1"] = u

	svc := NewService(repo)
	err := svc.ReactivateUser(context.Background(), "u1", "super_admin", "", "caller-id", "127.0.0.1")

	assert.ErrorIs(t, err, ErrUserAlreadyActive)
	assert.Equal(t, "active", repo.users["u1"].Status)
}

func TestReactivateUser_DeptAdminOutOfDepartmentIsNotFound(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u1", "dept-2", "role-emp", "employee")
	u.Status = "inactive"
	repo.users["u1"] = u

	svc := NewService(repo)
	err := svc.ReactivateUser(context.Background(), "u1", "department_admin", "dept-1", "caller-id", "127.0.0.1")

	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, "inactive", repo.users["u1"].Status, "an out-of-scope target must not change")
}

func TestReactivateUser_PeerRankIsForbiddenBeforeWrite(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("u2", "dept-1", "role-da", "department_admin")
	u.Status = "inactive"
	repo.users["u2"] = u

	svc := NewService(repo)
	err := svc.ReactivateUser(context.Background(), "u2", "department_admin", "dept-1", "caller-id", "127.0.0.1")

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "inactive", repo.users["u2"].Status, "a rank violation must not write")
}

func TestReactivateUser_SelfIsForbidden(t *testing.T) {
	repo := newMockRepo()
	u := makeUser("caller-id", "dept-1", "role-emp", "employee")
	u.Status = "inactive"
	repo.users["caller-id"] = u

	svc := NewService(repo)
	err := svc.ReactivateUser(context.Background(), "caller-id", "super_admin", "", "caller-id", "127.0.0.1")

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "inactive", repo.users["caller-id"].Status)
}
