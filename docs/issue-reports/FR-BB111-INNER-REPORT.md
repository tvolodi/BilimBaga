# FR-BB111: Implementation Inner Report

**Date**: 2026-05-14T00:00:00Z
**Pipeline**: A — Feature Development
**Commit**: (see below)

## Summary

Implemented the Admin Shell (FR-BB111), delivering the persistent navigation chrome — collapsible sidebar, top bar with role badge and logout, and route-aware breadcrumbs — that wraps every admin-facing page. Route guards enforce authentication and role-based access control so unauthenticated users are redirected to `/login` (with the original URL preserved) and employees are redirected to `/portal`. The first functional admin screens were delivered: a full-featured Users List page with column sorting and filter controls (department, role, status), User Create Drawer, User Edit Drawer, and a two-step CSV Bulk Import Modal. All user-visible strings are sourced from the i18n locale files (en, kk, ru) with no hardcoded text in components. The `usehooks-ts` library was added to support `useLocalStorage` for persisting sidebar collapsed state.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/api/departments.ts` | created |
| `frontend/src/layouts/AdminLayout.tsx` | created |
| `frontend/src/components/admin/Sidebar.tsx` | created |
| `frontend/src/components/admin/TopBar.tsx` | created |
| `frontend/src/components/admin/Breadcrumb.tsx` | created |
| `frontend/src/components/admin/RoleBadge.tsx` | created |
| `frontend/src/components/admin/StatusBadge.tsx` | created |
| `frontend/src/pages/admin/users/UsersListPage.tsx` | created |
| `frontend/src/pages/admin/users/UserCreateDrawer.tsx` | created |
| `frontend/src/pages/admin/users/UserEditDrawer.tsx` | created |
| `frontend/src/pages/admin/users/ImportModal.tsx` | created |
| `frontend/src/pages/admin/departments/DepartmentsPage.tsx` | created |
| `frontend/src/App.tsx` | modified — wired AdminLayout route, removed old AdminShell |
| `frontend/src/locales/en.json` | modified — added admin shell strings |
| `frontend/src/locales/kk.json` | modified — added admin shell strings |
| `frontend/src/locales/ru.json` | modified — added admin shell strings |
| `frontend/src/pages/AdminShell.tsx` | deleted — replaced by AdminLayout |
| `frontend/package.json` | modified — added usehooks-ts |
| `docs/requirements/FR-BB111.Frontend-admin-shell.md` | modified — status set to Validated |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC-1 | Sidebar.tsx renders collapsible nav with all 7 links; active link highlighted via NavLink |
| AC-2 | TopBar.tsx displays full_name, RoleBadge, logout calls POST /api/v1/auth/logout |
| AC-3 | Breadcrumb.tsx reflects route hierarchy from useMatches() |
| AC-4 | AdminLayout.tsx redirects to /login if no auth token; stores location.state.from |
| AC-5 | AdminLayout.tsx redirects employee role to /portal |
| AC-6 | UsersListPage.tsx renders table with Full Name, Email, Department, Role, Status, Actions; client-side sortable |
| AC-7 | Filter controls for Department, Role, Status pass query params to API via useUsers hook |
| AC-8 | UserCreateDrawer.tsx invalidates users list query on success |
| AC-9 | UserEditDrawer.tsx pre-fills full_name, department_id, role_id from selected user |
| AC-10 | ImportModal.tsx two-step CSV flow; invalidates users list on commit |
| AC-11 | useLocalStorage('sidebar-collapsed') in Sidebar.tsx |
| AC-12 | All strings from i18n locale files; zero hardcoded English text in component files |

## Test Results

- Backend: not applicable (frontend-only change)
- Frontend: TypeScript build passes (`npm run build` exit 0); no runtime test suite run (frontend tests are component tests requiring a browser runtime — deferred per project test strategy)

## Migration Applied

none

## Known Limitations

- DepartmentsPage.tsx is a shell placeholder; full CRUD implementation is scoped to FR-BB17.
- Frontend component tests (Vitest + Testing Library) are not yet set up for the admin shell components; this is tracked in the backlog.
