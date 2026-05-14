# FR-BB58 — Frontend: Employee Record

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB58 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB53 |

## Description
Provides HR admins and examiners with a comprehensive individual employee record page. Accessible from the users management list, it presents three sections: an employee information header, a paginated session history table, and track-level progress summary cards. Certificate download links appear inline in the session table where applicable. All data is fetched from the per-employee record API (FR-BB53).

## Acceptance Criteria
- [ ] AC-1: The employee record page is accessible via a "View Record" link/button in the admin users list (FR-BB18); the link is visible only to users with `role IN (examiner, hr_admin, super_admin)`.
- [ ] AC-2: The employee info header displays: full name, department, role badge, account status badge (active/inactive), and email address.
- [ ] AC-3: The session history table shows at minimum: exam name, date taken, score (percentage), pass/fail badge, time taken, status, certificate download link.
- [ ] AC-4: The certificate download link is shown only for rows where a certificate exists (`certificate_id != null`); clicking it calls `GET /admin/sessions/:id/certificate` and triggers a file download.
- [ ] AC-5: The session history table supports sorting by "Date Taken" (default descending) and "Score"; sort state is reflected in URL query parameters.
- [ ] AC-6: Pagination renders 20 rows per page with Previous/Next controls and a total count label.
- [ ] AC-7: Three track progress summary cards (security, safety, loyalty) are displayed in a row; each card shows: track name, `questions_answered`, `last_activity` (relative date), and a list of required exam chips.
- [ ] AC-8: Each required exam chip shows the exam title and a coloured status indicator: green check (passed), red X (failed/not passed), grey dash (never attempted, i.e. `passed = null`).
- [ ] AC-9: The page uses two separate `useQuery` calls — one for `useEmployeeRecord(userId, page, sort, dir)` and one for `useEmployeeProgress(userId)` — with appropriate `queryKey` arrays.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.

## Technical Specification

### Frontend Components

#### Page: `EmployeeRecordPage` (`src/pages/admin/EmployeeRecordPage.tsx`)
- Route: `/admin/users/:userId/record`
- Guarded by `RequireRole(['examiner','hr_admin','super_admin'])`
- Uses `useEmployeeRecord(userId, page, sort, dir)` + `useEmployeeProgress(userId)`
- URL params for sort/pagination via `useSearchParams`

#### Hooks
```ts
// src/api/employees.ts

export function useEmployeeRecord(
  userId: string,
  page: number,
  sort: 'date' | 'score',
  dir: 'asc' | 'desc'
) {
  return useQuery({
    queryKey: ['employee-record', userId, page, sort, dir],
    queryFn: () =>
      apiGet<EmployeeRecord>(`/admin/users/${userId}/record?page=${page}&per_page=20&sort=${sort}&dir=${dir}`),
    placeholderData: keepPreviousData,
  });
}

export function useEmployeeProgress(userId: string) {
  return useQuery({
    queryKey: ['employee-progress', userId],
    queryFn: () => apiGet<EmployeeProgress>(`/admin/users/${userId}/progress`),
    staleTime: 5 * 60 * 1000,
  });
}
```

#### Component: `EmployeeInfoHeader` (`src/components/employees/EmployeeInfoHeader.tsx`)
```tsx
// Props: { user: EmployeeRecordUser }
// Layout: avatar with initials + side column
// Role badge: shadcn Badge with variant per role
// Status badge: green "Active" or red "Inactive"
// Email: mailto link
```

#### Component: `SessionHistoryTable` (`src/components/employees/SessionHistoryTable.tsx`)
```tsx
// Props: { sessions: SessionRecord[]; onSort: fn; sortCol: string; sortDir: string }
// Built on shadcn Table
// Pass/fail badge: <Badge variant="success"> / <Badge variant="destructive"> / <Badge variant="outline"> for grading_pending
// Time taken: formatted with formatDuration utility
// Certificate link: download button, only rendered when certificate_id != null
// Sortable column headers
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
// Skeleton layout matching the full page structure
// Used while both queries are loading
```

### Certificate Download for Admin

```ts
// When examiner downloads certificate from employee record:
async function downloadAdminCertificate(sessionId: string, verificationCode: string) {
  const blob = await apiGetBlob(`/admin/sessions/${sessionId}/certificate`);
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `certificate-${verificationCode}.pdf`;
  a.click();
  window.URL.revokeObjectURL(url);
}
```

### i18n Keys Required

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
  "employee_record.attempts": "{{count}} attempt(s)"
}
```

### TypeScript Types

```ts
export interface EmployeeRecord {
  user_id: string;
  full_name: string;
  department: string | null;
  role: string;
  status: 'active' | 'inactive';
  email: string;
  sessions: SessionRecord[];
  total_count: number;
  page: number;
  per_page: number;
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

## Notes
- The employee info header user data (name, department, role, status, email) is embedded in the `GET /admin/users/:id/record` response to avoid a separate user-profile API call.
- Track progress cards use `staleTime: 5 * 60 * 1000` since progress changes only when sessions are submitted; avoiding over-fetching on each pagination change.
- The "View Record" entry point in the users list is a router link added to the existing `UsersListPage` (FR-BB18 frontend) as part of this requirement's implementation.
