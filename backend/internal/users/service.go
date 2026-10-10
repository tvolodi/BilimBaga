package users

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/email"
)

const bcryptCost = 12

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Service is the business-logic interface for the users domain.
type Service interface {
	ListUsers(ctx context.Context, callerRole, callerDeptID string, f ListFilters) (*ListResult, error)
	GetUser(ctx context.Context, id, callerRole, callerUserID, callerDeptID string) (*User, error)
	GetMe(ctx context.Context, userID string) (*User, error)
	UpdateMyLocale(ctx context.Context, userID string, locale *string) (*LocaleUpdate, error)
	CreateUser(ctx context.Context, req CreateRequest, callerRole, callerDeptID, callerUserID, ip string) (*CreateResponse, error)
	UpdateUser(ctx context.Context, id string, req UpdateRequest, callerRole, callerDeptID, callerUserID, ip string) (*User, error)
	DeactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error
	ReactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error
	ResetPassword(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error)
	UnlockUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*User, error)
	ImportUsers(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error)
	ListRoles(ctx context.Context, callerRole string) ([]RoleRow, error)
	// RemindEmployee sends a manual overdue-exam reminder (FR-BB510).
	RemindEmployee(ctx context.Context, actorID, userID, examID string) (*RemindResult, error)
}

type service struct {
	repo     Repository
	emailSvc *email.EmailService
	reminder Reminder
	now      func() time.Time
	// canPerm reports whether a role holds resource:action. nil means "no
	// permission information": GetUser then grants only self-access to
	// non-super_admin roles (default-deny).
	canPerm  PermissionChecker
	permsFor func(role string) []string
	// locales supplies the tenant available_locales used to validate preferred_locale (FR-BB116).
	// nil means "unknown": every non-null locale is then refused (fail-closed).
	locales LocaleSource
}

// LocaleSource reports the tenant's configured available_locales (satisfied by tenant.Service).
type LocaleSource interface {
	GetAvailableLocales() []string
}

// WithLocaleSource attaches the tenant locale source used by UpdateMyLocale.
func WithLocaleSource(svc Service, src LocaleSource) Service {
	if s, ok := svc.(*service); ok {
		s.locales = src
	}
	return svc
}

// PermissionChecker reports whether role holds the resource:action permission
// (satisfied by (*rbac.Cache).Has).
type PermissionChecker func(role, resource, action string) bool

// WithPermissionsLookup attaches a role -> "resource:action" lookup used to stop
// custom-role callers from assigning roles more powerful than their own.
func WithPermissionsLookup(svc Service, fn func(role string) []string) Service {
	if s, ok := svc.(*service); ok {
		s.permsFor = fn
	}
	return svc
}

// WithPermissionChecker attaches a PermissionChecker to a Service built by NewService.
func WithPermissionChecker(svc Service, fn PermissionChecker) Service {
	if s, ok := svc.(*service); ok {
		s.canPerm = fn
	}
	return svc
}

// Scoping model (FR-BB117, default-deny): only super_admin is organisation-wide.
// EVERY other role -- built-in department_admin or an admin-created custom role --
// is confined to its own department, and a caller without a department sees and
// may touch nothing. Scoping must never branch on a built-in role name other than
// super_admin, otherwise a custom role would fall through to org-wide access.
func isOrgWide(callerRole string) bool { return callerRole == "super_admin" }

// inCallerScope reports whether a target department is within the caller's scope.
func inCallerScope(callerRole, callerDeptID string, target *string) bool {
	if isOrgWide(callerRole) {
		return true
	}
	return callerDeptID != "" && target != nil && *target == callerDeptID
}

// NewService creates a new Service backed by the given Repository.
// An optional EmailService may be passed as the second argument to enable
// transactional email delivery; pass nil or omit to disable.
func NewService(repo Repository, emailSvc ...*email.EmailService) Service {
	s := &service{repo: repo}
	if len(emailSvc) > 0 && emailSvc[0] != nil {
		s.emailSvc = emailSvc[0]
		s.reminder = emailReminder{svc: emailSvc[0]}
	}
	return s
}

