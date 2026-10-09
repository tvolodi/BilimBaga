#!/usr/bin/env bash
# FR-BB65 AC-3 / D3: run Lighthouse (pinned v11, via npx, nothing installed globally) against a LOCAL url.
# Usage: scripts/perf/lighthouse.sh <url>            (or LH_URL=<url> scripts/perf/lighthouse.sh)
# Env:   LH_PRESET      mobile (default, gating) | desktop
#        LH_RUNS        number of runs; the gate uses the median (default 1; the FR asks for 3)
#        LH_MIN_SCORE   minimum Performance score, 0-100 (default 90, from FR-BB65 AC-3)
#        LH_OUT_DIR     report dir (default docs/test-reports/lighthouse, relative to repo root)
#        LH_LABEL       file name prefix (default: derived from the url path, e.g. portal)
#        LH_CHROME_FLAGS  override chrome flags (default: --headless=new)
#        LH_USER_DATA_DIR  Chrome profile dir that is already signed in (the pages are behind auth);
#                          create it once by logging in with a headed Chrome started with --user-data-dir
#        LH_EXTRA_HEADERS_FILE  JSON file passed to --extra-headers (e.g. an Authorization header)
# The run fails (exit 3) if the page Lighthouse ended on differs from the requested path (login redirect).
#        ALLOW_REMOTE=1 required for any non-local host. bilimbaga-test.ai-dala.com is ALWAYS refused.
# Writes <out>/<label>-<preset>-run<N>.report.{json,html}; exits 1 when the median is below the minimum,
# 2 on usage/safety errors, 3 when a run produced no score.
set -eu

url=${1:-${LH_URL:-}}
if [ -z "$url" ]; then
  echo "usage: lighthouse.sh <url>   (url of a LOCAL stack, e.g. http://localhost/portal)" >&2
  exit 2
fi

preset=${LH_PRESET:-mobile}
runs=${LH_RUNS:-1}
min=${LH_MIN_SCORE:-90}
case "$preset" in mobile|desktop) ;; *) echo "invalid LH_PRESET: $preset" >&2; exit 2 ;; esac
case "$runs" in ''|*[!0-9]*|0) echo "invalid LH_RUNS: $runs" >&2; exit 2 ;; esac
case "$min" in ''|*[!0-9]*) echo "invalid LH_MIN_SCORE: $min" >&2; exit 2 ;; esac
[ "$min" -le 100 ] || { echo "invalid LH_MIN_SCORE: $min" >&2; exit 2; }

# ---- target guard (same policy as scripts/lib/target-guard.ts and the k6 script) ----
case "$url" in
  http://*|https://*) ;;
  *) echo "refusing: url must start with http:// or https:// (got: $url)" >&2; exit 2 ;;
esac
rest=${url#*://}
auth=${rest%%[/?#]*}
bs=$(printf '\\')
case "$auth" in
  *"$bs"*|*%*) echo "refusing: forbidden characters in host part: $auth" >&2; exit 2 ;;
esac
if printf '%s' "$auth" | LC_ALL=C grep -q '[^!-~]'; then
  echo "refusing: non-ASCII or whitespace in host part" >&2; exit 2
fi
auth=${auth##*@}                      # drop userinfo; real host follows the last '@'
case "$auth" in
  \[*) host=${auth#\[}; host=${host%%\]*} ;;
  *)   host=${auth%%:*} ;;
esac
host=$(printf '%s' "$host" | tr 'A-Z' 'a-z')
while [ "${host%.}" != "$host" ]; do host=${host%.}; done
[ -n "$host" ] || { echo "refusing: empty host" >&2; exit 2; }

case "$host" in
  bilimbaga-test.ai-dala.com|*.bilimbaga-test.ai-dala.com)
    echo "refusing: $host is the customer demo (DEC-001); there is no override" >&2; exit 2 ;;
esac
local_host=0
case "$host" in
  localhost|*.localhost|::1) local_host=1 ;;
  127.*.*.*)  # only a pure dotted-quad 127/8 address (not 127.0.0.1.evil.com)
    case "$host" in
      *[!0-9.]*) ;;
      *.*.*.*.*|*..*) ;;
      *) local_host=1 ;;
    esac ;;
