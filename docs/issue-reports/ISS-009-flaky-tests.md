# ISS-009 Flaky test audit (GitHub #9)

Date: 2026-10-09

## Method
- Reviewed docs/test-reports/: no recurring failures or flaky-test records (coverage baseline, e2e static audit, perf only).
- `go test -p 2 -count=3 ./...` : all 29 packages pass.
- `npx vitest run --maxWorkers=2` twice: 88 files / 632 tests pass both runs.
- `go test -race -p 2 -count=1 ./...` : two data races found (below).

## Findings (test-only data races, fixed)
1. `internal/auth` `fakeMailer.calls` appended without a lock while
   `TestForgotPassword_ConcurrentRequests_CannotExceedLimit` calls ForgotPassword from 40 goroutines.
   Fix: mutex in `fakeMailer`.
2. `internal/email` `mockRepo.logAttempt*` fields written from the async send goroutine and polled
   by `TestTriggerPasswordResetLink_SendsLinkAndNeverLogsToken`. Fix: mutex + `logged()` snapshot accessor.

No tests skipped or deleted; assertions unchanged. After fix, `go test -race ./...` is clean.
No clock/ordering/map-iteration flakiness observed in 3x Go and 2x vitest runs.
