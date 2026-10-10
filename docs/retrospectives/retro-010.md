# Retro 010

Period: closures ranked 164 to 178 by closedAt (#375 to #405), 2026-10-09 20:21Z to 2026-10-10 02:50Z.

## Metrics

| Metric | Value |
|---|---|
| Closures in scope | 15 |
| Cycle time, median / p90 | 3.1 h / 20.7 h (two populations, see Corrections) |
| Reopens | 1 of 15 (#42) |
| Mix | product 4, test and swarm-test 2, process 9 |

## Applied changes (PR links)

Pending: the retro-apply issue for decisions 1 to 8 is filed after this PR merges (see the next retro-apply task).

Author: bb-architect, trigger T2. Scope: ranks 164 to 178 by `closedAt`, #375 (2026-10-09 20:21:31Z) to #405 (2026-10-10 02:50:22Z). Data pulled 02:54Z to 03:10Z from GitHub (issue comments, label events, PR files, check runs) and `swarm/state/`. Rule text checked against `origin/main` at `12e5bf9` or later; each Before string below occurs once.

The 15 issues: 375, 380, 324, 384, 395, 370, 386, 387, 42, 392, 371, 229, 393, 1, 405.

## Corrections to the metrics table

- **Reopens: 1 of 15, not 2.** #42 was reopened (UAT PARTIAL 19:52Z, 400 vs 422). #371 is the UAT task issue that reported reopens; it was not reopened itself. The reopens it reported (#176 and #249, both now at reopen 2) are outside this scope.
- **Mix.** Product work: 4 (#324, #384, #42, #229). Test and swarm-test fixes: 2 (#370, #405). Process-generated: 9 (retro-apply #375, #380; coverage sweeps #386, #387, #392, #393; health check #395; UAT task #371; sweep task #1).
- **Cycle time has two populations.** Items created and closed inside one active window took 1 to 26 min (#375, #380, #384, #395, #392, #393, #405). Items that crossed the 21:01Z to 02:22Z gap took 5.5 to 6.6 h (#370, #386, #387). The p90 (20.7 h) is #1, #42 and #229, which carried the 14:07Z to 19:56Z `.env` block from retro-009.

## Went well

- **Dev loop is fast and exact.** Claim to PR: 1 to 10 min on all 9 dev items. No dev `failed` result. Both retro-apply PRs (381, 382) matched their Before strings; dev2 noticed that retro-009 decision 5 overlapped #375 item 5 and held it instead of applying it twice.
- **Feature split worked.** #324 backend (PR 383) and #384 frontend (PR 385) merged 5 min apart and were both verified by UAT on main `8ee4e45` at 20:44Z, with the 404, 403, 409 and 401 cases named.
- **BA rulings are fast and precise.** #405: 404 by design, stale spec, line numbers named, 1 min 14 s after the claim. #42: per-case 400 vs 422 table, docs rule first (PR 378).
- **Test-only fast path worked when CI was green.** #392 and #393 went review to done 2 s and 9 s after their merge comments, with the merge commit named. Compare the 100.9 min median in retro-008.
- **Full E2E sweep finally ran.** #1: 210 passed, 3 failed, 4 skipped on main `dd85cd3`. Each failure was filed (#405, #406, #407). Two of the three were stale specs, one is a test-data defect; no product regression.
- **Pinned-SHA merge retry** worked again (#42, PR 377, base moved, same SHA).
- **UAT evidence quality.** Every PASS names the main SHA and the observed status codes.

## Went badly

1. **No swarm activity for 5 h 21 min.** No issue comment by any role between 21:00:58Z (#390) and 02:22:49Z. Workers are event-driven since PR 341, so when the Supervisor stops, everything stops. #370, #386 and #387 sat at `status:uat` for 5.5 to 6.5 h; #390 sat in review. Root cause not derivable: `swarm/state/ticks.log` does not exist, although retro-009 decision 6 (tick log) merged at 20:26Z in PR 382. `supervisor.json` and `workers.json` are stamped 01:29:13Z but describe work from about 16Z on 2026-10-09 (PR 283, #330, #331, "retro at closed 109"). The gap may be an owner pause; the files cannot say.
2. **The T4 architect gate was never applied.** PRs 372, 381 and 382 change `swarm/`; PR 383 changes `backend/internal/router/router.go` and the users authorisation rules (9 files, 218 lines). All four merged with `merge-ok` and no `architect-decision:` comment. The repository contains no `architect-decision:` comment at all. `supervisor.md` step 4b was live from 18:22Z. Root cause: no `bb-architect` session was running (no checkpoint before 02:09Z), and the protocol does not say what the Supervisor does then. Second cause: "RBAC or permission code" is not a path, so it cannot be applied mechanically.
3. **Test-only closes on red checks, again.** The retro-008 rule (re-run once, then a local run with the command in the close comment) was live from 20:21:24Z. PR 388 merged 20:51Z with `docker-build` failed; PR 389 merged 20:57Z, and on its merge commit `12e5bf9` `backend-tests`, `build` and `docker-build` all failed (Docker Hub 429). Neither was re-run. Both went to `status:uat` instead of the fast path, waited through the gap, and were closed "per the supervisor's direction" with no local command. Risk is retired: `a03ce7c` (PR 397) has all checks green and contains both changes. Root cause: the rule names "cancelled or flaked" and the Supervisor read a 429 as a third category.
4. **Closed with two status labels.** #324 closed with `status:in-progress` and `status:done`; #395 closed with `status:ready` and `status:done`, 3 h 57 min after retro-009 decision 4 (label step in the close) merged. Also #145, #157 and #193 are open at `status:done` since about 15Z on 2026-10-09, which tick step 1 says to close in the same tick. This is the third retro with a label defect and a new sentence each time. Root cause: tooling. A multi-label edit done by hand from a prose rule fails at a steady rate.
5. **#405 closed 11 s after merge on a branch run.** UAT ran PR 409's head on the live stack (12 of 12) before merge, which was the right check. The protocol says a branch PASS does not change status, and the test-only exception names `*_test.go` and `*.test.tsx` only, not `frontend/e2e/`. The close was sensible and outside the rules. Protocol gap.
6. **Status-code contract found one case at a time by UAT.** #42 (400 vs 422), #176 (413, reopen 2), #405 (403 vs 404): three BA rulings in 7 h, each after a UAT FAIL or an E2E failure. `api-conventions.md` gains a rule per incident and nothing in CI checks it.
7. **Default work has low yield.** Sweep 3 backend (#392) raised `internal/exams` from 63.2% to 67.3% in 135 lines. Four coverage sweeps closed while #249 (user-visible, reopen 2) is at `status:ready` and 4 E2E tests are skipped with no issue.
8. **Customer demo host touched as default work.** #395 ran `ssh hetzner-prod` and a health GET against the bilimbaga-test host on a Supervisor-generated task. `supervisor.md` step 5 says "Infra gets QA health/redeploy once QA exists"; QA does not exist (#106), so the check was pointed at the demo. `CLAUDE.md` Owner authority 5 says the demo is touched only on the owner's order for that action. The check was read-only and found real gaps (#350, #396, recovery point older than 26 h), but it is the owner's decision. See Open recommendations 1.
9. **Retro overhead.** Three retros were cut within 32 min on 2026-10-09 (19:26Z, 19:47Z, 19:58Z) because the trigger counts closures and UAT closes in batches. 2 of this scope's 15 closures are retro-apply issues. The PROTOCOL label state machine is now one paragraph of about 330 words.

## Root causes, grouped

| Cause | Items |
|---|---|
| Tooling: mechanical rules kept as prose | 3, 4, and the missing tick log in 1 |
| Protocol gap | 2 (architect down, vague path), 5 (E2E specs), 9 (trigger) |
| Role-prompt gap | 7 (default work order), 8 (health target) |
| Environment | 1 (session stop or pause), 3 (Docker Hub 429 on CI) |
| Architecture | 6 (no executable error contract) |

The pattern across retro-007 to retro-010: decisions were applied as text within minutes, and four of them were not followed in the next period (T4 hold, test-only re-run, close label step, tick log). More sentences are not moving these metrics. Decisions 1 to 3 below replace prose with scripts or with a restart.

## Decisions

1. **Status changes and closes go through one script.** Metric: closes with two status labels (2 of 15) and open issues at `status:done` (3) to 0.
   - New file `swarm/bin/set-status.sh <issue> <ready|in-progress|review|uat|blocked|done> [--close] [--pr <n>]`: removes every other `status:*` label and adds the new one in a single `gh issue edit`; with `--close` it requires `done`, checks `gh pr view <n> --json state` is MERGED when `--pr` is given, removes `role:*`, then closes. A test in `swarm/tests/` with a stubbed `gh` covers: two statuses set, close without MERGED refused, report-only close.
   - `swarm/PROTOCOL.md`, label state machine. Before: `Every status change is one \`gh issue edit\` that also removes the previous \`status:*\` label.` After: `Every status change and every close runs \`swarm/bin/set-status.sh <n> <status> [--close] [--pr <n>]\`, which removes the other \`status:*\` labels; no role edits status labels by hand.`

2. **Supervisor reloads its own rules and proves its ticks.** Metric: live rules not followed in the next period (4) to 0; tick gap derivable.
   - `swarm/RETRO.md` step 6. Before: `Role changes take effect when a session restarts or re-reads the file: the Supervisor sends each affected worker a \`reload-role\` message (first line self-contained) after the merge.` After: the same text, with this appended: ` The Supervisor then re-reads \`swarm/roles/supervisor.md\`, \`_common.md\` and \`PROTOCOL.md\` from \`origin/main\` itself, and writes one line \`<UTC> reload retro-NNN\` to \`swarm/state/ticks.log\`.`
   - New file `swarm/bin/tick-log.sh "<report line>"`: appends `<UTC> <line>` to `swarm/state/ticks.log`. `swarm/roles/supervisor.md` step 7. Before: `and append that report as one line, prefixed by the UTC time, to \`swarm/state/ticks.log\`` After: `and append that report with \`swarm/bin/tick-log.sh "<report>"\``
   - Proposed issue (type:infra, role:dev): `ensure-up.ps1` treats a last `ticks.log` line older than 30 min as a stalled Supervisor and nudges it, as it does for workers. Retro-009 recommendation 2 asked for this; the gap repeated.

3. **T4 gate: explicit paths, and a rule for an absent architect.** Metric: T4 PRs merged without a decision (4 of 4) to 0.
   - `swarm/PROTOCOL.md`, T4. Before: `\`backend/internal/auth*\`, RBAC or permission code, \`backend/migrations/\` (new file)` After: `\`backend/internal/auth*\`, \`backend/internal/rbac/\`, \`backend/internal/roles/\`, \`backend/internal/middleware/\`, \`backend/internal/deptscope/\`, \`backend/internal/router/router.go\`, \`backend/migrations/\` (new file)`
   - `swarm/PROTOCOL.md`, Waiting. Before: `A T4 PR gets no \`merge-ok\` until the issue carries an \`architect-decision:\` comment other than \`changes\`.` After: the same sentence, then: ` If \`bb-architect\` is not in \`ListAgents\`, the Supervisor starts it with \`swarm/bin/ensure-up.ps1\` and holds the PR; T4 is not skipped because the architect is down. Two exceptions need no decision: a retro-apply PR whose diff equals the Decisions of a retro the architect drafted, and a PR that changes only \`swarm/tests/\`.`

4. **A registry 429 is a flake.** Metric: test-only closes on a red check without a re-run or local command (2 of 4) to 0.
   - `swarm/PROTOCOL.md`, test-only exception. Before: `Such a close is not a supervisor decision for an unexplained result.` After: `A check that fails on an image pull (Docker Hub 429, \`toomanyrequests\`) is a flake under this rule: \`gh run rerun <run-id> --failed\` once, then the local command. No issue is closed on a Supervisor direction alone while a required check is red.`

5. **E2E spec-only PRs join the test-only exception.** Metric: closes outside the rules for spec fixes (2: #405, #406) to 0.
   - `swarm/PROTOCOL.md`. Before: `exception: a PR that changes only \`*_test.go\` or \`*.test.tsx\` files, with every required check green on its merge commit, goes review -> done.` After: `exception: a PR that changes only \`*_test.go\`, \`*.test.tsx\` or \`frontend/e2e/**\` files, with every required check green on its merge commit, goes review -> done. For \`frontend/e2e/**\` the close comment also names a live run of the changed specs on the PR head or on main.`

6. **Default work ordered by value.** Metric: coverage sweeps closed while a reopened bug or a skipped E2E test is open (4) to 0.
   - `swarm/roles/supervisor.md` step 5. Before: `Dev gets tech-debt (coverage, lint/vet, flaky tests, TODOs)` After: `Dev gets default work only when no \`role:dev status:ready\` issue exists, in this order: skipped or stale E2E specs, flaky tests, lint/vet, TODOs, then coverage; a coverage task names one package and a target of at least +15 points`
   - Same step. Before: `Infra gets QA health/redeploy once QA exists.` After: `Infra gets QA health/redeploy once QA exists; until then Infra gets no self-generated work that reaches a remote host.` Applies now; Open recommendation 1 may replace it.

7. **Retro trigger: a minimum gap.** Metric: retros per day (3 in 32 min on 2026-10-09) and retro-apply share of closures (2 of 15).
   - `swarm/RETRO.md`, Owner line. Before: `checked each tick.` After: `checked each tick. No retro starts less than 4 h after the previous cut unless step 8 applies; the scope is then every closure since the cut, not only the first \`interval\`, and step 7 stores the count at the pull.`
   - This changes the retro-008 cut rule on purpose: the scope still leaves no closure unreviewed, and a batch close no longer produces a retro for a 10 min window.

8. **Error contract as a test (architecture, proposed issue, not a rule).** Metric: UAT reopens caused by status or error code (#42, #176 twice) to 0. File `type:tech-debt role:dev prio:p1`: a table test in `backend/internal/router` (the `users_authz_test.go` harness) that sends each error class of `api-conventions.md` sections 6 and 8 (malformed JSON, unknown key, invalid field value, oversize body, out-of-scope id, unknown id, no token) to every mutating route and asserts status and code. New routes fail the test until listed. This PR is T4 (router) and comes to the architect.

Carried, no change: retro-008 decisions 1, 2, 5, 6, 7 and retro-009 decisions 1 to 4 show no counter-evidence in this scope.

Supervisor actions for the next tick (not rule changes):
- Close #145, #157 and #193 (open at `status:done`), or reopen them with a reason. Remove `status:in-progress` from #324 and `status:ready` from #395.
- Route #249 to the architect under T7 (reopen 2, still open). #176 reached reopen 2 and closed at 02:50Z without T7; it is in the next retro's scope.
- Route PR 383 (#324) as a post-merge T4 review; it is the only product PR that passed the gate unreviewed. PRs 372, 381 and 382 need no review (stub fix and retro-apply).
- File one issue for the 4 skipped dept-admin-scoping subtree tests from #1 if none exists.
- Rewrite `supervisor.json` and `workers.json` from live state; their content is from about 16Z on 2026-10-09.

## Way of working and model fit

- **The team's shape is right; the Supervisor is the single point of failure.** Event-driven workers removed idle polling and claim collisions (1 in this scope, #42 at 16:13Z, before the rule). The cost is that one stopped session stops six. Decision 2 and its watchdog issue address that.
- **Wrong role for a task: none in scope.** BA before dev on #405 and #176 was the right order and cost under 2 min each.
- **Duplicated work: none merged.** Retro-008 and retro-009 both carried the escalations-timestamp change; dev2 caught it.
- **Architecture drift: one area.** Error status and code rules live in a document and are discovered by UAT. Decision 8 makes them executable. No other drift seen in the four product PRs: handlers are thin, rules are in `users/service.go`, no migration, no new package.
- **Tech debt trend.** Backend coverage is rising where sweeps aim (portal 59.7% to 97.9%), but sweeps choose by lowest number, not by risk. `internal/questions` (57.7%) was skipped twice because of an open branch. Decision 6 puts coverage last.
- **The work is drifting from the "ready" definition.** Ready is no critical issue, BA and UAT satisfied. In this scope 9 of 15 closures were process work and no closure was a roadmap feature beyond FR-BB18 AC-13 and FR-BB116. The useful readiness signals were the E2E sweep (3 failures, none a product regression) and UAT on #249. A T8 readiness review is reasonable once #249 and #407 close.
- **Model fit, workers (Sonnet per `_common.md` rule 9).** Good fit for bounded, specified tasks: exact text edits, endpoint plus tests, coverage, spec fixes, UAT runs with evidence. Two weak spots: a dev covers the named input and not the input class (#288 in retro-008, #176 raw CSV here), and a dev delivers the half of a full-stack issue that matches its preference (#324). Both are handled by splitting at dispatch and by decision 8, not by a model change.
- **Model fit, Supervisor.** Its judgment calls were sound (splitting #324, accepting the 429 as infrastructure, running PR 409 on the stack before merge). Its failures were all long-horizon mechanical compliance: four live rules missed, stale state files, no tick log. That is a context and tooling problem first. Decisions 1 to 3 move those steps into scripts. If the misses continue after that, the next step is a Supervisor restart after each retro-apply, then a stronger model for the Supervisor only; the model choice is the owner's (Open recommendation 3).
- **Model fit, architect.** The role existed on paper for 8 h and reviewed nothing. Until this task no trigger reached it.

## Open recommendations for the user

1. **Health checks on the bilimbaga-test host (owner decision).** #395 used `ssh hetzner-prod` and a health GET on the customer demo as Supervisor-generated work. Either give a standing order ("read-only health check of the demo host is allowed, at most N per day"), or keep decision 6's stop. The check itself is useful: it found that the last database dump is from 2026-10-09 06:53Z and that no backup is scheduled (#350).
2. **Docker Hub rate limit on CI (owner, needs a credential).** Unauthenticated pulls hit 429 on two consecutive merge commits and turned `backend-tests`, `build` and `docker-build` red. Options: a Docker Hub token as a repository secret with `docker/login-action`, or pull the base images from `mirror.gcr.io` / GHCR. Workflow files and secrets are yours.
3. **Supervisor model and uptime (owner decision).** Say whether the 21:01Z to 02:22Z stop was your pause. If not, the Supervisor needs the watchdog from decision 2. If the Supervisor keeps missing live rules after decisions 1 to 3, consider a stronger model for that one session.
4. **Still owner-blocked:** #232, #263 (holds `migration.lock`), #284, #350, #351, #396, #32. Unchanged from retro-009 except #396 (new, low severity).
