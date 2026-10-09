---
id: ISS-158
title: GET /audit?user_id=bad returns 200 instead of 422
status: resolved
severity: low
layer: backend
module: audit
tags: [user_id, actor_id, UUIDQuery, VALIDATION_ERROR]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-141]
regression_test: backend/internal/audit/handler_test.go
---

## Symptom
`GET /api/v1/audit?user_id=<non-uuid>` returned 200 (UAT smoke of #141).

## Root Cause
The audit handler only read `actor_id`; `user_id` was an unknown param, silently ignored (filter dropped, 200 with unfiltered list). Clients using the natural name `user_id` got neither filtering nor validation.

## Fix Applied
`parseFilters` (shared by List and Export) now accepts `user_id` as an alias of `actor_id`, validated through `api.UUIDQuery` (malformed -> 422 VALIDATION_ERROR). `actor_id` wins if both are given.

Sweep of all handlers for query params: remaining UUID-bearing query params (`actor_id`, `category_id`, `tag_ids`/`tag_id`, `ids`, `exam_id`, `department_id`, `role_id`) were already validated by #141. No `session_id`, `created_by`, `assignee_id`, `employee_id` query params exist. Others (`actor`, `action`, `entity_type`, `locale`, `type`, `status`, dates, ints, `dir`) do not reach UUID columns. Nothing left unvalidated.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/audit/handler.go | user_id alias, validated |
| backend/internal/audit/handler_test.go | 422 (list+export) and valid-passthrough tests |

## Regression Test
`TestHandler_UserIDNonUUID_Returns422`, `TestHandler_UserIDValid_PassedAsActorFilter`.

## Resolution Results
- Tests: `go test -p 1 ./...` all packages ok
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
