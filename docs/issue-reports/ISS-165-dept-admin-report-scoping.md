---
id: ISS-165
title: department_admin not scoped to its department on reports/analytics/dashboard endpoints
status: resolved
severity: high
layer: backend
module: reports
tags: [department_admin, authorization, idor, reports, dashboard, GetUserRecord, GetExamAnalytics, StreamExamResultSessions, deptscope]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-163]
regression_test: backend/internal/reports/scope_test.go
---

## Symptom

GitHub issue #165. A `department_admin` of department A could read data about every department:

1. `GET /admin/users/{id}/record` - any user's record (200)
2. `GET /admin/users/{id}/record/export` - CSV of any user (200)
3. `GET /admin/exams/{id}/analytics` - all attempts of all departments
4. `GET /admin/dashboard` - same data as `super_admin`

`GET /users` and `GET /users/{id}` were already scoped (exact department; 403 otherwise).

## Root Cause

The reports endpoints are gated only by `rbac.RequirePermission(reports:read)`. `department_admin` holds `reports:read` (migration 005) and `reports.Handler`/`Service`/`Repository` never looked at the caller's role or department; the department-scope rule lived only in `users.Service`. Same class of gap in the other admin read endpoints keyed by session id (admin session result, grading, certificate) and in the grading queue and AI exam insights (which also cached an all-department result per exam).

## Fix Applied

Decisions (recorded on the issue):

- **Who is scoped**: only `department_admin` (RBAC stores no scope; `super_admin`, `hr_admin`, `examiner` unchanged).
- **Scope** = the caller's department **and its descendants** (recursive over `departments.parent_id`; the dashboard SQL already used the same subtree semantics for assignments). Note: `GET /users*` keeps its pre-existing exact-department rule (out of scope here; a descendant is visible in reports but not in the user list - a follow-up could widen it).
- **Per-employee record / progress / CSV**: 404 if the user does not exist, else 403 `FORBIDDEN` / "insufficient permissions" (identical to `GET /users/{id}`) when the user is outside the subtree. A caller may always read its own record.
- **Exam analytics / results CSV**: aggregates computed over participants in the subtree only (attempts, pass rate, distribution, per-question stats, answer distribution, CSV rows and CSV question columns).
- **Dashboard** (and PDF export): every list and aggregate is computed over the subtree. The exam list shows only exams that have at least one assigned user in the subtree (an exam with `0/0/0` for the whole subtree is noise), with `assigned/completed/passed` counted over subtree users only. `super_admin` list is unchanged (still includes unassigned exams with zero counts, issue #38).
- **No department**: a `department_admin` without a department sees an empty scope (records of others 403, aggregates empty).
- **Mechanism**: new package `internal/deptscope`. The scope is derived from the authenticated principal in the request context (`ctxkeys` role + department), so it cannot be forgotten by a handler: `deptscope.FromContext(ctx).Arg()` is bound as a nullable uuid parameter and `deptscope.Predicate(col, "$N")` expresses "user in subtree or param IS NULL" in SQL. Reports service uses `repo.UserInScope` for single-user checks; session-keyed endpoints in other packages use `deptscope.RequireSessionInScope` middleware (403 when the session owner is outside the subtree; unknown/malformed ids fall through to the handler's own 404/400; fails closed with 500 if no store).
- No migration (uses `users.department_id`, `departments.parent_id`, `exam_sessions.user_id`, all existing).

## Endpoint audit

| Endpoint | Result |
|---|---|
| GET /admin/users/{id}/record | FIXED (service check, 403) |
| GET /admin/users/{id}/record/export | FIXED (service check before any output, 403) |
| GET /admin/users/{id}/progress | FIXED (same gap, same check) |
| GET /admin/exams/{id}/analytics | FIXED (aggregates over subtree) |
| GET /admin/exams/{id}/results/export | FIXED (`StreamExamResultSessions`, `GetExamQuestions` filtered) |
| GET /admin/dashboard | FIXED (completion, overdue, recent activity, avg by track) |
| GET /admin/dashboard/export (PDF) | FIXED (completion in range, top/bottom questions) |
| GET /admin/ai/insights/{examId} | FIXED (aggregates over subtree; department_admin neither reads nor writes the shared per-exam cache) |
| GET /admin/sessions/{id}/result | FIXED (middleware, 403) |
| GET /admin/sessions/{id}/certificate | FIXED (middleware, 403) |
| GET /admin/grading (queue) | FIXED (queue filtered to subtree) |
| GET /admin/grading/{sessionId} | FIXED (middleware, 403) |
| POST /admin/grading/{sessionId}/answers/{questionId} | FIXED (middleware, 403) - writes on another department's answers |
| GET /admin/ai/loyalty-summary/{sessionId} | scoped already (service: non-super_admin must be in the admin's exact department); stricter than subtree, left unchanged |
| POST /admin/users/{userId}/remind | not needed (stub returning `{data:null}`, no user data) |
| GET /users, GET /users/{id} | scoped already (exact department) - the model |
| PUT/DELETE/POST /users/*, import, reset-password, unlock | scoped already in users service (exact department) |
| GET /exams, /exams/{id}, rules/sections/eligible-counts | not needed (exam configuration, no employee data) |
| GET /exams/{id}/assignments | NOT FIXED - residual: returns `assignee_name` of directly assigned users and per-assignment stats org-wide. Exam configuration endpoint with name+count exposure only; recommended follow-up issue |
| GET /audit, /audit/export | not needed (`audit:read` not granted to department_admin, migration 005) |
| GET /departments | not needed (organisation structure, no employee data) |
| GET /questions*, categories, tags | not needed (content, no employee data) |
| /portal/* | not needed (self-service, principal's own data only) |

## Files Changed

| File | Change |
|------|--------|
| backend/internal/deptscope/{scope,store,middleware}.go | new: scope from ctx, SQL predicate, store, guards |
| backend/internal/reports/{repository,service,handler,model}.go | scoped SQL, `UserInScope`, `ErrForbidden`, 403 mapping |
| backend/internal/ai/{repository,service}.go | insights scoped, shared cache bypassed for department_admin |
| backend/internal/sessions/repository.go | grading queue scoped |
| backend/internal/router/router.go | `sessionScope` guard on 4 session-keyed routes |
| tests: reports/scope_test.go, deptscope/*_test.go, ai/scope_test.go, sessions/scope_test.go, router/router_test.go | see below |
| frontend/e2e/dept-admin-scoping.spec.ts | live e2e with two departments + child (not executed) |

## Regression Test

- `reports/scope_test.go`: service (own dept, other dept 403, target without department, caller without department, own record, super_admin/hr_admin/examiner/no principal unchanged, 404 beats 403, lookup error), handlers (403 shape, super_admin never probes), repository SQL via fake driver (subtree filter + bound argument for 12 scoped queries, NULL for other roles, zero-uuid for department-less admin, HAVING hides unrelated exams, `UserInScope`).
- `deptscope/deptscope_test.go`: scope derivation, descendant predicate, store SQL, middleware matrix.
- `ai/scope_test.go`, `sessions/scope_test.go`, router wiring test.
- `frontend/e2e/dept-admin-scoping.spec.ts`: A / A1 (descendant) / B employees, department_admin of A.

## Resolution Results

- Tests: `go test -p 1 ./...` all packages ok; `go vet ./...` clean; staticcheck 0 findings; schemaguard green.
- Migration applied: no
- Build clean: yes
- Live DB / e2e: not executed (no stack allowed in this run) - label `needs-live-db`.

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|-------------- |
