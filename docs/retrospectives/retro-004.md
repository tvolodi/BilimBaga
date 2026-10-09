# Retro 004

**Period**: closed issues 64-78 (15 issues; counter baseline 63).

## Metrics
| Metric | Value |
|---|---|
| Closed in period | 15 (bug 7, test 5, tech-debt 2, infra 1) |
| Cycle time created->closed | median 36 min, p90 86 min, max 191 min |
| Reopens | 1 (#131: seed fixture sent `body` instead of `text`, found by the first live run of its specs) |
| Failed results / reassignments | 0 |
| Own-merge regressions | 0 new (docker-build + Playwright-list CI added after retro 003) |
| Real defects caught by UAT/review before merge | 9 (privilege escalation in role assignment (#185), seed-guard bypass, tsc fail #142, CSV empty export #163, department scoping gaps #165, blank option text #173, email case #164, admin default password #152, force_password_change not enforced #160) |
| False positives | 1 (#158, closed unmerged; a dev almost added API surface for a non-existent param) |
| Open PRs waiting on UAT | 4 (#180, #177, #182, #185), 5th (#186) merged |
| Escalations to user | 9 open entries (see Open recommendations) |

## Went well
- Customer issue #7 fully handled: 3 bugs fixed or in PR, role management designed (FR-BB117) and built within hours.
- Security posture improved a lot: default-admin password hole found on public QA, fixed (#162) and server-side enforcement (#180) in review; privilege escalation caught in review cycle 2.
- UAT proposed (and Supervisor accepted) an integration branch to test PRs together: queue 5 -> 1 run.
- Dev discipline: devs refused unsafe actions (login on frozen host, force-push), held merges for needs-live-db, deviated from my instructions when they were wrong (#142).
- Persistence work (roster, checkpoints) started; heartbeat files in use.

## Went badly
- UAT serialisation remains the throughput limit (4-5 PRs at once); host memory 2-4 GB free all period.
- Supervisor over-specified details (e.g. 403 vs BA's 404) and sent conflicting guidance twice; BA decisions should be read before dev instructions.
- Review nits sometimes become follow-up issues faster than they are drained (backlog ~20 ready).
- No scheduled backup on bilimbaga-test, default admin possibly present there (user decision).

## Decisions (apply via `retro-apply`)
1. Supervisor: before sending a design instruction on an issue, read the BA decision comments on it first; BA decisions win.
2. UAT: integration-branch testing is the default when >= 3 PRs wait (document in uat.md).
3. Supervisor: backlog cap, do not file p2 follow-ups while > 15 ready dev issues exist unless security/correctness; fold small nits into one grouped issue per area.
4. PROTOCOL: false-positive findings: a UAT/BA finding must cite the endpoint contract (doc or code line) before it becomes a dev issue.
5. Dev: needs-live-db PRs must include the UAT recipe in a PR comment (already practice; make it a rule).

## Applied changes
Applied in #193: decisions 1-5 in swarm/roles/supervisor.md, uat.md, dev1.md, dev2.md and PROTOCOL.md section 4 (plus dev rule: check claimed-by and git worktree list before claiming).

## Open recommendations for the user
- bilimbaga-test (customer demo): scheduled DB backup missing; possible default admin credential; legacy mixed-case emails check; next deploy needs migrations 031-034 (api self-migrates).
- Anthropic API key missing for AI features.
- Scheduled Task registration for swarm persistence (#145) is yours to run; deploy key and kill drill likewise.
- CLAUDE.md references three guides that do not exist; `make migrate` needs a subcommand (#146).
- infra.settings.json allows Bash(ssh *) with no deny for the protected host.
- Host memory: free RAM 2-4 GB; close other apps or add RAM for faster UAT lanes.
