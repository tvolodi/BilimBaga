#!/usr/bin/env bash
# Set an issue's status label, optionally its role, and optionally close it (retro-010 decision 1, retro-011 #518).
# Usage: set-status.sh <issue> <ready|in-progress|review|uat|blocked|done> [--role <dev|ba|uat|infra>] [--close] [--pr <n>]
#   One gh issue edit removes every other status:* label and adds the new one.
#   --role adds role:<role> and removes every other role:* label in the same edit (the state machine flips the role
#   at each merge: review -> uat, role:uat). It cannot be combined with done.
#   Status done removes role:* in the same edit (#518: closed issues carry no role, with or without --close).
#   --close requires status done, then closes the issue.
#   --pr <n> is only valid with --close: the close is refused unless gh pr view <n> reports MERGED.
#   Without --pr, --close is a report-only close (no PR).
# All checks run before any label is changed, so a refused close leaves the issue as it was.
set -eu

usage() {
  echo "usage: set-status.sh <issue> <ready|in-progress|review|uat|blocked|done> [--role <dev|ba|uat|infra>] [--close] [--pr <n>]" >&2
  exit 2
}

[ "$#" -ge 2 ] || usage
issue=$1 status=$2
shift 2
close=0 pr="" role="" role_given=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --close) close=1; shift ;;
    --pr) [ "$#" -ge 2 ] || usage; pr=$2; shift 2 ;;
    --role) [ "$#" -ge 2 ] || usage; role=$2; role_given=1; shift 2 ;;
    *) usage ;;
  esac
done

case "$issue" in ''|*[!0-9]*) echo "invalid issue (digits): $issue" >&2; exit 2 ;; esac
case "$status" in ready|in-progress|review|uat|blocked|done) ;; *) echo "invalid status: $status" >&2; exit 2 ;; esac
if [ "$role_given" -eq 1 ]; then
  case "$role" in dev|ba|uat|infra) ;; *) echo "invalid role: $role (use dev, ba, uat or infra)" >&2; exit 2 ;; esac
  [ "$status" != done ] || { echo "--role cannot be combined with done (done clears role:*)" >&2; exit 2; }
fi
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

status_rm=() role_rm=() role_add=()
if [ "$role_given" -eq 1 ]; then role_add+=("--add-label" "role:$role"); fi
while IFS= read -r label; do
  [ -n "$label" ] || continue
  case "$label" in
    status:*) [ "$label" = "status:$status" ] || status_rm+=("--remove-label" "$label") ;;
    role:*)
      if [ "$role_given" -eq 1 ] && [ "$label" = "role:$role" ]; then
        :  # already the role being set: keep it
      elif [ "$close" -eq 1 ] || [ "$status" = done ] || [ "$role_given" -eq 1 ]; then
        role_rm+=("--remove-label" "$label")
      fi ;;
  esac
done <<< "$labels"

gh issue edit "$issue" --add-label "status:$status" \
  ${role_add[@]+"${role_add[@]}"} ${status_rm[@]+"${status_rm[@]}"} ${role_rm[@]+"${role_rm[@]}"}
if [ "$role_given" -eq 1 ]; then
  echo "issue #$issue: status:$status role:$role"
else
  echo "issue #$issue: status:$status"
fi

if [ "$close" -eq 1 ]; then
  gh issue close "$issue"
  echo "issue #$issue: closed"
fi
