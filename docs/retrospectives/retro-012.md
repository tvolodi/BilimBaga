# Retro 012

Status: architect draft adopted by the Supervisor (RETRO.md steps 3-4). Decisions 1-3 and 5-10 are applied through the apply issue; decision 4 is its own issue (CI change, T4 review).


Author: bb-architect, trigger T2. Scope: the 23 swarm issues closed after the retro-011 cut (2026-10-11 04:36:40Z), #517 (04:45Z) to #570 (08:41Z). Data pulled 08:41Z to 08:55Z from GitHub (issues, comments, label events, PRs with files and comments) and `swarm/state/`. Rule text checked against `origin/main` at `9e62656` or later; each Before string below occurs once in its file.

Agenda items 6 and 7 of the Supervisor (its own misses, the owner-go friction) are taken from the Supervisor's task message; I could not measure them from GitHub.

## Metrics

| Metric | Value |
|---|---|
| Closures in scope | 23 in 4 h 5 min of activity; the retro started on time (`tick-end.sh` printed `RETRO DUE` at the first tick after the 4 h gap) |
| Cycle time, all | median 19 min, p90 53 min |
| By type (n, median, p90 in min) | bug 9, 26, 51; tech-debt 7, 15, 50; feature 6, 29, 55; test 1, 14, 14 |
| PRs opened since 04:25Z | 34: 33 merged, 1 closed unmerged (PR 531) |
| PRs reviewed by the architect | 15; 9 approved at the first head, 6 got one `changes` (PRs 521, 524, 527, 529, 550, 573); none needed a third head |
| Architect decisions | 21 on PRs (15 approve, 6 changes), plus 3 designs before dispatch (#528, #534, the plan on #546) |
| PR opened to first decision | median 1 min, longest 4 min |
| Last decision to merge | median 3 min, longest 11 min |
| PR opened to merge | first-head approvals: median 3 min; PRs with a `changes`: median 10 min |
| T4 PRs merged without a decision | 1 (PR 533: 95 files, 2819 lines, size trigger) |
| Reopens | 0 |
| Closed with one status label, `done` | 22 of 23 |
| Closed with a wrong status or a `role:*` left | 1 (#570: `status:review`, `role:dev`) |
| Lines in `ticks.log` | 11 (retro-011: 1) |
| Open and waiting for the owner | #32, #396, #473, #519, #528, #560, plus the demo and QA redeploy |

## Went well

- **Retro-011 decisions held.** The tick log has a line per tick, the retro triggered by script and on time, review requests came from `review-request.sh` with a correct SHA every time (retro-011 had three wrong ones), and 22 of 23 closures have clean labels.
- **Design before dispatch worked.** Three designs were posted before code (#528, #534, #546). The largest PR of the scope (PR 558: 48 files, 2637 lines, a migration, the certificate path) was approved at its first head. Compare the two security chains of retro-011, which took four and five issues each.
- **Review is not the bottleneck.** A request waited one minute at the median. A `changes` round added about seven minutes to a PR.
- **Each `changes` was a real defect.** A whitespace-blind `same-change.sh` (PR 521); a `--role` flag missing so rule 16 could not be followed (524); a 120 s default behind Cloudflare's 100 s cut and a longer wait for grading (527); a prompt marker that a nested marker defeats (529); the partial score still in the submit response (550); an audit row lost in a race (573).
- **Devs found things the review did not ask for.** dev1 found that the 15 s `WriteTimeout` cuts every slow AI answer (PR 527). dev2 measured before building and showed that the indexes I had proposed for #563 change nothing, then chose a cache. A real-Postgres test caught an SQL bug the fake driver missed (PR 562).
- **Workers kept refusing relayed owner approvals,** and the Supervisor now lists what waits for the owner in every tick line.
- **UAT closes on evidence without paid calls** where a stub provider is enough (#545, #546), and leaves #528 open for the part that needs the live key.

## Went badly

1. **A closing keyword bypassed the close rule.** PR 573's body says `Fixes #570`. GitHub closed the issue at the merge (08:41:22Z) with `status:review` and `role:dev` still on it, before the close rule (green `backend-tests` on the merge commit) could be checked. `set-status.sh` cannot help when GitHub closes the issue itself.
2. **The size trigger of T4 was missed once.** PR 533 (95 files, untracking UAT output) merged in three minutes with no decision. The change was a mechanical deletion and harmless, but the Supervisor decides T4 by reading the file list, and `review-request.sh` does not say whether a PR is T4.
3. **The merge step contradicts itself.** `dev1.md` and `dev2.md` step 5 and PROTOCOL "Merge" tell the dev to merge `origin/main`, push, and then merge with `--match-head-commit <merge-ok SHA>`. After the push the head is a new SHA, so the pinned merge cannot succeed as written. Devs work around it; `same-change.sh` exists for exactly this and is used only on the architect's side.
4. **Workers filed default work themselves.** dev2 filed #538 "after the queue was empty"; dev1's sweep #530 produced PR 531, which was closed. `_common.md` rule 10 and the dev files already forbid this. The text is there and was not followed.
5. **Real-Postgres tests are invisible until CI.** They skip without `TEST_DATABASE_URL`, so a dev's local run is green with the test not run (PR 550's fixture bug showed only in CI; the #547 flake hit docs-only PRs). In CI a skipped test also passes; PASS lines are printed only for two name prefixes, so the exams and audit Postgres tests leave no evidence.
6. **Rule 15 names a mechanism that does not do the job.** `--env-file` only feeds `${VAR}` interpolation in the compose file; it puts nothing into a container. A role following rule 15 literally gets a stack without the key, which is how the scratch copy of retro-011 came about.
7. **Rule 9 states a model the workers are not running.** It fixes the commit trailer to "Claude Sonnet 5.5"; workers report their own model as Haiku 5.5. Either the trailer or the rule is wrong on every commit.
8. **Supervisor facts from memory (its own list).** A wrong claim owner on #563 and a stale premise about labels on #540. Both were statements about issue state made without reading the issue in that tick.
9. **Owner-go friction continues (Supervisor's list).** bb-uat's paid calls, the Lighthouse prompt and the demo and QA redeploy waited for hours; the stack lock was held for about five hours by a role that was itself waiting. Retro-011 decision 3 made the waiting visible; it did not shorten it.
10. **My own misses.** I reviewed the AI code twice (PRs 509, 512) and noted that grading "switches on by itself" without reading its call path; the provider call inside the submit transaction was found only at the third review (#528). On PR 569 I marked the uncompared `Description` as "not blocking"; it became #570, whose first fix had a race. My index proposal for #563 was wrong; the dev's measurement corrected it.
11. **Depth on one feature before anyone has seen it.** 12 of 23 closures concern AI grading or the waiting-for-grading experience. None of it is deployed. The demo is now about 95 PRs behind main.

## Root causes, grouped

| Cause | Items |
|---|---|
| Tooling: a mechanical step left to reading or memory | 1, 2, 5, 8 |
| Rule text wrong or self-contradicting | 3, 6, 7 |
| Rule present, not followed | 4 |
| Owner availability and where orders are given | 9, 11 |
| Architect review depth | 10 |

## Decisions

1. **No closing keywords in PR bodies.** Metric: issues closed by GitHub outside `set-status.sh` (1) to 0.
   - `swarm/roles/_common.md`, new rule 17: `17. Issue links. A PR body and a commit message name their issue as \`Refs #<n>\`. Never \`Closes\`, \`Fixes\` or \`Resolves\`: GitHub then closes the issue at the merge, before its close rule is checked and without \`set-status.sh\`.`
   - `swarm/bin/review-request.sh`: add one output line `closing keyword: none` or `closing keyword: <word> #<n> (remove it)`, from the PR body. Test with a stubbed `gh`.
   - Supervisor action now: `swarm/bin/set-status.sh 570 done` after quoting the green `backend-tests` of the merge commit.

2. **`review-request.sh` says whether a PR is T4.** Metric: T4 PRs merged without a decision (1) to 0.
   - `swarm/bin/review-request.sh`: add a line `T4: yes (<first matching path>)`, `T4: yes (size: <files> files, <lines> lines)` or `T4: no`, computed from the path list of PROTOCOL T4 and the 15 files or 600 lines threshold. Exceptions printed as `T4: no (swarm/tests only)`. Test: one case per path class, the size case, the exception.
   - `swarm/roles/supervisor.md` step 2. Before: `\`pr-open\`: check mergeability and file overlap with other open PRs; lower-risk first; send \`merge-ok\` when clean.` After: `\`pr-open\`: run \`swarm/bin/review-request.sh <pr>\` first; \`T4: yes\` goes to the architect with that output and gets no \`merge-ok\` before the decision. Then check mergeability and file overlap with other open PRs; lower-risk first; send \`merge-ok\` when clean.`

3. **One merge procedure that can be followed.** Metric: merges that cannot match the pinned SHA as written (every merge after a main merge) to 0.
   - `swarm/PROTOCOL.md`, Merge. Before: `The dev merges \`origin/main\` into the branch (merge, not rebase), pushes, and runs \`gh pr merge <n> --squash --match-head-commit <full SHA>\`. If the merge is rejected (base moved), merge \`origin/main\` again and retry with the same SHA.` After: `A branch that is not behind \`origin/main\` is merged with \`gh pr merge <n> --squash --match-head-commit <merge-ok SHA>\`. A branch that is behind: the dev merges \`origin/main\` into it (merge, not rebase), re-runs the tests and pushes, then runs \`swarm/bin/same-change.sh <merge-ok SHA> <new head>\`. On \`SAME\`, with the required checks green on the new head, the dev merges with \`--match-head-commit <new head>\` and quotes the \`SAME\` line in \`result done\`. On \`DIFFERENT\` the dev does not merge and sends \`result pr-open\` again.`
   - `swarm/roles/dev1.md` and `dev2.md`, step 5. Before: `On \`merge-ok\`: merge origin/main into the branch, re-run tests, push, \`gh pr merge <n> --squash --match-head-commit <merge-ok SHA>\`,` After: `On \`merge-ok\`: merge as PROTOCOL "Merge" says (pinned SHA if not behind; after a main merge \`same-change.sh\` must print \`SAME\`, then the new head is pinned),`

4. **Real-Postgres tests cannot pass by not running.** Metric: Postgres tests with no PASS line in CI (exams, audit) to 0; fixture bugs first seen in CI (1) to 0. Proposed issue, `type:test prio:p1 role:dev`, plus one rule.
   - Issue: `testpg` gets `Base(t) string`: it returns `TEST_DATABASE_URL`, skips when it is empty, and fails the test instead when `REQUIRE_TEST_DB=1`. Every real-Postgres test takes the URL from it; a source guard fails on a test file that reads `TEST_DATABASE_URL` itself. `tests.yml` sets `REQUIRE_TEST_DB=1` for `backend-tests` and replaces the prefix step with one verbose run of all packages that import `testpg`.
   - `swarm/roles/_common.md`, new rule 18: `18. Real-Postgres tests. A PR that adds or changes one runs it locally against a throwaway \`postgres:16\` container (never the UAT stack) and quotes its PASS lines in the PR body. A local run where the test skipped is not a run.`

5. **Rule 15 names the mechanism that works.** Metric: stacks started without the key by following the rule (1 known) to 0.
   - `swarm/roles/_common.md` rule 15. Before: `A stack that needs the owner's \`.env\` is started with \`--env-file <path to the main checkout's .env>\`.` After: `A stack that needs the owner's \`.env\` takes it by path: a throwaway compose override, passed with \`-f\`, that gives the service \`env_file: <absolute path to the main checkout's .env>\`. \`--env-file\` alone only fills \`\${VAR}\` in the compose file and puts nothing into the container. The override holds the path, never a value, and is not committed.`

6. **The commit trailer names the model that wrote the commit.** Metric: commits with a wrong model trailer (all worker commits, if the workers' report is right) to 0.
   - `swarm/roles/_common.md` rule 9. Before: `9. Conventional commits ending with \`Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>\`.` After: `9. Conventional commits ending with the \`Co-Authored-By:\` line your own session's harness gives you. Do not copy a model name from this file or from another role.`
   - Which model the workers should run is the owner's choice; see Open recommendation 4.

7. **Default work is filed by the Supervisor only.** Metric: self-filed default-work issues (2) to 0.
   - `swarm/roles/_common.md` rule 10. Before: `10. Nothing to do -> end your turn and wait. Default work arrives from the Supervisor as a \`task\`.` After: `10. Nothing to do -> end your turn and wait. Default work arrives from the Supervisor as a \`task\`. You file an issue yourself only for a defect or gap you met while doing a task, and its body says \`found-in: #<n>\`. An issue for coverage, a sweep or a clean-up with no \`found-in\` is closed by the Supervisor unread.`

8. **Issue state is read, not remembered.** Metric: Supervisor statements about a claim or a label that were wrong (2) to 0.
   - `swarm/roles/supervisor.md` step 2, append: ` Before a message or a comment states who holds an issue, what its labels are or whether its PR is merged, read it in this tick (\`gh issue view <n> --json labels,comments\`, \`gh pr view <n> --json state\`); never from an earlier tick.`

9. **A role that waits for the owner gives up the stack.** Metric: hours the stack lock is held by a waiting role (about 5) to 0.
   - `swarm/PROTOCOL.md`, Stack. Before: `(\`swarm/locks/stack.lock\`, UAT by default).` After: `(\`swarm/locks/stack.lock\`, UAT by default; a role that reports \`owner-order-needed\` or otherwise waits for the owner releases the lock in the same turn and takes it again when the order arrives).`

10. **The architect reads the call path of what a change switches on.** Metric: defects in reviewed code found only at a later review (1: #528) to 0.
    - `swarm/roles/architect.md`, Tasks step 1, append: ` When a change turns a feature on or makes it reachable (a key, a flag, a new route), follow every caller of the code it enables to the request or job that runs it, and state in the decision where it runs (in a request, in a transaction, in a background job).`
    - Same file, step 3, append: ` A gap you see is either in the fix list or filed as an issue in the same turn; "not blocking, when next touched" is not used for a gap in a rule the PR itself introduces.`

Carried, no change: retro-011 decisions 1 to 7 show no counter-evidence. Retro-011 decision 9 (the 58.6 s call) is #519 and is merged as PR 527; it stays open for the paid measurement.

Supervisor actions for the next tick (not rule changes):
- Fix #570's labels and close state with the script (decision 1).
- File the `testpg.Base` issue (decision 4).
- Review PR 533 after the fact is not needed: it deleted tracked output and changed `.gitignore`; I read the file list.

## Way of working and model fit

- **Shape.** Right, and faster than at retro-011: median cycle 19 min against 93 min for issues born in scope. The gain comes from design before dispatch and from scripts replacing prose.
- **Where time goes.** Nothing inside the swarm waits long. Everything that waits, waits for the owner: seven open items and two redeploys. The swarm now produces faster than it can be released or even shown.
- **Rate of `changes`.** 6 of 15 reviewed PRs (40%); retro-011 had 16 of 54 decisions (30%). The rate did not fall, but its cost did: one round, about seven minutes, no PR needed a third head. I would not try to lower the rate itself; five of the six were on security or money paths.
- **Wrong role for a task: none.** BA turned two review findings into validated requirements (FR-BB322, FR-BB73 revision 2) within 20 minutes each, before dev started.
- **Duplicated work: none merged.** PR 531 was closed unmerged; #538 was merged and useful, but unasked.
- **Architecture drift: none new.** One finalise path for manual and AI grading, one eligibility predicate shared by the job and the result page, one helper for test schemas. Two things grew by iteration and should be watched: `GradeAnswer` now has six ordered refusals, and the exam update has a second write path for one column.
- **Tech debt trend.** Product debt is flat to better (guard tests added with each fix). Test-infrastructure debt showed up: Postgres tests that skip silently (decision 4).
- **Progress against "ready".** No critical issue is open. BA and UAT are satisfied on what UAT can run without paid calls. The next step of the ready definition is the owner's walkthrough, and that needs a deploy. More features do not move it.
- **Model fit, workers.** Good in this scope, including judgment: measuring before building (#563), flagging residues instead of hiding them (#570 draft path), refusing relayed approvals. If the workers are in fact on Haiku, the results do not argue for a change; the two weak spots were rule compliance (default work) and a missed call-path question that the architect also missed.
- **Model fit, Supervisor.** Mechanical compliance is now good where a script exists. The remaining misses are facts stated from memory (decision 8) and one T4 classification by eye (decision 2).
- **Model fit, architect.** Review speed is adequate. The misses were of one kind: judging a change by its diff without following what it switches on (decision 10).

## Open recommendations for the user

1. **Release before more depth (owner decision).** The demo is about 95 PRs behind main. Twelve of the last 23 closures refine AI grading, which nobody outside the swarm has seen. I recommend one QA redeploy and a walkthrough before further AI work. Before any host with real exams is redeployed with a key: every existing exam becomes `suggest`, and answers waiting for grading from the last 24 hours get a suggestion within minutes.
2. **Standing orders with bounds (owner decision).** Each paid call, Lighthouse run and local download waits for you in the session of the role that does it. One bounded order per kind, typed once in that session and written into `CLAUDE.md` by you, would remove most of the waiting: for example "bb-uat may make up to 20 paid AI calls a day on the local stack" and "Lighthouse and k6 may run against localhost".
3. **Decisions waiting for you.** DEC-003 (#560): who may set an exam to `auto`. The default AI grading mode (built as `suggest`). Whether to rotate the x.ai key after the scratch copy. The translated labels for audit actions. #428 (visual baselines).
4. **Worker model (owner, information).** `_common.md` says the workers run Sonnet; they report Haiku. Say which you intend. The quality of this scope does not argue for a change.
5. **Still owner-blocked:** #32 (performance run, partial), #396 (host hygiene), #473 (needs a redeploy), #519 and #528 (need paid calls for the last evidence).

## Applied changes (PR links)

See the retro-012 apply issues; links added when the PRs merge.
