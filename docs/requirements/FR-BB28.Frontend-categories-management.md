# FR-BB28 — Frontend: Categories Management

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB28 |
| Phase | 2 — Content Management |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB21, FR-BB111 |

## Description
Provides a GUI for the hierarchical category taxonomy that every question is tagged with. Categories are the primary navigation axis of the Question Bank (FR-BB27), so without a way to create/rename/reorder/delete them via the UI, the question editor's category picker is useless beyond whatever was seeded at install time. This requirement covers a tree view with full CRUD, drag-and-drop sort within a parent, and protective behaviour when a category is in use by questions.

Backend endpoints (FR-BB21) are fully implemented:
- `GET    /api/v1/categories` — full tree, available to any authenticated user
- `POST   /api/v1/categories` — body `{ name, parent_id?: string|null, track?: string|null, sort_order: int }`, `categories:manage` permission (`examiner+` and `super_admin`)
- `PUT    /api/v1/categories/:id` — body supports `{ name?, parent_id?, track?, sort_order?, clear_parent?: bool }` for partial updates; `clear_parent: true` forces `parent_id` to NULL
- `DELETE /api/v1/categories/:id` — returns `409 Conflict` with `CATEGORY_IN_USE` if any question references it

## Acceptance Criteria
- [ ] AC-1: Route `/admin/categories` (nested under `AdminLayout`) renders the full category tree from `GET /api/v1/categories`; each node shows the name, optional `track` label as a small badge, and child count.
- [ ] AC-2: Nodes are sorted by `sort_order` ascending, then by `name` ascending as a tie-breaker; the tree supports collapse/expand at every level.
- [ ] AC-3: A "New Category" button opens a modal with fields: `name` (required, max 100 chars), `parent_id` (category picker; empty = top-level), `track` (free text, optional; max 50 chars), `sort_order` (integer, default 0); submit calls `POST /api/v1/categories` and invalidates the `['categories']` query.
- [ ] AC-4: Each row's actions menu offers "Add child", "Edit" (opens modal with all four fields pre-filled — name, parent, track, sort_order), and "Delete".
- [ ] AC-5: The Edit modal supports moving a category to a different parent (including to top level via a "Top level" option that sends `clear_parent: true`). On submit, only the changed fields are sent in the PUT body.
- [ ] AC-6: Drag-and-drop reorder is supported within the same parent: dragging a node onto another sibling triggers `PUT /api/v1/categories/:id` with the new `sort_order`; cross-parent drag is **disabled** in v1 (cross-parent moves go through the Edit modal).
- [ ] AC-7: Delete uses a confirm dialog. On `409 Conflict` with `CATEGORY_IN_USE`, the toast shows the count of questions still using it (if returned by the backend in `error.details.in_use_count`) and the row is not removed.
- [ ] AC-8: On `409 Conflict` with `CATEGORY_CYCLE` (attempt to set a parent that would create a cycle), the parent picker shows an inline validation error.
- [ ] AC-9: Action menu items "Add child", "Edit", and "Delete" are hidden when the current user lacks `categories:manage` (use `useMe()` and check `permissions`); non-authors see the tree read-only.
- [ ] AC-10: All user-visible strings come from `src/locales/{en,kk,ru}.json` under the `categories.*` namespace.

## Technical Specification

### Frontend Components

```
src/
  pages/admin/categories/
    CategoriesPage.tsx
    CategoryEditModal.tsx        # serves both create and edit
    CategoryDeleteConfirm.tsx
  components/admin/categories/
    CategoryTree.tsx             # recursive with drag-and-drop wrapper
    CategoryTreeNode.tsx
  api/
    categories.ts                # new file
```

### `src/api/categories.ts`

```typescript
export interface CategoryNode {
  id: string;
  name: string;
  parent_id: string | null;
  track: string | null;
  sort_order: number;
  created_at: string;
  updated_at: string;
  children: CategoryNode[];
}

export interface CategoryCreate {
  name: string;
  parent_id?: string | null;
  track?: string | null;
  sort_order: number;
}

export interface CategoryUpdate {
  name?: string;
  parent_id?: string | null;
  track?: string | null;
  sort_order?: number;
  clear_parent?: boolean;
}

export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: async () => (await apiFetch('/api/v1/categories')).data as CategoryNode[],
    staleTime: 300_000,
  });
}

export function useCreateCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CategoryCreate) =>
      apiFetch('/api/v1/categories', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  });
}

export function useUpdateCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: CategoryUpdate }) =>
      apiFetch(`/api/v1/categories/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  });
}

export function useDeleteCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/v1/categories/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  });
}
```

### Drag-and-drop

Use `@dnd-kit/core` + `@dnd-kit/sortable` (already used in many React 18 codebases; if not yet installed, add via `npm install @dnd-kit/core @dnd-kit/sortable`). Constrain drag to siblings under the same parent (`SortableContext` per child list). On drag-end, compute the new `sort_order` as the average of the surrounding siblings' values to avoid renumbering, or fall back to a full renumber when collisions occur.

### Error code → message mapping

| Backend error code | Locale key |
|--------------------|-----------|
| `CATEGORY_IN_USE` | `categories.errors.inUse` (with `{{count}}` interpolation) |
| `CATEGORY_CYCLE` | `categories.errors.cycle` (inline on parent picker) |
| `PARENT_NOT_FOUND` | `categories.errors.parentNotFound` |
| `INVALID_NAME` | `categories.errors.invalidName` (inline) |

### i18n Keys

```json
{
  "categories": {
    "title": "Categories",
    "actions": {
      "new": "New Category",
      "addChild": "Add child",
      "edit": "Edit",
      "delete": "Delete"
    },
    "form": {
      "name": "Name",
      "parent": "Parent category",
      "parentTopLevel": "(Top level)",
      "track": "Track",
      "trackHelp": "Optional grouping label (e.g. \"compliance\", \"safety\")",
      "sortOrder": "Sort order"
    },
    "deleteConfirm": {
      "title": "Delete category",
      "description": "This cannot be undone."
    },
    "errors": {
      "inUse": "Cannot delete: {{count}} question(s) still use this category",
      "cycle": "Cannot set this parent — it would create a cycle",
      "parentNotFound": "Selected parent no longer exists",
      "invalidName": "Name is required (max 100 characters)"
    },
    "empty": "No categories yet. Click \"New Category\" to start."
  }
}
```

Provide localised values for `kk` and `ru` in the same shape. Add a `nav.categories` key and a sidebar entry (icon: `FolderTree`) so the page is reachable; visibility filtered to users with `categories:read` (everyone) but the actions are gated by `categories:manage`.

## Notes
- The `track` field is a freeform grouping label used by Phase 7 AI assistance; in v1 it's just a typed string with a list of recently-used suggestions in an autocomplete.
- The category picker inside the Edit modal must **exclude the category being edited and its descendants** from the parent options (preventing cycles locally before hitting the API).
- React Query `staleTime: 300_000` (5 min) for `useCategories` matches the same value used by `useTags` in FR-BB27.

## Out of Scope
Cross-parent drag-and-drop (do it via Edit), bulk delete, category-level RBAC overrides, AI-suggested category tagging — all deferred.

## Test Strategy
- **Unit**: `CategoryTree` sorts children by `sort_order` then `name`; parent picker in Edit modal correctly excludes self + descendants.
- **Integration**: Drag-and-drop sort triggers a single `PUT` with the new `sort_order`; cross-parent drag is rejected.
- **E2E**: Create top-level → add child → reorder siblings → reparent via Edit modal → delete unused → attempt delete of in-use category (409 toast).
