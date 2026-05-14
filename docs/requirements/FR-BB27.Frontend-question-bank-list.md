# FR-BB27 — Frontend: Question Bank List

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB27 |
| Phase | 2 — Content Management |
| Priority | 3 |
| Status | Draft |
| Depends On | FR-BB23, FR-BB25 |

## Description
A filterable, sortable, paginated data table providing an overview of the entire question bank. Content managers can filter by multiple dimensions simultaneously, perform inline status transitions without opening the editor, bulk-export or bulk-archive selections, and trigger CSV/JSON imports via an upload dialog with a parsed preview. The page is the primary entry point for question management and links into the full editor (FR-BB26) for detailed editing.

## Acceptance Criteria
- [ ] AC-1: The page renders a data table with columns: stem preview (≤ 120 chars of default locale, truncated with ellipsis), type badge, difficulty badge, category (leaf name), status badge, locale coverage indicators (one icon per configured locale, filled if translation exists), created by (user display name), and last updated (relative time); columns are non-wrapping on desktop.
- [ ] AC-2: The filter bar provides: a category tree picker (hierarchical), a tag multi-select, difficulty chips (easy / medium / hard, multi-select), a type filter dropdown, a status filter dropdown, and a locale coverage filter ("missing translation for locale X"); all filters are combinable and update the URL query string so the filtered view is shareable.
- [ ] AC-3: Inline status transition is available via a dropdown menu on the status badge in the table row (not a full page navigation); clicking a transition calls `POST /api/v1/questions/:id/status` and updates the row optimistically via React Query cache mutation; if the transition fails the row reverts and a toast error is shown.
- [ ] AC-4: Row checkboxes enable multi-select; a bulk action bar appears at the bottom of the table when ≥ 1 row is selected, offering "Export selected (CSV)" and "Export selected (JSON)" which call `GET /api/v1/questions/export` with the selected IDs, and "Archive selected" which calls `POST /api/v1/questions/:id/status` for each selected question in parallel, showing a progress indicator.
- [ ] AC-5: The "Import" button opens a modal dialog; the user selects a CSV or JSON file; the frontend immediately calls `POST /api/v1/questions/import?dry_run=true` and displays the response in a preview table inside the modal (valid count, error rows highlighted in red with error text, warning rows highlighted in amber with similarity info); a "Confirm Import" button triggers the commit call.
- [ ] AC-6: The table is paginated with a default of 20 rows per page; page size is selectable (20 / 50 / 100); pagination state is reflected in the URL (`?page=2&per_page=50`); navigating away and back restores the previous filter/page state via URL params.
- [ ] AC-7: The table is sortable by `created_at`, `updated_at`, and `difficulty` (clicking the column header toggles asc/desc); sort state is reflected in the URL and persists through filter changes.
- [ ] AC-8: "New Question" button navigates to `/questions/new`; clicking a row's stem preview navigates to `/questions/:id/edit`; a separate "View versions" action in the row menu opens a slide-over panel showing the version history from `GET /api/v1/questions/:id/versions`.
- [ ] AC-9: An empty state is shown when no questions match the current filters, with a clear message and a "Clear filters" button; a loading skeleton (not a spinner) is shown during the initial data fetch.
- [ ] AC-10: All user-visible strings are sourced from `src/locales/{locale}.json`; the page renders correctly in Kazakh, Russian, and English; locale coverage indicators use accessible `aria-label` attributes (not colour alone) to convey status.

## Technical Specification

### Frontend Components

#### Page: `QuestionBankPage`
- Route: `/questions`
- Fetches: `GET /api/v1/questions` (paginated + filtered), `GET /api/v1/categories`, `GET /api/v1/tags`
- Mutations: `POST /api/v1/questions/:id/status` (inline), `POST /api/v1/questions/import`, `GET /api/v1/questions/export` (download trigger)

