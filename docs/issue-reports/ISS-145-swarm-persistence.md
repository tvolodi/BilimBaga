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

## Supervisor safety conditions (pre-merge hardening)
1. Fail-safe live list: `ensure-up.ps1` now uses `Get-RecognisedLiveAgents` (swarm/lib.ps1). It proceeds only if `claude agents --json` is a JSON array with at least one entry whose `name` is a non-empty string. Command failure, empty output, `null`, non-JSON, an object, `[]`, scalars, entries with other field names or a non-string name all make the run exit 1 doing nothing: no relaunch, no stop, no nudge, no state writes. A valid recognised list that lacks a role still relaunches that role. Known consequence: if no session at all is live (`[]`, e.g. after a reboot) the watchdog does nothing; use `up.ps1` for a cold start.
2. Supervisor stop rules: stop happens only when the heartbeat is stale, a nudge exists, it is strictly older than `supervisor_grace_min` (10 min), AND the heartbeat re-read immediately before stopping is still stale and the session has a pid. Otherwise the nudge is cleared and nothing is stopped. A fresh heartbeat at any point clears the nudge.
3. Tests added (`swarm/tests/test-ensure-up.ps1`): W1-W6 (fresh, stale->nudge only, nudge <10 min, exactly 10 min, nudge >10 min -> stop planned under -WhatIf, refreshed heartbeat -> no stop and nudge cleared, stale again -> nudge first), C1/C2 (8-min relaunch cooldown), and 9 fail-safe cases each run under -WhatIf and as a real run (safe: it exits before any action) asserting zero relaunch/stop/nudge.
4. Also fixed: PowerShell 5.1 `ConvertFrom-Json` returns a JSON array as one object when assigned; the list is now explicitly unrolled.

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