// ListUsers returns a paginated, filtered user list. Department admins are automatically
// scoped to their own department; super admins see all users.
func (s *service) ListUsers(ctx context.Context, callerRole, callerDeptID string, f ListFilters) (*ListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}

	var deptScope *string
	if !isOrgWide(callerRole) {
		if callerDeptID == "" {
			// Non-super_admin without a department has no scope: empty result.
			return &ListResult{Items: []User{}, Meta: Meta{Page: f.Page, PerPage: f.PerPage, Total: 0}}, nil
		}
		deptScope = &callerDeptID
	}

	users, total, err := s.repo.List(ctx, f, deptScope)
	if err != nil {
		return nil, fmt.Errorf("users.ListUsers: %w", err)
	}
	return &ListResult{
		Items: users,
		Meta:  Meta{Page: f.Page, PerPage: f.PerPage, Total: total},
	}, nil
}

// GetUser fetches a single user, enforcing the following access rules:
//   - super_admin: any user
//   - department_admin: users in their own department, or themselves
//   - any other authenticated user: themselves only
func (s *service) GetUser(ctx context.Context, id, callerRole, callerUserID, callerDeptID string) (*User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch {
	case isOrgWide(callerRole):
		// full access
	case callerUserID == id:
		// self-access
	case callerRole == "department_admin" || (s.canPerm != nil && s.canPerm(callerRole, "users", "read")):
		// department-scoped read for roles holding users:read
		if !inCallerScope(callerRole, callerDeptID, u.DepartmentID) {
			return nil, ErrNotFound
		}
	default:
		return nil, ErrForbidden
	}
	return u, nil
}

// GetMe returns the calling user's own profile.
func (s *service) GetMe(ctx context.Context, userID string) (*User, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("users.GetMe: %w", err)
	}
	return u, nil
}

// UpdateMyLocale sets (or, with a nil locale, clears) the caller's own preferred_locale
// (FR-BB116 AC-2). A non-nil locale must be one of the tenant's available_locales; anything
// else is ErrValidation. The caller can only ever address its own record, so no role check
// applies. The returned LocaleUpdate carries the previous value for the audit entry.
func (s *service) UpdateMyLocale(ctx context.Context, userID string, locale *string) (*LocaleUpdate, error) {
	current, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("users.UpdateMyLocale: %w", err)
	}
	previous := current.PreferredLocale
	if locale != nil && !s.isAvailableLocale(*locale) {
		return nil, fmt.Errorf("%w: preferred_locale must be one of the tenant available locales", ErrValidation)
	}
	if err := s.repo.SetPreferredLocale(ctx, userID, locale); err != nil {
		return nil, fmt.Errorf("users.UpdateMyLocale: %w", err)
	}
	updated, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("users.UpdateMyLocale: reload: %w", err)
	}
	return &LocaleUpdate{User: updated, Previous: previous}, nil
}

// isAvailableLocale reports whether code is one of the tenant's available_locales.
func (s *service) isAvailableLocale(code string) bool {
	if s.locales == nil {
		return false
	}
	for _, l := range s.locales.GetAvailableLocales() {
		if l == code {
			return true
		}
	}
	return false
}

