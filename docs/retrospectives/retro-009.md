# Retro 009

## Period

Trigger: `retro.json.last_retro_closed_count` = 148 (retro-008 cut at #241, closed 2026-10-09 19:47:56Z, rank 148). Closed swarm issues at the pull: 163. Difference 15 = interval 15. Trigger met.

Scope (RETRO.md rule: the first `interval` closures after the stored cut, by `closedAt`): ranks 149 to 163, 2026-10-09 19:48:36Z (#367) to 19:58:35Z (#31). The brief's "about 19:47Z to 20:03Z" is a rough label; the rule gives the range above. Data was read at about 20:05Z, so the BA ruling (20:04Z) and the state of PRs 376 and 377 are at that pull.

The 15 issues, in closedAt order: 367, 248, 360, 365, 9, 27, 126, 149, 184, 209, 210, 226, 228, 71, 31.

Types: tech-debt 9 (367, 248, 9, 126, 184, 209, 210, 228, 71), test 3 (365, 27, 31), bug 2 (360, 226), infra 1 (149).

Cut for the next retro: `last_retro_closed_count` = 163 (the count at the cut point). Next trigger: closed count 178. Rank 163 is #31, so there were no closures after the cut at the pull.

Not in scope: #241 and earlier (retro-008); #375 (retro-008 apply issue, open, status:ready); #370, #371, #249, #232, #42, #176, #284, #263, #350, #351 (open at the pull, reviewed here only where they show a pattern).

### Supervisor observations (a) to (h), checked against evidence

| # | Observation | Verdict |
|---|---|---|
| a | PR 369 and PR 366 merged in the period; dev applied a one-character fix | Partly. PR 369 merged 19:48:29Z (squash to `e25f0cf`), inside the scope. PR 366 (retro-007 doc) merged 19:40:13Z, before the scope. The fix is commit `13d41aa469`, one line (1 insertion, 1 deletion), a semicolon in the PROTOCOL label state machine |
| b | UAT closed 11 of 13 remaining `status:uat` items under #371; reopened #42 and #176 | Confirmed. PASS comments on #9, #27, #126, #149, #184, #209, #210, #226, #228, #71, #31 (19:51:31Z to 19:58:34Z). #42 PARTIAL FAIL 19:52:39Z, #176 FAIL 19:53:54Z, both reopened to `role:dev status:ready`. #371 itself has no comment and is still open with `status:ready role:uat` |
| c | #42 and #176 touch `PATCH /users/me` and disagree on 400 vs 422; both held; BA asked | Confirmed. PR 376 (#176, 20:00 to 20:01) and PR 377 (#42, 20:02) opened after the reopens; neither merged at the cut. BA ruled at 20:04Z (comment on both issues, docs rule merged as PR 378 at 20:04:19Z). The ruling is in Decisions 1 |
| d | PR 373 squash failed with "Base branch was modified", succeeded on retry with the same pinned SHA | Confirmed. #249 comment at 19:59: first attempt hit "Base branch was modified", retry with `b3b926c6` succeeded, merged 19:59:21Z |
| e | `status:uat` queue fell from 17 to 2 | Confirmed. 17 at 19:47:56Z is the retro-008 figure. Open `status:uat` at the pull: 2 (#249, #370) |
| f | Stale `status:blocked` on #1, #32, #229 cleared after the .env blocker was resolved in #357 | Confirmed, with a lag. Blocked 14:07:29Z to 14:07:36Z by the Supervisor (UAT: no `.env` in the uat worktree). #357 closed 19:03:35Z. Cleared 19:56:26Z to 19:56:33Z, 53 min after the blocker closed. The retro-007 rule that clears such labels went live at 19:48:29Z, so the labels were cleared 8 min after the rule existed |
| g | Supervisor's merge-ok on PR 374 was made directly, as it was docs-only | Not verifiable on GitHub: every merge in the window shows the owner account. The protocol has no docs-only exception, so a direct merge is a deviation. Decisions 2 |
| h | #324 held through the period while the cap was above 5, dispatchable at 2 | Confirmed, with a correction. #324 (`type:feature`, `status:ready` since 17:36:22Z) has no claim at the pull. Open `status:uat` fell to 5 at about 19:54Z (estimated from the scope events), and fell to 2 by the pull. Both devs claimed reopened UAT fixes at 19:59Z (#42, #176, #249). #324 had no claim at the pull (about 20:05Z) |

Retro-007 decisions 1 to 8 are live from 19:48:29Z (`e25f0cf`). Visible effects in the window: close only after merge (all 15 scope issues), full-SHA merge pin (PR 373 retry), oldest-first UAT order (UAT FAIL/PASS batch), blocked-label cleanup (#1, #32, #229 at 19:56Z). The triage issue #365 was filed by the Supervisor at 19:23Z, before the rule.

## Metrics

Method: cycle = `createdAt` to `closedAt`. Status spans from label events (`gh api repos/tvolodi/BilimBaga/issues/N/events`), 15 scope issues. Medians and p90 nearest rank. Times in minutes unless marked.

| Metric | Value |
|---|---|
| Cycle time, all 15 | median 591.9 (9 h 52 min), p90 857.7 (14 h 18 min); min 10.7 (#367), max 862.0 (#9) |
| Cycle time, tech-debt (9) | median 591.9, p90 862.0 |
| Cycle time, test (3) | median 854.5, p90 857.7 |
| Cycle time, bug (2) | median 299.1, p90 554.1 |
| Cycle time, infra (1) | 725.4 |
| Created to first `status:in-progress` (14) | median 264.8 (4 h 25 min), max 541.6 (#9 waited from 05:29Z to 14:31Z). Queue wait, not dev time |
| `status:ready` total per issue (15) | median 291.4, max 546.5 |
| `status:in-progress` span (14) | median 3.6, max 105.3 (#31) |
| `status:review` span (14) | median 2.3; tail 191.1 (#226, PR 235 opened 10:58Z, merged 14:07Z) and 186.0 (#31, PR 238 opened 11:01Z, merged 14:07Z) |
| `status:uat` span (13) | median 295.9 (4 h 56 min), max 356.8 (#226). Largest single share of cycle time |
| Reopen rate, label-based (`status:uat` to `status:ready`), scope | 1 of 15 (6.7%): #226 (PARTIAL 10:56Z, round 2 in PR 235) |
| Reopen rate, `reopen: N` tag | 0 of 1. The #226 send-back has no tag. `uat.md` asks for one. retro-008 Decision 6 moves the count to label transitions |
| Reopen events in the window, open issues (after the scope) | 3: #249 (FAIL 19:48Z, fixed PR 373, back in `status:uat` 19:59Z), #42 (PARTIAL FAIL 19:52:39Z), #176 (FAIL 19:53:54Z). None had the tag |
| Failed results, scope | UAT PARTIAL 1 (#226). Dev failed result 0. BA 0. Infra 0 |
| Reassignments (stall) | 0 |
| Parks / releases, scope | 1: #27 released to ready 08:23Z (dev2, "p1 bugs take precedence"), not a stall |
| attempts >= 2, scope | 1: #226 (round 2) |
| Claims by role, scope | bb-dev1 5 (248, 149, 210, 226, 228). bb-dev2 9 (367, 360, 9, 27, 126, 184, 209, 71, 31). BA, UAT, Infra 0. Triage #365 is Supervisor-only |
| Claim collisions, scope | 2: #209 (dev2 14:54:57Z, dev1 14:56:12Z, withdrawn 14:56:42Z) and #71 (dev2 15:02:53Z, dev1 15:05:11Z, withdrawn 15:05:17Z). Resolved by the withdrawal rule in 0.5 to 2 min |
| Claim gap, all dev claims | No claim comment between 10:57Z (#233) and 14:06Z (#240). 3 h 9 min. Ready p2 items were waiting in this gap (#9, #27, #71, #126, #184) |
| Supervisor silence on GitHub | No Supervisor comment between 11:02Z (#31 incident note) and 14:06Z (#233 reconcile). The next actions were the 14:06 to 14:07Z batch: merge-ok on PR 235 and PR 238, `status:blocked` on #1, #32, #229 |
| Idle time per role | Not derivable. Ticks are not recorded. The claim gap above is the proxy |
| Escalations in the window | Not derivable. `swarm/state/escalations.json` has 17 entries (1 incident, 8 needs-user, 8 recommendation), all dated `2026-10-09` with no time. The only incident (lighthouse, #31) is at 11:02Z, before the window |
| `status:blocked`, scope issues | 0 of 15 |
| `status:blocked`, swarm-wide, cleared in the window | #1, #32, #229: 14:07:29Z to about 19:56:30Z, about 5 h 49 min each |
| `status:blocked`, open at the pull (about 20:05Z) | #284 since 15:30:31Z (4 h 35 min). #263 since 16:34:33Z (3 h 31 min). #350, #351 since 18:44Z (1 h 21 min). #232 since 19:49:27Z (16 min) |
| `status:uat` open queue | 43 at 19:20Z (retro-008 peak). 17 at 19:47:56Z. 2 at the pull (#249, #370). Cap 5 |
| Duplicate or invalid closes | 0 in scope. No `not planned` or duplicate close |
| Closed before its PR merged | 0 of 14 that have a PR. #365 has no PR (triage) |
| Closed with a status label still set | 1: #365, closed 19:50:06Z with `status:ready role:uat` still set. Still set at the pull |
| PR conflicts and holds | "Base branch was modified": 1 (PR 373, retry with the same SHA passed). `DIRTY`: PR 377 (#42) at the pull. Held pending BA ruling: PR 376 (#176) and PR 377 (#42), both resolved by BA at 20:04Z (PR 378) |
| Architect routing | 0 architect tasks in the window (no `architect-review` or `architect-decision` comments). No T1 to T8 trigger fired |
| Lock contention | None recorded in GitHub. `stack.lock` and `migration.lock` holds are not in GitHub: not derivable |
| Memory holds | None recorded in comments |
| Cross-window note | #287 closed 19:47:33Z on three named cases while six watchdog cases still failed; #370 filed 19:47:39Z, fixed in PR 372 (merged 19:59Z), still open at the pull with `status:uat` |

## Went well

- **Retro-007 rules are live.** PR 369 merged at 19:48:29Z. The Supervisor's diff check caught a semicolon error before merge (`13d41aa469`, one line). The close-after-merge rule held: all 15 scope issues had `status:done` before close, and all 14 with a PR were closed after the PR merged.
- **UAT batch throughput.** Eleven scope issues got a PASS between 19:51:31Z and 19:58:34Z. Each comment names the main SHA (`e25f0cf`) and the live evidence. The `status:uat` queue fell from 17 (19:47:56Z) to 2 at the pull.
- **Fast UAT-to-fix round trips.** #249: FAIL 19:48Z, claimed 19:59Z, PR 373 merged 19:59:21Z (about 11 min). #42 and #176: reopened 19:52:39Z and 19:53:54Z, claimed 19:59Z, PRs 377 and 376 opened 20:01Z to 20:02Z (about 9 min).
- **Retry with the same pinned SHA worked.** PR 373 "Base branch was modified" on the first attempt, merged on retry with `b3b926c6`. The protocol rule was followed as written.
- **Contract conflict answered precisely.** BA's 20:04Z ruling is a per-case table (body shape 400 INVALID_BODY, field value 422 VALIDATION_ERROR), names the merge order, and lands the docs rule first as PR 378.
- **Claim collisions resolved fast.** Both collisions in scope were withdrawn within 0.5 to 2 min; no double work was merged.
- **No duplicate or invalid closes** in scope.
- **Retro trigger on time.** The count difference reached 15 at 19:58:35Z (count 163). The cut is exact.

## Went badly

- **UAT wait is most of the cycle time.** `status:uat` median 295.9 min, 13 spans. Root causes, in order:
  1. The uat worktree had no `.env`. UAT reported (#229, 14:07Z) that its settings deny reading, copying and creating `.env`, so UAT could not provision it. Live runs were blocked from 14:07Z until Infra added `.env` during #357 (closed 19:03:35Z). The blocker lasted about 5 h. The owner-level fix is Open recommendation 1.
  2. While blocked, UAT kept items at `status:uat` with stack-free checks ("Kept status:uat for live re-verify", #176 and #31 at 14:22Z). Those items waited for the live run.
  3. No UAT live run is visible between 19:03:35Z and 19:49Z (46 min). Cause not derivable.
  4. The oldest-first order and the triage issue are retro-007 rules that went live at 19:48:29Z. The queue had reached 43 at 19:20Z. The Supervisor filed #365 at 19:23Z on its own judgment, before the rule.
- **Supervisor silent on GitHub for 3 h 4 min.** No Supervisor comment between 11:02Z and 14:06Z. In that window no dev claim was posted (10:57Z to 14:06Z), and two PRs sat in `status:review` for 186 and 191 min until merge-ok in the 14:07Z batch. Root cause not derivable: ticks are not recorded (Decision 6 makes the next gap visible). A tick report line per tick would show whether this was a missed tick or an idle one.
- **p2 backlog starved.** 9 of 15 scope items were p2 tech-debt. Created 05:29Z to 10:48Z, claimed median 4 h 25 min after creation. Dispatch is prio then age, and both devs were on p1 and feature work until the 14:06Z tick. Capacity question: Open recommendation 3.
- **Stale blocked labels.** #1, #32, #229 stayed blocked 5 h 49 min. The blocker closed 19:03:35Z. The retro-007 cleanup rule went live at 19:48:29Z and cleared them at 19:56Z. The rule did not exist for the first 45 min after the blocker closed, and no tick cleared them earlier.
- **Closed with a status label set.** #365 (Supervisor's own triage close, 19:50:06Z) still has `status:ready role:uat`. The close rule says this is a defect for the next reconcile, and the reconcile did not clear it by the pull. Decision 4 puts the label step into the close itself.
- **Reopen comments lack the tag.** 0 of 4 send-backs in the window (#226, #249, #42, #176) use `reopen: N`, although `uat.md` asks for it. retro-008 Decision 6 already changes the count to label transitions, so no new decision here. The tag is still wanted for humans reading the thread.
- **API status codes split across two PRs.** #42 and #176 each set status codes for `PATCH /users/me` (and #176 for the reminder endpoint) without a rule for body-shape errors. `api-conventions.md` section 6 had no body-shape vs value rule until PR 378 (20:04Z). Both UAT FAILs (19:52Z, 19:53Z) came from that gap, not from a dev error. PR 377 is `DIRTY` at the pull.
- **#228 split into two PRs with status bounces.** PR 264 (items 2 and 3) merged 14:50Z, PR 267 (item 1) merged 14:55Z. The issue went review, in-progress, review, in-progress, review between 14:48Z and 14:55Z. Work per item was right, but the issue has no way to track two PRs.

## Decisions

Each change has the exact before and after text on `origin/main` at `c46795e` (swarm files unchanged since `4cacad3`). Each names the metric it should move. These are role and protocol changes only. Owner-reserved items are under "Open recommendations for the user". No line below overlaps the retro-008 changes in #375 (`supervisor.md` steps 1 and 4, `dev1.md` and `dev2.md` step 3, PROTOCOL test-only exception and Roles line, RETRO.md steps 1 and 7). #375 and this apply issue touch different lines and can be applied in either order.

1. **`swarm/PROTOCOL.md`, Merge bullet: two PRs that disagree on a status or error code.** Metric: PR conflicts and holds (2 held, 1 DIRTY); target one BA question per contract conflict and no dev rework from a later ruling.
   - Before: `Delete the remote branch only after \`gh pr view <n> --json state\` shows MERGED.`
   - After: `Delete the remote branch only after \`gh pr view <n> --json state\` shows MERGED. Two open PRs that give different status or error codes for the same endpoint: the Supervisor sends \`merge-ok\` to neither, files one \`question\` with \`"kind":"design"\` to BA naming both PRs and both issues, and merges in the order of BA's answer; the later PR rebases onto the first. A docs rule that settles the code goes first as its own PR.`

2. **`swarm/PROTOCOL.md`, Merge bullet: docs-only PRs the Supervisor authored.** Metric: merge path for retro and swarm-doc PRs (PR 374 merged with no dev step; the protocol had no path for it). Evidence is the Supervisor's report, not GitHub (one instance).
   - Before: same sentence as Decision 1, `Delete the remote branch only after \`gh pr view <n> --json state\` shows MERGED.`
   - After: same as Decision 1 with this sentence after it: `A docs-only PR that the Supervisor authored (retro, swarm docs) is merged by the Supervisor after CI is green, with \`gh pr merge <n> --squash --match-head-commit <full SHA>\`; the dev merge step does not apply.`
   - Note: Decision 1 and Decision 2 change the same bullet. Apply both in one edit.

3. **`swarm/roles/supervisor.md`, tick step 2: review wait visible.** Metric: review wait tail (191 min #226, 186 min #31). Target: every PR in `status:review` without `merge-ok` is named with its age in the next tick report.
   - Before: `\`pr-open\`: check mergeability and file overlap with other open PRs; lower-risk first; send \`merge-ok\` when clean.`
   - After: `\`pr-open\`: check mergeability and file overlap with other open PRs; lower-risk first; send \`merge-ok\` when clean. Each tick report names every PR still in \`status:review\` with no \`merge-ok\`, with its age.`

4. **`swarm/PROTOCOL.md`, label state machine: report-only closes clear labels in the close.** Metric: closes with a status label set (1 in scope, #365). Target 0.
   - Before: `A close with \`status:ready\`, \`status:in-progress\`, \`status:review\` or \`status:uat\` set is a defect for the next tick's reconcile.`
   - After: `A close with \`status:ready\`, \`status:in-progress\`, \`status:review\` or \`status:uat\` set is a defect for the next tick's reconcile. A report-only close (triage or sweep, no PR) runs the label step first: remove each \`role:*\` and \`status:*\` label that is set, add \`status:done\`, then \`gh issue close <n>\`.`

5. **`swarm/PROTOCOL.md`, Escalation section: timestamps.** Metric: escalations in a window (not derivable: 17 date-only entries). Target: derivable in the next retro.
   - Before: `The user is informed through reports only.`
   - After: `The user is informed through reports only. Each \`swarm/state/escalations.json\` entry has \`utc\` as a full ISO-8601 time (for example \`2026-10-09T19:20:00Z\`), \`kind\` and \`detail\`.`

6. **`swarm/roles/supervisor.md`, tick step 7: tick record.** Metric: tick gaps and idle time per role (not derivable this period; the 10:57Z to 14:06Z silence was visible only from GitHub). Target: a tick log that a retro can read.
   - Before: `7. Update \`workers.json\` and \`retro.json.last_tick_utc\`; end with a 5-line report (dispatched, merged, filed, escalated, idle roles); schedule the next tick.`
   - After: `7. Update \`workers.json\` and \`retro.json.last_tick_utc\`; end with a 5-line report (dispatched, merged, filed, escalated, idle roles) and append that report as one line, prefixed by the UTC time, to \`swarm/state/ticks.log\`; schedule the next tick.`

Carried, no change here: retro-007 decisions 1 to 8 (live since 19:48:29Z, see Period). retro-008 decisions 1 to 7 are pending in #375.

Supervisor actions for the next tick (not role changes): none outstanding from this scope. #249 is back in `status:uat` for re-verification after PR 373. #370 is open with `status:uat` after PR 372 merged; UAT has to verify it on main.

## Applied changes (PR links)

- pending retro-apply issue #<n>

## Open recommendations for the user

1. **UAT `.env` access (owner decision).** The uat worktree had no `.env` from 14:07Z to 19:03Z, about 5 h. The `uat.settings.json` deny rule blocks reading, copying and creating `.env`, so UAT could not provision it. Options: (a) allow UAT to copy `.env` from the main checkout (settings edit, owner-only, the swarm cannot make it); (b) have Infra provision `.env` into `.claude/worktrees/uat` when the worktree is created. This is the largest single cause of the UAT wait.
2. **Supervisor silence 11:02Z to 14:06Z (owner check).** GitHub shows no Supervisor activity for 3 h 4 min and no dev claim for 3 h 9 min while p2 work was ready. Check whether the Supervisor loop was stopped. If it stops, the Supervisor needs a restart path (the #145 watchdog covers the workers only). Decision 6 gives the next retro the tick data.
3. **p2 priority (owner decision).** Nine of 15 scope items were p2 tech-debt that waited a median 4 h 25 min for a claim. Decide whether p2 tech-debt goes ahead of new features in the anti-idle engine, or stays behind them.
4. **Owner-blocked items still open.** #232 (AI key and rate-limit setting for the cost guard, blocked since 19:49Z), #263 (ANTHROPIC key, holds the migration lock while blocked), #284 (snapshot branch privacy), #350 (no scheduled DB backup for bilimbaga-test), #351 (hetzner-prod disk at 85%). As in retro-008; no new owner items from this scope.
5. **retro-008 apply (#375).** Still open at `status:ready`. Apply it, or apply it together with this retro's changes in one PR, before the next retro reads the rules.
