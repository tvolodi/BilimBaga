# ISS-149 Swarm backup job, status page, restart drill

Issue: #149 (Refs, not Closes). Branch: swarm/149-swarm-backup-status.

## Changes
- `swarm/backup-state.ps1` (new): dated copy of state, roster and role settings to `%USERPROFILE%\.swarm-backups\bilimbaga`; prune only own `^\d{8}-\d{6}$` directories beyond newest 20 (after a successful copy, skipping links); text snapshot pushed to branch `swarm-state` via a temp `git init` repo and a plain push (working checkout untouched, failure only warns).
- `swarm/status.ps1` (new): roster, live sessions (recognised-schema fail-safe), checkpoints with heartbeat age, issue queue counts.
- `swarm/lib.ps1`: `Remove-Secrets`, `Get-SwarmIssues`, `Get-QueueCounts`, `Get-Checkpoint`.
- `swarm/ensure-up.ps1`: hourly `Invoke-Backup` (skipped under -WhatIf/stub); never affects the liveness result.
- `swarm/README.md`: backup/status docs and the restart drill (not executed).
- `swarm/tests/test-backup-status.ps1`: dry-run tests.

## Safety
Tests use temp BackupRoot, temp state dir, stubs and a temp bare repo; nothing was pushed to GitHub, nothing created under `.swarm-backups`, no session or task touched.

## Root cause / note
Native git stderr under `$ErrorActionPreference='Stop'` in Windows PowerShell 5.1 aborts the push block; the block runs with `Continue` and checks exit codes explicitly.