// CreateUser creates a new user with a generated temporary password.
// Department admins may only create users within their own department.
func (s *service) CreateUser(ctx context.Context, req CreateRequest, callerRole, callerDeptID, callerUserID, ip string) (*CreateResponse, error) {
	req.Email = api.NormalizeEmail(req.Email)
	if req.Email == "" || !emailRegex.MatchString(req.Email) {
		return nil, fmt.Errorf("%w: email is required and must be a valid email address", ErrValidation)
	}
	if req.FullName == "" {
		return nil, fmt.Errorf("%w: full_name is required", ErrValidation)
	}
	if req.RoleID == "" {
		return nil, fmt.Errorf("%w: role_id is required", ErrValidation)
	}

	// Validate role assignment rules.
	if err := s.checkRoleAssignment(ctx, req.RoleID, callerRole); err != nil {
		return nil, err
	}

	// Department admin scope check.
	if !inCallerScope(callerRole, callerDeptID, req.DepartmentID) {
		return nil, ErrForbidden
	}

	tmpPwd, err := generateTempPassword()
	if err != nil {
		return nil, fmt.Errorf("users.CreateUser: generate password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPwd), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("users.CreateUser: hash password: %w", err)
	}

	u, err := s.repo.Create(ctx, req.Email, req.FullName, string(hash), req.DepartmentID, req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("users.CreateUser: %w", err)
	}

	return &CreateResponse{User: *u, TemporaryPassword: tmpPwd}, nil
}

// UpdateUser updates full_name, department_id and role_id for an existing user.
// Department admins may only update users in their own department and may not assign
// the super_admin role.
func (s *service) UpdateUser(ctx context.Context, id string, req UpdateRequest, callerRole, callerDeptID, callerUserID, ip string) (*User, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Out-of-scope targets look exactly like unknown ids (404): no existence probing.
	if !inCallerScope(callerRole, callerDeptID, existing.DepartmentID) {
		return nil, ErrNotFound
	}
	// FR-BB117 D-1: authorise against the target's role before any write or secret.
	if err := s.checkTargetActionable(existing, callerRole, callerUserID); err != nil {
		return nil, err
	}
	// Nobody, super_admin included, may change their own role (FR-BB117 D-4).
	if id == callerUserID && !sameID(req.RoleID, existing.RoleID) {
		return nil, ErrForbidden
	}
	// A scoped caller may not move a user out of its own department.
	if !isOrgWide(callerRole) && !inCallerScope(callerRole, callerDeptID, req.DepartmentID) {
		return nil, ErrForbidden
	}

	if req.FullName == "" {
		return nil, fmt.Errorf("%w: full_name is required", ErrValidation)
	}
	if req.RoleID == "" {
		return nil, fmt.Errorf("%w: role_id is required", ErrValidation)
	}

	// An unchanged role needs no assignment check: the target check above already covers it.
	if !sameID(req.RoleID, existing.RoleID) {
		if err := s.checkRoleAssignment(ctx, req.RoleID, callerRole); err != nil {
			return nil, err
		}
	}

	u, err := s.repo.Update(ctx, id, req.FullName, req.DepartmentID, req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("users.UpdateUser: %w", err)
	}

	return u, nil
}

// DeactivateUser sets a user's status to inactive and revokes all their refresh tokens.
// Department admins may only deactivate users in their own department.
func (s *service) DeactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !inCallerScope(callerRole, callerDeptID, existing.DepartmentID) {
		return ErrNotFound
	}
	// Nobody, super_admin included, may deactivate themselves (FR-BB117 D-4).
	if id == callerUserID {
		return ErrForbidden
	}
	// FR-BB117 D-1: authorise against the target's role before any write or secret.
	if err := s.checkTargetActionable(existing, callerRole, callerUserID); err != nil {
		return err
	}

	if err := s.repo.Deactivate(ctx, id); err != nil {
		return fmt.Errorf("users.DeactivateUser: %w", err)
	}
	_ = s.repo.RevokeAllTokens(ctx, id)

	return nil
}

// ReactivateUser sets a deactivated user back to active (FR-BB18 AC-13). It is authorised
// exactly like DeactivateUser, and the decision runs before any write. It does not reset the
// password, the lockout state or the revoked refresh tokens: the user signs in again with
// the previous password. Reactivating an active user is ErrUserAlreadyActive, with no change.
func (s *service) ReactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !inCallerScope(callerRole, callerDeptID, existing.DepartmentID) {
		return ErrNotFound
	}
	// Nobody, super_admin included, may reactivate themselves (same rule as deactivate, FR-BB117 D-4).
	if id == callerUserID {
		return ErrForbidden
	}
	// FR-BB117 D-1: the target's rank is checked before any status change.
	if err := s.checkTargetActionable(existing, callerRole, callerUserID); err != nil {
		return err
	}
	if existing.Status == "active" {
		return ErrUserAlreadyActive
	}

	if err := s.repo.Reactivate(ctx, id); err != nil {
		return fmt.Errorf("users.ReactivateUser: %w", err)
	}
	return nil
}

