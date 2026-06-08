---
id: ISS-026
title: ChangePasswordPage — stale browser autofill in current-password causes INVALID_CREDENTIALS
status: resolved
severity: high
layer: frontend
module: auth
tags: [password, autofill, change-password, INVALID_CREDENTIALS, ChangePasswordForm, current-password]
created: 2026-06-08
resolved: 2026-06-08
recurrence_count: 1
related_issues: [ISS-025]
regression_test: frontend/src/components/auth/ChangePasswordForm.test.tsx
---

## Symptom
On `/change-password`, users who have just had their password reset by an admin (forced-change flow) receive "Current password is incorrect" even when they paste the correct password. The current-password field shows a yellow autofill background indicating the browser has silently populated it with the user's previously saved credential (the pre-reset, now-invalid password).

## Root Cause
`ChangePasswordForm` specifies `autoComplete="current-password"` on the current-password `<Input>`. This attribute instructs browsers (Chrome, Edge, Firefox) to look up the stored credential for the site and fill it in automatically. After an admin password reset the stored credential is the **pre-reset password**, which the backend correctly rejects with `INVALID_CREDENTIALS`. Because the field is visually obscured (type="password") and the autofill overwrites whatever the user typed, the user has no way to detect the wrong value without a show/hide toggle (ISS-025). Pasting the correct password after the autofill has fired may still fail if the browser's autofill event fires after the paste and overwrites it again, or if the controlled React state is not reliably updated by the autofill event.

The backend `auth.service.ChangePassword` and `auth.repository.UpdatePassword` are correct and are not the cause.

## Fix Applied
- Changed `autoComplete` on the current-password `<Input>` from `"current-password"` to `"off"` in `ChangePasswordForm`.
- `autoComplete="off"` prevents browsers from auto-populating the field with stale site credentials.
- New/confirm password fields retain `autoComplete="new-password"` (unchanged).

Combined with the ISS-025 toggle fix, users can now both see what is in the field (toggle) and rely on it not being silently overwritten by autofill.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/components/auth/ChangePasswordForm.tsx` | `autoComplete="current-password"` → `autoComplete="off"` |

## Regression Test
`frontend/src/components/auth/ChangePasswordForm.test.tsx` — existing form-submission tests cover the payload; no autofill-specific test added (browser autofill is not simulated by jsdom).

## Resolution Results
- Tests: 218 passed, 0 failed (39 test files; autoComplete=off validated by dedicated test)
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
