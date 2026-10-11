#!/usr/bin/env bash
# Does a later PR head carry the same change as the approved head? (retro-011 decision 2)
# Prints SAME when the two heads differ only by a merge of the main branch, DIFFERENT otherwise. An architect decision
# stays valid for SAME only; any other new head goes back to the architect.
# Usage: same-change.sh <approved-sha> <new-sha>
# Env:   SWARM_MAIN_REF  the main branch (tests); default origin/main.
# Exit:  0 printed SAME or DIFFERENT; 2 usage or an unknown commit.
set -eo pipefail

[ "$#" -eq 2 ] || { echo 'usage: same-change.sh <approved-sha> <new-sha>' >&2; exit 2; }
main=${SWARM_MAIN_REF:-origin/main}

full_sha() {
  git rev-parse --verify -q "$1^{commit}" || { echo "unknown commit: $1" >&2; exit 2; }
}
approved=$(full_sha "$1")
new=$(full_sha "$2")

# The same commit is the same change.
if [ "$approved" = "$new" ]; then
  echo SAME
  exit 0
fi

# patch_id <full-sha>: the patch-id of the change from the merge base with main to the head. --binary keeps binary
# content in the diff, and --verbatim keeps whitespace: without them an indentation-only or binary change would match.
patch_id() {
  local base
  base=$(git merge-base "$main" "$1") || { echo "no merge base with $main for $1" >&2; exit 2; }
  git diff --binary "$base" "$1" | git patch-id --verbatim | awk '{print $1}'
}
a=$(patch_id "$approved")
n=$(patch_id "$new")

# An empty change (no diff at all) proves nothing, so it never matches.
if [ -n "$a" ] && [ "$a" = "$n" ]; then
  echo SAME
else
  echo DIFFERENT
fi
