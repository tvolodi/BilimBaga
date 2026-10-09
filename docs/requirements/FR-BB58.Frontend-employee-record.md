# FR-BB58 — Frontend: Employee Record

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB58 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB53 |

## Description
Provides examiners and department admins with a comprehensive individual employee record page. Accessible from the users management list, it presents three sections: an employee information header, a paginated session history table, and track-level progress summary cards. Certificate download links appear inline in the session table where applicable. Data is fetched from three endpoints: the existing user profile endpoint, the per-employee record API (FR-BB53), and the per-employee progress API (FR-BB53).

## Scope

| Layer | Items |
|-------|-------|
| Database | Read-only; no schema changes |
| API endpoints | `GET /api/v1/users/:id` (user profile, existing), `GET /api/v1/admin/users/:id/record` (session history, FR-BB53), `GET /api/v1/admin/users/:id/progress` (track progress, FR-BB53), `GET /api/v1/admin/sessions/:id/certificate` (PDF download, existing) |
| Frontend pages/components | `src/pages/admin/EmployeeRecordPage.tsx` (new), `src/components/employees/EmployeeInfoHeader.tsx` (new), `src/components/employees/SessionHistoryTable.tsx` (new), `src/components/employees/TrackProgressCards.tsx` (new), `src/components/employees/ExamStatusChip.tsx` (new), `src/components/employees/EmployeeRecordSkeleton.tsx` (new), `src/api/employees.ts` (new hooks) |
| i18n keys | `employee_record.*` prefix added to `src/locales/en.json`, `src/locales/kk.json`, `src/locales/ru.json` |

## Acceptance Criteria
- [ ] AC-1: The employee record page is accessible via a "View Record" link/button in the admin users list (FR-BB18); the link is visible only to users with `role IN (examiner, department_admin, super_admin)`.
- [ ] AC-2: The employee info header displays: full name, department name, role badge, account status badge (active/inactive), and email address, all sourced from `GET /api/v1/users/:id`.
- [ ] AC-3: The session history table shows at minimum: exam name, date taken, score (percentage), pass/fail badge, time taken, status, certificate download link.
- [x] AC-4: The certificate download link is shown only for rows where a certificate exists (`certificate_id != null`); clicking it calls `GET /api/v1/admin/sessions/:id/certificate` and triggers a file download. If the endpoint returns an error (e.g., `SESSION_NOT_PASSED`, `EXAM_NOT_CERTIFIABLE`), an i18n-keyed toast notification is shown.
- [ ] AC-5: The session history table is displayed sorted by Date Taken descending (server default); no additional client-side sort controls are provided.
- [ ] AC-6: Pagination renders 20 rows per page with Previous/Next controls and a total count label sourced from `meta.total` (the top-level `meta` field of the paginated API response, accessed as `record.meta.total` from the `useEmployeeRecord` hook result).
- [ ] AC-7: Three track progress summary cards (security, safety, loyalty) are displayed in a row; each card shows: track name, `questions_answered`, `last_activity` (relative date), and a list of required exam chips.
- [ ] AC-8: Each required exam chip shows the exam title and a coloured status indicator: green check (passed), red X (failed/not passed), grey dash (never attempted, i.e. `passed = null`).
- [ ] AC-9: The page uses three separate `useQuery` calls — `useUser(userId)` (user profile, re-used from `src/api/users.ts`), `useEmployeeRecord(userId, page)` (session history), and `useEmployeeProgress(userId)` (track progress) — with appropriate `queryKey` arrays.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.
- [ ] AC-11: When any of the three queries returns an error, the page renders an error message (using i18n key `employee_record.load_error`) and a retry button (using i18n key `employee_record.retry`) that re-triggers all failed queries.
- [ ] AC-12: While any of the three queries is in loading state (before first data arrives), `EmployeeRecordSkeleton` is rendered, matching the full page structure (header, table, cards).

## Technical Specification

### Frontend Components

#### Page: `EmployeeRecordPage` (`src/pages/admin/EmployeeRecordPage.tsx`)
- Route: `/admin/users/:userId/record`
- Guarded by `RequireRole(['examiner','department_admin','super_admin'])`
- Uses `useUser(userId)` + `useEmployeeRecord(userId, page)` + `useEmployeeProgress(userId)`
- URL param `page` for pagination via `useSearchParams`
- Renders `EmployeeRecordSkeleton` while any query is loading (AC-12)
- Renders error state with retry button when any query has an error (AC-11)

