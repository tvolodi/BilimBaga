# FR-BB27 — Frontend: Question Bank List

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB27 |
| Phase | 2 — Content Management |
| Priority | 3 |
| Status | Draft |
| Depends On | FR-BB21, FR-BB23, FR-BB25, FR-BB26 |

## Description
A filterable, sortable, paginated data table providing an overview of the entire question bank. Content managers can filter by multiple dimensions simultaneously, perform inline status transitions without opening the editor, bulk-export or bulk-archive selections, and trigger CSV/JSON imports via an upload dialog with a parsed preview. The page is the primary entry point for question management and links into the full editor (FR-BB26) for detailed editing.

## Acceptance Criteria
- [ ] AC-1: The page renders a data table with columns: stem preview (≤ 120 chars of default locale, truncated with ellipsis), type badge, difficulty badge, category (leaf name), status badge, locale coverage indicators (one icon per configured locale, filled if translation exists), created by (user display name), and last updated (relative time); columns are non-wrapping on desktop.
- [ ] AC-2: The filter bar provides: a category tree picker (hierarchical), a tag multi-select, difficulty chips (easy / medium / hard, multi-select), a type filter dropdown, a status filter dropdown, and a locale coverage filter ("missing translation for locale X"); all filters are combinable and update the URL query string so the filtered view is shareable. Status filter supports multi-select (any combination of draft, review, active, archived). Difficulty filter supports multi-select. Tag filter supports multi-select. Filters are serialized to URL as comma-separated values (e.g., ?statuses=active,review).
- [ ] AC-3: Inline status transition is available via a dropdown menu on the status badge in the table row (not a full page navigation); clicking a transition calls `POST /api/v1/questions/:id/status` and updates the row optimistically via React Query cache mutation; if the transition fails the row reverts and a toast error is shown.
- [ ] AC-4: Row checkboxes enable multi-select; a bulk action bar appears at the bottom of the table when ≥ 1 row is selected, offering "Export selected (CSV)" and "Export selected (JSON)" which call `GET /api/v1/questions/export?ids=uuid1,uuid2,...` (comma-separated selected IDs; when more than 100 items are selected, the export uses the current filter state instead — see i18n key `questionBank.bulk.exportLimit`), and "Archive selected" which calls `POST /api/v1/questions/:id/status` for each selected question in parallel, showing a progress indicator.
- [ ] AC-5: The "Import" button opens a modal dialog; the user selects a CSV or JSON file; the frontend immediately calls `POST /api/v1/questions/import?dry_run=true` and displays the response in a preview table inside the modal (valid count, error rows highlighted in red with error text, warning rows highlighted in amber with similarity info); a "Confirm Import" button triggers the commit call.
- [ ] AC-6: The table is paginated with a default of 20 rows per page; page size is selectable (20 / 50 / 100); pagination state is reflected in the URL (`?page=2&per_page=50`); navigating away and back restores the previous filter/page state via URL params.
- [ ] AC-7: The table is sortable by `created_at`, `updated_at`, and `difficulty` (clicking the column header toggles asc/desc); sort state is reflected in the URL and persists through filter changes.
- [ ] AC-8: "New Question" button navigates to `/admin/questions/new`; clicking a row's stem preview navigates to `/admin/questions/:id/edit`; a separate "View versions" action in the row menu opens a slide-over panel showing the version history from `GET /api/v1/questions/:id/versions`.
- [ ] AC-9: An empty state is shown when no questions match the current filters, with a clear message and a "Clear filters" button; a loading skeleton (not a spinner) is shown during the initial data fetch.
- [ ] AC-10: All user-visible strings are sourced from `src/locales/{locale}.json`; the page renders correctly in Kazakh, Russian, and English; locale coverage indicators use accessible `aria-label` attributes (not colour alone) to convey status.
- [ ] AC-11: When bulk archiving a selection that includes questions whose current status does not allow the 'archived' transition, those rows display a per-row error indicator and the successfully-transitioned rows update to 'archived'; a summary toast (`questionBank.bulk.archivePartialResult`) reports the count of successes and failures.
- [ ] AC-12: A search bar matches questions whose default-locale stem text contains the search string (case-insensitive). Debounced 300 ms before triggering a new request. The search term syncs to the URL as ?search=. If the search input is cleared, the query param is removed and the full list is shown.

## Scope

