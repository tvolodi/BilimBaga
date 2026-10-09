# Retro 001

**Period**: swarm start 2026-10-09 -> 16 closed issues (counter baseline 0).

## Metrics
| Metric | Value |
|---|---|
| Closed issues | 16 (BA 8, UAT 3, dev 2, infra 3) |
| Median issue age at close (created->closed, GitHub timestamps) | ~5 min; max 22 min (#11 infra redeploy) |
| Reopens | 0 |
| Failed results / reassignments | 0 failed; 1 premature `done` (UAT #14, push failed, corrected) |
| Escalations (user-facing) | 1 recommendation (no scheduled DB backup on bilimbaga-test) |
| Blocked | #1/#2 blocked ~15 min by foreign process on port 8080 (resolved via #12) |
| Idle incidents | BA idle repeatedly (queue of BA work shallow); UAT idle while #1 blocked |
| Incidents | shared repo-root checkout switched under BA by UAT; host memory exhaustion killed live E2E (#1) |

Note: issue timestamps are compressed (clock of the host vs GitHub), so cycle times are indicative only.

## Went well
- Port-conflict handled without touching a foreign process; parametrization shipped in ~10 min (#12).
- Reviewer subagent found real defects (#37 first review CHANGES REQUESTED).
- BA drift check downgraded 7 stale README rows; 6 real dev issues came out of it.
- Infra was careful (dump first, rollback tags, test env only).

## Went badly
- Devs cannot force-push (settings) so rebases became merge commits or server-side squash; works but noisy.
- Shared checkout contention (UAT vs BA) lost a commit onto the wrong branch.
- Memory: 8 Claude sessions + docker + vite + playwright on one 32 GB host; harness killed live E2E.
- Idle notices arrive with stale timestamps, causing false "idle" signals; BA work is a thin queue and starts generating low-value reviews.
- Dev queue grows (19 ready) faster than 2 devs drain it.
- deploy/redeploy-test.sh drifted from the host reality (#51).

## Decisions (to apply via `retro-apply` issue)
1. PROTOCOL s.1/s.5: every role except Supervisor works in its own git worktree; the repo-root checkout is read-only for everyone but the Supervisor (moves: contention incidents -> 0).
2. PROTOCOL s.5: merging rule: "force-push denied -> merge origin/main into the branch" is the sanctioned path; squash-merge hides it (moves: conflict noise).
3. PROTOCOL new s.10 Resources: devs run vitest/go test with capped parallelism; UAT live runs need >= 8 GB free; Supervisor checks free memory each tick when UAT is running (moves: killed runs -> 0).
4. roles/supervisor.md: treat an idle notice older than the last message sent to that worker as stale; do not re-dispatch (moves: duplicate messages).
5. roles/supervisor.md anti-idle: cap BA self-generated work; prefer dev-queue draining; BA reviews only after a UAT report exists (moves: low-value BA issues).
6. Add a third dev lane only if memory allows; otherwise accept the backlog.

## Applied changes
Pending: issue "retro-001: apply swarm improvements" (role:dev, action retro-apply).

## Open recommendations for the user
- bilimbaga-test DB has no scheduled backup (decide whether the data matters).
- Consider force-push permission policy for dev worktree branches (own branches only).
- Host memory: 8 sessions + WSL docker + browsers is tight; consider closing other apps during UAT runs.
