#!/usr/bin/env bash
# Verifies bin/review-request.sh (retro-011 decision 2; retro-012 decisions 1 and 2; #575 review) with a stubbed gh: every
# field comes from gh, the checks are counted and named, a bad argument is refused, a gh failure exits 1, the closing-keyword
# line reads the title, the body and the commit messages in each of the three reference forms, and the T4 verdict follows
# PROTOCOL T4 path by path, at the size limits, and with the swarm/tests exemption at any size. Works in a temp dir; no
# network call.
set -eu
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
mkdir -p "$t/bin"

# Stub gh: accepts only `gh pr view 42 --json <the eight fields>`; prints the fixture named by STUB_PR_JSON.
cat > "$t/bin/gh" <<'STUB'
#!/usr/bin/env bash
[ "${1:-}" = pr ] && [ "${2:-}" = view ] && [ "${3:-}" = 42 ] && [ "${4:-}" = --json ] \
  && [ "${5:-}" = number,headRefName,headRefOid,files,statusCheckRollup,body,title,commits ] \
  || { echo "stub gh: unexpected: $*" >&2; exit 9; }
[ "${STUB_GH_FAIL:-0}" = 1 ] && { echo "stub gh: failure" >&2; exit 1; }
cat "$STUB_PR_JSON"
STUB
chmod +x "$t/bin/gh"

# pr <files-json> <body-json> [title-json] [commits-json]: the answer of gh for PR 42, with one green check. The title and
# the commit messages are clean unless given.
pr() {
  local title=${3:-'"Example change"'}
  local commits=${4:-'[{"messageHeadline":"chore: example","messageBody":""}]'}
  printf '{"number":42,"headRefName":"swarm/42-x","headRefOid":"0123456789abcdef0123456789abcdef01234567","files":%s,"statusCheckRollup":[{"name":"backend-tests","conclusion":"SUCCESS"}],"body":%s,"title":%s,"commits":%s}' \
    "$1" "$2" "$title" "$commits"
}

# files <count> <additions> <deletions> <path-prefix>: a files list of <count> files, each with those counts.
files() {
  local i out=""
  for ((i = 0; i < $1; i++)); do
    out+="${out:+,}{\"path\":\"$4$i\",\"additions\":$2,\"deletions\":$3}"
  done
  printf '[%s]' "$out"
}

# one <path> <additions> <deletions>: a files list of one file.
one() { printf '[{"path":"%s","additions":%s,"deletions":%s}]' "$1" "$2" "$3"; }

cat > "$t/mixed.json" <<'EOF'
{"number":42,"headRefName":"swarm/42-x","headRefOid":"0123456789abcdef0123456789abcdef01234567",
 "files":[{"path":"swarm/bin/x.sh","additions":2,"deletions":1},{"path":"swarm/tests/test-x.sh","additions":3,"deletions":0}],
 "statusCheckRollup":[
   {"name":"backend-tests","conclusion":"SUCCESS"},
   {"name":"lint","conclusion":"FAILURE"},
   {"name":"build","status":"IN_PROGRESS","conclusion":""},
   {"context":"legacy-status","state":"SUCCESS"}],
 "body":"Refs #42","title":"Example change","commits":[{"messageHeadline":"chore: example","messageBody":""}]}
EOF
cat > "$t/green.json" <<'EOF'
{"number":42,"headRefName":"swarm/42-x","headRefOid":"0123456789abcdef0123456789abcdef01234567",
 "files":[],"statusCheckRollup":[{"name":"backend-tests","conclusion":"SUCCESS"}],"body":"","title":"Example change","commits":[]}
EOF

fail=0
check() { if [ "$2" = ok ]; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }
yn() { if "$@"; then echo ok; else echo no; fi; }
run() { PATH="$t/bin:$PATH" STUB_PR_JSON="$1" bash "$here/bin/review-request.sh" 42 2>&1; }
# expect <name> <fixture> <exact line>: the output has that whole line.
expect() { local out; out=$(run "$2"); check "$1" "$(yn grep -qxF -- "$3" <<<"$out")"; }

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

