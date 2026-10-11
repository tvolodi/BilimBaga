# Retro 011

Status: architect draft adopted by the Supervisor (RETRO.md steps 3-4). Decisions 1-7 are applied through the apply issue linked below; 8 and 9 are filed as issues.


Author: bb-architect, trigger T2. Scope: ranks 179 to 237 by `closedAt` (59 issues), #406 (2026-10-10 02:50Z) to #510 (2026-10-11 04:23Z). Data pulled 2026-10-11 04:24Z to 04:40Z from GitHub (issues, comments, label events, merged PRs with files) and `swarm/state/`. Rule text checked against `origin/main` at `29cbdd7`.

The Supervisor sent no metrics table this time, so the table below is mine. Idle time per role is not derivable (see Went badly 2).

## Metrics

| Metric | Value |
|---|---|
| Closures in scope | 59 (interval is 15; the retro ran 44 closures late) |
| Cycle time, all | median 137 min, p90 1101 min |
| Cycle time, issues created inside the scope (48) | median 93 min, p90 320 min |
| By type (n, median, p90 in min) | bug 24, 59, 259; tech-debt 21, 258, 1136; infra 8, 711, 1286; feature 3, 35, 35; test 3, 137, 137 |
| Merged PRs | 60, none merged with a failed check |
| PR open to merge | T4 PRs (27): median 10 min; other PRs (33): median 7 min |
| T4 PRs merged without an architect decision | 1 of 27 (PR 424); retro-010 had 4 of 4 |
| Architect decisions | 54 comments on 27 issues; 16 were `changes`; 11 issues got at least one `changes` |
| Reopens (`reopen:` comments) | 0 |
| Follow-up issues | 23 of 59 titles are follow-ups; 11 came from architect comments |
| Closed with two status labels | 0 of 59 (retro-010: 2 of 15) |
| Closed with a `role:*` label left | 14 of 59 |
| Open now with two status labels | 2 (#461, #511: `ready` and `in-progress`) |
| Activity | 2026-10-10 02:22Z to 09:26Z, then 18.7 h with two comments, then 2026-10-11 04:09Z onward |
| Claims | bb-dev1 19, bb-dev2 15, bb-ba 3 |
| Lines in `swarm/state/ticks.log` | 1 (`reload retro-010`) |

## Went well

- **Close hygiene is fixed.** All 59 closures carry one status label, `status:done`. Retro-010 decision 1 (`set-status.sh`) moved its metric to zero.
- **The T4 gate works.** 26 of 27 protected or oversize PRs waited for a decision, and the gate cost about three minutes at the median. It caught real defects before merge: a spoofable `True-Client-IP` (PR 476), dark mode switched on before pages were ready (PR 454), unreadable status colours in dark (PR 467), a guard bypass with `host:port@` (PR 497), a test that would fail on a fixed date (PR 456).
- **No reopen and no red merge.** Zero `reopen:` comments, zero merged PRs with a failed check. The Docker Hub 429 rule was not needed in this scope.
- **Dev throughput.** 60 PRs in about seven active hours. Follow-ups from review were small and merged in 3 to 14 minutes each.
- **UAT evidence on main.** Every close names the main SHA and the observed result. The standing gate for nginx files (`scripts/proxy-chain/run.sh`, two PASS lines) was run on each head, including both heads of PR 482.
- **Workers refused relayed owner approvals.** bb-infra (#351) and bb-uat (#32) declined to touch the demo host or download tools on a relayed "owner says go". That is `CLAUDE.md` Owner authority 3 applied correctly.
- **The x.ai change took 20 minutes once unblocked.** Design approval 04:03Z, PR 509 at 04:09Z, merged 04:17Z, live-key UAT with four checks done 04:23Z.

## Went badly

1. **The retro did not trigger.** The count passed 15 at about 04:37Z on 2026-10-10 and the 4 h gap ended at 07:03Z. The Supervisor ticked until 09:26Z and ran no retro; the count reached 59. Root cause: the check is a prose step (`supervisor.md` step 6) that depends on the Supervisor's memory, like the four rules retro-010 found.
2. **No tick log, stale state files.** `ticks.log` has one line. `workers.json` was last written at 02:59:49Z on 2026-10-10. Retro-010 decision 2 gave the Supervisor `tick-log.sh`; it was not called once in seven hours of ticks. The 18.7 h gap may be an owner pause or a usage limit; the files cannot say, for the second retro in a row.
3. **Relayed owner orders cost about 22 h on four items.** #32 was refused at 06:02Z and ran at 04:18Z the next day. #351, #350 and #419 were refused at 05:51Z; PR 482 was approved twice and deployed at 04:17Z the next day. Each started only when the owner spoke in the session of the role that does the work. The refusals were right; the loss is the time nobody told the owner where to say it.
4. **The architect held #232 for the wrong thing.** I made dev work wait for the owner to confirm the x.ai order in my session. The code is inert without a key, and only the owner can supply one, so building it sent no data anywhere. The hold cost 22 h for a 20 min change. Root cause: role gap. The architect treated a design decision as if it needed the owner's order.
5. **A secret was copied.** UAT loaded the owner's x.ai key through "an uppercase `XAI_API_KEY` copy in UAT scratch, outside the repo" (#263, 04:21Z; reused on #232). No value was printed, but the decision on #232 said no agent copies the value, and the owner's instruction was that the value is never copied. The app reads the lowercase name, so the copy was not needed. Root cause: the rule lived in one issue comment, not in `_common.md`, and UAT's stack has no documented way to take the main checkout's `.env`.
6. **Security fixes were designed inside review, one defect at a time.** Password epoch: #439, #455, #457, #466 (four issues, three `changes`). Client address: #475, #478, #481, #484, #480 (five issues, three `changes`). Several of those rounds were my misses: the first fix list for PR 453 left out the token issue time and the same-second case; my design for #475 chose a recursive `X-Forwarded-For` walk that a private client could exploit (#481). Root cause: no design step before dispatch for security work; the invariant was found by iterating on a diff.
7. **Approval and merge-ok SHAs drift.** The protocol tells the dev to merge `origin/main` into the branch before the squash, which changes the head. PR 482 needed a second architect decision for a head that differed only by that merge (7 h between the two). The Supervisor also sent me one mistyped SHA (PR 460) and two wrong branch names (PRs 497, 501). Root cause: tooling. SHAs are copied by hand, and nothing says when an approval survives a main merge.
8. **Architect comments ignore the 15-line cap.** Most decisions in this scope run 20 to 30 lines. The cap is not followed because conditions, close rule and evidence do not fit in it. A rule nobody can follow should be changed.
9. **Review requests outside the queue.** bb-infra sent a T4 request for PR 513 straight to the architect. It was handled, but the Supervisor learned of it only from my result.
10. **Label leftovers.** 14 closed issues keep a `role:*` label, which `set-status.sh --close` should remove. #461 and #511 are open with `ready` and `in-progress` together, both set by bb-infra. Either the script is not used by every role or `--close` has a path that skips the role step.
11. **x.ai endpoint and latency.** The design chose `/chat/completions`, which x.ai's current reference no longer lists (it works today). The first live insights call took 58.6 s against a 60 s client timeout. Later cold calls took about 9 s. One slow answer from the provider turns the feature into a 503.

## Root causes, grouped

| Cause | Items |
|---|---|
| Tooling: mechanical steps kept as prose | 1, 2, 7, 10 |
| Protocol gap | 3 (no path for an owner order to reach the executing role), 9 |
| Role-prompt gap | 4, 5, 8 |
| Architecture | 6 (no design step for security work), 11 |
| Environment | 2 (pause or limit, not derivable) |

Pattern since retro-007: a rule that became a script moved its metric (two status labels: 2 to 0; T4 without decision: 4 of 4 to 1 of 27). A rule that stayed a sentence for the Supervisor did not (tick log, retro check, state files). Decisions 1 and 2 follow that.

## Decisions

1. **One script ends the tick and says when a retro is due.** Metric: retro lateness (44 closures) to 0; tick log lines per tick (0) to 1.
   - New file `swarm/bin/tick-end.sh "<report>"`: appends the report to `ticks.log` (as `tick-log.sh` does), writes `retro.json.last_tick_utc`, counts closed swarm issues live, and prints `RETRO DUE (<diff> closures, last cut <UTC>)` when the difference is at least `interval` and the last cut is 4 h old. Test in `swarm/tests/` with a stubbed `gh`: due, not due by count, not due by gap.
   - `swarm/roles/supervisor.md` step 7. Before: `Update \`workers.json\` and \`retro.json.last_tick_utc\`; end with a 5-line report (dispatched, merged, filed, escalated, idle roles) and append that report with \`swarm/bin/tick-log.sh "<report>"\`; schedule the next tick.` After: `Update \`workers.json\`; end with a 5-line report (dispatched, merged, filed, escalated, idle roles) and run \`swarm/bin/tick-end.sh "<report>"\`; if it prints \`RETRO DUE\`, the retro starts in this tick, before any dispatch; schedule the next tick.`
   - Proposed issue, carried from retro-010 decision 2 and still not done: `ensure-up.ps1` treats a last `ticks.log` line older than 30 min as a stalled Supervisor.

2. **An approval survives a merge of main, checked by a script.** Metric: second decisions for an unchanged change (1, 7 h) to 0; wrong SHAs or branch names in review requests (3) to 0.
   - New file `swarm/bin/same-change.sh <approved-sha> <new-sha>`: prints `SAME` when `git diff $(git merge-base origin/main X) X | git patch-id` is equal for both, otherwise `DIFFERENT`. New file `swarm/bin/review-request.sh <pr>`: prints PR number, branch, full head SHA, changed paths and check state from `gh pr view`, for pasting into the task message.
   - `swarm/PROTOCOL.md`, Waiting. Before: `A T4 PR gets no \`merge-ok\` until the issue carries an \`architect-decision:\` comment other than \`changes\`.` After: the same sentence, then: ` The decision names a head SHA. It stays valid for a later head when \`swarm/bin/same-change.sh <approved> <new>\` prints \`SAME\` (only \`origin/main\` was merged in); the Supervisor quotes that line in the merge-ok. Any other new head goes back to the architect. Review requests are built with \`swarm/bin/review-request.sh\`, never typed.`

3. **An owner order goes to the role that executes it.** Metric: hours lost between a refused relay and the start of work (about 22 h on four items) to under one tick.
   - `swarm/roles/supervisor.md` step 4b, append: ` The Supervisor does not relay an owner approval for an action that \`CLAUDE.md\` reserves to the owner (demo or any remote host, internet downloads, secrets). It comments \`owner-order-needed: say it in the <role> session\` on the issue, sets \`status:blocked\`, and lists the issue under "waiting for owner" in every tick report until the role reports the order received.`
   - No change to `CLAUDE.md` Owner authority 3. See Open recommendation 1.

4. **Secrets rule in `_common.md`.** Metric: copies of a secret made by an agent (1) to 0.
   - `swarm/roles/_common.md`, new numbered rule at the end: `Secrets. An agent never reads, prints, copies or moves the value of a key, token or password: no \`cat .env\`, no \`docker compose config\`, no \`env\` dump of a container, no second file that holds the value, under any name. A stack that needs the owner's \`.env\` is started with \`--env-file <path to the main checkout's .env>\`. That a secret is loaded is shown by behaviour (a 200, a startup line), never by its value.`
   - Supervisor action now: bb-uat deletes the scratch copy and confirms on #263. See Open recommendation 2.

5. **Security work gets its design before dispatch.** Metric: `changes` decisions and follow-up issues per security fix (password epoch: 3 and 3; client address: 3 and 4) to at most 1 and 1.
   - `swarm/roles/supervisor.md` step 4b, append: ` A \`type:bug prio:p1\` issue whose title starts with \`Security\`, and any issue that will change a T4 auth or client-address path, goes to the architect under T1 before dispatch. The architect answers with the invariant in one sentence, the cases the tests must cover, and the files in scope; the dev implements that list.`
   - `swarm/roles/architect.md`, Tasks step 1, append: ` For a security change, write the invariant and the abuse cases (who controls each input, what happens at the boundary values) before reading the diff, and check the diff against that list.`

6. **The architect holds only what has an effect.** Metric: hours a design waited for an owner confirmation that the code did not need (22 h) to 0.
   - `swarm/roles/architect.md`, Tasks step 2. Before: `or \`escalate-owner\` (reserved user decisions).` After: `or \`escalate-owner\` (reserved user decisions). Escalate or hold only the step that has the effect: a deploy, a paid or data-sending call, a key on a host. Code that does nothing until the owner configures it is approved on its merits, with the owner's part named as a condition of the later step.`

7. **Decision comments: the decision in the first lines, evidence below.** Metric: decisions over the cap (most of 54) to 0, with the reader still able to stop early.
   - `swarm/roles/architect.md`, Tasks step 3. Before: `at most 15 lines: decision, reasons, risks, what to do next.` After: `the decision, the head SHA, the conditions before merge and the close rule in the first 8 lines; reasons and what was checked below, as short as the case allows.`
   - Same file, Rules. Before: `One decision per message;` After: `One decision per message; a review request comes from the Supervisor, and a request sent by another role is answered and reported to the Supervisor in the result;`

8. **Label leftovers (proposed issue, not a rule).** Metric: closed issues with a `role:*` label (14) and open issues with two status labels (2) to 0. File `type:tech-debt role:dev`: find why `set-status.sh --close` left `role:*` on #510, #505, #477, #419, #407, #406, #402, #394, #263, #232, #193, #176, #157, #145 (script path or hand edits), fix it, and add the case to the script's test. Supervisor action now: clean those 14 and fix #461 and #511 with the script.

9. **AI provider follow-up (proposed issue, not a rule).** File `type:tech-debt role:dev prio:p1`: the insights call came within 1.4 s of the 60 s client timeout. Measure five cold calls; then either raise the client timeout with a matching nginx `proxy_read_timeout`, or set a faster model in `AI_MODEL`. Add a client for `/v1/responses` only if `/chat/completions` starts to fail. This PR handles a secret and comes to the architect.

Carried, no change: retro-010 decisions 1, 3, 4, 5, 6, 7 show no counter-evidence in this scope. Retro-010 decision 2 is replaced by decision 1 above.

Supervisor actions for the next tick (not rule changes):
- Have bb-uat delete the scratch key copy and confirm (decision 4).
- Clean the label leftovers (decision 8).
- Rewrite `workers.json` from live state.
- #473 is at `status:uat` since PR 508 merged at 09:20Z on 2026-10-10; it closes only after a redeploy, so mark it `status:blocked` with the reason instead of leaving it in the UAT queue.

## Way of working and model fit

- **Shape.** The team's shape is right. The Supervisor is still the single point of failure, now for its own bookkeeping more than for dispatch.
- **Where time goes.** Active work is fast: 93 min median for an issue born in the scope, 7 to 10 min from PR to merge. The long tails are all waiting for the owner (infra median 711 min) or for the swarm to be running at all. More agents would not move either number.
- **UAT queue.** Not a bottleneck in this scope. Sampled issues spent 23 to 51 min in `status:uat`. Review follow-ups waited about 5 h in `status:ready` behind dark mode and the security fixes, which was the right priority order.
- **Wrong role for a task: one.** The architect acted as an owner-approval gate on #232 (decision 6).
- **Duplicated work.** #505 and #507 are the same defect filed twice, two minutes apart; closed without harm.
- **Architecture drift: one area, now closed.** Client address had three readers (rate limiter, audit, request log) with three rules; it is now one helper with a source guard test. Password revocation is one function with a guard test. Both arrived by iteration, not by design (decision 5).
- **Tech debt trend.** Better. Guard tests were added for each class of defect found (token storage, `apiFetch`, client address, password stamp source, design palette, nginx files). The debt that grew is in swarm bookkeeping, not in the product.
- **Follow-up share.** 23 of 59 closures are follow-ups, 11 from architect comments. Each was small and closed the same day. The architect misjudged one as optional (#472 was required). From now each follow-up in a decision says whether it blocks the close.
- **Progress against "ready".** Product closures in scope: FR-BB319, FR-BB320 Parts A to C, FR-BB321 dark mode, five security fixes, the AI provider. Open and critical: none. What stands between main and the customer is a demo redeploy, which is the owner's release decision, and FR-BB65 performance evidence (#32 partial, #511 at Lighthouse 66 against a gate of 90).
- **Model fit, workers.** Good for specified work; the fix lists were implemented exactly. The weak spot is unchanged: a dev covers the named case, not the class. Decision 5 gives the class in advance.
- **Model fit, Supervisor.** Judgment is sound (priority order, holding T4 PRs, refusing to merge on a pending check). Mechanical compliance failed again on the three steps that are still prose. Decision 1 removes two of them. If the tick log is still empty after it, restart the Supervisor after each retro-apply; a model change is the owner's call.
- **Model fit, architect.** 54 decisions in about nine active hours, one task at a time as PROTOCOL Routing requires; the queue never held more than two requests. Five of my decisions needed a correction by a later issue (listed in Went badly 6 and in the follow-up item above). The reviews that worked ran the thing in a throwaway container; the ones that missed reasoned from the diff alone.

## Open recommendations for the user

1. **Where to give an order (owner habit, no rule change).** An order for the demo host, a download or a secret works only when you type it in the session of the role that will do it (bb-infra, bb-uat). Told to the Supervisor, it is refused by the worker and waits. With decision 3 the Supervisor will tell you which session to use.
2. **The x.ai key was copied once (owner decision).** bb-uat made a second file with the key, outside the repo, to start its stack. It will be deleted. Decide whether you want to rotate the key; nothing shows it was printed or left the machine.
3. **Demo release (owner decision).** Main is 60 PRs ahead of the demo: dark mode, the token-storage fix, the default tenant colour, client-address and rate-limit changes, healthchecks, the AI provider (off until a key is on the host). The edge nginx part is already live. The rest needs one redeploy outside an exam session.
4. **AI on other hosts (owner decision).** QA and the demo have no key, so AI answers 503 there. With a key, exam results, authors' source text and employees' short-text answers go to x.ai, and grading has no daily cap. The live grading path is untested.
5. **Swarm uptime (owner, information).** Say whether the stop from 09:26Z on 2026-10-10 to 04:09Z on 2026-10-11 was your pause or a usage limit. The state files cannot tell, for the second retro running.
6. **Still owner-blocked:** #396 (host hygiene), #428 (visual baselines, needs a fixed Playwright container), #32 (performance run, partial), #473 and #461/#511 (reach hosts only with a redeploy). Docker Hub rate limit on CI and branch-protection required checks are unchanged from retro-010.

## Applied changes (PR links)

See the retro-011 apply issue (decisions 1-7), issues for decisions 8 and 9; links added when the PRs merge.
