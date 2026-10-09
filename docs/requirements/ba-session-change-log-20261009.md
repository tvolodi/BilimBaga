# BA session change log (2026-10-09)

Files changed by the `bb-ba` session, with the reason. All changes are docs-only and merged via squash PRs.

## Repository files

| File | Change | Why | PR |
|------|--------|-----|----|
| `docs/requirements/FR-BB51.Dashboard-metrics-API.md` | AC-1..7 and AC-10 ticked; AC-8 rewritten (single-tenant schema, no `tenant_id`) | Resolve conformance gap G2 (PR77-PR83-PR86): stale checkboxes and obsolete tenant scoping text | #247 |
| `docs/requirements/README.md` | FR-BB51 row: "Partial (only AC-9 perf budget unverified)" | Record the remaining gap precisely | #247 |
| `docs/requirements/README.md` | FR-BB317 row: Partial -> Implemented (#40 shipped) | `UsersListPage.tsx` uses `DepartmentTreeSelect`, issue #40 closed | #255 |
| `docs/requirements/rules-twins/README.md` | New | Index of original -> twin files | #257 |
| `docs/requirements/rules-twins/CLAUDE2.md` | New | Workflow/convention/pipeline rules of `CLAUDE.md` without permission rules | #257 |
| `docs/requirements/rules-twins/PROTOCOL2.md` | New | Same for `swarm/PROTOCOL.md` | #257 |
| `docs/requirements/rules-twins/_common2.md` | New | Same for `swarm/roles/_common.md` | #257 |
| `docs/requirements/rules-twins/ba2.md` | New | Same for `swarm/roles/ba.md` | #257 |
| `docs/requirements/rules-twins/dev1_2.md` | New | Same for `swarm/roles/dev1.md` | #257 |
| `docs/requirements/rules-twins/dev2_2.md` | New | Same for `swarm/roles/dev2.md` | #257 |
| `docs/requirements/rules-twins/infra2.md` | New | Same for `swarm/roles/infra.md` | #257 |
| `docs/requirements/rules-twins/supervisor2.md` | New | Same for `swarm/roles/supervisor.md` | #257 |
| `docs/requirements/rules-twins/uat2.md` | New | Same for `swarm/roles/uat.md` | #257 |
| `docs/requirements/ba-session-change-log-20261009.md` | New | This list | this PR |

## Non-repository changes

| Item | Change | Why |
|------|--------|-----|
| GitHub issue #241 | Created (`role:dev status:ready`) | FR-BB58 AC-4 gap G2: code-specific download error toast and SessionHistoryTable test |
| `swarm/state/ba.json` (git-ignored) | Checkpoint heartbeat written each tick | PROTOCOL section 11 |

## Not changed (read only)
`CLAUDE.md`, `swarm/PROTOCOL.md`, `swarm/roles/*`, `*.settings.json`, `.claude/commands/*`. FR-BB116 spec reviewed (#42), no edit needed.