| Layer | Items |
|-------|-------|
| Frontend | New page `QuestionBankPage` (`src/pages/admin/questions/QuestionBankPage.tsx`); components: `FilterBar`, `QuestionTable`, `BulkActionBar`, `ImportModal`, `VersionHistorySlideover`, `Pagination` |
| Router | New route `/admin/questions` registered in `App.tsx` under AdminLayout |
| i18n | New key namespace `questionBank.*` in `src/locales/{kk,ru,en}.json` |
| Backend | Extended under FR-BB23 via Code Fixer: added `?search=`, `?sort=`, `?order=` query params; `GET /api/v1/questions` list item now includes `category_name` (JOIN), `created_by_name` (JOIN), `stem_preview` (default locale stem text), and `tags` (tag name array) — separate `tag_ids`/`tag_names` fields replaced by a single `tags` string array. Also pluralised `?status`→`?statuses`, `?difficulty`→`?difficulties`, `?tag_id`→`?tag_ids` (all accept comma-separated multi-values; backward-compat aliases retained). |
| Database | No changes |

## Technical Specification

### Frontend Components

#### Page: `QuestionBankPage`
- Route: `/admin/questions` (nested under AdminLayout; requires authentication)
- Roles: `super_admin`, `department_admin`, `examiner` (guarded by `RequireRole`)
- Register in `App.tsx` as:
  ```tsx
  <Route path="questions" element={<RequireRole roles={['super_admin','department_admin','examiner']}><QuestionBankPage/></RequireRole>} />
  ```
  nested inside the `AdminLayout` route. The inner `RequireRole` is intentional defence-in-depth — the outer `AdminLayout` guard protects the layout shell; the inner guard enforces role policy at the route level regardless of how the route is reached.
- Fetches: `GET /api/v1/questions` (paginated + filtered), `GET /api/v1/categories`, `GET /api/v1/tags`
- Mutations: `POST /api/v1/questions/:id/status` (inline), `POST /api/v1/questions/import`, `GET /api/v1/questions/export` (download trigger)

```
QuestionBankPage
├── PageHeader
│   ├── Title ("Сұрақ банкі")
│   └── ImportButton + NewQuestionButton
├── FilterBar
│   ├── SearchInput               ← shadcn Input with a search icon, 300 ms debounce
│   ├── CategoryTreeFilter        ← Popover + recursive tree checkboxes
│   ├── TagMultiSelect            ← shadcn MultiSelect or Combobox
│   ├── DifficultyChips           ← toggle chips (easy / medium / hard)
│   ├── TypeFilterDropdown        ← shadcn Select, single-value
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
/admin/questions?page=1&per_page=20&sort=updated_at&order=desc
  &statuses=active,review&difficulties=hard,medium&tag_ids=uuid1,uuid2&search=photosynthesis
  &type=single
  &category_id=uuid-cat
  &locale_missing=en
```

#### TypeScript Types

```typescript
interface QuestionListItem {
  id: string;              // UUID v4
  type: 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert';
  difficulty: 'easy' | 'medium' | 'hard';
  status: 'draft' | 'review' | 'active' | 'archived';
  category_id: string;         // UUID v4
  category_name: string;       // resolved by backend JOIN
  default_locale: string;
  version: number;
  locale_coverage: string[];   // e.g. ['en', 'kk']
  stem_preview: string;        // default locale stem text
  tags: string[];              // tag names (not IDs)
  created_by: string;          // UUID v4
  created_by_name: string;     // resolved by backend JOIN
  created_at: string;          // UTC ISO 8601
  updated_at: string;          // UTC ISO 8601
}

interface QuestionFilters {
  page?: number;
  per_page?: number;
  sort?: 'created_at' | 'updated_at' | 'difficulty';
  order?: 'asc' | 'desc';
  category_id?: string;   // UUID v4
  tag_ids?: string[];     // UUID v4 array → ?tag_ids=uuid1,uuid2 (backend filters by question having these tag IDs)
  statuses?: Array<'draft' | 'review' | 'active' | 'archived'>; // → ?statuses=active,review
  difficulties?: Array<'easy' | 'medium' | 'hard'>; // → ?difficulties=hard,medium
  type?: 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert';
  locale_missing?: string;
  search?: string;
}

interface ImportDryRunResult {
  valid_count: number;
  error_rows: ImportErrorRow[];
  warning_rows: ImportWarningRow[];
}

interface ImportErrorRow {
  row: number;
  errors: string[];
}

interface ImportWarningRow {
  row: number;
  similarity_match: {
    question_id: string; // UUID v4
    score: number;       // 0.0–1.0
    stem_preview: string;
  };
}
```

