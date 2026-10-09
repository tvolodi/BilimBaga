# Retro 007

## Period

Scope (RETRO.md rule, last `interval` = 15 closed swarm issues by `closedAt`): #349 (18:48:18Z) to #354 (19:26:39Z). The 15 issues: 349, 357, 358, 346, 303, 306, 311, 315, 327, 331, 334, 335, 343, 347, 354.

Counter: `retro.json.last_retro_closed_count` = 109 (retro-006 cut at #61, 16:14:02Z). The closed count at this retro's data cut is 133, so 24 closures are since the last retro. The 9 earlier ones are outside the scope and are not reviewed here: #240, #290, #7, #296, #200, #309, #183, #322, #342 (16:22Z to 18:35Z).

Trigger: the difference reached 15 at 19:25:27Z (#306 was closure 124, inside the 11-issue UAT batch #303 to #354, 19:25:17Z to 19:26:39Z). The last tick before it was 19:23:31Z (difference 13). The next recorded tick is 19:28:27Z. Data snapshot: 19:28Z. Closures after the snapshot (#299, #319, #355 at 19:32Z to 19:33Z) are outside this retro; the closed count is now 137.

## Metrics

Sources: `gh issue list --label swarm --state closed`, `gh api repos/tvolodi/BilimBaga/issues/N/events` for all 177 swarm issues (label events), issue comments. Status durations come from label events. "Not derivable" means GitHub holds no data for it.

| Metric | Value |
|---|---|
| Closed in scope | 15: tech-debt 12, infra 2, test 1 |
| Cycle time created->closed, all 15 | median 80 min, p90 156 min (nearest rank), max 167 min |
| Cycle time by type | tech-debt (12): median 81.5, p90 156. infra (2): 1 and 5 min. test (1): 36 min |
| Queue wait created->in-progress | tech-debt 0 to 5 min (filed and claimed together). #349 and #357 never in progress (closed from `status:ready`). #346 claimed by BA 32 min after filing |
| Dev work in-progress->status:uat (12 tech-debt, incl. review and merge) | median 6.5 min, range 4 to 11 |
| UAT wait status:uat->done (11 test-only items closed in one batch) | median 88 min, range 18 to 156 |
| status:uat open queue (open issues, from label events) | 21 at 16:00Z, 23 at 16:30Z, 26 at 17:00Z, 33 at 18:00Z, 40 at 19:00Z, 43 at 19:20Z (peak), 32 at 19:30Z. Cap 5: above cap the whole window. Not comparable with retro-006's "31 at 16:00Z" (its method is not recorded) |
| Reopen rate (`reopen:` comments) | 0 of 15 (0%). No `status:uat -> status:ready` send-back in the window. Two UAT FAIL or PARTIAL results (#342, #346) went to BA, not dev, and carry no `reopen:` tag |
| Failed results / reassignments | 0 failed, 0 reassigned in scope |
| attempts >= 2 | 2, both bb-dev1: #306 (squash rejected on a short SHA, retry) and #343 (rejected because the base moved, retry with the same SHA). Infra #290 needed one extra push after its backend-tests job failed on staticcheck |
| Claims by role (scope) | bb-dev1 10, bb-dev2 2, bb-ba 1, bb-infra 2 (no `claimed-by` on infra items; work is in comments) |
| Idle proxy, bb-dev1 (gap from its merge to its next claim, n=9) | median 8 min, max 21 min (17:19Z to 17:40Z, 18:22Z to 18:43Z). Proxy only: a dev may have had no ready work or no notice |
| Idle proxy, bb-dev2 | 74 min from its 16:49Z merge to its next scope claim at 18:03Z. Proxy only (two claims in scope) |
| Idle time per role, other roles | Not derivable. No tick history is persisted; `workers.json` is a snapshot (19:28:27Z) |
| Supervisor-filed issues in window | 11 carry the "Filed by bb-supervisor" marker (#346, #347, #348, #349, #354, #355, #357, #309, #322, #342, #365). Issue bodies of the dev tech-debt items do not say who filed them, so their authorship is not derivable |
| Escalations (`escalations.json`, 2026-10-09) | 2 in window: 16:17:30Z (Go toolchain missing; owner action, cleared 16:20Z) and 16:34:53Z (#263 needs an ANTHROPIC key, still open). The 14:10Z entry is before the window |
| status:blocked, scope issues | 0 of 15 |
| status:blocked, swarm-wide | #290 13.5 min (16:06:45Z to 16:20:17Z, owner unblock). #263 blocked 16:17Z to 16:20Z, then again from 16:34:33Z (still open, about 3 h). #350 and #351 from 18:44Z (owner approval, open). #284 from 15:30Z (owner decision, open). #1, #32, #229 from 14:07Z: still labelled at 19:28Z (about 5 h 20 min) although their blocker was gone by about 16:07Z |
| Closed without `status:done` | Scope: 4 of 15 (#349, #357, #346 at `status:ready`; #358 at `status:uat`). All 24 since retro-006: 9 (adds #290 at `status:review`, #7 with no status, #309, #322, #342 at `status:ready`) |
| Closed before the PR merged | 1 of 24: #290 closed 16:29:34Z, PR #294 merged 16:29:43Z |
| Duplicate or invalid issues | 0 in scope. No `duplicate` close in the window. #7 (umbrella) closed 16:34Z after its four children were done |
| PR conflicts | None recorded in scope. Two merge rejections: short SHA (#306), base moved (#343) |
| Branch deleted before merge confirmed | 1: #306 cleanup deleted the remote branch and closed PR #307. Restored at the same SHA and reopened (16:59Z) |
| Lock contention | No recorded wait in scope. `migration.lock` was held by #263 while it was blocked on a key (escalation text; not re-verified). `stack.lock` handed to infra at 17:08Z after UAT released it |
| Memory holds mentioned | 0 in scope |

Post-scope status at 19:33Z (not in the metrics): #299, #319, #355 closed by UAT; #317, #330, #348 still held (the post-merge-smoke run was cancelled, see Went badly).

## Went well

- **Retro-006 decisions landed.** Apply PR #297 merged 16:35:07Z. The text is on origin/main: label state machine (uat->done only after merge), claim ack, dev1/dev2 feature hold, supervisor tick 1 (open `status:done`), search before filing, memory hold line, `_common.md` rule 14. The retro-005 gap noted in retro-006 (memory line missing) is closed.
- **No feature self-claims and no double dispatch in the window.** #324 (FR-BB18 AC-13, ready since 17:36Z) stayed unclaimed as the feature hold intends. Zero `claimed-by` conflicts.
- **Fast Supervisor-task pick-up.** #347 claimed 17 s after filing, #354 claimed 11 s after filing.
- **Fast dev loop on tech-debt.** In-progress to `status:uat` median 6.5 min over 12 items.
- **Scenario-versus-design routes closed in minutes.** #342: UAT FAIL at 18:32Z, BA decision at 18:33Z, PR #345 merged 18:33Z, UAT PASS at 18:35Z. #346: UAT PARTIAL, BA decisions PRs #363 and #364 (merged 19:17Z), UAT PASS at 19:18Z. #309: BA correction PR #310 at 16:58Z.
- **Test-only triage worked.** #365 Part 1 closed 11 items in 2 minutes (19:25:14Z to 19:26:39Z). Each was checked by the merged PR file list (only `*_test.go` or `*.test.tsx`) and CI on the merge commit. This is a sound verification and is the basis of decision 3.
- **Infra and Go changes.** #290 (Go 1.27) was unblocked by the owner in the infra session at 16:20Z and merged at 16:29Z. The stack was rebuilt by 17:09Z. #357 (`PUBLIC_APP_URL` on the local stack) took 1 minute, and #358 (`.env.example`) 10 minutes.
- **Zero reopen-tagged send-backs and zero failed results in scope.**

## Went badly

- **UAT queue above cap for the whole window.** 21 open at 16:00Z, 43 at 19:20Z, cap 5. Feature work (#324, #322 requirement) waited for the whole period. Root cause: a protocol gap plus an environment block. UAT was blocked from 14:07Z to about 16:07Z (escalation 14:10Z: no `.env` in the worktree). Thirteen items were labelled `status:uat` between 14:07Z and 16:29Z and were still waiting at 19:20Z (oldest 14:07Z).
- **The triage issue undercounted the queue.** #365 says "the status:uat queue is 30 items". At 19:20Z there were 43. The 13 oldest items (#9, #27, #31, #42, #71, #126, #149, #176, #184, #209, #210, #226, #228) are not in the triage list.
- **UAT ran new scenario issues ahead of the queue.** #342 (18:17Z to 18:35Z) and #346 (18:43Z to 19:18Z) were BA-ordered scenario runs. They were worked while older queue items waited. UAT was not idle: it did those runs. The UAT role has no rule on the order of queue work when the cap is exceeded.
- **Test-only work went through UAT.** 12 of the 43 queued items (about 28%) were test-only tech-debt. Their median UAT wait was 88 min for changes that cannot change behaviour. Protocol gap: no fast path for test-only PRs.
- **Closed without `status:done`.** 4 of 15 in scope (9 of 24). The Supervisor closed #349, #357, #346, #342 with `status:ready` and #358 with `status:uat`. Protocol gap: the state machine names the uat->done step but no close step for BA, infra and UAT-test items.
- **Stale `status:blocked`.** #1, #32, #229 are still labelled about 5 h 20 min after the blocker was gone. Protocol gap: no rule that whoever clears a blocker removes the label.
- **Dev idle gaps.** bb-dev1 waited 21 min twice (17:19Z to 17:40Z, 18:22Z to 18:43Z) with no task. Both fall in the cap period, when the only ready dev item was probably the held feature #324. The cause is inferred; the Supervisor's ready list at those times is not recorded. The anti-idle rule "keep at least 2 ready per role" counts a held feature.
- **Merge tooling.** A short SHA was rejected (#306). A remote-branch cleanup closed PR #307 before the merge was confirmed. A base-moved rejection (#343) needed a retry. #334's merge-ok was pinned to a short SHA (2b21896). Protocol gap: `merge-ok` has no full-SHA form, and the dev's cleanup runs before the merge is verified.
- **CI cancelled UAT verifications.** `post-merge-smoke` runs are cancelled by the per-ref concurrency group (cancel-in-progress). A cancelled run cannot be retried. At 19:32Z to 19:33Z, #317, #330 and #348 were still held for this reason; #299, #319 and #355 were closed later. Environment or tooling gap. The workflow setting is the owner's call (see recommendations).
- **Retro trigger.** The closed count crossed the threshold at 19:25:27Z, in an 11-issue batch. The tick at 19:23:31Z was correctly below it. The next recorded tick was 19:28:27Z. The gap is about 3 min, not the 10 min in the brief. Role-prompt gap: tick step 6 is not explicit that the difference is recomputed on every tick with the live count.
- **Owner-blocked items piling up.** #263 (owner key) still holds `migration.lock` while blocked. #284, #350 and #351 wait for owner decisions. Blocked items are reports only and the swarm cannot clear them.

### Supervisor observations (a) to (j), checked against evidence

| # | Observation | Verdict |
|---|---|---|
| a | Event-driven workers went idle; Supervisor filled the idle work | Partly confirmed. PR #341 merged 18:22:25Z (not 18:17Z). Supervisor-filed default work confirmed (#347, #354 "Supervisor task"; #349, #357 infra checks). bb-dev1 idle gaps 0 to 21 min. Reload-role messages: not derivable |
| b | Queue at 30 against cap 5; #324 held; UAT idle until #365 at 19:23Z | Queue and hold confirmed (21 to 43, #324 held). "UAT idle" not supported: UAT ran #342 and #346. The queue count in #365 (30) was wrong (43 open at 19:20Z) |
| c | 11 test-only closed on green main; 6 held on cancelled or flaked checks | Confirmed. The 11 are #303, #306, #311, #315, #327, #331, #334, #335, #343, #347, #354. The 6 held were #299, #317, #319, #330, #348, #355; later 3 closed, 3 still held |
| d | UAT FAILs routed to BA; BA fixed scenarios in #345, #363, #364 | Confirmed. Also PR #310 (#309, dept-admin S1 step 11) |
| e | Supervisor missed the trigger about 10 min after the batch took the count from 122 to 133 | Batch confirmed (19:25:17Z to 19:26:39Z, 122 to 133). Delay about 3 min to the next recorded tick. Why the tick did not compare the difference is not verifiable from GitHub |
| f | Relayed Architect order (T1-T8, PR #339) held until the owner confirmed | Not verifiable from GitHub. PR #339 merged 18:11:45Z; PR #341 applied the triggers 18:22:25Z. The hold and confirmation happened in sessions |
| g | Go 1.27 system install blocked until the owner confirmed; Go toolchain vanished and was reinstalled | Confirmed. #290 blocked 16:06:45Z, unblocked 16:20:19Z on the owner's instruction in the infra session; infra ran the winget install (not the owner). Escalation 16:17:30Z says Go was missing. Note: #290 closed at review, 9 s before its merge |
| h | PR 301 requested for merge by an outside session; merged on owner instruction | Merge confirmed (16:49:37Z). The request and instruction are not in GitHub |
| i | Squash failures on short SHAs; a branch delete closed PR 307, restored at the same SHA | Confirmed (#306 correction at 16:59Z; #343 base-moved retry; #334 short pin) |
| j | Merge-ok pins moved to full SHAs after a dev squashed the wrong head twice | Not verifiable. No comment in the window records a wrong-head squash. Short pins are confirmed (#334) |

## Decisions

Each change has exact before and after text on origin/main. Each names the metric it should move. These are role and protocol changes only. Reserved items are under "Open recommendations for the user".

1. **`swarm/roles/supervisor.md`, tick step 6, retro trigger every tick.** Metric: retro trigger latency, 1 tick late this period -> 0.
   - Before: `6. Retro check: closed swarm issues minus \`retro.json.last_retro_closed_count\` >= 15 -> run retro per \`swarm/RETRO.md\`.`
   - After: `6. Retro check, every tick: count closed swarm issues live (\`gh issue list --label swarm --state closed --limit 1000 --json number\`), subtract \`retro.json.last_retro_closed_count\`; if the difference is >= \`retro.json.interval\`, run the retro per \`swarm/RETRO.md\` in this tick. Compare the difference only, never the absolute count and never a stored next-trigger number.`

2. **`swarm/PROTOCOL.md`, label state machine: close rule.** Metric: closed without `status:done` 4 of 15 -> 0; closed before merge 1 -> 0.
   - Before (end of line 13): `... Every status change is one \`gh issue edit\` that also removes the previous \`status:*\` label.`
   - After (same line, appended): ` An issue is closed only with \`status:done\` set and only after \`gh pr view <n> --json state\` shows MERGED for its PR (issues without a PR: after the check named in the issue). The Supervisor closes BA, infra and UAT-test items through the same step. A close with \`status:ready\`, \`status:in-progress\`, \`status:review\` or \`status:uat\` set is a defect for the next tick's reconcile.`

3. **`swarm/PROTOCOL.md`, label state machine: test-only fast path.** Metric: test-only items entering `status:uat` 12 -> 0; their UAT wait median 88 min -> about 0.
   - Before (fragment of line 13): `review -> uat (merged, role flips to \`role:uat\`);`
   - After: `review -> uat (merged, role flips to \`role:uat\`); exception: a PR that changes only \`*_test.go\` or \`*.test.tsx\` files, with CI green on its merge commit, goes review -> done. The Supervisor checks \`gh pr view <n> --json files\` and the CI state, closes the issue with a comment that names the PR and merge commit, and no UAT step runs.`
   - Basis: #365 Part 1 applied this check to 11 items with no regression.

4. **`swarm/roles/supervisor.md`, tick step 6, UAT cap triage.** Metric: triage coverage 30 of 43 -> 43 of 43; queue peak 43 -> 10 or less.
   - Before (second sentence of step 6): `UAT queue cap: more than 5 \`status:uat\` open -> hold feature dispatches, give devs tech-debt/test work.`
   - After: `UAT queue cap: more than 5 \`status:uat\` open -> hold feature dispatches, give devs tech-debt/test work, and in the same tick file one \`role:uat\` triage issue (if none is open) listing every open \`status:uat\` item from \`gh issue list --label status:uat --state open --json number\`, not a subset.`

5. **`swarm/roles/uat.md`, Work, queue order.** Metric: age of oldest `status:uat` item (about 5 h at 19:20Z) -> under 2 h.
   - Before: `- Verify \`role:uat status:uat\` issues with the scenario from \`docs/uat-scenarios\` or targeted Playwright.`
   - After: same line, plus: ` When more than 5 \`status:uat\` issues are open, verify them oldest first, before any scenario or sweep issue, and put the open count in each \`result\`.`

6. **`swarm/PROTOCOL.md`, Branches, merges: full SHA and branch cleanup.** Metric: merge rejections on SHA or base 2 -> 0; accidental PR closes 1 -> 0.
   - Before: `- Merge: Supervisor checks \`gh pr view --json mergeable,mergeStateStatus\` and overlap with other open PRs, then sends \`merge-ok\`. The dev merges \`origin/main\` into the branch (merge, not rebase), pushes, \`gh pr merge --squash\`.`
   - After: `- Merge: Supervisor checks \`gh pr view --json mergeable,mergeStateStatus\` and overlap with other open PRs, then sends \`merge-ok\` with the full 40-character head SHA (\`gh pr view <n> --json headRefOid\`). The dev merges \`origin/main\` into the branch (merge, not rebase), pushes, and runs \`gh pr merge <n> --squash --match-head-commit <full SHA>\`. If the merge is rejected (base moved), merge \`origin/main\` again and retry with the same SHA. Delete the remote branch only after \`gh pr view <n> --json state\` shows MERGED.`

7. **`swarm/roles/dev1.md` and `swarm/roles/dev2.md`, On a task, step 5 (same text in both).** Metric: as decision 6.
   - Before: `On \`merge-ok\`: merge origin/main into the branch, re-run tests, push, \`gh pr merge --squash --delete-branch\`, labels to \`status:uat role:uat\`, send \`result done\`.`
   - After: `On \`merge-ok\`: merge origin/main into the branch, re-run tests, push, \`gh pr merge <n> --squash --match-head-commit <merge-ok SHA>\`, delete the branch only after MERGED, labels to \`status:uat role:uat\` (or the test-only close in PROTOCOL), send \`result done\`.`

8. **`swarm/roles/supervisor.md`, tick steps 1 and 5 (stale blocked, ready minimum).** Metric: stale `status:blocked` 3 -> 0; dev merge-to-claim gap median 8 min, max 21 -> 5 min or less.
   - Step 1 before: `...open issue at \`status:done\`: close it in the same tick).`
   - Step 1 after: `...open issue at \`status:done\`: close it in the same tick, \`status:blocked\` whose blocker is gone: clear the label and comment).`
   - Step 5 before: `Keep >= 2 \`status:ready\` per role; file issues yourself if needed.`
   - Step 5 after: `Keep >= 2 \`status:ready\` per role; file issues yourself if needed. While the UAT cap holds features, a held \`type:feature\` does not count toward a dev's 2.`

## Applied changes (PR links)
- pending retro-apply issue #<n>

## Open recommendations for the user

- **CI cancels UAT verifications.** The `post-merge-smoke` job is cancelled by the per-ref concurrency group. A cancelled run cannot be retried, so UAT could not finish 6 test-only checks (#317, #330, #348 still held at 19:33Z). Choose whether superseded post-merge runs should cancel each other, or be queued. The workflow file is yours to decide.
- **Owner decisions still open.** #263 (ANTHROPIC key, holds `migration.lock` while blocked), #284 (snapshot branch priority), #350 (no scheduled DB backup for bilimbaga-test), #351 (hetzner-prod root disk at 85%; reclaiming the docker build cache needs your approval).
- **Retro counter and leftovers.** `retro.json.last_retro_closed_count` should be set to 133 (the data cut of this retro), not 137 (the current count). Closures after the cut then fall into the next retro. RETRO.md does not say what happens to closures beyond the interval (9 here were not reviewed). Choose whether leftovers roll into the next retro. RETRO.md is outside this retro's role and PROTOCOL scope.
- **Supervisor cleanup, not a role change.** Three issues are still open at `status:done` from before the period (#145, #157, #193). The tick 1 rule on origin/main covers them. Three issues are still labelled `status:blocked` after their blocker was gone (#1, #32, #229). The Supervisor can clear these now.
- **Local worktrees.** `git worktree list` still shows agent, ba, dev, uat and event-driven worktrees under `.claude/worktrees/` and two outside it. This is local cleanup for the Supervisor after their branches are merged or abandoned.
- **Test host.** `bilimbaga-test` health check (#349) passed with two warnings (#350, #351). Deploying current main and rotating the default admin password remain owner decisions (retro-006 recommendation, unchanged).
