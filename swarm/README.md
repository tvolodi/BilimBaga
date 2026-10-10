# BilimBaga agent swarm

Six LIVE interactive Claude Code sessions that run a perpetual cycle:
**system test (E2E/UAT) -> register issues in GitHub -> resolve -> test -> new issues -> ...** without waiting for the user.

| Role | Session | Purpose |
|------|---------|---------|
| Supervisor | `bb-supervisor` | live manager (`/loop`): reconciles GitHub, dispatches, detects stalls, generates work (anti-idle), runs retrospectives. Never edits code. |
| Dev1, Dev2 | `bb-dev1`, `bb-dev2` | implement/fix in separate git worktrees (`.claude/worktrees/dev1|dev2`) via the existing `requirement-implementation` / `issue-resolution` pipelines |
| BA | `bb-ba` | requirements + UAT scenarios (`business-analyst`, `requirement-development/validation`) |
| UAT | `bb-uat` | owns the live stack, runs UAT/E2E, files issues (`uat-runner`, `e2e-repair`) |
| Infra | `ai-dala-infra-fc` (reused) or `bb-infra` | QA environment (bilimbaga-qa, proposed) in the sibling `ai-dala-infra` repo, BilimBaga resources only; bilimbaga-test is frozen (customer demo) |

## Architecture
- **Transport**: native cross-session messaging (`ListAgents`, `SendMessage`); addresses and message format in `PROTOCOL.md`. Scripts can list live sessions with `claude agents --json`.
- **Durable state**: GitHub issues + labels (state machine in PROTOCOL.md), `docs/handoffs/`, `swarm/state/*.json`. A crashed session loses nothing: it restarts, reads its labeled queue and resumes.
- **Anti-idle**: every worker runs a `/loop` tick that polls its label queue and falls back to default work; the Supervisor additionally generates work for any role with an empty queue.
- **One permission mode** (`bypassPermissions`) for all sessions so cross-session messages are never held for approval.
- **Audit**: `RETRO.md`, retrospective every ~15 closed issues, output in `docs/retrospectives/`.

## Start / stop
```
powershell -File swarm\up.ps1            # all roles, supervisor last; running sessions are skipped
powershell -File swarm\up.ps1 -Roles dev1,uat
powershell -File swarm\down.ps1          # stops bb-* sessions (add -IncludeInfra for the infra session)
```
`up.ps1` creates nothing but tabs (Windows Terminal window `bilimbaga-swarm`, or separate windows). Dev worktrees must exist: `git worktree add --detach .claude/worktrees/dev1 origin/main` (and dev2), and be fast-forwarded to `origin/main` (`git -C .claude/worktrees/dev1 switch --detach origin/main`).

## Persistence and auto-restart (the PC stays on)
- **`roster.json`** (committed) is the single manifest: role key, `-n` name, cwd, settings path, prompt, resume prompt, model, permission mode. `up.ps1`, `ensure-up.ps1`, `down.ps1` and `bin/checkpoint.sh` all read it; nothing is hardcoded. Every prompt starts with "re-read your checkpoint file and your issue queue before acting".
- **Session ids** are recorded by `up.ps1` (and refreshed from `claude agents --json` when it exposes them) in `swarm/state/sessions.json` (git-ignored). Relaunch = `claude --resume <id> -n <name> --settings <role settings> ...`; if the id is gone the tab falls back to a fresh session with a new `--session-id`. `up.ps1 -Fresh` ignores saved ids.
- **`ensure-up.ps1`**: a dumb liveness check. Reads `claude agents --json`, relaunches any missing `bb-*` role resumed in a Windows Terminal tab with the same env cleaning as `up.ps1`. It never touches `ai-dala-infra*` (and starts `bb-infra` only if it was launched before and no `ai-dala-infra*` session is live). If `claude agents --json` fails it does nothing. It also watches `swarm/state/supervisor.json`: heartbeat older than 45 min while the session is live -> nudge (`supervisor.nudge.json`), 10 min later -> stop and relaunch it resumed.
- **Scheduled Task** `BilimBaga-Swarm-EnsureUp` (user level, interactive, no elevation; at logon, then every 5 min). Preview, then install it yourself (the swarm never registers it):
```
powershell -File swarm\install-ensure-task.ps1 -WhatIf   # preview
powershell -File swarm\install-ensure-task.ps1           # register (once, by the user)
powershell -File swarm\down.ps1                          # stops sessions AND unregisters the task (-KeepTask to keep it)
powershell -File swarm\install-ensure-task.ps1 -Uninstall
```
  The task points at the main checkout's `swarm\ensure-up.ps1`, so merge to `main` and update the main checkout first.
- **Restart rule**: a restarted worker re-reads its checkpoint file, its label queue and `docs/handoffs/` before acting (PROTOCOL section 12).
- **Dry runs**: `up.ps1`, `ensure-up.ps1`, `down.ps1` and the installer all support `-WhatIf`; `-AgentsJsonFile <file>` replaces `claude agents --json` and `SWARM_STATE_DIR` redirects state. Tests: `powershell -File swarm\tests\test-ensure-up.ps1` and `bash swarm/tests/test-checkpoint-roster.sh` (neither launches, stops or registers anything).

