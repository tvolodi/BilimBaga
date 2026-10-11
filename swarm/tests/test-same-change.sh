#!/usr/bin/env bash
# Verifies bin/same-change.sh (retro-011 decision 2) on a temp git repo: a head that only merges main is SAME; a head with a
# further change is DIFFERENT; the same commit is SAME; an empty change never matches; bad arguments fail.
# Works in a temp dir; touches no real repository.
set -eu
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT

repo=$t/repo
git init -q -b main "$repo"
cd "$repo"
git config core.autocrlf false
g() { git -c user.name=t -c user.email=t@t "$@"; }

printf 'one\n' > a.txt
g add -A && g commit -qm base
g checkout -q -b feature
printf 'feature\n' > b.txt
g add -A && g commit -qm feature
approved=$(git rev-parse HEAD)

g checkout -q main
printf 'main two\n' > c.txt
g add -A && g commit -qm "main advances"
main_tip=$(git rev-parse HEAD)

g checkout -q feature
g merge -q --no-edit main
merged=$(git rev-parse HEAD)
printf 'more\n' > d.txt
g add -A && g commit -qm "extra change"
extra=$(git rev-parse HEAD)

g checkout -q -b empty-change main
g commit -q --allow-empty -m "nothing"
empty=$(git rev-parse HEAD)

fail=0
check() { if [ "$2" = ok ]; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }
run() { SWARM_MAIN_REF=main bash "$here/bin/same-change.sh" "$@" 2>/dev/null; }

out=$(run "$approved" "$merged") && code=0 || code=$?
check "a head that only merges main is SAME" "$([ "$out" = SAME ] && [ "$code" -eq 0 ] && echo ok || echo no)"

out=$(run "$approved" "$extra") && code=0 || code=$?
check "a head with a further change is DIFFERENT" "$([ "$out" = DIFFERENT ] && [ "$code" -eq 0 ] && echo ok || echo no)"

out=$(run "$approved" "$approved") && code=0 || code=$?
check "the same commit is SAME" "$([ "$out" = SAME ] && [ "$code" -eq 0 ] && echo ok || echo no)"

out=$(run "$main_tip" "$empty") && code=0 || code=$?
check "two empty changes never match: DIFFERENT" "$([ "$out" = DIFFERENT ] && [ "$code" -eq 0 ] && echo ok || echo no)"

code=0; run "$approved" >/dev/null || code=$?
check "one argument is a usage error (exit 2)" "$([ "$code" -eq 2 ] && echo ok || echo no)"

code=0; run "$approved" "0000000000000000000000000000000000000000" >/dev/null || code=$?
check "an unknown commit is an error (exit 2)" "$([ "$code" -eq 2 ] && echo ok || echo no)"

# Whitespace and binary changes must not match (the default patch-id drops whitespace and ignores binary content).
g checkout -q -b ws-approved "$main_tip"
mkdir -p ci
printf 'steps:\n  - run: a\n' > ci/ci.yml
g add -A && g commit -qm "indent two"
ws_approved=$(git rev-parse HEAD)

g checkout -q -b ws-new "$main_tip"
mkdir -p ci
printf 'steps:\n      - run: a\n' > ci/ci.yml
g add -A && g commit -qm "indent six"
ws_new=$(git rev-parse HEAD)

out=$(run "$ws_approved" "$ws_new") && code=0 || code=$?
check "an indentation-only YAML change is DIFFERENT" "$([ "$out" = DIFFERENT ] && [ "$code" -eq 0 ] && echo ok || echo no)"

g checkout -q -b bin-approved "$main_tip"
printf '\x00\x01\x02' > img.bin
g add -A && g commit -qm "binary a"
bin_approved=$(git rev-parse HEAD)

g checkout -q -b bin-new "$main_tip"
printf '\x00\x01\x03' > img.bin
g add -A && g commit -qm "binary b"
bin_new=$(git rev-parse HEAD)

out=$(run "$bin_approved" "$bin_new") && code=0 || code=$?
check "a changed binary file is DIFFERENT" "$([ "$out" = DIFFERENT ] && [ "$code" -eq 0 ] && echo ok || echo no)"

exit "$fail"
