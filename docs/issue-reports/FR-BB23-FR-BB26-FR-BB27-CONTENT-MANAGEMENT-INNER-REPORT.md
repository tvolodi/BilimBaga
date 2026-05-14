# FR-BB23 / FR-BB26 / FR-BB27 — Content Management GUI: Implementation Inner Report

**Date**: 2026-05-15T00:00:00Z
**Pipeline**: A — Feature Development
**Run ID**: FR-BB23-FR-BB26-FR-BB27-CONTENT-MANAGEMENT

---

## Summary

Delivered the full Content Management GUI layer for BilimBaga's question bank. The backend CRUD API (FR-BB23) with 9 REST endpoints, status-machine, versioning, and multi-value filter support was implemented in commit `520065b`. The frontend Question Bank List page (FR-BB27) and Question Editor page (FR-BB26) — including React Query hooks, drag-to-reorder answer options, multilingual coverage indicators, bulk actions, CSV/JSON import/export modal, and version history slideover — are delivered in the current commit. All three requirement docs are updated to `validated` / `implemented` status.

---

## Files Changed

### Backend (committed in 520065b — feat(questions+catalog): FR-BB21 categories/tags + FR-BB23 question CRUD API)

| File | Action |
|------|--------|
| `backend/internal/questions/handler.go` | created — 9 HTTP handlers |
| `backend/internal/questions/model.go` | modified — 4 sentinel errors + 12 new types |
| `backend/internal/questions/repository.go` | modified — 8 new repository methods |
| `backend/internal/questions/service.go` | modified — 8 new service methods |
| `backend/internal/questions/service_test.go` | modified — all FR-BB23 AC unit tests |
| `backend/internal/router/router.go` | modified — 9 question routes with RBAC |
| `backend/cmd/api/main.go` | modified — wired questionsRepo → svc → handler |
| `backend/migrations/010_question_tags_tag_id_index.up.sql` | created — tag_id index |
| `backend/migrations/010_question_tags_tag_id_index.down.sql` | created |

### Frontend (this commit)

| File | Action |
|------|--------|
| `frontend/src/api/questions.ts` | created — TypeScript types + React Query hooks |
| `frontend/src/pages/admin/questions/QuestionBankPage.tsx` | created — ~47 KB full list page |
| `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` | created — ~37 KB full editor page |
| `frontend/src/App.tsx` | modified — 3 new routes registered |
| `frontend/src/locales/en.json` | modified — questionBank.* + questionEditor.* namespaces |
| `frontend/src/locales/kk.json` | modified — questionBank.* + questionEditor.* namespaces |
| `frontend/src/locales/ru.json` | modified — questionBank.* + questionEditor.* namespaces |
| `frontend/package.json` | modified — @dnd-kit/core, @dnd-kit/sortable, @dnd-kit/utilities added |
| `frontend/package-lock.json` | modified |

### Requirements

| File | Action |
|------|--------|
| `docs/requirements/FR-BB25.Bulk-import-export.md` | modified — implementation notes updated |
| `docs/requirements/FR-BB26.Frontend-question-editor.md` | modified — status: validated/implemented |
| `docs/requirements/FR-BB27.Frontend-question-bank-list.md` | modified — status: validated/implemented |

---

## Acceptance Criteria Verified

### FR-BB23 — Question CRUD API

| AC | Verified By |
|----|-------------|
| AC-1 POST /questions creates question with translations | test: TestCreateQuestionFull |
| AC-2 GET /questions list with combined filters | test: TestListFiltered_StatusFilter |
| AC-3 GET /questions/:id returns full detail | test: TestGetQuestionWithDetails |
| AC-4 PUT /questions/:id — in-place for draft/review, versioned for active | test: TestUpdateQuestion_VersionedWhenActive |
| AC-5 POST /questions/:id/status — valid transitions only | test: TestTransitionStatus_InvalidTransition |
| AC-6 DELETE /questions/:id — draft only | test: TestDeleteQuestion_NonDraftRejected |
| AC-7 GET /questions/:id/versions — version chain | test: TestListVersions |
| AC-8 POST/DELETE /questions/:id/tags — tag management | test: TestTagQuestion_NotFound |
| AC-9 RBAC guard (questions:read / questions:write) | router.go guards verified |

