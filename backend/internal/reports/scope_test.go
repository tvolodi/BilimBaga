package reports

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-165: department_admin is limited to its department subtree on every
// reports endpoint; other roles are unchanged.

const (
	deptA     = "11111111-1111-1111-1111-111111111111"
	adminUser = "22222222-2222-2222-2222-222222222222"
	otherUser = "33333333-3333-3333-3333-333333333333"
	zeroUUID  = "00000000-0000-0000-0000-000000000000"
)

func principal(role, deptID string) context.Context {
	ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, role)
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, adminUser)
	return context.WithValue(ctx, ctxkeys.CtxDepartmentID, deptID)
}

// scopedRequest builds a request carrying the chi {id} param and the principal.
func scopedRequest(role, deptID, id string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	ctx := context.WithValue(principal(role, deptID), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func neverProbed(t *testing.T) func(context.Context, string) (bool, error) {
	return func(context.Context, string) (bool, error) {
		t.Helper()
		t.Fatal("unexpected scope probe")
		return false, nil
	}
}

// ── Service: per-employee record / progress / CSV ─────────────────────────────

func TestScope_GetUserRecord(t *testing.T) {
	cases := []struct {
		name      string
		ctx       context.Context
		target    string
		inScope   bool
		wantErr   error
		wantProbe bool
	}{
		{"own department (incl. descendants) OK", principal("department_admin", deptA), otherUser, true, nil, true},
		{"other department not found", principal("department_admin", deptA), otherUser, false, ErrNotFound, true},
		{"target without department not found", principal("department_admin", deptA), otherUser, false, ErrNotFound, true},
		{"department_admin without department not found", principal("department_admin", ""), otherUser, false, ErrNotFound, true},
		{"own record always OK", principal("department_admin", ""), adminUser, false, nil, false},
		{"super_admin unchanged", principal("super_admin", ""), otherUser, false, nil, false},
		{"examiner unchanged", principal("examiner", deptA), otherUser, false, nil, false},
		{"no principal unchanged", context.Background(), otherUser, false, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			probed := false
			historyCalled := false
			repo := &mockRepo{
				userInScopeFn: func(_ context.Context, id string) (bool, error) {
					probed = true
					assert.Equal(t, tc.target, id)
					return tc.inScope, nil
				},
				getUserSessionHistoryFn: func(context.Context, string, int, int) ([]SessionRecord, error) {
					historyCalled = true
					return []SessionRecord{}, nil
				},
			}
			rec, _, err := NewService(repo).GetUserRecord(tc.ctx, tc.target, 1, 20)
			assert.Equal(t, tc.wantProbe, probed)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, rec)
				assert.False(t, historyCalled, "no data may be read for an out-of-scope user")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.target, rec.UserID)
		})
	}
}

func TestScope_GetUserRecord_UnknownUser(t *testing.T) {
	repo := &mockRepo{getUserInfoFn: func(context.Context, string) (*userInfoRow, error) { return nil, ErrNotFound }}
	_, _, err := NewService(repo).GetUserRecord(principal("department_admin", deptA), otherUser, 1, 20)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestScope_GetUserRecord_ScopeLookupError(t *testing.T) {
	boom := errors.New("db down")
	repo := &mockRepo{userInScopeFn: func(context.Context, string) (bool, error) { return false, boom }}
	_, _, err := NewService(repo).GetUserRecord(principal("department_admin", deptA), otherUser, 1, 20)
	require.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, ErrNotFound, "a failed scope lookup is a 500, not a hidden 404")
}

func TestScope_GetUserProgress(t *testing.T) {
	repo := &mockRepo{userInScopeFn: func(context.Context, string) (bool, error) { return false, nil }}
	svc := NewService(repo)

	_, err := svc.GetUserProgress(principal("department_admin", deptA), otherUser)
	require.ErrorIs(t, err, ErrNotFound)

	repo.userInScopeFn = func(context.Context, string) (bool, error) { return true, nil }
	res, err := svc.GetUserProgress(principal("department_admin", deptA), otherUser)
	require.NoError(t, err)
	assert.Equal(t, otherUser, res.UserID)

	repo.userInScopeFn = neverProbed(t)
	_, err = svc.GetUserProgress(principal("super_admin", ""), otherUser)
	require.NoError(t, err)
}

