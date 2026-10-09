# Files that regulate agent permissions, blocks and restrictions (2026-10-09)

Index only: nothing here was changed. Order: enforced rules first, then rules that agents follow by instruction.

## 1. Enforced (settings with allow/deny rules)

| File | Scope | What it regulates |
|------|-------|-------------------|
| `swarm/roles/supervisor.settings.json` | Supervisor session | Allowed/denied tools and paths; writes limited to `swarm/state/` and `docs/retrospectives/` |
| `swarm/roles/dev1.settings.json` | Dev1 session | Dev permissions and deny list |
| `swarm/roles/dev2.settings.json` | Dev2 session | Dev permissions and deny list |
| `swarm/roles/ba.settings.json` | BA session | Writes only to `docs/requirements`, `docs/uat-scenarios`, `docs/handoffs`, `docs/issue-reports`; denies code, config, `CLAUDE.md`, `Makefile`, `.env`, migrations 001-030, `swarm/roles/**`, `swarm/PROTOCOL.md`, `swarm/up.ps1`, `swarm/down.ps1`, `swarm/state/**`, anything under `.claude/**`; denies `go`, `npm`, `make`, `docker`, force-push, push to main, `gh repo delete/edit`, `gh secret/auth/ssh-key/gpg-key`, `winget`/`choco`/`scoop`, global npm installs, `rm -rf` of `/` or `~` |
| `swarm/roles/uat.settings.json` | UAT session | Tester permissions; no source edits |
| `swarm/roles/infra.settings.json` | Infra session | Deny-by-default scope (BilimBaga QA only) |
| `C:\Users\tvolo\.claude\settings.json` | All sessions on this machine (user global) | `defaultMode: bypassPermissions`, `skipDangerousModePermissionPrompt: true`, a short Bash allow list. No deny rules |

Note: every swarm role runs in `bypassPermissions`, so the deny lists in the role settings files are the only technical guardrail (PROTOCOL section 2). There is no project-level `.claude/settings*.json`.

## 2. Instruction files (followed by agents, not technically enforced)

| File | Restrictions it contains |
|------|--------------------------|
| `CLAUDE.md` (project) | Orchestrator never edits files or runs commands itself; all changes via pipelines; working files only under `docs/`; no migration edits; no secrets in source; forbidden output patterns |
| `C:\Users\tvolo\.claude\CLAUDE.md` (user global) | Reserved for the user: deleting directories outside the project, system-wide installs, acting on the open internet, force-pushing over others' commits, external communication; destroying git history only when a backup survives |
| `swarm/PROTOCOL.md` | Permission mode and no permission laundering (s2), worktree-only work and lock rules (s5), no force-push or history rewrite, no edits to existing migrations, frozen `bilimbaga-test.ai-dala.com` (s8), 3-attempt cap (s6) |
| `swarm/roles/_common.md` | Never ask the user, no permission laundering, repo root stays clean |
| `swarm/roles/ba.md`, `dev1.md`, `dev2.md`, `uat.md`, `infra.md`, `supervisor.md` | Per-role "never edit" statements, write scopes, stack/lock ownership, frozen-host rules (infra, uat), BA self-generated cap |
| `docs/requirements/DEC-001.Environments-production-class-demo-and-qa.md` | Decision behind the frozen demo environment and QA-only test traffic |
| `.github/agents/*.agent.md`, `.github/agents/README.md` | Per-agent scope and boundaries |
| `.github/instructions/backend-conventions.instructions.md` | Backend convention rules |
| `.claude/commands/*.md` | Subagent pipeline prompts; each limits what that subagent may do |

## 3. Workflow-only twins (no restrictions)
`docs/requirements/rules-twins/` holds permission-free versions of the instruction files in section 2; see its README.

## 4. Not found
No `.claudeignore`, no project `.claude/settings.json` or `settings.local.json`. The `.vscode/settings.json` files are editor settings, not agent permissions.
