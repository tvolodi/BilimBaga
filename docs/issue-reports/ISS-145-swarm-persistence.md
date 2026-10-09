---
id: ISS-145
title: Swarm does not survive a session crash or reboot (no session-id persistence, no auto-restart)
status: resolved
severity: medium
layer: config
module: swarm
tags: [swarm, up.ps1, ensure-up.ps1, roster.json, --resume, Scheduled Task]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-148, ISS-149]
regression_test: swarm/tests/test-ensure-up.ps1
---

## Symptom
GitHub issue #145: after a crashed claude session or a reboot the swarm sessions stay dead; `up.ps1` only starts fresh conversations from a hardcoded role table and nothing restarts missing roles.

## Root Cause
Roles, prompts and paths were hardcoded in `up.ps1`; session ids were never recorded, so a relaunch could not `--resume`; no watchdog existed; `checkpoint.sh` defaulted to a host-specific path.

## Fix Applied
- `swarm/roster.json` (committed manifest) + `swarm/lib.ps1` (roster, tokens, session state, launch spec).
- `up.ps1` reads the roster, records session ids in git-ignored `swarm/state/sessions.json`, relaunches with `--resume <id>` (same `-n`, same settings; fresh fallback), supports `-WhatIf`, `-Fresh`, `-AgentsJsonFile`.
- `ensure-up.ps1`: liveness check, resumed relaunch of missing `bb-*` roles, never touches `ai-dala-infra*`, supervisor stale-heartbeat watchdog (nudge at 45 min, restart resumed after 10 more).
- `install-ensure-task.ps1` registers the user-level task (logon + every 5 min); `down.ps1` unregisters it. Neither was run for real.
- `bin/checkpoint.sh` resolves the main checkout from the roster instead of a host path.
- Docs: `swarm/README.md`, `swarm/PROTOCOL.md` section 12 (restarted worker re-reads checkpoint, label queue, `docs/handoffs/`), `roles/supervisor.md`, `TEMPLATE.md`.

## Files Changed
| File | Change |
|------|--------|
| swarm/roster.json, swarm/lib.ps1, swarm/bin/record-session.ps1 | new |
| swarm/up.ps1, swarm/down.ps1, swarm/bin/checkpoint.sh | changed |
| swarm/ensure-up.ps1, swarm/install-ensure-task.ps1 | new |
| swarm/tests/test-ensure-up.ps1, swarm/tests/test-checkpoint-roster.sh | new tests |
| swarm/state/sessions.example.json, README.md, PROTOCOL.md, TEMPLATE.md, roles/supervisor.md | docs/examples |

## Regression Test
`swarm/tests/test-ensure-up.ps1` (dry run only: -WhatIf, stubbed `claude agents --json`, temp `SWARM_STATE_DIR`) and `swarm/tests/test-checkpoint-roster.sh` (temp git repo + worktree).

## Resolution Results
- Tests: all PASS (see PR); parser syntax check of every script clean; generated launch commands parse as valid PowerShell.
- Migration applied: no
- Build clean: n/a (PowerShell/bash only)
- Scheduled Task registration: NOT performed (user must run `swarm\install-ensure-task.ps1`).

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
