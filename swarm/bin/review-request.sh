#!/usr/bin/env bash
# Prints what a review request needs, read from gh, so nobody types a SHA, a branch name or a T4 verdict by hand
# (retro-011 decision 2; retro-012 decisions 1 and 2).
# Usage: review-request.sh <pr-number>
# Output: the PR number, branch, full head SHA, checks (green count, and the names of any not green), the closing-keyword
#         line (the body must name its issue as Refs #n), the T4 verdict, then the changed paths.
# Exit:   0 ok; 1 gh failed; 2 usage.
set -eo pipefail

if [ "$#" -ne 1 ] || ! [[ "$1" =~ ^[0-9]+$ ]]; then
  echo 'usage: review-request.sh <pr-number>' >&2
  exit 2
fi

json=$(gh pr view "$1" --json number,headRefName,headRefOid,files,statusCheckRollup,body) \
  || { echo "gh pr view $1 failed" >&2; exit 1; }

# jq_lf: jq on Windows writes CRLF line endings; the CR is removed so every field is compared as plain text.
jq_lf() { jq "$@" | tr -d '\r'; }

# protected <path>: the path is a protected path of PROTOCOL T4. swarm/tests/ is not protected. Any change under
# backend/migrations/ counts, since gh gives no status per file and a new file cannot be told from an edit here.
protected() {
  case "$1" in
    swarm/tests/*) return 1 ;;
    backend/internal/auth*|backend/internal/rbac/*|backend/internal/roles/*|backend/internal/middleware/*) return 0 ;;
    backend/internal/deptscope/*|backend/internal/router/router.go|backend/migrations/*) return 0 ;;
    deploy/*|swarm/*|docs/architecture-guide.md|docs/requirements/api-conventions.md) return 0 ;;
  esac
  case "$(basename "$1")" in
    docker-compose*.yml|docker-compose*.yaml|compose*.yml|compose*.yaml) return 0 ;;
  esac
  return 1
}

# t4_verdict: reads "path<TAB>additions<TAB>deletions" lines. In order: the first protected path; more than 15 files or
# 600 changed lines (additions plus deletions); a change to swarm/tests/ alone. Otherwise no.
t4_verdict() {
  local files=0 lines=0 first="" non_tests=0 path add del
  while IFS=$'\t' read -r path add del; do
    [ -n "$path" ] || continue
    files=$((files + 1))
    lines=$((lines + add + del))
    case "$path" in swarm/tests/*) ;; *) non_tests=1 ;; esac
    if [ -z "$first" ] && protected "$path"; then first=$path; fi
  done
  if [ "$files" -eq 0 ]; then
    echo "T4: no"
  elif [ -n "$first" ]; then
    echo "T4: yes ($first)"
  elif [ "$files" -gt 15 ] || [ "$lines" -gt 600 ]; then
    echo "T4: yes (size: $files files, $lines lines)"
  elif [ "$non_tests" -eq 0 ]; then
    echo "T4: no (swarm/tests only)"
  else
    echo "T4: no"
  fi
}

# closing_keyword <body>: GitHub closes an issue that a PR body names with one of these words, at the merge and outside
# set-status.sh (retro-012 decision 1). The line names each such phrase, or says none.
closing_keyword() {
  local hits n joined
  hits=$(grep -oiE '\b(close[sd]?|fix(e[sd])?|resolve[sd]?)[[:space:]]*:?[[:space:]]+#[0-9]+' <<<"$1" || true)
  if [ -z "$hits" ]; then
    echo "closing keyword: none"
    return
  fi
  n=$(wc -l <<<"$hits" | tr -d ' ')
  joined=$(paste -sd, - <<<"$hits" | sed 's/,/, /g')
  if [ "$n" -gt 1 ]; then
    echo "closing keyword: $joined (remove them)"
  else
    echo "closing keyword: $joined (remove it)"
  fi
}

printf '%s' "$json" | jq_lf -r '
  (.statusCheckRollup // []) as $checks
  | ([$checks[] | select((.conclusion // .state) != "SUCCESS") | (.name // .context)]) as $bad
  | "PR \(.number)",
    "branch \(.headRefName)",
    "head \(.headRefOid)",
    "checks \($checks | length - ($bad | length))/\($checks | length) green"
      + (if ($bad | length) > 0 then "; not green: " + ($bad | join(", ")) else "" end)
'
closing_keyword "$(printf '%s' "$json" | jq_lf -r '.body // ""')"
printf '%s' "$json" | jq_lf -r '.files[]? | [.path, (.additions // 0), (.deletions // 0)] | @tsv' | t4_verdict
printf '%s' "$json" | jq_lf -r '"files:", (.files[]? | "  " + .path)'
