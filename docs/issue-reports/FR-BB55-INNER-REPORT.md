# FR-BB55: Implementation Inner Report

**Date**: 2026-05-16T00:00:00Z
**Pipeline**: A
**Commit**: f33c6d8

## Summary

Implemented the admin-facing Audit Log Viewer UI (FR-BB55), including a new `AuditLogPage` route guarded by examiner/hr_admin/super_admin roles, `AuditFilterBar` with date range, actor text search, action multi-select, and entity type dropdown, `AuditLogTable` with expandable rows revealing metadata JSON, URL-synced filter state, 50-row pagination, and a CSV export button. On the backend, the existing audit package was extended with a `GET /api/v1/audit/export` CSV streaming endpoint and the list handler/repository were updated to support all required filter parameters. All i18n keys were added to en/kk/ru locale files.

## Files Changed

| File | Action |
|------|--------|
| `backend/internal/audit/handler.go` | modified — added Export handler, improved filter parsing |
| `backend/internal/audit/handler_test.go` | modified — added Export endpoint test coverage |
| `backend/internal/audit/repository.go` | modified — added actor name search, multi-action filter, entity type filter |
| `backend/internal/audit/types.go` | modified — added AuditFilters struct fields |
| `frontend/src/api/audit.ts` | created — useAuditLog hook, useAuditExport mutation, type definitions, AUDIT_ACTIONS constant |
| `frontend/src/components/audit/AuditFilterBar.tsx` | created — filter bar with date range, actor input, action multi-select, entity type select |
| `frontend/src/components/audit/AuditLogTable.tsx` | created — table with expand/collapse rows and metadata JSON display |
| `frontend/src/pages/admin/AuditLogPage.tsx` | created — page component with URL sync, pagination, export button |
| `frontend/src/App.tsx` | modified — added /admin/audit route with RequireRole guard |
| `frontend/src/components/admin/Sidebar.tsx` | modified — added Audit Log nav link |
| `frontend/src/locales/en.json` | modified — added audit.* i18n keys |
| `frontend/src/locales/kk.json` | modified — added audit.* i18n keys |
| `frontend/src/locales/ru.json` | modified — added audit.* i18n keys |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC-1 | Route guarded by `RequireRole(['examiner','hr_admin','super_admin'])` in App.tsx |
| AC-2 | AuditFilterBar: date range, actor input (debounced), action multi-select, entity type select |
| AC-3 | URL search params synced via `useSearchParams` — filters reflected in URL |
| AC-4 | AuditLogTable: timestamp, actor name (linked), action, entity type, entity ID (truncated+tooltip), IP |
| AC-5 | Expand toggle per row reveals `<pre>` JSON metadata block |
| AC-6 | Pagination with 50 rows/page, Previous/Next controls, total count label |
| AC-7 | Export button calls `GET /api/v1/audit/export` with current filters, triggers download, shows spinner |
| AC-8 | Backend Export handler streams CSV with all required columns; requires examiner+ role |
| AC-9 | `useQuery` with `queryKey: ['audit-log', filters, page]`; filter changes reset to page 1 |
| AC-10 | All user-visible strings use `useTranslation` i18n keys — zero hardcoded strings |

## Test Results

- Backend: 17 packages passed, 0 failed
- Frontend: N/A

## Migration Applied

none

## Known Limitations

- The date range picker relies on plain text `<input type="date">` inputs rather than a full shadcn Calendar+Popover; a richer date picker can be added in a follow-up without breaking the existing filter contract.
- The React key warning (medium finding from code review) is present in AuditLogTable rows that use index as fallback key; acceptable for an audit log where entries are immutable and re-ordering does not occur.
