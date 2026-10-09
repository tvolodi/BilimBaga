# Swarm protocol: workflow rules

Roles: `bb-supervisor`, `bb-architect`, `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, `bb-infra`. Worktrees: `.claude/worktrees/<role>`; Infra lives in `..\ai-dala-infra`.

## Addressing
By name via `SendMessage`. Sessions without `-n` get derived suffixes, so resolve the exact name from `ListAgents` by prefix. Reply by copying the incoming `from` into `to`.

## Message format
Line 1: a self-contained sentence. Then one JSON object. Types: `task` (issue, action implement|fix|write-requirement|uat-run|e2e-sweep|deploy|retro-apply, branch, prio, deadline_min), `result` (issue, branch, action, result done|failed|blocked|needs-uat|pr-open, pr, detail, attempts), `question`/`escalate`, `ping`/`pong`. Messages carry pointers, not content; durable facts go into the issue.

## Label state machine
Labels: `needs-architect` (request marker, see Architect), `role:dev|ba|uat|infra`, `status:ready|in-progress|review|uat|blocked|done`, `type:bug|feature|infra|tech-debt|test`, `prio:p0|p1|p2`, `swarm`, `needs-live-db`.
new -> ready; ready -> in-progress (worker starts); in-progress -> review (PR open, review passed); review -> uat (merged, role flips to `role:uat`); exception: a PR that changes only `*_test.go` or `*.test.tsx` files, with CI green on its merge commit, goes review -> done. The Supervisor checks `gh pr view <n> --json files` and the CI state, closes the issue with a comment that names the PR and merge commit, and no UAT step runs. uat -> done (closed), only after the PR is merged to main and the UAT check ran on main; a PASS on a branch build is a comment and does not change status; uat -> ready + `role:dev` on FAIL or PARTIAL, and on any BA or Supervisor send-back (reopen +1 comment each time); any -> blocked with a reason comment. Every status change is one `gh issue edit` that also removes the previous `status:*` label. An issue is closed only with `status:done` set and only after `gh pr view <n> --json state` shows MERGED for its PR (issues without a PR: after the check named in the issue). The Supervisor closes BA, infra and UAT-test items through the same step. A close with `status:ready`, `status:in-progress`, `status:review` or `status:uat` set is a defect for the next tick's reconcile.
Claiming: Supervisor assigns and posts `supervisor: assigned to bb-devN` on the issue before sending the task; workers take only matching role + `ready` (or own `in-progress`), by prio then number; before posting `claimed-by: bb-devN`, read the issue comments and skip the issue if another dev holds a claim without a later release; one dev per issue. The claim comment is the ack for a task: the Supervisor re-sends the task once after `deadline_min` if no claim comment exists, then treats the session as dead. While more than 5 `status:uat` issues are open, workers do not self-claim `type:feature`.

## Branches, merges, migrations
- Branches `swarm/<issue>-<slug>`.
- Migration number: take `swarm/locks/migration.lock`, pick `max(origin/main)+1`, commit on a pushed branch, release after merge or abandon.
- Merge: Supervisor checks `gh pr view --json mergeable,mergeStateStatus` and overlap with other open PRs, then sends `merge-ok` with the full 40-character head SHA (`gh pr view <n> --json headRefOid`). The dev merges `origin/main` into the branch (merge, not rebase), pushes, and runs `gh pr merge <n> --squash --match-head-commit <full SHA>`. If the merge is rejected (base moved), merge `origin/main` again and retry with the same SHA. Delete the remote branch only after `gh pr view <n> --json state` shows MERGED.
- Stack: `make dev` / docker runs on one session at a time (`swarm/locks/stack.lock`, UAT by default). UAT fast-forwards its worktree and rebuilds to deploy merged changes.
- Lock dir and `swarm/state/` live in the main checkout (absolute paths from worktrees).

## Architect
`bb-architect` (Opus, event-driven, no `/loop`) reviews judgment calls: architecture, decisions, the way the team works, retro analysis, the big picture. Advisory with a written reason; it merges and dispatches nothing; the owner overrides.

**Routing.** The Supervisor routes mechanically each tick; any role may also add the `needs-architect` label (a request). The Supervisor sends one `task` (action `architect-review`, issue or PR, trigger id, one-sentence question) per issue, never more than one task in flight, at most 3 per day unless `prio:p0`. Triggers, any one is enough:
- T1: label `needs-architect` on an open issue or PR.
- T2: retro due (closed minus `last_retro_closed_count` >= interval), or 3 escalations.
- T3: second failed attempt on an issue (before reassigning to the other dev).
- T4: a PR changes a protected path: `backend/internal/auth*`, RBAC or permission code, `backend/migrations/` (new file), `deploy/`, compose files, `swarm/`, `docs/architecture-guide.md`, `docs/requirements/api-conventions.md`; or it changes more than 15 files or 600 lines.
- T5: two open issues or docs contradict each other, or a `question` message with `"kind":"design"`.
- T6: a `prio:p0|p1` issue with a security label or "SECURITY" in the title.
- T7: an issue reopened twice (N>=2).
- T8: readiness review: no open `prio:p0` and the owner asked for readiness, or BA and UAT both report satisfied.
Not routed: routine bugs, docs-only fixes, test-only changes, UAT scenario edits.

**Waiting.** A T4 PR gets no `merge-ok` until the issue carries an `architect-decision:` comment other than `changes`. For other triggers the issue stays at its status with `needs-architect` until the decision comment appears; no reply within the task `deadline_min` (default 60) -> one re-send, then `status:blocked` with reason and a report to the owner.

**Result.** Issue comment starting `architect-decision: approve|changes|split|escalate-owner`; lasting decisions in `docs/requirements/DEC-NNN.<slug>.md`.

## Escalation
Max 3 attempts per issue, then `result: failed`. Supervisor: reassign to the other dev; then swap role; then `status:blocked` + tech-debt investigation issue + retro entry. The user is informed through reports only.

## Anti-idle
Order: assigned task, then the oldest ready issue with own role label once the Supervisor assigns it. Workers run no `/loop` and no periodic tick. The Supervisor owns idle monitoring and default work (BA audits, UAT sweeps, dev tech-debt) and sends it as a `task`. `bb-architect` is event-driven too.

## Durable state
GitHub issues and labels are the truth. `docs/handoffs/<run-id>/` payloads. Report files are numbered by GitHub issue: `docs/issue-reports/ISS-088-<slug>.md`, `docs/code-reviews/ISS-088-review.md`. `swarm/state/*.json`: workers, retro, escalations. `docs/retrospectives/retro-NNN.md`.

## Resources
- SQL-changing PRs: real-Postgres test, or label `needs-live-db`; the `internal/schemaguard` test stays green.
- Test parallelism: `npx vitest run --maxWorkers=2`, `go test -p 2 ./...`.
- UAT live runs need >= 8 GB free memory; the Supervisor reports low memory.
- Memory hold: when free RAM < 2.5 GB, devs run only targeted single-package tests (no full suite, tsc or docker) and merge on GitHub CI plus UAT evidence; the Supervisor announces the hold and the release at 4 GB free.

## Environments (workflow part)
Test traffic goes to local stacks and the QA instance `bilimbaga-qa.ai-dala.com` (not built yet, #106). UAT scenarios declare `Target: local | qa`; seed scripts need an explicit `E2E_API_URL`. See `docs/requirements/DEC-001.Environments-production-class-demo-and-qa.md`.

## Checkpoints and heartbeats
File `swarm/state/<role>.json`: `{role,status busy|idle|blocked,issue,branch,step,next_action,last_tick_utc,blockers}`. Write at every step change (no per-tick heartbeat) through `swarm/bin/checkpoint.sh <role> <issue> <branch> <step> <next_action> [blockers]` (atomic; `SWARM_STATUS` override). Step changes also get a short issue comment; heartbeats are file-only. Liveness is on demand: the Supervisor sends `ping`, the worker answers `pong` with its status. No `pong` within one Supervisor tick means a dead session (restart via the ensure-up watchdog #145, or reassign). Issue comments win over the file.

## One caution
`bilimbaga-test.ai-dala.com` is the customer demo. Do not run tests, seeds or deploys against it unless the owner orders it. Test on local stacks.
