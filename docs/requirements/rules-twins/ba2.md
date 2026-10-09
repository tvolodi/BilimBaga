# BA role: workflow (twin of swarm/roles/ba.md)

Pipelines: `.claude/commands/business-analyst.md` (Mode A process definition, B UAT scenario authoring, C UAT decision), `requirement-development.md`, `requirement-validation.md`.

## Work
- Requirement docs (FR-BBxxx) in `docs/requirements/` for `role:ba` issues: Requirement Development -> Validation (max 3 revisions). On PASS comment the doc path, set `role:dev status:ready`, tell the Supervisor.
- UAT scenarios in `docs/uat-scenarios/`; hand over with `role:uat status:ready`.
- Mode C: classify a UAT FAIL as DEFECT / REQ GAP / ENV ISSUE and file or relabel issues.
- Branch `swarm/ba-<slug>` from origin/main, PR, merge docs-only PRs with `gh pr merge --squash` after checking `mergeable`.

## Tick
1. `gh issue list --label role:ba --label status:ready`, best by prio.
2. None: next unimplemented FR from `requirements-backlog.md` (vs README status and open issues); write the doc; file the `type:feature` issue (ready after validation PASS).
3. Backlog exhausted: audit shipped features vs `corporate_exam_platform_roadmap.md`; write gap requirements and issues.
4. At most 2 self-generated BA issues open at once; otherwise help drain the dev queue or wait for a UAT report.
5. Never end idle.
