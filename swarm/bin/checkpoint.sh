#!/usr/bin/env bash
# Write a swarm checkpoint/heartbeat file atomically (PROTOCOL section 11).
# Usage: checkpoint.sh <role> <issue> <branch> <step> <next_action> [blockers]
# Env:   SWARM_STATE_DIR  override target dir (tests); default is the MAIN checkout's swarm/state.
#        SWARM_STATUS     busy (default) | idle | blocked
set -eu

if [ "$#" -lt 5 ] || [ "$#" -gt 6 ]; then
  echo "usage: checkpoint.sh <role> <issue> <branch> <step> <next_action> [blockers]" >&2
  exit 2
fi

role=$1 issue=$2 branch=$3 step=$4 next=$5 blockers=${6:-}
status=${SWARM_STATUS:-busy}

case "$role" in
  ''|*[!A-Za-z0-9_-]*) echo "invalid role: $role" >&2; exit 2 ;;
esac
case "$issue" in
  null|'') issue=null ;;
  *[!0-9]*) echo "invalid issue (digits or null): $issue" >&2; exit 2 ;;
esac
case "$status" in
  busy|idle|blocked) ;;
  *) echo "invalid SWARM_STATUS: $status" >&2; exit 2 ;;
esac

dir=${SWARM_STATE_DIR:-/c/Users/tvolo/dev/ai-dala/BilimBaga/swarm/state}
[ -d "$dir" ] || { echo "state dir not found: $dir" >&2; exit 1; }

# JSON string escape: backslash, quote, then all control chars (0x00-0x1F) to spaces.
esc() {
  local s=$1
  local bs=$'\\' dq='"'
  s=${s//"$bs"/"$bs$bs"}
  s=${s//"$dq"/"$bs$dq"}
  printf '%s' "$s" | tr '\000-\037' ' '
}

now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
tmp=$(mktemp "$dir/.${role}.XXXXXX")
trap 'rm -f "$tmp"' EXIT

printf '{"role":"%s","status":"%s","issue":%s,"branch":"%s","step":"%s","next_action":"%s","last_tick_utc":"%s","blockers":"%s"}\n' \
  "$(esc "$role")" "$status" "$issue" "$(esc "$branch")" "$(esc "$step")" "$(esc "$next")" "$now" "$(esc "$blockers")" > "$tmp"

mv -f "$tmp" "$dir/$role.json"
trap - EXIT
echo "checkpoint written: $dir/$role.json ($now)"
