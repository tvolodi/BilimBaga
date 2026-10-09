package users

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-240 / FR-BB117 D-4: strict subset, no self role change / self deactivate (super_admin
// included), 404 for unknown and out-of-scope ids, malformed ids, ambiguous import names.

const (
	uuidA = "11111111-1111-4111-8111-111111111111"
	uuidB = "22222222-2222-4222-8222-222222222222"
)

// ---- strict subset ---------------------------------------------------------

func TestD4_StrictSubset_EqualPermissionSetIsPeer(t *testing.T) {
	_, svc := d1Setup()
	s := svc.(*service)
	// equal set (different name): refused for management and for assignment
	assert.ErrorIs(t, s.canReachRole("da_clone", "department_admin"), ErrForbidden)
	assert.ErrorIs(t, s.roleAssignableBy("da_clone", "department_admin"), ErrForbidden)
	assert.ErrorIs(t, s.canReachRole("peer_custom", d1Custom), ErrForbidden)
	assert.ErrorIs(t, s.canReachRole(d1Custom, d1Custom), ErrForbidden, "a custom role never reaches its own role")
	// a proper subset stays allowed
	assert.NoError(t, s.canReachRole("narrow_custom", "department_admin"))
	assert.NoError(t, s.canReachRole("narrow_custom", d1Custom))
	// a superset is refused
	assert.ErrorIs(t, s.canReachRole("portal_custom", d1Custom), ErrForbidden)
}

func TestD4_StrictSubset_CreateAndUpdateCannotMintPeer(t *testing.T) {
	repo, svc := d1Setup()
	ctx := context.Background()
	repo.roles["da_clone"] = "role-clone"
	repo.roles["peer_custom"] = "role-peerc"

	_, err := svc.CreateUser(ctx, CreateRequest{Email: "x@example.com", FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-clone"}, "department_admin", "dept-1", "da", "")
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.CreateUser(ctx, CreateRequest{Email: "x@example.com", FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-peerc"}, d1Custom, "dept-1", "cust", "")
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.UpdateUser(ctx, "emp", UpdateRequest{FullName: "E", DepartmentID: strPtr("dept-1"), RoleID: "role-clone"}, "department_admin", "dept-1", "da", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "employee", repo.users["emp"].RoleName, "no write on refusal")
	_, err = svc.UpdateUser(ctx, "emp", UpdateRequest{FullName: "E", DepartmentID: strPtr("dept-1"), RoleID: "role-narrow"}, "department_admin", "dept-1", "da", "")
	assert.NoError(t, err, "proper subset still assignable on update")
}

func TestD4_Import_PeerAndSensitiveRowsRejected_CommitWritesNothing(t *testing.T) {
	repo, svc := d1Setup()
	repo.roles["department_admin"] = "role-da"
	repo.roles["examiner"] = "role-ex"
	repo.roles["employee"] = "role-emp"
	repo.roles["sensitive_custom"] = "role-sens"
	repo.roleByID["role-sens"] = "sensitive_custom"
	repo.roles["da_clone"] = "role-clone"
	rows := []CSVRow{
		{RowNum: 2, Email: "a@example.com", FullName: "A", DepartmentName: "Engineering", RoleName: "department_admin"},
		{RowNum: 3, Email: "b@example.com", FullName: "B", DepartmentName: "Engineering", RoleName: "sensitive_custom"},
		{RowNum: 4, Email: "c@example.com", FullName: "C", DepartmentName: "Engineering", RoleName: "da_clone"},
		{RowNum: 5, Email: "d@example.com", FullName: "D", DepartmentName: "Engineering", RoleName: "employee"},
	}
	before := len(repo.users)
	res, err := svc.ImportUsers(context.Background(), rows, true, "department_admin", "dept-1", "da", "")
	require.NoError(t, err)
	require.Len(t, res.Valid, 1)
	assert.Equal(t, "d@example.com", res.Valid[0].Email)
	require.Len(t, res.Errors, 3)
	for _, e := range res.Errors {
		assert.Contains(t, *e.Error, "cannot be assigned")
	}
	assert.Equal(t, before+1, len(repo.users), "only the allowed row was written")

	// examiner-level caller: examiner row (peer) is refused
	res, err = svc.ImportUsers(context.Background(), []CSVRow{{RowNum: 2, Email: "e@example.com", FullName: "E", DepartmentName: "Engineering", RoleName: "examiner"}}, true, "examiner", "dept-1", "ex", "")
	require.NoError(t, err)
	assert.Len(t, res.Errors, 1)
	assert.Len(t, res.Valid, 0)
}

// ---- self protection -------------------------------------------------------

func TestD4_SuperAdmin_CannotChangeOwnRoleOrDeactivateSelf(t *testing.T) {
	repo, svc := d1Setup()
	ctx := context.Background()
	_, err := svc.UpdateUser(ctx, "sa", UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: "role-da"}, "super_admin", "dept-1", "sa", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "super_admin", repo.users["sa"].RoleName)

	assert.ErrorIs(t, svc.DeactivateUser(ctx, "sa", "super_admin", "dept-1", "sa", ""), ErrForbidden)
	assert.Equal(t, "active", repo.users["sa"].Status)
	assert.Empty(t, repo.revoked)

	// same-role self update (rename) is still fine, also with different id casing
	_, err = svc.UpdateUser(ctx, "sa", UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: "ROLE-SA"}, "super_admin", "dept-1", "sa", "")
	assert.NoError(t, err)
}

func TestD4_NonSuperAdmin_CannotDeactivateSelfOrChangeOwnRole(t *testing.T) {
	repo, svc := d1Setup()
	ctx := context.Background()
	for _, c := range []struct{ id, role string }{{"da", "department_admin"}, {"cust", d1Custom}, {"ex", "examiner"}} {
		assert.ErrorIs(t, svc.DeactivateUser(ctx, c.id, c.role, "dept-1", c.id, ""), ErrForbidden, c.role)
		assert.Equal(t, "active", repo.users[c.id].Status, c.role)
	}
	_, err := svc.UpdateUser(ctx, "cust", UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: "role-narrow"}, d1Custom, "dept-1", "cust", "")
	assert.ErrorIs(t, err, ErrForbidden, "custom caller changing own role")
	assert.Equal(t, d1Custom, repo.users["cust"].RoleName)
}

// ---- 404 for unknown and out-of-scope ---------------------------------------

func TestD4_UnknownAndOutOfScopeAreIndistinguishable(t *testing.T) {
	repo, svc := d1Setup()
	repo.users["far"] = makeUser("far", "dept-2", "role-emp", "employee")
	ctx := context.Background()
	for _, id := range []string{"far", "does-not-exist"} {
		_, e1 := svc.GetUser(ctx, id, "department_admin", "da", "dept-1")
		_, e2 := svc.UpdateUser(ctx, id, UpdateRequest{FullName: "x", DepartmentID: strPtr("dept-1"), RoleID: "role-emp"}, "department_admin", "dept-1", "da", "")
		e3 := svc.DeactivateUser(ctx, id, "department_admin", "dept-1", "da", "")
		_, e4 := svc.ResetPassword(ctx, id, "department_admin", "dept-1", "da", "")
		_, e5 := svc.UnlockUser(ctx, id, "department_admin", "dept-1", "da", "")
		for i, e := range []error{e1, e2, e3, e4, e5} {
			assert.ErrorIs(t, e, ErrNotFound, "%s op %d", id, i)
		}
	}
	assertNoWrites(t, repo, "far")
}

func TestD4_NoDepartmentCallerOrTargetIs404(t *testing.T) {
	repo, svc := d1Setup()
	repo.users["nodept"] = &User{ID: "nodept", RoleID: "role-emp", RoleName: "employee", Status: "active", FullName: "Test nodept"}
	ctx := context.Background()
	// scoped caller vs target without a department
	assert.ErrorIs(t, svc.DeactivateUser(ctx, "nodept", "department_admin", "dept-1", "da", ""), ErrNotFound)
	_, err := svc.ResetPassword(ctx, "nodept", "department_admin", "dept-1", "da", "")
	assert.ErrorIs(t, err, ErrNotFound)
	// scoped caller without a department vs anything
	_, err = svc.UnlockUser(ctx, "emp", "department_admin", "", "da", "")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = svc.UpdateUser(ctx, "emp", UpdateRequest{FullName: "x", DepartmentID: strPtr("dept-1"), RoleID: "role-emp"}, "department_admin", "", "da", "")
	assert.ErrorIs(t, err, ErrNotFound)
	// create / import (no target id): still 403 / row error
	_, err = svc.CreateUser(ctx, CreateRequest{Email: "n@example.com", FullName: "N", RoleID: "role-emp"}, "department_admin", "", "da", "")
	assert.ErrorIs(t, err, ErrForbidden)
	res, _ := svc.ImportUsers(ctx, []CSVRow{{RowNum: 2, Email: "n@example.com", FullName: "N", DepartmentName: "Engineering", RoleName: "employee"}}, false, "department_admin", "", "da", "")
	assert.Len(t, res.Errors, 1)
}

func TestD4_InactiveTargetsFollowSameRules(t *testing.T) {
	repo, svc := d1Setup()
	ctx := context.Background()
	repo.users["emp"].Status = "inactive"
	_, err := svc.ResetPassword(ctx, "emp", "department_admin", "dept-1", "da", "")
	assert.NoError(t, err)
	_, err = svc.UnlockUser(ctx, "emp", "department_admin", "dept-1", "da", "")
	assert.NoError(t, err)
	assert.Equal(t, "inactive", repo.users["emp"].Status, "reset/unlock never reactivate")
	repo.users["da"].Status = "inactive"
	_, err = svc.ResetPassword(ctx, "da", "department_admin", "dept-1", "other", "")
	assert.ErrorIs(t, err, ErrForbidden, "peer rule applies to inactive targets")
}

func TestD4_UpdateForbiddenTargetWithRoleChangeWritesNothing(t *testing.T) {
	repo, svc := d1Setup()
	_, err := svc.UpdateUser(context.Background(), "da", UpdateRequest{FullName: "Hacked", DepartmentID: strPtr("dept-1"), RoleID: "role-emp"}, "department_admin", "dept-1", "other", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "Test da", repo.users["da"].FullName)
	assert.Equal(t, "department_admin", repo.users["da"].RoleName)
}

// ---- malformed ids ----------------------------------------------------------

func TestD4_Repository_MalformedIDs(t *testing.T) {
	f := &recDB{}
	db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "not-a-uuid")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = repo.GetRoleNameByID(ctx, "not-a-uuid")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = repo.Create(ctx, "a@example.com", "A", "h", nil, "not-a-uuid")
	assert.ErrorIs(t, err, ErrValidation)
	_, err = repo.Create(ctx, "a@example.com", "A", "h", strPtr("bad"), uuidA)
	assert.ErrorIs(t, err, ErrValidation)
	_, err = repo.Update(ctx, uuidA, "A", nil, "not-a-uuid")
	assert.ErrorIs(t, err, ErrValidation)
	_, err = repo.Update(ctx, uuidA, "A", strPtr("bad"), uuidB)
	assert.ErrorIs(t, err, ErrValidation)
	assert.Empty(t, f.queries, "no SQL is issued for a malformed id")
}

func TestD4_Service_MalformedRoleIDIs422ForScopedCaller(t *testing.T) {
	_, svc := d1Setup()
	_, err := svc.CreateUser(context.Background(), CreateRequest{Email: "n@example.com", FullName: "N", DepartmentID: strPtr("dept-1"), RoleID: "not-a-uuid"}, "department_admin", "dept-1", "da", "")
	assert.ErrorIs(t, err, ErrValidation)
}

// ---- ambiguity --------------------------------------------------------------

func TestD4_Repository_AmbiguousDepartmentName(t *testing.T) {
	f := &recDB{idRows: []string{uuidA, uuidB}}
	db := sqlx.NewDb(sql.OpenDB(recConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	_, err := NewRepository(db).GetDepartmentIDByName(context.Background(), "Sales")
	assert.ErrorIs(t, err, ErrAmbiguousName)

	f.idRows = []string{uuidA}
	id, err := NewRepository(db).GetDepartmentIDByName(context.Background(), "Sales")
	require.NoError(t, err)
	assert.Equal(t, uuidA, id)

	f.idRows = []string{}
	_, err = NewRepository(db).GetDepartmentIDByName(context.Background(), "Sales")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestD4_Import_AmbiguousDepartmentIsRowError_NothingWritten(t *testing.T) {
	repo, svc := d1Setup()
	repo.ambiguous = map[string]bool{"Engineering": true}
	before := len(repo.users)
	for _, commit := range []bool{false, true} {
		res, err := svc.ImportUsers(context.Background(), []CSVRow{
			{RowNum: 2, Email: "a@example.com", FullName: "A", DepartmentName: "Engineering", RoleName: "employee"},
			{RowNum: 3, Email: "b@example.com", FullName: "B", DepartmentName: "Sales", RoleName: "employee"},
		}, commit, "super_admin", "", "sa", "")
		require.NoError(t, err)
		require.Len(t, res.Errors, 1)
		assert.Contains(t, *res.Errors[0].Error, "ambiguous department name")
		assert.Equal(t, 2, res.Errors[0].RowNum)
		assert.Len(t, res.Valid, 1)
	}
	assert.Equal(t, before+1, len(repo.users), "only the unambiguous row was committed (once)")
}

func TestD4_Import_CommitUsesValidatedIDs(t *testing.T) {
	repo, svc := d1Setup()
	var gotDept, gotRole string
	repo.createFn = func(_ context.Context, _, _, _ string, d *string, roleID string) (*User, error) {
		gotDept, gotRole = *d, roleID
		return &User{}, nil
	}
	_, err := svc.ImportUsers(context.Background(), []CSVRow{{RowNum: 2, Email: "a@example.com", FullName: "A", DepartmentName: "Sales", RoleName: "employee"}}, true, "super_admin", "", "sa", "")
	require.NoError(t, err)
	assert.Equal(t, "dept-2", gotDept)
	assert.Equal(t, "role-emp", gotRole)
}

// ---- handler mapping --------------------------------------------------------

func realHandler() (*mockRepo, *Handler) {
	repo, svc := d1Setup()
	return repo, NewHandler(svc, nil)
}

func TestD4_Handler_UpdateAndCreateMapStatuses(t *testing.T) {
	repo, h := realHandler()
	repo.users["far"] = makeUser("far", "dept-2", "role-emp", "employee")

	put := func(id, role, dept, caller string, body map[string]any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+id, bytes.NewReader(b))
		req = withChiID(withAuthCtx(req, caller, role, dept), id)
		w := httptest.NewRecorder()
		h.UpdateUser(w, req)
		return w
	}
	upd := map[string]any{"full_name": "X", "department_id": "dept-1", "role_id": "role-emp"}

	assert.Equal(t, http.StatusForbidden, put("da", "department_admin", "dept-1", "other", upd).Code, "peer target")
	assert.Equal(t, http.StatusNotFound, put("far", "department_admin", "dept-1", "da", upd).Code, "out of scope")
	assert.Equal(t, http.StatusNotFound, put("ghost", "department_admin", "dept-1", "da", upd).Code, "unknown")
	assert.Equal(t, http.StatusForbidden, put("sa", "super_admin", "", "sa", map[string]any{"full_name": "X", "role_id": "role-da"}).Code, "super_admin self role change")

	create := func(body map[string]any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := withAuthCtx(httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b)), "da", "department_admin", "dept-1")
		w := httptest.NewRecorder()
		h.CreateUser(w, req)
		return w
	}
	w := create(map[string]any{"email": "n@example.com", "full_name": "N", "department_id": "dept-1", "role_id": "role-da"})
	assert.Equal(t, http.StatusForbidden, w.Code, "peer role assignment")
	assert.NotContains(t, w.Body.String(), "temporary_password")
	w = create(map[string]any{"email": "n@example.com", "full_name": "N", "department_id": "dept-1", "role_id": "not-a-uuid"})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code, "malformed role id")
}

func TestD4_Handler_DeactivateResetUnlockMapStatuses(t *testing.T) {
	repo, h := realHandler()
	repo.users["far"] = makeUser("far", "dept-2", "role-emp", "employee")
	call := func(fn http.HandlerFunc, id, role, dept, caller string) int {
		req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), caller, role, dept), id)
		w := httptest.NewRecorder()
		fn(w, req)
		return w.Code
	}
	assert.Equal(t, http.StatusForbidden, call(h.DeactivateUser, "sa", "super_admin", "", "sa"), "self deactivate")
	for name, fn := range map[string]http.HandlerFunc{"deactivate": h.DeactivateUser, "reset": h.ResetPassword, "unlock": h.UnlockUser} {
		assert.Equal(t, http.StatusNotFound, call(fn, "far", "department_admin", "dept-1", "da"), name+" out of scope")
		assert.Equal(t, http.StatusNotFound, call(fn, "ghost", "department_admin", "dept-1", "da"), name+" unknown")
	}
}
