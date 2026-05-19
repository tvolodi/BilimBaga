# ISS-003 Inner Report — Login Show/Hide Password Toggle

**Run ID:** iss-003
**Pipeline:** B (Bug Fix)
**Date:** 2026-05-18
**Status:** CLOSED

---

## Issue Summary

The login form had no way for users to reveal or conceal their password while typing.
This was a usability gap — users could not verify their typed password, leading to
unnecessary failed login attempts.

---

## Root Cause

`LoginForm.tsx` used a plain `<input type="password">` with no toggle control.
No show/hide button, no state variable for visibility, and no i18n keys existed for
the feature in any locale file.

---

## Fix Applied

**File:** `frontend/src/components/auth/LoginForm.tsx`

- Added `showPassword` boolean state (default `false`).
- Replaced static `type="password"` with dynamic `type={showPassword ? "text" : "password"}`.
- Added an icon-button inside the password field container (Eye / EyeOff from `lucide-react`)
  that toggles `showPassword` on click.
- Button is `type="button"` to prevent accidental form submission.
- `aria-label` sourced from i18n keys for full accessibility.

**Files:** `frontend/src/locales/en.json`, `ru.json`, `kk.json`

- Added `auth.showPassword` and `auth.hidePassword` keys to all three locales.

---

## Tests

**File:** `frontend/src/components/auth/LoginForm.test.tsx`

New test cases added:
- Password field initially has `type="password"`.
- Clicking the toggle button changes field to `type="text"`.
- Clicking toggle again changes field back to `type="password"`.
- Toggle button has correct accessible `aria-label` from i18n.

All 177 frontend tests pass (32 test files). No backend changes. No migrations.

---

## Commit

**Hash:** `0c8a5d3`
**Message:** `fix(auth): add show/hide password toggle to login form`

**Files changed (6):**
- `frontend/src/components/auth/LoginForm.tsx`
- `frontend/src/components/auth/LoginForm.test.tsx`
- `frontend/src/locales/en.json`
- `frontend/src/locales/ru.json`
- `frontend/src/locales/kk.json`
- `docs/issue-reports/ISS-003-login-no-show-password-toggle.md`

---

## Verification

| Check | Result |
|-------|--------|
| Frontend tests | 177 passed, 0 failed |
| Backend tests | Not applicable (no backend changes) |
| Migrations | None required |
| i18n key parity | 659 keys present in all 3 locales |
| Handoffs staged | No (excluded per Release Finalizer policy) |
