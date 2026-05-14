# FR-BB56 — Frontend: Admin Dashboard

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB56 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB51 |

## Description
Implements the main admin landing page that surfaces KPI metrics, a completion rate bar chart, an overdue employees table, and a recent activity feed. All data is fetched from the dashboard metrics API (FR-BB51) with a 5-minute stale window. HR admins and examiners land on this page after login. The page uses recharts for the bar chart visualisation and shadcn primitives for all other UI.

## Acceptance Criteria
- [ ] AC-1: The dashboard page is the default landing page for users with `role IN (examiner, hr_admin, super_admin)` after login.
- [ ] AC-2: Four KPI cards are displayed in a row: "Total Employees", "Active Exams", "Completion Rate (last 30 days)", "Pass Rate (last 30 days)"; each card shows an icon, a numeric value, and a label.
- [ ] AC-3: "Completion Rate" is computed on the frontend as `SUM(completed_count) / SUM(assigned_count)` across `completion_rate_by_exam`; displayed as a percentage with one decimal place.
- [ ] AC-4: "Pass Rate" is computed as `SUM(passed_count) / SUM(assigned_count)` across `completion_rate_by_exam`.
- [ ] AC-5: The completion rate bar chart uses `recharts BarChart`; X-axis shows exam titles (truncated to 20 chars); Y-axis shows percentage 0–100%; bars use the tenant's `primary_color`.
- [ ] AC-6: The overdue employees table shows name, exam name, and deadline; the deadline is formatted as a human-readable relative date (e.g. "3 days ago") using `date-fns`.
- [ ] AC-7: Each overdue row has a "Send Reminder" action button; clicking it calls `POST /api/v1/admin/users/:userId/remind` with `{ exam_id }` and shows a success/error toast; the endpoint must exist (stub if not yet implemented).
- [ ] AC-8: The recent activity feed renders the last 20 sessions as a scrollable list; each item shows employee name, exam name, score badge (coloured by pass/fail), and relative time.
- [ ] AC-9: The page uses `useQuery` with `queryKey: ['dashboard']` and `staleTime: 5 * 60 * 1000`; a manual "Refresh" button invalidates the query.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.

## Technical Specification

### Frontend Components

#### Page: `AdminDashboardPage` (`src/pages/admin/AdminDashboardPage.tsx`)
- Route: `/admin/dashboard` (also `/admin` redirect target)
- Guarded by `RequireRole(['examiner','hr_admin','super_admin'])`
- Uses `useDashboard()` hook

#### Hook: `useDashboard`
```ts
// src/api/dashboard.ts
export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: () => apiGet<DashboardMetrics>('/admin/dashboard'),
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}
```

#### Component: `KpiCard` (`src/components/dashboard/KpiCard.tsx`)
```tsx
// Props: { icon: ReactNode; label: string; value: string | number; description?: string }
// Built on shadcn Card
// Icon rendered in a rounded square with bg-primary/10 colour
```

#### Component: `KpiRow` (`src/components/dashboard/KpiRow.tsx`)
```tsx
// Props: { data: DashboardMetrics; employeeCount: number; activeExamCount: number }
// Renders 4 KpiCard components in a CSS grid (grid-cols-4, gap-4)
// Derives completionRate and passRate from completion_rate_by_exam array
```

#### Component: `CompletionBarChart` (`src/components/dashboard/CompletionBarChart.tsx`)
```tsx
// Props: { data: ExamCompletionRate[]; primaryColor: string }
// Uses recharts BarChart (responsive container, 100% width, 280px height)
// Two bars per exam: completedCount (solid primary) and assignedCount (muted background)
// Custom tooltip showing exam title, completed/assigned, pass rate
// X-axis: exam title truncated; tooltip shows full title
```

#### Component: `OverdueTable` (`src/components/dashboard/OverdueTable.tsx`)
```tsx
// Props: { employees: OverdueEmployee[] }
// Renders shadcn Table
// Deadline cell: relative time via formatDistanceToNow(new Date(deadline))
// "Send Reminder" button per row → useRemindEmployee mutation
```

