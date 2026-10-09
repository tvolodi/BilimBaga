---
id: ISS-053
title: Users List department filter is a flat select that ignores the department tree
status: resolved
severity: low
layer: frontend
module: users
tags: [DepartmentTreeSelect, FR-BB317, UsersListPage, department_id]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/admin/users/UsersListPage.test.tsx
---

## Symptom
`/admin/users` department filter (`frontend/src/pages/admin/users/UsersListPage.tsx`) was a native `<Select>` listing only root departments; child departments were not selectable. FR-BB317.4 requires `DepartmentTreeSelect`. Found by BA drift check (GitHub #40).

## Root Cause
FR-BB317 integration replaced the pickers in the drawers and Exam Wizard Step 3 (and the legacy `pages/users/UsersListPage.tsx`), but the routed admin page was missed.

## Fix Applied
Replaced the `<Select>` with `<DepartmentTreeSelect>` (value `filters.department_id ?? null`, onChange `setFilter('department_id', id ?? '')`, clearable, placeholder and aria-label `users.filters.department`). Removed the now-unused `useDepartments` import/call. No new i18n keys needed (check:i18n green).

## Files Changed
| File | Change |
|------|--------|
| frontend/src/pages/admin/users/UsersListPage.tsx | Use DepartmentTreeSelect for department filter |
| frontend/src/pages/admin/users/UsersListPage.test.tsx | New tests |

## Regression Test
`frontend/src/pages/admin/users/UsersListPage.test.tsx`: tree combobox rendered (not native select); nested child hidden until expanded/searched, selecting it sends `department_id` to the API; clearing removes the param.

## Resolution Results
- Tests: 319 passed, 0 failed (56 files); tsc, lint, check:i18n clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
