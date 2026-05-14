package auth

import (
	"context"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// UserIDFromCtx returns the user ID stored in ctx by Authenticate(), or "" if absent.
func UserIDFromCtx(ctx context.Context) string { return ctxkeys.UserIDFromCtx(ctx) }

// RoleFromCtx returns the role stored in ctx by Authenticate(), or "" if absent.
func RoleFromCtx(ctx context.Context) string { return ctxkeys.RoleFromCtx(ctx) }

// DepartmentIDFromCtx returns the department ID stored in ctx by Authenticate().
// May be an empty string if the user has no department assigned.
func DepartmentIDFromCtx(ctx context.Context) string { return ctxkeys.DepartmentIDFromCtx(ctx) }

// TenantIDFromCtx returns the tenant ID stored in ctx by TenantContext().
func TenantIDFromCtx(ctx context.Context) string { return ctxkeys.TenantIDFromCtx(ctx) }