// ResetPassword generates a new temporary password, hashes it, stores it, and returns
// the plaintext — which is the only time it is ever visible. Department admins may only
// reset passwords for users in their own department.
func (s *service) ResetPassword(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !inCallerScope(callerRole, callerDeptID, existing.DepartmentID) {
		return nil, ErrNotFound
	}
	// FR-BB117 D-1: authorise against the target's role before any write or secret.
	if err := s.checkTargetActionable(existing, callerRole, callerUserID); err != nil {
		return nil, err
	}

	tmpPwd, err := generateTempPassword()
	if err != nil {
		return nil, fmt.Errorf("users.ResetPassword: generate password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPwd), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("users.ResetPassword: hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, id, string(hash), s.clock()); err != nil {
		return nil, fmt.Errorf("users.ResetPassword: %w", err)
	}

	if s.emailSvc != nil {
		s.emailSvc.TriggerPasswordReset(id, tmpPwd)
	}

	return &ResetPasswordResponse{TemporaryPassword: tmpPwd}, nil
}

// UnlockUser clears the lockout of a user early and returns the refreshed record (FR-BB115).
// Department admins may only unlock users in their own department.
func (s *service) UnlockUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*User, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !inCallerScope(callerRole, callerDeptID, existing.DepartmentID) {
		return nil, ErrNotFound
	}
	// FR-BB117 D-1: authorise against the target's role before any write or secret.
	if err := s.checkTargetActionable(existing, callerRole, callerUserID); err != nil {
		return nil, err
	}
	if err := s.repo.Unlock(ctx, id); err != nil {
		return nil, fmt.Errorf("users.UnlockUser: %w", err)
	}
	return s.repo.GetByID(ctx, id)
}

// ImportUsers validates, and optionally commits, rows from a CSV bulk import.
// Without commit=true it returns a preview of valid and invalid rows with no DB writes.
// With commit=true all valid rows are created; invalid rows are listed in the response.
func (s *service) ImportUsers(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error) {
	preview := &ImportPreview{
		Valid:  []ImportRowResult{},
		Errors: []ImportRowResult{},
	}

	for _, row := range rows {
		row.Email = api.NormalizeEmail(row.Email)
		deptID, roleID, errStr := s.validateImportRow(ctx, row, callerRole, callerDeptID)
		if errStr != "" {
			errCopy := errStr
			preview.Errors = append(preview.Errors, ImportRowResult{
				RowNum:         row.RowNum,
				Email:          row.Email,
				FullName:       row.FullName,
				DepartmentName: row.DepartmentName,
				RoleName:       row.RoleName,
				Error:          &errCopy,
			})
			continue
		}

		if commit {
			if err := s.commitImportRow(ctx, row, deptID, roleID); err != nil {
				msg := err.Error()
				preview.Errors = append(preview.Errors, ImportRowResult{
					RowNum:         row.RowNum,
					Email:          row.Email,
					FullName:       row.FullName,
					DepartmentName: row.DepartmentName,
					RoleName:       row.RoleName,
					Error:          &msg,
				})
				continue
			}
		}

		preview.Valid = append(preview.Valid, ImportRowResult{
			RowNum:         row.RowNum,
			Email:          row.Email,
			FullName:       row.FullName,
			DepartmentName: row.DepartmentName,
			RoleName:       row.RoleName,
		})
	}

	return preview, nil
}

// validateImportRow resolves the row's department and role once and returns their ids with
// an empty error string, or an error string if the row is invalid. The ids are reused for the
// commit so the validated department is exactly the one written.
func (s *service) validateImportRow(ctx context.Context, row CSVRow, callerRole, callerDeptID string) (deptID, roleID, errStr string) {
	if row.Email == "" || !emailRegex.MatchString(row.Email) {
		return "", "", "invalid email"
	}
	if row.FullName == "" {
		return "", "", "full_name is required"
	}

	deptID, err := s.repo.GetDepartmentIDByName(ctx, row.DepartmentName)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return "", "", fmt.Sprintf("unknown department: %s", row.DepartmentName)
		case errors.Is(err, ErrAmbiguousName):
			return "", "", fmt.Sprintf("ambiguous department name: %s matches more than one department", row.DepartmentName)
		}
		return "", "", "department lookup failed"
	}

	if !inCallerScope(callerRole, callerDeptID, &deptID) {
		return "", "", fmt.Sprintf("department %s is outside your scope", row.DepartmentName)
	}

	roleID, err = s.repo.GetRoleIDByName(ctx, row.RoleName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", "", fmt.Sprintf("unknown role: %s", row.RoleName)
		}
		return "", "", "role lookup failed"
	}
	if err := s.checkRoleAssignment(ctx, roleID, callerRole); err != nil {
		if errors.Is(err, ErrForbidden) {
			return "", "", fmt.Sprintf("role %s cannot be assigned by your role", row.RoleName)
		}
		return "", "", "role check failed"
	}

	return deptID, roleID, ""
}

