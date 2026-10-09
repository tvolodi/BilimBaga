# Code Review: ISS-150 / ISS-152 (admin default password)

Reviewer: Code Reviewer subagent. Scope: staged diff (read-only; tests not executed, host memory low).

## Verdict: PASS (with one MEDIUM design caveat and several LOW items; no blockers)

## What was verified
- Secrets: BOOTSTRAP_ADMIN_PASSWORD is never logged or placed in errors (error strings are static; test asserts no leak). Only the generated one-time password is returned and logged once (main.go, zlog Warn). It is not persisted in plain text.
- Atomicity: both UPDATEs are compare-and-swap on `password_hash = old`, so a rotated password can never be overwritten, and concurrent replicas are safe (loser gets BootstrapRaced). Verified in SQL, and the fake store models it.
- Rotated admin: `HasDefaultAdminPassword` gate returns BootstrapNotDefault with zero writes (tested).
- Migration 033: guarded by exact 029/030 hash literals (test cross-reads the literals from 029/030), idempotent, does not stamp password_changed_at, down is hash-guarded. Correct.
- Startup failures: weak/default env password returns a static error, main exits 1 on stderr; invalid BOOTSTRAP_ADMIN_GENERATE fails config.Load. Acceptable fail-fast.
- SetAdminPassword stamps password_changed_at and clears lockout, which invalidates old tokens (ISS-105 epoch).

## Findings

1. MEDIUM - force_password_change appears to be advisory only. In `backend/internal/**` the flag is only read into the model/JSON (model.go:20,64); no server-side middleware or login handler logic blocks other endpoints while it is true. A caller using the public default credentials still gets a working token and full super_admin API access; enforcement depends on the SPA. The new warning and flag reduce risk only for UI users. Recommend a follow-up ticket: server-side gate (reject non-change-password routes while the flag is set) or have the operator always set BOOTSTRAP_ADMIN_PASSWORD for exposed deployments (README already says so). Not a regression introduced here, hence not blocking.

2. LOW - Env password validation runs before the admin lookup, so a weak BOOTSTRAP_ADMIN_PASSWORD crashes startup even when the admin was already rotated and the value would be ignored. Defensible fail-fast, but could surprise operators; consider only validating when it would be applied, or document it.

3. LOW - No maximum length check. ValidateComplexity has no cap; bcrypt rejects >72 bytes (newer x/crypto returns ErrPasswordTooLong, older silently truncates). Result is either a fatal start with a generic "hash password" error (no leak) or silent truncation. Add a `len > 72` check with a clear static message and a test.

4. LOW - Generated password goes to the structured log (zlog field `one_time_password`). Any log shipper will retain it indefinitely. Documented as "once", but note the risk; if the log is lost the admin is locked out with no recovery path other than DB reset. Consider writing to stderr only, or documenting this.

5. LOW - In a multi-replica race with BOOTSTRAP_ADMIN_GENERATE, the replica that loses logs nothing (correct), but the winner's log line is the only copy; operator must find the right replica's logs. Documentation nit.

6. LOW - Test adequacy: service logic is well covered (no admin, store error, rotated, apply, weak/default, generated, forced, race, hash literals, migration file text). Gaps: no test of the pg store SQL (CAS clause, password_changed_at) against a real DB; no test of main.go outcome/log branches; no test that Raced with Generate leaks no password; no 72-byte test. The migration test is a text check, not an execution.

7. INFO - A new bcrypt compare (cost 12, ~250 ms) runs at each startup while the hash is non-default; negligible. `.env.example` files and README are consistent with behavior. seed-test-env.ts change is benign (E2E_ADMIN_PASS only compared to the in-script default, not logged).

## Summary
Security logic is sound and atomic; migration is correct. Ship, and open a follow-up for server-side enforcement of force_password_change (finding 1) and the 72-byte guard (finding 3).
