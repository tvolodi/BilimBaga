package audit_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/stretchr/testify/assert"
)

// TestWrite_NilWriter_IsNoOp verifies that calling Write on a nil *Writer
// does not panic and is a no-op.
func TestWrite_NilWriter_IsNoOp(t *testing.T) {
	var w *audit.Writer
	ctx := context.WithValue(context.Background(), ctxkeys.CtxTenantID, "public")
	r := httptest.NewRequest("POST", "/", nil)

	// Must not panic.
	assert.NotPanics(t, func() {
		w.Write(ctx, r, "test.action", "entity", nil, nil)
	})
}

// TestWrite_MissingTenantID_DoesNotPanic verifies that when tenant_id is
// absent from the context, Write does not panic and silently drops the event.
func TestWrite_MissingTenantID_DoesNotPanic(t *testing.T) {
	// Writer backed by a nil DB is only safe when tenantID is missing, because
	// the method returns early before attempting any DB operations.
	w := audit.NewWriter(nil, nil)
	r := httptest.NewRequest("POST", "/", nil)

	assert.NotPanics(t, func() {
		w.Write(context.Background(), r, "test.action", "entity", nil, nil)
	})
}
