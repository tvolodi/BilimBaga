---
run_id: ISS-002
pipeline: B
date: 2026-05-17
agent: Release Finalizer
---

# ISS-002 Inner Report — E2E Full-Walkthrough False-Positive Passes

## Summary

Pipeline B completed successfully. The E2E full-walkthrough suite had six systemic defects that allowed all 22 tests to pass even when the application was visibly broken. All defects were resolved by hardening assertions in `frontend/e2e/full-walkthrough.spec.ts`.

## Root Cause (from Issue Resolution)

Six defects caused systematic false-positive passes:

1. Optional `if (await element.isVisible())` guards wrapping required UI interactions — absent elements silently skipped assertions.
2. Dashboard test (03) matched error-state text (`<p class="text-xl">Load error</p>`), so a broken dashboard still passed.
3. Console-error threshold set to `< 10`, permitting 9 real JS errors.
4. `loginAsAdmin` helper did not assert the resulting URL — a failed auth that redirected to `/login` went undetected.
5. Navigation `aria-label` pattern was a latent mismatch hazard.
6. `waitForContent` used `networkidle` without a follow-on data assertion, allowing races with React Query re-renders.

## Fix

Rewrote `full-walkthrough.spec.ts`:
- `loginAsAdmin` asserts `toHaveURL(/\/admin/)` after navigation.
- Test 03 replaced dashboard selector with `getByRole('heading', { level: 1 })` + `not.toContainText(/load error/i)`.
- All `if (isVisible)` guards replaced with hard `expect(...).toBeVisible()` assertions (tests 04, 05, 06, 07, 08, 09, 10, 14, 17).
- Test 14 (Branding): removed `catch(() => false)` so form-load failures now fail the test.
- Test 16 (Sidebar nav): aria-label aligned to `"Main navigation"`.
- Test 17: removed `test.skip()`.
- Test 22 (Console errors): tightened to `toHaveLength(0)`.

## Files Changed

| File | Change |
|------|--------|
| `frontend/e2e/full-walkthrough.spec.ts` | Hardened all 22 tests |
| `docs/issue-reports/ISS-002-e2e-walkthrough-false-positive-passes.md` | Issue report (created) |
| `docs/issue-reports/README.md` | Added ISS-002 entry |
| `docs/handoffs/ISS-002/step-01a-pre-review.json` | Pre-review handoff payload (created) |

## Test Results

| Suite | Passed | Failed |
|-------|--------|--------|
| E2E full walkthrough | 22 | 0 |
| TypeScript compile | clean | — |
| Backend tests | not run (no backend changes) | — |

## Migration

None applied (no database changes).

## Commit

`test(e2e): harden full-walkthrough assertions to eliminate false-positive passes`

## Recurrence Risk

Low. The fix is structural — hard `expect` assertions cannot silently skip. Any future regression in the application will now surface as a test failure rather than a green pass.
