# Retro 005

**Period**: closed issues 79-94 (16 issues; counter baseline 78).

## Metrics
| Metric | Value |
|---|---|
| Closed in period | 16 (bug 9, test 3, feature 2, tech-debt 2) |
| Cycle time created->closed | median 106 min, p90 154 min, max 172 min (up from 36 min in retro 004: UAT queue and memory holds) |
| Reopens / sent back by UAT | 3 (#173 translations PUT still blank, #176 I-10 returned 400 instead of 413, #206 delete 204 parsed as JSON) |
| Failed results | 0 |
| Own-merge regressions | 0 on main; 2 caught by UAT before merge (#176, #206) |
| Security defects caught before merge | 3 (privilege escalation in role assignment (#185), CSV formula injection (#191), department scoping gaps (#165)) |
| Customer issue #7 | 4/4 addressed: exam-start errors (#132), users role filter (#133), questions language/versions (#134), roles management (FR-BB117, #185+#206) |
| Escalations to user | ~12 open entries, no new categories |

## Went well
- UAT integration branch of 6 PRs in one run found 4 semantic merge hazards before any merge; strict sequence (#180 -> #177 -> #182 -> #194 -> #185) merged with zero main breakage.
- Independent review cycles caught real bugs (escalation, CSV injection, blank option text, 413 behaviour).
- Devs honoured low-memory holds and merged on GitHub CI plus UAT evidence without local runs.
- needs-live-db label and TEST_DATABASE_URL tests gave real-Postgres evidence for migrations 033/034/035.

## Went badly
- Host memory 1.5-2.5 GB all period; UAT and devs serialised; cycle time tripled.
- Supervisor double-assigned #173 once and over-specified once more; mitigated by the retro-004 rule.
- Some issues carried two status labels (ready+in-progress/uat); needed manual fixes five times.
- Follow-up issue volume still high; 19 ready dev issues.
- Heartbeat files stale during long UAT runs (UAT checkpoints only per step).

## Decisions (apply via `retro-apply`)
1. Label hygiene: a worker setting a new status:* label removes the old one in the same command (gh issue edit --remove-label ... --add-label ...); document in PROTOCOL s4.
2. UAT checkpoint at least every 10 minutes during long runs (sub-step progress) so the Supervisor heartbeat rule stays meaningful.
3. Memory protocol: when free RAM < 2.5 GB, devs stop running tests/tsc/docker (edit-only) and merge on GitHub CI plus UAT evidence; Supervisor announces the hold and release thresholds (2.5 / 4 GB) in PROTOCOL s10.
4. Supervisor: before assigning an issue, check the issue's `claimed-by` comments and the worker checkpoints to avoid double assignment.
5. Follow-up issues: group nits per area (rule exists); cap ready dev queue reporting; BA drains by closing stale p2s monthly.

## Applied changes
Pending: issue "retro-005: apply swarm improvements".

## Open recommendations for the user
- bilimbaga-test (customer demo): backup, default admin credential, legacy mixed-case emails, blank option text queries, next deploy applies migrations 031-035 (api self-migrates; 035 never aborts startup).
- Anthropic API key; Scheduled Task registration (#145); CLAUDE.md missing guides; infra.settings.json ssh deny.
- RAM: more memory or fewer concurrent apps would roughly halve cycle time.
