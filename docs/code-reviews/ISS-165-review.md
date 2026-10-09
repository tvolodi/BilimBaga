# Code Review: ISS-165 department_admin report scoping

Verdict: PASS (no Critical/High findings; 5 Medium/Low items, none blocking)

Method: static read of `git diff` and new files; nothing was executed (host memory constraint).
Coverage verified: deptscope (scope/store/middleware), reports repository/service/handler, ai repository/service, sessions ListGradingQueue, router wiring, every `/admin/*` route in router.go.

## What was checked and found correct
- Predicate: `$N::uuid IS NULL OR col IN (users in recursive dept subtree)`. NULL arg = unrestricted; restricted with no department binds the nil-UUID, so it matches nothing (fail-closed). `sc_` aliases cannot shadow outer aliases. Unqualified `user_id` in reports/ai queries only appears inside `FROM exam_sessions` subqueries, and the predicate's inner tables (`users`, `departments`) have no `user_id` column, so it is not ambiguous.
- $N numbering is correct in every modified query: reports (`$1` completion/overdue/recent/avg-track; `$2` analytics and CSV queries; `$3` dashboard range and top/bottom), ai (`$1` exam, `$2` scope; all four occurrences in the per-question query), sessions grading queue (count `$4`, rows `$6` after LIMIT/OFFSET `$4/$5`; all args used).
- Dashboard HAVING `($N IS NULL OR COUNT(DISTINCT ra.user_id) > 0)` hides exams with no assigned user in scope for restricted callers and is a no-op for others. The `AND @SCOPE@` in the LEFT JOIN ON clause correctly scopes assigned/completed/passed counts.
- 404 vs 403: record/progress run GetUserInfo (404) before authorizeUser (403), which matches GET /users/{id}. Session middleware passes unknown sessions to the handler (404), returns 403 only when the row exists and is out of scope, 500 on lookup error, and fails closed with a nil store for department_admin. Malformed ids are already 404'd by RequireUUIDPathParams.
- Cache: `scoped` skips both GetInsightCache read and UpsertInsightCache write, even with forceRefresh=false, so no cross-scope leak either way. The scope is read from ctx inside the repository, so it cannot be forgotten by a caller.
- All routes in router.go audited: the session-keyed admin routes (result, grading detail/grade, certificate) are wired. loyalty-summary is already department-checked in the ai service (documented in the issue report). `/admin/users/{userId}/remind` is a stub with no user data.
- Responses use the {data,error} envelope; errors are wrapped; handlers stay thin (they map ErrForbidden only).

## Findings

1. Low - dead code: `deptscope.RequireUserInScope` (middleware.go:14-20) and `Store.UserInScope`'s middleware path are not wired anywhere (only `RequireSessionInScope` is used in router.go). Scoping for users is done in the reports service. Fix: remove it or use it, and keep tests only for what ships. (The `selfAllowed` parameter exists only for this unused path.)

2. Low - inconsistent unknown-user behaviour: `StreamUserRecordCSV` (reports/service.go:449-452) calls authorizeUser without a prior GetUserInfo, so for a department_admin a nonexistent user id returns 403, whereas `/record` and `/progress` return 404. Other roles still get a header-only 200 CSV (unchanged). Fix: call GetUserInfo first for parity, or accept and document it.

3. Low/Medium - cost/abuse: for department_admin, `GetInsights` (ai/service.go:153-215) now makes an Anthropic call on every request because the cache is bypassed. Confirm the route has the AI rate limit/usage cap; consider a per-(exam, department) cache key instead of bypass.

4. Low - test quality: the scope tests use a fake `database/sql` driver and assert SQL text and bound args (`strings.Contains`, arg order), so they do not execute the recursive predicate, the HAVING, or $N inference against Postgres. The Playwright e2e spec (frontend/e2e/dept-admin-scoping.spec.ts) is the only behavioural coverage. Suggested fix: add one DB-backed test (or a documented e2e run) covering a descendant department, a sibling department, a NULL-department admin, and the dashboard HAVING.

5. Low - style: sessions `baseWhere` was changed from a `const` to a closure that concatenates a Predicate (sessions/repository.go ~1231). The concatenation inputs are fixed strings ("$4"/"$6"), so no injection risk, but a comment noting the param position coupling (scopeParam must equal the arg index) would help; `withScope` is duplicated across reports, ai and deptscope (3 copies). Consider exporting `deptscope.WithScope(q, col, param)`.

## Conclusion
No authorization bypass found among the audited endpoints or SQL paths. Fail-closed behaviour is verified in code for a missing department, a nil store and lookup errors. Findings are advisory.

## Addendum 2026-10-09 (BA decision on #165, revision after PASS)

Out-of-scope ids now return 404 instead of 403, with the same code/message/body as the real not-found of each endpoint (`USER_NOT_FOUND` for record, progress and record CSV; `SESSION_NOT_FOUND` / "Session not found." for session result, certificate, grading detail and grade answer). The record CSV export now checks existence first, so unknown ids 404 for every role (findings 2 resolved). The unused `RequireUserInScope` was removed (finding 1 resolved); `ErrForbidden` no longer exists in `reports`. The statements above about 403 refer to the previous revision. The role wording is `examiner` (no `hr_admin`). Tests assert out-of-scope and unknown-id responses are byte-identical (reports handlers, deptscope middleware). Verdict unchanged: PASS.