// commitImportRow writes one valid import row to the database using the ids resolved by
// validateImportRow.
func (s *service) commitImportRow(ctx context.Context, row CSVRow, deptID, roleID string) error {
	tmpPwd, err := generateTempPassword()
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPwd), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if _, err := s.repo.Create(ctx, row.Email, row.FullName, string(hash), &deptID, roleID); err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			return fmt.Errorf("email already exists: %s", row.Email)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// validateIDs rejects a malformed role_id or department_id (ErrValidation, mapped to 422)
// before it reaches Postgres, where the uuid cast would fail with a 500. Called by the
// repository write methods, the layer that owns the uuid columns.
func validateIDs(roleID string, departmentID *string) error {
	if roleID != "" && !api.IsUUID(roleID) {
		return fmt.Errorf("%w: role_id must be a valid UUID", ErrValidation)
	}
	if departmentID != nil && !api.IsUUID(*departmentID) {
		return fmt.Errorf("%w: department_id must be a valid UUID", ErrValidation)
	}
	return nil
}

// sameID compares two UUID strings case-insensitively.
func sameID(a, b string) bool { return strings.EqualFold(a, b) }

// builtinRank is the strict rank hierarchy of the built-in roles (FR-BB117 D-1, Supervisor
// decision): super_admin > department_admin > examiner > employee. A built-in caller may
// act on, or assign, a built-in role only if that role's rank is STRICTLY LOWER than its own.
// A role name absent from this table is a custom role: it has no rank, is treated as ranking
// below department_admin, and is governed by the permission-subset rule instead. Do not add
// ad-hoc role-name comparisons elsewhere; extend this table.
var builtinRank = map[string]int{
	"employee":         1,
	"examiner":         2,
	"department_admin": 3,
	"super_admin":      4,
}

// sensitiveCustomPerms may never be conferred by a non-super_admin caller through role assignment.
var sensitiveCustomPerms = map[string]bool{"roles:read": true, "roles:manage": true, "tenant:manage": true}

// checkRoleAssignment returns ErrForbidden unless the caller may assign roleID (FR-BB117 D-1).
// super_admin may assign any role; everyone else follows canReachRole plus, for custom roles,
// a ban on roles:read / roles:manage / tenant:manage.
func (s *service) checkRoleAssignment(ctx context.Context, roleID, callerRole string) error {
	if isOrgWide(callerRole) {
		return nil
	}
	roleName, err := s.repo.GetRoleNameByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: role not found", ErrValidation)
		}
		return fmt.Errorf("checkRoleAssignment: %w", err)
	}
	return s.roleAssignableBy(roleName, callerRole)
}

// roleAssignableBy is the name-based assignment decision shared by checkRoleAssignment
// (create/update validation) and ListRoles (the per-role `assignable` flag), so the two can
// never drift. It applies to a non-super_admin caller: canReachRole plus, for custom roles,
// a ban on roles:read / roles:manage / tenant:manage.
func (s *service) roleAssignableBy(roleName, callerRole string) error {
	if err := s.canReachRole(roleName, callerRole); err != nil {
		return err
	}
	if _, builtin := builtinRank[roleName]; !builtin && s.permsFor != nil {
		for _, p := range s.permsFor(roleName) {
			if sensitiveCustomPerms[p] {
				return ErrForbidden
			}
		}
	}
	return nil
}

// checkTargetActionable enforces FR-BB117 D-1: a non-super_admin caller may reset the
// password of, edit, deactivate or unlock a user only if canReachRole allows the target's
// role. Self-service is preserved: a caller acting on its own record while the stored role
// matches its token role is allowed (own role change is blocked separately in UpdateUser).
// A target with an unknown/legacy (empty) role is refused (fail-closed). It must run
// before any password generation or repository write.
func (s *service) checkTargetActionable(target *User, callerRole, callerUserID string) error {
	if isOrgWide(callerRole) {
		return nil
	}
	if target == nil || target.RoleName == "" || callerRole == "" {
		return ErrForbidden
	}
	if callerUserID != "" && target.ID == callerUserID && target.RoleName == callerRole {
		return nil
	}
	return s.canReachRole(target.RoleName, callerRole)
}

