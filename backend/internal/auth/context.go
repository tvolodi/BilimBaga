package auth

import "context"

type contextKey int

const (
	ctxUserID       contextKey = iota
	ctxRole
	ctxDepartmentID
	ctxTenantID
)

// UserIDFromCtx returns the user ID stored in ctx by Authenticate(), or "" if absent.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

// RoleFromCtx returns the role stored in ctx by Authenticate(), or "" if absent.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxRole).(string)
	return v
}

// DepartmentIDFromCtx returns the department ID stored in ctx by Authenticate().
// May be an empty string if the user has no department assigned.
func DepartmentIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxDepartmentID).(string)
	return v
}

// TenantIDFromCtx returns the tenant ID stored in ctx by TenantContext().
func TenantIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxTenantID).(string)
	return v
}
