# Reusing the swarm in another project

Copy `swarm/`, `.github/ISSUE_TEMPLATE/swarm-task.md`, and run the label creation snippet from README (labels listed in PROTOCOL section 4).

| File | Kind | Change needed |
|------|------|---------------|
| `swarm/PROTOCOL.md` | generic (sections 3, 4, 6, 7, 9) + project-specific (section 1 table, 5 locks, 8 infra) | edit names, lock list, infra scope |
| `swarm/RETRO.md` | generic | none |
| `swarm/roles/_common.md` | generic | project name/path line, commit trailer |
| `swarm/roles/supervisor.md` | generic | worker names, backlog source |
| `swarm/roles/dev1.md`, `dev2.md` | mostly generic | pipeline prompt names under `.claude/commands/`, test commands |
| `swarm/roles/ba.md`, `uat.md` | project-specific | doc paths, test commands (`npm run test:e2e:live`), stack health URL |
| `swarm/roles/infra.md` | project-specific | infra repo, resources |
| `swarm/roles/*.settings.json` | project-specific | path deny lists, migration range, infra patterns |
| `swarm/up.ps1`, `down.ps1` | generic launcher | `$def` table (names, cwd, prompts), repo/infra paths |
| `swarm/state/*.example.json` | generic | none |
| `.gitignore` additions | generic | `.claude/worktrees/`, `swarm/state/*.json`, `swarm/locks/` |
| `.github/ISSUE_TEMPLATE/swarm-task.md` | generic | none |

Prerequisites in the target project: `.claude/commands/*.md` pipelines per role, a `gh`-authenticated repo, `docs/requirements/requirements-backlog.md` (anti-idle source for BA).
