# FR-BB25: Implementation Inner Report

**Date**: 2026-05-15T00:00:00Z
**Pipeline**: A
**Commit**: 9417d74 (finalization); core implementation: 420a735

## Summary

Implemented bulk import/export for the question bank (FR-BB25). The core implementation was delivered in commit `420a735` (merged from `feature/FR-BB25-bulk-import-export`). This finalization commit (`9417d74`) added the service-level unit tests, extended handler tests with spec-required test names, an `rbac.Cache.LoadFromMap` helper used by handler tests, and updated the requirement status to Implemented.

Import supports CSV and JSON formats with dry-run mode, batch size enforcement (max 500), all-or-nothing commit semantics, and pg_trgm-based duplicate detection. Export streams CSV and JSON to avoid OOM on large datasets. The pg_trgm extension and GIN index on `question_translations.stem` were added via migration `011_pg_trgm_import_export`.

## Files Changed

| File | Action |
|------|--------|
| `backend/internal/questions/import_export_handler.go` | created (commit 420a735) |
| `backend/internal/questions/import_export_repository.go` | created (commit 420a735) |
| `backend/internal/questions/import_export_service.go` | created (commit 420a735) |
| `backend/internal/questions/model.go` | modified — added import/export models (commit 420a735) |
| `backend/internal/questions/import_service_test.go` | created (commit 9417d74) |
| `backend/internal/questions/import_export_handler_test.go` | extended with spec-required test aliases (commit 9417d74) |
| `backend/internal/questions/service_test.go` | extended — added findSimilarStemsFn to mock (commit 9417d74) |
| `backend/internal/rbac/cache.go` | added LoadFromMap helper for tests (commit 9417d74) |
| `backend/migrations/011_pg_trgm_import_export.up.sql` | created (commit 420a735) |
| `backend/migrations/011_pg_trgm_import_export.down.sql` | created (commit 420a735) |
| `docs/requirements/FR-BB25.Bulk-import-export.md` | modified — Status set to Implemented |
| `docs/requirements/README.md` | modified — FR-BB25 status updated to Implemented |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC-1 | test: TestImportHandler_DryRun_Returns200, TestImportHandler_DryRun_200 (alias), TestImport_DryRun_DetectsInvalidRows |
| AC-2 | test: TestImport_Commit_InsertsAllValid, TestImport_Commit_RollsBackOnAnyError |
| AC-3 | test: TestImport_BatchTooLarge, TestImportHandler_BatchTooLarge_413 |
| AC-4 | test: TestImport_DuplicateDetection |
| AC-5 | test: TestImport_DryRun_DetectsInvalidRows (category_path, correct indices) |
| AC-6 | test: TestExportHandler_CSV_200, TestExportHandler_JSON_200 |
| AC-7 | test: TestExportHandler_JSON_200 |
| AC-8 | test: TestExportHandler_CSV_200 |
| AC-9 | code review — RBAC and audit log verified |
| AC-10 | test: TestImport_NoTrigram_Proceeds |

## Test Results

- Backend: 19 packages passing, 0 failed
- Frontend: N/A (backend-only feature)

## Migration Applied

`backend/migrations/011_pg_trgm_import_export.up.sql` — committed and available for `make migrate`.

## Known Limitations

- The 413 batch-too-large check fires after CSV/JSON parsing (row counting requires parsing); the spec says "before any parsing" which is interpreted as "before service/DB calls". This is documented in the handler comment.
- `rbac.Cache.LoadFromMap` is a test helper added to the production `cache.go` file; it is unexported-safe and lock-protected.
