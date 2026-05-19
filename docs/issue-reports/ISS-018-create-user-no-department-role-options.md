---
id: ISS-018
title: Create/Edit User drawers show no Department or Role options
status: resolved
severity: high
layer: frontend
module: users
tags: [select, dropdown, department, role, useRoles, useDepartments]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/users/UserCreateDrawer.test.tsx
---

## Symptom

On Admin → Users → "Create User" drawer (and the "Edit User" drawer), the Department and Role fields show placeholder text ("Выберите отдел", "Выберите роль") but no selectable options. Users cannot pick a department or role when creating or editing a user account.

## Root Cause

Both `UserCreateDrawer.tsx` and `UserEditDrawer.tsx` used plain `<Input>` text fields for `department_id` and `role_id`. The inputs displayed the placeholder translation keys but never fetched or rendered actual options.

Additionally, no `useRoles()` hook existed in `frontend/src/api/users.ts`, so even if the UI had attempted to render a select, there was no way to fetch `GET /api/v1/users/roles`.

Backend endpoints were working correctly:
- `GET /api/v1/departments` — registered and guarded by `departments:read`
- `GET /api/v1/users/roles` — registered and guarded by `users:read`

The bug was purely in the frontend presentation layer.

## Fix Applied

1. **`frontend/src/api/users.ts`**: Added `RoleRow` interface (`{ id, name }`) and `useRoles()` React Query hook that calls `GET /api/v1/users/roles`.

2. **`frontend/src/pages/users/UserCreateDrawer.tsx`**:
   - Added imports for `Select` (shadcn/ui), `useRoles`, and `useDepartments`.
   - Called `useDepartments()` and `useRoles()` inside the component.
   - Replaced the department `<Input>` with a `<Select>` whose options are populated from the departments list.
   - Replaced the role `<Input>` with a `<Select>` whose options are populated from the roles list.

3. **`frontend/src/pages/users/UserEditDrawer.tsx`**: Same changes as `UserCreateDrawer` — both drawers had the same underlying defect.

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/api/users.ts` | Add `RoleRow` interface; add `useRoles()` hook |
| `frontend/src/pages/users/UserCreateDrawer.tsx` | Replace `<Input>` with `<Select>` for department and role; wire `useDepartments` + `useRoles` |
| `frontend/src/pages/users/UserEditDrawer.tsx` | Same as above |
| `frontend/src/pages/users/UserCreateDrawer.test.tsx` | New regression test (3 cases) |

## Regression Test

`frontend/src/pages/users/UserCreateDrawer.test.tsx` — 3 test cases:
1. `renders department options from GET /api/v1/departments` — verifies `Engineering` and `HR` options render.
2. `renders role options from GET /api/v1/users/roles` — verifies `employee`, `manager`, `admin` options render.
3. `shows placeholder options when API returns empty lists` — verifies at least 2 placeholder `<option>` elements render when both APIs return empty arrays.

## Resolution Results

- Tests: 3 passed, 0 failed
- Migration applied: no
- Build clean: yes (TypeScript: no errors)

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Initial report — Create User drawer shows empty selects | Fixed in this issue |
