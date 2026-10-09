# Code Review: ISS-148 (swarm checkpoints + heartbeats)

Verdict: PASS (no Critical/High; 1 Medium, 3 Low)

Scope: swarm/PROTOCOL.md (section 11), swarm/roles/_common.md (rule 14), swarm/roles/supervisor.md (heartbeat rule), swarm/bin/checkpoint.sh, swarm/state/checkpoint.example.json, .gitattributes, docs/issue-reports/ISS-148-checkpoints-heartbeats.md.

## Verified
- Numbering: new section 11 follows 10; _common.md rule 14 follows 13; references to "section 6" (escalation) and the supervisor stall step are correct.
- Consistency: N = 45 min matches the existing stall rule; GitHub comments remain source of truth; "heartbeat is not sent as messages" (section 3) still holds (file-based).
- No permission laundering: nothing changes permission mode, reserved-user actions, or merge/lock rules; the script only writes inside the state dir.
- Git ignore: `swarm/state/*.json` ignored, `*.example.json` allowed, so checkpoint.example.json is committed and role files are not.
- Script (tested only with SWARM_STATE_DIR on a fresh temp dir): `bash -n` OK; set -eu; arg count check; role whitelist (blocks `a/b` path traversal); issue digits/null; status whitelist; args never eval'd (`$(echo hi)` and backticks stored literally); quotes/backslash/newline/tab escaped and output parsed by Python json; atomic temp file in same dir + mv, trap cleanup; missing dir returns rc 1; exec bit 100755 and `.gitattributes` forces LF for *.sh.

## Findings

### Medium
- M1: esc() handles only CR/LF/TAB. Other control chars (0x00-0x1F, e.g. \x01) produce invalid JSON (reproduced: json.load fails "Invalid control character"). A corrupt file would be misread by the Supervisor. Fix: strip/replace remaining control chars (e.g. `s=$(printf '%s' "$s" | tr '\000-\037' ' ')`, careful that `$(...)` strips trailing newlines, acceptable here).

### Low
- L1: Hardcoded default `/c/Users/tvolo/...` is Git-Bash-on-Windows only; fails with a clear "state dir not found" elsewhere. Acceptable given PROTOCOL says so; consider deriving from `git rev-parse --git-common-dir`.
- L2: `retro.example.json` does not list `stale_heartbeat_min`; the setting is only discoverable from PROTOCOL. Also the section 9 durable-state list (PROTOCOL line 86) omits the new `<role>.json` files.
- L3: Role list includes `supervisor`, but no rule says who checks the Supervisor's own heartbeat (only the user/reports). Also `mktemp` yields mode 0600 files that `mv` keeps (fine on Windows, may matter if readers differ in user).

## Not tested
Concurrent writers (each role writes only its own file by rule) and non-Windows hosts.