#### Hook: `useRemindEmployee`
```ts
export function useRemindEmployee() {
  return useMutation({
    mutationFn: ({ userId, examId }: { userId: string; examId: string }) =>
      apiPost(`/admin/users/${userId}/remind`, { exam_id: examId }),
    onSuccess: () => toast.success(t('dashboard.reminder_sent')),
    onError: () => toast.error(t('dashboard.reminder_error')),
  });
}
```

#### Component: `RecentActivityFeed` (`src/components/dashboard/RecentActivityFeed.tsx`)
```tsx
// Props: { activities: RecentActivity[] }
// Scrollable list (max-h-96 overflow-y-auto)
// Each item: avatar with initials, name + exam name, score badge, relative time
// Score badge: green if passed, red if failed, grey if grading_pending
```

### KPI Calculation (frontend)

```ts
function computeKpis(data: DashboardMetrics) {
  const totalAssigned = data.completion_rate_by_exam.reduce((s, e) => s + e.assigned_count, 0);
  const totalCompleted = data.completion_rate_by_exam.reduce((s, e) => s + e.completed_count, 0);
  const totalPassed = data.completion_rate_by_exam.reduce((s, e) => s + e.passed_count, 0);

  const completionRate = totalAssigned > 0
    ? ((totalCompleted / totalAssigned) * 100).toFixed(1) + '%'
    : '—';
  const passRate = totalAssigned > 0
    ? ((totalPassed / totalAssigned) * 100).toFixed(1) + '%'
    : '—';

  return { completionRate, passRate };
}
```

### i18n Keys Required

```json
{
  "dashboard.title": "Dashboard",
  "dashboard.kpi_employees": "Total Employees",
  "dashboard.kpi_active_exams": "Active Exams",
  "dashboard.kpi_completion_rate": "Completion Rate",
  "dashboard.kpi_pass_rate": "Pass Rate",
  "dashboard.kpi_last_30_days": "Last 30 days",
  "dashboard.chart_title": "Completion Rate by Exam",
  "dashboard.chart_completed": "Completed",
  "dashboard.chart_assigned": "Assigned",
  "dashboard.overdue_title": "Overdue Employees",
  "dashboard.overdue_empty": "No overdue assignments.",
  "dashboard.col_employee": "Employee",
  "dashboard.col_exam": "Exam",
  "dashboard.col_deadline": "Deadline",
  "dashboard.send_reminder": "Send Reminder",
  "dashboard.reminder_sent": "Reminder sent successfully.",
  "dashboard.reminder_error": "Failed to send reminder.",
  "dashboard.activity_title": "Recent Activity",
  "dashboard.activity_empty": "No recent activity.",
  "dashboard.refresh": "Refresh"
}
```

### TypeScript Types

```ts
export interface DashboardMetrics {
  completion_rate_by_exam: ExamCompletionRate[];
  overdue_employees: OverdueEmployee[];
  recent_activity: RecentActivity[];
  avg_score_by_track: { security: number | null; safety: number | null; loyalty: number | null };
}

export interface ExamCompletionRate {
  exam_id: string;
  title: string;
  assigned_count: number;
  completed_count: number;
  passed_count: number;
}

export interface OverdueEmployee {
  user_id: string;
  name: string;
  exam_title: string;
  deadline: string;
}

export interface RecentActivity {
  session_id: string;
  employee_name: string;
  exam_title: string;
  score_pct: number | null;
  passed: boolean | null;
  submitted_at: string;
}
```

## Notes
- `recharts` must be added to `package.json` if not already present: `npm install recharts`.
- `date-fns` must be added for relative time formatting: `npm install date-fns`.
- "Total Employees" and "Active Exams" KPIs require separate lightweight API calls (`GET /admin/users?count=true` and `GET /admin/exams?status=active&count=true`) or can be embedded in the dashboard response — coordinate with the backend team which approach is preferred.
- The "Send Reminder" stub endpoint can return HTTP 200 with an empty data payload until the notification system is built in a later phase.
