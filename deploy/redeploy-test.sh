#!/usr/bin/env bash
# Redeploy BilimBaga to the test environment on hetzner-prod.
# Run as root on the host: bash /opt/apps/bilimbaga-test/deploy/redeploy-test.sh
set -euo pipefail

APP_DIR=/opt/apps/bilimbaga-test
COMPOSE="docker compose --project-directory $APP_DIR -f $APP_DIR/deploy/docker-compose.test.yml"
PAT_FILE=/root/.config/ai-dala-infra/github.token
DATE=$(date +%Y%m%d)

echo "=== BilimBaga test redeploy: $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="

# 1. Pull latest code
cd "$APP_DIR"
if [[ -f "$PAT_FILE" ]]; then
  PAT=$(cat "$PAT_FILE")
  git remote set-url origin "https://${PAT}@github.com/tvolodi/BilimBaga.git"
fi
git pull
CURRENT_REF=$(git rev-parse --short HEAD)
echo "Git ref: $CURRENT_REF"

# Restore unauthenticated remote URL so the PAT is never stored in git config
git remote set-url origin "https://github.com/tvolodi/BilimBaga.git"

# 2. Tag rollback images (best-effort — ignore if images don't exist yet)
docker tag bilimbaga-test:latest "bilimbaga-test:rollback-${DATE}" 2>/dev/null || true
docker tag bilimbaga-api-test:latest "bilimbaga-api-test:rollback-${DATE}" 2>/dev/null || true

# 3. Build images
echo "--- Building frontend image ---"
docker build -f deploy/Dockerfile -t bilimbaga-test:latest .

echo "--- Building API image ---"
docker build -f backend/Dockerfile -t bilimbaga-api-test:latest ./backend

# 4. Restart containers
echo "--- Restarting containers ---"
$COMPOSE up -d --force-recreate

# 5. Health check (retry for up to 30 s)
echo "--- Health check ---"
for i in $(seq 1 10); do
  STATUS=$(curl -sf http://127.0.0.1:3111/api/v1/health | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('status',''))" 2>/dev/null || true)
  if [[ "$STATUS" == "ok" ]]; then
    echo "Health check passed (attempt $i)"
    break
  fi
  if [[ $i -eq 10 ]]; then
    echo "ERROR: health check did not pass after 10 attempts" >&2
    exit 1
  fi
  echo "Waiting... ($i/10)"
  sleep 3
done

echo "=== Done. Deployed ref: $CURRENT_REF ==="
