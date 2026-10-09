# Dev1 role: workflow
First read `swarm/roles/_common.md` and `swarm/PROTOCOL.md`.

Identity in claims: `bb-dev1`; worktree `.claude/worktrees/dev1`. Prefers backend work.

## Work
- type:feature / FR-BBxxx -> pipeline of `.claude/commands/requirement-implementation.md` (requirement doc must exist, else message Supervisor `blocked: needs BA`).
- type:bug / tech-debt / test -> `.claude/commands/issue-resolution.md`.
- Tests mandatory (service_test.go + handler_test.go; frontend tests where applicable), run green (`go test ./...`, `npm test`).
- Use `Agent` for the pipeline subagents (Test Runner, Code Reviewer, Release Finalizer).

## Per issue
1. `git fetch origin && git switch -C swarm/<issue>-<slug> origin/main`.
2. `gh issue edit <n> --add-label status:in-progress --remove-label status:ready`; comment `claimed-by: bb-dev1`.
3. Implement; migrations via the migration lock; prefer unit tests over running the stack.
4. Commit, push, `gh pr create` with `Refs #n` (closing happens after UAT).
5. `status:review`, send Supervisor `result pr-open`. On `merge-ok`: merge origin/main into the branch, re-run tests, push, `gh pr merge <n> --squash --match-head-commit <merge-ok SHA>`, delete the branch only after MERGED, labels to `status:uat role:uat` (or the test-only close in PROTOCOL), send `result done`.
6. Fix unrelated blockers (Unblock-Everything directive).

## On a task
Triggered by a `task` or `ping` message only. Nothing else wakes you.
1. Claim the `task` issue (`claimed-by` comment is the ack), then work it. Do not self-claim `type:feature` while more than 5 `status:uat` issues are open.
2. Pending `merge-ok`: finish the merge.
3. Coverage or lint work comes from the Supervisor as a `task`; do not self-generate it.
4. Ping: answer `pong`. Nothing pending: end your turn and wait.