# Closing keywords (retro-012 decision 1; #575 review): the title, the body and the commit messages, in the three forms.
pr "$(one backend/internal/sessions/a.go 1 0)" '"Refs #570"' > "$t/refs.json"
expect "a clean title, body and commit have no closing keyword" "$t/refs.json" "closing keyword: none"
pr "$(one backend/internal/sessions/a.go 1 0)" '"Fixes #570"' > "$t/fixes.json"
expect "a Fixes line in the body is reported to be removed" "$t/fixes.json" "closing keyword: Fixes #570 in body (remove it)"
pr "$(one backend/internal/sessions/a.go 1 0)" '"Closes #1. Resolves #2"' > "$t/two.json"
expect "two closing phrases in the body are both reported" "$t/two.json" "closing keyword: Closes #1 in body, Resolves #2 in body (remove them)"
pr "$(one backend/internal/sessions/a.go 1 0)" '"Fixes tvolodi/BilimBaga#570"' > "$t/repo-ref.json"
expect "owner/repo#n after a closing word is reported" "$t/repo-ref.json" "closing keyword: Fixes tvolodi/BilimBaga#570 in body (remove it)"
pr "$(one backend/internal/sessions/a.go 1 0)" '"Closes https://github.com/tvolodi/BilimBaga/issues/570"' > "$t/url-ref.json"
expect "an issue URL after a closing word is reported" "$t/url-ref.json" "closing keyword: Closes https://github.com/tvolodi/BilimBaga/issues/570 in body (remove it)"
pr "$(one backend/internal/sessions/a.go 1 0)" '"Refs #575"' '"Example change"' '[{"messageHeadline":"fix: x","messageBody":"Fixes #5"}]' > "$t/commit-kw.json"
expect "a Fixes line in a commit message is reported, with its place" "$t/commit-kw.json" "closing keyword: Fixes #5 in commit message (remove it)"
pr "$(one backend/internal/sessions/a.go 1 0)" '"Refs #575"' '"Fixes #5"' > "$t/title-kw.json"
expect "a Fixes phrase in the title is reported, with its place" "$t/title-kw.json" "closing keyword: Fixes #5 in title (remove it)"

# T4 by path class (retro-012 decision 2): each protected path of PROTOCOL T4 is yes, and the verdict names that path.
for p in backend/internal/auth/login.go backend/internal/authz.go backend/internal/rbac/cache.go \
         backend/internal/roles/x.go backend/internal/middleware/x.go backend/internal/deptscope/x.go \
         backend/internal/router/router.go backend/migrations/099_x.up.sql deploy/nginx.conf \
         swarm/bin/x.sh docs/architecture-guide.md docs/requirements/api-conventions.md \
         docker-compose.yml scripts/proxy-chain/docker-compose.yml; do
  pr "$(one "$p" 1 0)" '"Refs #42"' > "$t/t4.json"
  expect "T4 yes for $p" "$t/t4.json" "T4: yes ($p)"
done
pr "$(one backend/internal/sessions/service.go 1 0)" '"Refs #42"' > "$t/plain.json"
expect "an ordinary path is T4 no" "$t/plain.json" "T4: no"
pr "$(one backend/internal/router/other.go 1 0)" '"Refs #42"' > "$t/router-other.json"
expect "only router.go itself is protected, not its neighbours" "$t/router-other.json" "T4: no"

# The first protected path in the list is the one named.
pr '[{"path":"backend/internal/sessions/a.go","additions":1,"deletions":0},{"path":"deploy/x.yml","additions":1,"deletions":0},{"path":"swarm/bin/y.sh","additions":1,"deletions":0}]' '"Refs #42"' > "$t/first.json"
expect "the first protected path is named" "$t/first.json" "T4: yes (deploy/x.yml)"

# Size (PROTOCOL T4: more than 15 files or 600 lines; additions plus deletions).
pr "$(files 16 1 0 backend/internal/sessions/f)" '"Refs #42"' > "$t/16files.json"
expect "16 files is T4 yes by size" "$t/16files.json" "T4: yes (size: 16 files, 16 lines)"
pr "$(files 15 1 0 backend/internal/sessions/f)" '"Refs #42"' > "$t/15files.json"
expect "15 files is not over the limit" "$t/15files.json" "T4: no"
pr "$(one backend/internal/sessions/big.go 500 101)" '"Refs #42"' > "$t/601.json"
expect "601 changed lines is T4 yes by size" "$t/601.json" "T4: yes (size: 1 files, 601 lines)"
pr "$(one backend/internal/sessions/big.go 500 100)" '"Refs #42"' > "$t/600.json"
expect "600 changed lines is not over the limit" "$t/600.json" "T4: no"

# The exemption (PROTOCOL: a change to swarm/tests/ alone is not T4, at any size). A mix with an ordinary path is decided
# by that path.
pr "$(one swarm/tests/test-x.sh 1 0)" '"Refs #42"' > "$t/tests-only.json"
expect "a swarm/tests-only change is T4 no" "$t/tests-only.json" "T4: no (swarm/tests only)"
pr "$(one swarm/tests/test-big.sh 500 200)" '"Refs #42"' > "$t/tests-big.json"
expect "a swarm/tests-only change of 700 lines is still T4 no (the exemption comes before the size rule)" "$t/tests-big.json" "T4: no (swarm/tests only)"
pr '[{"path":"swarm/tests/test-x.sh","additions":1,"deletions":0},{"path":"backend/internal/sessions/a.go","additions":1,"deletions":0}]' '"Refs #42"' > "$t/tests-mixed.json"
expect "a swarm/tests change with an ordinary path is T4 no, without the exception line" "$t/tests-mixed.json" "T4: no"

exit "$fail"
