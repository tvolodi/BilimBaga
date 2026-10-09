# Code Review: ISS-105 account-recovery hardening

Scope: staged diff on `swarm/105-recovery-hardening` (17 files). `go vet` and `go test` for `auth`, `router` and `users` pass.

## Verdict: APPROVE

No Critical or High findings. Non-blocking notes follow.

## Blocking issues

None.

## Non-blocking findings

- [Medium] `recovery_service.go` ForgotPassword: `PurgeExpiredResetTokens` (a DELETE) and a transaction with a `FOR UPDATE` lookup now run for every unauthenticated request, including unknown emails. The old code deliberately kept unknown-email calls write-free. This is the price of uniform work. It is acceptable only because the route has a rate limiter. Confirm the public limiter covers `/auth/forgot-password`. Consider purging probabilistically or on a schedule instead.
- [Medium] "Structurally equal work" is only partly true at the DB level. For the nil UUID, `IssueResetToken` runs 1 statement (begin, lock, no row, rollback). A real user runs 4 statements plus a commit. The 400 ms handler pad is what hides this, not the service. The comment says as much, but the padding is the real control. The 400 ms floor must stay well above the p99 of the known path. Consider logging when the pad is exceeded.
- [Medium] `hardening_test.go` `TestForgotPassword_ConcurrentRequests_CannotExceedLimit` exercises the in-memory mock, whose mutex stands in for the row lock. It proves the service wiring, not the SQL guarantee. The txFake tests only assert statement order. There is no real-Postgres test of the lock. It is correct by inspection (see below), but a gap.
- [Low] `IssueResetToken` uses `BeginTxx(ctx, nil)`, which takes the server default isolation. Under READ COMMITTED (Postgres default) a waiter blocks on `FOR UPDATE`, and the following `COUNT(*)` takes a fresh snapshot that sees the first transaction's committed insert, so the semantics are right. Under REPEATABLE READ the snapshot is taken at the first statement, so the count would miss it. Pass `&sql.TxOptions{Isolation: sql.LevelReadCommitted}` to make this independent of server config.
- [Low] Clock mixing: `CompleteReset` stamps `password_changed_at` from the app clock (`now`). Admin `users.UpdatePassword` stamps from the DB clock (`now()`). Access tokens use the app clock. If app and DB clocks differ by more than 1 s, a fresh login immediately after an admin reset could be rejected as `TOKEN_REVOKED`. Use one source (prefer the DB `now()` in `CompleteReset` too).
- [Low] iat edge case: the comparison is `iat < changedAt.Unix()` (strictly before, second resolution). A token minted in the same second as the reset, but before it, is still accepted for up to 1 s, because `iat` is truncated to seconds. This is a conscious trade-off to avoid rejecting a fresh post-reset login in the same second. It is acceptable, and the refresh tokens are revoked as well.
- [Low] The 10 s cache means a reset takes up to 10 s to take effect on a given instance. This is documented and the tests cover it. When the cache hits the 10,000-entry cap it is cleared wholesale (a brief thundering herd on the users table). This is fine at current scale.
- [Low] Self-service `auth.UpdatePassword` (change-password, `repository.go:158`) does not stamp `password_changed_at`. This looks intentional, so the user's own session survives. It is worth stating in the issue doc, since other sessions of that user remain valid.
- [Low] The `Referrer-Policy` meta tag is set from `useEffect`, after first paint. The nginx header on the document is the real control. The meta tag is only a dev-server and defence-in-depth fallback. The global header is already `same-origin`, so the leak risk being closed was same-origin only.

## Checklist verification

| Area | Result |
|------|--------|
| SQL columns vs migrations | `password_reset_tokens(user_id, token_hash, expires_at, used_at, created_at)` matches 031. `users.password_changed_at TIMESTAMPTZ` matches 032 and is nullable (zero time, no rejection). `users.status = 'active'` is in use. The down migration is present and 032 is a new file, so no existing migration was edited. |
| Atomicity | The lock is taken first, then the count, then the invalidate and insert, all in one transaction and committed once. Rollback is deferred. Correct under READ COMMITTED (see the Low note on isolation). |
| Fail closed | A lookup error returns 500 (`INTERNAL_ERROR`). `ErrNotFound` returns 401. A missing `iat` returns 401. Only the epoch-enabled middleware requires `iat`. All issued tokens carry `iat` (`service.go:298`). Until migration 032 is applied, every authenticated request returns 500 (fail closed by design; `make migrate` handles it). |
| Middleware wiring | The only production `Authenticate` call is in the router, which now passes the DB-backed epoch lookup. A nil DB falls back to the old behaviour for route tests only. |
| Handler padding | Padding applies after a successful service call. The response is written and flushed, then the audit write runs. If a wrapping writer lacked `Flusher`, the known path would carry the audit latency, so it is only a minor residual. Error paths (validation or DB) are not padded, which leaks nothing about existence. |
| nginx | `location = /reset-password` repeats every security header, so the `add_header` non-inheritance is handled. It adds `no-referrer` and `no-store`. `try_files /index.html =404` serves the SPA. The frontend route is `/reset-password` (`App.tsx:113`). The config test regex is valid and the path `../../../deploy/nginx.conf` resolves from `backend/internal/router`. It is the only nginx config in the repo. A host vhost, if any, proxies through and does not strip the headers. |
| Tests | Meaningful: sequence equality across known, unknown, inactive and throttled; padding bounds; lock-then-count order in one transaction; no insert at the limit; the reset stamps the column; reject, accept, missing-iat, unknown-user and error fail-closed cases; cache TTL; `iat` equality edge. Gaps are listed above (no real-DB test). The frontend test covers mount and unmount of the meta tag. |
| Conventions | Errors are wrapped with context. SQL is in the repository layer. No `os.Getenv` in handlers. The response envelope is unchanged. |

## AC coverage (issue #105 items)

1. Uniform ForgotPassword work and handler padding: covered.
2. Atomic IssueResetToken (row lock): covered. The real-DB test is missing, which is non-blocking.
3. Referrer-Policy no-referrer (nginx and meta): covered.
4. `password_changed_at` (migration 032) and `WithPasswordEpoch` with 10 s cache: covered.
