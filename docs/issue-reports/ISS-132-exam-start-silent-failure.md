---
id: ISS-132
title: Employee presses Start, exam does not start (no feedback)
status: resolved
severity: critical
layer: frontend
module: sessions
tags: [StartExamModal, useCreateSession, EXAM_OUTSIDE_WINDOW, INSUFFICIENT_QUESTIONS, SESSION_ALREADY_OPEN, silent failure]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx
---

## Symptom
Customer list #7 item 4 (GitHub #132): "Cannot start exam: the Start button is pressed but the exam does not start."
The confirm button spins for a moment and the modal stays on screen with no message.

## Root Cause
Analysis only (no live stack was used). Two independent defects combine; the first one turns every refused start into the reported symptom.

1. **Failure is swallowed by the UI (always present).**
   - `frontend/src/pages/EmployeePortal/index.tsx` `handleConfirm` called `createSession.mutate(undefined, { onSuccess })` with no `onError`, and nothing rendered `createSession.error`.
   - `frontend/src/pages/EmployeePortal/StartExamModal.tsx` had no error slot.
   - `frontend/src/api/portal.ts` `apiFetch` threw a bare `Error(message)` (the backend `code` was dropped) and `await res.json()` threw a `SyntaxError` on non-JSON gateway responses.
   - Result: any 4xx/5xx from `POST /api/v1/portal/exams/{id}/sessions` leaves the modal open, the spinner stops, nothing else changes.
2. **Start is offered when the backend will refuse it (the actual refusals).**
   `portal.computeStatus` (`backend/internal/portal/service.go:121`) and `ListAssignedExams` (`portal/repository.go:82`) ignore `available_from`/`available_until`, so an active exam outside its window is shown as `not_started` with a Start button, while `sessions.service.CreateSession` (`backend/internal/sessions/service.go:117-123`) answers 422 `EXAM_OUTSIDE_WINDOW`. Similarly a stale card whose session was opened elsewhere gets 409 `SESSION_ALREADY_OPEN`.
3. **Empty exams "start" into an empty session.** `CreateSession` (`service.go`, rule loop) accepted zero resolved questions (exam without rules, or a manual rule whose questions were all archived) and inserted an `exam_sessions` row; the test `TestCreateSession_WithinWindow` even asserted "No rules -> session creates with zero questions". The employee lands on an empty exam page, and `HasOpenSession` then blocks further starts (409) until the empty session expires.
4. **Backend gave no diagnostics.** `sessions/handler.go` `CreateSession` mapped errors to responses without logging the wrapped cause, so a support report could not be traced to a rule.

No RBAC cause: the route is registered without `RequirePermission` (`router.go:231`, "any authenticated user"). No SQL/schema cause found (queries checked against migrations; schemaguard unchanged). Route, method, body, token, `{data,error}` envelope and the `session_id` field used for navigation all match between frontend and backend.

## Fix Applied
- `api/portal.ts`: `apiFetch` now throws `PortalApiError` carrying `code`, and tolerates non-JSON bodies (`ERR_HTTP`).
- `EmployeePortal/index.tsx`: maps the error code to `portal.startError.<CODE>` (fallback `portal.startError.generic`), passes it to the modal, and refreshes the exam list on error.
- `StartExamModal.tsx`: renders the message in a `role="alert"` paragraph.
- i18n: `portal.startError.*` added to en/ru/kk (`npm run check:i18n` green: 788 keys).
- `sessions/service.go`: non-adaptive exam resolving to zero questions returns `ErrInsufficientQuestions` (422) before any insert.
- `sessions/handler.go`: logs every refused start with exam/user/department and the wrapped error.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/api/portal.ts | error code propagation, non-JSON tolerance |
| frontend/src/pages/EmployeePortal/index.tsx | onError refresh, translated error |
| frontend/src/pages/EmployeePortal/StartExamModal.tsx | error alert |
| frontend/src/locales/{en,ru,kk}.json | portal.startError.* |
| backend/internal/sessions/service.go | zero-question guard |
| backend/internal/sessions/handler.go | refusal logging |

## Regression Test
- Frontend: `EmployeePortal.test.tsx` "Start exam flow (ISS-132)": success navigates, coded error shows alert, 502 non-JSON shows generic alert, error clears on reopen.
- Backend: `service_test.go` `TestCreateSession_NoRules_ReturnsInsufficientQuestions`, `..._ManualRuleAllQuestionsArchived_...`, `..._AdaptiveExamWithoutRules_Starts`; `handler_test.go` `TestCreateSession_Handler_LogsRefusalCause`.
- E2E (unexecuted): `frontend/e2e/exam-start-failure.spec.ts`.

Why existing tests missed it: the frontend tests only covered the list rendering (never clicking Start/Begin); the Playwright test 05 returns early when no Start button exists and never asserts the failure path; the backend tests mocked the repo and treated "zero questions" as valid.

## Resolution Results
- Backend: go vet clean, `go test -p 2 ./...` green, staticcheck 0 findings.
- Frontend vitest/tsc/playwright --list: NOT run (node_modules junction refused by the sandbox). `check:i18n` run and green.
- Migration applied: no
- Build clean: backend yes; frontend unverified

## Open follow-ups (not changed here)
- Portal list should hide or disable Start outside the availability window (needs `available_from/until` in the portal query and DTO).
- If UAT still shows "nothing happens" with an alert now visible, the alert text/code identifies the real refusal; if no request is sent at all, capture the browser console and network tab.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
