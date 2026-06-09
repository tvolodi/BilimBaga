---
id: ISS-031
title: "UAT Defect: User Onboarding — Employee Portal has no logout button"
status: resolved
severity: medium
layer: frontend
module: portal
tags: [uat, portal, ux, logout]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/layouts/PortalLayout.test.tsx
---

## Symptom
UAT Scenario: `User Onboarding — UAT Scenario`, Scenario 1, Step 15
Actor: Employee
Action: Click "Logout" after completing forced password change and being redirected to Employee Portal
Expected: Employee is returned to the login page
Actual: No logout button exists in the Employee Portal. `PortalLayout` (`frontend/src/layouts/PortalLayout.tsx`) only contains tab navigation and a `LocaleSwitcher`. There is no Sign Out button or user menu. The employee has no way to log out from the portal via the UI.
Screenshot: test-results/uat-user-onboarding-202606-be0d7-sword-change-on-first-login-chromium-uat/test-failed-1.png

## Root Cause
`PortalLayout.tsx` was created with only tab navigation (My Exams, My Results) and a `LocaleSwitcher`. Unlike `AdminLayout`, which renders a `TopBar` component that includes a Sign Out button with `useLogout()`, `PortalLayout` was never given a logout affordance. The `useLogout` hook and `Button` component already existed and were working — they simply were not wired into the portal nav bar.

## Fix Applied
Added a Sign Out button inline in `PortalLayout.tsx` following the same pattern as `TopBar`:
- Imported `useNavigate` from `react-router-dom`, `LogOut` from `lucide-react`, `Button` from `@/components/ui/button`, and `useLogout` from `@/api/auth`.
- Added `handleLogout` function that calls `logout.mutateAsync()` then navigates to `/login` with `replace: true`.
- Wrapped `<LocaleSwitcher />` in a `<div className="flex items-center gap-3">` and added the `<Button>` with `aria-label={t('common.signOut')}` and `<LogOut>` icon to its right.
- The `common.signOut` i18n key was already present in all three locale files (en/ru/kk) — no locale changes needed.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/layouts/PortalLayout.tsx` | Added Sign Out button with `useLogout()` and navigate-to-login handler |
| `frontend/src/layouts/PortalLayout.test.tsx` | New test file: 3 tests covering nav render, button presence, and logout navigation |

## Regression Test
`frontend/src/layouts/PortalLayout.test.tsx` — 3 tests:
1. Renders portal navigation tabs
2. Renders a Sign Out button with correct `aria-label`
3. Calls `logout.mutateAsync()` and navigates to `/login` when Sign Out is clicked

## Resolution Results
- Tests: 233 passed, 0 failed (42 test files)
- Migration applied: no
- Build clean: yes
