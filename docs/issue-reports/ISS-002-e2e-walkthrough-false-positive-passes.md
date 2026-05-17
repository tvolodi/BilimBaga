---
id: ISS-002
title: E2E full walkthrough test has systematic false-positive passes
status: resolved
severity: high
layer: frontend
module: auth
tags: [e2e, playwright, false-positive, isVisible, soft-assertion, walkthrough]
created: 2026-05-17
resolved: 2026-05-17
recurrence_count: 1
related_issues: []
regression_test: frontend/e2e/full-walkthrough.spec.ts
---

## Symptom
The 22-test `full-walkthrough.spec.ts` suite reports all passing even when screens are broken. The user opens the application and the first screen is visibly non-functional, yet the E2E suite did not catch it.

## Root Cause
Six systemic defects in the test file:

1. **`if (await element.isVisible())` guards everywhere** — form interactions that MUST succeed on a healthy screen were wrapped in optional guards. If the element was absent (due to a bug), the test silently skipped the assertion and passed.

2. **Dashboard test (03) matched error-state text** — `page.locator('h1, [data-testid="kpi-card"], .text-xl').first()` matched the error paragraph `<p class="text-xl">Load error</p>`, so a broken dashboard still passed.

3. **Console-error threshold too high** — `expect(realErrors.length).toBeLessThan(10)` allowed 9 real JS errors to pass as green.

4. **`loginAsAdmin` did not assert the resulting URL** — if auth failed and the app redirected to `/login`, the helper returned silently and all subsequent navigations would also be unauthenticated, each passing for the wrong reasons.

5. **Navigation aria-label mismatch** — the test used `/main navigation/i` but the Sidebar renders `aria-label="Main navigation"`. Case-insensitive regex happened to match, but the mismatch was a latent hazard.

6. **`waitForContent` used `networkidle` without a follow-on data assertion** — React Query re-renders after `networkidle` fires, so assertions immediately after `waitForContent` could run while the page still showed a loading skeleton or error state.

## Fix Applied

Rewrote `full-walkthrough.spec.ts` with:

- `loginAsAdmin` now asserts `toHaveURL(/\/admin/)` after navigation; redirects to `/login` throw immediately with a clear message.
- Test 03 replaced with `getByRole('heading', { level: 1 })` + `not.toContainText(/load error/i)`.
- Test 04: `if (isVisible)` → hard `expect(createBtn).toBeVisible()` + required drawer assertion.
- Test 05 (Departments "coming soon"): hard assert heading + "coming soon" text instead of optional add-button guard.
- Test 06 (Categories): `expect(addBtn).toBeVisible()` required; fill the name field as required.
- Test 07 (Tags): `expect(createBtn).toBeVisible()` required.
- Test 08 (Question bank): `expect(searchInput).toBeVisible()` + `expect(newBtn).toBeVisible()` as hard assertions.
- Test 09 (Question editor): type selector, stem field, difficulty selector all asserted with `expect(...).toBeVisible()`.
- Test 10 (Exams list): create exam link/button asserted as visible.
- Test 14 (Branding): removed `catch(() => false)` — form load failure now fails the test.
- Test 16 (Sidebar nav): aria-label pattern aligned to `"Main navigation"`.
- Test 17 (User create validation): removed `test.skip()` — button absence now fails.
- Test 22 (Console errors): changed to `toHaveLength(0)`.

## Files Changed
| File | Change |
|------|--------|
| `frontend/e2e/full-walkthrough.spec.ts` | Hardened all 22 tests: replaced optional `if (isVisible)` guards with `expect(...).toBeVisible()`, fixed auth URL assertion, tightened console-error check |

## Regression Test
`frontend/e2e/full-walkthrough.spec.ts` — the file itself is the regression test. Running it against a working stack must produce 22 green tests; running it against a broken first screen will now fail test 03 or the loginAsAdmin auth gate.

## Resolution Results
- Tests: 22 passed, 0 failed (full E2E suite run against live stack)
- Migration applied: no
- Build clean: yes (TypeScript compiles cleanly)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
| 2026-05-17 | User observed broken first screen despite green E2E report | Hardened all 22 test assertions |
