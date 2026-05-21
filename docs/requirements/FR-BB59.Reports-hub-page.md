# FR-BB59 — Frontend: Reports Hub Page

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB59 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | implemented |
| Depends On | FR-BB51, FR-BB52, FR-BB53, FR-BB54, FR-BB56, FR-BB57 |

## Description
Provides a single consolidated page at `/admin/reports` that surfaces all reporting and export capabilities for admin users. The page exposes two sections: a Dashboard PDF Export card with a date-range picker that downloads a PDF report from the existing export endpoint, and an Exam Reports table that lists all exams with per-row "Analytics" navigation links and "Export CSV" download buttons. The route fills the gap caused by the existing sidebar "Reports" nav item that currently points to `/admin/reports` but has no matching route. No new backend endpoints are introduced; all data comes from existing Phase 5 APIs.

## Acceptance Criteria
- [ ] AC-1: Navigating to `/admin/reports` renders a page titled "Reports" (i18n key `reports.title`) and does not redirect or show a blank screen.
- [ ] AC-2: The route is accessible only to roles `super_admin`, `examiner`, and `hr_admin`; any other authenticated role is redirected to `/login`.
- [ ] AC-3: The Dashboard PDF Export card contains a date-range picker with "From" and "To" date inputs and an "Export PDF" button; clicking the button downloads the PDF from `GET /api/v1/admin/dashboard/export?from=<ISO8601>&to=<ISO8601>`.
- [ ] AC-4: The "Export PDF" button shows a loading state (spinner + disabled) while the request is in flight and reverts to its default state on completion or error.
- [ ] AC-5: The Exam Reports table fetches all exams from `GET /api/v1/exams` using a `useQuery` hook with `queryKey: ['exams-list-reports']`; a skeleton loader is shown while the query is pending.
- [ ] AC-6: Each row of the Exam Reports table displays the exam title, its status badge, an "Analytics" link that navigates to `/admin/exams/:id/analytics`, and an "Export CSV" button.
- [ ] AC-7: Clicking "Export CSV" on an exam row triggers a file download from `GET /api/v1/admin/exams/:id/results/export`; the button shows a per-row loading state while the download is in progress.
- [ ] AC-8: When the exams list is empty, the Exam Reports table renders an empty-state message (i18n key `reports.exams_empty`) instead of an empty table body.
- [ ] AC-9: Zero hardcoded user-visible strings; all text is sourced from `useTranslation` using keys in the `reports.*` namespace across `en.json`, `kk.json`, and `ru.json`.
- [ ] AC-10: The page is responsive: on viewport widths ≥ 768 px the two sections are arranged in a two-column grid; on narrower viewports they stack vertically.
- [ ] AC-11: The page is reachable from the admin sidebar "Reports" nav item without any additional configuration change.

## Technical Specification

### Files to Create

| File | Purpose |
|------|---------|
| `frontend/src/pages/admin/ReportsPage.tsx` | Route-level page component |
| `frontend/src/api/reports.ts` | React Query hooks and fetch helpers for this page |

### Files to Modify

| File | Change |
|------|--------|
| `frontend/src/App.tsx` | Add lazy import and `<Route path="reports" …>` inside the `/admin` layout route |
| `frontend/src/locales/en.json` | Add `"reports": { … }` namespace |
| `frontend/src/locales/kk.json` | Add `"reports": { … }` namespace |
| `frontend/src/locales/ru.json` | Add `"reports": { … }` namespace |

### Routing

```tsx
// App.tsx — lazy import (alongside other admin lazy imports)
const ReportsPage = lazy(() =>
  import('@/pages/admin/ReportsPage').then((m) => ({ default: m.ReportsPage })),
)

// Inside the /admin <Route> block
<Route
  path="reports"
  element={
    <RequireRole roles={['super_admin', 'examiner', 'hr_admin']}>
      <ReportsPage />
    </RequireRole>
  }
/>
```

### API Hooks (`frontend/src/api/reports.ts`)

```ts
import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { ExamListItem } from '@/api/exams'

// Fetch all exams for the reports table.
// Alternatively, import and use useExams() from '@/api/exams' directly.
export function useExamsList() {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['exams-list-reports'],
    queryFn: async (): Promise<ExamListItem[]> => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      const res = await fetch('/api/v1/exams', {
        credentials: 'include',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!res.ok) throw new Error('ERR_FETCH_EXAMS')
      const body = await res.json()
      return body.data.items as ExamListItem[]
    },
    staleTime: 60 * 1000,
  })
}

// Download dashboard PDF — called imperatively, not via useQuery
export async function downloadDashboardPdf(from: string, to: string): Promise<void> {
  const res = await fetch(
    `/api/v1/admin/dashboard/export?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    { credentials: 'include' },
  )
  if (!res.ok) throw new Error('ERR_EXPORT_PDF')
  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `dashboard-report-${from}-${to}.pdf`
  document.body.appendChild(a)
  a.click()
  window.URL.revokeObjectURL(url)
}

