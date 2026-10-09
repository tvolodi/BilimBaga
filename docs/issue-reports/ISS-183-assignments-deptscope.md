---
id: ISS-183
title: department_admin sees organisation-wide assignee names and counts in GET /exams/{id}/assignments
status: resolved
severity: medium
layer: backend
module: exams
tags: [assignments, deptscope, ListAssignmentsWithStats, assignee_name, ISS-165, ISS-218]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-165, ISS-218]
regression_test: backend/internal/exams/assignments_scope_test.go
---

## Symptom
GET /exams/{id}/assignments returned every assignment of the exam with the
assignee name (user full name or department name) and per-assignment stats
(total_users / completed_count / passed_count) for the whole organisation, to any
caller holding exams:read, including department_admin, examiner and custom roles.

## Decision (dev1)
Scope assignee names and counts: YES. Every role except super_admin sees only
assignments inside its department subtree (internal/deptscope, same rule as
reports). super_admin sees all. Exam configuration (non-assignee fields, GET
/exams/{id}) stays visible; only the assignment list is scoped.

Per-assignment rules for a restricted caller (subtree S of the caller's dept):
- assignee_type=user: listed only if the user is in S (else row omitted, name hidden).
- assignee_type=department: listed only if that department is in S. An ancestor or
  sibling department assignment is omitted (its name would leak); counts of an
  in-scope department are unchanged (its users are all in S).
- assignee_type=all: listed (no name, it carries no identity) but total_users,
  completed_count, passed_count count only users in S.
- No department / unknown role: S is empty -> only the name-less "all" row, with zero counts.

## Root Cause
`postgresRepository.ListAssignmentsWithStats` (and the `listAssignmentsNoSessions`
fallback) had no scope predicate; ISS-165/218 scoped reports only.

## Fix Applied
- `ListAssignmentsWithStats(ctx, examID, sc deptscope.Scope)`; the service derives
  the scope with `deptscope.FromContext(ctx)` so a caller cannot forget it.
- SQL: recursive CTE `sc_t` (subtree of `$2`), `scopeAssignmentsQuery` expands
  `@SCOPE_ASSIGNEE@` / `@SCOPE_USERS@` / `@SCOPE_SESSIONS@` using
  `deptscope.Predicate`. `$2` NULL = unrestricted. Both the main query and the
  pre-FR-BB35 fallback are scoped. Handler unchanged (thin).
- Out-of-scope assignees are filtered, not 404: the endpoint is a list under an
  exam that remains visible.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/exams/repository.go | scope param, CTE, scopeAssignmentsQuery |
| backend/internal/exams/service.go | pass deptscope.FromContext(ctx) |
| backend/internal/exams/service_test.go | mockRepo signature + lastScope |
| backend/internal/exams/assignments_scope_test.go | new tests |

## Regression Test
assignments_scope_test.go: scope passed per role (super_admin unrestricted; dept admin,
examiner, custom role, no-dept, empty role restricted), scoped rows only returned,
exam still visible, SQL marker expansion guard, handler context propagation.
No real-Postgres test exists (no DB in the swarm env): the SQL itself is verified
only by the string guard; PR labelled needs-live-db.

## Sibling endpoints that leak the same data (not fixed here)
- Ancestor-department assignments are hidden from a child-dept admin by design; not a leak.
- `internal/email/repository.go` (assignment notification resolution) and
  `internal/portal` / `internal/sessions` read exam_assignments for the caller's own
  eligibility only: no cross-department exposure found.
- Pre-existing quirk, unchanged: department stats use a 2-level (not fully
  recursive) user count and direct-member session counts.

## Resolution Results
- Tests: go test -p 2 ./... all packages ok (incl. schemaguard), go vet clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
