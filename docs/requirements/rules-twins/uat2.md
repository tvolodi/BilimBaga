# UAT role: workflow (twin of swarm/roles/uat.md)

Pipelines: `.claude/commands/uat-runner.md`, `e2e-repair.md` (test part only; register issues instead of fixing), `test-run-error-resolution.md`. Owns the live stack.

## Work
- Stack health: `curl http://localhost:${BB_API_PORT:-8080}/api/v1/health`. Down: `git pull --ff-only` in `.claude/worktrees/uat`, `make dev` (background), wait for health; `make migrate` after merges with migrations. Port 8080 taken by a foreign process: `BB_API_PORT=18080 make dev` and export `E2E_API_URL`.
- Before each run fast-forward the worktree to `origin/main` and rebuild if needed.
- Verify `role:uat status:uat` issues with the scenario from `docs/uat-scenarios` or targeted Playwright. PASS: close, comment, `status:done`. FAIL: `role:dev status:ready`, comment with report path and `reopen: N`.
- `needs-live-db` issues first, against the migrated live DB; a SQL PR with neither label nor real-Postgres test is treated as `needs-live-db` and reported.
- System tests: `cd frontend && npm run test:e2e:live`, parse `e2e-results.json`, plus feature-area sweeps. Each failure becomes an issue (`gh issue create --label swarm,role:dev,status:ready,type:bug,prio:pX` with repro, expected/actual, report and screenshot paths); search for duplicates and group same-root-cause failures.
- Scenarios declare `Target: local | qa`; non-local targets need `E2E_API_URL` and `E2E_BASE_URL`.
- Reports to `docs/uat-reports/` and `docs/test-reports/` via a docs PR (`swarm/uat-<date>`).

## Tick
1. Stack health. 2. `gh issue list --label role:uat --state open`: needs-live-db, other `status:uat`, then `status:ready`, by prio. 3. None: regression sweep (full E2E or the longest-untested area, dates in `docs/uat-reports/`) plus exploratory testing of `gh pr list --state merged --limit 10`. 4. Checkpoint with sub-step progress at least every 10 min on long runs. 5. After each run send the Supervisor a `result` (passed/failed, issues filed). Never end idle.
