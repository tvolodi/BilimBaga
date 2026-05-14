package departments

import (
	"context"
	"errors"
	"fmt"
)

// Service is the business-logic interface for the departments domain.
type Service interface {
	ListTree(ctx context.Context) ([]*DepartmentNode, error)
	Create(ctx context.Context, req CreateRequest, userID, ipAddress string) (*DepartmentNode, error)
	Update(ctx context.Context, id string, req UpdateRequest, userID, ipAddress string) (*DepartmentNode, error)
	Delete(ctx context.Context, id, userID, ipAddress string) error
}

type service struct {
	repo Repository
}

// NewService creates a new Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// ListTree fetches all departments and assembles them into a nested tree.
func (s *service) ListTree(ctx context.Context) ([]*DepartmentNode, error) {
	depts, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("departments.ListTree: %w", err)
	}
	return buildTree(depts), nil
}

// buildTree assembles a flat slice of Department records into a nested tree.
// The input slice is assumed to be ordered by name so that sibling ordering is alphabetical.
func buildTree(depts []Department) []*DepartmentNode {
	nodes := make(map[string]*DepartmentNode, len(depts))

	// First pass: create all nodes.
	for i := range depts {
		d := &depts[i]
		nodes[d.ID] = &DepartmentNode{
			ID:        d.ID,
			Name:      d.Name,
			ParentID:  d.ParentID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			Children:  []*DepartmentNode{},
		}
	}

	// Second pass: wire children to parents, collect roots in original slice order.
	var roots []*DepartmentNode
	for i := range depts {
		d := &depts[i]
		node := nodes[d.ID]
		if d.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodes[*d.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Orphaned node (parent not loaded): surface as a root.
			roots = append(roots, node)
		}
	}

	if roots == nil {
		roots = []*DepartmentNode{}
	}
	return roots
}

// Create creates a new department with an optional parent.
func (s *service) Create(ctx context.Context, req CreateRequest, userID, ipAddress string) (*DepartmentNode, error) {
	if req.ParentID != nil {
		if _, err := s.repo.GetByID(ctx, *req.ParentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("departments.Create: check parent: %w", err)
		}
	}

	created, err := s.repo.Create(ctx, req.Name, req.ParentID)
	if err != nil {
		return nil, fmt.Errorf("departments.Create: %w", err)
	}

	return &DepartmentNode{
		ID:        created.ID,
		Name:      created.Name,
		ParentID:  created.ParentID,
		CreatedAt: created.CreatedAt,
		UpdatedAt: created.UpdatedAt,
		Children:  []*DepartmentNode{},
	}, nil
}

// Update renames a department.
func (s *service) Update(ctx context.Context, id string, req UpdateRequest, userID, ipAddress string) (*DepartmentNode, error) {
	updated, err := s.repo.Update(ctx, id, req.Name)
	if err != nil {
		return nil, fmt.Errorf("departments.Update: %w", err)
	}

	return &DepartmentNode{
		ID:        updated.ID,
		Name:      updated.Name,
		ParentID:  updated.ParentID,
		CreatedAt: updated.CreatedAt,
		UpdatedAt: updated.UpdatedAt,
		Children:  []*DepartmentNode{},
	}, nil
}

// Delete removes a department after verifying it has no children and no users.
func (s *service) Delete(ctx context.Context, id, userID, ipAddress string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("departments.Delete: %w", err)
	}

	hasChildren, err := s.repo.HasChildren(ctx, id)
	if err != nil {
		return fmt.Errorf("departments.Delete: check children: %w", err)
	}
	if hasChildren {
		return ErrDepartmentHasChildren
	}

	hasUsers, err := s.repo.HasUsers(ctx, id)
	if err != nil {
		return fmt.Errorf("departments.Delete: check users: %w", err)
	}
	if hasUsers {
		return ErrDepartmentNotEmpty
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("departments.Delete: %w", err)
	}

	return nil
}
