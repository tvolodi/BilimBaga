# FR-BB317 — Department Treeview Selector

## Summary

Replace all flat `<Select>` department pickers across the admin UI with a reusable `DepartmentTreeSelect` component that renders the full department hierarchy as an interactive tree. Users can expand and collapse parent nodes, search by name, and select a leaf or branch department. The component is fully i18n'd, keyboard-accessible, and compatible with controlled form patterns (React Hook Form or plain `useState`).

---

## Background

BilimBaga's department model is hierarchical: each department may have a `parent_id` pointing to another department, and the `GET /api/v1/departments` endpoint already returns the full nested tree (`children` arrays). However, every department-picker in the current UI is a **flat HTML `<select>`** that iterates only the root items returned from `useDepartments()`, silently discarding any child departments from the visible options. As the number of departments grows and nesting deepens, operators cannot see the hierarchy, cannot distinguish same-name sibling departments under different parents, and cannot navigate the tree efficiently.

---

## Scope

The following screens currently use a flat department `<Select>` and must be updated:

| Screen | File | Usage |
|--------|------|-------|
| Create User drawer | `frontend/src/pages/users/UserCreateDrawer.tsx` | Assigns a department to a new user |
| Edit User drawer | `frontend/src/pages/users/UserEditDrawer.tsx` | Changes a user's department |
| Users List filter bar | `frontend/src/pages/admin/users/UsersListPage.tsx` | Filters the user table by department |
| Exam Wizard — Step 3 Assignments | `frontend/src/pages/ExamWizard/Step3Assignments.tsx` | Selects a department as exam assignee |

---

## Functional Requirements

### FR-BB317.1 — `DepartmentTreeSelect` Shared Component

> **Installation Prerequisite**: The `shadcn/ui Popover` component (`frontend/src/components/ui/popover.tsx`) is not currently installed in the project. Installing it via `npx shadcn@latest add popover` must be performed as part of this requirement's implementation before the component can be built.

A new reusable component `DepartmentTreeSelect` must be created at `frontend/src/components/DepartmentTreeSelect.tsx`.

**Props interface:**

```typescript
interface DepartmentTreeSelectProps {
  /** Currently selected department ID, or null for "none selected". */
  value: string | null
  /** Callback fired when the user selects a department node. */
  onChange: (id: string | null) => void
  /** Whether the field is disabled (e.g. form submission in progress). */
  disabled?: boolean
  /** Placeholder text when nothing is selected. */
  placeholder?: string
  /** Allow clearing the selection (renders an "unset" option). Default: true. */
  clearable?: boolean
  /** Additional CSS class on the trigger button. */
  className?: string
  /** aria-label for the trigger button (for accessibility). */
  'aria-label'?: string
}
```

**Behaviour:**

1. The component fetches the department tree via the existing `useDepartments()` React Query hook — no additional API calls.
2. It renders as a **popover** — a trigger button showing the selected department name (or placeholder) that opens a floating panel containing the tree.
3. The tree panel contains a **search/filter input** at the top. Typing narrows visible nodes to those whose names contain the query string (case-insensitive), automatically expanding ancestor nodes of matching results.
4. Each tree node renders with an expand/collapse toggle (chevron icon) when it has children, and indentation proportional to its depth.
5. Selecting a node calls `onChange(id)`, closes the popover, and displays the selected node's name in the trigger.
6. If `clearable` is true (default), a "clear" action is available either as an `×` button on the trigger or as a dedicated "None" item at the top of the tree.
7. The component uses **shadcn/ui `Popover`** for the floating panel and **shadcn/ui `Button`** for the trigger. Indentation and expand icons use Tailwind utilities and Lucide icons respectively (`ChevronRight`, `ChevronDown`).
8. The component must handle three states visibly:
   - **Loading**: while `useDepartments()` is fetching, the trigger is disabled with a spinner.
   - **Empty**: if the tree has no nodes, show a `t('departmentTree.empty')` message inside the panel.
   - **Error**: if the query errors, show a `t('departmentTree.error')` message inside the panel.

### FR-BB317.2 — User Create Drawer Integration

In `frontend/src/pages/users/UserCreateDrawer.tsx`, replace the `<Select>` department field with `<DepartmentTreeSelect>`.

- `value` bound to `form.department_id`
- `onChange` sets `form.department_id` (null when cleared)
- `disabled` set while `createUser.isPending`
- `placeholder` uses `t('users.form.department_placeholder')`

### FR-BB317.3 — User Edit Drawer Integration

