# FR-BB114 — Frontend: Audit Log Viewer (Phase 1)

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB114 |
| Phase | 1 — Foundation |
| Priority | 4 |
| Status | Draft |
| Depends On | FR-BB19, FR-BB111 |
| Supersedes (partial) | Pulls forward the basic viewer portion of FR-BB55; FR-BB55 remains for advanced filtering/correlation in Phase 5 |

## Description
Provides a minimal but production-ready GUI for the audit log so super admins can inspect "who did what" from day one. Every audit-worthy backend mutation already writes to `audit_log` (FR-BB19), and the `GET /api/v1/audit` and `GET /api/v1/audit/export` endpoints are implemented and tested — but there is currently no GUI, so the data is unreachable without direct database access. This requirement delivers the table, filters, and CSV export trigger; the richer cross-entity correlation, diff viewer, and saved-search features remain in FR-BB55 (Phase 5).

Backend endpoints (FR-BB19) are fully implemented:
- `GET /api/v1/audit?actor_id=&action=&entity_type=&from=&to=` — paginated list, `audit:read` permission
- `GET /api/v1/audit/export?…same filters…` — CSV download, capped at 10 000 rows, `audit:read` permission

## Acceptance Criteria
- [ ] AC-1: Route `/admin/audit` (nested under `AdminLayout`, guarded by `RequireRole roles={['super_admin']}`) renders a data table with columns: When (relative + absolute tooltip), Actor (user display name + role badge; "system" if `actor_id` is null), Action (e.g. `user.create`), Entity (type + truncated ID with copy-on-click), IP, and a "View metadata" expand button.
- [ ] AC-2: Clicking "View metadata" expands the row to show pretty-printed JSON of the `metadata` field (using a code block with monospace font); only one row is expanded at a time.
- [ ] AC-3: The filter bar provides: Actor (user picker — searches users via existing `useUsers` hook, returns `actor_id`), Action (free-text input — exact match), Entity Type (select: `user`, `department`, `category`, `tag`, `question`, `tenant`), and a date range picker that maps to `from`/`to` query params (ISO 8601 UTC, inclusive on both ends).
- [ ] AC-4: Filter state is reflected in the URL (`?actor_id=&action=&entity_type=&from=&to=&page=`) so a filtered view is shareable; navigating away and back restores the filters.
- [ ] AC-5: Pagination uses 50 rows per page; the table shows page N of M and supports prev/next; React Query key includes the full filter object.
- [ ] AC-6: An "Export CSV" button in the page header triggers a download by navigating to `GET /api/v1/audit/export?…` with the current filter set as query params (no extra confirmation dialog). The download attaches the `Authorization: Bearer` token via a fetch-then-blob flow (since `<a download>` cannot set headers).
- [ ] AC-7: When fewer than 1 page of results is returned, the page size selector is hidden; when zero results match, an empty state is shown with a "Clear filters" button.
- [ ] AC-8: All user-visible strings come from `src/locales/{en,kk,ru}.json` under the `audit.*` namespace; `action` values themselves are displayed verbatim (they are stable codes like `user.create`, not translated).
- [ ] AC-9: The sidebar in `FR-BB111` is extended with a new "Audit" nav item visible only when `user.role_name === 'super_admin'`; the item links to `/admin/audit`.
- [ ] AC-10: All non-mutating; this page is read-only. No row action can mutate audit entries.

## Technical Specification

### Frontend Components

```
src/
  pages/admin/audit/
    AuditLogPage.tsx
    AuditFilterBar.tsx
    AuditTable.tsx
    AuditRow.tsx
    AuditMetadataPanel.tsx       # pretty-printed JSON
  api/
    audit.ts                     # new file: useAuditEntries, downloadAuditCsv
```

### `src/api/audit.ts`

