# ISS-138 - FR-BB117 role management frontend (inner report)

Branch: swarm/138-roles-frontend. Refs #138 (backend: #137 / PR #185).

## Delivered
- `/admin/roles` (`pages/admin/roles/RolesPage.tsx`): table (name, description, System/Custom badge, users, permissions, actions), Create button, View (system) / Edit + Delete (custom), delete confirmation dialog, success status banner.
- `RoleFormDialog.tsx` + `PermissionMatrix.tsx`: grouped-by-resource checkbox matrix fed by `GET /roles/permissions`; roles:read, roles:manage, tenant:manage disabled; system roles fully read-only.
- `api/roles.ts`: `useRolesAdmin`, `usePermissionsCatalogue`, `useCreateRole`, `useUpdateRole`, `useDeleteRole` (shared `apiFetch`); invalidate `['roles']` and `['users','roles']`; `roleErrorKey()` maps error CODE to `roles.errors.*`.
- Guards (AC-16): `RequireRole` gains `permission` and `allowCustomRole`; `useMyPermissions` (custom roles only, GET /users/me, trusted only if `role_name` matches token role); Sidebar filters by permission for custom roles; `AdminHome` replaces the `/admin` -> dashboard redirect so a custom role lands on its first allowed page (or sees a notice) and cannot loop. Built-in roles use the unchanged role lists and never fetch /users/me. All `/admin/*` routes got `permission` props per the AC-16 mapping.
- `Dialog` (shared ui): focus trap + focus return on close (AC-17).
- i18n en/ru/kk: `nav.roles`, `roles.*` (876 keys, parity green).

## Tests
vitest: PermissionMatrix, RolesPage (list/system protection/create/edit/delete/error mapping), api/roles (error mapping), permissionGuards (RequireRole, Sidebar, AdminHome incl. custom role with users:read, error/stale cache deny, built-in unchanged). Full suite 590 passed, tsc and eslint clean, check:i18n green.

## Not verified
Live behaviour (no stack run per low-memory rules): see PR "UAT must check live". No e2e spec added.
