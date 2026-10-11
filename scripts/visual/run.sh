#!/usr/bin/env bash
# FR-BB320 AC-11: visual regression tests in the pinned Playwright container.
#
#   scripts/visual/run.sh test     compare the screenshots with the committed baselines
#   scripts/visual/run.sh update   regenerate the baselines; commit the PNG files that change
#
# Baselines are made only here, inside the Linux container, never on Windows or macOS. The image tag is the exact
# @playwright/test version resolved in frontend/package-lock.json. The stack is throwaway: a Postgres 16 container and the
# API binary (built here for linux/amd64) on one Docker network, and the Playwright container runs the API and the preview
# server itself, so the browser talks to localhost. The target guard (scripts/lib/target-guard.ts) accepts localhost only.
set -euo pipefail
# Git Bash would rewrite container paths such as /work into Windows paths; this switches that off (no effect on Linux).
export MSYS_NO_PATHCONV=1

MODE=${1:-}
case "$MODE" in
  test|update) ;;
  *) echo "usage: scripts/visual/run.sh test|update" >&2; exit 2 ;;
esac

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
# Windows node and docker need C:/ paths when this runs under Git Bash; cygpath converts, elsewhere the path is unchanged.
host_path() {
  if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi
}
HROOT=$(host_path "$ROOT")
VERSION=$(cd "$ROOT/frontend" && node -p "require('./package-lock.json').packages['node_modules/@playwright/test'].version")
IMAGE="mcr.microsoft.com/playwright:v${VERSION}-jammy"
RUN="visual-$$"
NET="bb-${RUN}-net"
PG="bb-${RUN}-pg"
APP="bb-${RUN}-app"

cleanup() {
  docker rm -f "$APP" "$PG" >/dev/null 2>&1 || true
  docker network rm "$NET" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# The exact image, from the official registry. Nothing is piped from the network into a shell (CLAUDE.md Owner authority 6).
docker image inspect "$IMAGE" >/dev/null 2>&1 || docker pull "$IMAGE"

# Throwaway secrets for this run only: generated here, passed as environment, never written to a file.
DB_PASS=$(openssl rand -hex 16)
JWT=$(openssl rand -hex 32)

mkdir -p "$ROOT/.visual"
(cd "$ROOT/backend" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$HROOT/.visual/api" ./cmd/api)

docker network create "$NET" >/dev/null
docker run -d --name "$PG" --network "$NET" --pull never \
  -e POSTGRES_USER=bilimbaga -e POSTGRES_PASSWORD="$DB_PASS" -e POSTGRES_DB=bilimbaga postgres:16 >/dev/null
for _ in $(seq 1 60); do
  docker exec "$PG" pg_isready -U bilimbaga -d bilimbaga >/dev/null 2>&1 && break
  sleep 1
done

# tr strips CR in case a checkout gave the script CRLF endings.
docker run --rm --name "$APP" --network "$NET" --shm-size=1g \
  -v "$HROOT/scripts:/work/scripts:ro" \
  -v "$HROOT/frontend:/work/frontend" \
  -v "bb-visual-node-modules:/work/frontend/node_modules" \
  -v "$HROOT/backend/migrations:/work/backend/migrations:ro" \
  -v "$HROOT/.visual/api:/work/backend/bin/api:ro" \
  -e VISUAL_MODE="$MODE" \
  -e DB_HOST="$PG" -e DB_PORT=5432 -e DB_NAME=bilimbaga -e DB_USER=bilimbaga -e DB_PASSWORD="$DB_PASS" \
  -e JWT_SECRET="$JWT" -e API_PORT=8080 -e API_BASE_URL=http://localhost:8080 \
  -e PUBLIC_APP_URL=http://localhost:4173 \
  -e E2E_BASE_URL=http://localhost:4173 -e E2E_API_URL=http://localhost:8080 \
  -w /work/frontend "$IMAGE" \
  bash -c "tr -d '\r' < /work/scripts/visual/in-container.sh | bash -s"
