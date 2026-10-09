# ISS-37 - Authenticated file downloads send no Bearer token

Issue: #37 (FR-BB45/46/58/59). Evidence: docs/handoffs/ba-drift-check/report.md.

## Root cause
Portal certificate (ResultActions.tsx, ResultsTable.tsx), admin certificate
(downloadAdminCertificate), dashboard PDF (downloadDashboardPdf) and exam CSV
(downloadExamCsv) used bare `fetch`/anchor navigation relying on cookies only, so the
backend (Bearer-only auth middleware) answered 401. Failures were silent (portal) or
surfaced raw error codes / nothing (admin).

## Fix
- New `frontend/src/api/download.ts`: `downloadFile(qc, url, filename)` reads the token from
  the React Query cache (same as apiFetch), sends `Authorization: Bearer`, on 401 refreshes once via
  `POST /api/v1/auth/refresh` (same endpoint as useRefreshToken, updates the cache) and retries,
  then fetch -> blob -> object URL -> anchor click, always revoking the URL. Throws a typed
  `DownloadError`; `downloadErrorKey()` maps to i18n keys.
  Note: apiFetch itself does no 401 refresh (refresh is proactive in useRefreshToken); the helper
  adds a one-shot reactive refresh.
- The four call sites now use the helper; `qc` is the first param of the exported helpers.
- UI shows `download.failed` / `download.session_expired` (en/ru/kk) in role=alert on
  portal result, portal history, admin employee record, Reports page (PDF + CSV).

## Tests
- `api/download.test.ts` (Bearer header, no-token, 401 refresh+retry, refresh failure, server code,
  URL revoke, wrapper endpoints), `ResultActions.test.tsx`, `ResultsTable.test.tsx` (error alert).
- tsc clean; `npm test` 48 files / 271 tests pass; check:i18n 732 keys in 3 locales.

## Review
Self-review performed by the Issue Resolution agent (separate Code Reviewer subagent not spawned:
no Agent tool available in this context). Backend untouched.
