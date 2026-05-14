package users

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Service is the business-logic interface for the users domain.
type Service interface {
	ListUsers(ctx context.Context, callerRole, callerDeptID string, f ListFilters) (*ListResult, error)
	GetUser(ctx context.Context, id, callerRole, callerUserID, callerDeptID string) (*User, error)
	GetMe(ctx context.Context, userID string) (*User, error)
	CreateUser(ctx context.Context, req CreateRequest, callerRole, callerDeptID, callerUserID, ip string) (*CreateResponse, error)
	UpdateUser(ctx context.Context, id string, req UpdateRequest, callerRole, callerDeptID, callerUserID, ip string) (*User, error)
	DeactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error
	ResetPassword(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error)
	ImportUsers(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error)
}

type service struct {
	repo Repository
}

// NewService creates a new Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
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
	if callerRole == "department_admin" && callerDeptID != "" {
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

	switch callerRole {
	case "super_admin":
		// full access
	case "department_admin":
		sameUser := callerUserID == id
		sameDept := callerDeptID != "" && u.DepartmentID != nil && *u.DepartmentID == callerDeptID
		if !sameUser && !sameDept {
			return nil, ErrForbidden
		}
	default:
		if callerUserID != id {
			return nil, ErrForbidden
		}
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

// CreateUser creates a new user with a generated temporary password.
// Department admins may only create users within their own department.
func (s *service) CreateUser(ctx context.Context, req CreateRequest, callerRole, callerDeptID, callerUserID, ip string) (*CreateResponse, error) {
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
	if callerRole == "department_admin" {
		if req.DepartmentID == nil || *req.DepartmentID != callerDeptID {
			return nil, ErrForbidden
		}
	}

	tmpPwd, err := generateTempPassword()
	if err != nil {
		return nil, fmt.Errorf("users.CreateUser: generate password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPwd), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("users.CreateUser: hash password: %w", err)
	}

	u, err := s.repo.Create(ctx, strings.ToLower(req.Email), req.FullName, string(hash), req.DepartmentID, req.RoleID)
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

	if callerRole == "department_admin" {
		if existing.DepartmentID == nil || *existing.DepartmentID != callerDeptID {
			return nil, ErrForbidden
		}
	}

	if req.FullName == "" {
		return nil, fmt.Errorf("%w: full_name is required", ErrValidation)
	}
	if req.RoleID == "" {
		return nil, fmt.Errorf("%w: role_id is required", ErrValidation)
	}

	if err := s.checkRoleAssignment(ctx, req.RoleID, callerRole); err != nil {
		return nil, err
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

	if callerRole == "department_admin" {
		if existing.DepartmentID == nil || *existing.DepartmentID != callerDeptID {
			return ErrForbidden
		}
	}

	if err := s.repo.Deactivate(ctx, id); err != nil {
		return fmt.Errorf("users.DeactivateUser: %w", err)
	}
	_ = s.repo.RevokeAllTokens(ctx, id)

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

	if callerRole == "department_admin" {
		if existing.DepartmentID == nil || *existing.DepartmentID != callerDeptID {
			return nil, ErrForbidden
		}
	}

	tmpPwd, err := generateTempPassword()
	if err != nil {
		return nil, fmt.Errorf("users.ResetPassword: generate password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPwd), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("users.ResetPassword: hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return nil, fmt.Errorf("users.ResetPassword: %w", err)
	}

	return &ResetPasswordResponse{TemporaryPassword: tmpPwd}, nil
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
		errStr := s.validateImportRow(ctx, row, callerRole, callerDeptID)
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
			if err := s.commitImportRow(ctx, row, callerUserID, ip); err != nil {
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

// validateImportRow returns an error string if the row is invalid, or "" if valid.
func (s *service) validateImportRow(ctx context.Context, row CSVRow, callerRole, callerDeptID string) string {
	if row.Email == "" || !emailRegex.MatchString(row.Email) {
		return "invalid email"
	}
	if row.FullName == "" {
		return "full_name is required"
	}

	deptID, err := s.repo.GetDepartmentIDByName(ctx, row.DepartmentName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Sprintf("unknown department: %s", row.DepartmentName)
		}
		return "department lookup failed"
	}

	if callerRole == "department_admin" && deptID != callerDeptID {
		return fmt.Sprintf("department %s is outside your scope", row.DepartmentName)
	}

	_, err = s.repo.GetRoleIDByName(ctx, row.RoleName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Sprintf("unknown role: %s", row.RoleName)
		}
		return "role lookup failed"
	}

	return ""
}

// commitImportRow writes one valid import row to the database.
func (s *service) commitImportRow(ctx context.Context, row CSVRow, callerUserID, ip string) error {
	deptID, err := s.repo.GetDepartmentIDByName(ctx, row.DepartmentName)
	if err != nil {
		return fmt.Errorf("resolve department: %w", err)
	}
	roleID, err := s.repo.GetRoleIDByName(ctx, row.RoleName)
	if err != nil {
		return fmt.Errorf("resolve role: %w", err)
	}

	tmpPwd, err := generateTempPassword()
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPwd), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	u, err := s.repo.Create(ctx, strings.ToLower(row.Email), row.FullName, string(hash), &deptID, roleID)
	if err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			return fmt.Errorf("email already exists: %s", row.Email)
		}
		return fmt.Errorf("create user: %w", err)
	}

	_ = u // suppress unused warning
	return nil
}

// checkRoleAssignment returns ErrForbidden if a department_admin tries to assign the
// super_admin role. super_admin callers may assign any role.
func (s *service) checkRoleAssignment(ctx context.Context, roleID, callerRole string) error {
	if callerRole == "super_admin" {
		return nil
	}
	roleName, err := s.repo.GetRoleNameByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: role not found", ErrValidation)
		}
		return fmt.Errorf("checkRoleAssignment: %w", err)
	}
	if roleName == "super_admin" {
		return ErrForbidden
	}
	return nil
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