```typescript
export interface AuditEntry {
  id: string;
  actor_id: string | null;
  action: string;
  entity_type: string | null;
  entity_id: string | null;
  ip: string;
  metadata: unknown;
  created_at: string; // ISO 8601 UTC
}

export interface AuditFilters {
  actor_id?: string;
  action?: string;
  entity_type?: string;
  from?: string; // ISO 8601
  to?: string;   // ISO 8601
  page?: number;
  per_page?: number;
}

export function useAuditEntries(filters: AuditFilters) {
  return useQuery({
    queryKey: ['audit', filters],
    queryFn: async () => {
      const qs = new URLSearchParams(
        Object.entries(filters)
          .filter(([, v]) => v !== undefined && v !== '')
          .map(([k, v]) => [k, String(v)])
      );
      const res = await apiFetch(`/api/v1/audit?${qs}`);
      return res.data as { items: AuditEntry[]; meta: { total: number } };
    },
    staleTime: 0, // audit data should always be fresh
  });
}

// CSV export: fetch-then-blob to attach the Authorization header.
export async function downloadAuditCsv(filters: AuditFilters) {
  const qs = new URLSearchParams(
    Object.entries(filters)
      .filter(([, v]) => v !== undefined && v !== '')
      .map(([k, v]) => [k, String(v)])
  );
  const res = await fetch(`/api/v1/audit/export?${qs}`, {
    headers: { Authorization: `Bearer ${getAccessToken()}` },
  });
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `audit-${new Date().toISOString()}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}
```

### i18n Keys (new — append to each locale)

```json
{
  "audit": {
    "title": "Audit Log",
    "columns": {
      "when": "When",
      "actor": "Actor",
      "action": "Action",
      "entity": "Entity",
      "ip": "IP",
      "metadata": "Metadata"
    },
    "filters": {
      "actor": "Actor",
      "action": "Action",
      "entityType": "Entity type",
      "from": "From",
      "to": "To",
      "clear": "Clear filters"
    },
    "entityTypes": {
      "user": "User",
      "department": "Department",
      "category": "Category",
      "tag": "Tag",
      "question": "Question",
      "tenant": "Tenant"
    },
    "actor": {
      "system": "system"
    },
    "actions": {
      "exportCsv": "Export CSV",
      "viewMetadata": "View metadata"
    },
    "empty": {
      "title": "No audit entries match",
      "description": "Try widening the date range or clearing filters."
    }
  }
}
```

Provide localised values for `kk` and `ru` in the same shape.

### Sidebar nav addition (in `src/components/admin/Sidebar.tsx`)

```typescript
// Insert into NAV_ITEMS — visibility filtered downstream by role
{ key: 'audit', icon: ScrollText, path: '/admin/audit', labelKey: 'nav.audit', roles: ['super_admin'] },
```

Add `nav.audit` translation key to all three locales.

## Notes
- This is a **simple viewer**, not a full forensic tool. FR-BB55 will add: cross-entity timeline view, diff rendering for before/after metadata, saved searches, and tenant-wide aggregation. Both requirements are intentionally on the roadmap.
- `metadata` is `json.RawMessage` on the backend; the frontend pretty-prints it with `JSON.stringify(value, null, 2)` inside a `<pre>` block.
- The audit log can grow large; the 50-rows-per-page default plus the backend's 10 000-row CSV cap is intentionally conservative.
- `actor_id` is nullable on the backend (system actions); when null the table shows "system" and the actor filter is bypassed.

## Out of Scope
Diff viewer for before/after state, saved searches, multi-tenant aggregation (super-tenant view), real-time tail / SSE updates — all deferred to FR-BB55.

## Test Strategy
- **Unit**: `AuditTable` renders with mixed `actor_id = null` and populated rows; metadata expand toggles correctly.
- **Integration**: Filter state ↔ URL query params via `useSearchParams`; CSV export triggers a `fetch` to `/api/v1/audit/export` with the current filters.
- **E2E**: Filter by entity_type=`user` → table reduces; click "Export CSV" → file downloads; non–super-admin admin-area user navigating to `/admin/audit` is redirected to `/login`; an `employee` is redirected to `/portal` (FR-BB111 AC-5).
