# FR-BB29 — Frontend: Tags Management

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB29 |
| Phase | 2 — Content Management |
| Priority | 2 |
| Status | Implemented |
| Depends On | FR-BB21, FR-BB111 |

## Description
Provides a GUI for the flat tag vocabulary used by questions (FR-BB22) and the question bank filter (FR-BB27). Tags are the secondary classification axis (categories being the primary, hierarchical one). Without a UI, the question editor's tag input can only attach pre-seeded tags or — if it auto-creates on submit — produces an uncurated pile of near-duplicates ("safety", "Safety", "safe-ty"). This requirement delivers a single management page where examiners can list, search, rename-by-recreating, and delete tags, with visibility into usage count.

Backend endpoints (FR-BB21, extended) are fully implemented:
- `GET    /api/v1/tags` — flat list, requires `tags:read` permission; each row includes `usage_count` (number of questions referencing the tag)
- `POST   /api/v1/tags` — body `{ name }`, requires `tags:manage` permission, returns `409` on duplicate name
- `PUT    /api/v1/tags/:id` — body `{ name }`, requires `tags:manage` permission; renames in place so existing `question_tags` links are preserved; returns `409` on duplicate name, `404` if missing, `400` on invalid name
- `DELETE /api/v1/tags/:id` — requires `tags:manage` permission, returns `409` if any question references the tag

## Acceptance Criteria
- [x] AC-1: Route `/admin/tags` (nested under `AdminLayout`) renders a sortable table with columns: Name, Usage count, Created at, Actions. Default sort is by name ascending.
- [x] AC-2: The Usage count column is sourced from the `usage_count` field on each row of `GET /api/v1/tags`. The column is sortable; clicking the header toggles asc/desc.
- [x] AC-3: A search input filters the visible tags client-side by name (case-insensitive substring); debounced 200 ms.
- [x] AC-4: A "New Tag" button opens a small modal with a single `name` field (required, max 64 chars — matches `tags.MaxTagNameLength` in the backend); submit calls `POST /api/v1/tags`. Duplicate names produce an inline error mapped from backend `ERR_TAG_DUPLICATE`.
- [x] AC-5: Each row's actions menu offers "Rename" and "Delete". Rename opens a modal pre-filled with the current name; submit calls `PUT /api/v1/tags/:id`. Renaming preserves all question links (a confirm sub-text in the modal explains this). Duplicate names produce an inline error.
- [x] AC-6: Delete uses a confirm dialog; on `409 Conflict` with `ERR_TAG_IN_USE`, the toast shows the usage count (already visible in the table row) and the row is not removed.
- [x] AC-7: The table is paginated client-side (default 50 rows per page; configurable 25/50/100). Backend pagination is **not** added in v1 because tag vocabularies are expected to stay under a few hundred items; this assumption is documented in Notes.
- [x] AC-8: The "Rename" and "Delete" menu items are hidden when the current user lacks `tags:manage` (department admins and employees see the list read-only).
- [x] AC-9: All user-visible strings come from `src/locales/{en,kk,ru}.json` under the `tags.*` namespace.
- [x] AC-10: A sidebar nav entry "Tags" (icon: `Hash`) is added in `Sidebar.tsx`, visible to all admin roles, linking to `/admin/tags`.
- [x] AC-11: The `useTags` query is reused by FR-BB27 (Question Bank filters) and FR-BB26 (Question editor tag picker); create / rename / delete mutations all invalidate the `['tags']` query so both consumers refresh.

## Technical Specification

### Frontend Components

```
src/
  pages/admin/tags/
    TagsPage.tsx
    TagCreateModal.tsx
    TagDeleteConfirm.tsx
  api/
    tags.ts                      # new file
```

### `src/api/tags.ts`

```typescript
export interface Tag {
  id: string;
  name: string;
  created_at: string;
  usage_count: number;
}

export function useTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: async () => (await apiFetch('/api/v1/tags')).data as Tag[],
    staleTime: 300_000,
  });
}

export function useCreateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch('/api/v1/tags', { method: 'POST', body: JSON.stringify({ name }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  });
}

export function useUpdateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      apiFetch(`/api/v1/tags/${id}`, { method: 'PUT', body: JSON.stringify({ name }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  });
}

export function useDeleteTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/v1/tags/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  });
}
```

### Error code → message mapping

| Backend error code | Locale key |
|--------------------|-----------|
| `ERR_TAG_DUPLICATE` | `tags.errors.duplicate` (inline on create / rename) |
| `ERR_TAG_IN_USE` | `tags.errors.inUse` (with `{{count}}` from the row's `usage_count`) |
| `ERR_INVALID_NAME` | `tags.errors.invalidName` (inline) |
| `ERR_NOT_FOUND` | `tags.errors.notFound` (toast — tag was deleted in another tab) |

### i18n Keys

```json
{
  "tags": {
    "title": "Tags",
    "columns": {
      "name": "Name",
      "usage": "Usage",
      "createdAt": "Created",
      "actions": "Actions"
    },
    "actions": {
      "new": "New Tag",
      "rename": "Rename",
      "delete": "Delete"
    },
    "form": {
      "name": "Tag name",
      "nameHelp": "Lowercase, max 64 characters"
    },
    "search": "Search tags",
    "renameModal": {
      "title": "Rename tag",
      "description": "All questions using this tag will keep the link."
    },
    "deleteConfirm": {
      "title": "Delete tag",
      "description": "This cannot be undone. The tag will be removed from all questions that currently use it."
    },
    "errors": {
      "duplicate": "A tag with this name already exists",
      "inUse": "Cannot delete: {{count}} question(s) still use this tag",
      "invalidName": "Name is required (max 64 characters)",
      "notFound": "Tag no longer exists"
    },
    "empty": "No tags yet. Click \"New Tag\" to start."
  }
}
```

Provide localised values for `kk` and `ru`.

## Notes
- The 64-char limit comes from `tags.MaxTagNameLength` in `backend/internal/tags/types.go`. Keep the frontend `maxLength` attribute in sync.
- The backend trims and lowercases tag names server-side via `tags.NormalizeName`. The frontend should still display the canonical (already-normalised) form returned by the API rather than echoing the typed input — so the rename modal field is populated from the API response, not the user's last-typed value.
- `usage_count` is computed by `GET /api/v1/tags` via a `LEFT JOIN question_tags` aggregation; it returns `0` when the `question_tags` table is not yet provisioned, so the column is safe to render unconditionally.
- Audit entries are written for `tag.create`, `tag.update`, and `tag.delete` automatically by the backend — the frontend has no audit responsibility.

## Out of Scope
Tag merge / dedup, tag-level RBAC, AI-suggested tagging — all deferred.

## Test Strategy
- **Unit**: search debounce filters the table without re-fetching; rename and delete dialogs only fire mutations on confirm; `usage_count` column sorts correctly.
- **Integration**: MSW-mocked `409 ERR_TAG_IN_USE` shows the usage-count toast; `409 ERR_TAG_DUPLICATE` on both create and rename shows inline form error.
- **E2E**: Create → search → rename (verify a question that referenced the old tag still references the new name) → delete (success); attempt to delete a tag attached to a question (409 toast).
