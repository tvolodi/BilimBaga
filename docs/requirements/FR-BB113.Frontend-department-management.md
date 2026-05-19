# FR-BB113 — Frontend: Department Management

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB113 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Validated |
| Depends On | FR-BB17, FR-BB111 |

## Description
Replaces the current `DepartmentsPage.tsx` "Coming Soon" placeholder with a fully functional GUI for the department hierarchy. Super admins can create, rename, and delete departments in a nested tree view. This is currently the only way (apart from direct SQL) for an operator to manage the org structure that every user record references via `department_id`; without it the User Create/Edit drawer department picker shows only departments seeded at install time.

Backend endpoints (FR-BB17) are fully implemented:
- `GET    /api/v1/departments` — returns full nested tree; available to any authenticated user
- `POST   /api/v1/departments` — body `{ name, parent_id?: string|null }`; `super_admin` only
- `PUT    /api/v1/departments/:id` — body `{ name }`; `super_admin` only (reparenting not supported by backend)
- `DELETE /api/v1/departments/:id` — `super_admin` only; returns `409` if department has users (`DEPARTMENT_NOT_EMPTY`) or child departments (`DEPARTMENT_HAS_CHILDREN`)

## Acceptance Criteria
- [ ] AC-1: Route `/admin/departments` (nested under `AdminLayout`) renders the full department tree returned by `GET /api/v1/departments`; each node shows the department name, a child count badge (hidden when 0), and a row actions menu.
- [ ] AC-2: The tree supports collapse/expand at every level; collapsed/expanded state persists in component state for the lifetime of the page (no `localStorage` needed).
- [ ] AC-3: A "New Department" button in the page header opens a modal with fields `name` (text, required, max 100 chars) and `parent_id` (department select, optional — empty = top-level); submit calls `POST /api/v1/departments`; on success: modal closes, `['departments']` query is invalidated, a success notification is shown.
- [ ] AC-4: Each row's actions menu offers "Add child" (opens the same create modal with `parent_id` pre-filled to that node), "Rename" (opens a rename modal calling `PUT /api/v1/departments/:id`), and "Delete" (opens a confirm dialog calling `DELETE /api/v1/departments/:id`).
- [ ] AC-5: On `409 Conflict` from delete, a notification surfaces the backend reason: `DEPARTMENT_NOT_EMPTY` → i18n key `departments.errors.notEmpty`; `DEPARTMENT_HAS_CHILDREN` → `departments.errors.hasChildren`. The tree is not mutated.
- [ ] AC-6: On `409 Conflict` from create/rename with `DUPLICATE_NAME`, the name field shows an inline validation error (`departments.errors.duplicateName`); no toast is shown for this case.
- [ ] AC-7: Action menu items "Add child", "Rename", and "Delete" are hidden from the UI when `useMe().role_name !== 'super_admin'`. Non–super-admins see the tree read-only.
- [ ] AC-8: All user-visible strings come from `src/locales/{en,kk,ru}.json` under the `departments.*` namespace; zero hardcoded strings in components.
- [ ] AC-9: The existing `DepartmentsPage.tsx` placeholder is replaced in-place (same file path); the route in `App.tsx` requires no changes.
- [ ] AC-10: The User Create/Edit drawer's department dropdown continues to work — `useDepartments()` is reused by the drawer, and creating/deleting a department invalidates the same `['departments']` query key so the picker refreshes without a page reload.
- [ ] AC-11: `DepartmentsPage.test.tsx` is replaced with tests that: (a) render a mocked 2-level tree and assert node names appear; (b) assert the "New Department" button is absent for non-super-admin users; (c) assert the "New Department" button is present for `super_admin`. Each modal component (`DepartmentCreateModal`, `DepartmentRenameModal`, `DepartmentDeleteConfirm`) has its own `.test.tsx` file with at least one passing test.

## Technical Specification

### No Database Schema Changes
The departments table is fully implemented (FR-BB17). No migrations needed.

### API Contract (already implemented — reference only)

**GET /api/v1/departments**
- Auth: any authenticated user (Bearer JWT)
- Response `200`: `{ data: DepartmentNode[], error: null }`
  ```json
  [{ "id": "uuid", "name": "string", "parent_id": null, "created_at": "ISO8601", "updated_at": "ISO8601", "children": [...] }]
  ```

**POST /api/v1/departments**
- Auth: `super_admin`
- Body: `{ "name": "string", "parent_id": "uuid|null" }`
- Response `201`: `{ data: DepartmentNode, error: null }`
- Errors: `400 INVALID_BODY`, `404 NOT_FOUND` (parent), `409 DUPLICATE_NAME`

