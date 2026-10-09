# Code Review: ISS-059 (GitHub #59) - E2E test 21 split into 21a/21b

Verdict: APPROVE

## Scope
- frontend/e2e/full-walkthrough.spec.ts (test 21 -> 21a, 21b)
- docs/issue-reports/ISS-059-e2e-portal-redirect-wrong-assertion.md

## Findings
- Root cause confirmed: /portal is RequireRole ['employee']; RequireRole sends no token -> /login, non-employee role -> /admin. The old regex /\/login|\/portal/ could never match for the admin storageState session. Test defect, not a product bug.
- 21a: fresh context with empty storageState and explicit baseURL; asserts /login; context closed in finally. Correct and leak-safe.
- 21b: reuses loginAsAdmin, then /portal; asserts /\/admin/ (matches the /admin -> /admin/dashboard chain). Correct. Assertion is no longer vacuous.
- Both tests remain in chromium-live-admin testMatch; no config change needed.
- Issue report is accurate and consistent with the diff; live E2E honestly marked not run.

## Non-blocking notes
- 21a does not exercise the employee-positive path; that is covered by the chromium-live-employee project.
- Live execution is unverified here (not run per instructions); recommend confirming in the next UAT/E2E run.
