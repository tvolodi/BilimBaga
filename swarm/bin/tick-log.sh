#!/usr/bin/env bash
# Append one Supervisor tick report line to swarm/state/ticks.log as "<UTC> <line>" (retro-010 decision 2).
# Usage: tick-log.sh "<report line>"
# Env:   SWARM_STATE_DIR  override the state dir (tests); default is <main checkout>/swarm/state.
set -eu

[ "$#" -eq 1 ] || { echo 'usage: tick-log.sh "<report line>"' >&2; exit 2; }
line=$1
case "$line" in
  '') echo "empty report line" >&2; exit 2 ;;
  *$'\n'*|*$'\r'*) echo "report must be a single line" >&2; exit 2 ;;
esac

# Target dir: SWARM_STATE_DIR, or the main checkout's swarm/state (works from a worktree).
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
if [ -n "${SWARM_STATE_DIR:-}" ]; then
  dir=$SWARM_STATE_DIR
else
  common=$(git -C "$here" rev-parse --git-common-dir 2>/dev/null) || { echo "cannot resolve main checkout via git" >&2; exit 1; }
  dir=$(cd "$here" && cd "$(dirname "$common")" && pwd)/swarm/state
fi
[ -d "$dir" ] || { echo "state dir not found: $dir" >&2; exit 1; }

now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
printf '%s %s\n' "$now" "$line" >> "$dir/ticks.log"
echo "tick logged: $dir/ticks.log ($now)"
