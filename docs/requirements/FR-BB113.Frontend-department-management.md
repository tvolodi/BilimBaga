# FR-BB113 — Frontend: Department Management

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB113 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB17, FR-BB111 |

## Description
Replaces the current `DepartmentsPage.tsx` "Coming Soon" placeholder with a fully functional GUI for the department hierarchy. Super admins can create, rename, reparent, and delete departments in a nested tree view. This is currently the only way (apart from direct SQL) for an operator to manage the org structure that every user record references via `department_id`, and without it the existing `User Create Drawer` department picker shows only departments seeded at install time.

Backend endpoints (FR-BB17) are fully implemented:
- `GET    /api/v1/departments` — returns full tree, available to any authenticated user
- `POST   /api/v1/departments` — body `{ name, parent_id?: string|null }`, `super_admin` only
- `PUT    /api/v1/departments/:id` — body `{ name }`, `super_admin` only
- `DELETE /api/v1/departments/:id` — `super_admin` only; returns `409` if the department has users or children

## Acceptance Criteria
- [ ] AC-1: Route `/admin/departments` (nested under `AdminLayout`) renders the full department tree returned by `GET /api/v1/departments`; each node shows the name, child count badge, and a row actions menu.
- [ ] AC-2: The tree supports collapse/expand at every level; the collapsed/expanded state of each node persists across page navigation in component state (no persistence to `localStorage` required).
- [ ] AC-3: A "New Department" button in the page header opens a modal with fields `name` (text, required, max 100 chars) and `parent_id` (department picker, optional — empty = top-level); submit calls `POST /api/v1/departments`; success closes the modal, invalidates the `['departments']` query, and shows a success toast.
- [ ] AC-4: Each row's actions menu offers "Add child" (opens the same create modal with `parent_id` pre-filled), "Rename" (opens a small rename modal that calls `PUT /api/v1/departments/:id`), and "Delete" (opens a confirm dialog that calls `DELETE /api/v1/departments/:id`).
- [ ] AC-5: On `409 Conflict` from delete, the toast surfaces the backend reason: `DEPARTMENT_NOT_EMPTY` → "Cannot delete: department has users", `DEPARTMENT_HAS_CHILDREN` → "Cannot delete: department has child departments". The tree is not mutated.
- [ ] AC-6: On `409 Conflict` from create/rename with `DUPLICATE_NAME`, the relevant form field shows an inline validation error using the existing form-error pattern from `UserCreateDrawer`.
- [ ] AC-7: Action menu items "Add child", "Rename", and "Delete" are hidden from the UI when the current user's `role_name !== 'super_admin'` (the backend already enforces this; hiding is purely UX). Non–super-admins still see the tree in read-only form.
- [ ] AC-8: All user-visible strings come from `src/locales/{en,kk,ru}.json` under the `departments.*` namespace; zero hardcoded strings.
- [ ] AC-9: The existing `DepartmentsPage.tsx` placeholder file is replaced — not duplicated — and the route in `App.tsx` continues to point to `pages/admin/departments/DepartmentsPage.tsx`.
- [ ] AC-10: The User Create/Edit drawer's department dropdown continues to work — i.e. the `useDepartments` hook is reused, and creating a new department invalidates the same `['departments']` query so the picker refreshes without a page reload.

## Technical Specification

### Frontend Components

```
src/
  pages/admin/departments/
    DepartmentsPage.tsx          # replaces existing placeholder
    DepartmentCreateModal.tsx    # used for both "New" and "Add child"
    DepartmentRenameModal.tsx
    DepartmentDeleteConfirm.tsx
  components/admin/departments/
    DepartmentTree.tsx           # recursive renderer
    DepartmentTreeNode.tsx       # single node row with actions menu
  api/
    departments.ts               # extend existing file with mutations
```

### `src/api/departments.ts` — additions

```typescript
export function useCreateDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; parent_id?: string | null }) =>
      apiFetch('/api/v1/departments', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  });
}

export function useUpdateDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      apiFetch(`/api/v1/departments/${id}`, { method: 'PUT', body: JSON.stringify({ name }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  });
}

export function useDeleteDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/v1/departments/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  });
}
```

### Error code → toast mapping

| Backend error code | Toast key |
|--------------------|-----------|
| `DUPLICATE_NAME` | `departments.errors.duplicateName` (inline on form) |
| `DEPARTMENT_NOT_EMPTY` | `departments.errors.notEmpty` |
| `DEPARTMENT_HAS_CHILDREN` | `departments.errors.hasChildren` |
| `PARENT_NOT_FOUND` | `departments.errors.parentNotFound` |

### i18n Keys (new — append to each locale)

```json
{
  "departments": {
    "title": "Departments",
    "actions": {
      "new": "New Department",
      "addChild": "Add child",
      "rename": "Rename",
      "delete": "Delete"
    },
    "form": {
      "name": "Name",
      "parent": "Parent department",
      "parentTopLevel": "(Top level)"
    },
    "deleteConfirm": {
      "title": "Delete department",
      "description": "This cannot be undone."
    },
    "errors": {
      "duplicateName": "A department with this name already exists at this level",
      "notEmpty": "Cannot delete: department has users",
      "hasChildren": "Cannot delete: department has child departments",
      "parentNotFound": "Selected parent department no longer exists"
    },
    "empty": "No departments yet. Click \"New Department\" to start."
  }
}
```

Provide localised values for `kk` and `ru` in the same shape.

## Notes
- The tree response from the API is already nested (`children: []`), so no client-side aggregation is needed.
- Reparenting via drag-and-drop is **out of scope** for v1. The backend `PUT` endpoint only accepts `{ name }` per AC-4 of FR-BB17, so reparent would require backend changes first.
- The `RoleBadge`/`StatusBadge` styling from `FR-BB111` is the visual reference for the child-count badge.
- The "Add child" action pre-fills `parent_id` from the row but still lets the user change it in the modal so the same modal serves both flows.

## Out of Scope
Drag-and-drop reparenting, department-level permission overrides, bulk import of departments, and audit log surfaced inside the department page — all deferred.

## Test Strategy
- **Unit**: `DepartmentTree` renders a 3-level nested fixture; actions menu visibility responds to `useMe().role_name`.
- **Integration**: MSW-mocked `409 Conflict` with each error code produces the correct toast/inline error.
- **E2E**: Create top-level → create child → rename → delete child → delete parent (success); attempt to delete a department with a user → confirm 409 toast.
