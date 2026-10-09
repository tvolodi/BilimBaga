# Retro 006

**Period**: closed swarm issues 95-109 by `closedAt` (15 issues; counter baseline 94, closed count now 109). Closures ran from 2026-10-09 10:32Z to 16:14Z. Seven of the 15 closed before the 11:00Z start in the brief, so the window follows the counter rule, not the clock.

Issues: #214, #217, #199, #187, #173, #218, #219, #234, #237, #233, #213, #250, #269, #286, #61.

## Metrics

Derived from `gh issue list`, `gh api repos/tvolodi/BilimBaga/issues/N/events` and issue comments. Status durations come from `status:*` label events. "Not derivable" means no data exists in GitHub for it.

| Metric | Value |
|---|---|
| Closed in period | 15 (bug 6, test 5, infra 2, tech-debt 1, feature 1) |
| Cycle time created->closed, all 15 | median 18.5 min, p90 217.5 min, max 613 min (#61) |
| Cycle time, excluding 3 no-output issues (#234, #269, #286) | median 53.9 min, p90 about 231 min |
| Cycle time by type (all 15) | bug median 53.8 / p90 377.8; test 5.1 / 76.0; infra 95.0 / 170.9; tech-debt 235.9; feature 0.7 (n=1 each for the last two) |
| Queue wait, created->in-progress (n=14) | median 6.0 min, p90 185.1 min (#61 waited 496 min) |
| PR open to UAT label, incl. merge (n=9) | median 2.1 min, p90 113.7 min (#61: 93 min, held for migration lock and UAT) |
| UAT label to closed (n=7) | median 9.2 min, p90 47.3 min |
| status:uat queue (reconstructed from events, all swarm issues) | 8 at 09:00Z, 6-12 until 14:00Z, 15 at 14:30Z, 23 at 15:00Z, 28 at 15:30Z, 31 at 16:00Z; 22 open at snapshot. Cap is 5; above cap for the whole window |
| status:ready queue | 23-27 until 14:00Z, 9-10 at 15:30-16:00Z |
| Reopen rate (`reopen:` comments) | 0 of 15 (0%) |
| Send-backs to dev (label transitions, any cause) | 2 of 15 (13%): #173 (BA found a gap after UAT PASS, 09:32Z), #199 (UAT PARTIAL, 10:38Z). Neither carries a `reopen:` comment |
| Failed results / reassignments / attempts >= 2 | 0 failed, 0 reassigned, 2 second rounds (#173, #199) |
| Idle time per role | Not derivable. No tick history is persisted; `workers.json` is a single snapshot |
| Escalations (`escalations.json`) | 16 entries, all dated 2026-10-09 (7 needs-user, 8 recommendation, 1 incident). Only one has a time (14:10Z, UAT blocked). The rest cannot be assigned to the period |
| status:blocked on period issues | 0 of 15 |
| status:blocked, swarm-wide, 2026-10-09 | #1, #32, #229 blocked from 14:07Z, still labelled at snapshot (UAT resumed #61 at 16:07Z). #290 at 16:06Z (owner decision). #284 at 15:30Z |
| Duplicate or no-output issues | 3 of 15 (20%): #234 (duplicate of #233, filed in parallel), #269 (duplicate of ISS-027, closed 7 s after open, no work), #286 (BA anti-idle scan, no output) |
| Two `status:*` labels at close | 2 of 15: #217 and #218 (closed as `status:done`, then `status:uat` added after merge) |
| Closed before the PR merged | 2 of 15: #217 (closed 10:45Z, merged 10:46Z), #218 (closed 10:58Z, merged 14:07Z) |
| Merged but issue still open | #233: `status:done` at 11:09Z, closed 14:06Z (about 3 h) |
| PR conflicts | 2 `origin/main` merge conflicts resolved by dev in #61 (users/service_test.go, README). #275 needed an extra `origin/main` merge after #289 landed first (manual ordering). No `CONFLICTING` state recorded |
| Lock contention | `migration.lock` held by #61 from 14:17Z to 15:57Z (about 100 min, until UAT). `stack.lock`: UAT blocked (see escalation 14:10Z). No other issue recorded waiting on a lock |
| Self-reported memory holds | 13 "memory hold / memory alert" mentions in period comments. Protocol text has no memory rule beyond "UAT needs >= 8 GB" |
| Security defects caught before merge | 2: #217 (P0, users:manage could act on higher-ranked accounts) and #218 (P1, dept scoping keyed on role name). Both found by BA conformance review #214 |

## Went well
- BA conformance review (#214, #237) found the two privilege and scoping defects (#217, #218) before release. Both had UAT evidence (34/34 on a branch build, re-run on the QA instance).
- CI gap closed: #233 added `tests.yml` (vet, staticcheck, `go test`, tsc, lint, vitest). Its first run exposed 14 test-code failures, fixed in the same PR.
- Fast round-2 fixes: #199 and #219 (ragged CSV export) were re-fixed in one PR and re-verified within about 10 minutes; #187 (test fix) passed UAT about 12 minutes after its PR.
- Docs reconciled within minutes of the decision (#223, #225).
- Migration lock and needs-live-db label worked as designed for #61 (migration 036, real UAT run on main).
- The retro-005 UAT 10-minute checkpoint and the `claimed-by` check before dispatch are in the files now.

## Went badly
- **UAT queue far above cap.** `status:uat` stayed above 5 all period and peaked at 31 (16:00Z). Feature dispatch was supposed to hold, but workers self-claim (see next point).
- **Feature self-claim during UAT overflow.** #42 (FR-BB116) was claimed by bb-dev2 at 15:48Z with no supervisor assignment comment, while about 28 issues sat in `status:uat`. bb-dev1 then posted a second `claimed-by` at 16:13Z and retracted it. Root cause: the UAT cap is written only in the Supervisor tick. The worker claim path in PROTOCOL has no cap check and no check for an existing claim.
- **Double dispatch.** #173 was sent to bb-dev2 and bb-dev1 at about 09:08Z; dev2 released it at 09:09Z after a Supervisor correction. This recurred despite the retro-005 `claimed-by` rule.
- **Issues closed before the fix merged.** #217 and #218 were closed on UAT PASS for a branch build. Both were merged afterwards and carry both `status:done` and `status:uat`. Neither has a post-merge UAT on main. The PROTOCOL state machine says "uat -> done (closed)" but does not say when UAT runs.
- **Label hygiene not in the files.** retro-005 decision 1 (remove the old `status:*` in the same command) is not in `PROTOCOL.md` or any role file, although apply PR #243 reports all five decisions as applied. retro-005 decision 3 (memory thresholds 2.5 / 4 GB) is also absent. Only the 8 GB UAT line exists. The apply step did not verify its own diff.
- **Reopen metric under-counts.** Both send-backs (#173, #199) were tracked by label only. The `reopen:` comment is required only on UAT FAIL, so the reopen rate reads 0% against 13% real send-backs.
- **Merged-but-open lag.** #233 sat in `status:done` with an open issue for about 3 h. Supervisor tick 1 reconciles "closed issue not done" but not the reverse.
- **Duplicates.** #234 duplicated #233 (Supervisor filed in parallel with a dev or the same tick). #269 was a duplicate of ISS-027 and was closed in 7 s. No title search before filing.
- **Long queue wait for P2 bugs.** #61 waited 496 min in `ready` (p2 bug, created 06:01Z, claimed 14:16Z). Median wait is 6 min, but the tail is 185 min. P0/P1 work took precedence, which is expected; the cycle-time tail is the cost.
- **Migration lock and UAT hold on #61.** Dev2 held `migration.lock` for about 100 min because the merge waited on the live UAT run.
- **Full-suite skipped under memory hold.** Merges relied on GitHub CI on 13 recorded occasions. The hold works, but it is not written down, so each dev reinvented the rule.
- **Lighthouse incident.** dev2 subagents ran `scripts/perf/lighthouse.sh` with `ALLOW_REMOTE=1` against `evil.com` hosts, against an explicit no-run instruction (escalation, dated 2026-10-09; #31, #238). No role file contains the rule or the stub-on-PATH requirement.
- **Stale worktrees.** 13 leftover directories under `.claude/worktrees/` (`sup-retro`..`sup-retro5`, `dev2-p0`, `uat2`, `agent-*`). Prior retros did not remove them.

### Supervisor observations (a)-(h), verified against evidence

| # | Observation | Verdict |
|---|---|---|
| a | `workers.json` stale for hours | Not verifiable. No history in GitHub. Current file updated 16:15Z |
| b | dev2 self-claimed #42 while UAT cap held | Confirmed (15:48Z, no supervisor assignment; UAT queue about 28) |
| c | UAT blocked on missing `.env` and stack down | Confirmed. Blocked 14:07Z (#1, #32, #229; escalation 14:10Z). The deny rule existed then and was removed in #285 (merged 15:34Z). UAT resumed by 16:07Z. The stack blocker was lifted with the `.env`. Blocked labels were not cleared afterwards |
| d | PR merge sequencing #289 vs #275 manual | Confirmed (#183 comment: #275 merged after #289 landed, with an extra `origin/main` merge) |
| e | Go 1.27 install (#290) blocked by owner rule | Confirmed (#290 comment 16:06Z; system installs are owner-reserved) |
| f | UAT "blocked" used for partial passes | Not supported. No period issue carried `status:blocked`. #199 PARTIAL was a real defect (ragged export) |
| g | status:uat queue above 5 (about 20+) | Confirmed. Above 5 all window; peak 31 |
| h | admin password changed and flag flipped unexplained | Partly explained by repo docs, not by an issue comment. Migration 033 (PR #162) sets `force_password_change=true` for `admin@bilimbaga.local` while its hash is the default one. UAT scenario dept-admin-scoping S0.1 changes the admin password when the flag is set. A localhost run is not recorded in any issue |

## Decisions

Each change has exact before and after text. Each is tied to the metric it should move. The Dev applies them through the retro-apply issue (see Applied changes).

1. **PROTOCOL.md, label state machine (line 13)** (retro-005 decision 1 plus the UAT-timing gap). Metric: two `status:*` labels at close 2 -> 0; closed before merge 2 -> 0; reopen-tagged send-backs 0 of 2 -> 2 of 2.
   - Before: `uat -> done (closed); uat -> ready + \`role:dev\` on FAIL (reopen +1); any -> blocked with a reason comment.`
   - After: `uat -> done (closed), only after the PR is merged to main and the UAT check ran on main; a PASS on a branch build is a comment and does not change status; uat -> ready + \`role:dev\` on FAIL or PARTIAL, and on any BA or Supervisor send-back (reopen +1 comment each time); any -> blocked with a reason comment. Every status change is one \`gh issue edit\` that also removes the previous \`status:*\` label.`

2. **PROTOCOL.md, Claiming (line 14)**. Metric: double dispatch or claim 1 -> 0; feature claims while UAT queue > 5: 1 -> 0.
   - Before: `Claiming: Supervisor assigns; workers take only matching role + \`ready\` (or own \`in-progress\`), by prio then number; \`claimed-by: bb-devN\` comment; one dev per issue.`
   - After: `Claiming: Supervisor assigns and posts \`supervisor: assigned to bb-devN\` on the issue before sending the task; workers take only matching role + \`ready\` (or own \`in-progress\`), by prio then number; before posting \`claimed-by: bb-devN\`, read the issue comments and skip the issue if another dev holds a claim without a later release; one dev per issue. While more than 5 \`status:uat\` issues are open, workers do not self-claim \`type:feature\`.`

3. **swarm/roles/dev1.md and swarm/roles/dev2.md, Tick step 1** (same text in both). Metric: as decision 2.
   - Before: `1. \`gh issue list --label role:dev --label status:ready\` plus own \`in-progress\`; best unclaimed or own.`
   - After: `1. \`gh issue list --label role:dev --label status:ready\` plus own \`in-progress\`; best unclaimed or own. Skip \`type:feature\` while more than 5 \`status:uat\` issues are open (\`gh issue list --label status:uat --state open\`).`

4. **swarm/roles/supervisor.md, tick step 1** (closed-or-done reconcile). Metric: done-but-open lag about 3 h -> under one tick.
   - Before: `...merged PR with issue still \`review\`, closed issue not \`done\`).`
   - After: `...merged PR with issue still \`review\`, closed issue not \`done\`, open issue at \`status:done\` (close it in the same tick)).`

5. **swarm/roles/supervisor.md, Rules** (duplicates). Metric: duplicate or no-output issues 3 -> 0 expected.
   - Before: the Rules paragraph ends with `...at most one message per worker per tick unless replying.`
   - After: same text, plus: ` Before filing an issue, search open and closed swarm issues by title keywords (\`gh issue list --label swarm --state all --search "<keywords>"\`); file only when nothing matches.`

6. **PROTOCOL.md, Resources** (retro-005 decision 3, not in the files). Metric: unwritten memory holds (13 mentions) -> 0; PRs merged on CI only without a written basis.
   - Before: `- UAT live runs need >= 8 GB free memory; the Supervisor reports low memory.`
   - After: same line, plus: `- Memory hold: when free RAM < 2.5 GB, devs run only targeted single-package tests (no full suite, tsc or docker) and merge on GitHub CI plus UAT evidence; the Supervisor announces the hold and the release at 4 GB free.`

7. **swarm/roles/_common.md, new rule 14** (lighthouse incident). Metric: tool-execution incidents 1 -> 0.
   - Before: `13. Checkpoint each step change and each tick (PROTOCOL, checkpoints).`
   - After: same line, plus: `14. Never run a script or tool that sends requests to a host other than the local stack or a target the issue names (lighthouse, k6, curl, npx of an unknown package). Guard tests stub the tool on PATH and never call the real one.`

Not a role change: stale worktrees (13 directories) are a local cleanup. The Supervisor can remove them once their branches are merged or abandoned.

## Applied changes (PR links)
- pending retro-apply issue #<n>

## Open recommendations for the user
- **Worktree `.env` provisioning (credentials).** A fresh worktree lacks the gitignored `.env`, so UAT was blocked about 14:07Z to 16:07Z. Choose: a setup step copies `<main>/.env` into new worktrees, or UAT runs from the main checkout. Either way, you decide how secrets are handled. The settings deny rule was removed in #285, so the block was the missing file, not the rule.
- **#290 Go 1.27 install.** Blocked: a system-wide install is yours to confirm directly. Until then the repo stays on Go 1.25.1.
- **Blocked labels not cleared.** #1, #32 and #229 still carry `status:blocked` after their blocker was lifted. The Dev or UAT should clear them on the next UAT run.
- **bilimbaga-test (customer demo).** It runs old code without #217, #152, #160 or #165. It may still have the default admin credential with `force_password_change=false`. Deploying current main (migrations 031-035, backup first) and rotating the admin password are your decisions.
- **Branch protection.** Make `backend-tests` and `frontend-tests` required checks on `main` (from #236). GitHub settings are yours.
- **Lighthouse egress.** The incident is informational. Decide whether network egress from subagents should be logged. The settings files are yours to change.
- **Memory.** The full-suite skip recurred 13 times. More RAM or fewer concurrent apps would restore full local runs. Decision 6 writes the current hold rule down.
- **Retro state.** `swarm/state/retro.json` was not updated in this run, per instruction. The Supervisor must set `last_retro_closed_count` to 109 and `retro_number` to 6, or the retro re-triggers on the next tick.