#### i18n Keys
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
  "questionBank.filter.search": "Сұрақтарды іздеу…",
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
  "questionBank.bulk.archivePartialResult": "{{success}} мұрағатталды, {{failed}} сәтсіз",
  "questionBank.empty.title": "Сұрақтар жоқ",
  "questionBank.empty.description": "Сүзгілерге сәйкес сұрақтар табылмады",
  "questionBank.import.title": "Сұрақтарды импорттау",
  "questionBank.import.validCount": "Жарамды жолдар: {{count}}",
  "questionBank.import.errors": "Қателер",
  "questionBank.import.warnings": "Ескертулер",
  "questionBank.import.confirm": "Импорттауды растау",
  "questionBank.versions.title": "Нұсқалар тарихы",
  "questionBank.actions.edit": "Өңдеу",
  "questionBank.actions.archive": "Мұрағаттау",
  "questionBank.actions.viewVersions": "Нұсқаларды көру",
  "questionBank.pagination.previous": "Алдыңғы",
  "questionBank.pagination.next": "Келесі",
  "questionBank.pagination.pageOf": "{{page}} / {{total}} бет",
  "questionBank.pagination.pageSize": "Беттегі жолдар",
  "questionBank.sort.asc": "Өсу бойынша",
  "questionBank.sort.desc": "Кему бойынша",
  "questionBank.status.draft": "Жоба",
  "questionBank.status.review": "Тексеру",
  "questionBank.status.active": "Белсенді",
  "questionBank.status.archived": "Мұрағатта",
  "questionBank.error.loadFailed": "Жүктеу қатесі",
  "questionBank.error.statusTransitionFailed": "Мәртебені өзгерту қатесі",
  "questionBank.error.deleteFailed": "Жою қатесі",
  "questionBank.error.exportFailed": "Экспорт қатесі",
  "questionBank.skeleton.loading": "Жүктелуде...",
  "questionBank.localeCoverage.present": "{{locale}} тілінде бар",
  "questionBank.localeCoverage.missing": "{{locale}} тілінде жоқ",
  "questionBank.import.dropzone": "Файлды осында жіберіңіз немесе таңдаңыз",
  "questionBank.import.cancel": "Болдырмау",
  "questionBank.import.progress": "{{n}} / {{total}} өңделуде",
  "questionBank.bulk.exportProgress": "{{n}} сұрақ экспортталуда",
  "questionBank.bulk.exportLimit": "100-ден астам таңдалды — ағымдағы сүзгілермен экспорт жасалады",
  "questionBank.confirmDelete.title": "Сұрақты жою",
  "questionBank.confirmDelete.description": "Бұл әрекетті кері қайтару мүмкін емес.",
  "questionBank.confirmDelete.confirm": "Жою",
  "questionBank.confirmDelete.cancel": "Болдырмау"
}
```

## Notes
- Multi-value filter params are serialized as comma-separated strings in the URL (e.g., ?statuses=active,review). The backend `GET /api/v1/questions` accepts these comma-separated values and ANDs them with other filters. See URL Parameter Shape section above.
- The `tag_ids` filter sends **UUID v4 values** (not tag names) as a comma-separated string: `?tag_ids=uuid1,uuid2`. The backend resolves which questions are associated with those tag IDs via the `question_tags` join table. The `tags` field returned in the list item contains resolved tag **names** (strings) — this is display-only and is not used for filtering.
- AC-4 bulk-selected export requires FR-BB25 AC-6 to support an optional `?ids=` query parameter (comma-separated UUIDs). When `?ids=` is present, it overrides filter params and exports only those question IDs. See FR-BB25 for the endpoint specification.
- All filter state should be managed in the URL (using React Router `useSearchParams`) rather than component state, so filtered views are bookmarkable and shareable.
- React Query key for the list should include the full filter/sort/page object so each unique combination is cached separately: `['questions', filterParams]`.
- Bulk archive operations should be dispatched as `Promise.allSettled()` — not sequentially — to maximise parallelism; each resolved/rejected result updates the corresponding row in the cache.
- The import modal dry-run preview table should virtualise rows if `error_rows + warning_rows > 50` to keep the DOM manageable.
- Locale coverage icons should use distinct shapes (not only colours) for accessibility: a checkmark icon for present, an X icon for missing, each with `aria-label` carrying the locale name and status.
- The `VersionHistorySlideover` panel reuses the query `GET /api/v1/questions/:id/versions` with stale-while-revalidate; it does not need its own route.
- **React Query Configuration**:
  - `useQuestions` (questions list): `staleTime: 30_000`, `refetchOnWindowFocus: false`
  - `useCategories` (category tree): `staleTime: 300_000` (5 min — rarely changes)
  - `useTags`: `staleTime: 300_000` (5 min)
  - `useQuestionVersions`: `staleTime: 0` (always fresh)

## Out of Scope

Full question editing (FR-BB26), translation management per locale, exam assignment (Phase 3), and employee-facing question views are out of scope.

## Test Strategy

- **Unit tests**: `QuestionTable` component rendered with MSW-mocked API responses; assert columns, sort toggle, empty state, loading skeleton.
- **Integration tests**: Filter state → URL query param sync via `useSearchParams`; assert that applying a filter updates the URL and that navigating back restores the filter state.
- **E2E tests**: Inline status transition (click status dropdown → confirm → optimistic update → toast on success/failure) and bulk export flow (select rows → click Export CSV → assert download triggered with correct `?ids=...` parameter).
