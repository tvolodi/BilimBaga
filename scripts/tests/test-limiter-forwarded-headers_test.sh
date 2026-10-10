#!/usr/bin/env bash
# #484: test-limiter-forwarded-headers.sh must refuse a PROXY_TEST_BASE that is not this machine, before it sends a
# request. A curl stub on PATH records every call, so a request that gets out shows up in the log. No network is used.
#   usage: bash scripts/tests/test-limiter-forwarded-headers_test.sh
# Exit:  0 all cases pass; 1 a case failed.
set -u

here=$(cd "$(dirname "$0")" && pwd)
script="$here/../test-limiter-forwarded-headers.sh"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/bin"
cat > "$tmp/bin/curl" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "$CURL_LOG"
case "$*" in *"-X POST"*) echo 429 ;; *) echo 200 ;; esac
EOF
chmod +x "$tmp/bin/curl"

fail=0
pass() { echo "PASS $1"; }
bad() { echo "FAIL $1"; fail=1; }

# run_case <name> <base> -> sets out, code, calls
run_case() {
  export CURL_LOG="$tmp/calls.log"
  : > "$CURL_LOG"
  out=$(PATH="$tmp/bin:$PATH" PROXY_TEST_BASE="$2" bash "$script" 2>&1)
  code=$?
  calls=$(wc -l < "$CURL_LOG" | tr -d ' ')
}

# Bases that point off this machine must be refused with no request sent.
for base in \
  "https://bilimbaga-test.ai-dala.com" \
  "https://bilimbaga.ai-dala.com" \
  "http://bilimbaga-qa.ai-dala.com" \
  "http://localhost.example.com" \
  "http://127.0.0.1.example.com" \
  "http://localhost@bilimbaga.ai-dala.com" \
  "http://192.168.1.10:8080" \
  "http://10.0.0.5" \
  "localhost:3111" \
  "http://" \
  "http://localhost:80@evil.example.com" \
  "http://user@localhost" \
  "http://localhost/path@evil.example.com" \
  "http://localhost/?next=a@b"; do
  run_case refused "$base"
  if [ "$code" = 3 ] && [ "$calls" = 0 ] && [[ "$out" == *refused* ]]; then
    pass "refuses $base before any request (exit 3)"
  else
    bad "refuses $base: exit=$code requests=$calls out=$out"
  fi
done

# Local bases are accepted and the checks run.
for base in \
  "http://localhost" \
  "http://localhost:3111" \
  "http://127.0.0.1:80" \
  "http://[::1]:80"; do
  run_case local "$base"
  if [ "$code" = 0 ] && [ "$calls" -gt 0 ] && [[ "$out" == *"PASS limiter on"* ]]; then
    pass "accepts local base $base"
  else
    bad "accepts local base $base: exit=$code requests=$calls out=$out"
  fi
done

# An unset base defaults to the local container nginx, so it is accepted too.
run_case default ""
if [ "$code" = 0 ] && [ "$calls" -gt 0 ]; then
  pass "accepts the default base (unset PROXY_TEST_BASE)"
else
  bad "default base: exit=$code requests=$calls out=$out"
fi

exit "$fail"
