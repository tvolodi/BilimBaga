#!/usr/bin/env bash
# Redeploy BilimBaga to the test environment on hetzner-prod.
# PROTECTED TARGET: bilimbaga-test.ai-dala.com is a customer demo (production-class, #107).
# This script is an ops artifact for the human owner only. Swarm roles (dev, UAT, Infra, Supervisor)
# must NEVER run it or deploy to this host.
#
# Run as root on the host: sudo bash /opt/apps/bilimbaga-test/deploy/redeploy-test.sh
#
# Code is updated with a fast-forward-only pull over the checkout's existing `origin`
# remote (a read-only deploy-key SSH alias, git@github.com-bilimbaga:tvolodi/BilimBaga.git).
# This script never reads any credential file and never rewrites git remotes.
set -euo pipefail

APP_DIR=/opt/apps/bilimbaga-test
COMPOSE="docker compose --project-directory $APP_DIR -f $APP_DIR/deploy/docker-compose.test.yml"
# Timestamped so a same-day re-run never overwrites an earlier rollback point.
STAMP=$(date -u +%Y%m%d-%H%M%S)

echo "=== BilimBaga test redeploy: $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="

# 1. Pull latest code (fast-forward only, on the existing remote)
cd "$APP_DIR"
if [[ -n "$(git status --porcelain)" ]]; then
  echo "ERROR: working tree in $APP_DIR is not clean; refusing to deploy:" >&2
  git status --short >&2
  exit 1
fi
PREVIOUS_REF=$(git rev-parse --short HEAD)
git pull --ff-only origin main
CURRENT_REF=$(git rev-parse --short HEAD)
echo "Git ref: $PREVIOUS_REF -> $CURRENT_REF"

# 2. Tag rollback images (best-effort: ignore if images don't exist yet)
docker tag bilimbaga-test:latest "bilimbaga-test:rollback-${STAMP}" 2>/dev/null || true
docker tag bilimbaga-api-test:latest "bilimbaga-api-test:rollback-${STAMP}" 2>/dev/null || true
echo "Rollback tags: bilimbaga-test:rollback-${STAMP}, bilimbaga-api-test:rollback-${STAMP}"

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
    echo "Rollback: docker tag bilimbaga-test:rollback-${STAMP} bilimbaga-test:latest && docker tag bilimbaga-api-test:rollback-${STAMP} bilimbaga-api-test:latest && $COMPOSE up -d --force-recreate" >&2
    exit 1
  fi
  echo "Waiting... ($i/10)"
  sleep 3
done

echo "=== Done. Deployed ref: $CURRENT_REF ==="
