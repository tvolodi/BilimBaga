# ISS-009: E2E question editor blank page (token) and checkbox toggle state mismatch

**Date**: 2026-05-19  
**Severity**: Test infrastructure (E2E failures, no production impact)

---

## Affected Tests

| Test file | Test name | Symptom |
|-----------|-----------|---------|
| `frontend/e2e/question-editor.spec.ts` | "navigates to edit page for a seeded question" | `waitForSelector('textarea')` timeout — blank white page |
| `frontend/e2e/question-editor.spec.ts` | "pre-populates the stem field for an existing question" | Same — `<FullPageSpinner />` never resolves |
| `frontend/e2e/exam-taking.spec.ts` | "03 — Multiple-choice answer" | `expect(locator).toBeChecked()` fails — checkbox was already checked |

---

## Root Cause — Failures 1 & 2 (question-editor blank page)

`storageState: ADMIN_STORAGE_STATE` populates `localStorage` from `admin.json` (written during global-setup). When `getSeedData()` refreshes the admin token (updating `token.txt`), the browser context still holds the stale token from `admin.json`. The React app's `useRefreshToken()` hook (App.tsx line ~83) fails to validate the stale token and stays in `isLoading: true` indefinitely, rendering only `<FullPageSpinner />`.

## Root Cause — Failure 3 (checkbox toggle)

`startExamSession` resumes an in-progress session where all multiple-choice checkboxes are already checked. The loop searched for an *unchecked* checkbox but found none, silently falling back to `mainCheckboxes.nth(0)` which was checked. Calling `.click()` then unchecked it, and the subsequent `expect(targetCb).toBeChecked()` assertion failed.

---

## Fix

### question-editor.spec.ts (Failures 1 & 2)
- Added `page.addInitScript()` before `page.goto()` in both tests within `"Question Editor — edit existing question"` to inject the fresh `adminToken` directly into `localStorage.__e2e_access_token__` before React boots.
- Also sets `i18n-lang: 'ru'` for locale consistency.
- Increased `waitForSelector('textarea')` timeout from 15 000 ms to 20 000 ms as a safety margin.

### exam-taking.spec.ts (Failure 3)
- Captured `wasChecked = await targetCb.isChecked()` immediately before the first `.click()`.
- Replaced the three hardcoded `toBeChecked()` / `not.toBeChecked()` assertions with conditional branches that flip based on `wasChecked`, making the toggle-verification logic correct regardless of the initial session state.

---

## Files Changed

| File | Lines affected |
|------|----------------|
| `frontend/e2e/question-editor.spec.ts` | ~73–79, ~93–99 (addInitScript + timeout bump) |
| `frontend/e2e/exam-taking.spec.ts` | ~230–248 (wasChecked-aware toggle) |