In `frontend/src/pages/users/UserEditDrawer.tsx`, replace the `<Select>` department field with `<DepartmentTreeSelect>`.

- `value` bound to `form.department_id`
- `onChange` sets `form.department_id` (null when cleared)
- `disabled` set while `updateUser.isPending`
- `placeholder` uses `t('users.form.department_placeholder')`

### FR-BB317.4 — Users List Filter Bar Integration

In `frontend/src/pages/admin/users/UsersListPage.tsx`, replace the department filter `<Select>` with `<DepartmentTreeSelect>`.

- `value` bound to `filters.department_id ?? null`
- `onChange` calls `setFilter('department_id', id ?? '')` — clears the filter when `null`
- `clearable` must be `true` (filter can always be reset)
- `placeholder` uses `t('users.filters.department')`

### FR-BB317.5 — Exam Wizard Step 3 Department Assignee Integration

In `frontend/src/pages/ExamWizard/Step3Assignments.tsx`, when `pending.assignee_type === 'department'`, replace the flat `<Select>` with `<DepartmentTreeSelect>`.

- `value` bound to `pending.assignee_id || null`
- `onChange` sets `pending.assignee_id` (empty string when cleared, to align with existing `!pending.assignee_id` guard)
- `disabled` set while `addAssignment.isPending`
- `placeholder` uses `t('departmentTree.selectPlaceholder')`

### FR-BB317.5a — Fix `getAssigneeName()` to Resolve Nested Department Names

In `frontend/src/pages/ExamWizard/Step3Assignments.tsx`, the existing `getAssigneeName()` helper resolves assignment display names by calling `departments.find((d) => d.id === a.assignee_id)`. This only searches root nodes and will return a raw UUID string for any nested department selected via `DepartmentTreeSelect`.

A shared utility function `findDepartmentById` must be introduced (e.g. in `frontend/src/api/departments.ts` or a co-located util file) with the following signature:

```typescript
function findDepartmentById(nodes: Department[], id: string): Department | undefined
```

The function performs a recursive depth-first search through `nodes` and their nested `children` arrays, returning the first `Department` whose `id` matches the given `id`, or `undefined` if not found.

`getAssigneeName()` in `Step3Assignments.tsx` must be updated to call `findDepartmentById(departments, a.assignee_id)` instead of the flat `departments.find(...)`, so that both root-level and nested department assignments resolve to their human-readable names.

---

## Data Model

The `GET /api/v1/departments` endpoint returns an array of root-level `DepartmentNode` objects with nested `children`. The frontend `Department` type (already defined in `frontend/src/api/departments.ts`) mirrors this shape:

```typescript
interface Department {
  id: string           // UUID v4
  name: string
  parent_id: string | null
  created_at: string   // UTC ISO 8601
  updated_at: string   // UTC ISO 8601
  children: Department[]
}
```

`useDepartments()` returns `Department[]` (the root nodes). The component must recursively walk `children` to render the full tree.

---

## API Contract

No new endpoints are required.

| Endpoint | Method | Purpose | Already exists |
|----------|--------|---------|----------------|
| `/api/v1/departments` | GET | Returns full department tree as nested JSON | ✅ Yes |

The existing `useDepartments()` React Query hook (query key `['departments']`) must be used as-is. No new hooks are needed.

---

## UI/UX Specification

### Tree structure
- Root nodes are rendered at depth 0 (no indentation).
- Each level of nesting adds `pl-4` (1 rem) of padding.
- A `ChevronRight` icon precedes the node name for nodes with children; it rotates to `ChevronDown` when expanded.
- Leaf nodes (no children) render without a toggle icon, with the same indentation as sibling non-leaf nodes.

### Single-select behaviour
- Only one department may be selected at a time.
- The selected node is visually highlighted (e.g. `bg-accent` on its row).
- The trigger button displays the selected department name.
- If a parent node is selected, the trigger shows that parent's name — selecting a parent is allowed (a user or assignment can belong to a non-leaf department).

### Search / filter
- A text input at the top of the popover panel filters visible nodes.
- Nodes not matching the query and not ancestors of matches are hidden.
- Ancestor nodes of matches are expanded automatically and rendered even if they do not match the query themselves, so the matching descendants remain reachable.
- Clearing the search input restores the full tree.

### Empty / loading / error states
- **Loading**: trigger button is disabled; a `Loader2` spinner appears inside.
- **Empty tree**: popover panel shows `t('departmentTree.empty')`.
- **Query error**: popover panel shows `t('departmentTree.error')` with a warning icon.

