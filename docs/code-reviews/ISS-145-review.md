# Code Review: ISS-145 swarm persistence

Reviewer: Code Reviewer subagent. Scope: uncommitted changes in worktree dev1 (roster.json, lib.ps1, up.ps1, ensure-up.ps1, down.ps1, install-ensure-task.ps1, bin/checkpoint.sh, bin/record-session.ps1, tests/*, README, PROTOCOL section 12, TEMPLATE, roles/supervisor.md).

## Verdict: FAIL

One High finding (resume fallback re-launches a fresh session whenever claude is killed). Everything else is Medium or Low. Tests: `bash swarm/tests/test-checkpoint-roster.sh` 3/3 PASS; `powershell swarm/tests/test-ensure-up.ps1` ALL PASSED (all runs were -WhatIf with a stubbed agents file; nothing launched, stopped or registered). The tests cannot see finding 1 because they only parse the generated command.

## Findings

1. **[High] swarm/lib.ps1:112 (also affects down.ps1:25 and ensure-up.ps1:82)** — Resume fallback fires on any non-zero exit, not only on a failed resume. The tab command is `claude --resume <id> ...; if ($LASTEXITCODE -ne 0) { <fresh launch> }` inside a `-NoExit` shell. `down.ps1` and the supervisor restart in `ensure-up.ps1` use `Stop-Process -Force`, which gives claude a non-zero exit code, so the surviving tab immediately starts a brand-new `--session-id` session with the same `-n` name. Consequences: `down.ps1` does not stop resumed roles (they come back fresh, history lost, id overwritten by record-session); the supervisor watchdog ends up with two `bb-supervisor` sessions (old tab's fresh one plus the resumed one) and the resumed id is clobbered. Ctrl+C/crash does the same. Fix: only fall back when the resume fails fast, e.g. record `$t0 = Get-Date` and fall back only if `$LASTEXITCODE -ne 0 -and ((Get-Date)-$t0).TotalSeconds -lt 20`; or better, check that the session transcript for `<id>` exists under `~/.claude/projects/*/<id>.jsonl` before choosing resume mode, and drop the in-tab fallback. Add a test that asserts the generated command contains the guard.

2. **[Medium] swarm/ensure-up.ps1:48-56** — No relaunch cooldown. The task fires every 5 min; a launched tab that is slow to register in `claude agents --json` (cold start, resume of a large session, machine just logged in) can be relaunched again on the next run, creating duplicates under the same `-n`. Fix: write `swarm/state/ensure-launch.<key>.json` with a UTC time on relaunch and skip that role for ~10 min.

3. **[Medium] swarm/lib.ps1:44-63 and ensure-up.ps1:50** — Liveness is name-only (`name -like "bb-x*"`), and the stubs in the tests carry only `name`/`pid`. If real `claude agents --json` also lists stopped/finished sessions or has a status field, dead roles will be seen as live and never relaunched (or the reverse). The real command was not run in this review (safety rule), so this is unverified. Fix: confirm the JSON schema once (status/pid fields) and filter on a live status plus a running pid (`Get-Process -Id`); add that to the stub tests.

4. **[Medium] swarm/lib.ps1:56-60** — `Get-AgentSessionId` falls back to the `id` property if it looks like a GUID. If `id` is an agent/task id rather than the conversation session id, `Update-SessionsFromLive` (called by up.ps1:28 and ensure-up.ps1:43) will overwrite a good saved session id with a wrong one, and later `--resume` would fail (then silently go fresh). Fix: accept only `sessionId`/`session_id`, or verify the schema first (see 3).

5. **[Medium] swarm/ensure-up.ps1:64-68** — Watchdog is blind when `swarm/state/supervisor.json` is missing, unreadable or lacks `last_tick_utc` (`$tick` null, no action). A supervisor that hangs before ever writing a heartbeat, or whose file was deleted, is never nudged or restarted. Fix: when the file is missing/unparsable and the session is live, treat age as time since the process start (or since the last relaunch marker, see 2), or at least log a warning.

6. **[Medium] swarm/ensure-up.ps1:79-85** — Relaunch after supervisor kill: `Stop-Process` is followed immediately by `Start-Resumed`, with no wait for the old process to exit, and `sessions` was read before the kill. Combined with finding 1 this duplicates; even after fixing 1, add `Wait-Process -Id $pid -Timeout 15` before relaunch. Also guard against a null/absent `pid` (`$supLive.pid`) before `Stop-Process -Id`.

7. **[Low] swarm/lib.ps1:75-83 and bin/record-session.ps1** — `Save-Session` is read-modify-write without a lock. up.ps1 saves fresh ids synchronously so it is safe today, but tab-side `record-session.ps1` calls (fallback path) can race with each other or with ensure-up and drop an update. Fix: use a named mutex around Save-Session.

8. **[Low] swarm/install-ensure-task.ps1:28** — Repetition is copied from a `-Once` trigger onto an `-AtLogOn` trigger. This works on current Windows but the duration is not set explicitly, so indefinite repetition depends on defaults. Fix: set `-RepetitionDuration ([TimeSpan]::MaxValue)` (or `New-TimeSpan -Days 9999`) explicitly and verify `Repetition.Duration` after install on the host. Also `-Uninstall` (line 18) throws if the task does not exist; add `-ErrorAction SilentlyContinue` or an existence check like down.ps1 has.

9. **[Low] swarm/down.ps1:8** — No `$ErrorActionPreference = 'Stop'` and `Get-LiveAgents` failure aborts after the task was already uninstalled, leaving sessions running with no watchdog (acceptable, but print a clear message). A running ensure-up instance could still relaunch roles between uninstall and stop; hold the same Global mutex in down.ps1 while stopping.

10. **[Low] swarm/bin/checkpoint.sh:47** — `dir=${SWARM_STATE_DIR:-$(resolve_dir)} || exit 1`: a failing `resolve_dir` inside the default expansion does not trigger `|| exit 1`; the script still exits 1 later via the "state dir not found:" check, but with an empty path in the message. Fix: call `resolve_dir` into a variable first, then apply the override. Also `sed` parsing of roster.json assumes one key per line (true today; `main_checkout` is on its own line).

11. **[Low] swarm/tests/test-ensure-up.ps1:103-106** — Test 9 is titled "missing live listing => do nothing" but actually tests an empty stub (relaunch expected). The real failure path (`claude agents --json` non-zero -> exit 1, nothing launched) is untested. Temp dir/env cleanup is skipped if a check throws (use try/finally). If a real ensure-up run holds the Global mutex, the ensure-up tests print "another run active" and fail spuriously; add a mutex-name override for tests.

12. **[Low] swarm/roster.json:2 / PROTOCOL.md section 12** — Docs state the fresh-fallback behaviour as designed ("fresh session only when no id is known or resume fails"); update after fixing 1 so it says resume failure is detected by session-file absence or fast exit.

## Requirement coverage

- up.ps1 records ids (git-ignored swarm/state/sessions.json), relaunch via `--resume` with same `-n` and per-role `--settings`: covered, except the over-eager fallback (finding 1).
- ensure-up.ps1 liveness-only, resumes missing bb-* roles with the same env cleaning, never touches ai-dala-infra*: covered (name filter `bb-*`, `reuse_live_prefix` skip, no Stop-Process on infra; down.ps1 only touches `ai-dala-infra*` with -IncludeInfra).
- Scheduled Task installer (logon + every 5 min), not registered by the swarm, uninstall in down.ps1: covered (task not registered by tests; down.ps1 unregisters unless -KeepTask).
- roster.json manifest used by up/ensure-up/down/checkpoint.sh: covered.
- Supervisor stale-heartbeat watchdog (nudge at 45 min, relaunch 10 min later): covered with the gaps in findings 5 and 6; supervisor.md reads the nudge marker.
- checkpoint.sh resolves main checkout from roster: covered and tested from a worktree.
- Restart-rule docs (README, PROTOCOL section 12, supervisor.md, TEMPLATE) and preamble in every roster prompt: covered.
- Secrets: none found; session ids and nudge files are git-ignored (`swarm/state/*.json`).

## To reach PASS

Apply finding 1 (and add a regression test for it). Findings 2-6 are strongly recommended in the same pass; 7-12 are optional.

---

# Cycle 2

## Verdict: PASS

Cycle-1 High (finding 1) and the strongly recommended Medium items are fixed. Tests re-run, all -WhatIf/stubbed (nothing launched, stopped or registered; no real `claude agents --json`): `bash swarm/tests/test-checkpoint-roster.sh` 3/3 PASS; `powershell swarm/tests/test-ensure-up.ps1` ALL PASSED, including the new regression check "fresh fallback only when resume dies within 20 s".

## Fix verification

| # | Cycle-1 finding | Status | Evidence |
|---|-----------------|--------|----------|
| 1 | High: fresh fallback on any non-zero exit | Fixed | lib.ps1:113 records `$t0` and falls back only if `$LASTEXITCODE -ne 0` and elapsed < 20 s. A later kill (down.ps1, supervisor restart) no longer spawns a new session. Regression test asserts the guard (test-ensure-up.ps1:92). Residual: a stop within 20 s of launch would still fall back (acceptable). |
| 2 | Medium: no relaunch cooldown | Fixed | ensure-up.ps1:29-42 persists `ensure-up.relaunch.json` per role, skips for 8 min, writes only when not -WhatIf. Not covered by a test (WhatIf writes nothing), see N1. |
| 3 | Medium: name-only liveness / schema unverified | Deferred | Still unverified by design (running real command forbidden). Low-risk: wrong schema degrades to "never relaunch" or "relaunch + cooldown", no duplicates storm now. |
| 4 | Medium: `id` fallback | Fixed | Get-AgentSessionId (lib.ps1:55-61) accepts only `sessionId`/`session_id` with GUID shape. |
| 5 | Medium: blind watchdog on bad heartbeat | Partly fixed | Unparsable supervisor.json uses file mtime (ensure-up.ps1:73). Missing file is still no action (N2, low). |
| 6 | Medium: restart without waiting | Fixed | ensure-up.ps1:89 guards null pid, `Wait-Process -Timeout 15`, then re-reads sessions before `Start-Resumed`. |
| 8 | Low: `-Uninstall` throws if absent | Fixed | install-ensure-task.ps1:18. Explicit repetition duration still not set (see N3). |
| 7, 9-12 | Low | Deferred | Accepted by author as low. |

## New / residual observations (all Low, non-blocking)

- N1: Cooldown logic has no automated test (needs a non-WhatIf path or a state-dir stub pre-seeded with `ensure-up.relaunch.json` plus -WhatIf; the read path is testable that way).
- N2: If `supervisor.json` is absent while bb-supervisor is live, the watchdog still does nothing.
- N3: install-ensure-task.ps1:29 relies on default indefinite repetition; verify `Repetition.Duration` once on the host after the user installs.
- N4: down.ps1 does not use `$ErrorActionPreference='Stop'`, and `Stop-Process -Id $a.pid` would error on a null pid from an unexpected schema.
- N5: In `-WhatIf`, cooldown skip messages are printed but a cooldown file is never created, so WhatIf previews always show a relaunch.

No source edited in this review.
