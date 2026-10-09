---
id: ISS-210
title: Results CSV export buffers whole result without a row cap; stale "without buffering" comment
status: resolved
severity: medium
layer: backend
module: reports
tags: [StreamExamResultsCSV, EXPORT_TOO_LARGE, EXPORT_MAX_ROWS, writeBufferedCSV]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-163, ISS-141]
regression_test: backend/internal/reports/export_cap_test.go
---

## Symptom
GET /admin/exams/{id}/results/export loads every session into memory (service) and again into csvBuffer (handler) with no bound; comments claimed "without buffering". FR-BB54 AC-8 requires the export to be bounded.

## Root Cause
ISS-163 made the handler buffer the body, and the service collects all session rows to batch-fetch scores; no cap was added.

## Fix Applied
- `reports.NewService(repo, opts...)` with `WithMaxExportRows(n)` (default `DefaultMaxExportRows` = 200000). Scanning stops at the cap and returns `ErrExportTooLarge`.
- Handler maps it to 422 `EXPORT_TOO_LARGE` ({data:null,error:{code,message}}) with a hint to narrow by date range.
- Typed `Config.ExportMaxRows` from `EXPORT_MAX_ROWS` (>=1), documented in `.env.example`, wired in `cmd/api/main.go`.
- Stale comments fixed.
- G2: non-UUID exam id is 404 via `RequireUUIDPathParams`; added the route to the router test.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/reports/service.go | cap, option, error, comments |
| backend/internal/reports/handler.go | 422 mapping |
| backend/internal/config/config.go | ExportMaxRows |
| backend/cmd/api/main.go | wiring |
| backend/.env.example | EXPORT_MAX_ROWS |
| backend/internal/reports/export_cap_test.go | cap/boundary/default/handler tests |
| backend/internal/config/config_ai_test.go | config test |
| backend/internal/router/router_test.go | non-UUID export route 404 |

## Regression Test
export_cap_test.go (service + handler), router_test.go (G2).

## Resolution Results
- Tests: go test -p 2 ./... all pass
- Migration applied: no
- Build clean: yes (go vet clean)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
