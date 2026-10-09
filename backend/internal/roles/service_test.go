package roles

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pUsersRead   = "11111111-1111-4111-8111-000000000001"
	pExamsRead   = "11111111-1111-4111-8111-000000000002"
	pRolesRead   = "11111111-1111-4111-8111-000000000003"
	pRolesManage = "11111111-1111-4111-8111-000000000004"
	pTenant      = "11111111-1111-4111-8111-000000000005"
	pUnknown     = "11111111-1111-4111-8111-0000000000ff"
	sysRoleID    = "22222222-2222-4222-8222-000000000001"
	customID     = "22222222-2222-4222-8222-000000000002"
)

type mockRepo struct {
	roles    map[string]*Role
	perms    map[string]Permission
	users    map[string]int
	nextID   int
	deleted  []string
	createFn func() error
}

func newMockRepo() *mockRepo {
	m := &mockRepo{
		roles: map[string]*Role{
			sysRoleID: {ID: sysRoleID, Name: "super_admin", IsSystem: true, Permissions: []string{"users:read"}},
			customID:  {ID: customID, Name: "qa_lead", Permissions: []string{"users:read"}},
		},
		perms: map[string]Permission{
			pUsersRead:   {ID: pUsersRead, Resource: "users", Action: "read"},
			pExamsRead:   {ID: pExamsRead, Resource: "exams", Action: "read"},
			pRolesRead:   {ID: pRolesRead, Resource: "roles", Action: "read"},
			pRolesManage: {ID: pRolesManage, Resource: "roles", Action: "manage"},
			pTenant:      {ID: pTenant, Resource: "tenant", Action: "manage"},
		},
		users: map[string]int{},
	}
	return m
}

