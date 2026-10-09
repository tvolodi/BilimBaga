---
id: ISS-054
title: AuditLogTable logs React "unique key" warning (bare fragment in entries.map)
status: resolved
severity: low
layer: frontend
module: audit
tags: [unique "key" prop, AuditLogTable, Fragment, entries.map, TableBody]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-052]
regression_test: frontend/src/components/audit/AuditLogTable.test.tsx
---

## Symptom
Console error on /admin/audit: `Each child in a list should have a unique "key" prop ... Check the render method of TableBody. It was passed a child from AuditLogTable.` Fails E2E test 22 (no console errors on main admin screens). GitHub issue #58.

## Root Cause
`entries.map` in `AuditLogTable.tsx` returned a bare `<>...</>` fragment. Keys were set on the inner `TableRow`s, but React needs the key on the outermost element returned from the map callback, i.e. the fragment.

## Fix Applied
Imported `Fragment` from react, wrapped each entry in `<Fragment key={entry.id}>`, and removed the now-redundant `key` props on the inner rows.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/components/audit/AuditLogTable.tsx | Keyed Fragment; dropped inner keys |
| frontend/src/components/audit/AuditLogTable.test.tsx | New regression test |

## Regression Test
`AuditLogTable.test.tsx` renders four entries, expands two (main + meta rows), and asserts `console.error` was never called / no "unique key" warning. Verified it fails against the unfixed component.

## Resolution Results
- Tests: 336 passed, 0 failed (59 files)
- Migration applied: no
- Build clean: yes (tsc --noEmit, eslint)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
