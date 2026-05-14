// Package ctxkeys defines shared typed context keys and their accessors.
// It is intentionally tiny — no imports beyond "context" — so that it can be
// imported by any domain package without creating import cycles.
package ctxkeys

import "context"

type contextKey int

const (
	CtxUserID       contextKey = iota // authenticated user's UUID string
	CtxRole                           // authenticated user's role name
	CtxDepartmentID                   // authenticated user's department UUID string
	CtxTenantID                       // current tenant identifier string
)

// UserIDFromCtx returns the user ID stored in ctx by auth.Authenticate(), or "" if absent.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxUserID).(string)
	return v
}

// RoleFromCtx returns the role stored in ctx by auth.Authenticate(), or "" if absent.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxRole).(string)
	return v
}

// DepartmentIDFromCtx returns the department ID stored in ctx by auth.Authenticate().
// May be an empty string if the user has no department assigned.
func DepartmentIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxDepartmentID).(string)
	return v
}

// TenantIDFromCtx returns the tenant ID stored in ctx by auth.TenantContext().
func TenantIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxTenantID).(string)
	return v
}
