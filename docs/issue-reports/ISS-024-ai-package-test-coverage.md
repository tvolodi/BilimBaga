---
id: ISS-024
title: backend/internal/ai low test coverage (client, insights/loyalty handlers, repository)
status: resolved
severity: low
layer: backend
module: ai
tags: [coverage, httpAnthropicClient, HandleGetInsights, GetLoyaltyNarrative, fakedb]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-082, ISS-093]
regression_test: backend/internal/ai/client_test.go
---

## Symptom
Issue #24: `go test -cover ./internal/ai/` reported 36.9% (baseline doc). At branch start actual coverage was 53.6%; client.go, HandleGetInsights, GetLoyaltyNarrative and most repository funcs were untested.

## Root Cause
Missing tests. No production defect.

## Fix Applied
Tests only, no production code changed. Seam for the Anthropic HTTP client: tests build `httpAnthropicClient` directly with an `http.Client` whose Transport rewrites the host to an `httptest.Server` (the client hard-codes the URL; no production seam needed). Repository funcs use the existing `fakedb_test.go` driver; handlers use an injectable Service stub.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/ai/client_test.go | new: GenerateText success/non-200/bad JSON/API error/transport error/cancelled ctx/nil ctx; prompt builders |
| backend/internal/ai/handler_insights_loyalty_test.go | new: HandleGetInsights + GetLoyaltyNarrative auth, params, error mapping, {data,error} envelope |
| backend/internal/ai/repository_more_test.go | new: all remaining repository funcs (scan mapping, NULL handling, polarity inversion, error wrapping) |
| backend/internal/ai/service_branches_test.go | new: service error/non-fatal branches |

## Regression Test
Above files.

## Resolution Results
- Coverage ./internal/ai: 53.6% (branch start; 36.9% in baseline doc) -> 94.5%
- `go vet ./...` clean, `go test -p 2 ./...` all green
- Migration applied: no. Build clean: yes. SQL production code unchanged (schemaguard green).

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
