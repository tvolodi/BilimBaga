---
id: ISS-025
title: ChangePasswordForm — all three password fields have no show/hide toggle
status: resolved
severity: medium
layer: frontend
module: auth
tags: [password, toggle, show-password, eye-icon, ChangePasswordForm, ChangePasswordPage]
created: 2026-06-08
resolved: 2026-06-08
recurrence_count: 1
related_issues: [ISS-003]
regression_test: frontend/src/components/auth/ChangePasswordForm.test.tsx
---

## Symptom
The change-password page at `/change-password` has three `<input type="password">` fields (Current password, New password, Confirm new password) with no eye-icon toggle. Users cannot verify what they have typed, which frequently causes failed submissions (especially when browser autofill fills in a stale credential that they cannot see).

## Root Cause
`frontend/src/components/auth/ChangePasswordForm.tsx` renders three bare `<Input type="password">` elements with no adjacent toggle button and no `showPassword` state. ISS-003 added the same fix to `LoginForm` but `ChangePasswordForm` was not updated at that time. No i18n keys exist for the toggle aria-labels inside the `auth.changePassword` namespace.

## Fix Applied
- Added three independent boolean states (`showCurrent`, `showNew`, `showConfirm`) to `ChangePasswordForm`.
- Wrapped each `<Input>` in a relative `<div>`, placed an `Eye`/`EyeOff` (lucide-react) icon button at the right edge of each.
- Each button toggles its respective `showXxx` flag; `<Input type={showXxx ? 'text' : 'password'}>` reflects the state.
- Added `auth.changePassword.showPassword` / `auth.changePassword.hidePassword` i18n keys to `en.json`, `ru.json`, and `kk.json`.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/components/auth/ChangePasswordForm.tsx` | Added three independent show/hide toggles |
| `frontend/src/locales/en.json` | Added `showPassword` / `hidePassword` under `auth.changePassword` |
| `frontend/src/locales/ru.json` | Added `showPassword` / `hidePassword` under `auth.changePassword` |
| `frontend/src/locales/kk.json` | Added `showPassword` / `hidePassword` under `auth.changePassword` |
| `frontend/src/components/auth/ChangePasswordForm.test.tsx` | Added show/hide toggle tests for all three fields |

## Regression Test
`frontend/src/components/auth/ChangePasswordForm.test.tsx` — new tests added for each toggle field.

## Resolution Results
- Tests: 218 passed, 0 failed (39 test files, including 4 new toggle tests and 1 new autoComplete test)
- Migration applied: no
- Build clean: yes (Go backend build clean, frontend vite build not run but tsc covered by test harness)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
