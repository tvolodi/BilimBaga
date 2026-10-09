# Conformance Review: PR #77, PR #83, PR #86

- Date: 2026-10-09
- Reviewer: Business Analyst (static code review; no live stack, no UAT executed, tests not re-run)
- Base: origin/main at 90fee47 (worktree `ba`, branch swarm/ba-fr510-scenario-conformance)
- Scope: PR #77 (6c002d1, issues #38 and #75), PR #83 (8ce7e63, issue #82), PR #86 (7982bf9, issue #44)
- Scenario compared: `docs/uat-scenarios/dashboard-completion-rate-20261009.md`; also `docs/uat-scenarios/cert-public-verification-20261009.md`
- Severity: p1 only for security, data exposure or a broken core flow; p2 behaviour/spec defect; p3 minor or documentation.

## 1. PR #77 - reports LEFT JOIN (FR-BB51 AC-2, #38) and tenant_id removal (#75)

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 1.1 | FR-BB51 AC-2: every active exam listed, including those without assignments | PASS | `backend/internal/reports/repository.go:183` `LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id`; `WHERE e.status = 'active'` kept (line ~186) |
| 1.2 | Counts stay distinct-user and zero for no assignment | PASS | `COUNT(DISTINCT ... ra.user_id)` ignores NULL, so 0; unchanged formulas at `repository.go:~178-181`; test `repository_completion_test.go:24` (`ExamWithoutAssignmentsReturnedWithZeros`) |
| 1.3 | Same fix for the date-ranged dashboard/PDF query | PASS | `repository.go:883` LEFT JOIN; date filter stays in the `LEFT JOIN exam_sessions` ON clause so unassigned exams are not dropped; test `repository_completion_test.go:47` |
| 1.4 | Dashboard handler returns 200 with zero-count exam | PASS | `repository_completion_test.go:90` `TestGetDashboard_200_UnassignedExamZeroCounts`; empty-list case `:68` |
| 1.5 | #75: nonexistent `tenant_id` predicates removed from CSV export, user record CSV, completion-range, top/bottom questions | PASS | `repository.go:748-770`, `826-851`, `883-940` (no `tenant_id` in those queries); parameter indexes renumbered consistently ($1,$2 for dates) |
| 1.6 | #75: swallowed errors now logged | PASS | `reports/handler.go` slog.Error added at CSV exports (lines ~153-176) and PDF build/generate (~221-231); response still 500 `INTERNAL_ERROR` for PDF |
| 1.7 | Schema guard for reports | PASS | `reports/schema_guard_test.go:16` `TestRepositorySQL_NoTenantIDColumn`; fake DB `fakedb_test.go` |
| 1.8 | e2e asserts exports return 200 | PASS | `frontend/e2e/downloads-bearer.spec.ts` (9 lines changed) |
| 1.9 | FR-BB51 AC-8 text "All queries are scoped by tenant_id" | GAP p3 (spec drift) | Single-tenant schema has no tenant column on `exams`/`exam_sessions`; code comment `repository.go:81-85` records that tenantID params are unused. AC-8 is now vacuous and should be reworded or marked N/A in `docs/requirements/FR-BB51.Dashboard-metrics-API.md` |
| 1.10 | CSV export failure after headers are sent | GAP p3 | `handler.go` CSV paths can only log (headers already written); a client sees a truncated 200 file. Acceptable, but export scenarios cannot detect it via status; compare row counts instead |
| 1.11 | FR-BB51 AC-2 text says "every active exam in the tenant" while the export-range query (1.3) also drops exams only through `status = 'active'` | PASS | consistent |
| 1.12 | FR-BB51 requirement checkboxes | GAP p3 | AC checkboxes in `FR-BB51...md` remain `[ ]` (lines 16-25) though code exists; documentation hygiene |

Verdict PR #77: PASS (code). No p1/p2 gaps.