func (m *mockRepo) List(context.Context) ([]Role, error) {
	var out []Role
	for _, r := range m.roles {
		out = append(out, *r)
	}
	return out, nil
}
func (m *mockRepo) GetByID(_ context.Context, id string) (*Role, error) {
	r, ok := m.roles[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *r
	cp.UserCount = m.users[id]
	return &cp, nil
}
func (m *mockRepo) ListPermissions(context.Context) ([]Permission, error) {
	var out []Permission
	for _, p := range m.perms {
		out = append(out, p)
	}
	return out, nil
}
func (m *mockRepo) PermissionsByIDs(_ context.Context, ids []string) ([]Permission, error) {
	var out []Permission
	for _, id := range ids {
		if p, ok := m.perms[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *mockRepo) keys(ids []string) []string {
	out := []string{}
	for _, id := range ids {
		out = append(out, m.perms[id].Key())
	}
	return out
}
func (m *mockRepo) Create(_ context.Context, name, desc string, ids []string) (string, error) {
	if m.createFn != nil {
		if err := m.createFn(); err != nil {
			return "", err
		}
	}
	for _, r := range m.roles {
		if r.Name == name {
			return "", ErrNameTaken
		}
	}
	m.nextID++
	id := fmt.Sprintf("33333333-3333-4333-8333-%012d", m.nextID)
	m.roles[id] = &Role{ID: id, Name: name, Description: desc, Permissions: m.keys(ids)}
	return id, nil
}
func (m *mockRepo) Update(_ context.Context, id, desc string, ids []string) error {
	r, ok := m.roles[id]
	if !ok || r.IsSystem {
		return ErrNotFound
	}
	r.Description = desc
	r.Permissions = m.keys(ids)
	return nil
}
func (m *mockRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.roles[id]; !ok {
		return ErrNotFound
	}
	delete(m.roles, id)
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockRepo) CountUsers(_ context.Context, id string) (int, error) { return m.users[id], nil }

func newSvc(repo *mockRepo, reloads *int, reloadErr error) Service {
	return NewService(repo, func(context.Context) error {
		*reloads++
		return reloadErr
	})
}

func TestCreate_Valid_ReloadsCache(t *testing.T) {
	repo, n := newMockRepo(), 0
	role, err := newSvc(repo, &n, nil).Create(context.Background(),
		CreateRequest{Name: "exam_viewer", Description: "d", Permissions: []string{pUsersRead, pExamsRead, pUsersRead}})
	require.NoError(t, err)
	assert.Equal(t, "exam_viewer", role.Name)
	assert.False(t, role.IsSystem)
	assert.ElementsMatch(t, []string{"users:read", "exams:read"}, role.Permissions)
	assert.Equal(t, 1, n)
}

func TestCreate_BadNames(t *testing.T) {
	for _, name := range []string{"", "ab", "Admin", "1abc", "has space", "has-dash", "a" + string(make([]byte, 0)) + "bcdefghijklmnopqrstuvwxyz0123456789"} {
		repo, n := newMockRepo(), 0
		_, err := newSvc(repo, &n, nil).Create(context.Background(), CreateRequest{Name: name})
		assert.ErrorIs(t, err, ErrValidation, name)
		assert.Equal(t, 0, n)
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Create(context.Background(), CreateRequest{Name: "qa_lead"})
	assert.ErrorIs(t, err, ErrNameTaken)
	assert.Equal(t, 0, n, "no cache reload on failure")
}

func TestCreate_UnknownAndMalformedPermission(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Create(context.Background(), CreateRequest{Name: "abc", Permissions: []string{pUnknown}})
	assert.ErrorIs(t, err, ErrValidation)
	_, err = newSvc(repo, &n, nil).Create(context.Background(), CreateRequest{Name: "abc", Permissions: []string{"not-a-uuid"}})
	assert.ErrorIs(t, err, ErrValidation)
	assert.Len(t, repo.roles, 2, "nothing written")
}

func TestCreate_ForbiddenPermissions_PrivilegeEscalationGuard(t *testing.T) {
	for _, id := range []string{pRolesRead, pRolesManage, pTenant} {
		repo, n := newMockRepo(), 0
		_, err := newSvc(repo, &n, nil).Create(context.Background(), CreateRequest{Name: "abc", Permissions: []string{pUsersRead, id}})
		require.ErrorIs(t, err, ErrValidation)
		assert.Contains(t, err.Error(), repo.perms[id].Key(), "names the offending permission")
		assert.Len(t, repo.roles, 2)
	}
}

func TestCreate_CacheReloadFailure(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, errors.New("db down")).Create(context.Background(), CreateRequest{Name: "abc"})
	assert.ErrorIs(t, err, ErrCacheReload)
	assert.Len(t, repo.roles, 3, "DB change stays committed")
}

func TestUpdate_Custom_ReplacesPermissionsAndDiffs(t *testing.T) {
	repo, n := newMockRepo(), 0
	name := "qa_lead"
	res, err := newSvc(repo, &n, nil).Update(context.Background(), customID,
		UpdateRequest{Name: &name, Description: "new", Permissions: []string{pExamsRead}})
	require.NoError(t, err)
	assert.Equal(t, "new", res.Role.Description)
	assert.Equal(t, []string{"exams:read"}, res.PermissionsAdded)
	assert.Equal(t, []string{"users:read"}, res.PermissionsRemoved)
	assert.Equal(t, 1, n)
}

func TestUpdate_SystemRoleImmutable(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Update(context.Background(), sysRoleID, UpdateRequest{Permissions: []string{pExamsRead}})
	assert.ErrorIs(t, err, ErrSystemImmutable)
	assert.Equal(t, []string{"users:read"}, repo.roles[sysRoleID].Permissions)
	assert.Equal(t, 0, n)
}

func TestUpdate_NameChangeRejected(t *testing.T) {
	repo, n := newMockRepo(), 0
	other := "renamed"
	_, err := newSvc(repo, &n, nil).Update(context.Background(), customID, UpdateRequest{Name: &other, Permissions: []string{}})
	assert.ErrorIs(t, err, ErrValidation)
	assert.Equal(t, "qa_lead", repo.roles[customID].Name)
}

func TestUpdate_ForbiddenPermissionAndMissingPermissions(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Update(context.Background(), customID, UpdateRequest{Permissions: []string{pRolesManage}})
	assert.ErrorIs(t, err, ErrValidation)
	_, err = newSvc(repo, &n, nil).Update(context.Background(), customID, UpdateRequest{})
	assert.ErrorIs(t, err, ErrValidation, "permissions required")
	assert.Equal(t, []string{"users:read"}, repo.roles[customID].Permissions)
}

func TestUpdate_NotFound(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Update(context.Background(), pUnknown, UpdateRequest{Permissions: []string{}})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDelete_OK_ReloadsCache(t *testing.T) {
	repo, n := newMockRepo(), 0
	deleted, err := newSvc(repo, &n, nil).Delete(context.Background(), customID)
	require.NoError(t, err)
	assert.Equal(t, "qa_lead", deleted.Name)
	assert.Equal(t, []string{customID}, repo.deleted)
	assert.Equal(t, 1, n)
}

func TestDelete_SystemRole409(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Delete(context.Background(), sysRoleID)
	assert.ErrorIs(t, err, ErrSystemImmutable)
	assert.Empty(t, repo.deleted)
}

func TestDelete_InUseReportsCount(t *testing.T) {
	repo, n := newMockRepo(), 0
	repo.users[customID] = 3
	_, err := newSvc(repo, &n, nil).Delete(context.Background(), customID)
	require.ErrorIs(t, err, ErrInUse)
	var inUse *InUseError
	require.True(t, errors.As(err, &inUse))
	assert.Equal(t, 3, inUse.Count)
	assert.Contains(t, err.Error(), "3")
	assert.Empty(t, repo.deleted)
	assert.Equal(t, 0, n)
}

func TestDelete_NotFound(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, nil).Delete(context.Background(), pUnknown)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDelete_CacheReloadFailure(t *testing.T) {
	repo, n := newMockRepo(), 0
	_, err := newSvc(repo, &n, errors.New("boom")).Delete(context.Background(), customID)
	assert.ErrorIs(t, err, ErrCacheReload)
}

func TestListAndGet(t *testing.T) {
	repo, n := newMockRepo(), 0
	svc := newSvc(repo, &n, nil)
	l, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, l, 2)
	p, err := svc.ListPermissions(context.Background())
	require.NoError(t, err)
	assert.Len(t, p, 5)
	_, err = svc.Get(context.Background(), pUnknown)
	assert.ErrorIs(t, err, ErrNotFound)
}