### Keyboard navigation
- `Enter` / `Space` on the trigger opens the popover.
- `Escape` closes the popover without changing selection.
- Arrow keys (`Up` / `Down`) navigate between visible (non-collapsed) nodes.
- `Enter` on a node selects it and closes the popover.
- `ArrowRight` expands a collapsed parent; `ArrowLeft` collapses an expanded parent.
- Focus is trapped within the popover while it is open.

### Accessibility
- The trigger button has `role="combobox"`, `aria-haspopup="tree"`, `aria-expanded`, and supports a custom `aria-label` prop.
- The tree panel has `role="tree"`.
- Each node has `role="treeitem"`, `aria-expanded` (for parent nodes), and `aria-selected`.
- All icons are `aria-hidden="true"`.

---

## Acceptance Criteria

1. `DepartmentTreeSelect` exists at `frontend/src/components/DepartmentTreeSelect.tsx` and is exportable.
2. Opening the component's popover displays the department hierarchy as a nested, expandable tree; child departments are visible by expanding parent nodes.
3. Selecting any node (leaf or parent) closes the popover and shows that node's name in the trigger button.
4. Clicking the clear/reset action sets the value to `null`/empty and shows the placeholder text.
5. Typing in the search input hides non-matching nodes while keeping matching nodes and their ancestors visible.
6. The trigger is disabled and shows a loading indicator while `useDepartments()` is in a pending state.
7. An error state is displayed (with i18n text) if `useDepartments()` returns an error.
8. `UserCreateDrawer` — department field uses `DepartmentTreeSelect`; selecting a nested department submits its UUID to the API correctly.
9. `UserEditDrawer` — department field uses `DepartmentTreeSelect`; pre-selected value matches the user's current `department_id` on open.
10. `UsersListPage` — department filter uses `DepartmentTreeSelect`; selecting a department applies the filter; clearing resets it.
11. `ExamWizard Step 3` — when assignee type is `department`, the selector uses `DepartmentTreeSelect`; selecting a department enables the "Save" button.
12. All displayed strings are sourced from i18n locale files — no hardcoded English text in the component.
13. The component passes keyboard navigation: tree navigable by arrow keys, `Enter` selects, `Escape` closes.
14. The component has `role="tree"` / `role="treeitem"` ARIA attributes and does not produce any axe accessibility violations on the affected screens.
15a. All existing unit test suites pass with no test files removed or test cases deleted.
15b. Any test file that references the old department `<select>` element is updated to target `DepartmentTreeSelect` via `role='combobox'` or an explicit `data-testid`; overall test coverage for affected components must not decrease.

---

## i18n Keys Required

Add the following keys to all three locale files (`en.json`, `ru.json`, `kk.json`) under a new `"departmentTree"` namespace:

| Key | English value |
|-----|---------------|
| `departmentTree.selectPlaceholder` | `"Select department"` |
| `departmentTree.searchPlaceholder` | `"Search departments…"` |
| `departmentTree.empty` | `"No departments found"` |
| `departmentTree.error` | `"Failed to load departments"` |
| `departmentTree.clearSelection` | `"Clear selection"` |
| `departmentTree.expandNode` | `"Expand"` |
| `departmentTree.collapseNode` | `"Collapse"` |

> Note: `users.form.department_placeholder` and `users.filters.department` already exist and are reused where appropriate; only the new keys above need to be added.

---

## Out of Scope

- **Multi-select mode**: selecting multiple departments simultaneously is not required by this feature. The component is designed single-select only.
- **Editing department names from the picker**: the `DepartmentTreeSelect` is a read/select component only; creating, renaming, or deleting departments remains on the dedicated Departments admin page.
- **Backend changes**: no new API endpoints, migrations, or Go code changes are needed.
- **Employee portal**: the employee-facing exam portal does not expose department pickers; this requirement is admin-UI only.
- **Virtualization**: virtual scrolling for very large trees is out of scope for the initial implementation.

---

## Dependencies

| ID | Title | Status | Reason |
|----|-------|--------|--------|
| FR-BB17 | Department Management | Implemented | Department backend API (`GET /api/v1/departments`) and the hierarchical data model this component relies on |
| FR-BB113 | Frontend: Department Management | Draft | Confirms the `DepartmentNode` tree shape is already surfaced in the frontend |
| FR-BB62 | Full i18n Coverage | Implemented | Establishes the pattern for locale key additions this requirement follows |
