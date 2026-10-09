# Code Review: ISS-232 (insights cost guard)

Result: PASS

Checks run: `go vet ./internal/ai ./internal/config ./cmd/...` clean; `go test -p 2 -race ./internal/ai ./internal/config` pass.

## Findings
- [Medium] backend/go.mod:27 - `golang.org/x/sync v0.18.0` is marked `// indirect` but is imported directly by internal/ai/service.go. `go mod tidy -diff` moves it to the direct require block. -> run `go mod tidy` (cosmetic, build is unaffected).
- [Medium] backend/internal/ai/service.go:~216 - The singleflight key (tenant|exam|scopeKey) omits `forceRefresh`. A `?refresh=true` caller that joins an in-flight non-refresh call can get the cache-hit result the leader returned (leader re-checks cache inside the flight) instead of a fresh paid result. Not a cost/security issue (cost-safe); minor staleness vs AC-4. -> optionally add refresh to the key.
- [Medium] backend/internal/ai/service.go (generateInsights) - The cap is check-then-act against ai_usage_log, which is written after the Anthropic call. Concurrent requests for different exams by the same user can overshoot the cap by the concurrency level. Acceptable for a cost guard; note it as a known limitation.
- [Low] Followers of a flight are not charged to their own daily cap (only the leader's user is checked/logged). Intended by coalescing (one paid call), but a follower with an exhausted cap still receives the result.
- [Low] Usage-log failure is logged only, so that paid call is not counted toward the cap (pre-existing pattern).
- [Low] backend/internal/ai/scope_cache_test.go:175 - the edited comment is garbled ("(was: no limit; GenerateQuestions calls checkRateLimit)") -> tidy the wording.

## Checklist notes
- No secrets; no os.Getenv in handlers/services (config via typed Config, wired in main.go); no SQL concatenation (parameterized, UTC time).
- Errors wrapped with context; 429 AI_RATE_LIMITED uses the standard envelope; handler stays thin; SQL is in the repository.
- Config: negative values rejected, 0 disables, default 50; .env.example updated; no migration needed (ai_usage_log reused; durable scope_key migration explicitly deferred, documented).
- Detached shared context is safe: the Anthropic client has a 60s HTTP timeout; caller cancellation is handled via select on ctx.Done(). Result copies per caller avoid shared-slice races.
- Scope-lookup failure fails safe (no cache, no coalescing, cap still enforced).
- Tests cover cap exceeded/under/disabled/default, cache hits free, refresh capped, singleflight pays once (-race), distinct scopes, cancelled caller, 429 envelope, repo SQL, config parsing.

## Summary
No Critical or High findings; PASS with minor cleanups (go mod tidy, optional refresh in flight key).
