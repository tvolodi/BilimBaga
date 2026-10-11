#!/usr/bin/env bash
# Prints what a review request needs, read from gh, so nobody types a SHA or a branch name by hand (retro-011 decision 2).
# Usage: review-request.sh <pr-number>
# Output: the PR number, branch, full head SHA, checks (green count, and the names of any not green), changed paths.
# Exit:   0 ok; 1 gh failed; 2 usage.
set -eo pipefail

if [ "$#" -ne 1 ] || ! [[ "$1" =~ ^[0-9]+$ ]]; then
  echo 'usage: review-request.sh <pr-number>' >&2
  exit 2
fi

json=$(gh pr view "$1" --json number,headRefName,headRefOid,files,statusCheckRollup) \
  || { echo "gh pr view $1 failed" >&2; exit 1; }

printf '%s' "$json" | jq -r '
  (.statusCheckRollup // []) as $checks
  | ([$checks[] | select((.conclusion // .state) != "SUCCESS") | (.name // .context)]) as $bad
  | "PR \(.number)",
    "branch \(.headRefName)",
    "head \(.headRefOid)",
    "checks \($checks | length - ($bad | length))/\($checks | length) green"
      + (if ($bad | length) > 0 then "; not green: " + ($bad | join(", ")) else "" end),
    "files:",
    (.files[]? | "  " + .path)
'
