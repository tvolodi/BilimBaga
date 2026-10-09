# Swarm protocol: workflow rules

Roles: `bb-supervisor`, `bb-architect`, `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, `bb-infra`. Worktrees: `.claude/worktrees/<role>`; Infra lives in `..\ai-dala-infra`.

## Addressing
By name via `SendMessage`. Sessions without `-n` get derived suffixes, so resolve the exact name from `ListAgents` by prefix. Reply by copying the incoming `from` into `to`.

## Message format
Line 1: a self-contained sentence. Then one JSON object. Types: `task` (issue, action implement|fix|write-requirement|uat-run|e2e-sweep|deploy|retro-apply, branch, prio, deadline_min), `result` (issue, branch, action, result done|failed|blocked|needs-uat|pr-open, pr, detail, attempts), `question`/`escalate`, `ping`/`pong`. Messages carry pointers, not content; durable facts go into the issue.

## Label state machine
Labels: `needs-architect` (request marker, see Architect), `role:dev|ba|uat|infra`, `status:ready|in-progress|review|uat|blocked|done`, `type:bug|feature|infra|tech-debt|test`, `prio:p0|p1|p2`, `swarm`, `needs-live-db`.
new -> ready; ready -> in-progress (worker starts); in-progress -> review (PR open, review passed); review -> uat (merged, role flips to `role:uat`); uat -> done (closed), only after the PR is merged to main and the UAT check ran on main; a PASS on a branch build is a comment and does not change status; uat -> ready + `role:dev` on FAIL or PARTIAL, and on any BA or Supervisor send-back (reopen +1 comment each time); any -> blocked with a reason comment. Every status change is one `gh issue edit` that also removes the previous `status:*` label.
Claiming: Supervisor assigns and posts `supervisor: assigned to bb-devN` on the issue before sending the task; workers take only matching role + `ready` (or own `in-progress`), by prio then number; before posting `claimed-by: bb-devN`, read the issue comments and skip the issue if another dev holds a claim without a later release; one dev per issue. While more than 5 `status:uat` issues are open, workers do not self-claim `type:feature`.

## Branches, merges, migrations
- Branches `swarm/<issue>-<slug>`.
- Migration number: take `swarm/locks/migration.lock`, pick `max(origin/main)+1`, commit on a pushed branch, release after merge or abandon.
- Merge: Supervisor checks `gh pr view --json mergeable,mergeStateStatus` and overlap with other open PRs, then sends `merge-ok`. The dev merges `origin/main` into the branch (merge, not rebase), pushes, `gh pr merge --squash`.
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
Order: assigned task, oldest ready issue with own role label, role's self-generated default work. Every worker except `bb-architect` runs a dynamic `/loop` (5-10 min).

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
File `swarm/state/<role>.json`: `{role,status busy|idle|blocked,issue,branch,step,next_action,last_tick_utc,blockers}`. Write at every step change and once per tick (idle workers too) through `swarm/bin/checkpoint.sh <role> <issue> <branch> <step> <next_action> [blockers]` (atomic; `SWARM_STATUS` override). Step changes also get a short issue comment; heartbeats are file-only. A `busy` heartbeat older than 45 min (`stale_heartbeat_min`) means stalled; idle/blocked older than 2x that means a dead session. Issue comments win over the file.

## One caution
`bilimbaga-test.ai-dala.com` is the customer demo. Do not run tests, seeds or deploys against it unless the owner orders it. Test on local stacks.
