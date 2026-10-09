# Code Review: ISS-114 QA instance compose/redeploy

Verdict: PASS (0 Critical, 0 High)

Files: deploy/docker-compose.qa.yml, deploy/redeploy-qa.sh, .env.qa.example (compared to the test equivalents).

## Isolation from bilimbaga-test
- Compose project: explicit `name: bilimbaga-qa` (default network `bilimbaga-qa_default`, distinct). OK
- Images: bilimbaga-qa / bilimbaga-api-qa (test: bilimbaga-test / bilimbaga-api-test); rollback tags use the same QA names. OK
- Volume: bilimbaga_qa_pgdata (project-scoped, distinct). OK
- Host port: 127.0.0.1:3114 (test 3111), loopback only; health check uses 3114. OK
- Paths: APP_DIR=/opt/apps/bilimbaga-qa, compose file is the qa one. OK
- No stray test references (grep of deploy/ and env example: only comments).
- Script is a faithful copy of redeploy-test.sh (set -euo pipefail, clean-tree guard, ff-only pull, best-effort tags, health retry, rollback hint); no new risk.

## Secrets
- .env.qa.example has placeholders only; JWT_SECRET/DB_PASSWORD are replace-with-... values. DB/cookie/URL values are QA-specific. OK

## Findings
- Low: no `.env.qa.example` counterpart pattern issue, but nothing in the repo provisions a host nginx vhost for bilimbaga-qa.ai-dala.com (test has deploy/nginx/bilimbaga-test.conf proxying to 3111). A QA vhost proxying to 3114 is needed for the instance to be reachable; likely a separate step.
- Info: `env_file: .env` resolves relative to the compose file dir in Compose v2; identical to the test file, so parity is preserved (host must place .env wherever test does).
- Info: both instances share the host Docker build cache and `docker build` tags only; no cross-impact.
- Info: ensure DB_NAME/DB_USER/DB_PASSWORD in the QA .env are not reused from test.
