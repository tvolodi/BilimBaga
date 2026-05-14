package departments

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type mockRepo struct {
	depts      []Department
	getByIDFn  func(ctx context.Context, id string) (*Department, error)
	createFn   func(ctx context.Context, name string, parentID *string) (*Department, error)
	updateFn   func(ctx context.Context, id, name string) (*Department, error)
	hasUsersFn func(ctx context.Context, id string) (bool, error)
	hasChildFn func(ctx context.Context, id string) (bool, error)
	deleteFn   func(ctx context.Context, id string) error
	auditLog   []string
}

func (m *mockRepo) GetAll(_ context.Context) ([]Department, error) {
	return m.depts, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*Department, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	for i := range m.depts {
		if m.depts[i].ID == id {
			return &m.depts[i], nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Create(ctx context.Context, name string, parentID *string) (*Department, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name, parentID)
	}
	d := Department{ID: "new-id", Name: name, ParentID: parentID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.depts = append(m.depts, d)
	return &d, nil
}

func (m *mockRepo) Update(ctx context.Context, id, name string) (*Department, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, name)
	}
	for i := range m.depts {
		if m.depts[i].ID == id {
			m.depts[i].Name = name
			return &m.depts[i], nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) HasUsers(ctx context.Context, id string) (bool, error) {
	if m.hasUsersFn != nil {
		return m.hasUsersFn(ctx, id)
	}
	return false, nil
}

func (m *mockRepo) HasChildren(ctx context.Context, id string) (bool, error) {
	if m.hasChildFn != nil {
		return m.hasChildFn(ctx, id)
	}
	return false, nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockRepo) WriteAuditLog(_ context.Context, _ *string, action, _ string, _ map[string]any) error {
	m.auditLog = append(m.auditLog, action)
	return nil
}

// --- buildTree tests ---

func TestBuildTree_EmptyInput(t *testing.T) {
	result := buildTree(nil)
	assert.Empty(t, result)
}

func TestBuildTree_RootsOnly(t *testing.T) {
	now := time.Now()
	depts := []Department{
		{ID: "a", Name: "Alpha", CreatedAt: now, UpdatedAt: now},
		{ID: "b", Name: "Beta", CreatedAt: now, UpdatedAt: now},
	}
	tree := buildTree(depts)
	require.Len(t, tree, 2)
	assert.Equal(t, "Alpha", tree[0].Name)
	assert.Equal(t, "Beta", tree[1].Name)
	assert.Empty(t, tree[0].Children)
	assert.Empty(t, tree[1].Children)
}

func TestBuildTree_NestedChildren(t *testing.T) {
	now := time.Now()
	parentID := "p1"
	depts := []Department{
		{ID: "p1", Name: "Parent", CreatedAt: now, UpdatedAt: now},
		{ID: "c1", Name: "Child", ParentID: &parentID, CreatedAt: now, UpdatedAt: now},
	}
	tree := buildTree(depts)
	require.Len(t, tree, 1)
	assert.Equal(t, "Parent", tree[0].Name)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "Child", tree[0].Children[0].Name)
}

func TestBuildTree_OrphanedNodeBecomesRoot(t *testing.T) {
	now := time.Now()
	missingParent := "missing"
	depts := []Department{
		{ID: "c1", Name: "Orphan", ParentID: &missingParent, CreatedAt: now, UpdatedAt: now},
	}
	tree := buildTree(depts)
	require.Len(t, tree, 1)
	assert.Equal(t, "Orphan", tree[0].Name)
}

// --- ListTree ---

func TestListTree_ReturnsTree(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		depts: []Department{
			{ID: "1", Name: "Root", CreatedAt: now, UpdatedAt: now},
		},
	}
	svc := NewService(repo)
	tree, err := svc.ListTree(context.Background())
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "Root", tree[0].Name)
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	node, err := svc.Create(context.Background(), CreateRequest{Name: "Engineering"}, "user-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "Engineering", node.Name)
	assert.Empty(t, node.Children)
	assert.Contains(t, repo.auditLog, "department.create")
}

func TestCreate_WithParent_ParentNotFound(t *testing.T) {
	parentID := "nonexistent"
	repo := &mockRepo{}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateRequest{Name: "Child", ParentID: &parentID}, "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestCreate_DuplicateName(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ string, _ *string) (*Department, error) {
			return nil, ErrDuplicateName
		},
	}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateRequest{Name: "Duplicate"}, "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateName))
}

// --- Update ---

func TestUpdate_Success(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		depts: []Department{
			{ID: "d1", Name: "Old Name", CreatedAt: now, UpdatedAt: now},
		},
	}
	svc := NewService(repo)

	node, err := svc.Update(context.Background(), "d1", UpdateRequest{Name: "New Name"}, "user-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "New Name", node.Name)
	assert.Contains(t, repo.auditLog, "department.update")
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	_, err := svc.Update(context.Background(), "nonexistent", UpdateRequest{Name: "X"}, "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestUpdate_DuplicateName(t *testing.T) {
	repo := &mockRepo{
		updateFn: func(_ context.Context, _, _ string) (*Department, error) {
			return nil, ErrDuplicateName
		},
	}
	svc := NewService(repo)

	_, err := svc.Update(context.Background(), "d1", UpdateRequest{Name: "Taken"}, "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateName))
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		depts: []Department{{ID: "d1", Name: "Dept", CreatedAt: now, UpdatedAt: now}},
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "d1", "user-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Contains(t, repo.auditLog, "department.delete")
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "missing", "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestDelete_HasChildren(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		depts:      []Department{{ID: "d1", Name: "Dept", CreatedAt: now, UpdatedAt: now}},
		hasChildFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "d1", "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDepartmentHasChildren))
}

func TestDelete_HasUsers(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		depts:      []Department{{ID: "d1", Name: "Dept", CreatedAt: now, UpdatedAt: now}},
		hasUsersFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "d1", "user-1", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDepartmentNotEmpty))
}