**PUT /api/v1/departments/:id**
- Auth: `super_admin`
- Body: `{ "name": "string" }`
- Response `200`: `{ data: DepartmentNode, error: null }`
- Errors: `400 INVALID_BODY`, `404 NOT_FOUND`, `409 DUPLICATE_NAME`

**DELETE /api/v1/departments/:id**
- Auth: `super_admin`
- Response `204` (no body)
- Errors: `404 NOT_FOUND`, `409 DEPARTMENT_HAS_CHILDREN`, `409 DEPARTMENT_NOT_EMPTY`

### Frontend Implementation Notes

#### Component Structure

```
src/
  pages/admin/departments/
    DepartmentsPage.tsx               ← replace existing placeholder
    DepartmentsPage.test.tsx          ← replace existing stub tests
    DepartmentCreateModal.tsx         ← new; used for "New Department" and "Add child"
    DepartmentRenameModal.tsx         ← new
    DepartmentDeleteConfirm.tsx       ← new
  components/admin/
    DepartmentTree.tsx                ← new; recursive tree renderer
    DepartmentTreeNode.tsx            ← new; single row + actions menu
```

Follow the same file-per-modal pattern as `frontend/src/pages/admin/tags/` (TagCreateModal, TagRenameModal, TagDeleteConfirm).

#### `src/api/departments.ts` — full replacement

The current file has a local `apiFetch` that only supports GET and does not accept method/body. Replace the entire file:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

export interface Department {
  id: string
  name: string
  parent_id: string | null
  created_at: string
  updated_at: string
  children: Department[]
}

export function useDepartments() {
  const qc = useQueryClient()
  return useQuery<Department[], Error>({
    queryKey: ['departments'],
    queryFn: () => apiFetch<Department[]>(qc, '/api/v1/departments'),
  })
}

export function useCreateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; parent_id?: string | null }) =>
      apiFetch<Department>(qc, '/api/v1/departments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  })
}

