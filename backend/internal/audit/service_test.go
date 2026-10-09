package audit_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var auditCols = []string{"id", "tenant_id", "actor_id", "actor_name", "action", "entity_type", "entity_id", "ip", "metadata", "created_at"}

func queueList(f *fakeDB, total int64, n int) {
	f.queue([]string{"count"}, [][]driver.Value{{total}})
	rows := make([][]driver.Value, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, []driver.Value{"id", "t1", nil, nil, "auth.login", nil, nil, nil, []byte(`{}`), time.Now().UTC()})
	}
	f.queue(auditCols, rows)
}

func lastArgs(f *fakeDB) []driver.Value { return f.args[len(f.args)-1] }

func TestService_List_ClampsPaging(t *testing.T) {
	for _, tc := range []struct {
		name          string
		page, perPage int
		wantLimit     int64
		wantOffset    int64
	}{
		{"defaults", 0, 0, 50, 0},
		{"cap", 1, 1000, 200, 0},
		{"third page", 3, 10, 10, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, f := newFakeDB(t)
			queueList(f, 2, 2)
			entries, total, err := audit.NewService(db).List(context.Background(), "t1", audit.AuditFilters{}, tc.page, tc.perPage)
			require.NoError(t, err)
			assert.Equal(t, 2, total)
			assert.Len(t, entries, 2)
			a := lastArgs(f)
			assert.Equal(t, "t1", a[0])
			assert.Equal(t, tc.wantLimit, a[len(a)-2])
			assert.Equal(t, tc.wantOffset, a[len(a)-1])
		})
	}
}

func TestService_List_AppliesAllFilters(t *testing.T) {
	db, f := newFakeDB(t)
	queueList(f, 0, 0)
	actorID, actor, action, et := "u1", "ann", "auth.login", "user"
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	entries, total, err := audit.NewService(db).List(context.Background(), "t1", audit.AuditFilters{
		ActorID: &actorID, Actor: &actor, Action: &action, EntityType: &et, From: &from, To: &to,
	}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NotNil(t, entries, "empty result must be a non-nil slice")
	assert.Empty(t, entries)

	q := f.queries[len(f.queries)-1]
	assert.Contains(t, q, "al.tenant_id = $1", "tenant scope must be present")
	for _, frag := range []string{"al.actor_id", "ILIKE", "al.action", "al.entity_type", "created_at >=", "created_at <="} {
		assert.Contains(t, q, frag)
	}
	a := lastArgs(f)
	assert.Equal(t, "t1", a[0])
	assert.Equal(t, "u1", a[1])
	assert.Equal(t, "%ann%", a[2])
}

func TestService_List_CountError(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, _, err := audit.NewService(db).List(context.Background(), "t1", audit.AuditFilters{}, 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit.Service.List")
	assert.ErrorContains(t, err, "boom")
}

func TestService_List_ScanError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(1)}})
	f.queue([]string{"bogus"}, [][]driver.Value{{"x"}}) // column with no matching struct field
	_, _, err := audit.NewService(db).List(context.Background(), "t1", audit.AuditFilters{}, 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit.Service.List")
}

func TestService_Export_UsesMaxRows(t *testing.T) {
	db, f := newFakeDB(t)
	queueList(f, 3, 3)
	entries, err := audit.NewService(db).Export(context.Background(), "t1", audit.AuditFilters{})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	a := lastArgs(f)
	assert.Equal(t, int64(10000), a[len(a)-2])
	assert.Equal(t, int64(0), a[len(a)-1])
}

func TestService_Export_Error(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("boom")
	_, err := audit.NewService(db).Export(context.Background(), "t1", audit.AuditFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit.Service.Export")
}

func writerCtx(tenant, user string) context.Context {
	ctx := context.Background()
	if tenant != "" {
		ctx = context.WithValue(ctx, ctxkeys.CtxTenantID, tenant)
	}
	if user != "" {
		ctx = context.WithValue(ctx, ctxkeys.CtxUserID, user)
	}
	return ctx
}

func TestWriter_InsertsRow(t *testing.T) {
	db, f := newFakeDB(t)
	w := audit.NewWriter(db, nil)
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "10.0.0.1"
	id := "ent-1"

	w.Write(writerCtx("t1", "u1"), r, "exam.create", "exam", &id, map[string]any{"k": "v"})

	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "INSERT INTO audit_log")
	a := f.args[0]
	assert.Equal(t, "t1", a[0])
	assert.Equal(t, "u1", a[1])
	assert.Equal(t, "exam.create", a[2])
	assert.Equal(t, "exam", a[3])
	assert.Equal(t, "ent-1", a[4])
	assert.Equal(t, "10.0.0.1", a[5])
	assert.JSONEq(t, `{"k":"v"}`, string(a[6].([]byte)))
}

func TestWriter_NullableFieldsAndDefaultMetadata(t *testing.T) {
	db, f := newFakeDB(t)
	w := audit.NewWriter(db, nil)
	w.Write(writerCtx("t1", ""), httptest.NewRequest("GET", "/", nil), "a.b", "", nil, nil)

	require.Len(t, f.args, 1)
	a := f.args[0]
	assert.Nil(t, a[1], "no user -> NULL actor")
	assert.Nil(t, a[3], "empty entity type -> NULL")
	assert.Nil(t, a[4])
	assert.Equal(t, "{}", string(a[6].([]byte)))
}

func TestWriter_UnmarshalableMetadata_FallsBackToEmptyObject(t *testing.T) {
	db, f := newFakeDB(t)
	w := audit.NewWriter(db, nil)
	w.Write(writerCtx("t1", "u1"), httptest.NewRequest("GET", "/", nil), "a.b", "", nil, map[string]any{"c": make(chan int)})

	require.Len(t, f.args, 1)
	assert.Equal(t, "{}", string(f.args[0][6].([]byte)))
}

func TestWriter_DBError_IsSwallowed(t *testing.T) {
	db, f := newFakeDB(t)
	f.xErr = errors.New("insert failed")
	w := audit.NewWriter(db, nil)
	assert.NotPanics(t, func() {
		w.Write(writerCtx("t1", "u1"), httptest.NewRequest("GET", "/", nil), "a.b", "", nil, nil)
	})
	assert.Len(t, f.queries, 1)
}
