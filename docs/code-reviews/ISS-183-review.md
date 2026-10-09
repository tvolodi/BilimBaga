# Code Review: ISS-183 (run swarm-183)

Result: PASS

Scope: backend/internal/exams/{repository.go,service.go,service_test.go,assignments_scope_test.go}, docs/issue-reports/ISS-183-assignments-deptscope.md.
Checks run: go vet and go test on ./internal/exams ./internal/deptscope are clean. No real DB, so the SQL was reviewed by reasoning only.

## SQL analysis
- $2 typing: every use is `$2::uuid` (outer CTE seed, `deptscope.Predicate`). The type is inferred consistently, so there is no "could not determine data type" risk. $1 is only compared with uuid columns.
- Recursive CTE placement: `WITH RECURSIVE sc_t(id) AS (...), assignment_base AS (...)` is valid chaining. The nested `WITH RECURSIVE sc_t` inside `Predicate` sits in a sub-select and shadows the outer one, which Postgres allows. Both versions are equivalent. The outer sc_t is referenced from `ea.assignee_id IN (SELECT id FROM sc_t)`, which is in the WHERE of the assignment_base CTE, so it is in scope. The fallback query defines its own sc_t.
- NULL unrestricted: for super_admin, Arg() is nil, so `$2::uuid IS NULL` is true and the whole assignee filter passes. The sc_t seed `= NULL` yields nothing, which is harmless. The users/sessions predicates short-circuit true. Behaviour matches the old output.
- Restricted caller without a department: Arg() is the all-zero UUID, so sc_t is empty. Only the "all" row is shown, with zero counts. Fails closed.
- ea.assignee_id (UUID, migration 013) against users.id and departments.id (UUID): no type mismatch. `assignee_type='user'` rows are filtered via the Predicate subselect. `assignee_type='all'` has a NULL assignee_id and is guarded by the `assignee_type='all'` OR branch before any id comparison.
- 'all' counts: total_users uses `users.id` in `Predicate`. completed/passed use `(ab.assignee_type='all' AND Predicate(es.user_id))`. The user and department branches are unchanged, and are in-scope by construction because the row was filtered. The scoping is correct.
- Fallback query (listAssignmentsNoSessions): same markers, same two args ($1, $2), same filtering. The binding is consistent with the main query.
- Marker replacement: strings.ReplaceAll after the raw literal. No user input is concatenated, only fixed fragments plus the constant "$2" parameter, so the query stays parameterized.

## Findings
- [Medium] assignments_scope_test.go: `TestListAssignments_ReturnsOnlyScopedRepoRows` mostly tests the mock, because the mock itself performs the filtering. It adds little. The real SQL has only a string-expansion guard. Mitigation: the PR is labelled needs-live-db, and it should be exercised against Postgres before release.
- [Low] DeleteAssignment is not scoped (an out-of-scope assignment can still be deleted by id). This is outside the ISS-183 list-endpoint scope. Suggest a follow-up issue.
- [Low] Pre-existing: department stats use 2-level counts. The issue report already notes this as unchanged.
- No Critical or High findings. The handler is unchanged and thin. SQL stays in the repository. Errors are wrapped. No secrets and no os.Getenv.

AC coverage (from issue report): admin/examiner/custom role scoped, super_admin unrestricted, 'all' counts scoped, fallback scoped, exam still visible: all covered.

Summary: the SQL is sound by inspection; recommend a live-DB smoke before merge.
