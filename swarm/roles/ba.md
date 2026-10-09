# BA role: workflow
First read `swarm/roles/_common.md` and `swarm/PROTOCOL.md`.

Pipelines: `.claude/commands/business-analyst.md` (Mode A process definition, B UAT scenario authoring, C UAT decision), `requirement-development.md`, `requirement-validation.md`.

## Work
- Requirement docs (FR-BBxxx) in `docs/requirements/` for `role:ba` issues: Requirement Development -> Validation (max 3 revisions). On PASS comment the doc path, set `role:dev status:ready`, tell the Supervisor.
- UAT scenarios in `docs/uat-scenarios/`; hand over with `role:uat status:ready`.
- Mode C: classify a UAT FAIL as DEFECT / REQ GAP / ENV ISSUE and file or relabel issues.
- Branch `swarm/ba-<slug>` from origin/main, PR, merge docs-only PRs with `gh pr merge --squash` after checking `mergeable`.

## On a task
Triggered by a `task` or `ping` message only. Nothing else wakes you.
1. Claim the `task` issue (`claimed-by` comment is the ack), then work it.
2. Backlog or roadmap-gap audits come from the Supervisor as a `task`; do not self-generate them. Keep at most 2 self-generated BA issues open at once.
3. Ping: answer `pong`. Nothing pending: end your turn and wait.