```
QuestionBankPage
├── PageHeader
│   ├── Title ("Сұрақ банкі")
│   └── ImportButton + NewQuestionButton
├── FilterBar
│   ├── CategoryTreeFilter        ← Popover + recursive tree checkboxes
│   ├── TagMultiSelect            ← shadcn MultiSelect or Combobox
│   ├── DifficultyChips           ← toggle chips (easy / medium / hard)
│   ├── TypeFilterDropdown        ← shadcn Select, multi-value
│   ├── StatusFilterDropdown      ← shadcn Select, multi-value
│   └── LocaleCoverageFilter      ← "Missing for:" locale picker
├── QuestionTable
│   ├── TableHeader (sortable columns)
│   ├── TableBody
│   │   └── QuestionRow (per question)
│   │       ├── Checkbox
│   │       ├── StemPreview
│   │       ├── TypeBadge           ← shadcn Badge
│   │       ├── DifficultyBadge     ← shadcn Badge (colour-coded)
│   │       ├── CategoryCell
│   │       ├── StatusDropdown      ← inline transition
│   │       ├── LocaleCoverageIcons (one per locale)
│   │       ├── CreatedByCell
│   │       ├── UpdatedAtCell       ← react-time-ago or date-fns
│   │       └── RowActionsMenu
│   │           ├── Edit
│   │           ├── View Versions   → VersionHistorySlideover
│   │           └── Archive
│   └── EmptyState / LoadingSkeleton
├── BulkActionBar (visible when selection > 0)
│   ├── SelectedCount
│   ├── ExportCSVButton
│   ├── ExportJSONButton
│   └── ArchiveSelectedButton (with progress)
├── Pagination
│   ├── PageSizeSelector (20/50/100)
│   └── PageNavigator
└── ImportModal
    ├── FileUpload (CSV/JSON)
    ├── DryRunResultsTable
    │   ├── ValidCountSummary
    │   ├── ErrorRowsTable (red)
    │   └── WarningRowsTable (amber)
    └── ConfirmImportButton
```

#### URL Parameter Shape
```
/questions?page=1&per_page=20&sort=updated_at&order=desc
  &status=active,review
  &difficulty=hard,medium
  &type=single
  &category_id=uuid-cat
  &tag_id=uuid-tag1,uuid-tag2
  &locale_missing=en
```

#### Key i18n Keys (representative)
```json
{
  "questionBank.title": "Сұрақ банкі",
  "questionBank.newQuestion": "Жаңа сұрақ",
  "questionBank.import": "Импорт",
  "questionBank.filter.category": "Санат",
  "questionBank.filter.tag": "Тег",
  "questionBank.filter.difficulty": "Қиындық",
  "questionBank.filter.type": "Түрі",
  "questionBank.filter.status": "Мәртебесі",
  "questionBank.filter.localeMissing": "Аудармасы жоқ",
  "questionBank.filter.clearAll": "Сүзгілерді тазалау",
  "questionBank.column.stem": "Сұрақ",
  "questionBank.column.type": "Түрі",
  "questionBank.column.difficulty": "Қиындық",
  "questionBank.column.category": "Санат",
  "questionBank.column.status": "Мәртебесі",
  "questionBank.column.coverage": "Аударма",
  "questionBank.column.createdBy": "Жасаған",
  "questionBank.column.updatedAt": "Жаңартылған",
  "questionBank.bulk.selected": "{{count}} сұрақ таңдалды",
  "questionBank.bulk.exportCsv": "CSV-ге экспорт",
  "questionBank.bulk.exportJson": "JSON-ге экспорт",
  "questionBank.bulk.archive": "Мұрағаттау",
  "questionBank.empty.title": "Сұрақтар жоқ",
  "questionBank.empty.description": "Сүзгілерге сәйкес сұрақтар табылмады",
  "questionBank.import.title": "Сұрақтарды импорттау",
  "questionBank.import.validCount": "Жарамды жолдар: {{count}}",
  "questionBank.import.errors": "Қателер",
  "questionBank.import.warnings": "Ескертулер",
  "questionBank.import.confirm": "Импорттауды растау",
  "questionBank.versions.title": "Нұсқалар тарихы"
}
```

## Notes
- All filter state should be managed in the URL (using React Router `useSearchParams`) rather than component state, so filtered views are bookmarkable and shareable.
- React Query key for the list should include the full filter/sort/page object so each unique combination is cached separately: `['questions', filterParams]`.
- Bulk archive operations should be dispatched as `Promise.allSettled()` — not sequentially — to maximise parallelism; each resolved/rejected result updates the corresponding row in the cache.
- The import modal dry-run preview table should virtualise rows if `error_rows + warning_rows > 50` to keep the DOM manageable.
- Locale coverage icons should use distinct shapes (not only colours) for accessibility: a checkmark icon for present, an X icon for missing, each with `aria-label` carrying the locale name and status.
- The `VersionHistorySlideover` panel reuses the query `GET /api/v1/questions/:id/versions` with stale-while-revalidate; it does not need its own route.
