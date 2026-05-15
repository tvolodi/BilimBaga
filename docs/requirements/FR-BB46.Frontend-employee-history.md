# FR-BB46 — Frontend: Employee History

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB46 |
| Phase | 4 — Results & Certificates |
| Priority | 2 |
| Status | Implemented |
| Depends On | FR-BB41 |

## Description
Adds a "My Results" tab to the employee portal that presents a paginated, sortable table of all the employee's past exam sessions across all exams. Each row shows key outcome data and provides direct links to the detailed result view and certificate download where applicable. An empty state is shown when the employee has no completed sessions.

## Scope

| Layer | Items |
|-------|-------|
| Database | Read queries on existing `exam_sessions`, `exams`, and `users` tables — no schema changes required |
| API endpoints | `GET /api/v1/portal/results` — new handler, service method, and repository query in `backend/internal/sessions/` |
| Frontend pages/components | `MyResultsPage`, `ResultsTable`, `ResultsTableSkeleton`, `EmptyResults` components; `useMyResults` React Query hook; `PortalLayout` shell with tab navigation hosting both "My Exams" and "My Results" tabs |
| i18n keys | `history.*` key prefix added to `en.json`, `ru.json`, `kk.json`; `common.loadError` key used for API error state |

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
- [ ] AC-11: When the API returns a non-2xx response, a visible inline error message (i18n key `common.loadError`) is displayed and the table is not rendered.

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

#### Portal Shell: `PortalLayout`

The employee portal is refactored to introduce a `PortalLayout` wrapper component that renders top-level tab navigation using the shadcn `Tabs` primitive. Routes are restructured as sub-routes under the `/portal/*` wildcard:

| Tab | Route | Component |
|-----|-------|-----------|
| My Exams | `/portal` or `/portal/exams` | `EmployeePortalPage` (existing) |
| My Results | `/portal/results` | `MyResultsPage` (new) |

`PortalLayout` sits at the `/portal` route level in the React Router config and renders `<Tabs>` with `<TabsList>` containing both tab triggers. The active tab is driven by the current URL so navigating directly to `/portal/results` pre-selects the correct tab. The existing `EmployeePortalPage` is kept unchanged and rendered as the default (`index`) child route.

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
    "meta": {
      "page": 1,
      "per_page": 20,
      "total": 7
    }
  },
  "error": null
}
```

### Backend Implementation

**Package**: `backend/internal/sessions/`

**Handler** (`handler.go`)
```go
// GetMyResults handles GET /api/v1/portal/results
// Reads page, per_page, sort, dir from query params; calls svc.GetMyResults.
func (h *Handler) GetMyResults(w http.ResponseWriter, r *http.Request)
```

**Service interface** (`service.go`)
```go
GetMyResults(ctx context.Context, userID string, page, perPage int, sort, dir string) (*MyResultsResponse, error)
```
`sort` is validated to `"date"` or `"score"`; `dir` to `"asc"` or `"desc"` — invalid values default to `date/desc`.

**Repository query outline** (`repository.go`)
```sql
SELECT
  es.id            AS session_id,
  e.id             AS exam_id,
  e.title          AS exam_title,
  es.submitted_at,
  es.score_pct,
  es.passed,
  es.time_taken_seconds,
  (es.passed AND e.certificate_enabled) AS certificate_available
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
WHERE es.user_id = $1
  AND es.status = 'submitted'
ORDER BY <sort_col> <dir>
LIMIT $2 OFFSET $3;
-- A separate COUNT(*) query (same WHERE) provides the total for the meta object.
```

**Model** (`model.go`) — new types for FR-BB46:
```go
type MyResultsItem struct {
    SessionID          string    `json:"session_id"`
    ExamID             string    `json:"exam_id"`
    ExamTitle          string    `json:"exam_title"`
    SubmittedAt        time.Time `json:"submitted_at"`
    ScorePct           float64   `json:"score_pct"`
    Passed             bool      `json:"passed"`
    TimeTakenSeconds   *int      `json:"time_taken_seconds"`
    CertificateAvailable bool    `json:"certificate_available"`
}

type MyResultsMeta struct {
    Page    int `json:"page"`
    PerPage int `json:"per_page"`
    Total   int `json:"total"`
}

type MyResultsResponse struct {
    Sessions []MyResultsItem `json:"sessions"`
    Meta     MyResultsMeta   `json:"meta"`
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
  meta: {
    page: number;
    per_page: number;
    total: number;
  };
}
```

## Out of Scope

- Admin-facing session history views (covered by FR-BB42 / grading queue and reporting features).
- CSV or PDF export of the results list.
- Server-side search or filtering by exam name (pagination + sort only; text search is a future enhancement).
- Pre-generation or storage of certificate files — certificates are generated on-demand when the download link is clicked.
- Editing or deleting past session records.

## Test Strategy

| Layer | Test Type | What to Test |
|-------|-----------|--------------|
| `useMyResults` hook | Unit (React Testing Library + MSW) | Correct `queryKey`, `placeholderData` behaviour on page/sort change, error state surfaces when API returns 5xx |
| `ResultsTable` | Component | Renders correct column headers; pass badge renders green, fail red; sort toggle calls `onSort` with correct arguments; certificate cell shown/hidden based on `certificate_available` |
| `EmptyResults` | Component | Renders i18n message `history.empty` and CTA link to `/portal/exams` when sessions array is empty |
| `MyResultsPage` | Component (integration) | Error banner rendered with `common.loadError` when `useMyResults` returns error; skeleton shown during loading |
| Backend `GetMyResults` handler | Unit (Go httptest) | 200 with valid token and sessions; 401 without token; 400 for invalid sort/dir values |
| Backend service | Unit | Correct delegation to repository; pagination math (offset = (page-1)*perPage) |

## Notes
- `certificate_available` is a computed boolean from the backend (`passed AND exam.certificate_enabled`) to avoid requiring the frontend to know exam configuration.
- Sort state is stored in URL query params (not component state) so the page is bookmarkable and shareable.
- Time taken is formatted on the frontend using a shared `formatDuration(seconds: number): string` utility, not on the backend.
