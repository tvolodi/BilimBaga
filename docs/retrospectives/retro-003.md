# Retro 003

**Period**: closed issues 33-63 (31 issues; retro 002 counter was 32; this retro ran late because I compared against 62 instead of 32+15=47: the trigger check must use `retro.json.last_retro_closed_count + interval`).

## Metrics
| Metric | Value |
|---|---|
| Closed in period | 31 (bug 10, test 6, infra 6, tech-debt 6, feature 3) |
| Cycle time created->closed | median 26 min, p90 101 min, max 113 min |
| Reopens | 0 |
| Failed results / reassignments | 0 |
| Regressions introduced by our own merges | 3 (#112: playwright global-setup CJS break; #112: frontend Docker build break (TS2307); #86: malformed verification code 500) |
| Real customer bugs found | 4 (issue #7) -> 3 root-caused in 1 UAT cycle (#132, #133, #134), 1 = missing feature (FR-BB117) |
| Escalations to user | 5 open (see below) |

## Went well
- Customer complaints (issue #7) split, reproduced locally with root causes and handed to devs within ~15 min; the freeze of bilimbaga-test (production-class) was applied immediately, QA instance bilimbaga-qa built by Infra and live.
- Independent review caught real defects (seed-guard bypass via encoded hostnames, tsc failure in #142, missing Authorization in ai.ts).
- needs-live-db label and UAT-first rule exist and are used; migration lane serialised via lock.
- BA cap worked; BA output moved to conformance reviews and decisions (DEC-001, FR-BB117).
- Checkpoint/heartbeat protocol (PROTOCOL s11) merged.

## Went badly
- Three regressions were merged because checks were static (unit tests with fake drivers, no docker build, no real playwright start). A docker-build CI job now exists (#127).
- Retro trigger miscomputed (above): process error of the Supervisor.
- UAT is still the bottleneck: 4-6 issues waiting at status:uat; one live stack, branch checks serial; migration.lock held for ~45+ min by one PR waiting for UAT, blocking #137/#150.
- Host memory 3.4-5 GB free: full live E2E (#1 remainder, #2) never completed.
- Dev self-merge when UAT is slow (e.g. #142 merged before re-check) is acceptable only when SQL unchanged; keep as explicit rule.

## Decisions (apply via `retro-apply`)
1. supervisor.md/RETRO.md: retro trigger = closed_count - retro.json.last_retro_closed_count >= interval (never compare absolute counts).
2. PROTOCOL s5: migration.lock holder must release it within 30 min of PR open if UAT has not picked it up: UAT checks lock-holding PRs first; Supervisor breaks stale lock after 60 min (existing rule) only if PR is not open.
3. PROTOCOL: a PR's own SQL unchanged since UAT PASS may be merged without a second UAT pass (document the condition); any SQL change needs-live-db again.
4. CI: add Playwright smoke (list + one local spec) and docker-build to required checks for PRs touching scripts/, frontend/e2e, Dockerfile, deploy/ (partly done in #127).
5. UAT capacity: second UAT lane using a second local stack on another port (18081) when memory allows >= 8 GB; otherwise keep dev work to tech-debt when status:uat > 5 (rule exists, enforce).
6. Supervisor: run a post-merge smoke (build + docker build) on main after each merge batch (can be a CI workflow on push to main).

## Applied changes
Pending: issue "retro-003: apply swarm improvements".

## Open recommendations for the user
- bilimbaga-test (customer demo, production-class): no scheduled DB backup; seeded admin has default password and force_password_change=false (fix shipped for new deploys via #150, existing instance is your call); next deploy there needs migrations 031/032 applied (api self-migrates at start).
- Anthropic API key absent: FR-BB71/74/75 live narrative cannot be exercised.
- swarm/roles/infra.settings.json allows Bash(ssh *) without a deny for the protected host.
- Scheduled Task registration (swarm persistence #145) is yours to run.
- Host memory: consider closing HandBrake/BitTorrent/Firefox during UAT, or more RAM for WSL.
