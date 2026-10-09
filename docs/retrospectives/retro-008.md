# Retro 008

## Period

Trigger: `retro.json.last_retro_closed_count` = 133 (retro-007 cut at #354, 19:26:39Z). Live closed count at the data pull (2026-10-09 ~19:54Z): 149. Difference 16 >= interval 15. The trigger was crossed at #241 (19:47:56Z, rank 148). The last recorded tick (19:43:25Z, count rank 144, difference 11) was correctly below it.

Scope (RETRO.md rule, applied as the first `interval` closures after the stored cut, by `closedAt`): ranks 134 to 148, #355 (19:32:38Z) to #241 (19:47:56Z). The 15 issues: 355, 299, 319, 330, 348, 317, 253, 313, 326, 338, 305, 288, 268, 287, 241.

Cut: `last_retro_closed_count` = 148. Rank 149, #367 (retro-007 apply issue, closed 19:48:36Z), is outside this scope and rolls into the next retro. Next trigger: closed count 163.

Not in scope: the 11-item #365 Part 1 batch (ranks 119 to 133, 19:25:17Z to 19:26:39Z) and its retro-007 counterparts were reviewed in retro-007. The 9 closures at ranks 110 to 118 are also not reviewed (retro-007 left them out; see Went badly). The brief's "19:20Z to 19:50Z" is a rough label; the rule gives 19:32:38Z to 19:47:56Z.

Data sources: `gh issue list --label swarm --state closed` (149), `gh issue list --state all` (180 swarm issues), `gh api repos/tvolodi/BilimBaga/issues/N/events` for all 180, issue comments, `gh pr view` for PRs 252, 368, 369, 341. `swarm/state/escalations.json` read from the main checkout. Note: the `actor` of every label and close event is the owner account (`tvolodi`), so GitHub does not identify which role made a change; roles are read from comment text.

## Metrics

Method: cycle = createdAt to closedAt. Status spans = label events (`status:*` labeled to unlabeled). Medians and p90 (nearest rank) over the 15 scope issues.

| Metric | Value |
|---|---|
| Scope closures | 15: type:tech-debt 8 (355, 299, 319, 330, 348, 317, 268, 287), type:bug 7 (253, 313, 326, 338, 305, 288, 241). All 15 closed with `status:done` set |
| Cycle time, all 15 | median 147.9 min, p90 310.1, min 34.3, max 341.1 |
| Cycle time, tech-debt (8) | median 124.2 min, p90 293.4 |
| Cycle time, bug (7) | median 177.1 min, p90 341.1 |
| Created to first `status:in-progress` | median 0.6 min, max 27.5 (queue wait for a worker to claim) |
| `status:ready` span (13 spans) | median 4.5 min; max 76.7 min (#253, dependency, see Went badly) |
| `status:in-progress` span (17 spans) | median 3.1 min, max 15.1 |
| `status:review` span (16 spans) | median 4.3 min, max 10.8 |
| `status:uat` span (16 spans, #288 has two) | median 130.7 min, max 304.9. Total 2,225 of 2,566 cycle minutes (87%) |
| Last `status:uat` to closed | median 119.7 min, range 2.5 to 304.9 |
| Test-only items (355, 299, 319, 330, 348, 317; PRs 359, 302, 325, 333, 353, 318) | `status:uat` span median 100.9 min, range 24.2 to 164.4 |
| Reopen rate, label-based (`status:uat` to `status:ready`) | 1 of 15 (6.7%): #288 |
| Reopen rate, `reopen: N` tag | 0 of 15. The #288 send-back comment says "reopened for dev" with no tag |
| Failed results | 1 UAT PARTIAL FAIL (#288). 0 dev failures, 0 BA, 0 infra in scope |
| Reassignments (stall) | 0. One dependency release by dev1 (#253, 14:48:55Z) |
| attempts >= 2 | 1 issue: #288 (PR 289, then PR 368 after the reopen) |
| Claims by role | bb-dev1 5 (299, 317, 253, 288, 241); bb-dev2 10 (355, 319, 330, 348, 313, 326, 338, 305, 268, 287); BA, infra, UAT 0 |
| Idle time per role | Not derivable (no tick history). Proxy only: the create-to-claim median above |
| Escalations in window | Not derivable. `escalations.json` has 17 entries, all `"utc": "2026-10-09"` with no time. `swarm/state/` is local and not in git |
| `status:blocked`, scope issues | 0 of 15 |
| `status:blocked`, swarm-wide, open at pull | 8 issues. #1, #32, #229 since 14:07Z (about 5 h 46 min); #284 since 15:30Z (4 h 23 min); #263 since 16:34Z (3 h 19 min); #350, #351 since 18:44Z (1 h 9 min); #232 since 19:49Z (5 min) |
| `status:uat` open queue (from label events, open issues only) | 43 at 19:20Z (peak; 21 at 16:00Z, 33 at 18:00Z), 32 at 19:26:39Z, 17 at 19:47:56Z. Live count at pull: 13 (gh). Cap 5 |
| Oldest open `status:uat` items | 13 items, `status:uat` since 14:07:27Z to 16:29:38Z. They are the 13 not in the #365 triage list; none was verified in this period |
| Duplicate or invalid closes | 0 in scope (no `not planned` or duplicate close) |
| Closed before PR merged | 0 of 15 (each PR merged before its issue closed; #288: PR 368 at 19:43:31Z, close 19:46:09Z) |
| Closed without a PR | 0 in scope |
| PR conflicts | None recorded in scope or in the window (no conflict, base-moved or rejected comments) |
| Merge rejections | 0 in window |
| Lock contention | None recorded. `stack.lock` holds are not in GitHub, so not derivable |
| Memory holds | 0 recorded |
| Architect routing (T1 to T8) | 0 architect tasks in the window (no `architect-review` or `needs-architect` comments) |

## Went well

- **Closed only after merge, with `status:done`.** All 15 closed with `status:done` set, none before its PR merged. This was done by hand before the retro-006 and retro-007 close rules were live, so it is not yet a rule result.
- **Fast dev loop.** Created to claimed median 0.6 min; in-progress to `status:uat` median 9.8 min across the scope (4.8 to 229.6; the 229.6 is #253, which was released once).
- **Live UAT evidence on main.** UAT's Part 2 checks of #313, #326, #338, #305, #241, #268 and #287 each cite the main commit they ran on and the exact observed result. Those comments are reviewable without a rerun.
- **Fast #288 round trip.** UAT PARTIAL FAIL 19:35:09Z, reopened 19:35:11Z, claimed 19:37:31Z (Supervisor task), PR 368 opened 19:39:09Z, merged 19:43:31Z (8 min 22 s after the reopen), UAT PASS on adc229e 19:46:06Z, closed 19:46:09Z.
- **Retro trigger on time.** Crossed at 19:47:56Z. The 19:43:25Z tick was correctly below it. No missed trigger.
- **Retro-007 apply PR 369 landed.** Two commits. The second, `13d41aa469` (semicolon in the PROTOCOL label state machine, 27 s after the first), fixed the one defect found before the merge at 19:48:29Z (`e25f0cf`). The applied text matches retro-007 decisions 1 to 8 (checked against `git diff 6f39892 e25f0cf -- swarm/`).
- **Test-only items were checked, not assumed.** Each test-only close comment names the PR, the merge commit, and the checks it relied on.

## Went badly

- **UAT wait is most of the cycle time.** 2,225 of 2,566 cycle minutes (87%) were `status:uat` time. The UAT queue was above cap (5) for the whole period: 43 at its 19:20Z peak. Root cause: one UAT session verifies everything, and six of the 15 scope items were test-only with no behaviour change (median wait 100.9 min). Retro-007 decision 3 (test-only fast path) went live at 19:48:29Z, after these closes.
- **Test-only closes on cancelled or flaky CI, decided case by case.** 5 of the 6 test-only items (299, 319, 330, 348, 317) had a cancelled `post-merge-smoke` on the merge commit; #330 also had a docker.io metadata fetch error first. Four were closed on a "supervisor decision" with tests, docker-build and security green (299, 319, 330, 348). #317 was closed after a local `go test -p 2 ./internal/tags/...` run. #355 closed after a green re-run. Root cause: the fast-path rule says "CI green" and does not say what to do when a check is cancelled and cannot be re-run (the per-ref concurrency group cancels superseded runs; see the owner recommendation).
- **#253 waited for an unmerged dependency.** Created 14:24:46Z. Claimed by dev1 14:48:41Z; released 14:48:55Z because the endpoint existed only in open PR #252. Re-routed (handoff note at 15:27:16Z), merged #252 at 15:57:16Z, re-claimed by dev1 at 16:05:34Z. Ready wait about 101 min; cycle 310 min. Root cause: dispatch does not check whether the issue depends on a PR that is not yet on main.
- **#288 reopened after the first fix.** PR 289 covered the unknown but well-formed question id. UAT found that a malformed UUID still returned 500 (reopened 19:35:11Z). Root cause: the dev test covered the named input and not the input class. The UAT send-back also had no `reopen: N` tag (rule in `uat.md`), so tag-based reopen counts (Architect T7) cannot see it.
- **Stale `status:blocked`.** #1, #32, #229 have been labelled since 14:07Z, about 5 h 46 min at the pull. Retro-007 decision 8 (clear blocked labels whose blocker is gone) went live at 19:48:29Z; the labels were still set at the pull. The Supervisor should clear them in its next tick (action, not a role change).
- **The triage issue and the queue did not match.** #365 (filed 19:23:27Z) said "30 items". The live queue was 43 at 19:20Z. Its 30 were 17 test-only plus 13 behaviour items. The 13 oldest items (uat since 14:07Z to 16:29Z) were not in it and are still open. Retro-007 decision 4 (list every open item) went live at 19:48:29Z, after #365 was filed.
- **#370: six failing swarm test cases since #341.** Filed by UAT at 19:47:39Z (not the Supervisor), claimed by dev1 at 19:49:43Z as a Supervisor task, PR 372 opened 19:51:12Z. Its cause (PR 372 comment): the test's all-roles stub predates `bb-architect` (#341, merged 18:22:25Z). Root cause: a role was added without updating the swarm test fixtures.
- **Escalations are not timestamped.** All 17 entries carry a date only, so an in-window count is not possible. Any "escalations since" check needs a time.
- **Retro scope rule left closures unreviewed.** Retro-007 took the last 15 closures before its cut and left 9 earlier ones out (ranks 110 to 118). The rule in `RETRO.md` did not say what happens to closures outside the interval.
- **Owner-blocked and environment-blocked items.** #263 (ANTHROPIC key; holds the migration lock while blocked), #284 (snapshot branch privacy), #350 (no scheduled DB backup for bilimbaga-test), #351 (hetzner-prod disk 85%). #232 was blocked by UAT at 19:49:27Z (not verifiable on the local stack).
- **Post-merge-smoke cancellation again.** Same cause as retro-007. It held 5 of the 6 test-only closes in scope.

### Supervisor observations (a) to (j), checked against evidence

| # | Observation | Verdict |
|---|---|---|
| a | Retro-007 rule changes (PR 369) applied by a dev, one-character fix requested | Confirmed. PR 369 (`feb992ef47`, `13d41aa469`), merged 19:48:29Z as `e25f0cf`. The fix is the semicolon in the PROTOCOL label state machine |
| b | UAT #365 closed 11 test-only items, held 6; accepted tests-green for 4, re-checked 1 by local go test | Confirmed for the six (299, 319, 330, 348, 317, 355). The four accepted on tests-green are 299, 319, 330, 348; #317 is the local `go test`. #355 also closed, after its smoke re-run went green. The 11 closed items are in retro-007's scope, not verified here |
| c | Queue went from 30 to 17 | Partly. 17 at 19:47:56Z is confirmed. The start was 43 at 19:20Z (30 was the #365 count, which was wrong). Live count at pull: 13 |
| d | UAT re-verified #288 on adc229e and it closed | Confirmed. PASS comment 19:46:06Z on adc229e; closed 19:46:09Z |
| e | UAT reopened #288 after a partial fix; a dev fixed it in PR 368 within about 20 minutes | Confirmed, shorter. Reopen 19:35:11Z; PR 368 merged 19:43:31Z, 8 min 22 s |
| f | Closed-count trigger missed for about 10 minutes at 19:23Z after a batch | Not supported as stated. The batch was 19:25:17Z to 19:26:39Z (retro-007 scope). The trigger crossed at 19:25:27Z; the next tick was 19:28:27Z, so about 3 min. This period: no miss (see Went well). Retro-007 decision 1 (check every tick) is applied on origin/main |
| g | The dev's apply PR needed a one-character fix (semicolon) before merge | Confirmed (see a) |
| h | PR 368 merge-ok pinned to a full SHA with `--match-head-commit`; squash succeeded first time | Partly. First-attempt success confirmed (single merge at 19:43:31Z, no retry recorded). The dev's comment gives the full SHA `aa821706e1d6405aa1bd54e2703f02fdd71d5df6`, equal to the PR head. The flag itself is not in GitHub. The full-SHA rule landed in `e25f0cf` at 19:48:29Z, after this PR |
| i | New dev issue #370 filed outside the Supervisor's dispatch; decide whether to keep it ready | Filed by UAT (body: "Found by UAT (#365 / #287 re-check)"), not the Supervisor. Already claimed at 19:49:43Z as a Supervisor task and in review (PR 372). Keeping it was the right call; no action needed. The cause is in Went badly |
| j | Event-driven worker rules (PR 341) live since 18:22Z; workers idle between Supervisor tasks; Supervisor filed sweeps | Partly. PR 341 merged 18:22:25Z (confirmed). Supervisor-assigned sweeps confirmed (#355, #348, and #347 and #354 in retro-007). Idle time per role not derivable. Claim latency median 0.6 min |

## Decisions

Each change has the exact before and after text on origin/main at e25f0cf. Each names the metric it should move. These are role, protocol and procedure changes only. Owner-reserved items are under "Open recommendations for the user".

1. **`swarm/roles/supervisor.md`, tick step 4: dependency gate.** Metric: wait of dependent issues (#253 about 101 min ready with an open dependency); target: no dispatch of an issue whose dependency is open.
   - Before: `4. Dispatch to idle workers: best ready issue by prio then age; check \`claimed-by\` and checkpoints; send \`task\` with \`notify_when_idle: true\`; set \`role:*\` (feature without doc -> ba; bug -> dev; test -> uat; deploy -> infra); backend-heavy to Dev1, frontend-heavy to Dev2.`
   - After: `4. Dispatch to idle workers: best ready issue by prio then age; check \`claimed-by\` and checkpoints; send \`task\` with \`notify_when_idle: true\`; set \`role:*\` (feature without doc -> ba; bug -> dev; test -> uat; deploy -> infra); backend-heavy to Dev1, frontend-heavy to Dev2. An issue whose body or comments say \`needs #N\` is not dispatched while #N is open or not on main; the Supervisor waits for #N to merge, then dispatches.`

2. **`swarm/roles/dev1.md` and `swarm/roles/dev2.md`, step 3 (same text in both).** Metric: reopen rate (1 of 15, #288) to 0; attempts >= 2 (1) to 0.
   - Before: `3. Implement; migrations via the migration lock; prefer unit tests over running the stack.`
   - After: `3. Implement; migrations via the migration lock; prefer unit tests over running the stack. For a bug where bad input gives a 500 or a wrong status, the handler test covers malformed, unknown and empty values of that input, not only the value named in the issue.`

3. **`swarm/PROTOCOL.md`, label state machine, test-only exception.** Metric: judgment calls on non-green test-only closes (5 of 6 in scope) to 0; test-only `status:uat` span median 100.9 min to under 15 min (the fast path works).
   - Before: `exception: a PR that changes only \`*_test.go\` or \`*.test.tsx\` files, with CI green on its merge commit, goes review -> done.`
   - After: `exception: a PR that changes only \`*_test.go\` or \`*.test.tsx\` files, with every required check green on its merge commit, goes review -> done. A check that is cancelled or flaked is re-run once; if it still does not finish, the Supervisor runs the same test command locally on the merge commit and puts the command and result in the close comment. Such a close is not a supervisor decision for an unexplained result.`

4. **`swarm/PROTOCOL.md`, Roles line: role changes update test fixtures.** Metric: failing swarm test cases after a role change (6 after #341) to 0.
   - Before: `Roles: \`bb-supervisor\`, \`bb-architect\`, \`bb-dev1\`, \`bb-dev2\`, \`bb-ba\`, \`bb-uat\`, \`bb-infra\`. Worktrees: \`.claude/worktrees/<role>\`; Infra lives in \`..\ai-dala-infra\`.`
   - After: `Roles: \`bb-supervisor\`, \`bb-architect\`, \`bb-dev1\`, \`bb-dev2\`, \`bb-ba\`, \`bb-uat\`, \`bb-infra\`. A PR that adds or renames a role also updates the role stubs in \`swarm/tests/\` in the same PR. Worktrees: \`.claude/worktrees/<role>\`; Infra lives in \`..\ai-dala-infra\`.`

5. **`swarm/PROTOCOL.md`, Escalation section: timestamps.** Metric: in-window escalations, not derivable (17 date-only entries) to derivable.
   - Before: `## Escalation` followed by `Max 3 attempts per issue, then \`result: failed\`.` (the rest of the section is unchanged)
   - After: the same text, with this sentence appended: ` Each entry in \`swarm/state/escalations.json\` has \`utc\` as a full ISO-8601 time (for example \`2026-10-09T19:20:00Z\`), plus \`kind\` and \`detail\`.`

6. **`swarm/roles/supervisor.md`, tick step 1: reopen count.** Metric: reopen rate derivable from labels (1 of 15, tag count 0 of 15) to derivable for every reopen; T7 (reopened twice) can fire.
   - Before (end of step 1): `...open issue at \`status:done\`: close it in the same tick, \`status:blocked\` whose blocker is gone: clear the label and comment).`
   - After: the same text, with this sentence appended after the closing parenthesis: ` The reopen count N is the number of \`status:uat -> status:ready\` transitions in the label events; the \`reopen: N\` tag is not required.`

7. **`swarm/RETRO.md`, step 1 and step 7: cut rule.** Metric: closures left unreviewed per retro (9 in retro-007) to 0.
   - Before (step 1): `take issues closed since the last retro (the last \`interval\` ones).`
   - After: `take the first \`interval\` closures after the last retro by \`closedAt\` (ranks \`last_retro_closed_count\`+1 to \`last_retro_closed_count\`+\`interval\`). Period names the rank range and the first and last issue.`
   - Before (step 7): `(\`last_retro_closed_count\` = current closed count, \`retro_number\` + 1)`
   - After: `(\`last_retro_closed_count\` = the cut from step 1, that is last + interval, not the live count; \`retro_number\` + 1). Closures after the cut are the next retro's scope.`

Carried, no new change (retro-007 decisions 1 to 8 are live from 19:48:29Z; check them in retro-009): decision 1 (every-tick retro check), decision 2 (close only at `status:done` after MERGED), decision 3 (test-only fast path, amended above), decision 4 (full triage list), decision 5 (oldest-first UAT), decision 6 (full-SHA merge pin), decision 7 (dev merge flags), decision 8 (stale blocked, and the 2-ready rule for held features).

Supervisor actions for the next tick (not role changes): clear `status:blocked` on #1, #32, #229 (blocker gone since about 16:07Z per retro-007); give the 13 oldest `status:uat` items an explicit UAT order; #249 (reopened by UAT at 19:48:42Z) and #232 (blocked 19:49:27Z) are for retro-009.

## Applied changes (PR links)
- pending retro-apply issue #<n>

## Open recommendations for the user

- **Post-merge-smoke runs cancel each other.** The per-ref concurrency group cancels a superseded run, and a cancelled run cannot be retried. It held 5 of the 6 test-only closes in this scope and 3 in retro-007. Choose whether superseded post-merge runs cancel each other or queue. The workflow file is yours to change.
- **Owner-blocked items.** #263 (ANTHROPIC key; holds `migration.lock` while blocked), #284 (should the snapshot branch be private; the repo is public), #350 (no scheduled DB backup for bilimbaga-test), #351 (hetzner-prod root disk at 85%; reclaiming docker build cache needs your approval). All still open at the pull.
- **bilimbaga-test admin password.** Escalations in `escalations.json` (#152) say the default admin credential may still be set on the customer demo. Checking or rotating it is your call; the swarm does not log in there.
- **Local worktrees.** `git worktree list` still shows about 15 worktrees, several for merged branches. Cleanup is local and can be done by the Supervisor after confirming the branches are merged.
- **Idle time is not derivable.** If you want an idle metric in future retros, the Supervisor would need to keep a tick log (not a role change; say if you want it).
