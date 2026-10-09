---
id: ISS-008
title: Tech debt - resolve TODO/FIXME markers and go vet findings in backend
status: resolved
severity: low
layer: backend
module: tenant
tags: [TODO, FIXME, go vet, staticcheck]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/router/router_test.go (TestRouter_ProtectedRoutesRequireBearerToken, existing)
---

## Symptom
Issue #8: audit TODO/FIXME/XXX/HACK markers and `go vet` findings in backend/.

## Root Cause
One stale TODO; the rest are false positives.

## Fix Applied
Marker table (grep `TODO|FIXME|XXX|HACK` over backend/*.go):

| file:line | marker text | action |
|-----------|-------------|--------|
| backend/internal/tenant/handler.go:48 | `TODO: add super_admin RBAC middleware once FR-BB15/FR-BB16 are implemented.` | removed as obsolete - `PUT /tenant/config` is already behind `rbac.RequirePermission(rbacCache, "tenant", "manage")` (internal/router/router.go:79); comment replaced with a factual note. Route auth is covered by the existing router test. |
| backend/internal/auth/handler_test.go:494 | `...notavalidhashXXXX...` | not a marker (dummy bcrypt string); no action |

No FIXME/HACK markers exist. No account-recovery TODOs found (recovery_* untouched). No follow-up issues were needed.

`go vet ./...`: zero findings. `gofmt -l`: only pre-existing CRLF files in internal/tenant (not touched beyond one comment line). `staticcheck` (already installed) reports pre-existing S1016 (struct literal vs conversion) and U1000 (unused code) style findings across many packages (e.g. ratelimit/middleware.go:15 `retryAfterSeconds`, sessions/service.go:603 `parseTagIDs`, reports/service.go:53 `queryResult`). These are not TODO/vet items, would touch many files (some under in-flight work) and were left out of scope; recommend a separate cleanup issue.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/tenant/handler.go | comment only (no behaviour change) |

## Regression Test
None added - comment-only change, no behaviour change; route protection already covered by TestRouter_ProtectedRoutesRequireBearerToken.

## Resolution Results
- Tests: `go test -p 2 ./...` all packages ok
- Migration applied: no
- Build clean: yes (`go vet ./...` clean)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
