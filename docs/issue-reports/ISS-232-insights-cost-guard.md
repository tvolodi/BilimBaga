---
id: ISS-232
title: AI insights cost guard - no per-user cap, no request coalescing on GetInsights
status: resolved
severity: medium
layer: backend
module: ai
tags: [GetInsights, singleflight, AI_INSIGHTS_DAILY_LIMIT, AI_RATE_LIMITED, ai_usage_log, scopedInsightCache]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-165, ISS-218]
regression_test: backend/internal/ai/insights_costguard_test.go
---

## Symptom
GET /admin/ai/insights/{examId} (issues #232, #184): only GenerateQuestions had a per-user limit; every cold or `?refresh=true` request called Anthropic, and N concurrent cold requests for the same exam + scope made N paid calls.

## Root Cause
`aiService.GetInsights` had no rate check and no request coalescing. Scoped (non-super_admin) results were only cached in-process after the call finished.

## Fix Applied
- Per-user daily cap on paid insight calls: `AI_INSIGHTS_DAILY_LIMIT` (typed `Config.AIInsightsDailyLimit`, default 50, 0 disables, negative rejected) passed via `ai.WithInsightsDailyLimit`. Counted from `ai_usage_log` (feature `exam_insights`, last 24 h) through new `Repository.CountAIUsageLastDay`, same mechanism as GenerateQuestions. Cache hits are free; paid calls (cold or refresh) are checked. Exceeded -> `ErrAIRateLimited` -> HTTP 429 `{data:null,error:{code:"AI_RATE_LIMITED",message}}`.
  Deliberately service-level rather than an `internal/ratelimit` HTTP middleware: that package is per-IP/per-minute and counts cache hits; the cap must count only paid calls and survive restarts/replicas.
- `singleflight` (golang.org/x/sync) keyed tenant|exam|scopeKey around the paid path; the shared call is detached from the leader's cancellation (`context.WithoutCancel`), re-checks the cache inside the flight, and each caller gets a copy. Scope-lookup failure skips coalescing and cache (fail safe).
- Item 3 (durable scope_key migration, PK (exam_id, scope_key)) is DEFERRED: `swarm/locks/migration.lock` is held by dev2 (#61, migration 036). The in-process scope-keyed cache from ISS-218 stays. Follow-up needed with needs-live-db.
- #184 leftover `RequireUserInScope`: no longer exists in the code (already removed); nothing to do. GET /users exact-department vs reports subtree alignment not addressed here.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/ai/service.go | daily cap, singleflight, options, GetInsights split into cache/limit/generate |
| backend/internal/ai/repository.go | CountAIUsageLastDay |
| backend/internal/ai/handler.go | 429 AI_RATE_LIMITED mapping for insights |
| backend/internal/config/config.go, backend/.env.example, backend/cmd/api/main.go | AI_INSIGHTS_DAILY_LIMIT wiring |
| backend/go.mod | golang.org/x/sync direct |

## Regression Test
backend/internal/ai/insights_costguard_test.go (cap exceeded/under/disabled/default, cache hits free, refresh capped, singleflight pays once with -race, distinct scopes not coalesced, cancelled caller, 429 envelope, CountAIUsageLastDay SQL), backend/internal/config/config_ai_test.go, handler error-mapping case.

## Resolution Results
- Tests: `go test -p 2 ./...` all pass (incl. internal/schemaguard); `go test -race ./internal/ai` pass
- Migration applied: no (deferred)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
