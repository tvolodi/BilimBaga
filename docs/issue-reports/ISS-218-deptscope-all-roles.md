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
