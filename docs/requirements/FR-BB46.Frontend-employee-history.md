# FR-BB46 — Frontend: Employee History

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB46 |
| Phase | 4 — Results & Certificates |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB41 |

## Description
Adds a "My Results" tab to the employee portal that presents a paginated, sortable table of all the employee's past exam sessions across all exams. Each row shows key outcome data and provides direct links to the detailed result view and certificate download where applicable. An empty state is shown when the employee has no completed sessions.

## Acceptance Criteria
- [ ] AC-1: The "My Results" tab is visible in the employee portal navigation; it is active only when the user navigates to the corresponding route.
- [ ] AC-2: The table displays at least the following columns: exam name, date taken (`submitted_at`), score percentage, pass/fail badge, time taken, certificate download link.
- [ ] AC-3: The certificate download link is shown in a row only when `passed = true` AND the exam had `certificate_enabled = true`; otherwise the cell is empty.
- [ ] AC-4: Columns "Date Taken" and "Score" are sortable; clicking the column header toggles ascending/descending order; sort state is reflected via `?sort=score&dir=asc` query params.
- [ ] AC-5: Pagination renders 20 rows per page with Previous/Next controls and a "Showing X–Y of Z results" count label.
- [ ] AC-6: An empty state component (illustration + message, i18n key `history.empty`) is rendered when the fetched sessions array is empty and no filters are active.
- [ ] AC-7: Clicking an exam name in the table navigates to the result detail page (`/portal/sessions/:id/result`).
- [ ] AC-8: The certificate download link calls `GET /portal/sessions/:id/certificate` and triggers a browser file download without navigating away.
- [ ] AC-9: The table uses `useQuery` with `queryKey: ['my-results', page, sort, dir]`; changing page or sort parameters invalidates and refetches.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.

## Technical Specification

### Frontend Components

#### Page: `MyResultsPage` (`src/pages/portal/MyResultsPage.tsx`)
- Route: `/portal/results`
- Tab in the employee portal shell alongside "My Exams"
- Uses `useMyResults(page, sort, dir)` hook

#### Hook: `useMyResults`
```ts
// src/api/sessions.ts
export function useMyResults(page: number, sort: 'date' | 'score', dir: 'asc' | 'desc') {
  return useQuery({
    queryKey: ['my-results', page, sort, dir],
    queryFn: () =>
      apiGet<MyResultsResponse>(`/portal/results?page=${page}&sort=${sort}&dir=${dir}&per_page=20`),
    placeholderData: keepPreviousData,
  });
}
```

#### Component: `ResultsTable` (`src/components/results/ResultsTable.tsx`)
```tsx
// Props: { sessions: SessionHistoryItem[]; onSort: (col) => void; sortCol: string; sortDir: string }
// Built on shadcn Table primitive
// Pass/fail badge: <Badge variant="success"> / <Badge variant="destructive">
// Certificate cell: <Button variant="ghost" size="sm" onClick={handleDownload}>
```

#### Component: `ResultsTableSkeleton`
```tsx
// 5 skeleton rows rendered during loading state
// Uses shadcn Skeleton component
```

#### Component: `EmptyResults`
```tsx
// Props: none
// Renders centred SVG illustration + i18n message + link to assigned exams
```

### API Contract

The backend returns sessions **aggregated across all exams** for the calling user. This reuses the `GET /portal/exams/:id/history` design but via a new endpoint:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/portal/results` | any authenticated | All completed sessions for the caller |

Query params: `page`, `per_page`, `sort` (`date` | `score`), `dir` (`asc` | `desc`)

**Response (200)**
```json
{
  "data": {
    "sessions": [
      {
        "session_id": "uuid-session",
        "exam_id": "uuid-exam",
        "exam_title": "Fire Safety Fundamentals",
        "submitted_at": "2026-05-14T10:30:00Z",
        "score_pct": 84.50,
        "passed": true,
        "time_taken_seconds": 1423,
        "certificate_available": true
      }
    ],
    "total_count": 7,
    "page": 1,
    "per_page": 20
  },
  "error": null
}
```

### i18n Keys Required

```json
{
  "history.title": "My Results",
  "history.empty": "You have not completed any exams yet.",
  "history.empty_cta": "View My Assigned Exams",
  "history.col_exam": "Exam",
  "history.col_date": "Date Taken",
  "history.col_score": "Score",
  "history.col_status": "Status",
  "history.col_time": "Time Taken",
  "history.col_certificate": "Certificate",
  "history.download_certificate": "Download",
  "history.pagination_info": "Showing {{from}}–{{to}} of {{total}} results"
}
```

### TypeScript Types

```ts
export interface SessionHistoryItem {
  session_id: string;
  exam_id: string;
  exam_title: string;
  submitted_at: string;
  score_pct: number;
  passed: boolean;
  time_taken_seconds: number | null;
  certificate_available: boolean;
}

export interface MyResultsResponse {
  sessions: SessionHistoryItem[];
  total_count: number;
  page: number;
  per_page: number;
}
```

## Notes
- `certificate_available` is a computed boolean from the backend (`passed AND exam.certificate_enabled`) to avoid requiring the frontend to know exam configuration.
- Sort state is stored in URL query params (not component state) so the page is bookmarkable and shareable.
- Time taken is formatted on the frontend using a shared `formatDuration(seconds: number): string` utility, not on the backend.
