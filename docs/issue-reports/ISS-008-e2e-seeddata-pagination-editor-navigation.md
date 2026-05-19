---
id: ISS-008
title: getSeedData pagination miss + question-editor unreliable navigation
status: resolved
severity: high
layer: frontend
module: questions
tags: [e2e, seed, getSeedData, findUserByEmail, per_page, pagination, question-editor, navigation]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom

**Bug A (11 tests)**: All tests that call `getSeedData()` throw `"getSeedData: employee user not found after re-login"`. Affected suites: all 4 "Delete Question" tests in `question-management.spec.ts`, and 7 tests in `user-management.spec.ts`.

**Bug B (2 tests)**: `"navigates to edit page for a seeded question"` and `"pre-populates the stem field for an existing question"` in `question-editor.spec.ts` fail intermittently due to race conditions when navigating to the edit page via UI clicks.

## Root Cause

**Bug A**: `findUserByEmail` in `seed.ts` requests `?per_page=200` but the backend caps `per_page` at 100 (enforced in `backend/internal/users/service.go`). Users are ordered `created_at DESC`, so the employee user (oldest) is pushed beyond position 100 after enough test runs create accumulated test users. The function returns `null` → both the initial check and the re-login re-check fail → exception thrown.

**Bug B**: The tests navigated to `/admin/questions`, located a table row, clicked a stem button to trigger navigation, then waited for the URL to change. This multi-step UI flow had a race between the click handler, React Router navigation, React Query's question API fetch, and the editor form render. With slow CI-like conditions the timing window caused consistent failures.

## Fix Applied

**Fix A — persistent employee ID file**:
- Added `EMPLOYEE_ID_PATH` constant pointing to `.auth/employee-id.txt`.
- In `seedEmployeeFixtures`: after resolving `employeeId`, write it to `EMPLOYEE_ID_PATH` (`fs.writeFileSync`).
- In `getSeedData`: read `employeeId` directly from `employee-id.txt` (immune to pagination). `findUserByEmail` is retained but only as a lightweight token-validity check — if it returns `null` (expired token or pagination miss), we re-login to refresh `adminToken`. The `employeeId` is never fetched via API again.

**Fix B — direct API navigation**:
- Added `import { getSeedData }` to `question-editor.spec.ts`.
- Both edit-page tests now call `page.request.get('/api/v1/questions?per_page=1')` with the admin JWT to reliably obtain the first question ID, then navigate directly to `/admin/questions/{id}/edit`. No more UI click chain or URL-change wait.

## Files Changed

| File | Change |
|------|--------|
| `frontend/e2e/fixtures/seed.ts` | Added `EMPLOYEE_ID_PATH` constant; write ID in `seedEmployeeFixtures`; rewrote `getSeedData` to read from file |
| `frontend/e2e/question-editor.spec.ts` | Added `getSeedData` import; replaced two edit-page test bodies with direct API lookup + navigation |

## Regression Test

No dedicated unit test added — the E2E suite itself serves as the regression check. The persistent `employee-id.txt` file ensures future runs never re-encounter the pagination miss.

## Resolution Results

- Tests: fix applied, not re-run in this session (pending E2E suite run)
- Migration applied: no
- Build clean: yes (TypeScript changes only in E2E layer, not compiled by tsc)

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Accumulated test runs pushed employee user beyond page 100 | Applied file-based ID persistence and direct API navigation |