#### Hooks
```ts
// src/api/employees.ts

// useUser is re-used from src/api/users.ts — no new hook needed for user profile.

// The GET /api/v1/admin/users/:id/record response has `meta` as a TOP-LEVEL sibling
// of `data`, not inside `data`. Standard apiGet helpers only unwrap `body.data`,
// so a local fetchPaginated helper is required to capture both fields.

interface ApiPaginatedResponse<T> {
  data: T;
  meta: PaginationMeta;
  error: null | { code: string; message: string };
}

export interface PaginatedResult<T> {
  data: T;
  meta: PaginationMeta;
}

async function fetchPaginated<T>(url: string): Promise<PaginatedResult<T>> {
  const res = await fetch(url, { credentials: 'include' });
  const body: ApiPaginatedResponse<T> = await res.json();
  if (body.error) throw new Error(body.error.message);
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return { data: body.data, meta: body.meta };
}

export function useEmployeeRecord(userId: string, page: number) {
  return useQuery({
    queryKey: ['employee-record', userId, page],
    queryFn: () =>
      fetchPaginated<EmployeeRecordData>(`/api/v1/admin/users/${userId}/record?page=${page}&per_page=20`),
    placeholderData: keepPreviousData,
  });
}

export function useEmployeeProgress(userId: string) {
  return useQuery({
    queryKey: ['employee-progress', userId],
    queryFn: () => apiGet<EmployeeProgress>(`/api/v1/admin/users/${userId}/progress`),
    staleTime: 5 * 60 * 1000,
  });
}
```

#### Component: `EmployeeInfoHeader` (`src/components/employees/EmployeeInfoHeader.tsx`)
```tsx
// Props: { user: User } — import User from 'src/api/users'
// Layout: avatar with initials + side column
// Role badge: shadcn Badge with variant per role (uses role_name from User)
// Status badge: green "Active" or red "Inactive" (uses status from User)
// Email: mailto link (uses email from User)
// Department: uses department_name from User; if null, render em dash "—"
```

#### Component: `SessionHistoryTable` (`src/components/employees/SessionHistoryTable.tsx`)
```tsx
// Props: { sessions: SessionRecord[] }
// Built on shadcn Table
// Pass/fail badge: <Badge variant="success"> / <Badge variant="destructive"> / <Badge variant="outline"> for grading_pending
//   Labels: employee_record.status_passed / employee_record.status_failed / employee_record.status_grading_pending
// Time taken: formatted with formatDuration utility
// Certificate link: download button, only rendered when certificate_id != null
// No sort controls — table is always sorted by Date Taken descending (server order)
```

#### Component: `TrackProgressCards` (`src/components/employees/TrackProgressCards.tsx`)
```tsx
// Props: { tracks: TrackSummary[] }
// Renders 3 shadcn Card components in a 3-column grid
// Each card: track title (capitalised), questions_answered stat,
//   last_activity (formatDistanceToNow or "Never"),
//   list of ExamStatusChip components
```

#### Component: `ExamStatusChip` (`src/components/employees/ExamStatusChip.tsx`)
```tsx
// Props: { title: string; passed: boolean | null; attempts: number }
// passed = true   → green bg, CheckCircle icon
// passed = false  → red bg, XCircle icon
// passed = null   → grey bg, Minus icon
// Tooltip: "{title} — {attempts} attempt(s)"
```

#### Component: `EmployeeRecordSkeleton`
```tsx
// Skeleton layout matching the full page structure (AC-12)
// Rendered while any of the three useQuery calls is in the initial loading state
// Mirrors the layout: header skeleton + table rows skeleton + 3 card skeletons
```

### Certificate Download for Admin

**Endpoint**: `GET /api/v1/admin/sessions/:id/certificate`  
**Permission**: `exams:read` (enforced at router level; `examiner`, `department_admin`, and `super_admin` hold this permission)  
**Success**: `200 OK`, `Content-Type: application/pdf`, `Content-Disposition: attachment; filename="certificate-{verificationCode}.pdf"`  
**Error codes**:
- `422 SESSION_NOT_SUBMITTED` — session has not been submitted yet
- `422 EXAM_NOT_CERTIFIABLE` — exam is not configured to issue certificates
- `422 SESSION_NOT_PASSED` — session did not reach the passing threshold
- `404 SESSION_NOT_FOUND` — session ID does not exist
- `500 ERR_INTERNAL` — PDF generation failed

```ts
// When examiner downloads certificate from employee record:
async function downloadAdminCertificate(sessionId: string, verificationCode: string) {
  const blob = await apiGetBlob(`/api/v1/admin/sessions/${sessionId}/certificate`);
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `certificate-${verificationCode}.pdf`;
  a.click();
  window.URL.revokeObjectURL(url);
}
```

### i18n Keys Required

> All keys below must be added to all three locale files: `src/locales/en.json`, `src/locales/kk.json`, and `src/locales/ru.json`.

