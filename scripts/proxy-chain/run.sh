#!/usr/bin/env bash
# #484: the real three-container chain for the sign-in limiter's client address: edge nginx -> container nginx -> API.
# Each runs in its own container, in a throwaway compose project with generated secrets. It proves that the client's
# own forwarded headers do not choose the bucket through the whole chain. It never touches the dev stack, the test
# host or prod, and it tears the project down on exit.
#   usage: bash scripts/proxy-chain/run.sh
#   env:   PROXY_CHAIN_PORT  (edge host port, default 18090, bound to 127.0.0.1)
# Needs Docker with Compose v2.
# Exit:  0 all checks pass; 1 a check failed; 2 the chain did not come up.
set -u

here=$(cd "$(dirname "$0")" && pwd)
compose_file="$here/docker-compose.yml"
project="bb-proxy-chain-$$"
port=${PROXY_CHAIN_PORT:-18090}
base="http://127.0.0.1:$port"
login="$base/api/v1/auth/login"

secret() { LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 40; }
export PROXY_CHAIN_DB_PASSWORD=$(secret)
export PROXY_CHAIN_JWT_SECRET=$(secret)
export PROXY_CHAIN_PORT=$port

compose() { docker compose -p "$project" -f "$compose_file" "$@"; }
code() { curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$@"; }
post_login() { code -X POST "$login" -H 'Content-Type: application/json' "$@" --data '{}'; }

cleanup() { compose down -v --remove-orphans >/dev/null 2>&1; }
trap cleanup EXIT

if ! compose up -d --build --wait --wait-timeout 300 >/dev/null 2>&1; then
  echo "FAIL the chain did not come up; last logs:"
  compose logs --tail 60
  exit 2
fi

up=0
for _ in $(seq 1 60); do
  if [ "$(code "$base/api/v1/health")" = 200 ]; then up=1; break; fi
  sleep 2
done
if [ "$up" != 1 ]; then
  echo "FAIL the chain is up but /api/v1/health through the edge never returned 200; last logs:"
  compose logs --tail 60
  exit 2
fi

fail=0

# 1. The limiter is on through the chain. One client (the edge's own address, since the edge replaces X-Real-IP):
#    the first 10 sign-in requests are not limited and the 11th is.
statuses=()
for i in $(seq 1 11); do statuses+=("$(post_login)"); done
ok=1
for i in 0 1 2 3 4 5 6 7 8 9; do [ "${statuses[$i]}" = 429 ] && ok=0; done
[ "${statuses[10]}" = 429 ] || ok=0
if [ "$ok" = 1 ]; then
  echo "PASS limiter on through the chain: requests 1-10 not limited, request 11 = 429"
else
  echo "FAIL limiter through the chain: statuses ${statuses[*]} (expected 1-10 not 429, 11 = 429; is DISABLE_RATE_LIMIT set?)"
  fail=1
fi

# 2. The same client, now with a rotating forwarded address in every header the chain could read. The bucket is
#    already spent by check 1 and the edge must not let a new address open a fresh one, so every request is 429.
#    If the chain counted any of these headers, the rotating addresses would get fresh buckets and be served (not 429).
rotated=()
for i in $(seq 1 11); do
  rotated+=("$(post_login -H "X-Forwarded-For: 203.0.113.$i" -H "True-Client-IP: 198.51.100.$i" -H "X-Real-IP: 192.0.2.$i")")
done
all429=1
for s in "${rotated[@]}"; do [ "$s" = 429 ] || all429=0; done
if [ "$all429" = 1 ]; then
  echo "PASS rotating X-Forwarded-For, True-Client-IP and X-Real-IP do not choose the bucket (11 of 11 = 429)"
else
  echo "FAIL a rotating header chose a bucket through the chain: statuses ${rotated[*]} (expected all 429)"
  fail=1
fi

echo "INFO one run, one stack: every check uses the same edge address, so the bucket is the chain's, not a fixture's"
exit "$fail"
