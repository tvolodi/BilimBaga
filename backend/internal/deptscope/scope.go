// Package deptscope implements department-subtree scoping for admin read APIs
// (ISS-165, ISS-218). Every role except super_admin (department_admin,
// examiner, every custom role, and an empty/unknown role) may only see data
// about employees of its own department and that department's descendants; a
// caller with no department sees nothing (FR-BB117 D-2). Scoping never depends
// on a role NAME other than the single unrestricted one, so a custom role with
// reports:read or grading:* cannot read org-wide data.
//
// The scope is derived from the authenticated principal carried in the request
// context (role + department id set by auth.Authenticate), so a handler,
// service or repository cannot forget to forward it.
package deptscope

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// RoleSuperAdmin is the only role exempt from department scoping.
const RoleSuperAdmin = "super_admin"

// RoleDepartmentAdmin is the built-in department administrator role.
const RoleDepartmentAdmin = "department_admin"

// RoleExaminer is the built-in examiner role. It is department-scoped like any
// other non-super_admin role, with one narrow exception: the manual-grading
// flow for exams the examiner created (see Scope.ExamOwnerID).
const RoleExaminer = "examiner"

// noDepartment is a UUID that matches no department. A restricted caller with
// no department of its own gets this value so its scope is the empty set.
const noDepartment = "00000000-0000-0000-0000-000000000000"

// Scope describes which employees a caller may see.
type Scope struct {
	// Restricted is true when the caller is limited to a department subtree.
	Restricted bool
	// DepartmentID is the root of the subtree (empty when the caller has none).
	DepartmentID string
	// UserID is the caller's own user id; a caller may always see itself.
	UserID string
	// ExamOwnerID is non-empty only for the built-in examiner role: the
	// examiner's own user id. Manual-grading endpoints (queue, detail, grade)
	// additionally accept sessions of exams whose created_by equals this id, so
	// an examiner can grade the exams it owns even when the candidates are
	// outside its department. It never widens reports, analytics, AI insights
	// or user records.
	ExamOwnerID string
}

// FromContext derives the scope from the authenticated principal in ctx.
// Only super_admin is unrestricted; any other role, including an empty or
// unknown one (no principal in ctx), fails closed to the caller's department
// subtree.
func FromContext(ctx context.Context) Scope {
	role := ctxkeys.RoleFromCtx(ctx)
	if role == RoleSuperAdmin {
		return Scope{}
	}
	s := Scope{
		Restricted:   true,
		DepartmentID: ctxkeys.DepartmentIDFromCtx(ctx),
		UserID:       ctxkeys.UserIDFromCtx(ctx),
	}
	if role == RoleExaminer {
		s.ExamOwnerID = s.UserID
	}
	return s
}

// Arg returns the SQL parameter for Predicate: nil (SQL NULL, unrestricted) or
// the subtree root department id.
func (s Scope) Arg() any {
	if !s.Restricted {
		return nil
	}
	if s.DepartmentID == "" {
		return noDepartment
	}
	return s.DepartmentID
}

// OwnerArg returns the SQL parameter for GradingPredicate's owner: nil (SQL
// NULL, matches no exam) unless the caller is an examiner with a known user id.
func (s Scope) OwnerArg() any {
	if !s.Restricted || s.ExamOwnerID == "" {
		return nil
	}
	return s.ExamOwnerID
}

// Predicate returns a SQL boolean expression that is true when the user id in
// userCol belongs to the department subtree identified by the uuid parameter
// param (for example "$2"), or when param is NULL (unrestricted caller).
// Aliases inside the expression are prefixed sc_ so they cannot clash with the
// surrounding query.
func Predicate(userCol, param string) string {
	return fmt.Sprintf(`(%[2]s::uuid IS NULL OR %[1]s IN (`+
		`SELECT sc_u.id FROM users sc_u WHERE sc_u.department_id IN (`+
		`WITH RECURSIVE sc_t(id) AS (`+
		`SELECT sc_d.id FROM departments sc_d WHERE sc_d.id = %[2]s::uuid `+
		`UNION ALL `+
		`SELECT sc_c.id FROM departments sc_c JOIN sc_t ON sc_c.parent_id = sc_t.id`+
		`) SELECT id FROM sc_t)))`, userCol, param)
}

// GradingPredicate is Predicate widened for the manual-grading flow only: true
// when the session's employee (userCol) is inside the subtree bound to param,
// OR the exam (examCol is the exams alias/column holding created_by) was
// created by the examiner bound to ownerParam. A NULL ownerParam never matches
// an exam, so only examiners with a known user id get the carve-out.
func GradingPredicate(userCol, param, examCol, ownerParam string) string {
	return fmt.Sprintf(`(%s.created_by = %s::uuid OR %s)`, examCol, ownerParam, Predicate(userCol, param))
}

// Scope keys for cache partitioning (ISS-218 supervisor decision). A key is a
// pure function of the set of department ids a caller may see, so callers with
// equal scopes share cache entries and callers with different scopes never do.
const (
	// ScopeKeyAll is the key of the unrestricted (super_admin) scope.
	ScopeKeyAll = "all"
	// ScopeKeyNone is the key of an empty department set (a restricted caller
	// with no department, or one whose department does not exist).
	ScopeKeyNone = "none"
)

// SubtreeSQL selects the ids of the department subtree rooted at $1 (a uuid).
const SubtreeSQL = `WITH RECURSIVE sc_t(id) AS (` +
	`SELECT sc_d.id FROM departments sc_d WHERE sc_d.id = $1::uuid ` +
	`UNION ` +
	`SELECT sc_c.id FROM departments sc_c JOIN sc_t ON sc_c.parent_id = sc_t.id` +
	`) SELECT id::text FROM sc_t ORDER BY id`

// SubtreeArg returns the $1 argument for SubtreeSQL: the subtree root, or the
// no-department sentinel (matches nothing) when the caller has none. It must
// only be used for a Restricted scope.
func (s Scope) SubtreeArg() string {
	if s.DepartmentID == "" {
		return noDepartment
	}
	return s.DepartmentID
}

// ScopeKey returns a stable cache-partition key for the department ids a
// caller may see. An unrestricted scope is always ScopeKeyAll regardless of
// ids; a restricted scope with no ids is ScopeKeyNone; otherwise it is the
// hex SHA-256 of the sorted, de-duplicated ids joined with ",". Hex digests
// are 64 characters, so they can never equal "all" or "none".
func ScopeKey(s Scope, ids []string) string {
	if !s.Restricted {
		return ScopeKeyAll
	}
	if len(ids) == 0 {
		return ScopeKeyNone
	}
	sorted := make([]string, len(ids))
	copy(sorted, ids)
	sort.Strings(sorted)
	uniq := sorted[:0]
	for i, id := range sorted {
		if i == 0 || id != sorted[i-1] {
			uniq = append(uniq, id)
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(uniq, ",")))
	return hex.EncodeToString(sum[:])
}