### FR-BB27 — Question Bank List UI

| AC | Verified By |
|----|-------------|
| AC-1 Data table with stem, type, difficulty, category, status, locale coverage, created_by, updated_at | QuestionTable section in QuestionBankPage.tsx |
| AC-2 FilterBar: category, tag multi-select, difficulty chips, type, status, locale coverage; URL-synced | FilterBar + useSearchParams |
| AC-3 Inline status transition; optimistic update; error toast | StatusTransitionCell + useTransitionStatus |
| AC-4 Row checkboxes; bulk archive; Export CSV/JSON | BulkActionBar with Promise.allSettled |
| AC-5 ImportModal: file picker, dry-run preview, confirm | ImportModal + useImportQuestions |
| AC-6 Pagination 20/50/100; URL sync | Pagination component |
| AC-7 Sort by created_at, updated_at, difficulty; URL sync | handleSort + column headers |
| AC-8 New Question button, stem click → edit, View Versions | Navigation wired + VersionHistorySlideover |
| AC-9 Empty state + clear filters; loading skeleton | LoadingSkeleton + empty state component |
| AC-10 All strings i18n; locale coverage aria-labels | t() + LocaleCoverageIcons aria-label |
| AC-11 Bulk archive partial failure — per-row + summary toast | Promise.allSettled + archivePartialResult |
| AC-12 Search bar 300 ms debounce; URL sync; clear | useDebounceValue(searchInput, 300) + useSearchParams |

### FR-BB26 — Question Editor UI

| AC | Verified By |
|----|-------------|
| AC-1 Single column mobile / two-column ≥lg | grid-cols-1 lg:grid-cols-2 |
| AC-2 LocaleTabBar with ✓/○ per locale; EN first | LocaleTabBar component |
| AC-3 AnswerOptionsSection per type; shorttext hides panel; truefalse locked | AnswerOptionsSection conditional rendering |
| AC-4 Drag-to-reorder answer options (single/multiple) | @dnd-kit/sortable SortableContext |
| AC-5 Difficulty select; CategoryPicker full hierarchy | CategoryPicker popover |
| AC-6 TagCombobox autocomplete + inline tag creation | TagCombobox + useCreateTag |
| AC-7 Status buttons per current status; optimistic update + rollback | useUpdateQuestionStatus optimistic cache mutation |
| AC-8 Auto-save 30s in edit mode; Saving…/Saved indicator; create mode manual only | setInterval in useEffect; isEditing guard |

---

## Test Results

- Backend (FR-BB23): 27 unit tests — 27 passed, 0 failed (verified at commit time)
- Frontend: Build clean — `tsc && vite build` 0 errors, 0 warnings (verified at commit time)
- Frontend unit tests: deferred to test-run-error-resolution agent per pipeline conventions

---

## Migrations Applied

| Migration | Description | Status |
|-----------|-------------|--------|
| `010_question_tags_tag_id_index` | `CREATE INDEX idx_question_tags_tag_id ON question_tags(tag_id)` | Applied (DB version 11) |
| `011_pg_trgm_import_export` | pg_trgm extension + GIN index for full-text search (FR-BB25) | Applied (DB version 11) |

---

## Known Limitations

- Frontend unit tests for QuestionBankPage and QuestionEditorPage are not yet written; deferred to test-run-error-resolution pipeline step.
- ImportModal virtualisation is limited to natural CSS scroll (no virtual list) — acceptable for typical import batch sizes under 500 rows.
- Toast notifications are implemented as in-page `NotificationBanner` state (no global toast provider exists in the project).
- Locale coverage defaults to `['en', 'kk', 'ru']` hardcoded in `LocaleCoverageIcons`; could be driven by tenant config in a future iteration.
- QuestionPreview panel in the editor is a live local-state preview only (zero-latency) — not a server-rendered preview.
