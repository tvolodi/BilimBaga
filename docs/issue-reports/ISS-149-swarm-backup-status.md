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

## Supervisor review changes
- Blocking fix: `Invoke-Backup` no longer runs inside the ensure-up mutex. It runs after `ReleaseMutex` via `Invoke-BoundedBackup` (`swarm/lib.ps1`): child process, 120 s hard timeout, whole process tree killed with `taskkill /T /F`. The throttle marker `swarm/state/backup.last.json` is written before the attempt, so a timeout retries at most hourly. The unrecognised-live-list path now also backs up after releasing the mutex, then exits 1.
- `backup-state.ps1`: every git call runs non-interactively (`GIT_TERMINAL_PROMPT=0`, `GCM_INTERACTIVE=never`, `GIT_ASKPASS`/`SSH_ASKPASS` removed, `core.hooksPath=NUL`, `commit.gpgsign=false`, `credential.interactive=never`).
- Snapshot dedupe: the fetched previous `swarm-snapshot.txt` is compared with the new one minus the timestamp line; unchanged means no commit and no push.
- Privacy note: the repo is PUBLIC, so branch `swarm-state` is public. Data is limited to issue titles (already public) plus roster names; session ids and token-like strings are excluded (`Remove-Secrets` scrub kept).
- Tests added: unchanged snapshot skipped, changed snapshot pushed, hung stub git killed by the timeout (runner returns, children dead), marker written, no retry within the hour, retry after.
