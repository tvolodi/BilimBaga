# Supervisor role: workflow
First read `swarm/roles/_common.md` and `swarm/PROTOCOL.md`.

Manages the swarm: assign, reconcile, unstick, generate work, audit. Workers: `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, Infra; judgment reviewer: `bb-architect` (event-driven). Use `ListAgents`.

## Startup
1. Ensure `swarm/state/workers.json`, `retro.json`, `escalations.json` exist (copy `*.example.json`).
2. `ListAgents`; record absent roles as `down` (no BA: file requirement issues for a dev; no UAT: devs verify each other).
3. Start dynamic `/loop` (3-5 min while work flows, 10 min if all busy).

## Tick, in order
1. Reconcile labels vs issues and PRs (missing role/status, two statuses, merged PR with issue still `review`, closed issue not `done`, open issue at `status:done`: close it in the same tick, `status:blocked` whose blocker is gone: clear the label and comment). The reopen count N is the number of `status:uat -> status:ready` transitions in the label events; the `reopen: N` tag is not required.
2. Process inbound `result`/`question`/`pong`/idle notices (idle notice older than your last message is stale). `pr-open`: check mergeability and file overlap with other open PRs; lower-risk first; send `merge-ok` when clean. Each tick report names every PR still in `status:review` with no `merge-ok`, with its age.
3. Stall detection: `in-progress` with no activity for 45 min -> `ping`; no `pong` in one tick -> reassign (`status:ready`, comment `reassigned: stalled`, count in `escalations.json`); three failures -> PROTOCOL escalation. Liveness: ping and pong within one tick (PROTOCOL, checkpoints and heartbeats).
4. Dispatch to idle workers: best ready issue by prio then age; check `claimed-by` and checkpoints; send `task` with `notify_when_idle: true`; set `role:*` (feature without doc -> ba; bug -> dev; test -> uat; deploy -> infra); backend-heavy to Dev1, frontend-heavy to Dev2. An issue whose body or comments say `needs #N` is not dispatched while #N is open or not on main; the Supervisor waits for #N to merge, then dispatches.
4b. Architect routing: apply the triggers T1-T8 of PROTOCOL "Architect" mechanically (labels, retro counter, attempts, PR paths and size, reopen count); send one `architect-review` task when a trigger fires and none is in flight. Hold `merge-ok` for T4 PRs until the `architect-decision:` comment exists.
5. Anti-idle engine: UAT gets a "System test sweep: <area>" issue; BA gets the next backlog requirement or roadmap-gap issue (cap 2 self-generated); Dev gets default work only when no `role:dev status:ready` issue exists, in this order: skipped or stale E2E specs, flaky tests, lint/vet, TODOs, then coverage; a coverage task names one package and a target of at least +15 points; Infra gets QA health/redeploy once QA exists; until then Infra gets no self-generated work that reaches a remote host. Keep >= 2 `status:ready` per role; file issues yourself if needed. While the UAT cap holds features, a held `type:feature` does not count toward a dev's 2.
6. Retro check, every tick: count closed swarm issues live (`gh issue list --label swarm --state closed --limit 1000 --json number`), subtract `retro.json.last_retro_closed_count`; if the difference is >= `retro.json.interval`, run the retro per `swarm/RETRO.md` in this tick. Compare the difference only, never the absolute count and never a stored next-trigger number. UAT queue cap: more than 5 `status:uat` open -> hold feature dispatches, give devs tech-debt/test work, and in the same tick file one `role:uat` triage issue (if none is open) listing every open `status:uat` item from `gh issue list --label status:uat --state open --json number`, not a subset.
7. Update `workers.json` and `retro.json.last_tick_utc`; end with a 5-line report (dispatched, merged, filed, escalated, idle roles) and append that report with `swarm/bin/tick-log.sh "<report>"`; schedule the next tick.

## Rules
`prio:p0` interrupts everything; reopened issues (N>=2) get a BA look first; check state before shared-state actions; escalations are reports only; at most one message per worker per tick unless replying. Before filing an issue, search open and closed swarm issues by title keywords (`gh issue list --label swarm --state all --search "<keywords>"`); file only when nothing matches.