// canReachRole is the single D-1 decision for a non-super_admin caller and a role (as a
// target's current role or a role being assigned).
//   - built-in caller, built-in role: allowed iff rank(role) < rank(caller).
//   - built-in caller, custom role: allowed iff every permission of the role is held by the caller.
//   - custom caller (ranks below department_admin): a built-in role of rank >= department_admin
//     is refused; every other role must be a permission subset of the caller's.
//
// Anything unresolvable (empty names, missing permission lookup, caller custom role holding no
// permissions) is refused.
func (s *service) canReachRole(roleName, callerRole string) error {
	if roleName == "" || callerRole == "" || roleName == "super_admin" {
		return ErrForbidden
	}
	callerRank, callerBuiltin := builtinRank[callerRole]
	roleRank, roleBuiltin := builtinRank[roleName]
	if callerBuiltin {
		if roleBuiltin {
			if roleRank < callerRank {
				return nil
			}
			return ErrForbidden
		}
		return s.permissionSubset(roleName, callerRole)
	}
	if roleBuiltin && roleRank >= builtinRank["department_admin"] {
		return ErrForbidden
	}
	if s.permsFor == nil || len(s.permsFor(callerRole)) == 0 {
		return ErrForbidden
	}
	return s.permissionSubset(roleName, callerRole)
}

// permissionSubset reports nil iff the permissions of roleName are a STRICT subset of those of
// callerRole (FR-BB117 D-4): a custom role with exactly the caller's permissions is a peer and
// is refused, so a caller can never mint or manage an equal-power role.
func (s *service) permissionSubset(roleName, callerRole string) error {
	if s.permsFor == nil || s.canPerm == nil {
		return ErrForbidden
	}
	rolePerms := s.permsFor(roleName)
	if len(rolePerms) == 0 {
		// Unknown/stale role (absent from the permission cache) or a role without
		// permissions: not provably a subset, so fail closed.
		return ErrForbidden
	}
	roleSet := make(map[string]bool, len(rolePerms))
	for _, p := range rolePerms {
		res, act, _ := strings.Cut(p, ":")
		if !s.canPerm(callerRole, res, act) {
			return ErrForbidden
		}
		roleSet[p] = true
	}
	// Strictness: the caller must hold at least one permission the role lacks.
	for _, p := range s.permsFor(callerRole) {
		if !roleSet[p] {
			return nil
		}
	}
	return ErrForbidden
}

// ListRoles returns all roles from the database, each annotated with whether the CURRENT
// caller may assign it (FR-BB117 D-1). super_admin gets true for every role; everyone else
// is evaluated by roleAssignableBy, the same rule checkRoleAssignment enforces on create and
// update. Only the boolean is exposed, never another role's permission list.
func (s *service) ListRoles(ctx context.Context, callerRole string) ([]RoleRow, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range roles {
		if isOrgWide(callerRole) {
			roles[i].Assignable = true
			continue
		}
		roles[i].Assignable = s.roleAssignableBy(roles[i].Name, callerRole) == nil
	}
	return roles, nil
}

// generateTempPassword generates a cryptographically secure 10-character password
// containing lowercase letters, uppercase letters, digits, and at least one special char.
func generateTempPassword() (string, error) {
	const (
		lowers   = "abcdefghijklmnopqrstuvwxyz"
		uppers   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits   = "0123456789"
		specials = "!@#$%^&*"
	)
	// Guarantee at least one character from each group.
	groups := []string{lowers, lowers, lowers, uppers, uppers, uppers, digits, digits, specials, specials}
	b := make([]byte, len(groups))
	for i, g := range groups {
		idx, err := randomIndex(len(g))
		if err != nil {
			return "", err
		}
		b[i] = g[idx]
	}
	// Fisher-Yates shuffle using crypto/rand.
	for i := len(b) - 1; i > 0; i-- {
		j, err := randomIndex(i + 1)
		if err != nil {
			return "", err
		}
		b[i], b[j] = b[j], b[i]
	}
	return string(b), nil
}

func randomIndex(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

// ErrValidation is a sentinel used to signal request validation failures.
var ErrValidation = errors.New("validation error")
