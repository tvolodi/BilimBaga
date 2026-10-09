---
id: ISS-141
title: Malformed UUID id params (query and path) reach Postgres and return 500
status: resolved
severity: medium
layer: backend
module: users
tags: [invalid input syntax for type uuid, department_id, actor_id, category_id, tag_ids, RequireUUIDPathParams]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-133]
regression_test: backend/internal/api/uuid_test.go
---

## Symptom
GET /api/v1/users?department_id=foo (and similar id filters/path ids elsewhere) returned 500 INTERNAL_ERROR because the raw string reached a UUID column ("invalid input syntax for type uuid"). Follow-up from ISS-133 review.

## Root Cause
Only ListUsers role_id was validated (ISS-133). Other handlers passed id-like query params and chi path params straight to repositories.

## Fix Applied
- New shared helpers in `internal/api/uuid.go`: `IsUUID`, `UUIDQuery`, `ValidateUUIDList`, `RequireUUIDPathParams`.
- Query params (422 VALIDATION_ERROR): users `department_id`, `role_id`; audit `actor_id` (list + export); questions list `category_id`, `tag_ids`, `tag_id`; questions export `ids`, `category_id`, `tag_ids`; sessions grading queue `exam_id` (found in review).
- Path params (404 NOT_FOUND, resource cannot exist): middleware mounted in the authenticated router group for `id, userId, sessionId, examId, questionId, tagId, sectionId, ruleId, assignmentId`. `{locale}` and public `/verify/{code}` (separate group; returns valid:false) are untouched.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/api/uuid.go | new helpers + middleware |
| backend/internal/users/handler.go | use UUIDQuery for department_id/role_id |
| backend/internal/audit/handler.go | parseFilters validates actor_id |
| backend/internal/questions/handler.go, import_export_handler.go | validate id filters |
| backend/internal/router/router.go | mount path-param middleware |
| tests | api/uuid_test.go, users, audit, questions handler tests |

## Regression Test
`internal/api/uuid_test.go` (helpers + middleware inside chi.Group), plus handler tests for users, audit, questions list/export (malformed -> 422, valid passes through).

## Resolution Results
- Tests: all packages pass (`go test -p 2 ./...`)
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

## Cycle 2 addendum

Added router-level tests (mount proof via real `router.New` with nil handlers + signed JWT) and a `chi.Walk` drift guard. Name list extracted to `api.UUIDPathParamNames`, shared by the router and the test. Non-UUID params found: `{locale}`, `{code}` (documented exceptions). No behavior change.
