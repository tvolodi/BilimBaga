# Code Review - ISS-026 (#26): tests for internal/{db,email,ctxkeys,router}

Verdict: APPROVE (PASS) - 0 Critical, 0 High, 0 Medium blocking, 4 Low/Medium advisory.

## Scope
Nine new untracked `_test.go` files. `git diff HEAD` is empty: no production file or existing test modified, no migrations touched.

## Verification run
`go test -p 2 -race -count=2 ./internal/email/ ./internal/db/ ./internal/ctxkeys/ ./internal/router/` - all ok, race detector clean. `go vet` on the four packages - clean.

## Checklist
- Determinism: no `time.Sleep`. The only fixed waits are `assertNoLog` (150ms, under the 200ms cap). Positive waits use channel signalling with a 5s timeout ceiling, so they return as soon as the event fires.
- Network: only a loopback fake SMTP on `127.0.0.1:0`; "refused" cases use `127.0.0.1:1`. The fake serve loop has a 5s conn deadline, and the listener is closed in `t.Cleanup`.
- Race safety: `stubRepo` guards `logs` with a mutex and signals through a buffered channel (cap 32). Config fields are set before the goroutine starts. In `TestTriggerAssignment/silent no-op paths` the test mutates different fields (`userErr`, `deptErr`, `allErr`) while earlier goroutines may still read other fields. This is race-free as written but fragile (see M1).
- Fakes: `fakeDB`, `migFake` and the SMTP fake are mutex-protected and closed via cleanup. The `migFake` answers exactly what golang-migrate issues. Windows path handled via `filepath.ToSlash` for the `file://` URL, and the tests pass on Windows.
- Assertion strength: good overall (error-prefix wrapping, locale-specific subjects, recipient and template names, header-injection check, sanitised `$N` in slow-query logs, ErrNoChange mapping, dirty-version message).
- Leaks: `StartDeadlineReminderScheduler` tests leave an `AfterFunc` timer pending (hours). It is inert (ctx cancelled) and harmless for the test process.

## Findings
- [Medium] M1 `email/service_more_test.go` `TestTriggerAssignment` "silent no-op paths": mutates `repo.userErr/deptErr/allErr` on the shared stub without the lock, while goroutines from prior calls could still be running. It is currently safe because the fields differ, but it breaks if a goroutine ever reads several fields. Suggest a fresh stub per case, or guarding setters with `repo.mu`.
- [Low] L1 `assertNoLog` is a negative assertion over a 150ms window. It can false-pass if the goroutine is delayed (for example on a loaded CI box). It cannot flake red, only miss a regression. Acceptable; a fake that counts calls synchronously would be stronger.
- [Low] L2 `TestTestSend` header-injection section: `t.Log` branch when `err == nil` and a weak loop. If the send fails nothing is asserted, and the "Bcc" check only inspects messages already delivered. Consider asserting that either an error is returned or the headers contain no injected Bcc, explicitly.
- [Low] L3 `startFakeSMTP` uses `t.Skipf` when loopback listen fails. This silently skips the coverage in locked-down environments. Prefer `t.Fatalf` if coverage is gated in CI.

## Summary
Deterministic, race-clean, loopback-only tests with strong assertions and no production changes. Approved; the advisory items can be followed up optionally.
