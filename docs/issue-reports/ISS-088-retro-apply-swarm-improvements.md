---
id: ISS-088
title: retro-002 apply swarm improvements (schemaguard extension, protocol rules)
status: resolved
severity: low
layer: config
module: audit
tags: [schemaguard, retro-002, needs-live-db, ISS numbering]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-82]
regression_test: backend/internal/schemaguard/schema_guard_test.go
---

## Symptom
Retro-002 decisions 1-5 pending: schema drift only found via live DB, ISS report numbers colliding, UAT queue bottleneck, BA cap only in supervisor.md.

## Root Cause
Process gaps. The existing schemaguard already walked all `backend/internal` packages, but nothing prevented a DB-calling file from escaping it (non-literal SQL), and multi-table queries with unqualified columns were unchecked.

## Fix Applied
- schemaguard: scans `internal` + `cmd`; `TestEveryDBCallingFileIsScanned` fails if a file with DB calls and SQL text yields no collected statement (allowlist with reason); `TestDynamicSQLFragments` checks `alias.column` tokens in `audit/repository.go` (runtime-built SQL); unqualified WHERE columns in multi-table queries must exist in some referenced table or be an `AS` alias; limits documented in package doc.
- Docs: PROTOCOL (ISS-<issue#> numbering, SQL PR rule, `needs-live-db` label), uat.md (verify needs-live-db first), supervisor.md (UAT queue cap), ba.md (max 2 self-generated), retro-002 applied section.
- Created GitHub label `needs-live-db`.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/schemaguard/schema_guard_test.go | coverage guard, dynamic fragment test, multi-table unqualified check, new tests |
| swarm/PROTOCOL.md, swarm/roles/{uat,supervisor,ba}.md | decisions 2-5 |
| docs/retrospectives/retro-002.md | applied changes |

## Regression Test
`TestEveryDBCallingFileIsScanned`, `TestDynamicSQLFragments`, `TestCheckQuery_FlagsUnqualifiedColumnInJoin`.

## Findings
No real schema mismatches found across all packages (10 files' worth of 300+ statements). Only false positives met during development (window aliases `rn`, `top_rank`, `bot_rank`), handled by the `AS alias` rule.

## Resolution Results
- Tests: `go test -p 2 ./...` all packages pass
- Migration applied: no
- Build clean: yes (build, vet)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