```json
{
  "employee_record.title": "Employee Record",
  "employee_record.info_section": "Employee Information",
  "employee_record.history_section": "Session History",
  "employee_record.progress_section": "Track Progress",
  "employee_record.col_exam": "Exam",
  "employee_record.col_date": "Date Taken",
  "employee_record.col_score": "Score",
  "employee_record.col_status": "Status",
  "employee_record.col_time": "Time Taken",
  "employee_record.col_certificate": "Certificate",
  "employee_record.download_certificate": "Download",
  "employee_record.history_empty": "No exam sessions recorded for this employee.",
  "employee_record.track_security": "Security",
  "employee_record.track_safety": "Safety",
  "employee_record.track_loyalty": "Loyalty",
  "employee_record.questions_answered": "Questions Answered",
  "employee_record.last_activity": "Last Activity",
  "employee_record.never": "Never",
  "employee_record.required_exams": "Required Exams",
  "employee_record.exam_passed": "Passed",
  "employee_record.exam_failed": "Not Passed",
  "employee_record.exam_not_attempted": "Not Attempted",
  "employee_record.attempts": "{{count}} attempt(s)",
  "employee_record.status_passed": "Passed",
  "employee_record.status_failed": "Failed",
  "employee_record.status_grading_pending": "Grading Pending",
  "employee_record.showing": "Showing {{from}}–{{to}} of {{total}}",
  "employee_record.load_error": "Failed to load employee record.",
  "employee_record.retry": "Retry"
}
```

### TypeScript Types

```ts
// EmployeeRecordData — the `data` field of GET /api/v1/admin/users/:id/record response.
// IMPORTANT: The response envelope for this endpoint has `meta` as a TOP-LEVEL sibling
// of `data` (not nested inside `data`). Standard apiGet helpers only unwrap `body.data`
// and would silently lose `meta`. Use fetchPaginated<EmployeeRecordData> (defined above)
// which returns PaginatedResult<EmployeeRecordData> = { data: EmployeeRecordData, meta: PaginationMeta }.
// Components access sessions via result.data.sessions and pagination via result.meta.total.
// Note: user profile fields (email, full_name, department_name, role_name, status) are NOT
// included here; they come from GET /api/v1/users/:id via the existing useUser() hook and
// the existing User type in src/api/users.ts.
export interface EmployeeRecordData {
  sessions: SessionRecord[];
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
}

export interface SessionRecord {
  session_id: string;
  exam_id: string;
  exam_title: string;
  started_at: string;
  submitted_at: string | null;
  score_pct: number | null;
  passed: boolean | null;
  time_taken_seconds: number | null;
  status: string;
  certificate_id: string | null;
}

export interface EmployeeProgress {
  user_id: string;
  full_name: string;
  tracks: TrackSummary[];
}

export interface TrackSummary {
  track: 'security' | 'safety' | 'loyalty';
  questions_answered: number;
  last_activity: string | null;
  required_exams: RequiredExam[];
}

export interface RequiredExam {
  exam_id: string;
  title: string;
  passed: boolean | null;
  attempts: number;
}
```

## Out of Scope

- Editing employee details from this page (handled by the Users management pages).
- Deleting or archiving sessions from this page.
- Certificate revocation or re-issuance.
- Exporting session history to CSV from this page (separate requirement if needed).
- Any changes to the backend API contracts for `/record`, `/progress`, or the certificate endpoint.
- Push notifications or real-time updates when a new session is submitted.

## Test Strategy

| Layer | Approach |
|-------|----------|
| Unit — hooks | Mock `fetchPaginated` / `apiGetBlob`; verify correct query keys, correct URL construction with `page` param (no sort/dir), `placeholderData` behaviour on page change; assert both `data.sessions` and `meta.total` are accessible from hook result. |
| Unit — components | Render `SessionHistoryTable` with fixture data; assert pass/fail badge variants and i18n labels, certificate link visibility, sort header click handlers; render `ExamStatusChip` for all three `passed` states. |
| Unit — `EmployeeInfoHeader` | Render with a `User` fixture; assert `role_name`, `status`, `email`, `department_name` are displayed; assert mailto href. |
| Unit — `EmployeeRecordSkeleton` | Assert it renders without errors and contains header, table, and card skeleton sections. |
| Integration — `EmployeeRecordPage` | Use React Testing Library + MSW to mock all three endpoints; assert all three sections render; assert skeleton appears before data resolves; assert error state + retry button appears when any query fails; assert retry button re-fires the failed queries. |
| i18n | Assert no hardcoded user-visible strings remain in components; verify all `employee_record.*` keys exist in all three locale files. |
```

## Notes
- User profile data (name, department, role, status, email) is fetched via the existing `useUser(userId)` hook (`GET /api/v1/users/:id`) and the existing `User` type from `src/api/users.ts`; `EmployeeRecordData` only carries the sessions array — pagination meta comes from the top-level `meta` field and is accessed via `PaginatedResult<EmployeeRecordData>.meta`.
- Track progress cards use `staleTime: 5 * 60 * 1000` since progress changes only when sessions are submitted; avoiding over-fetching on each pagination change.
- The "View Record" entry point in the users list is a router link added to the existing `UsersListPage` (FR-BB18 frontend) as part of this requirement's implementation.
