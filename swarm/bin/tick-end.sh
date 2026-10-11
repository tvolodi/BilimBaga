#!/usr/bin/env bash
# Ends one Supervisor tick (retro-011 decision 1): appends the report to ticks.log, stamps retro.json.last_tick_utc,
# counts the closed swarm issues live, and prints "RETRO DUE" when a retro must start in this tick.
# Usage: tick-end.sh "<report line>"
# Env:   SWARM_STATE_DIR  state dir (tests); default <main checkout>/swarm/state, as tick-log.sh.
#        SWARM_NOW_UTC    ISO-8601 UTC "now" (tests); default the current time.
# Retro rule (RETRO.md): closures since the last cut >= interval, and the last cut is at least 4 h old.
#        A retro.json without last_cut_utc has no gap check.
# Exit:  0 ok (retro due or not); 1 gh or state error, nothing written; 2 usage.
set -eo pipefail

[ "$#" -eq 1 ] || { echo 'usage: tick-end.sh "<report line>"' >&2; exit 2; }

# Target dir: SWARM_STATE_DIR, or the main checkout's swarm/state (works from a worktree).
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
if [ -n "${SWARM_STATE_DIR:-}" ]; then
  dir=$SWARM_STATE_DIR
else
  common=$(git -C "$here" rev-parse --git-common-dir 2>/dev/null) || { echo "cannot resolve main checkout via git" >&2; exit 1; }
  dir=$(cd "$here" && cd "$(dirname "$common")" && pwd)/swarm/state
fi
[ -f "$dir/retro.json" ] || { echo "retro.json not found in $dir" >&2; exit 1; }

now=${SWARM_NOW_UTC:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
now_s=$(date -u -d "$now" +%s) || { echo "SWARM_NOW_UTC is not a time: $now" >&2; exit 1; }

interval=$(jq -r '.interval // 15' "$dir/retro.json")
cut_closed=$(jq -r '.last_retro_closed_count // 0' "$dir/retro.json")
last_cut=$(jq -r '.last_cut_utc // empty' "$dir/retro.json")

# Count first: if gh fails, nothing is written.
closed_json=$(gh issue list --label swarm --state closed --json number --limit 1000) \
  || { echo "gh issue list failed; nothing written" >&2; exit 1; }
closed=$(printf '%s' "$closed_json" | jq 'length')
diff=$((closed - cut_closed))

bash "$here/tick-log.sh" "$1"

tmp=$(mktemp)
jq --arg t "$now" '.last_tick_utc = $t' "$dir/retro.json" > "$tmp" && mv "$tmp" "$dir/retro.json"

if [ "$diff" -lt "$interval" ]; then
  echo "retro not due: $diff of $interval closures since the last cut"
  exit 0
fi
if [ -n "$last_cut" ]; then
  cut_s=$(date -u -d "$last_cut" +%s)
  if [ $((now_s - cut_s)) -lt 14400 ]; then
    echo "retro not due: $diff closures, but the last cut ($last_cut) is less than 4 h old"
    exit 0
  fi
fi
echo "RETRO DUE ($diff closures, last cut ${last_cut:-none})"
