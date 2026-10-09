# FR-BB65 performance tooling (D1, D3)

Nothing here is run in CI. UAT (D4) runs it against a LOCAL stack only. Spec: `docs/requirements/FR-BB65.Performance.md`.

## Safety defaults

- `BASE_URL` has no default. Hosts other than localhost / `*.localhost` / 127.0.0.0/8 / `::1` are refused unless `ALLOW_REMOTE=1` (never set it by default).
- `bilimbaga-test.ai-dala.com` (and subdomains) is always refused, no override (DEC-001).
- `DISABLE_RATE_LIMIT=true` is a test-only API env; set it on the LOCAL API for the k6 run only and restore it afterward.

## k6 (AC-2): `docs/test-reports/k6-load-test.js`

Six tagged groups (`ep`), each with `p(95)<200` ms: `exams`, `users_me`, `tenant_config`, `categories`, `portal_exams` (employee token), `admin_dash` (admin token). Global: `http_req_failed` < 1 %, `checks` >= 99 %. Default profile 100 VUs, 2 m (`VUS`, `DURATION` override).

```bash
k6 run -e BASE_URL=http://localhost:8080 \
  -e K6_ADMIN_EMAIL=admin@bilimbaga.local -e K6_ADMIN_PASS=... \
  -e K6_EMPLOYEE_EMAIL=... -e K6_EMPLOYEE_PASS=... \
  --summary-export=docs/test-reports/k6-summary-<date>.json docs/test-reports/k6-load-test.js
```

Docker `--network host` reaches the host's localhost on Linux only; on Docker Desktop (Windows/macOS) use a native k6 install or run k6 inside the compose network against the API service (the guard refuses `host.docker.internal`). `TEST_TOKEN` alone (with `ALLOW_PARTIAL=1`) covers only the `exams` group and is not a full AC-2 run.

## Lighthouse (AC-3): `scripts/perf/lighthouse.sh <url>`

Pinned `lighthouse@11` via `npx`, Performance category, JSON+HTML into `docs/test-reports/lighthouse/` (`LH_OUT_DIR` overrides). Gate: median Performance >= `LH_MIN_SCORE` (90). The pages are behind auth: pass a signed-in Chrome profile (`LH_USER_DATA_DIR`) or `LH_EXTRA_HEADERS_FILE`; the run fails if it ends on a different path (login redirect). Mobile is gating per the FR; desktop is recorded (use `LH_MIN_SCORE=0` to record only). Exit 0 pass, 1 below threshold, 2 usage/refused target, 3 run failed.

```bash
LH_RUNS=3 bash scripts/perf/lighthouse.sh http://localhost/portal                 # mobile, gating
LH_RUNS=3 LH_PRESET=desktop bash scripts/perf/lighthouse.sh http://localhost/portal
```

## Bundle check (AC-7): `scripts/perf/bundle-check.sh [dist]`

Reads `frontend/dist` (`BUILD=1` builds it when missing), no network. Prints per-file gzip sizes and checks: initial JS (entry + modulepreload) <= 170 kB gzip (`BUDGET_INITIAL_JS_KB`), CSS <= 50 kB gzip (`BUDGET_CSS_KB`), >= 10 separate lazy chunks (`MIN_LAZY_CHUNKS`). Exit 1 when over budget. It does not verify that no admin page code sits in the entry chunk (only the lazy chunk count); check that by reading the entry chunk list above.

```bash
cd frontend && npm run build && bash ../scripts/perf/bundle-check.sh
```
