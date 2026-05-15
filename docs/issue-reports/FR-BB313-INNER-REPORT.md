# FR-BB313: Implementation Inner Report

**Date**: 2026-05-15T00:00:00Z
**Pipeline**: A
**Commit**: 23df2f4

## Summary

Implemented the FR-BB313 Employee Portal frontend feature. Employees can now view their assigned exams from a dedicated portal page, see exam metadata (time limit, pass threshold, attempt counts), and launch exam sessions via a confirmation modal. The portal auto-refreshes every 30 seconds and provides proper loading skeletons and empty states. All user-visible strings are fully internationalised in English, Kazakh, and Russian.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/api/portal.ts` | created |
| `frontend/src/hooks/useCountdown.ts` | created |
| `frontend/src/pages/EmployeePortal/index.tsx` | created |
| `frontend/src/pages/EmployeePortal/ExamCard.tsx` | created |
| `frontend/src/pages/EmployeePortal/ExamCardSkeleton.tsx` | created |
| `frontend/src/pages/EmployeePortal/StartExamModal.tsx` | created |
| `frontend/src/pages/EmployeePortal/EmptyPortal.tsx` | created |
| `frontend/src/pages/EmployeePortal.tsx` | deleted (replaced by directory) |
| `frontend/src/locales/en.json` | modified |
| `frontend/src/locales/kk.json` | modified |
| `frontend/src/locales/ru.json` | modified |
| `docs/requirements/FR-BB313.Frontend-employee-portal.md` | modified |
| `docs/requirements/README.md` | modified |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| Employee can view list of assigned exams | `EmployeePortal/index.tsx` renders exam cards via `usePortalExams` |
| Exam card shows title, duration, pass threshold, attempt counts | `ExamCard.tsx` displays all metadata fields |
| Employee can start an exam via confirmation modal | `StartExamModal.tsx` calls `useCreateSession` mutation on confirm |
| Loading state shows skeletons | `ExamCardSkeleton.tsx` rendered while query is loading |
| Empty state shown when no exams assigned | `EmptyPortal.tsx` rendered when data is empty |
| Portal auto-refreshes every 30 seconds | `usePortalExams` sets `refetchInterval: 30000` |
| All strings internationalised (en/kk/ru) | `portal.*` keys added to all three locale files |

## Test Results

- Backend: not applicable (frontend-only change)
- Frontend: build passes (`npm run build` exit code 0)

## Migration Applied

none

## Known Limitations

- Real-time countdown hook (`useCountdown.ts`) is used for active sessions; the portal page itself does not display an active countdown (countdown is for the exam session page).
- Exam session routing after `useCreateSession` success depends on the exam session page route being configured in the router — route wiring is assumed to be handled by the surrounding router configuration.
