---
id: ISS-003
title: Login form has no show/hide password toggle
status: resolved
severity: medium
layer: frontend
module: auth
tags: [password, login, toggle, show-password, eye-icon, LoginForm]
created: 2026-05-18
resolved: 2026-05-18
recurrence_count: 1
related_issues: []
regression_test: frontend/src/components/auth/LoginForm.test.tsx
---

## Symptom
The login form at `/login` always masks the password field. There is no eye icon or toggle button to reveal the typed password, so users cannot verify what they have entered and frequently fail authentication due to undetected typos.

## Root Cause
`frontend/src/components/auth/LoginForm.tsx` renders `<Input type="password" ...>` with no adjacent toggle button and no `showPassword` state. The component has no mechanism to switch between `type="password"` and `type="text"`. No i18n keys exist for the toggle's aria-label.

## Fix Applied
- Added `showPassword: boolean` state to `LoginForm`.
- Wrapped the password `<Input>` in a relative `<div>`, placed an icon `<button>` (Eye / EyeOff from `lucide-react`) at the right edge.
- Button toggles `showPassword`; `<Input type={showPassword ? 'text' : 'password'}>` reflects state.
- Added `auth.login.showPassword` / `auth.login.hidePassword` i18n keys in all three locales (en, ru, kk) for the button's `aria-label`.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/components/auth/LoginForm.tsx` | Added show/hide toggle with Eye/EyeOff icons |
| `frontend/src/locales/en.json` | Added `showPassword` / `hidePassword` keys |
| `frontend/src/locales/ru.json` | Added `showPassword` / `hidePassword` keys |
| `frontend/src/locales/kk.json` | Added `showPassword` / `hidePassword` keys |
| `frontend/src/components/auth/LoginForm.test.tsx` | Added toggle tests |

## Regression Test
`frontend/src/components/auth/LoginForm.test.tsx` — new tests: "toggles password visibility when eye icon clicked" and "eye button has accessible aria-label".

## Resolution Results
- Tests: 177 passed, 0 failed (all 32 test files green)
- Migration applied: no
- Build clean: yes (i18n check: all 659 keys present in all 3 locales)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
