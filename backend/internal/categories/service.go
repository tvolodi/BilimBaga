package categories

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const MaxNameLength = 128

type Service interface {
	ListTree(ctx context.Context) ([]*Node, error)
	Create(ctx context.Context, req CreateRequest) (*Node, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*Node, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListTree(ctx context.Context) ([]*Node, error) {
	rows, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("categories.ListTree: %w", err)
	}
	return buildTree(rows), nil
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*Node, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > MaxNameLength {
		return nil, ErrInvalidName
	}

	if req.ParentID != nil {
		if _, err := s.repo.GetByID(ctx, *req.ParentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrParentNotFound
			}
			return nil, fmt.Errorf("categories.Create: check parent: %w", err)
		}
	}

	created, err := s.repo.Create(ctx, name, req.ParentID, req.Track, req.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("categories.Create: %w", err)
	}
	return toNode(created), nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*Node, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > MaxNameLength {
			return nil, ErrInvalidName
		}
		current.Name = name
	}

	if req.ClearParent {
		current.ParentID = nil
	} else if req.ParentID != nil {
		if *req.ParentID == id {
			return nil, ErrCycle
		}
		if _, err := s.repo.GetByID(ctx, *req.ParentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrParentNotFound
			}
			return nil, fmt.Errorf("categories.Update: check parent: %w", err)
		}
		// Cycle detection: walk descendants of `id`; if new parent appears, reject.
		all, err := s.repo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("categories.Update: load tree: %w", err)
		}
		if isDescendant(all, id, *req.ParentID) {
			return nil, ErrCycle
		}
		current.ParentID = req.ParentID
	}

	if req.Track != nil {
		current.Track = req.Track
	}
	if req.SortOrder != nil {
		current.SortOrder = *req.SortOrder
	}

	updated, err := s.repo.Update(ctx, *current)
	if err != nil {
		return nil, fmt.Errorf("categories.Update: %w", err)
	}
	return toNode(updated), nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}

	hasChildren, err := s.repo.HasChildren(ctx, id)
	if err != nil {
		return fmt.Errorf("categories.Delete: %w", err)
	}
	if hasChildren {
		return ErrCategoryInUse
	}

	hasQuestions, err := s.repo.HasQuestions(ctx, id)
	if err != nil {
		return fmt.Errorf("categories.Delete: %w", err)
	}
	if hasQuestions {
		return ErrCategoryInUse
	}

	return s.repo.Delete(ctx, id)
}

// buildTree assembles a flat slice into a nested tree, preserving repo order.
func buildTree(rows []Category) []*Node {
	nodes := make(map[string]*Node, len(rows))
	for i := range rows {
		c := &rows[i]
		nodes[c.ID] = toNode(c)
	}

	var roots []*Node
	for i := range rows {
		c := &rows[i]
		n := nodes[c.ID]
		if c.ParentID == nil {
			roots = append(roots, n)
			continue
		}
		if parent, ok := nodes[*c.ParentID]; ok {
			parent.Children = append(parent.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	if roots == nil {
		roots = []*Node{}
	}
	return roots
}

// isDescendant returns true if `candidate` is `root` itself or transitively
// reachable from `root` through parent_id links.
func isDescendant(rows []Category, root, candidate string) bool {
	if root == candidate {
		return true
	}
	children := make(map[string][]string, len(rows))
	for _, c := range rows {
		if c.ParentID != nil {
			children[*c.ParentID] = append(children[*c.ParentID], c.ID)
		}
	}
	stack := []string{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, child := range children[n] {
			if child == candidate {
				return true
			}
			stack = append(stack, child)
		}
	}
	return false
}

func toNode(c *Category) *Node {
	return &Node{
		ID:        c.ID,
		Name:      c.Name,
		ParentID:  c.ParentID,
		Track:     c.Track,
		SortOrder: c.SortOrder,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Children:  []*Node{},
	}
}
