package categories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mockRepo ---

type mockRepo struct {
	rows         []Category
	createFn     func(ctx context.Context, name string, parentID, track *string, sortOrder int) (*Category, error)
	updateFn     func(ctx context.Context, c Category) (*Category, error)
	deleteFn     func(ctx context.Context, id string) error
	hasChildren  func(ctx context.Context, id string) (bool, error)
	hasQuestions func(ctx context.Context, id string) (bool, error)
}

func (m *mockRepo) GetAll(_ context.Context) ([]Category, error) { return m.rows, nil }

func (m *mockRepo) GetByID(_ context.Context, id string) (*Category, error) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			row := m.rows[i]
			return &row, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Create(ctx context.Context, name string, parentID, track *string, sortOrder int) (*Category, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name, parentID, track, sortOrder)
	}
	c := Category{
		ID: "new-id", Name: name, ParentID: parentID, Track: track,
		SortOrder: sortOrder, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.rows = append(m.rows, c)
	return &c, nil
}

func (m *mockRepo) Update(ctx context.Context, c Category) (*Category, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, c)
	}
	for i := range m.rows {
		if m.rows[i].ID == c.ID {
			m.rows[i] = c
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) HasChildren(ctx context.Context, id string) (bool, error) {
	if m.hasChildren != nil {
		return m.hasChildren(ctx, id)
	}
	return false, nil
}

func (m *mockRepo) HasQuestions(ctx context.Context, id string) (bool, error) {
	if m.hasQuestions != nil {
		return m.hasQuestions(ctx, id)
	}
	return false, nil
}

func ptr[T any](v T) *T { return &v }

// --- buildTree / isDescendant ---

func TestBuildTree_EmptyInput(t *testing.T) {
	assert.Empty(t, buildTree(nil))
}

func TestBuildTree_NestedAndOrphans(t *testing.T) {
	now := time.Now()
	rows := []Category{
		{ID: "r1", Name: "Root", CreatedAt: now, UpdatedAt: now},
		{ID: "c1", Name: "Child", ParentID: ptr("r1"), CreatedAt: now, UpdatedAt: now},
		{ID: "o1", Name: "Orphan", ParentID: ptr("missing"), CreatedAt: now, UpdatedAt: now},
	}
	tree := buildTree(rows)
	require.Len(t, tree, 2)
	assert.Equal(t, "Root", tree[0].Name)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "Child", tree[0].Children[0].Name)
	assert.Equal(t, "Orphan", tree[1].Name)
}

func TestIsDescendant_DetectsCycle(t *testing.T) {
	rows := []Category{
		{ID: "a"},
		{ID: "b", ParentID: ptr("a")},
		{ID: "c", ParentID: ptr("b")},
	}
	assert.True(t, isDescendant(rows, "a", "c"))
	assert.False(t, isDescendant(rows, "c", "a"))
}

// --- Create ---

func TestCreate_TrimsName(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	node, err := svc.Create(context.Background(), CreateRequest{Name: "  Phishing  "})
	require.NoError(t, err)
	assert.Equal(t, "Phishing", node.Name)
}

func TestCreate_InvalidName(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), CreateRequest{Name: "   "})
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestCreate_ParentNotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), CreateRequest{Name: "x", ParentID: ptr("missing")})
	assert.ErrorIs(t, err, ErrParentNotFound)
}

// --- Update ---

func TestUpdate_RejectsSelfParent(t *testing.T) {
	repo := &mockRepo{rows: []Category{{ID: "a", Name: "A"}}}
	svc := NewService(repo)
	_, err := svc.Update(context.Background(), "a", UpdateRequest{ParentID: ptr("a")})
	assert.ErrorIs(t, err, ErrCycle)
}

func TestUpdate_RejectsDescendantParent(t *testing.T) {
	repo := &mockRepo{rows: []Category{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B", ParentID: ptr("a")},
	}}
	svc := NewService(repo)
	// Trying to set A's parent to B (its own child) must fail.
	_, err := svc.Update(context.Background(), "a", UpdateRequest{ParentID: ptr("b")})
	assert.ErrorIs(t, err, ErrCycle)
}

func TestUpdate_ChangesName(t *testing.T) {
	repo := &mockRepo{rows: []Category{{ID: "a", Name: "Old"}}}
	svc := NewService(repo)
	node, err := svc.Update(context.Background(), "a", UpdateRequest{Name: ptr("New")})
	require.NoError(t, err)
	assert.Equal(t, "New", node.Name)
}

func TestUpdate_ClearParent(t *testing.T) {
	repo := &mockRepo{rows: []Category{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B", ParentID: ptr("a")},
	}}
	svc := NewService(repo)
	node, err := svc.Update(context.Background(), "b", UpdateRequest{ClearParent: true})
	require.NoError(t, err)
	assert.Nil(t, node.ParentID)
}

func TestUpdate_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Update(context.Background(), "missing", UpdateRequest{Name: ptr("X")})
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Delete ---

func TestDelete_BlockedByChildren(t *testing.T) {
	repo := &mockRepo{
		rows:        []Category{{ID: "a", Name: "A"}},
		hasChildren: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), "a")
	assert.ErrorIs(t, err, ErrCategoryInUse)
}

func TestDelete_BlockedByQuestions(t *testing.T) {
	repo := &mockRepo{
		rows:         []Category{{ID: "a", Name: "A"}},
		hasQuestions: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), "a")
	assert.ErrorIs(t, err, ErrCategoryInUse)
}

func TestDelete_Success(t *testing.T) {
	repo := &mockRepo{rows: []Category{{ID: "a", Name: "A"}}}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), "a")
	require.NoError(t, err)
	assert.Empty(t, repo.rows)
}

func TestDelete_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	err := svc.Delete(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}