func TestScope_StreamUserRecordCSV(t *testing.T) {
	repo := &mockRepo{userInScopeFn: func(context.Context, string) (bool, error) { return false, nil }}
	svc := NewService(repo)

	w := httptest.NewRecorder()
	err := svc.StreamUserRecordCSV(principal("department_admin", deptA), w, otherUser, "t")
	require.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, w.Body.String(), "nothing may be written for an out-of-scope user")

	repo.userInScopeFn = func(context.Context, string) (bool, error) { return true, nil }
	w = httptest.NewRecorder()
	require.NoError(t, svc.StreamUserRecordCSV(principal("department_admin", deptA), w, otherUser, "t"))
	assert.True(t, strings.HasPrefix(w.Body.String(), "exam_title,"))

	repo.userInScopeFn = neverProbed(t)
	w = httptest.NewRecorder()
	require.NoError(t, svc.StreamUserRecordCSV(principal("super_admin", ""), w, otherUser, "t"))
}

// ── Handler: out-of-scope is byte-identical to unknown (no existence leak) ──

func TestScope_Handlers_OutOfScopeIdenticalToUnknown(t *testing.T) {
	outOfScope := &mockRepo{userInScopeFn: func(context.Context, string) (bool, error) { return false, nil }}
	unknown := &mockRepo{getUserInfoFn: func(context.Context, string) (*userInfoRow, error) { return nil, ErrNotFound }}

	handlers := func(repo *mockRepo) map[string]func(http.ResponseWriter, *http.Request) {
		h := NewHandler(NewService(repo), nil)
		return map[string]func(http.ResponseWriter, *http.Request){
			"record":   h.GetUserRecord,
			"progress": h.GetUserProgress,
			"csv":      h.UserRecordCSV,
		}
	}
	scoped, real := handlers(outOfScope), handlers(unknown)
	for name := range scoped {
		t.Run(name, func(t *testing.T) {
			a := httptest.NewRecorder()
			scoped[name](a, scopedRequest("department_admin", deptA, otherUser))
			b := httptest.NewRecorder()
			real[name](b, scopedRequest("department_admin", deptA, otherUser))

			assert.Equal(t, http.StatusNotFound, a.Code)
			assert.Contains(t, a.Body.String(), `"code":"USER_NOT_FOUND"`)
			assert.Equal(t, b.Code, a.Code)
			assert.Equal(t, b.Body.String(), a.Body.String(), "response bodies must be byte-identical")
			assert.Equal(t, b.Header(), a.Header())
			assert.NotEqual(t, "text/csv; charset=utf-8", a.Header().Get("Content-Type"))
		})
	}
}

func TestScope_Handlers_OwnDepartmentAndSuperAdminOK(t *testing.T) {
	repo := &mockRepo{userInScopeFn: func(context.Context, string) (bool, error) { return true, nil }}
	h := NewHandler(NewService(repo), nil)

	for _, role := range []string{"department_admin", "super_admin"} {
		w := httptest.NewRecorder()
		h.GetUserRecord(w, scopedRequest(role, deptA, otherUser))
		assert.Equal(t, http.StatusOK, w.Code, role)

		w = httptest.NewRecorder()
		h.GetUserProgress(w, scopedRequest(role, deptA, otherUser))
		assert.Equal(t, http.StatusOK, w.Code, role)

		w = httptest.NewRecorder()
		h.UserRecordCSV(w, scopedRequest(role, deptA, otherUser))
		assert.Equal(t, http.StatusOK, w.Code, role)
		assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	}
}

func TestScope_Handlers_SuperAdminNeverProbesScope(t *testing.T) {
	repo := &mockRepo{userInScopeFn: neverProbed(t)}
	h := NewHandler(NewService(repo), nil)
	w := httptest.NewRecorder()
	h.GetUserRecord(w, scopedRequest("super_admin", "", otherUser))
	assert.Equal(t, http.StatusOK, w.Code)
}

// Dashboard, exam analytics and exam-results CSV carry the principal to the
// repository through the request context (filtered, never rejected).
func TestScope_Handlers_PrincipalReachesRepository(t *testing.T) {
	var seenRole string
	repo := &mockRepo{
		getCompletionFn: func(ctx context.Context) ([]*ExamCompletionRate, error) {
			seenRole = ctxkeys.RoleFromCtx(ctx)
			return []*ExamCompletionRate{}, nil
		},
	}
	h := NewHandler(NewService(repo), nil)
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(principal("department_admin", deptA))
	w := httptest.NewRecorder()
	h.GetDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "department_admin", seenRole)
}

// ── Repository: the department filter must be present in every scoped query ──

func lastArg(f *fakeDB, i int) driver.Value {
	a := f.args[i]
	return a[len(a)-1]
}