## 2. PR #83 - ai SQL fix and repo-wide schema guard (#82)

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 2.1 | `GetExamInsightData` no longer selects `exams.tenant_id` and no longer compares it | PASS | `backend/internal/ai/repository.go:~172-192`; struct `examInsightRow` has only title and passing score; interface comment line 37 |
| 2.2 | Per-question stats use `session_questions.sort_order` instead of nonexistent `order_num` | PASS | `ai/repository.go:215` `MIN(sq.sort_order) AS order_num`; the `ORDER BY` for session questions also uses `sort_order` (`:354`) |
| 2.3 | `GetSessionCategoryTrack` no longer uses nonexistent `exams.category_id`; resolves track from `exam_question_rules` -> `categories.track` | PASS | `ai/repository.go:~295-306` LATERAL on `exam_question_rules` ordered by `sort_order` LIMIT 1 |
| 2.4 | Behaviour semantics of 2.3 | GAP p3 (verify) | An exam with rules in several categories now takes the first rule's track; the old (broken) join assumed one category per exam. FR-BB51 AC-6 (track via `session_question_scores -> questions -> categories`) and FR-BB75 may expect a per-question or majority track. Confirm with product; current choice is deterministic |
| 2.5 | Tenant isolation lost for insights | GAP p3 | Exam lookup no longer checks tenant (`ai/repository.go:176-190`). Safe because the schema is single-tenant and the tenant middleware injects the constant `"public"`; if multi-tenancy is ever added, `exams` needs a column and these queries must be revisited. Recorded in `docs/issue-reports/ISS-82-tenant-id-audit.md` |
| 2.6 | Repo-wide schema guard | PASS | `backend/internal/schemaguard/schema_guard_test.go:289` replays migrations and checks every SQL literal under `internal/`; negative test `:333`; allow-list test `:352` (only `audit_log` has `tenant_id`; migration `007_audit_log.up.sql:7`) |
| 2.7 | Guard limits | GAP p3 | Static, regex-based: dynamically assembled SQL, CTE-aliased columns and `SELECT *` are not validated, so it reduces but does not eliminate runtime-only SQL errors. Remaining `tenant_id` uses in non-test code are `audit` (valid column), `sessions/repository.go:872,1444` and `sessions/autojob.go:192` (all `INSERT INTO audit_log`, valid) and router/tenant middleware |
| 2.8 | Unit tests for ai repository | PASS | `ai/repository_test.go:14,30,60,76` (not found; ignores tenant and uses sort_order; no `exams.category_id`; found path) with `ai/fakedb_test.go` |

Verdict PR #83: PASS. No p1/p2 gaps. It is mostly outside the scope of any UAT scenario except the AI insight scenario `docs/uat-scenarios/ai-content-authoring-20260609.md`, which should now run (before this fix the insight endpoint failed at runtime).

## 3. PR #86 - certificates verify 500 and PUBLIC_APP_URL warning (#44; FR-BB48 AC-5, AC-1)

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 3.1 | FR-BB48 AC-5: internal error returns 5xx so the SPA shows "temporarily unavailable" not "invalid" | PASS | `backend/internal/certificates/handler.go:~62-71` returns 500 `INTERNAL_ERROR` ("Verification is temporarily unavailable."), logs the underlying error via `slog.Error`; SPA maps `!res.ok` to the unavailable state (see `FR-BB48-FR-BB64-conformance-20261009.md` AC-5) |
| 3.2 | FR-BB43 AC-7: unknown code is 200 `valid:false`, never 404/500 | PASS (for unknown UUID) | Repository maps `sql.ErrNoRows` to `ErrNotFound` (`certificates/repository.go:~144`); service returns `Valid:false` (`service.go:104-108`); handler also keeps `errors.Is(err, ErrNotFound)` -> 200 (`handler.go:64`) |
| 3.3 | No internal details leaked | PASS | Test `certificates/handler_test.go:255` asserts no `secret-host` or `pq:` in the body and that the log holds the cause |
| 3.4 | FR-BB48 AC-4 / FR-BB43 AC-7: malformed code (not a UUID) must still be `valid:false` | **GAP p2 (regression risk)** | `certificates.verification_code` is a `UUID` column (`migrations/*certificates*.up.sql:6`). The handler passes the raw URL param straight to `WHERE verification_code = $1` (`repository.go:~134-141`) with no UUID validation anywhere in `handler.go`, `service.go` or `repository.go`. Postgres returns `invalid input syntax for type uuid` for `not-a-uuid`; that is not `sql.ErrNoRows`, so it is wrapped as a generic error. Before PR #86 the handler swallowed any error as 200 `valid:false`; after it, the result is **500** and the SPA shows "temporarily unavailable" instead of "invalid" (breaks `cert-public-verification-20261009.md` S3 step 5, and AC-4 "malformed UUID"). Also every scanner typo now logs an error. Not p1: no data exposure and the main valid-certificate flow is unaffected. Fix: validate the code with `uuid.Parse` in the service (or handler) and return `valid:false` before the query. The unit test only mocks the service, so this path is untested (`handler_test.go:255-300` use `some-code`/`nope` with mocks) |
| 3.5 | FR-BB48 AC-1: PUBLIC_APP_URL default warning | PASS | `config/config.go:75,80` sets `PublicAppURLDefaulted = os.Getenv("PUBLIC_APP_URL") == ""`; `cmd/api/main.go:75-78` warns once at startup; `config_test.go` covers it. Note: set-but-whitespace value would not count as defaulted (p3) |
| 3.6 | `.env.example` / compose carry PUBLIC_APP_URL | not verified here | outside the diff of #86; covered by FR-BB48 AC-1 review |
| 3.7 | Rate limit on the public verify route | n/a | Public route is under the global limiter (300/min), see `router/router.go:~62`; enumeration is bounded by UUID entropy |

