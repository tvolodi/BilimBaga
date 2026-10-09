# ISS-114 QA instance compose + redeploy script

Added `deploy/docker-compose.qa.yml`, `deploy/redeploy-qa.sh`, `.env.qa.example` for bilimbaga-qa.ai-dala.com.
Isolated from test: compose project `bilimbaga-qa`, images `bilimbaga-qa` / `bilimbaga-api-qa`, volume `bilimbaga_qa_pgdata`,
host port 127.0.0.1:3114, APP_DIR `/opt/apps/bilimbaga-qa`. Existing test files untouched. Nothing deployed.
Validated: `bash -n`, `docker compose config` render only (dummy empty .env outside repo).
