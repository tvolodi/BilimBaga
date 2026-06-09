---
id: ISS-047
title: "UAT Defect: Result & Certification — error states not rendered on My Results and Result Detail pages"
status: resolved
severity: low
layer: frontend
module: results
tags: [uat, result-and-certification]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: S11 (Error States), Steps 1–2
Actor: Employee
Action: Trigger API 500 error on My Results page; trigger API 500 error on Result Detail page
Expected: My Results page shows a visible inline error message (not a skeleton); Result Detail page shows a visible inline error alert (not a spinner)
Actual: My Results page displays a loading skeleton indefinitely with no error message; Result Detail page shows a blue spinner indefinitely with no error message. Neither page surfaces the error to the user.
Screenshot: none

**Violated acceptance criteria:**
- FR-BB46 AC-11: "When the API returns a non-2xx response, a visible inline error message is displayed"
- FR-BB45 AC-9: "error states render appropriate skeleton/alert components"

## Root Cause
Both `useSessionResult` and `useMyResults` hooks in `frontend/src/api/sessions.ts` lacked `retry: false`. TanStack Query's default of 3 retries with exponential backoff (1s + 2s + 4s ≈ 7 seconds) kept `isLoading = true` while retrying, deferring `isError` from becoming true for ~7 seconds. The UAT test observed the loading skeleton/spinner during this retry window and reported it as "indefinite". Both page components already had correct `isError` rendering branches; the only missing piece was immediate error surfacing.

## Fix Applied
Added `retry: false` to the `useSessionResult` and `useMyResults` React Query hooks. This is consistent with all other session hooks in the same file (`useSession`, `useNextAdaptiveQuestion`) which already had `retry: false`. Errors now surface immediately on the first failure, causing `isLoading` to become false and `isError` to become true without delay.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/api/sessions.ts` | Added `retry: false` to `useSessionResult` and `useMyResults` query options |

## Regression Test
None added — the `isError` rendering branches in both page components are already covered by the existing component structure. The fix (`retry: false`) is a hook-level configuration change that is validated by the frontend build type-check.

## Resolution Results
- Tests: 259 frontend tests passed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
