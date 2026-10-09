package roles

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)
	uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

const maxDescriptionLen = 500

// nonAssignable lists permissions that must never be granted to a custom role
// (privilege-escalation guard, AC-6).
var nonAssignable = map[string]bool{
	"roles:read":    true,
	"roles:manage":  true,
	"tenant:manage": true,
}

// CacheReloader rebuilds the in-memory RBAC cache (e.g. a closure over rbacCache.Load(db)).
type CacheReloader func(ctx context.Context) error

// Service is the business-logic interface of the roles domain.
type Service interface {
	List(ctx context.Context) ([]Role, error)
	Get(ctx context.Context, id string) (*Role, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	Create(ctx context.Context, req CreateRequest) (*Role, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*UpdateResult, error)
	Delete(ctx context.Context, id string) (*Role, error)
}

type service struct {
	repo   Repository
	reload CacheReloader
}

// NewService creates a Service. reload is invoked after every successful mutation.
func NewService(repo Repository, reload CacheReloader) Service {
	return &service{repo: repo, reload: reload}
}

func (s *service) List(ctx context.Context) ([]Role, error) { return s.repo.List(ctx) }

func (s *service) Get(ctx context.Context, id string) (*Role, error) { return s.repo.GetByID(ctx, id) }

func (s *service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// validatePermissions de-duplicates ids, checks format, existence and the
// non-assignable guard; it returns the unique ids and their sorted resource:action keys.
func (s *service) validatePermissions(ctx context.Context, ids []string) ([]string, []string, error) {
	seen := make(map[string]bool, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if !uuidRe.MatchString(id) {
			return nil, nil, fmt.Errorf("%w: permission id %q is not a valid UUID", ErrValidation, id)
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	perms, err := s.repo.PermissionsByIDs(ctx, unique)
	if err != nil {
		return nil, nil, fmt.Errorf("roles.validatePermissions: %w", err)
	}
	found := make(map[string]Permission, len(perms))
	for _, p := range perms {
		found[p.ID] = p
	}
	keys := make([]string, 0, len(unique))
	for _, id := range unique {
		p, ok := found[id]
		if !ok {
			return nil, nil, fmt.Errorf("%w: unknown permission id %s", ErrValidation, id)
		}
		if nonAssignable[p.Key()] {
			return nil, nil, fmt.Errorf("%w: permission %s cannot be granted to a custom role", ErrValidation, p.Key())
		}
		keys = append(keys, p.Key())
	}
	sort.Strings(keys)
	return unique, keys, nil
}

func (s *service) reloadCache(ctx context.Context) error {
	if s.reload == nil {
		return nil
	}
	if err := s.reload(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheReload, err)
	}
	return nil
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*Role, error) {
	if !nameRe.MatchString(req.Name) {
		return nil, fmt.Errorf("%w: name must match ^[a-z][a-z0-9_]{2,31}$", ErrValidation)
	}
	if len(req.Description) > maxDescriptionLen {
		return nil, fmt.Errorf("%w: description must be at most %d characters", ErrValidation, maxDescriptionLen)
	}
	ids, _, err := s.validatePermissions(ctx, req.Permissions)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.Create(ctx, req.Name, req.Description, ids)
	if err != nil {
		return nil, err
	}
	// The row is committed; a cache failure must not report success (AC-8).
	if err := s.reloadCache(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*UpdateResult, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil && *req.Name != existing.Name {
		return nil, fmt.Errorf("%w: role name is immutable", ErrValidation)
	}
	if existing.IsSystem {
		return nil, ErrSystemImmutable
	}
	if req.Permissions == nil {
		return nil, fmt.Errorf("%w: permissions is required", ErrValidation)
	}
	if len(req.Description) > maxDescriptionLen {
		return nil, fmt.Errorf("%w: description must be at most %d characters", ErrValidation, maxDescriptionLen)
	}
	ids, keys, err := s.validatePermissions(ctx, req.Permissions)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, id, req.Description, ids); err != nil {
		return nil, err
	}
	added, removed := diff(existing.Permissions, keys)
	if err := s.reloadCache(ctx); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &UpdateResult{Role: updated, PermissionsAdded: added, PermissionsRemoved: removed}, nil
}

func (s *service) Delete(ctx context.Context, id string) (*Role, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.IsSystem {
		return nil, ErrSystemImmutable
	}
	n, err := s.repo.CountUsers(ctx, id)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, &InUseError{Count: n}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		var inUse *InUseError
		if errors.As(err, &inUse) && inUse.Count == 0 {
			// FK violation raced with a user assignment; report a fresh count.
			if c, cerr := s.repo.CountUsers(ctx, id); cerr == nil {
				inUse.Count = c
			}
		}
		return nil, err
	}
	if err := s.reloadCache(ctx); err != nil {
		return nil, err
	}
	return existing, nil
}

// diff returns the keys in next not in prev (added) and in prev not in next (removed), sorted.
func diff(prev, next []string) (added, removed []string) {
	p := make(map[string]bool, len(prev))
	for _, k := range prev {
		p[k] = true
	}
	n := make(map[string]bool, len(next))
	for _, k := range next {
		n[k] = true
		if !p[k] {
			added = append(added, k)
		}
	}
	for _, k := range prev {
		if !n[k] {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	if added == nil {
		added = []string{}
	}
	if removed == nil {
		removed = []string{}
	}
	return added, removed
}
