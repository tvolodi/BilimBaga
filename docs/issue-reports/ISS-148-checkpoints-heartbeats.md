# ISS-148 Swarm checkpoints and heartbeats

Issue: #148 (origin #145). Docs/rules only plus one helper script.

## Changes
- `swarm/roles/_common.md`: rule 14 (checkpoint file + issue-comment mirror, heartbeat).
- `swarm/PROTOCOL.md`: new section 11 "Checkpoints and heartbeats" (file location in main checkout, absolute path, JSON schema, atomic write, issue mirror as source of truth, 45 min stale rule). Existing sections not renumbered.
- `swarm/roles/supervisor.md`: heartbeat rule under Tick step 3 (stall detection).
- `swarm/bin/checkpoint.sh`: atomic writer (mktemp in target dir + mv), bash/date/printf only, role/issue/status validation, JSON escaping.
- `swarm/state/checkpoint.example.json`: committed example (the state dir already commits `*.example.json`).
- `.gitattributes`: `*.sh text eol=lf` so the script keeps LF under core.autocrlf=true.

## Verification
`bash -n` OK; functional tests wrote to a temp dir via `SWARM_STATE_DIR` (quotes/backslashes escaped, idle/null issue, invalid role/issue rejected with exit 2, no temp files left). Real state files untouched.
