# FR-BB55 — Audit Log Viewer

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB55 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | uat-verified; amended 2026-10-11 (#544: AC-2 filter options come from the backend, AC-11 to AC-14), amendment validated (PASS) |
| Depends On | FR-BB19 |

## Description
Provides the admin-facing frontend for browsing, filtering, and exporting the audit log produced by FR-BB19. Examiners and HR admins can filter events by date range, actor, action type, and entity type. Each row is expandable to reveal the full metadata JSON. A CSV export button downloads the filtered result set. The backend audit log API (FR-BB19) is already implemented; this requirement covers the frontend UI and, if missing, the CSV export endpoint.

## Acceptance Criteria
- [ ] AC-1: The audit log page is accessible only to users with `role IN (examiner, hr_admin, super_admin)`; non-admin users are redirected.
- [ ] AC-2: The page displays a filter bar with: date range picker (from/to), free-text actor name search, action type multi-select dropdown and entity type filter dropdown, both populated from the backend (AC-11, AC-12; amended, it was a fixed list in the frontend that drifted from what the backend writes).
- [ ] AC-3: Filters are reflected in URL query parameters (`?from=&to=&actor=&action=&entity_type=`) so the view is bookmarkable and shareable.
- [ ] AC-4: The table shows at minimum: timestamp (formatted as local datetime), actor name (linked to user record), action code, entity type, entity ID (truncated to 8 chars with full value in tooltip), IP address.
- [ ] AC-5: Each row is expandable (click to toggle) to reveal the full metadata JSON rendered as a pretty-printed `<pre>` block.
- [ ] AC-6: Pagination renders 50 rows per page with Previous/Next controls and a total count label.
- [ ] AC-7: The "Export to CSV" button calls `GET /api/v1/audit/export` with the current filter params and triggers a browser download; the button shows a loading spinner while downloading.
- [ ] AC-8: The `GET /api/v1/audit/export` endpoint (add to FR-BB19's audit package if not present) streams a CSV with columns: `timestamp`, `actor_id`, `actor_name`, `action`, `entity_type`, `entity_id`, `ip_address`, `metadata_json`; requires examiner+ role.
- [ ] AC-9: The table uses `useQuery` with `queryKey: ['audit-log', filters, page]`; changing any filter resets to page 1.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.
- [ ] AC-11: Filter options come from the data. `GET /api/v1/audit/actions` (same authentication and permission as `GET /api/v1/audit`, same tenant scope) returns `{ data: { actions: string[], entity_types: string[] }, error: null }`: the distinct non-empty `action` and `entity_type` values of the caller's tenant in `audit_log`, sorted ascending. No new migration: a `SELECT DISTINCT action ... WHERE tenant_id = $1` (and the same for `entity_type`) is acceptable at the current volume; the dev notes the two query plans in the PR and adds an index only if a plan is poor. An empty log returns two empty arrays. Handler and repository tests cover the empty log, tenant isolation and sorting.
- [ ] AC-12: The frontend removes the constants `AUDIT_ACTIONS` (`api/audit.ts`) and `ENTITY_TYPES` (hard-coded in `AuditFilterBar.tsx`, which lacks real values such as `session_answer` and `exam_session`); the two dropdowns are filled by a `useAuditFilterOptions` query (`['audit-filter-options']`, `staleTime` five minutes). While it loads the dropdowns are disabled; if it fails the dropdowns stay disabled with an inline error and the log table still loads with the remaining filters. The `action` query parameters of AC-3 keep carrying raw action codes. `AuditFilterBar.test.tsx` (it imports `AUDIT_ACTIONS` and queries the options synchronously) is updated to mock the options query and wait for it.
- [ ] AC-13: Labels (recommendation, owner may drop this AC and keep raw codes). Each action code and each entity type is shown with a translated label: `audit.actions.<code with each "." replaced by "_">` and `audit.entityTypes.<type>` in `frontend/src/locales/{kk,ru,en}.json`. A code with no key is shown as the raw code (`defaultValue`), so a new backend action never breaks the page. In the table the raw code stays visible as secondary text and in the tooltip; the CSV export (AC-8) and the URL carry raw codes only. The dev adds keys for every action the backend writes at the time of the PR, found by `git grep` for `writer.Write(` and the five direct `INSERT INTO audit_log` sites outside `audit/writer.go` (four in `internal/sessions`, one in `internal/users`), with the entity types that occur (including `session_answer` and `exam_session`), and the existing key-parity check keeps kk, ru and en aligned.
- [ ] AC-14: The six entries of the old list that no backend code writes (`user.login`, `user.logout`, `session.start`, `session.expire`, `certificate.issue`, `tenant.update`) disappear with the constant, and so do the codes the backend does write but the list lacked (for example `auth.login.success`, `session.auto_submit`) because the options now come from the data; no replacement or alias is added. Action naming of the backend is not changed by this requirement (some codes use a plural or a different prefix, for example `users.unlock`, `tenant_config.update`, `auth.login.success`); that is history, not a defect to fix here.

## Technical Specification

### Frontend Components

#### Page: `AuditLogPage` (`src/pages/admin/AuditLogPage.tsx`)
- Route: `/admin/audit`
- Guarded by `RequireRole(['examiner','hr_admin','super_admin'])`
- Reads filters from URL search params via `useSearchParams`
- Renders `<AuditFilterBar>`, `<AuditLogTable>`, `<Pagination>`, `<ExportButton>`

#### Hook: `useAuditLog`
```ts
// src/api/audit.ts
export function useAuditLog(filters: AuditFilters, page: number) {
  return useQuery({
    queryKey: ['audit-log', filters, page],
    queryFn: () => apiGet<AuditLogResponse>(buildAuditURL(filters, page)),
  });
}

function buildAuditURL(filters: AuditFilters, page: number): string {
  const p = new URLSearchParams();
  if (filters.from)        p.set('from', filters.from);
  if (filters.to)          p.set('to', filters.to);
  if (filters.actor)       p.set('actor', filters.actor);
  if (filters.actions?.length) filters.actions.forEach(a => p.append('action', a));
  if (filters.entityType)  p.set('entity_type', filters.entityType);
  p.set('page', String(page));
  p.set('per_page', '50');
  return `/audit?${p.toString()}`;
}
```

#### Component: `AuditFilterBar` (`src/components/audit/AuditFilterBar.tsx`)
```tsx
// Props: { filters: AuditFilters; onChange: (f: AuditFilters) => void }
// Contains:
//   - DateRangePicker (from / to) using shadcn Calendar + Popover
//   - Input for actor name (debounced 300ms)
//   - MultiSelect for action codes (shadcn Command + Popover)
//   - Select for entity type
//   - "Clear Filters" button
```

#### Component: `AuditLogTable` (`src/components/audit/AuditLogTable.tsx`)
```tsx
// Props: { entries: AuditEntry[]; }
// Renders shadcn Table
// Each row has an expand toggle (chevron icon)
// Expanded row shows metadata JSON in <pre className="text-xs bg-muted p-2 rounded overflow-auto">
// Entity ID cell: truncated text + shadcn Tooltip showing full UUID
// Actor name: <Link to={`/admin/users/${entry.actor_id}`}>
```

#### Component: `ExportButton`
```tsx
// Props: { filters: AuditFilters }
// On click: calls apiGetBlob('/audit/export?' + params) → creates object URL → triggers download
// Renders Loader2 spinner while request is in flight via useMutation
```

### Known Action Codes (multi-select options)

```ts
export const AUDIT_ACTIONS = [
  'user.login', 'user.logout', 'user.create', 'user.update', 'user.deactivate', 'user.reactivate',
  'exam.create', 'exam.update', 'exam.publish', 'exam.archive',
  'session.start', 'session.submit', 'session.expire', 'session.pending_manual_grade',
  'answer.grade',
  'certificate.issue',
  'question.create', 'question.update', 'question.delete',
  'tenant.update',
] as const;
```

### Backend Addition: Audit Export Endpoint

If `GET /api/v1/audit/export` is not yet implemented in FR-BB19's audit package, add it:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/audit/export` | examiner+ | CSV export of filtered audit log |

Accepts the same query params as `GET /api/v1/audit` (from, to, actor, action, entity_type).

```go
// internal/audit/handler.go
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
    // build same filter struct as list handler
    // stream CSV via csv.NewWriter(w)
    filename := fmt.Sprintf("audit-%s.csv", time.Now().Format("20060102"))
    w.Header().Set("Content-Type", "text/csv; charset=utf-8")
    w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
    // ... stream rows ...
}
```

### i18n Keys Required

```json
{
  "audit.title": "Audit Log",
  "audit.filter_from": "From",
  "audit.filter_to": "To",
  "audit.filter_actor": "Actor name",
  "audit.filter_action": "Action",
  "audit.filter_entity": "Entity Type",
  "audit.filter_clear": "Clear Filters",
  "audit.col_timestamp": "Timestamp",
  "audit.col_actor": "Actor",
  "audit.col_action": "Action",
  "audit.col_entity_type": "Entity Type",
  "audit.col_entity_id": "Entity ID",
  "audit.col_ip": "IP Address",
  "audit.expand": "Show metadata",
  "audit.collapse": "Hide metadata",
  "audit.export_csv": "Export to CSV",
  "audit.exporting": "Exporting...",
  "audit.pagination_info": "Showing {{from}}–{{to}} of {{total}} entries"
}
```

### TypeScript Types

```ts
export interface AuditEntry {
  id: string;
  created_at: string;
  actor_id: string;
  actor_name: string;
  action: string;
  entity_type: string;
  entity_id: string;
  ip_address: string | null;
  metadata: Record<string, unknown>;
}

export interface AuditFilters {
  from?: string;
  to?: string;
  actor?: string;
  actions?: string[];
  entityType?: string;
}
```

## Notes
- The metadata column is stored as JSONB in PostgreSQL and serialised as a plain JSON object in the API response; the frontend renders it with `JSON.stringify(entry.metadata, null, 2)`.
- The date range picker defaults to "last 7 days" on initial page load to prevent loading the entire audit history.
- IP address may be null for actions triggered by internal background jobs (e.g. auto-submit).
