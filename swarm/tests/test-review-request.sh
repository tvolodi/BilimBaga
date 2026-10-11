#!/usr/bin/env bash
# Verifies bin/review-request.sh (retro-011 decision 2) with a stubbed gh: every field comes from gh, the checks are
# counted and named, a bad argument is refused, and a gh failure exits 1. Works in a temp dir; no network call.
set -eu
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
mkdir -p "$t/bin"

# Stub gh: accepts only `gh pr view 42 --json <the four fields>`; prints the fixture named by STUB_PR_JSON.
cat > "$t/bin/gh" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = pr ] && [ "${2:-}" = view ] && [ "${3:-}" = 42 ] && [ "${4:-}" = --json ] \
  && [ "${5:-}" = number,headRefName,headRefOid,files,statusCheckRollup ] \
  || { echo "stub gh: unexpected: $*" >&2; exit 9; }
[ "${STUB_GH_FAIL:-0}" = 1 ] && { echo "stub gh: failure" >&2; exit 1; }
cat "$STUB_PR_JSON"
EOF
chmod +x "$t/bin/gh"

cat > "$t/mixed.json" <<'EOF'
{"number":42,"headRefName":"swarm/42-x","headRefOid":"0123456789abcdef0123456789abcdef01234567",
 "files":[{"path":"swarm/bin/x.sh"},{"path":"swarm/tests/test-x.sh"}],
 "statusCheckRollup":[
   {"name":"backend-tests","conclusion":"SUCCESS"},
   {"name":"lint","conclusion":"FAILURE"},
   {"name":"build","status":"IN_PROGRESS","conclusion":""},
   {"context":"legacy-status","state":"SUCCESS"}]}
EOF
cat > "$t/green.json" <<'EOF'
{"number":42,"headRefName":"swarm/42-x","headRefOid":"0123456789abcdef0123456789abcdef01234567",
 "files":[],"statusCheckRollup":[{"name":"backend-tests","conclusion":"SUCCESS"}]}
EOF

fail=0
check() { if [ "$2" = ok ]; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }
yn() { if "$@"; then echo ok; else echo no; fi; }
run() { PATH="$t/bin:$PATH" STUB_PR_JSON="$1" bash "$here/bin/review-request.sh" 42 2>&1; }

out=$(run "$t/mixed.json")
check "prints the PR number and branch" "$(yn bash -c 'grep -qx "PR 42" <<<"$1" && grep -qx "branch swarm/42-x" <<<"$1"' _ "$out")"
check "prints the full head SHA" "$(yn grep -qx "head 0123456789abcdef0123456789abcdef01234567" <<<"$out")"
check "counts the checks and names those not green" "$(yn grep -qx "checks 2/4 green; not green: lint, build" <<<"$out")"
check "lists the changed paths" "$(yn bash -c 'grep -qx "  swarm/bin/x.sh" <<<"$1" && grep -qx "  swarm/tests/test-x.sh" <<<"$1"' _ "$out")"

out=$(run "$t/green.json")
check "all checks green has no not-green list" "$(yn bash -c 'grep -qx "checks 1/1 green" <<<"$1" && ! grep -q "not green" <<<"$1"' _ "$out")"

code=0; PATH="$t/bin:$PATH" STUB_PR_JSON="$t/mixed.json" STUB_GH_FAIL=1 bash "$here/bin/review-request.sh" 42 >/dev/null 2>&1 || code=$?
check "a gh failure exits 1" "$(yn [ "$code" -eq 1 ])"

code=0; PATH="$t/bin:$PATH" bash "$here/bin/review-request.sh" abc >/dev/null 2>&1 || code=$?
check "a non-numeric PR is a usage error (exit 2)" "$(yn [ "$code" -eq 2 ])"

code=0; PATH="$t/bin:$PATH" bash "$here/bin/review-request.sh" >/dev/null 2>&1 || code=$?
check "no PR is a usage error (exit 2)" "$(yn [ "$code" -eq 2 ])"

exit "$fail"
