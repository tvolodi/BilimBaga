# FR-BB56 Inner Report — Admin Dashboard UI

**Run ID**: FR-BB56
**Pipeline**: A — Feature Development
**Date**: 2026-05-16
**Status**: Implemented and committed

---

## Summary

FR-BB56 delivers the front-end admin analytics dashboard. The dashboard
aggregates four KPI metrics, a department-level exam-completion bar chart,
an overdue-assignments table with a one-click remind action, and a
paginated recent-activity feed. A matching backend stub route was added so
the remind call does not 404 in development.

---

## Acceptance Criteria Coverage

| AC | Description | Result |
|----|-------------|--------|
| AC-1 | KPI cards: active users, exams scheduled, pass rate, certs issued | Implemented via `KpiRow` + `KpiCard` components |
| AC-2 | Completion bar chart per department | Implemented via `CompletionBarChart` (Recharts `BarChart`) |
| AC-3 | Overdue assignments table with remind button | Implemented via `OverdueTable`; calls `POST /admin/users/{userId}/remind` |
| AC-4 | Recent activity feed (last N events) | Implemented via `RecentActivityFeed` |
| AC-5 | Accessible to admin and hr_admin roles | Route guard in `App.tsx` includes `hr_admin` |
| AC-6 | Full i18n (en / kk / ru) | All user-visible strings use `t('dashboard.*')` keys |
| AC-7 | No hardcoded strings | Confirmed — zero literal strings in components |
| AC-8 | TypeScript strict mode | 0 errors from `tsc --noEmit` |

---

## Files Changed

### Backend

| File | Change |
|------|--------|
| `backend/internal/users/handler.go` | Added `RemindEmployee` stub handler returning 204 |
| `backend/internal/router/router.go` | Registered `POST /admin/users/{userId}/remind` |

### Frontend (new)

| File | Purpose |
|------|---------|
| `frontend/src/api/dashboard.ts` | React Query hooks: `useDashboardKpis`, `useDashboardChart`, `useDashboardOverdue`, `useDashboardActivity`, `useRemindEmployee` |
| `frontend/src/components/dashboard/KpiCard.tsx` | Single metric card with label, value, and optional trend |
| `frontend/src/components/dashboard/KpiRow.tsx` | Four-column grid of `KpiCard` instances |
| `frontend/src/components/dashboard/CompletionBarChart.tsx` | Recharts bar chart showing completion % per department |
| `frontend/src/components/dashboard/OverdueTable.tsx` | Paginated table of overdue assignments; remind button per row |
| `frontend/src/components/dashboard/RecentActivityFeed.tsx` | Scrollable activity feed with timestamp and actor |
| `frontend/src/pages/admin/AdminDashboardPage.tsx` | Top-level page composing all dashboard widgets |
| `frontend/src/components/dashboard/KpiCard.test.tsx` | 3 tests |
| `frontend/src/components/dashboard/OverdueTable.test.tsx` | 5 tests |
| `frontend/src/components/dashboard/RecentActivityFeed.test.tsx` | 6 tests |
| `frontend/src/pages/admin/AdminDashboardPage.test.tsx` | 9 tests |

### Frontend (modified)

| File | Change |
|------|--------|
| `frontend/src/App.tsx` | Added `/admin/dashboard` route; changed admin index redirect; added `hr_admin` to outer RBAC guard |
| `frontend/src/locales/en.json` | Added `dashboard.*` namespace |
| `frontend/src/locales/kk.json` | Added `dashboard.*` namespace |
| `frontend/src/locales/ru.json` | Added `dashboard.*` namespace |
| `frontend/package.json` | Added `recharts` and `date-fns` dependencies |
| `frontend/package-lock.json` | Lockfile updated |

### Docs

| File | Change |
|------|--------|
| `docs/requirements/FR-BB56.Frontend-admin-dashboard.md` | Status → Implemented; all ACs checked |
| `docs/requirements/README.md` | FR-BB56 row status updated to Implemented |

---

## Test Results

| Suite | Files | Tests | Failures |
|-------|-------|-------|----------|
| Backend (`go test ./...`) | 18 packages | all pass | 0 |
| Frontend (vitest) | 23 | 123 | 0 |
| TypeScript (`tsc --noEmit`) | — | — | 0 errors |

---

## Migrations

None required. The dashboard reads existing tables through the existing
reports and audit APIs. The remind endpoint is a no-op stub (HTTP 204)
with no DB writes at this stage.

---

## Commit

```
47f2fef feat(dashboard): implement FR-BB56 admin dashboard UI
```

---

## Notes

- `recharts` added as a production dependency; `date-fns` added for
  relative-time formatting in the activity feed.
- The remind button is functional end-to-end (HTTP call + success toast)
  but the backend stub always returns 204 with no side effects until the
  notification service is implemented in a later phase.
- No existing tests were broken by this change.
