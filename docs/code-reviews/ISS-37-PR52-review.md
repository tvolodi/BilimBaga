# Code Review - PR #52 (ISS-37, authenticated downloads send Bearer)

Branch: swarm/37-download-bearer. Reviewer: Code Reviewer subagent. Verdict: **CHANGES REQUESTED**

## What is good
- `download.ts` reads the token from the `['auth','accessToken']` cache (same as apiFetch), sends `Authorization: Bearer` only in headers (never in URL, never logged).
- 401 retry is strictly one-shot: one refresh call, one retry, no loop. A second 401 throws `ERR_UNAUTHORIZED`. The refreshed token is written to the cache before the retry, and the retry re-reads it via `authInit`. A refresh failure (non-OK, `error`, missing token, network throw) returns false and surfaces as `ERR_UNAUTHORIZED`.
- Blob URL is revoked in a `finally`; it is created only after `res.ok`, so no leak on the error paths.
- All user-visible errors go through `t('download.failed' | 'download.session_expired')`. Keys exist in en/ru/kk with matching structure. Raw error codes are no longer shown (SessionHistoryTable fixed).
- The five intended call sites are converted: ResultActions, ResultsTable, `downloadAdminCertificate`, `downloadDashboardPdf`, `downloadExamCsv`. Call sites catch errors and render `role="alert"`.
- Tests assert the Authorization header, no-token case, 401 refresh+retry with new token and cache update, refresh failure, server error code, URL revoke, and the endpoint URLs of all three API wrappers.

## Blocking issues
1. **A sixth authenticated download is still broken (same bug class).** `frontend/src/pages/admin/questions/QuestionBankPage.tsx` ~line 779-784 (`handleExport`) does an anchor `a.href = /api/v1/questions/export?...; a.click()`. This is a bare navigation with no Bearer header, so the Bearer-only middleware returns 401. This is exactly the failure mode in #37. Convert it to `downloadFile(qc, url, \`questions.${format}\`)` with an i18n error alert, plus a test. If the team deliberately wants it split out, file a tracked follow-up issue and say so in the PR; per the project's unblock directive, fixing it here is preferred.
2. **Missing E2E Bearer assertion.** The code-review checklist (Critical) requires an E2E test asserting `Authorization: Bearer ...` on the wire via `page.waitForRequest` for new/changed API modules. The PR adds no `frontend/e2e` spec (only `categories.spec.ts` and `tags.spec.ts` use `waitForRequest`). Unit tests with a stubbed fetch do not catch the storageState-warms-cache class of bug. Add at least one E2E spec covering a download (e.g. admin dashboard PDF or portal certificate).

## Non-blocking / suggestions
- Checklist item "no custom fetch wrappers that bypass apiFetch": `downloadFile` is a deliberate parallel wrapper (blob responses, one-shot refresh). Acceptable, but add a short comment in `download.ts` explaining why it does not use `apiFetch`. Consider migrating the already-correct raw-fetch exporters (`exportAuditLog`, `exportEmployeeRecord`, `ExportCSVButton`) to `downloadFile` so they gain the 401 refresh and i18n error handling; they currently send Bearer but have no refresh and `ExportCSVButton` has no catch (unhandled rejection, no user feedback). They are not regressions.
- Call sites in `api/*.ts` take a `QueryClient` and perform fetches from components, not React Query hooks; this matches existing `apiFetch` usage patterns, so acceptable.
- The filename always uses the caller fallback and ignores `Content-Disposition`. Matches the previous behavior; fine.
- `URL.revokeObjectURL` runs synchronously right after `click()`. Works in current Chromium/Firefox, but deferring with `setTimeout(..., 0)` is safer. Note the tests assert revocation synchronously, so change them together if adopted.
- Concurrent 401s each trigger their own refresh (no single-flight). Low risk for user-initiated downloads.
- Test gaps: no test that a second 401 after a successful refresh does not trigger a third fetch (code is correct; add `expect(fetchMock).toHaveBeenCalledTimes(3)`). The `ResultsTable` test does not set a token or assert the Authorization header (covered in `ResultActions` and `download.test.ts`). The i18n error text assertions are English-only; there is no kk/ru render check (`check:i18n` covers key parity).
- `SessionHistoryTable` error banner lacks `role="alert"` (unlike the other sites) and still uses a `setTimeout` without cleanup (pre-existing).
- The issue report states "Self-review... separate Code Reviewer not spawned"; this document is that independent review.

## Verdict
CHANGES REQUESTED: fix or formally defer the QuestionBank export (blocking 1) and add the E2E Bearer assertion (blocking 2). Core helper logic is sound.
