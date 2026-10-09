# Code Review: ISS-149 (swarm backup, status page, restart drill)

Reviewer: Code Reviewer subagent. Scope: swarm/backup-state.ps1, status.ps1, lib.ps1, ensure-up.ps1, README.md, tests/test-backup-status.ps1 (uncommitted).
Verified: `swarm/tests/test-backup-status.ps1` run (temp dirs only): ALL PASSED (28 checks). Also probed `Get-SwarmIssues` with `[]` and a one-element array on PS 5.1: counts 0 and 1, correct.

## Verdict: PASS (with 1 Medium and 4 Low findings, none blocking)

## Focus areas

- Deletion safety: OK. Prune only considers direct children of BackupRoot that match `^\d{8}-\d{6}$`, are real directories (reparse points excluded), and sit under the resolved root. It runs only after a successful copy, honours `-WhatIf`, and `Keep >= 1` is enforced. Test covers foreign files and near-miss names (`20261009-1200`, `2026100a-120000`, `notes-2026`). `-BackupRoot` pointed at an arbitrary dir is still limited to strict-pattern children.
- Secrets: OK. Local backup contains state files (including sessions.json with session ids) but stays under `%USERPROFILE%`. The pushed snapshot has only roster key/name, label counts and issue titles, passed through `Remove-Secrets` (token prefixes, key/value forms, 32+ char blobs). Test confirms ghp_ and `password=` are scrubbed and no session ids. Remote URL is never written into the snapshot or warnings.
- Push safety: OK. Throwaway `git init` in a temp dir, fetch depth 1, checkout, single-file commit, plain `push` (no force). A non-fast-forward just fails and warns. Working checkout is never touched. Workflows trigger only on `main`/PRs, so pushing `swarm-state` starts no CI.
- Fail-safe: OK. Push, gh and snapshot errors only warn; ensure-up ignores backup failure; backup is skipped for `-WhatIf` and stub runs; unrecognised live list still short-circuits relaunch logic (backup is then run, harmless).
- PS 5.1: OK (tests ran under powershell.exe 5.1). `Set-Content -Encoding UTF8` writes a BOM; harmless.

## Findings

1. **Medium** - `ensure-up.ps1` `Invoke-Backup` runs synchronously inside the mutex with no timeout. A hung `git fetch/push` (credential prompt, network stall) or `gh issue list` would hold `Global\bilimbaga-swarm-ensure-up` and silently stop all future watchdog runs, defeating the purpose of ensure-up. Suggest: set `GIT_TERMINAL_PROMPT=0` (and `GCM_INTERACTIVE=never`) in backup-state.ps1, and run backup via `Start-Process`/job with a wait timeout (e.g. 120 s, then kill), or run it after releasing the mutex.
2. **Low** - A partially-copied dated folder (copy failed midway) remains, counts as "newest" for the `-MinIntervalMin` throttle (no retry for an hour) and counts toward `Keep`. Suggest removing the half-made `$dest` in the catch (it is own, strict-pattern, just created).
3. **Low** - Snapshot embeds a timestamp, so one commit per hour is pushed even when nothing changed (about 24 commits/day on `swarm-state`). Consider skipping the commit when content minus timestamp is unchanged.
4. **Low** - Git in the temp repo still honours global config (e.g. a global `core.hooksPath` or commit-signing). Pass `-c commit.gpgsign=false -c core.hooksPath=NUL` for determinism.
5. **Low** - `gh` output with non-ASCII titles (Cyrillic) may be mangled via the console codepage in PS 5.1; cosmetic only (set `[Console]::OutputEncoding` UTF-8 around the call).

## Notes
- Over-redaction (`token`/`secret` words, any 32+ char run) can alter some titles; acceptable per design comment.
- README restart drill is correctly documented as user-run and not executed by the swarm.
