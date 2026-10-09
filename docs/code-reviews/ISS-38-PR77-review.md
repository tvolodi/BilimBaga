# Code Review: ISS-38 / PR #77 (swarm/38-reports-left-join)

Verdict: **CHANGES REQUESTED** (one blocking item; the LEFT JOIN change itself is correct)

## Scope
- `backend/internal/reports/repository.go`: `JOIN` -> `LEFT JOIN resolved_assignments` in `GetCompletionRateByExam` and `GetDashboardCompletionRatesForRange`
- New: `fakedb_test.go`, `repository_completion_test.go`, `docs/issue-reports/ISS-38-completion-rate-left-join.md`

## SQL correctness (both queries): PASS
- Join semantics: `exams e LEFT JOIN resolved_assignments ra ... LEFT JOIN exam_sessions es ON es.exam_id=e.id AND es.user_id=ra.user_id`. Right-side filters are all in ON. WHERE holds only base-table predicates (`e.status='active'`, `e.tenant_id`), so the join does not degrade to inner.
- Range query: the `es.submitted_at BETWEEN $2 AND $3` filter sits in the sessions ON clause, so it does not drop exams or assigned users.
- NULL rows: for an unassigned exam, `ra.user_id` is NULL and `es.user_id = NULL` never matches. The result is one row where `COUNT(DISTINCT ra.user_id)` is 0 and both CASE counts are 0. COUNT ignores NULL, so no spurious 1.
- Fan-out: multiple sessions per user (attempts) multiply rows, but every aggregate is `COUNT(DISTINCT ra.user_id)`, so counts are not inflated. This was already the case. `resolved_assignments` uses UNION, so it is deduplicated.
- GROUP BY `e.id, e.title` covers all non-aggregated columns. The scan target is non-null int and COUNT never returns NULL.
- CTEs: the `resolved_assignments` CTEs are unchanged and not touched by the LEFT JOIN.

## Consumers: PASS
- `pdf.go:69` guards `AssignedCount > 0` (blank pass-rate cell).
- `CompletionBarChart.tsx:52-57` guards `assigned_count > 0`.
- `computeKpis` (`frontend/src/api/dashboard.ts:123,127`) guards `totalAssigned > 0`.
- The service passes rows through. The API returns raw counts only.

## Blocking issue
**B1. `GetDashboardCompletionRatesForRange` filters `e.tenant_id = $1`, but no migration gives `exams` a `tenant_id` column.**
- `012_exams.up.sql` creates `exams` without it, and `grep tenant_id backend/migrations` finds only `audit_log`.
- Against real Postgres this query most likely fails with `column e.tenant_id does not exist`, so the PDF export path that half of #38 targets can never return rows. The same applies to `es.tenant_id` in `StreamUserRecordSessions` and `es.tenant_id` / `e.tenant_id` in `StreamExamResultSessions`, if present.
- This predates the PR, but the fake-driver tests cannot catch it. The "fixed" function is unverified and likely unusable.
- Under the project "Unblock-Everything" directive, either:
  - (a) verify against a real DB (`make migrate` plus a query run). If the column is missing, drop the predicate or add a migration; or
  - (b) confirm that the column exists via a schema path I could not find, and note it in the issue report.

## Non-blocking
- N1. `GetCompletionRateByExam` (dashboard) takes no tenant argument and has no tenant filter, so it is not tenant-scoped. This is consistent with the single-tenant schema and unchanged by the PR. The LEFT JOIN now also surfaces unassigned active exams from any tenant, if multi-tenancy ever arrives. Add a TODO or a follow-up issue.
- N2. `resolved_assignments` direct-user rows (`assignee_type='user'`) are not filtered to active users, unlike the department and 'all' branches. A pre-existing inconsistency.
- N3. Exams with only inactive or empty-department assignees also appear with 0. This is consistent with AC-2.

## Test quality
Text-only assertions on the SQL (via the fake driver) are acceptable as a cheap guard against reverting to the inner join. They are not sufficient evidence for this fix, because join semantics, NULL aggregation, and column existence (see B1) are only checked by a real database. The mapping, service, and handler zero-count tests are fine but trivially pass; they would also pass before the fix.

Missing:
1. A Postgres-backed integration test (testcontainers, or the existing E2E/seed DB) with: an active exam with no assignments (expect a row with 0/0/0); an assigned exam with sessions across multiple attempts (expect no inflated counts); a draft/archived unassigned exam (expect absent); and the range query with a session outside the range (expect the exam is still listed, completed=0). This would also have caught B1.
2. A negative assertion that the regex `innerJoinAssignments` rejects `LEFT JOIN`. It does, via `^\s*JOIN`, but is fragile to formatting changes.
3. A PDF test for a zero-assigned row (blank pass rate, no panic) and a frontend `CompletionBarChart` test for `assigned_count: 0`.

## Checklist notes
- Parameterized queries only; no secrets; errors wrapped with context.
- Migration: none required for the join change (B1 may add one).
- The issue report is present and matches the change.
