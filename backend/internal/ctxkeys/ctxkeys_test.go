package ctxkeys_test

import (
	"context"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

func TestAccessors_ReturnStoredValues(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, "u-1")
	ctx = context.WithValue(ctx, ctxkeys.CtxRole, "super_admin")
	ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, "d-1")
	ctx = context.WithValue(ctx, ctxkeys.CtxTenantID, "t-1")

	if got := ctxkeys.UserIDFromCtx(ctx); got != "u-1" {
		t.Errorf("UserIDFromCtx = %q", got)
	}
	if got := ctxkeys.RoleFromCtx(ctx); got != "super_admin" {
		t.Errorf("RoleFromCtx = %q", got)
	}
	if got := ctxkeys.DepartmentIDFromCtx(ctx); got != "d-1" {
		t.Errorf("DepartmentIDFromCtx = %q", got)
	}
	if got := ctxkeys.TenantIDFromCtx(ctx); got != "t-1" {
		t.Errorf("TenantIDFromCtx = %q", got)
	}
}

func TestAccessors_EmptyWhenAbsentOrWrongType(t *testing.T) {
	empty := context.Background()
	wrong := context.WithValue(empty, ctxkeys.CtxUserID, 42) // not a string
	for name, ctx := range map[string]context.Context{"absent": empty, "wrongType": wrong} {
		if ctxkeys.UserIDFromCtx(ctx) != "" || ctxkeys.RoleFromCtx(ctx) != "" ||
			ctxkeys.DepartmentIDFromCtx(ctx) != "" || ctxkeys.TenantIDFromCtx(ctx) != "" {
			t.Errorf("%s: expected empty strings", name)
		}
	}
}

func TestKeysAreDistinct(t *testing.T) {
	keys := []any{ctxkeys.CtxUserID, ctxkeys.CtxRole, ctxkeys.CtxDepartmentID, ctxkeys.CtxTenantID}
	seen := map[any]bool{}
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate key %v", k)
		}
		seen[k] = true
	}
}