Verdict PR #86: PASS for AC-5 and AC-1 warning; one p2 gap (3.4) not covered by tests.

## 4. Gap summary

| ID | PR | Gap | Severity |
|----|----|-----|----------|
| G1 | #86 | Malformed (non-UUID) verification code yields 500 instead of 200 `valid:false`; SPA shows "unavailable" | p2 |
| G2 | #77 | FR-BB51 AC-8 (tenant scoping) obsolete vs single-tenant schema; AC checkboxes unticked | p3 |
| G3 | #77 | CSV export failures after headers are sent are log-only | p3 |
| G4 | #83 | Track resolution for exams with multiple rule categories picks first rule only | p3 |
| G5 | #83 | Tenant isolation removed from AI insights (acceptable single-tenant); schema guard is regex-level | p3 |
| G6 | #86 | `PublicAppURLDefaulted` ignores whitespace-only values | p3 |

**p1 gaps: none.** No security, data-exposure or core-flow break found.

## 5. What the dashboard-completion-rate scenario should now additionally verify

The existing scenario covers #38 well (S1-S4). With the merged code it should add:

1. S5 Export consistency becomes mandatory, not optional: `GET /api/v1/admin/dashboard/export` must return 200 with a PDF (previously a swallowed 500 through `es.tenant_id`/`e.tenant_id`, #75); the text should list `UAT-Dash-Unassigned` with 0 (range query now LEFT JOINs, `repository.go:883`). Assert `Content-Type: application/pdf`, `%PDF` magic and non-trivial size.
2. Add CSV scenarios: `GET` exam results CSV and user-record CSV as admin must return 200 with the dynamic header and rows for submitted/grading_pending sessions only (`repository.go:748-851`); for an exam with zero sessions, header-only file, still 200.
3. Date-range variants of the dashboard PDF (`from`/`to`): an unassigned active exam still appears with zeros (date filter sits in the ON clause); sessions outside the range are not counted; top/bottom questions section does not error (#75 `GetTopBottomQuestions`).
4. Regression guard for data: dashboard response for a tenant header other than default must behave the same (tenantID is ignored); assert no 500 when the tenant middleware injects `public`.
5. Examiner and dept-admin access to the export endpoint (reports:read) and 403 for employee on all export endpoints.
6. Server log check after each export (`docker compose logs api`): no `reports: ... failed` lines (slog.Error added by #77), so a silent failure is observable.
7. AI insight smoke (separate scenario `ai-content-authoring-20260609.md`): `GetExamInsightData` and the category-track lookup run without SQL errors on the live schema (#83); include an exam with rules in two categories to observe G4.
8. Public verify (add to `cert-public-verification-20261009.md` S3): confirm G1 live. Malformed code `not-a-uuid` expected 200 `valid:false`; the current code is predicted to return 500. If it does, mark DEFECT and route to Issue Resolution. Also add: temporarily stop the database (or API dependency) and assert the SPA shows "Verification temporarily unavailable" with Retry (FR-BB48 AC-5, PR #86), then recover and retry succeeds. Check the startup log for the `PUBLIC_APP_URL is not set` warning when the variable is unset and its absence when set.

## 6. Recommendations

1. Create a p2 issue for G1 (UUID pre-validation for `/api/v1/verify/{code}`) plus a handler or service test using a real malformed string against the repository fake.
2. Update `FR-BB51` AC-8 wording and tick implemented AC boxes; no code change needed.
3. Execute the extended scenarios live; none of PR #77, #83 or #86 is `uat-verified` yet (no files in `docs/uat-reports/` for them).
