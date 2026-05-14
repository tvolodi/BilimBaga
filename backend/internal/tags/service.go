package tags

import (
	"context"
	"fmt"
	"strings"
)

type Service interface {
	List(ctx context.Context) ([]Tag, error)
	Create(ctx context.Context, req CreateRequest) (*Tag, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*Tag, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context) ([]Tag, error) {
	out, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("tags.List: %w", err)
	}
	if out == nil {
		out = []Tag{}
	}
	return out, nil
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*Tag, error) {
	name := NormalizeName(req.Name)
	if name == "" || len(name) > MaxTagNameLength {
		return nil, ErrInvalidName
	}
	t, err := s.repo.Create(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("tags.Create: %w", err)
	}
	return t, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*Tag, error) {
	name := NormalizeName(req.Name)
	if name == "" || len(name) > MaxTagNameLength {
		return nil, ErrInvalidName
	}
	// Confirm the tag exists first so a missing id returns ErrNotFound rather than
	// a successful no-op (UPDATE ... RETURNING returns sql.ErrNoRows in that case,
	// but being explicit keeps the failure mode unambiguous and matches Delete).
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	t, err := s.repo.Update(ctx, id, name)
	if err != nil {
		return nil, fmt.Errorf("tags.Update: %w", err)
	}
	return t, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	inUse, err := s.repo.HasReferences(ctx, id)
	if err != nil {
		return fmt.Errorf("tags.Delete: %w", err)
	}
	if inUse {
		return ErrTagInUse
	}
	return s.repo.Delete(ctx, id)
}

// NormalizeName trims whitespace and lowercases the tag name.
// Exported so it can be reused by importers in later phases.
func NormalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
