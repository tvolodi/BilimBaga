package tags

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	rows      []Tag
	createFn  func(ctx context.Context, name string) (*Tag, error)
	updateFn  func(ctx context.Context, id, name string) (*Tag, error)
	deleteFn  func(ctx context.Context, id string) error
	hasRefFn  func(ctx context.Context, id string) (bool, error)
	getByIDFn func(ctx context.Context, id string) (*Tag, error)
}

func (m *mockRepo) GetAll(_ context.Context) ([]Tag, error) { return m.rows, nil }

func (m *mockRepo) GetByID(ctx context.Context, id string) (*Tag, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	for i := range m.rows {
		if m.rows[i].ID == id {
			t := m.rows[i]
			return &t, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Create(ctx context.Context, name string) (*Tag, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name)
	}
	t := Tag{ID: "new", Name: name, CreatedAt: time.Now()}
	m.rows = append(m.rows, t)
	return &t, nil
}

func (m *mockRepo) Update(ctx context.Context, id, name string) (*Tag, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, name)
	}
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Name = name
			t := m.rows[i]
			return &t, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockRepo) HasReferences(ctx context.Context, id string) (bool, error) {
	if m.hasRefFn != nil {
		return m.hasRefFn(ctx, id)
	}
	return false, nil
}

func TestNormalizeName(t *testing.T) {
	assert.Equal(t, "gdpr", NormalizeName("  GDPR  "))
	assert.Equal(t, "iso27001", NormalizeName("ISO27001"))
	assert.Equal(t, "", NormalizeName("   "))
}

func TestCreate_LowercasesAndTrims(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	tag, err := svc.Create(context.Background(), CreateRequest{Name: "  GDPR  "})
	require.NoError(t, err)
	assert.Equal(t, "gdpr", tag.Name)
}

func TestCreate_RejectsEmpty(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), CreateRequest{Name: "    "})
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestCreate_RejectsTooLong(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), CreateRequest{Name: strings.Repeat("a", MaxTagNameLength+1)})
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestCreate_DuplicateBubblesUp(t *testing.T) {
	repo := &mockRepo{createFn: func(_ context.Context, _ string) (*Tag, error) { return nil, ErrDuplicate }}
	svc := NewService(repo)
	_, err := svc.Create(context.Background(), CreateRequest{Name: "x"})
	assert.True(t, errors.Is(err, ErrDuplicate))
}

func TestDelete_BlockedWhenInUse(t *testing.T) {
	repo := &mockRepo{
		rows:     []Tag{{ID: "t1", Name: "gdpr"}},
		hasRefFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), "t1")
	assert.ErrorIs(t, err, ErrTagInUse)
}

func TestDelete_Success(t *testing.T) {
	repo := &mockRepo{rows: []Tag{{ID: "t1", Name: "gdpr"}}}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), "t1")
	require.NoError(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	err := svc.Delete(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdate_LowercasesAndTrims(t *testing.T) {
	repo := &mockRepo{rows: []Tag{{ID: "t1", Name: "old"}}}
	svc := NewService(repo)
	tag, err := svc.Update(context.Background(), "t1", UpdateRequest{Name: "  GDPR  "})
	require.NoError(t, err)
	assert.Equal(t, "gdpr", tag.Name)
}

func TestUpdate_RejectsEmpty(t *testing.T) {
	repo := &mockRepo{rows: []Tag{{ID: "t1", Name: "old"}}}
	svc := NewService(repo)
	_, err := svc.Update(context.Background(), "t1", UpdateRequest{Name: "   "})
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestUpdate_RejectsTooLong(t *testing.T) {
	repo := &mockRepo{rows: []Tag{{ID: "t1", Name: "old"}}}
	svc := NewService(repo)
	_, err := svc.Update(context.Background(), "t1", UpdateRequest{Name: strings.Repeat("a", MaxTagNameLength+1)})
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestUpdate_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Update(context.Background(), "missing", UpdateRequest{Name: "x"})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdate_DuplicateBubblesUp(t *testing.T) {
	repo := &mockRepo{
		rows:     []Tag{{ID: "t1", Name: "old"}},
		updateFn: func(_ context.Context, _, _ string) (*Tag, error) { return nil, ErrDuplicate },
	}
	svc := NewService(repo)
	_, err := svc.Update(context.Background(), "t1", UpdateRequest{Name: "taken"})
	assert.True(t, errors.Is(err, ErrDuplicate))
}

func TestList_EmptyReturnsEmptySlice(t *testing.T) {
	svc := NewService(&mockRepo{})
	out, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.Empty(t, out)
}
