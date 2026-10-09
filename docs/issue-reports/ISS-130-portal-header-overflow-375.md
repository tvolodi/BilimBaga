---
id: ISS-130
title: Employee portal header overflows horizontally at 375px in ru and kk
status: resolved
severity: medium
layer: frontend
module: users
tags: [PortalLayout, LocaleSwitcher, overflow, 375px, flex-wrap, FR-BB316]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/e2e/locale-switcher.spec.ts (AC6, live) + frontend/src/layouts/PortalLayout.test.tsx
---

## Symptom
At 375px, /portal scrollWidth is 408 (ru) / 457 (kk) vs 375 (en OK); language select and logout button pushed off-screen.

## Root Cause
PortalLayout nav was a non-wrapping flex row with px-6, tabs px-4, and a fixed 120px select; long ru/kk tab labels plus the right cluster exceeded the viewport.

## Fix Applied
Nav is `flex-wrap` with smaller gutters on mobile (px-2 sm:px-6), tabs px-2 sm:px-4 centered text, tab group `min-w-0 max-w-full`, right cluster `ml-auto shrink-0` with tighter gap, select `w-[104px] sm:w-[120px]`. sm: and above are unchanged from before.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/layouts/PortalLayout.tsx | wrap/shrink classes |
| frontend/src/components/LocaleSwitcher.tsx | narrower select below sm |
| frontend/src/layouts/PortalLayout.test.tsx | class-level regression test |

## Regression Test
Unit: class assertions. Live: e2e/locale-switcher.spec.ts AC6 (unchanged); not run here, UAT must re-run it live.

## Resolution Results
- Tests: 489 passed, 0 failed (vitest); tsc, lint, check:i18n clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