esac
if [ "$local_host" -ne 1 ] && [ "${ALLOW_REMOTE:-}" != "1" ]; then
  echo "refusing non-local host '$host'. Run against a local stack; ALLOW_REMOTE=1 only with explicit owner approval." >&2
  exit 2
fi

# ---- paths ----
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out=${LH_OUT_DIR:-$repo_root/docs/test-reports/lighthouse}
mkdir -p "$out"
label=${LH_LABEL:-}
if [ -z "$label" ]; then
  p=${rest#"$auth"}; p=${p%%[?#]*}
  label=$(printf '%s' "$p" | tr -c 'A-Za-z0-9' '-' | sed 's/^-*//; s/-*$//')
  [ -n "$label" ] || label=root
fi
case "$label" in *[!A-Za-z0-9._-]*) echo "invalid LH_LABEL: $label" >&2; exit 2 ;; esac

# Native node/npx on Git Bash wants Windows-style paths.
native() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi; }

command -v npx >/dev/null 2>&1 || { echo "npx not found (Node.js required)" >&2; exit 2; }

preset_arg=()
[ "$preset" = desktop ] && preset_arg=(--preset=desktop)
chrome_flags=${LH_CHROME_FLAGS:---headless=new}
if [ -n "${LH_USER_DATA_DIR:-}" ]; then
  [ -d "$LH_USER_DATA_DIR" ] || { echo "LH_USER_DATA_DIR not found: $LH_USER_DATA_DIR" >&2; exit 2; }
  chrome_flags="$chrome_flags --user-data-dir=$(native "$LH_USER_DATA_DIR")"
fi
hdr_arg=()
if [ -n "${LH_EXTRA_HEADERS_FILE:-}" ]; then
  [ -f "$LH_EXTRA_HEADERS_FILE" ] || { echo "LH_EXTRA_HEADERS_FILE not found" >&2; exit 2; }
  hdr_arg=(--extra-headers="$(native "$LH_EXTRA_HEADERS_FILE")")
fi
req_path=${rest#"$auth"}; req_path=${req_path%%[?#]*}; [ -n "$req_path" ] || req_path=/

scores=()
i=1
while [ "$i" -le "$runs" ]; do
  base=$out/$label-$preset-run$i
  echo "lighthouse run $i/$runs ($preset) $url" >&2
  npx --yes lighthouse@11 "$url" \
    --only-categories=performance \
    --output=json --output=html \
    --output-path="$(native "$base")" \
    --chrome-flags="$chrome_flags" \
    --quiet \
    ${preset_arg[@]+"${preset_arg[@]}"} ${hdr_arg[@]+"${hdr_arg[@]}"} >&2 || { echo "lighthouse failed on run $i" >&2; exit 3; }
  json=$base.report.json
  [ -f "$json" ] || { echo "no report written: $json" >&2; exit 3; }
  s=$(node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"));const c=r.categories&&r.categories.performance;console.log(c&&typeof c.score==="number"?Math.round(c.score*100):"")' "$(native "$json")") || s=
  case "$s" in ''|*[!0-9]*) echo "run $i produced no performance score (see $json)" >&2; exit 3 ;; esac
  final_path=$(node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"));try{console.log(new URL(r.finalDisplayedUrl||r.finalUrl).pathname)}catch(e){console.log("")}' "$(native "$json")") || final_path=
  if [ "${final_path%/}" != "${req_path%/}" ]; then
    echo "run $i ended on '$final_path' instead of '$req_path' (probably redirected to login): sign in via LH_USER_DATA_DIR / LH_EXTRA_HEADERS_FILE" >&2
    exit 3
  fi
  scores+=("$s")
  i=$((i + 1))
done

sorted=$(printf '%s\n' "${scores[@]}" | sort -n)
idx=$(( (runs + 1) / 2 ))             # lower median for even counts (conservative)
median=$(printf '%s\n' "$sorted" | sed -n "${idx}p")

printf '\n%-12s %-8s %-6s %s\n' label preset runs scores
printf '%-12s %-8s %-6s %s\n' "$label" "$preset" "$runs" "${scores[*]}"
printf 'median Performance: %s (minimum %s)\n' "$median" "$min"
if [ "$median" -ge "$min" ]; then
  echo "PASS"
else
  echo "FAIL: median $median < $min (record failing audits from the JSON, then apply D2)" >&2
  exit 1
fi
