# Retro 002

**Period**: closed issues 17-32 (since retro 001).

## Metrics
| Metric | Value |
|---|---|
| Closed in period | 16 (UAT-closed 12, dev 2, infra 1, other 1) |
| Cycle time created->closed | median 31 min, p90 45 min, max 60 min |
| Reopens | 1 (#39: UAT FAIL against an outdated expectation; BA decision fixed it, no code change) |
| Failed results / reassignments | 0 |
| Real product defects caught by UAT/review | 6 (#37 Bearer on downloads, #58 Fragment key, #75 dashboard export 500, #38 reports, #82 ai queries, #66 exports) |
| Escalations to user | 0 new (DB backup recommendation from retro 001 still open) |
| Incidents | memory pressure killed live E2E once (#1); ISS report-number collision (parallel subagents both took ISS-055); stale idle notices |
| Idle | BA idle by design after retro 001 cap; UAT became the bottleneck (6-7 issues waiting at times) |

## Went well
- Retro 001 changes applied in ~15 min (#68); worktree-only rule removed repo-root contention.
- Schema drift found: queries filtered on nonexistent `tenant_id`; dev audited 245 SQL statements and added `internal/schemaguard` test.
- Conditional merge-ok (merge when security-check green) removed a polling round-trip.
- UAT ran targeted live specs under memory pressure instead of waiting.
- Port parametrization (#12) unblocked live E2E without touching a foreign process.

## Went badly
- UAT is the single throughput bottleneck: 3 roles' output (2 devs + infra) funnel into one live-stack verifier on a memory-limited host.
- Most dev tests ran against fake SQL drivers; real-schema bugs (tenant_id, order_num, category_id) only surfaced via live DB. Fake-driver tests give false confidence.
- ISS-NNN numbering collides under parallel subagents.
- Remaining ~24 E2E tests of #1 and #2 still not run (needs >8 GB free).
- Idle notices still arrive stale (handled by rule, but noisy).
- BA 2-issue cap lives only in supervisor.md.

## Decisions (apply via `retro-apply` issue)
1. Add a CI/test step (dev, `schemaguard`-style) that validates SQL column references against migrations for all packages; extend beyond ai (moves: live-only schema defects -> 0).
2. PROTOCOL: ISS report numbers allocated by `docs/issue-reports/README.md` lock or derived from the GitHub issue number (ISS-<issue#>) to avoid collisions.
3. PROTOCOL/uat.md: dev PRs that change SQL must include a migration-applied Postgres test (testcontainers or the compose db) or mark `needs-live-db` so UAT verifies first, not last.
4. Supervisor: cap UAT queue; when >5 issues wait at status:uat, hold new feature dispatches to devs and give devs tech-debt/test work instead (moves: UAT backlog).
5. ba.md: write the self-generated cap (max 2 open) into the role file.

## Applied changes
Applied in issue #88 (PR `swarm/88-retro-apply-2`):
1. `internal/schemaguard` now scans all of `backend/internal` + `backend/cmd`, adds a coverage guard (`TestEveryDBCallingFileIsScanned`), a dynamic-SQL fragment check (audit repository) and an unqualified-column check for multi-table queries.
2. PROTOCOL section 9: report numbers = `ISS-<github issue number>`.
3. PROTOCOL section 10 + `roles/uat.md`: SQL-changing PRs need a real-Postgres test or the `needs-live-db` label; UAT verifies those first.
4. `roles/supervisor.md`: UAT queue cap (>5 at status:uat -> hold feature dispatch, devs get tech-debt/test work).
5. `roles/ba.md`: self-generated cap (max 2 open).

## Open recommendations for the user
- bilimbaga-test DB has no scheduled backup (carried over).
- ~24 E2E tests remain unrun: they need >8 GB free RAM; close other apps (HandBrake, BitTorrent, Firefox use several GB) during UAT or allow a swap/memory increase for WSL.
- Consider increasing UAT capacity (second UAT session needs a second stack and more RAM).
