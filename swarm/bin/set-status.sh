#!/usr/bin/env bash
# Set an issue's status label, and optionally close it (retro-010 decision 1).
# Usage: set-status.sh <issue> <ready|in-progress|review|uat|blocked|done> [--close] [--pr <n>]
#   One gh issue edit removes every other status:* label and adds the new one.
#   --close requires status done, removes role:* in the same edit, then closes the issue.
#   --pr <n> is only valid with --close: the close is refused unless gh pr view <n> reports MERGED.
#   Without --pr, --close is a report-only close (no PR).
# All checks run before any label is changed, so a refused close leaves the issue as it was.
set -eu

usage() {
  echo "usage: set-status.sh <issue> <ready|in-progress|review|uat|blocked|done> [--close] [--pr <n>]" >&2
  exit 2
}

[ "$#" -ge 2 ] || usage
issue=$1 status=$2
shift 2
close=0 pr=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --close) close=1; shift ;;
    --pr) [ "$#" -ge 2 ] || usage; pr=$2; shift 2 ;;
    *) usage ;;
  esac
done

case "$issue" in ''|*[!0-9]*) echo "invalid issue (digits): $issue" >&2; exit 2 ;; esac
case "$status" in ready|in-progress|review|uat|blocked|done) ;; *) echo "invalid status: $status" >&2; exit 2 ;; esac
if [ -n "$pr" ]; then
  [ "$close" -eq 1 ] || { echo "--pr is only valid with --close" >&2; exit 2; }
  case "$pr" in ''|*[!0-9]*) echo "invalid PR (digits): $pr" >&2; exit 2 ;; esac
fi
if [ "$close" -eq 1 ] && [ "$status" != done ]; then
  echo "--close requires status done (got: $status)" >&2
  exit 1
fi

# Refuse before touching any label: the PR must be merged for a close that names it.
if [ "$close" -eq 1 ] && [ -n "$pr" ]; then
  state=$(gh pr view "$pr" --json state --jq .state)
  if [ "$state" != MERGED ]; then
    echo "refused: PR $pr is $state, not MERGED; issue $issue left unchanged" >&2
    exit 1
  fi
fi

labels=$(gh issue view "$issue" --json labels --jq '.labels[].name')

status_rm=() role_rm=()
while IFS= read -r label; do
  [ -n "$label" ] || continue
  case "$label" in
    status:*) [ "$label" = "status:$status" ] || status_rm+=("--remove-label" "$label") ;;
    role:*) [ "$close" -eq 1 ] && role_rm+=("--remove-label" "$label") ;;
  esac
done <<< "$labels"

gh issue edit "$issue" --add-label "status:$status" \
  ${status_rm[@]+"${status_rm[@]}"} ${role_rm[@]+"${role_rm[@]}"}
echo "issue #$issue: status:$status"

if [ "$close" -eq 1 ]; then
  gh issue close "$issue"
  echo "issue #$issue: closed"
fi
