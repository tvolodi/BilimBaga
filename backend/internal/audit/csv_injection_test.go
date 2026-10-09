package audit_test

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-191: audit export guards every user-controlled text cell.
func TestHandler_Export_FormulaInjectionGuard(t *testing.T) {
	name := `=HYPERLINK("http://evil","x")`
	etype := "+1"
	ip := "@SUM(A1)"
	meta := []byte("-2+3")
	svc := &mockService{
		exportFn: func(_ context.Context, _ string, _ audit.AuditFilters) ([]audit.AuditEntry, error) {
			return []audit.AuditEntry{
				{ActorName: &name, Action: "\t=1", EntityType: &etype, IP: &ip, Metadata: meta, CreatedAt: time.Now().UTC()},
				{Action: "user.create", CreatedAt: time.Now().UTC()},
			}, nil
		},
	}
	h := audit.NewHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export", nil)
	req = req.WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()
	h.Export(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	recs, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 3)
	assert.Equal(t, `'=HYPERLINK("http://evil","x")`, recs[1][2])
	assert.Equal(t, "'\t=1", recs[1][3])
	assert.Equal(t, "'+1", recs[1][4])
	assert.Equal(t, "'@SUM(A1)", recs[1][6])
	assert.Equal(t, "'-2+3", recs[1][7])
	assert.Equal(t, "user.create", recs[2][3])
	assert.Equal(t, "", recs[2][2])
}
