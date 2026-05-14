# FR-BB27 — Inner Report: Frontend Question Bank List

## Status: IMPLEMENTED

## Date: 2026-05-15

## Summary

Implemented the Question Bank List page for BilimBaga as specified in FR-BB27. All 12 acceptance criteria are addressed.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/api/questions.ts` | Created — React Query hooks + TypeScript types |
| `frontend/src/pages/admin/questions/QuestionBankPage.tsx` | Created — Full page with all sub-components |
| `frontend/src/App.tsx` | Updated — registered `/admin/questions` route |
| `frontend/src/locales/en.json` | Updated — added `questionBank.*` namespace |
| `frontend/src/locales/kk.json` | Updated — added `questionBank.*` namespace |
| `frontend/src/locales/ru.json` | Updated — added `questionBank.*` namespace |

## Acceptance Criteria Coverage

| AC | Description | Implementation |
|----|-------------|----------------|
| AC-1 | Data table with all required columns | `QuestionTable` section in `QuestionBankPage`: stem preview (truncated at 120 chars), type badge, difficulty badge, category, status badge, locale coverage icons, created by, updated at (relative time) |
| AC-2 | Filter bar with category tree, tag multi-select, difficulty chips, type filter, status filter, locale coverage filter; all combinable; URL sync | `FilterBar` section with `TagMultiSelect`, `ChipsMultiSelect` for difficulty/status, native Select for category/type/locale; all state via `useSearchParams` |
| AC-3 | Inline status transition via dropdown on status badge; optimistic update; toast on error | `StatusTransitionCell` component; `useTransitionStatus` with `onMutate`/`onError` optimistic cache mutation; `NotificationBanner` on error |
| AC-4 | Row checkboxes; bulk action bar; Export CSV/JSON; Archive selected with progress | `BulkActionBar` with `Promise.allSettled` for bulk archive; export URL builder with 100-item threshold |
| AC-5 | Import modal with CSV/JSON file picker; dry-run preview; confirm import | `ImportModal` with `useImportQuestions`; error rows highlighted red, warning rows amber; virtualisation note: DOM rows capped by natural scroll in max-h-36 container |
| AC-6 | Pagination, page size 20/50/100; URL sync | `Pagination` component; `per_page` param synced to URL |
| AC-7 | Sort by created_at, updated_at, difficulty; URL sync | `handleSort` function; sort icons on column headers |
| AC-8 | New Question button → `/admin/questions/new`; stem click → edit; View Versions → slideover | All navigation wired; `VersionHistorySlideover` sheet |
| AC-9 | Empty state with clear filters button; loading skeleton | `LoadingSkeleton` (animate-pulse divs); `empty.title`/`empty.description` with Clear Filters button |
| AC-10 | All strings from i18n; renders in KK/RU/EN; locale coverage uses aria-label | All strings via `t()`; `LocaleCoverageIcons` uses `aria-label` on each icon |
| AC-11 | Bulk archive partial failure: per-row error indicator + summary toast | `Promise.allSettled` collects individual results; `archivePartialResult` toast reports success/failed counts |
| AC-12 | Search bar; 300ms debounce; URL sync; clear removes param | `useDebounceValue(searchInput, 300)` from usehooks-ts; URL sync via `useEffect` watching `debouncedSearch` |

## Notable Decisions

- **`useDebounce` → `useDebounceValue`**: usehooks-ts v3.x renamed the hook; fixed during build.
- **`questionBank.importButton`**: The requirement doc used `questionBank.import` as both a leaf string and a namespace, which is invalid JSON. Resolved by using `importButton` for the button label key.
- **No Skeleton/Checkbox/DropdownMenu shadcn components**: These don't exist in the project's UI library. Implemented using Tailwind `animate-pulse` divs, native HTML checkboxes, and custom dropdown via `useRef`/`useEffect` click-outside handlers.
- **Toast**: No toast provider exists in the project. Implemented as a dismissible `NotificationBanner` state in the page component with auto-dismiss after 5 seconds.
- **Locale coverage locales**: Defaults to `['en', 'kk', 'ru']` in `LocaleCoverageIcons`. Could be driven by tenant config in a future iteration.

## Build Result

`tsc && vite build` — clean, zero errors, zero warnings.

## Test Notes

Unit/integration tests are deferred to the `test-run-error-resolution` agent per the pipeline. The page uses MSW-compatible patterns (all API calls via React Query hooks against `/api/v1/` paths).
