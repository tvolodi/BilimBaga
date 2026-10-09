// Package deptscope implements department-subtree scoping for admin read APIs
// (ISS-165). A department_admin may only see data about employees of its own
// department and that department's descendants; every other role is unchanged.
//
// The scope is derived from the authenticated principal carried in the request
// context (role + department id set by auth.Authenticate), so a handler,
// service or repository cannot forget to forward it.
package deptscope

import (
	"context"
	"fmt"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// RoleDepartmentAdmin is the only role subject to department scoping.
const RoleDepartmentAdmin = "department_admin"

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
}

// FromContext derives the scope from the authenticated principal in ctx.
func FromContext(ctx context.Context) Scope {
	if ctxkeys.RoleFromCtx(ctx) != RoleDepartmentAdmin {
		return Scope{}
	}
	return Scope{
		Restricted:   true,
		DepartmentID: ctxkeys.DepartmentIDFromCtx(ctx),
		UserID:       ctxkeys.UserIDFromCtx(ctx),
	}
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
