# Retrospective / audit procedure

Owner: Supervisor. Trigger: `closed_count - retro.json.last_retro_closed_count >= interval` (default 15, configurable 10-20 via `swarm/state/retro.json.interval`), checked each tick. No retro starts less than 4 h after the previous cut unless step 8 applies; the scope is then every closure since the cut, not only the first `interval`, and step 7 stores the count at the pull. Always compare the difference, never an absolute count (retro-003). Counter state: `swarm/state/retro.json`:

```json
{"last_retro_closed_count": 0, "retro_number": 0, "interval": 15, "last_tick_utc": null}
```

## Procedure
1. `gh issue list --label swarm --state closed --json number,title,createdAt,closedAt,labels,comments --limit 1000`; take the first `interval` closures after the last retro by `closedAt` (ranks `last_retro_closed_count`+1 to `last_retro_closed_count`+`interval`). Period names the rank range and the first and last issue.
2. Metrics (compute with `gh` + `jq`, put in a table):
   - cycle time per issue (created -> closed), median and p90, split by type
   - time in each status (from label events: `gh api repos/{owner}/{repo}/issues/N/events`)
   - reopen rate (issues with `reopen:` comments / total)
   - failures per role (`failed` results, reassignments, `attempts` >= 2)
   - idle time per role (ticks where the role had no ready work or was reported idle; from `workers.json` history and Supervisor tick reports)
   - escalations (`escalations.json`) and `status:blocked` count and duration
   - duplicate or invalid issues, PR conflicts, lock contention incidents
2b. Send the metrics table to `bb-architect` (`architect-review`, trigger T2). The architect writes steps 3-4 (narrative, root causes, decisions, plus an assessment of the team's way of working and model fit) and returns the draft; you then continue at step 5.
3. Narrative: what went well, what went badly, root causes (protocol gap, role-prompt gap, tooling, environment).
4. Decisions: a numbered list of concrete changes to `swarm/roles/*` and `swarm/PROTOCOL.md` (exact text changes), each with the metric it should move. The Supervisor approves them itself (autonomy), except anything that touches reserved user decisions, which is only listed as a recommendation.
5. Write `docs/retrospectives/retro-NNN.md` (NNN = `retro_number + 1`, zero-padded to 3) on a branch `swarm/retro-NNN`, open a PR. Template:
   `# Retro NNN`, `Period`, `Metrics`, `Went well`, `Went badly`, `Decisions`, `Applied changes (PR links)`, `Open recommendations for the user`.
6. Apply improvements: create a `type:tech-debt` issue "retro-NNN: apply swarm improvements" with the exact diffs, label `role:dev`, assign it via the normal dispatch (action `retro-apply`). The Dev edits `swarm/roles/*` / `PROTOCOL.md` via a PR; Supervisor merges after the check. Role changes take effect when a session restarts or re-reads the file: the Supervisor sends each affected worker a `reload-role` message (first line self-contained) after the merge. The Supervisor then re-reads `swarm/roles/supervisor.md`, `_common.md` and `PROTOCOL.md` from `origin/main` itself, and writes one line `<UTC> reload retro-NNN` to `swarm/state/ticks.log`.
7. Update `retro.json` (`last_retro_closed_count` = the cut from step 1, that is last + interval, not the live count; `retro_number` + 1). Closures after the cut are the next retro's scope.. Never skip this, otherwise the retro re-triggers every tick.
8. Also run a retro on demand when 3 escalations accumulate or an incident (data loss, repeated merge conflict) happens.
