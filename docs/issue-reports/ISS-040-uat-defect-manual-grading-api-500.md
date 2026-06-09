---
id: ISS-040
title: "UAT Defect: Manual Grading — POST grading answer endpoint returns HTTP 500"
status: resolved
severity: high
layer: backend
module: grading
tags: [uat, manual-grading, postgresql, parameter-type-inference]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/sessions/grading_test.go
---

## Symptom
UAT Scenario: `Manual Grading — Submit Grades (Happy Path)`, Steps 15–18; also `Scenario 4 (all steps)` and `Scenario 1 Step 9`
Actor: Examiner
Action: Click "Submit All Grades" — frontend fires `POST /api/v1/admin/grading/{sessionId}/answers/{questionId}` for each question
Expected: HTTP 200 with graded answer response; on completion of all questions the session is finalized and examiner is navigated back to queue with success toast
Actual: HTTP 500 `{"code":"ERR_INTERNAL","message":"failed to grade answer"}` returned for every grading submission. No error logged in Go service layer. Session remains in pending state. All downstream scenarios (Scenario 4) are fully blocked. Scenario 1 Step 9 browser submit also produced the same 500 (same root cause suspected).
Screenshot: none

## Root Cause

In `postgresRepository.GradeAnswer` (step 6 — audit log), the INSERT query reused the same positional parameter `$2` (graderID) in two conflicting type contexts:

1. As the value for column `actor_id UUID` → PostgreSQL infers type **UUID**
2. Inside `jsonb_build_object('grader_id', $2, ...)` → PostgreSQL sees a `variadic "any"` context with no type constraint

PostgreSQL's type inference algorithm cannot reconcile a single parameter appearing in both a UUID-column context and an untyped `variadic any` context. The result is the error:

```
pq: could not determine data type of parameter $2
```

This error occurs before the transaction commits, so all grading submissions fail unconditionally. The handler had no `slog.Error` call before the 500 response, so no error appeared in application logs — making the bug invisible until direct inspection.

The same ambiguity applied to `$5` (sessionID) and `$6` (scorePct) which appeared only inside `jsonb_build_object` without any typed column context.

## Fix Applied

**File**: `backend/internal/sessions/repository.go`

1. **Broke parameter reuse**: `$2` (graderID) was removed from inside `jsonb_build_object` and replaced with a new 7th parameter `$7` (same value). This isolates `$2` to the `actor_id UUID` column context only, allowing PostgreSQL to infer its type unambiguously.

2. **Added explicit casts**: `$5::text`, `$6::numeric`, `$7::text` inside `jsonb_build_object` — these explicit casts tell PostgreSQL the expected types for parameters that appear only in a `variadic any` context, eliminating the type-inference failure.

3. **Added error logging**: `slog.Error("GradeAnswer failed", "error", err, ...)` added to the handler's default 500 case — this was absent previously, making production diagnosis impossible.

Before:
```sql
VALUES ($1, $2, 'answer.grade', 'session_answer', $3, $4,
        jsonb_build_object('grader_id', $2, 'session_id', $5, 'score_pct', $6))
-- ExecContext params: tenantID, graderID, questionID, actorIP, sessionID, scorePct
```

After:
```sql
VALUES ($1, $2, 'answer.grade', 'session_answer', $3, $4,
        jsonb_build_object('grader_id', $7::text, 'session_id', $5::text, 'score_pct', $6::numeric))
-- ExecContext params: tenantID, graderID, questionID, actorIP, sessionID, scorePct, graderID
```

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/sessions/repository.go` | Fixed `auditQ` parameter binding in `GradeAnswer`: separate `$7` for JSONB grader_id, added `::text`/`::numeric` casts |
| `backend/internal/sessions/handler.go` | Added `slog.Error` logging in the default 500 case of `HandleGradeAnswer` |

## Regression Test

Existing tests in `backend/internal/sessions/grading_test.go`, `service_test.go`, and `handler_test.go` cover the happy-path and error-path unit tests. These all pass (mock-based, so the PostgreSQL type-inference issue does not surface at unit-test level).

The fix was validated by live API testing against the actual PostgreSQL DB:
- Grading first question (partial): HTTP 200 `{"all_graded": false, "session_status": "grading_pending"}`
- Grading second question (final): HTTP 200 `{"all_graded": true, "session_status": "submitted", "final_score_pct": 82.5, "passed": true}`

A full DB-integration regression test would require a PostgreSQL fixture; no new unit test added since the bug is in SQL parameter binding that only manifests against a real PostgreSQL server.

## Resolution Results
- Tests: 26 packages, all PASS
- Migration applied: no (no schema changes required)
- Build clean: yes
- Live API verification: HTTP 200 for both partial and final grading

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-06-09 | UAT Manual Grading UAT run | Root cause identified via live API testing; fixed parameter binding and added error logging |
