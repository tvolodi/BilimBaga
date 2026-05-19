---
id: ISS-012
title: Question status transition returns 400 — frontend sends wrong JSON key
status: resolved
severity: high
layer: frontend
module: questions
tags: [400, status-transition, json-key-mismatch, useTransitionStatus, useUpdateQuestionStatus, target_status]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: []
regression_test: backend/internal/questions/handler_test.go
---

## Symptom
Clicking "Отправить на проверку" (Send for review) on the question edit page sends:
```
POST /api/v1/questions/{id}/status  →  400 Bad Request
```
Toast shows "Не удалось изменить статус." The console confirms the 400.

## Root Cause
Both `useTransitionStatus` (used by QuestionBankPage) and `useUpdateQuestionStatus` (used by
QuestionEditorPage) serialize the request body as `JSON.stringify({ target_status })`, producing:

```json
{"target_status":"review"}
```

The backend `statusTransitionReq` struct expects the field `"status"` (`json:"status"`). Go's
`encoding/json` silently ignores unknown keys, so `req.Status` is the zero value `""` after
decoding. The handler then immediately returns `400 Bad Request` with "status is required".

```go
type statusTransitionReq struct {
    Status string `json:"status"`   // ← expects "status", not "target_status"
}
if req.Status == "" {
    api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "status is required")
    return
}
```

## Fix Applied
Changed both mutations in `frontend/src/api/questions.ts` to send the correct key:

```diff
- body: JSON.stringify({ target_status }),
+ body: JSON.stringify({ status: target_status }),
```

Applied to `useTransitionStatus` (line ~261) and `useUpdateQuestionStatus` (line ~382).

Added a regression handler test `TestQHandlerTransitionStatus_WrongFieldNameKey_Returns400` in
`backend/internal/questions/handler_test.go` that sends `{"target_status":"review"}` and asserts
the backend returns 400, locking in the API contract.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/api/questions.ts` | Fixed JSON body key in `useTransitionStatus` and `useUpdateQuestionStatus` |
| `backend/internal/questions/handler_test.go` | Added regression test for wrong JSON key |

## Regression Test
`backend/internal/questions/handler_test.go` → `TestQHandlerTransitionStatus_WrongFieldNameKey_Returns400`
Sends `{"target_status":"review"}` and expects 400.

## Resolution Results
- Tests: 68 passed, 0 failed (questions package); all 21 backend packages ok
- Migration applied: no
- Build clean: yes (tsc --noEmit clean)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Bug report: 400 on "Отправить на проверку" | Fixed frontend JSON key mismatch |