// Download per-exam CSV — called imperatively, not via useQuery
export async function downloadExamCsv(examId: string, examTitle: string): Promise<void> {
  const res = await fetch(`/api/v1/admin/exams/${examId}/results/export`, {
    credentials: 'include',
  })
  if (!res.ok) throw new Error('ERR_EXPORT_CSV')
  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `exam-results-${examId}.csv`
  document.body.appendChild(a)
  a.click()
  window.URL.revokeObjectURL(url)
}
```

### Page Component Structure (`frontend/src/pages/admin/ReportsPage.tsx`)

```tsx
// Top-level layout: PageHeader + two-section grid
// Section 1 — DashboardExportCard
//   shadcn Card containing:
//     - CardHeader: title (reports.pdf_export_title)
//     - CardContent: two <Input type="date"> fields (from / to) + Button "Export PDF"
//     - Loading / error feedback via toast
// Section 2 — ExamReportsTable
//   shadcn Table with columns: Title | Status | Analytics | Export CSV
//   Data from useExamsList()
//   Skeleton rows (4) while loading
//   Empty state when data.length === 0
//   Per-row "Analytics" → <Link to={`/admin/exams/${exam.id}/analytics`}>
//   Per-row "Export CSV" → calls downloadExamCsv(exam.id, exam.title) with local loading state
```

### i18n Keys Required

All keys added to the `"reports"` object in each locale file:

| Key | English value |
|-----|---------------|
| `reports.title` | `"Reports"` |
| `reports.pdf_export_title` | `"Dashboard PDF Report"` |
| `reports.pdf_export_description` | `"Download a PDF summary of the admin dashboard for a selected date range."` |
| `reports.date_from` | `"From"` |
| `reports.date_to` | `"To"` |
| `reports.export_pdf` | `"Export PDF"` |
| `reports.exporting` | `"Exporting..."` |
| `reports.export_error` | `"Export failed. Please try again."` |
| `reports.exam_reports_title` | `"Exam Reports"` |
| `reports.exam_reports_description` | `"View analytics or download results for each exam."` |
| `reports.col_exam` | `"Exam"` |
| `reports.col_status` | `"Status"` |
| `reports.col_analytics` | `"Analytics"` |
| `reports.col_export` | `"Export"` |
| `reports.analytics_link` | `"View Analytics"` |
| `reports.export_csv` | `"Export CSV"` |
| `reports.exams_empty` | `"No exams found."` |
| `reports.exams_loading_error` | `"Failed to load exams."` |

### API Contract (existing endpoints — no new backend work)

#### GET `/api/v1/admin/dashboard/export`
- Query params: `from` (ISO 8601 date), `to` (ISO 8601 date)
- Success: binary `application/pdf` response — browser file download
- Error: `{ data: null, error: { code, message } }`

#### GET `/api/v1/exams`
- No params required for listing
- Success: `{ data: { items: ExamListItem[], meta: { page: number, per_page: number, total: number } }, error: null }`
- Fields used from each `ExamListItem`: `id`, `title`, `status`

#### GET `/api/v1/admin/exams/:id/results/export`
- No additional params
- Success: binary `text/csv` response — browser file download
- Error: `{ data: null, error: { code, message } }`

## Out of Scope
- Employee record CSV export (`GET /api/v1/admin/users/:id/record/export`) — that action lives on `EmployeeRecordPage` and is not duplicated here.
- Per-exam analytics data rendering — that is covered by `ExamAnalyticsPage` (FR-BB57); this page only links to it.
- Filtering or pagination of the exams list on this page — the table shows all exams.
- Any new backend endpoints or database migrations.
- Scheduling or emailing of reports.

## Test Strategy
- **Unit / component tests**: `ReportsPage` renders both sections; `DashboardExportCard` disables the button and shows loading text while download is in flight; `ExamReportsTable` renders skeleton rows when loading, renders empty-state when data is empty, and renders rows with correct content when data is provided.
- **Integration tests**: `downloadDashboardPdf` calls the correct URL with encoded `from`/`to` params; `downloadExamCsv` calls `/admin/exams/:id/results/export` and triggers a download.
- **Role guard test**: A user with role `department_admin` is redirected away from `/admin/reports`.
- **i18n test**: Switching locale updates all visible strings without a page reload.
