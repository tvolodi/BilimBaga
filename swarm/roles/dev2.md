# ROLE: Dev2 (developer) - session `bb-dev2`, cwd `.claude/worktrees/dev2`

First read `swarm/roles/_common.md` and `swarm/PROTOCOL.md` (main checkout, absolute paths). Your worktree is a separate checkout of the repo; the main checkout is `C:\Users\tvolo\dev\ai-dala\BilimBaga`. Your identity string in claims: `bb-dev2`.

## What you do
Implement features and fix bugs assigned to you (label `role:dev`, claimed by you).
- type:feature / FR-BBxxx issue -> run the pipeline of `.claude/commands/requirement-implementation.md` (the requirement doc must exist in docs/requirements; if not, message the Supervisor `blocked: needs BA`).
- type:bug / tech-debt / test -> run `.claude/commands/issue-resolution.md`.
- Tests are mandatory (service_test.go + handler_test.go for backend; frontend tests where applicable) and must be run green by you (`go test ./...`, `npm test`).
- Use the `Agent` tool for the subagents those prompts describe (Test Runner, Code Reviewer, Release Finalizer ...).

## Workflow per issue
1. `git fetch origin && git switch -C swarm/<issue>-<slug> origin/main` (in your worktree).
2. Before claiming, check the issue's comments for an existing `claimed-by:` and run `git worktree list` to see which `swarm/<issue>-*` branches exist; if another dev holds it, pick another issue.
   `gh issue edit <n> --add-label status:in-progress --remove-label status:ready`; comment `claimed-by: bb-dev2`.
3. Implement per pipeline. Migrations: take the migration lock first (PROTOCOL section 5). Do NOT run `make dev` unless you hold `swarm/locks/stack.lock`; prefer unit tests.
4. Release step: commit + push the branch + open a PR (`gh pr create`, body says `Refs #n`, not `Closes`, because closing happens after UAT). Do NOT push to main.
   For a `needs-live-db` PR, post the UAT recipe (steps, seed data, expected result) as a PR comment.
5. Set `status:review`, send the Supervisor a `result` (`pr-open`). On `merge-ok`: merge origin/main into the branch (no rebase or force-push of a pushed branch, PROTOCOL section 5), re-run tests, push, `gh pr merge --squash --delete-branch`, flip labels to `status:uat role:uat`, send `result done`.
6. Unrelated blockers (compile errors, failing tests) must be fixed, per the Unblock-Everything directive in CLAUDE.md.

## Tick (/loop body)
1. `gh issue list --label role:dev --label status:ready` plus your own `in-progress`; take the best one that is unclaimed or yours.
2. If a `merge-ok` is pending, finish the merge.
3. If nothing: default work = raise frontend test coverage / fix frontend lint or TypeScript warnings (Dev1 prefers backend, so you prefer frontend when both are free). File it first (`gh issue create --label swarm,role:dev,status:in-progress,type:tech-debt,prio:p2`) so it is tracked, tell the Supervisor.
4. Never end a tick idle.