export function useUpdateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      apiFetch<Department>(qc, `/api/v1/departments/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  })
}

export function useDeleteDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(qc, `/api/v1/departments/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  })
}
```

#### `DepartmentsPage.tsx` structure

Follow the `TagsPage.tsx` pattern:
- `useMe()` for role check (`canManage = role_name === 'super_admin'`)
- `useDepartments()` for data
- Local state: `createOpen`, `createParentId`, `renameTarget`, `deleteTarget`, `notification`
- Render a `<DepartmentTree>` component passing the root-level nodes, `canManage`, and action callbacks
- Show loading/error states identical to TagsPage
- Notification banner: reuse the inline `NotificationBanner` pattern from TagsPage

#### `DepartmentTree.tsx` / `DepartmentTreeNode.tsx`

- `DepartmentTree` maps root nodes → `DepartmentTreeNode`
- `DepartmentTreeNode` holds `expanded` local state; renders children recursively when expanded
- Indentation: `ml-4` per depth level
- Child count badge: `<span className="...">({children.length})</span>` hidden when `children.length === 0`
- Actions menu: same inline dropdown pattern as `ActionsMenu` in TagsPage; items: "Add child", "Rename", "Delete" (only rendered when `canManage`)

#### `DepartmentCreateModal.tsx`

- Props: `open`, `defaultParentId?: string | null`, `departments: Department[]`, `onClose()`
- Fields: `name` (Input, required), `parent_id` (native `<select>` listing all departments flat + "(Top level)" option)
- On submit: call `useCreateDepartment().mutateAsync()`; catch `DUPLICATE_NAME` → set inline field error; other errors → pass to parent via `onClose` or show notification

#### `DepartmentRenameModal.tsx`

- Props: `open`, `department: Department | null`, `onClose()`
- Single `name` field pre-filled with current name
- On submit: `useUpdateDepartment().mutateAsync()`; catch `DUPLICATE_NAME` → inline field error

#### `DepartmentDeleteConfirm.tsx`

- Props: `open`, `department: Department | null`, `isPending`, `onClose()`, `onConfirm()`
- Shows department name in confirmation text
- Uses `AlertDialog` from shadcn/ui (same as TagDeleteConfirm)

#### Error code → UI mapping

| Backend error code       | UI treatment |
|--------------------------|--------------|
| `DUPLICATE_NAME`         | Inline field error on `name` input |
| `DEPARTMENT_NOT_EMPTY`   | Notification banner (error) |
| `DEPARTMENT_HAS_CHILDREN`| Notification banner (error) |
| `NOT_FOUND`              | Notification banner (error) |

#### i18n Keys

Add to `src/locales/en.json` (replace the existing `"departments": "Departments"` nav string — move it to `"departments.title"` or keep `nav.departments` unchanged and add the `departments` namespace as a separate top-level key):

> **Note**: `nav.departments` already exists as a plain string. Add a new `departments` namespace key alongside it. Do NOT remove `nav.departments`.

```json
"departments": {
  "title": "Departments",
  "actions": {
    "new": "New Department",
    "addChild": "Add Child",
    "rename": "Rename",
    "delete": "Delete"
  },
  "form": {
    "name": "Name",
    "parent": "Parent Department",
    "parentTopLevel": "(Top level)"
  },
  "deleteConfirm": {
    "title": "Delete Department",
    "description": "This action cannot be undone. All users in this department must be reassigned first."
  },
  "errors": {
    "duplicateName": "A department with this name already exists at this level",
    "notEmpty": "Cannot delete: department has users assigned",
    "hasChildren": "Cannot delete: department has child departments",
    "parentNotFound": "Selected parent department no longer exists"
  },
  "empty": "No departments yet. Click \"New Department\" to get started.",
  "children": "children"
}
```

Provide equivalent translations for `kk.json` and `ru.json` in the same shape:

**ru.json**
```json
"departments": {
  "title": "Отделы",
  "actions": {
    "new": "Новый отдел",
    "addChild": "Добавить дочерний",
    "rename": "Переименовать",
    "delete": "Удалить"
  },
  "form": {
    "name": "Название",
    "parent": "Родительский отдел",
    "parentTopLevel": "(Верхний уровень)"
  },
  "deleteConfirm": {
    "title": "Удалить отдел",
    "description": "Это действие необратимо. Все пользователи этого отдела должны быть переназначены."
  },
  "errors": {
    "duplicateName": "Отдел с таким названием уже существует на этом уровне",
    "notEmpty": "Невозможно удалить: в отделе есть пользователи",
    "hasChildren": "Невозможно удалить: у отдела есть дочерние подразделения",
    "parentNotFound": "Выбранный родительский отдел больше не существует"
  },
  "empty": "Отделов пока нет. Нажмите «Новый отдел», чтобы начать.",
  "children": "дочерних"
}
```

**kk.json**
```json
"departments": {
  "title": "Бөлімдер",
  "actions": {
    "new": "Жаңа бөлім",
    "addChild": "Бала қосу",
    "rename": "Атауын өзгерту",
    "delete": "Жою"
  },
  "form": {
    "name": "Атауы",
    "parent": "Бас бөлім",
    "parentTopLevel": "(Жоғарғы деңгей)"
  },
  "deleteConfirm": {
    "title": "Бөлімді жою",
    "description": "Бұл әрекетті кері қайтару мүмкін емес. Бөлімдегі барлық пайдаланушыларды қайта тағайындау қажет."
  },
  "errors": {
    "duplicateName": "Бұл деңгейде осындай атаумен бөлім бар",
    "notEmpty": "Жою мүмкін емес: бөлімде пайдаланушылар бар",
    "hasChildren": "Жою мүмкін емес: бөлімде бала бөлімдер бар",
    "parentNotFound": "Таңдалған бас бөлім енді жоқ"
  },
  "empty": "Бөлімдер әзірше жоқ. Бастау үшін «Жаңа бөлім» түймесін басыңыз.",
  "children": "бала"
}
```

### Test Requirements

Replace `DepartmentsPage.test.tsx` with:

1. **Tree rendering test**: mock `useDepartments` to return a 2-level fixture; assert root node name visible; assert child node visible after clicking expand; assert child count badge shows correct number.
2. **Role-based visibility test**: mock `useMe` with `role_name: 'viewer'`; assert "New Department" button is absent; assert actions menu is absent.
3. **Super-admin visibility test**: mock `useMe` with `role_name: 'super_admin'`; assert "New Department" button is present.

For modals, write unit tests in each modal's own test file following the pattern in `TagCreateModal` (if it has one) or `BrandingSettingsPage.test.tsx`.

## Notes
- **Reparenting is out of scope**: the backend `PUT` endpoint only accepts `{ name }` — drag-and-drop or parent-change UI would require a backend change first.
- The `DepartmentCreateModal` "Add child" flow pre-fills `parent_id` but the user can change it (the modal serves both flows).
- The API tree response is already nested (`children: []`), so no client-side tree-building is needed.
- The `nav.departments` i18n key must NOT be removed — it is used by the sidebar navigation.
- The existing `useDepartments` query key `['departments']` is already used by `UserCreateDrawer` and `UserEditDrawer` — invalidating it on mutations automatically refreshes those dropdowns.
- The local `apiFetch` inside the current `departments.ts` must be removed; the shared `apiFetch` (used by other API modules) must be used instead so Authorization headers are handled consistently.

## Out of Scope
Drag-and-drop reparenting, department-level permission overrides, bulk import of departments, department-scoped audit log view, search/filter within the tree.