func TestScope_Repository_AggregateQueriesAreScoped(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	calls := map[string]func(r Repository, ctx context.Context){
		"completion":      func(r Repository, ctx context.Context) { _, _ = r.GetCompletionRateByExam(ctx) },
		"overdue":         func(r Repository, ctx context.Context) { _, _ = r.GetOverdueEmployees(ctx) },
		"recent":          func(r Repository, ctx context.Context) { _, _ = r.GetRecentActivity(ctx) },
		"avgTrack":        func(r Repository, ctx context.Context) { _, _ = r.GetAvgScoreByTrack(ctx) },
		"scoreDist":       func(r Repository, ctx context.Context) { _, _ = r.GetExamScoreDistribution(ctx, "e") },
		"summary":         func(r Repository, ctx context.Context) { _, _ = r.GetExamSummaryStats(ctx, "e") },
		"perQuestion":     func(r Repository, ctx context.Context) { _, _ = r.GetPerQuestionStats(ctx, "e") },
		"answerDist":      func(r Repository, ctx context.Context) { _, _ = r.GetAnswerDistribution(ctx, "e") },
		"examQuestions":   func(r Repository, ctx context.Context) { _, _ = r.GetExamQuestions(ctx, "e", "t") },
		"rangeCompletion": func(r Repository, ctx context.Context) { _, _ = r.GetDashboardCompletionRatesForRange(ctx, "t", from, from) },
		"topBottom":       func(r Repository, ctx context.Context) { _, _, _ = r.GetTopBottomQuestions(ctx, "t", from, from) },
		"examResultsCSV": func(r Repository, ctx context.Context) {
			if rows, err := r.StreamExamResultSessions(ctx, "e", "t"); err == nil {
				_ = rows.Close()
			}
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			// department_admin: subtree predicate present, bound to the department.
			db, f := newFakeDB(t)
			call(NewRepository(db), principal("department_admin", deptA))
			require.NotEmpty(t, f.queries)
			for i, q := range f.queries {
				assert.Contains(t, q, "WITH RECURSIVE sc_t", "query %d lacks the department-subtree filter", i)
				assert.NotContains(t, q, "@SCOPE@")
				assert.Equal(t, deptA, lastArg(f, i))
			}

			// department_admin without a department: matches nothing.
			db, f = newFakeDB(t)
			call(NewRepository(db), principal("department_admin", ""))
			assert.Equal(t, zeroUUID, lastArg(f, 0))

			// super_admin / examiner: parameter is NULL (unrestricted).
			for _, role := range []string{"super_admin", "examiner"} {
				db, f = newFakeDB(t)
				call(NewRepository(db), principal(role, deptA))
				assert.Nil(t, lastArg(f, 0), role)
			}
		})
	}
}

func TestScope_Repository_DashboardHidesExamsOutsideScope(t *testing.T) {
	db, f := newFakeDB(t)
	_, _ = NewRepository(db).GetCompletionRateByExam(principal("department_admin", deptA))
	q := f.queries[0]
	assert.Contains(t, q, "HAVING ($1::uuid IS NULL OR COUNT(DISTINCT ra.user_id) > 0)")
	assert.Contains(t, q, "LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id AND (")
}

func TestScope_Repository_UserInScope(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"in_scope"}, [][]driver.Value{{int64(1)}})
	ok, err := NewRepository(db).UserInScope(principal("department_admin", deptA), otherUser)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, f.queries[0], "WITH RECURSIVE sc_t")
	assert.Equal(t, []driver.Value{otherUser, deptA}, f.args[0])

	// Unknown user (no row) is "not in scope".
	db, _ = newFakeDB(t)
	ok, err = NewRepository(db).UserInScope(principal("department_admin", deptA), otherUser)
	require.NoError(t, err)
	assert.False(t, ok)

	// Out-of-scope user.
	db, f = newFakeDB(t)
	f.queue([]string{"in_scope"}, [][]driver.Value{{int64(0)}})
	ok, err = NewRepository(db).UserInScope(principal("department_admin", deptA), otherUser)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestScope_StreamUserRecordCSV_UnknownUserIsNotFound(t *testing.T) {
	repo := &mockRepo{getUserInfoFn: func(context.Context, string) (*userInfoRow, error) { return nil, ErrNotFound }}
	for _, ctx := range []context.Context{principal("department_admin", deptA), principal("super_admin", "")} {
		w := httptest.NewRecorder()
		err := NewService(repo).StreamUserRecordCSV(ctx, w, otherUser, "t")
		require.ErrorIs(t, err, ErrNotFound)
		assert.Empty(t, w.Body.String())
	}
}
