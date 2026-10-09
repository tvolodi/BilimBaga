# Code Review: ISS-132 (employee Start-exam gives no feedback)

Commit 86b91ad vs origin/main. Reviewer: Code Reviewer subagent. Frontend tests were reviewed by reading only (node_modules unavailable); backend `go test ./internal/sessions/` is green.

## Verdict: PASS (0 Critical, 0 High, 2 Medium, 4 Low)

## Checks
- Zero-question guard (`backend/internal/sessions/service.go:180-185`): placed after the rule loop and before any repo insert, so no `exam_sessions` row is created. Covers no rules, an empty manual rule, and all-archived manual questions. Random rules with too few questions are still caught earlier (line 165). Adaptive exams are exempt via `cfg.Adaptive` (the field exists on `examConfig`, model.go:118). Maps to 422 INSUFFICIENT_QUESTIONS through the existing handler case. Correct.
- Backend tests: the three new service tests and the handler log test are meaningful. `slog.SetDefault` is restored in `t.Cleanup`. The log test does not use `t.Parallel`, so there is no race on the global logger.
- Frontend error handling: `apiFetch` tolerates non-JSON bodies and attaches `code` (portal.ts). `handleClose` calls `createSession.reset()`, and the Dialog `onOpenChange` routes to `onClose`, so the error clears on close. The `role="alert"` paragraph is rendered only when there is a message. A fetch network rejection (TypeError, no code) resolves to the key `portal.startError.undefined`, which falls back to `generic`. Fine.
- i18n: the dynamic key `portal.startError.${code}` has a `defaultValue` fallback. en/ru/kk define the same 8 keys (ru verified; the author reports check:i18n green).
- Frontend tests (read-through):
  - Default language is `en`, because `i18n.ts` uses `localStorage ?? 'en'`.
  - `/start exam/i` matches only the card button, because `portal.card.start` = "Start exam" (en.json:669) and the modal title is not a button.
  - While the Radix dialog is open, outside content is aria-hidden, so `/begin exam/i` and `/cancel/i` are unambiguous.
  - `userEvent` and `msw` are in package.json, and `userEvent` is already used in other tests.
  - The 502 handler `new HttpResponse('<html>', {status:502})` yields a non-JSON body, which exercises the catch path.
  - The reopen test relies on `reset()` in `handleClose`, which is implemented.
  - I expect all four to pass. Residual risk is low, since they were not executed.

## Findings

### Medium
1. `backend/internal/sessions/handler.go:37` logs every failure at `Warn`. This includes the unexpected `ERR_INTERNAL` default branch (HTTP 500), which should be `Error`. It also includes routine business refusals (not assigned, attempts exhausted), which will add Warn noise. Suggest: `Error` when no sentinel matched, `Info` or `Warn` for the expected ones.
2. `service.go:182-183`: an adaptive exam with zero rules is allowed to start. I did not verify that adaptive question selection works without rules (`SelectNextAdaptiveQuestion` draws from a pool that is not obviously independent of rules). If adaptive exams need rules or a pool, they would still start into an empty session. Worth confirming with the adaptive requirement (FR-BB72), though it is outside this issue's scope.

### Low
3. `backend/internal/sessions/model.go:23`: the text of `ErrInsufficientQuestions` says "...to satisfy a random rule". It is now also used for zero-question exams. The wrapped log message disambiguates it, so this is cosmetic. The handler message ("question pool is too small") is acceptable.
4. `frontend/e2e/exam-start-failure.spec.ts:62-66`: the third test does not intercept the POST. It creates a real open session for the seeded employee, which can make later specs hit SESSION_ALREADY_OPEN or consume attempts. The first two tests also return silently when no Start button exists (an annotation is added only in test 1). Suggest `test.skip` with a reason, and cleanup or a dedicated exam for test 3. The spec was not run.
5. `frontend/src/pages/EmployeePortal/index.tsx:32-35`: `onError` invalidates the exam list, but `selectedExam` is a stale snapshot held in state. For SESSION_ALREADY_OPEN the modal stays open with the old exam data until the user closes it. This is acceptable, because the message tells the user to open it from the list. Not covered by a test (the invalidate-on-error behavior is untested).
6. Logging: `user_id` and `department_id` are UUIDs, not direct PII, and no email, name, or token is logged. This is fine and noted for completeness. The ISS doc correctly records that the frontend tests were not run.

## Not verified
- Frontend vitest, tsc, and Playwright were not executed (no node_modules). The read-through above found no failing assertions.
