#!/usr/bin/env bash
# Verifies bin/tick-end.sh (retro-011 decision 1) with a stubbed gh: RETRO DUE by count and by gap, and the cases that are
# not due. Works in a temp dir; touches no real state and makes no network call.
set -eu
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
mkdir -p "$t/bin" "$t/state"

# Stub gh: `gh issue list` prints STUB_CLOSED closed issues, or fails when STUB_GH_FAIL=1.
cat > "$t/bin/gh" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = issue ] && [ "${2:-}" = list ] || { echo "stub gh: unexpected: $*" >&2; exit 9; }
[ "${STUB_GH_FAIL:-0}" = 1 ] && { echo "stub gh: failure" >&2; exit 1; }
n=${STUB_CLOSED:-0}
printf '['
for i in $(seq 1 "$n"); do [ "$i" -gt 1 ] && printf ','; printf '{"number":%d}' "$i"; done
printf ']\n'
EOF
chmod +x "$t/bin/gh"

fail=0
check() { if [ "$2" = ok ]; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }
yn() { if "$@"; then echo ok; else echo no; fi; }

# write_retro <last_retro_closed_count> <last_cut_utc as JSON: "ISO" or null>
write_retro() {
  printf '{"last_retro_closed_count":%s,"retro_number":11,"interval":15,"last_tick_utc":null,"last_cut_utc":%s}\n' \
    "$1" "$2" > "$t/state/retro.json"
}

# run_tick <closed count> <now> <report> : prints the script output, returns its exit code.
run_tick() {
  PATH="$t/bin:$PATH" STUB_CLOSED="$1" SWARM_NOW_UTC="$2" SWARM_STATE_DIR="$t/state" \
    bash "$here/bin/tick-end.sh" "$3" 2>&1
}

now=2026-10-11T12:00:00Z

write_retro 15 '"2026-10-11T06:00:00Z"'
code=0; out=$(run_tick 30 "$now" "due by count and gap") || code=$?
check "due when closures >= interval and the last cut is 4 h or more old" \
  "$(yn bash -c '[[ "$1" == *"RETRO DUE (15 closures, last cut 2026-10-11T06:00:00Z)"* ]] && [ "$2" = 0 ]' _ "$out" "$code")"

write_retro 15 '"2026-10-11T06:00:00Z"'
code=0; out=$(run_tick 29 "$now" "not due by count") || code=$?
check "not due by count: 14 of 15 closures" \
  "$(yn bash -c '[[ "$1" == *"retro not due: 14 of 15"* ]] && [[ "$1" != *"RETRO DUE"* ]] && [ "$2" = 0 ]' _ "$out" "$code")"

write_retro 15 '"2026-10-11T10:00:00Z"'
code=0; out=$(run_tick 30 "$now" "not due by gap") || code=$?
check "not due by gap: the last cut is 2 h old" \
  "$(yn bash -c '[[ "$1" == *"less than 4 h old"* ]] && [[ "$1" != *"RETRO DUE"* ]] && [ "$2" = 0 ]' _ "$out" "$code")"

write_retro 15 '"2026-10-11T08:00:00Z"'
code=0; out=$(run_tick 30 "$now" "boundary") || code=$?
check "exactly 4 h since the last cut is due" \
  "$(yn bash -c '[[ "$1" == *"RETRO DUE"* ]]' _ "$out")"

write_retro 15 null
code=0; out=$(run_tick 30 "$now" "no last cut") || code=$?
check "no last_cut_utc: due by count alone, reported as none" \
  "$(yn bash -c '[[ "$1" == *"RETRO DUE (15 closures, last cut none)"* ]]' _ "$out")"

write_retro 15 null
code=0; out=$(run_tick 30 "$now" "tick report one") || code=$?
check "the report is appended to ticks.log" \
  "$(yn grep -q "tick report one" "$t/state/ticks.log")"
check "retro.json.last_tick_utc is stamped with the tick time" \
  "$(yn bash -c '[ "$(jq -r .last_tick_utc "$1")" = "$2" ]' _ "$t/state/retro.json" "$now")"

# gh failure: nothing is written.
write_retro 15 null
rm -f "$t/state/ticks.log"
code=0; out=$(PATH="$t/bin:$PATH" STUB_GH_FAIL=1 SWARM_NOW_UTC="$now" SWARM_STATE_DIR="$t/state" \
  bash "$here/bin/tick-end.sh" "failing tick" 2>&1) || code=$?
check "a gh failure exits non-zero" "$(yn [ "$code" -ne 0 ])"
check "a gh failure writes neither ticks.log nor last_tick_utc" \
  "$(yn bash -c '[ ! -f "$1/ticks.log" ] && [ "$(jq -r .last_tick_utc "$1/retro.json")" = null ]' _ "$t/state")"

# A report that is not one line is refused by tick-log.sh, and nothing is written.
write_retro 15 null
code=0; out=$(run_tick 30 "$now" "$(printf 'line one\nline two')") || code=$?
check "a multi-line report is refused and last_tick_utc is not written" \
  "$(yn bash -c '[ "$1" -ne 0 ] && [ "$(jq -r .last_tick_utc "$2/retro.json")" = null ]' _ "$code" "$t/state")"

# Usage.
code=0; PATH="$t/bin:$PATH" bash "$here/bin/tick-end.sh" >/dev/null 2>&1 || code=$?
check "no report argument is a usage error (exit 2)" "$(yn [ "$code" -eq 2 ])"

exit "$fail"
