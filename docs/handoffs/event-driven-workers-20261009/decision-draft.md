# Decision draft: event-driven workers (no worker /loop)

Status: DRAFT for the Supervisor, nothing applied. Author: BA role, at the owner's request (2026-10-09).
Owner's position: workers behave like servers (take a task, process it, sleep until useful). Delivery, retry, dead-session detection and "is the team idle" are the Supervisor's job. `/loop` solves none of the communication problems.

## Decision
1. Workers (dev1, dev2, ba, uat, infra) run no `/loop` and no periodic tick. After startup handshake and after each task they end their turn and wait for a message.
2. The ack is the claim: the worker comments `claimed-by: <role>` on the issue and flips the label to `status:in-progress`. A `result` message follows when done.
3. The Supervisor owns delivery: a `task` message carries `deadline_min`. No claim comment by then -> re-send once -> then treat the session as dead (restart via the ensure-up watchdog #145, or reassign per the existing escalation order).
4. Duplicate tasks are harmless: a worker that already has a claim comment on the issue ignores the repeat.
5. Liveness is on demand: the Supervisor sends `ping`, the worker answers `pong` with its status. No pong within a deadline = dead session. No heartbeat files, no timer wakeups.
6. "Never idle" and self-generated default work move to the Supervisor. It decides when the BA audits, when UAT runs a sweep, or that nothing should happen. A worker with no task does nothing.
7. GitHub issues and labels remain the durable queue and the source of truth. Messages are doorbells that carry pointers (unchanged).

## Rule edits needed (not applied)
| File | Change |
|------|--------|
| `swarm/roles/_common.md` rule 2 | Replace "start a dynamic /loop" with "after the handshake end the turn; wait for messages" |
| `swarm/roles/_common.md` rule 10 | Remove "role default work"; "nothing to do -> wait" |
| `swarm/roles/_common.md` rule 13 | Checkpoint at step changes only (no per-tick heartbeat) |
| `swarm/PROTOCOL.md` "Anti-idle" | Replace with the Supervisor's idle-monitoring duty; drop the worker /loop sentence |
| `swarm/PROTOCOL.md` "Checkpoints and heartbeats" | Drop stale-heartbeat staleness; liveness = ping/pong with deadline |
| `swarm/PROTOCOL.md` "Claiming" | Add: claim comment is the ack; Supervisor re-sends after `deadline_min` |
| `swarm/roles/ba.md` "Tick" | Replace with "On a task: ...", default-work steps move to a Supervisor-triggered `task` |
| Other role files | Same removal of tick sections |

## Open checks before applying
- Verify once, between two sessions with a finished turn, that a cross-session message wakes the receiver. Observed in the BA session (permission mode bypass); the other roles and modes are untested.
- Restart path: after a restart the worker still resumes own `in-progress` issues from GitHub (common rule 5, unchanged).
- Supervisor must not lose its own loop: it is the only role that keeps one (queue scan, ping sweep, idle-team detection).

## Risks accepted
- A message lost while a session is down is recovered only by the Supervisor's deadline retry, not by the worker polling.
- Less self-generated BA/UAT output until the Supervisor requests it (intended).
