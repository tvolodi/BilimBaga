#!/usr/bin/env bash
# Verifies bin/checkpoint.sh resolves the MAIN checkout's state dir from swarm/roster.json, also from a worktree.
# Works entirely inside a temp git repo; touches nothing of the real repo.
set -eu
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
main=$t/main
mkdir -p "$main/swarm/bin" "$main/swarm/state"
cp "$here/bin/checkpoint.sh" "$main/swarm/bin/"
cp "$here/roster.json" "$main/swarm/"
git -C "$main" init -q
git -C "$main" -c user.name=t -c user.email=t@t add -A
git -C "$main" -c user.name=t -c user.email=t@t commit -qm init
git -C "$main" worktree add -q "$t/wt" -b wt

fail=0
check() { if [ "$2" = ok ]; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }

(cd "$main" && bash swarm/bin/checkpoint.sh dev1 1 b s n >/dev/null)
[ -f "$main/swarm/state/dev1.json" ] && check "main checkout writes own state" ok || check "main checkout writes own state" no

mkdir -p "$t/wt/swarm/state"
(cd "$t/wt" && bash swarm/bin/checkpoint.sh dev2 2 b s n >/dev/null)
[ -f "$main/swarm/state/dev2.json" ] && [ ! -f "$t/wt/swarm/state/dev2.json" ] && check "worktree writes to main checkout state" ok || check "worktree writes to main checkout state" no

SWARM_STATE_DIR="$t/override" bash "$main/swarm/bin/checkpoint.sh" dev1 1 b s n 2>/dev/null && check "override dir must exist" no || check "override dir must exist" ok
exit $fail
