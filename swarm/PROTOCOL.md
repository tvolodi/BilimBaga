# Swarm protocol: workflow rules

Roles: `bb-supervisor`, `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, `bb-infra`. Worktrees: `.claude/worktrees/<role>`; Infra lives in `..\ai-dala-infra`.

## Addressing
By name via `SendMessage`. Sessions without `-n` get derived suffixes, so resolve the exact name from `ListAgents` by prefix. Reply by copying the incoming `from` into `to`.

## Message format
Line 1: a self-contained sentence. Then one JSON object. Types: `task` (issue, action implement|fix|write-requirement|uat-run|e2e-sweep|deploy|retro-apply, branch, prio, deadline_min), `result` (issue, branch, action, result done|failed|blocked|needs-uat|pr-open, pr, detail, attempts), `question`/`escalate`, `ping`/`pong`. Messages carry pointers, not content; durable facts go into the issue.

## Label state machine
Labels: `role:dev|ba|uat|infra`, `status:ready|in-progress|review|uat|blocked|done`, `type:bug|feature|infra|tech-debt|test`, `prio:p0|p1|p2`, `swarm`, `needs-live-db`.
new -> ready; ready -> in-progress (worker starts); in-progress -> review (PR open, review passed); review -> uat (merged, role flips to `role:uat`); uat -> done (closed); uat -> ready + `role:dev` on FAIL (reopen +1); any -> blocked with a reason comment.
Claiming: Supervisor assigns; workers take only matching role + `ready` (or own `in-progress`), by prio then number; `claimed-by: bb-devN` comment; one dev per issue.

## Branches, merges, migrations
- Branches `swarm/<issue>-<slug>`.
- Migration number: take `swarm/locks/migration.lock`, pick `max(origin/main)+1`, commit on a pushed branch, release after merge or abandon.
- Merge: Supervisor checks `gh pr view --json mergeable,mergeStateStatus` and overlap with other open PRs, then sends `merge-ok`. The dev merges `origin/main` into the branch (merge, not rebase), pushes, `gh pr merge --squash`.
- Stack: `make dev` / docker runs on one session at a time (`swarm/locks/stack.lock`, UAT by default). UAT fast-forwards its worktree and rebuilds to deploy merged changes.
- Lock dir and `swarm/state/` live in the main checkout (absolute paths from worktrees).

## Escalation
Max 3 attempts per issue, then `result: failed`. Supervisor: reassign to the other dev; then swap role; then `status:blocked` + tech-debt investigation issue + retro entry. The user is informed through reports only.

## Anti-idle
Order: assigned task, oldest ready issue with own role label, role's self-generated default work. Every worker runs a dynamic `/loop` (5-10 min).

## Durable state
GitHub issues and labels are the truth. `docs/handoffs/<run-id>/` payloads. Report files are numbered by GitHub issue: `docs/issue-reports/ISS-088-<slug>.md`, `docs/code-reviews/ISS-088-review.md`. `swarm/state/*.json`: workers, retro, escalations. `docs/retrospectives/retro-NNN.md`.

## Resources
- SQL-changing PRs: real-Postgres test, or label `needs-live-db`; the `internal/schemaguard` test stays green.
- Test parallelism: `npx vitest run --maxWorkers=2`, `go test -p 2 ./...`.
- UAT live runs need >= 8 GB free memory; the Supervisor reports low memory.

## Environments (workflow part)
Test traffic goes to local stacks and the QA instance `bilimbaga-qa.ai-dala.com` (not built yet, #106). UAT scenarios declare `Target: local | qa`; seed scripts need an explicit `E2E_API_URL`. See `docs/requirements/DEC-001.Environments-production-class-demo-and-qa.md`.

## Checkpoints and heartbeats
File `swarm/state/<role>.json`: `{role,status busy|idle|blocked,issue,branch,step,next_action,last_tick_utc,blockers}`. Write at every step change and once per tick (idle workers too) through `swarm/bin/checkpoint.sh <role> <issue> <branch> <step> <next_action> [blockers]` (atomic; `SWARM_STATUS` override). Step changes also get a short issue comment; heartbeats are file-only. A `busy` heartbeat older than 45 min (`stale_heartbeat_min`) means stalled; idle/blocked older than 2x that means a dead session. Issue comments win over the file.

## One caution
`bilimbaga-test.ai-dala.com` is the customer demo. Do not run tests, seeds or deploys against it unless the owner orders it. Test on local stacks.
