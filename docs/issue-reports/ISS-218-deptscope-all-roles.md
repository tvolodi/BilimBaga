---
id: ISS-218
title: Department scoping keyed on role name department_admin; custom roles and examiner see org-wide data
status: resolved
severity: high
layer: backend
module: reports
tags: [deptscope, FromContext, RoleDepartmentAdmin, custom role, reports:read, grading, FR-BB117 D-2]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-165]
regression_test: backend/internal/deptscope/deptscope_test.go
---

## Symptom
A custom role holding reports:read or grading:* (and the built-in examiner) got unrestricted org-wide
reports, analytics, AI insights, employee records and grading data (GitHub #218, conformance finding G1).

## Root Cause
`deptscope.FromContext` returned an unrestricted Scope for every role except the literal name
`department_admin`, i.e. scoping was an allow-list of one role name instead of a deny-list of one.

## Fix Applied
FR-BB117 D-2: only `super_admin` is unrestricted. Every other role, including an empty/unknown role
or a context with no principal, is restricted to its department subtree; no department => the existing
all-zero `noDepartment` sentinel (empty set).

Examiner grading carve-out (explicit and narrow): data model has no examiner-assignment table
(`exam_assignments` assigns exams to employees/departments, `assigned_by` is the assigner). The only
ownership notion is `exams.created_by`. So `Scope.ExamOwnerID` is set ONLY for the built-in `examiner`
role (= its user id) and is honoured ONLY by the manual-grading endpoints:
- queue: `deptscope.GradingPredicate` = `e.created_by = $owner OR subtree predicate` (sessions/repository.go)
- detail / grade answer: new `RequireGradingSessionInScope` + `Store.GradingSessionInScope` (router `gradingScope`)
Reports, analytics, AI insights/loyalty, user record/progress/CSV, `/admin/sessions/{id}/result` and
certificates keep the plain subtree scope. Custom grading roles get no carve-out (OPEN DECISION).

Consumers audited: reports (repository 16 call sites + service.authorizeUser), ai (repository insight
data, service insight cache bypass; loyalty uses its own non-super_admin department check), sessions
(grading queue), router (session middleware). Audit log does not use deptscope (org-wide, audit:read is
super_admin-only by seed). exams/users packages use their own role logic (not deptscope), untouched.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/deptscope/scope.go | deny-list FromContext, ExamOwnerID/OwnerArg, GradingPredicate |
| backend/internal/deptscope/store.go, middleware.go | GradingSessionInScope, RequireGradingSessionInScope |
| backend/internal/sessions/repository.go | grading queue uses GradingPredicate |
| backend/internal/router/router.go | grading detail/grade use gradingScope |
| tests in deptscope, reports, sessions, ai | see below |

## Regression Test
deptscope_test (super_admin unrestricted; dept_admin/examiner/custom/empty restricted; no dept => sentinel;
owner only for examiner; store + middleware carve-out), reports/scope_test (custom role on service,
handlers, every repository query), sessions/scope_test (grading queue owner/scope args per role),
ai/scope_test (insights data, cache bypass, loyalty forbidden for custom/examiner/empty role).
Existing ai GetInsights cache tests switched to a super_admin context (no-principal now fails closed).

## Resolution Results
- Tests: deptscope, reports, sessions, ai, router targeted packages pass; go vet clean on touched packages
- Not run (memory hold): whole-module build/vet/test, docker, e2e, live DB
- Migration applied: no
- Build clean: yes (touched packages)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

## Scope-keyed AI cache (Supervisor decision)
Supervisor decisions applied in cycle 2:
1. Examiner ownership = `exams.created_by` is accepted for now (note for the FR-BB117 doc area; the BA requirement file was not edited).
2. Custom roles with `grading:*` get NO carve-out (subtree-only); confirmed, test comment in `ai/scope_cache_test.go` and existing sessions/deptscope tests.
3. `exams/service.go` assignment listing (`assignsOrgWide`) left to issue #183, untouched.
4. AI insights are no longer an uncached paid call for every non-super_admin request.

Cache key design
- Scope key = `deptscope.ScopeKey(scope, ids)`: unrestricted (super_admin) = constant `all`; restricted with empty set (no department / unknown department) = constant `none`; otherwise hex SHA-256 of the sorted, de-duplicated subtree department ids joined with `,` (64 hex chars, so it can never equal `all`/`none`). `deptscope.SubtreeIDs` (recursive CTE, `SubtreeSQL`) supplies the ids; `ai.Repository.GetScopeDepartmentIDs` exposes it.
- Cache key = struct `{examID, scopeKey}` (not a joined string, so no delimiter collision on examID). Equal department-id sets share an entry regardless of role or subtree root; different sets and different exams never do.
- Schema: `ai_insight_cache.exam_id` is a `UUID PRIMARY KEY` with no scope column, so a composite key cannot be folded in without a migration. No migration was added (lock not held). Fallback implemented: super_admin (`all`) keeps using the existing DB table unchanged; every restricted caller uses a bounded in-process cache (`ai/insight_cache.go`, 512 entries, 24 h TTL = `insightCacheTTL`, expired-then-oldest eviction, copies on read/write). It is per API process: not shared across replicas and lost on restart, worst case an extra paid call, never a cross-scope read. A durable version needs a follow-up migration (`scope_key TEXT NOT NULL DEFAULT 'all'`, PK `(exam_id, scope_key)`).
- Isolation: scoped callers never read or write the DB row; super_admin never reads the in-memory map. `?refresh=true` bypasses the read and overwrites the entry. If the scope lookup fails the call degrades to uncached generation (data query is still scoped in SQL).
- Rate limit: unchanged. Note `GetInsights` never had a per-user rate limit (only `GenerateQuestions` calls `checkRateLimit`); usage logging per paid call is unchanged and cache hits log nothing. Flagged for the Supervisor rather than silently adding a limit.
- SQL: new `deptscope.SubtreeSQL` (read-only, live-DB label `needs-live-db` kept). No existing query text changed.
- Tests: `ai/scope_cache_test.go`, `deptscope/scopekey_test.go` (same scope hit/no second paid call, different scopes and exams separate, super_admin vs scoped never share both ways, no-department empty-set key, refresh, lookup failure, TTL/bound/eviction, usage logging, GenerateQuestions rate limit). Run: go vet and go test on `./internal/ai/...` and `./internal/deptscope/...` pass. Not run: whole-module build/test, docker, e2e, live DB.
