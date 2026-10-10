#!/usr/bin/env bash
# Verifies bin/set-status.sh with a stub gh first on PATH. The real gh is never called.
set -eu
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
mkdir -p "$t/stub"
cat > "$t/stub/gh" <<'EOF'
#!/usr/bin/env bash
# Stub gh: logs each call; answers "issue view" with GH_LABELS and "pr view" with GH_PR_STATE.
echo "$*" >> "$GH_LOG"
case "$1 $2" in
  "issue view") printf '%s\n' "$GH_LABELS" ;;
  "pr view") echo "$GH_PR_STATE" ;;
esac
exit 0
EOF
chmod +x "$t/stub/gh"
export PATH="$t/stub:$PATH"
export GH_LOG="$t/gh.log"

fail=0
check() { if [ "$2" = ok ]; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }
logged() { grep -qxF -- "$1" "$GH_LOG"; }
run() { : > "$GH_LOG"; bash "$here/bin/set-status.sh" "$@" >/dev/null 2>&1; }

# A: two status labels set; setting ready removes both others in one edit.
export GH_LABELS="$(printf 'role:dev\nstatus:review\nstatus:uat\nswarm')" GH_PR_STATE=OPEN
run 5 ready && logged "issue edit 5 --add-label status:ready --remove-label status:review --remove-label status:uat" \
  && check "two statuses set: one edit removes both others" ok || check "two statuses set: one edit removes both others" no

# B: close naming a PR that is not MERGED is refused before any label change.
export GH_LABELS="$(printf 'role:dev\nstatus:review\nswarm')" GH_PR_STATE=OPEN
if run 5 done --close --pr 9; then rc=0; else rc=$?; fi
[ "$rc" -ne 0 ] && ! grep -q "issue edit\|issue close" "$GH_LOG" \
  && check "close without MERGED refused, no label or close call" ok || check "close without MERGED refused, no label or close call" no

# C: report-only close (no PR): one edit sets done and removes role:* and other statuses, then closes.
export GH_LABELS="$(printf 'role:uat\nstatus:uat\nswarm')" GH_PR_STATE=OPEN
run 5 done --close && logged "issue edit 5 --add-label status:done --remove-label status:uat --remove-label role:uat" \
  && logged "issue close 5" && ! grep -q "pr view" "$GH_LOG" \
  && check "report-only close: edit clears labels, then closes, no PR check" ok || check "report-only close: edit clears labels, then closes, no PR check" no

# D: close naming a MERGED PR goes through.
export GH_LABELS="$(printf 'role:dev\nstatus:review\nswarm')" GH_PR_STATE=MERGED
run 5 done --close --pr 9 && logged "issue close 5" \
  && check "close with MERGED PR closes the issue" ok || check "close with MERGED PR closes the issue" no

# E: --close with a status other than done is refused.
export GH_LABELS="$(printf 'status:review\nswarm')" GH_PR_STATE=OPEN
if run 5 review --close; then rc=0; else rc=$?; fi
[ "$rc" -ne 0 ] && ! grep -q "issue edit\|issue close" "$GH_LOG" \
  && check "--close requires done" ok || check "--close requires done" no

# F: tick-log appends one UTC-prefixed line to ticks.log.
state=$t/state && mkdir -p "$state"
SWARM_STATE_DIR="$state" bash "$here/bin/tick-log.sh" "dispatched 1; merged 0" >/dev/null
grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]{8}Z dispatched 1; merged 0$' "$state/ticks.log" \
  && check "tick-log appends one UTC-prefixed line" ok || check "tick-log appends one UTC-prefixed line" no

exit $fail