## Backup, status page and restart drill (#149)
- **`backup-state.ps1`** (called hourly from `ensure-up.ps1`, throttled by `-MinIntervalMin 60`, skipped in dry runs): copies `swarm/state/*.json`, `roster.json` and `roles/*.settings.json` to `%USERPROFILE%\.swarm-backups\bilimbaga\yyyyMMdd-HHmmss` (outside the repo) and prunes ONLY its own folders matching `^\d{8}-\d{6}$` beyond the newest 20. Options: `-WhatIf`, `-BackupRoot`, `-Keep`, `-QueueJsonFile`, `-MinIntervalMin`, `-NowUtc`. `-PushRemote` and `-NoPush` exist for the tests only.
  - **Publishing is OFF (#284).** This repo is PUBLIC, so nothing swarm-related is pushed by default. The old `swarm-state` snapshot push is no longer run: it needs an explicit `-PushRemote <url>` (tests use a local bare repo), and it never falls back to `origin`. Backups and snapshots stay local.
  - **Gitignored:** `swarm/state/*` (except the `*.example.json` files), `swarm/locks/`, `swarm/backups/`, `swarm-snapshot.txt` and `.claude/worktrees/`. `swarm/tests/test-backup-status.ps1` checks these with `git check-ignore`.
  - **Agent directories** get their files directly from the main checkout, not from a branch: the role settings are passed by path (`roles/<role>.settings.json`, read by `up.ps1`), and checkpoint and tick writes go to the main checkout's `swarm/state` (`bin/checkpoint.sh`, `bin/tick-log.sh`). The test checks the settings files exist.
  - **Bounded**: `ensure-up.ps1` runs the backup AFTER releasing its mutex, in a child process with a 120 s hard timeout (process tree killed), so a hung git/gh call cannot block the watchdog. The marker `swarm/state/backup.last.json` is written before each attempt, so even a timed-out backup is retried at most hourly. All git calls run non-interactively (`GIT_TERMINAL_PROMPT=0`, `GCM_INTERACTIVE=never`, no hooks, no gpg signing).
- **`status.ps1`**: one page with roster vs live sessions (`claude agents --json`, same recognised-schema fail-safe: an unrecognised list shows "unknown", never "down"), each worker's checkpoint with heartbeat age (STALE past `stale_supervisor_min`) and open issue counts by `status:*` label.
- Tests (temp dirs, stub files and a temp bare repo only): `powershell -File swarm\tests\test-backup-status.ps1`.

### Restart drill (documented, to be run by the user once the scheduled task is registered)
Proves that a killed worker comes back with its context. NOT executed by the swarm: it needs the `BilimBaga-Swarm-EnsureUp` task registered by the user.
1. `powershell -File swarm\status.ps1`: note `bb-dev2` LIVE and its checkpoint (issue, step, next action).
2. Kill that one worker: close its Windows Terminal tab or `Stop-Process -Id <pid>` (pid from `claude agents --json`). Never kill several at once.
3. Within about 5 min (task interval; the relaunch cooldown is 8 min) `ensure-up.ps1` relaunches it with `claude --resume <saved id> -n bb-dev2 --settings ...`. Watch with `status.ps1`.
4. Confirm: the session is LIVE again under the same name, the first thing it does is re-read `swarm/state/dev2.json`, its label queue and `docs/handoffs/` (restart rule), and it continues the same issue/step. If the saved id was gone it starts fresh with the same prompt and still re-reads the checkpoint.
5. Pass criteria: restored within 10 min, same role name and settings, no duplicate session, other workers and `ai-dala-infra*` untouched. Record the outcome in the next retrospective.

## Troubleshooting
- *A worker does not react*: `claude agents --json` shows status; idle sessions may not wake on a message, the worker's `/loop` tick picks work up within 5-10 min. If the loop died, send any message or restart that role with `up.ps1 -Roles <role>`.
- *Messages not delivered / held*: permission-mode mismatch. All sessions must be `bypassPermissions`. Restart the odd one.
- *Names*: Claude Code may append a suffix to the session name (like `bilimbaga-e0`); always resolve peers from `ListAgents` by prefix (`bb-dev1`, `bb-uat` ...).
- *Stale lock*: delete `swarm/locks/*.lock` older than 60 min (Supervisor does it).
- *Stack down*: UAT runs `make dev`; check `curl localhost:8080/api/v1/health`.
- *Infra blocked*: ai-dala-infra `shared/agent-team.md` covers only Letflow; BilimBaga tasks follow the normal human approval gate there (only the user can extend it). Infra reports `status:blocked`.
- *Reset state*: delete `swarm/state/*.json` (not `.example.json`) and relaunch Supervisor.

See `TEMPLATE.md` for reuse in other projects.
