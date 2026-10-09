#!/usr/bin/env bash
# FR-BB65 AC-7 / D3: check the production bundle split and gzip budget. No network, nothing installed.
# Usage: scripts/perf/bundle-check.sh [dist_dir]     (default frontend/dist; run `npm run build` in frontend/ first)
# Env:   BUDGET_INITIAL_JS_KB  max gzipped size of the initial JS (entry + modulepreload) in kB (default 170, FR-BB65 D2)
#        BUDGET_CSS_KB         max gzipped size of all CSS in kB (default 50)
#        MIN_LAZY_CHUNKS       minimum number of separate JS chunks besides the entry (default 10, FR-BB65 E4)
#        BUILD=1               run `npm run build` in frontend/ first when dist is missing
# Exit:  0 within budget, 1 over budget / too few chunks, 2 usage or missing dist.
set -eu

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
dist=${1:-$repo_root/frontend/dist}
js_budget=${BUDGET_INITIAL_JS_KB:-170}
css_budget=${BUDGET_CSS_KB:-50}
min_chunks=${MIN_LAZY_CHUNKS:-10}
for v in "$js_budget" "$css_budget" "$min_chunks"; do
  case "$v" in ''|*[!0-9]*) echo "budget values must be integers (got: $v)" >&2; exit 2 ;; esac
done

if [ ! -d "$dist/assets" ] && [ "${BUILD:-}" = "1" ]; then
  echo "dist missing, building frontend..." >&2
  (cd "$repo_root/frontend" && npm run build) >&2
fi
[ -d "$dist/assets" ] || { echo "no $dist/assets (run 'npm run build' in frontend/ or set BUILD=1)" >&2; exit 2; }
[ -f "$dist/index.html" ] || { echo "no $dist/index.html" >&2; exit 2; }

gz_bytes() { gzip -9 -c -- "$1" | wc -c | tr -d ' \r'; }
kb() { local b=$1; echo "$(( b / 1024 )).$(( (b % 1024) * 10 / 1024 ))"; }
nl='
'

# Entry chunk(s): JS referenced by index.html via <script src>; preloads: <link rel="modulepreload" href>.
entry_files=$(grep -o '<script[^>]*src="[^"]*"' "$dist/index.html" | sed 's/.*src="\([^"]*\)"/\1/; s|^\./||; s|^/||' || true)
[ -n "$entry_files" ] || { echo "no <script src> found in index.html" >&2; exit 2; }
preload_files=$(grep -o '<link[^>]*rel="modulepreload"[^>]*>' "$dist/index.html" | sed 's/.*href="\([^"]*\)".*/\1/; s|^\./||; s|^/||' || true)

status=0
initial_total=0
lazy_chunks=0
entry_count=0
printf '%-52s %10s %10s  %s\n' file raw_kB gzip_kB role
for f in "$dist"/assets/*.js; do
  [ -f "$f" ] || continue
  rel=assets/$(basename "$f")
  role=lazy
  case "$nl$entry_files$nl" in *"$nl$rel$nl"*) role=entry ;; esac
  if [ "$role" = lazy ]; then
    case "$nl$preload_files$nl" in *"$nl$rel$nl"*) role=preload ;; esac
  fi
  raw=$(wc -c < "$f" | tr -d ' \r'); gz=$(gz_bytes "$f")
  printf '%-52s %10s %10s  %s\n' "$(basename "$f")" "$(kb "$raw")" "$(kb "$gz")" "$role"
  if [ "$role" = entry ] || [ "$role" = preload ]; then initial_total=$((initial_total + gz)); fi
  if [ "$role" = entry ]; then entry_count=$((entry_count + 1)); elif [ "$role" = lazy ]; then lazy_chunks=$((lazy_chunks + 1)); fi
done
[ "$entry_count" -ge 1 ] || { echo "entry script from index.html not found in $dist/assets" >&2; exit 2; }

css_total=0
for f in "$dist"/assets/*.css; do
  [ -f "$f" ] || continue
  raw=$(wc -c < "$f" | tr -d ' \r'); gz=$(gz_bytes "$f")
  printf '%-52s %10s %10s  %s\n' "$(basename "$f")" "$(kb "$raw")" "$(kb "$gz")" css
  css_total=$((css_total + gz))
done

js_limit=$((js_budget * 1024)); css_limit=$((css_budget * 1024))
echo
printf '%-34s %10s kB  budget %s kB  ' "initial JS (entry+preload, gzip)" "$(kb "$initial_total")" "$js_budget"
if [ "$initial_total" -le "$js_limit" ]; then echo OK; else echo OVER; status=1; fi
printf '%-34s %10s kB  budget %s kB  ' "CSS total (gzip)" "$(kb "$css_total")" "$css_budget"
if [ "$css_total" -le "$css_limit" ]; then echo OK; else echo OVER; status=1; fi
printf '%-34s %10s     minimum %s     ' "lazy JS chunks (not entry/preload)" "$lazy_chunks" "$min_chunks"
if [ "$lazy_chunks" -ge "$min_chunks" ]; then echo OK; else echo TOO_FEW; status=1; fi

if [ "$status" -eq 0 ]; then echo PASS; else echo FAIL >&2; fi
exit "$status"
