#!/usr/bin/env bash
# #481: plays the edge locally against the container nginx. The edge sets X-Real-IP to the address it saw, which
# is a fixed client in this script, and the client's own forwarded headers pass through. The per-IP sign-in limit
# (10 requests per minute) must count those requests as one client: a rotating X-Forwarded-For or True-Client-IP
# must not give a new bucket.
#
# A rotating X-Real-IP is NOT checked here: locally it is the edge's own address, so it is legitimate and not a finding.
# The three-container chain test (scripts/proxy-chain/) checks it, because there the edge overwrites it.
#
# Requires the local stack with rate limiting ON. The root .env sets DISABLE_RATE_LIMIT=true, so start the stack
# without it; with the limiter off the first check below fails, and nothing else is proven.
#   usage: bash scripts/test-limiter-forwarded-headers.sh
#   env:   PROXY_TEST_BASE  (default http://localhost:${HOST_HTTP_PORT:-80}, the container nginx; must be local)
# Exit:  0 all checks pass; 1 a rotating header chose the bucket, or the limiter is off; 2 the stack is not reachable;
#        3 PROXY_TEST_BASE is not this machine (refused before any request, #484).
set -u

base=${PROXY_TEST_BASE:-http://localhost:${HOST_HTTP_PORT:-80}}

# #484: the checks send sign-in requests, so a base that is not this machine (the demo, prod, a LAN host) is refused
# before any request leaves. The host is compared exactly; a userinfo or a suffix such as localhost.example.com fails.
case $base in
  http://*|https://*) ;;
  *) echo "refused: PROXY_TEST_BASE must be an http(s) URL on this machine, got '$base'"; exit 3 ;;
esac
hostport=${base#*://}
hostport=${hostport%%/*}
case $hostport in
  \[*) host=${hostport%%]*}] ;;
  *) host=${hostport%:*} ;;
esac
case $host in
  localhost|127.0.0.1|'[::1]') ;;
  *) echo "refused: PROXY_TEST_BASE must be localhost, 127.0.0.1 or [::1], got '$base' (this script sends sign-in requests)"; exit 3 ;;
esac

login="$base/api/v1/auth/login"

# A fresh client per check, so earlier runs and earlier checks in this run do not share a minute's bucket.
new_client() { echo "198.51.100.$(( (RANDOM % 200) + 20 ))"; }

code() { curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$@"; }

if [ "$(code "$base/api/v1/health")" != 200 ]; then
  echo "stack not reachable at $base (start the local stack, or set PROXY_TEST_BASE)"
  exit 2
fi

fail=0

# Sends 11 sign-in requests from one fixed X-Real-IP. $1 is the client, $2 the rotating header name ("" for none).
# Echoes the status of the 11th request.
burst() {
  local client=$1 hdr=$2 i last=0
  for i in $(seq 1 11); do
    if [ -n "$hdr" ]; then
      last=$(code -X POST "$login" -H 'Content-Type: application/json' -H "X-Real-IP: $client" -H "$hdr: 203.0.113.$((i + 10))" --data '{}')
    else
      last=$(code -X POST "$login" -H 'Content-Type: application/json' -H "X-Real-IP: $client" --data '{}')
    fi
  done
  echo "$last"
}

# 1. The limiter is on: a fixed client with no rotating header must reach 429 on the 11th request.
got=$(burst "$(new_client)" "")
if [ "$got" = 429 ]; then
  echo "PASS limiter on: a fixed client gets 429 on the 11th request"
else
  echo "FAIL limiter off or not reached: a fixed client's 11th request = $got (is DISABLE_RATE_LIMIT set?)"
  exit 1
fi

# 2. A rotating X-Forwarded-For must not choose the bucket (the container reads X-Real-IP only, #481).
got=$(burst "$(new_client)" "X-Forwarded-For")
if [ "$got" = 429 ]; then
  echo "PASS rotating X-Forwarded-For does not choose the bucket (11th = 429)"
else
  echo "FAIL rotating X-Forwarded-For chose the bucket: 11th request = $got"
  fail=1
fi

# 3. A rotating True-Client-IP must not choose the bucket (the edge clears it, the API ignores it, #475/#478).
got=$(burst "$(new_client)" "True-Client-IP")
if [ "$got" = 429 ]; then
  echo "PASS rotating True-Client-IP does not choose the bucket (11th = 429)"
else
  echo "FAIL rotating True-Client-IP chose the bucket: 11th request = $got"
  fail=1
fi

echo "INFO a rotating X-Real-IP is not checked: locally it is the edge's own address, not a finding"
exit "$fail"
