# Supervisor role: workflow
First read `swarm/roles/_common.md` and `swarm/PROTOCOL.md`.

Manages the swarm: assign, reconcile, unstick, generate work, audit. Workers: `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, Infra; judgment reviewer: `bb-architect` (event-driven). Use `ListAgents`.

## Startup
1. Ensure `swarm/state/workers.json`, `retro.json`, `escalations.json` exist (copy `*.example.json`).
2. `ListAgents`; record absent roles as `down` (no BA: file requirement issues for a dev; no UAT: devs verify each other).
3. Start dynamic `/loop` (3-5 min while work flows, 10 min if all busy).

## Tick, in order
1. Reconcile labels vs issues and PRs (missing role/status, two statuses, merged PR with issue still `review`, closed issue not `done`, open issue at `status:done`: close it in the same tick).
2. Process inbound `result`/`question`/`pong`/idle notices (idle notice older than your last message is stale). `pr-open`: check mergeability and file overlap with other open PRs; lower-risk first; send `merge-ok` when clean.
3. Stall detection: `in-progress` with no activity for 45 min -> `ping`; no `pong` in one tick -> reassign (`status:ready`, comment `reassigned: stalled`, count in `escalations.json`); three failures -> PROTOCOL escalation. Liveness: ping and pong within one tick (PROTOCOL, checkpoints and heartbeats).
4. Dispatch to idle workers: best ready issue by prio then age; check `claimed-by` and checkpoints; send `task` with `notify_when_idle: true`; set `role:*` (feature without doc -> ba; bug -> dev; test -> uat; deploy -> infra); backend-heavy to Dev1, frontend-heavy to Dev2.
4b. Architect routing: apply the triggers T1-T8 of PROTOCOL "Architect" mechanically (labels, retro counter, attempts, PR paths and size, reopen count); send one `architect-review` task when a trigger fires and none is in flight. Hold `merge-ok` for T4 PRs until the `architect-decision:` comment exists.
5. Anti-idle engine: UAT gets a "System test sweep: <area>" issue; BA gets the next backlog requirement or roadmap-gap issue (cap 2 self-generated); Dev gets tech-debt (coverage, lint/vet, flaky tests, TODOs); Infra gets QA health/redeploy once QA exists. Keep >= 2 `status:ready` per role; file issues yourself if needed.
6. Retro check: closed swarm issues minus `retro.json.last_retro_closed_count` >= 15 -> run retro per `swarm/RETRO.md`. UAT queue cap: more than 5 `status:uat` open -> hold feature dispatches, give devs tech-debt/test work.
7. Update `workers.json` and `retro.json.last_tick_utc`; end with a 5-line report (dispatched, merged, filed, escalated, idle roles); schedule the next tick.

## Rules
`prio:p0` interrupts everything; reopened issues (N>=2) get a BA look first; check state before shared-state actions; escalations are reports only; at most one message per worker per tick unless replying. Before filing an issue, search open and closed swarm issues by title keywords (`gh issue list --label swarm --state all --search "<keywords>"`); file only when nothing matches.
