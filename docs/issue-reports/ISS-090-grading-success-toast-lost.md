---
id: ISS-090
title: Grading success toast never visible after submit (FR-BB47 AC-9)
status: resolved
severity: low
layer: frontend
module: exams
tags: [GradingDetailPage, handleSubmitAll, toast, navigate, GradingQueuePage]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-060]
regression_test: frontend/src/pages/admin/GradingDetailPage.test.tsx
---

## Symptom
GradingDetailPage.handleSubmitAll called a local-state toast then navigate('/admin/grading'); the component unmounted so the toast was never seen (GitHub #84).

## Root Cause
Toast state lived in the unmounting detail page; no shared toast mechanism exists in the app.

## Fix Applied
Detail page navigates with state `{ gradingSuccess: true }`. GradingQueuePage reads it into local state on mount, renders a dismissible `role="status"` banner (auto-dismiss 5s, i18n `grading.success_toast` / `grading.dismiss_toast`), and clears the history state with a replace-navigation so refresh/back do not re-show it.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/pages/admin/GradingDetailPage.tsx | pass nav state instead of local success toast |
| frontend/src/pages/admin/GradingQueuePage.tsx | render/clear one-shot success banner |
| frontend/src/locales/{en,ru,kk}.json | grading.dismiss_toast |
| frontend/src/pages/admin/GradingDetailPage.test.tsx, GradingQueuePage.test.tsx | assert AC-9 end to end |

## Regression Test
GradingDetailPage.test.tsx "submits sequentially..." now renders the queue page and asserts the toast and dismissal.

## Resolution Results
- Tests: 422 passed, 0 failed
- Migration applied: no
- Build clean: yes (tsc, lint, check:i18n)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
