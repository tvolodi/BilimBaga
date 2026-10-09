# Code Review: ISS-249 (frontend 401 TOKEN_REVOKED handling), run swarm-249

Result: PASS

Verification: `vitest run apiFetch.test.ts LoginPage.test.tsx` -> 11/11 pass; `tsc --noEmit` clean.

## Findings

- [Medium] frontend/src/api/apiFetch.ts:~33 (+ lib/sessionRevoked.ts:~31) - concurrency: single-flight only dedupes *overlapping* refreshes (`inflight` is cleared in `finally` before callers resume). A second request that gets TOKEN_REVOKED slightly after the first refresh settled starts a second `/auth/refresh` (extra refresh-token rotation, hits the auth rate limit noted in auth.ts). Suggestion: before refreshing, compare the cached token with the `token` captured at call start; if it already differs, skip refresh and just retry.
- [Medium] sessionRevoked.ts:~46 - any refresh failure (network error, 5xx, 429 rate limit) is treated as "revoked" and ends the session. Suggestion: end session only on 401/403 refresh responses; rethrow original error on transient failures.
- [Medium] frontend/src/api/download.ts:~31 and api/auth.ts (useChangePassword etc.) - raw-fetch paths bypass apiFetch: download.ts has its own non-single-flight refresh and never ends the session / shows the notice on a persistent 401. Suggestion: reuse `refreshAccessTokenOnce` there (follow-up ticket acceptable).
- [Medium] Missing test: no concurrency test (two parallel revoked requests -> exactly one refresh call) and no test that the retry re-sends the same body/headers. Add both.
- [Low] Request body reuse: retry re-passes the same `options`; fine for string/FormData/Blob bodies (all current callers); a ReadableStream body would fail on retry. Document or guard.
- [Low] sessionRevoked.ts:`isTokenRevoked` is exported but unused (apiFetch inlines the check) - use it or remove.
- [Low] endRevokedSession does not call server `/auth/logout`, so the refresh cookie survives; a page reload will silently re-authenticate via useRefreshToken with fresh claims. Benign (arguably desired) but the notice is lost on reload; no refresh loop results (refetchInterval returns false for null data).
- [Low] LoginPage reads the flag via non-reactive `getQueryData`; works because the flag is set before the token is cleared. `useLogout` does not clear the flag (cleared only on next successful login; in-memory only, so harmless).

## Checks
- Refresh loops: none. Max one refresh + one retry per call; stale/same-token and second TOKEN_REVOKED both end session (tested).
- useRefreshToken/RequireAuth: token set to null with `setQueryData` on the same key; refetchInterval disabled for null; RequireAuth redirect works as with existing logout. E2E localStorage token handled.
- i18n: `auth.login.sessionRevoked` present in en/ru/kk; no hardcoded strings. Test asserts key resolves.
- Security: no secrets; all via apiFetch; no direct fetch in components.
- Critical/High: none.

Summary: PASS - no Critical/High issues; Medium items (redundant refresh after settled flight, transient refresh failures ending the session, download.ts parity, missing concurrency test) recommended as follow-ups.
