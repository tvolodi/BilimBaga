#!/usr/bin/env bash
# Prints what a review request needs, read from gh, so nobody types a SHA, a branch name or a T4 verdict by hand
# (retro-011 decision 2; retro-012 decisions 1 and 2; #575 review).
# Usage: review-request.sh <pr-number>
# Output: the PR number, branch, full head SHA, checks (green count, and the names of any not green), the closing-keyword
#         line (the title, the body and the commit messages must name their issue as Refs #n: a squash merge carries all
#         three), the T4 verdict, then the changed paths.
# Exit:   0 ok; 1 gh failed; 2 usage.
set -eo pipefail

if [ "$#" -ne 1 ] || ! [[ "$1" =~ ^[0-9]+$ ]]; then
  echo 'usage: review-request.sh <pr-number>' >&2
  exit 2
fi

json=$(gh pr view "$1" --json number,headRefName,headRefOid,files,statusCheckRollup,body,title,commits) \
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

# t4_verdict: reads "path<TAB>additions<TAB>deletions" lines. In order: a change to swarm/tests/ alone (PROTOCOL exempts
# it at any size); the first protected path; more than 15 files or 600 changed lines (additions plus deletions). Otherwise no.
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
  elif [ "$non_tests" -eq 0 ]; then
    echo "T4: no (swarm/tests only)"
  elif [ -n "$first" ]; then
    echo "T4: yes ($first)"
  elif [ "$files" -gt 15 ] || [ "$lines" -gt 600 ]; then
    echo "T4: yes (size: $files files, $lines lines)"
  else
    echo "T4: no"
  fi
}

# A closing word, then the issue it names: #n, owner/repo#n, or the issue's URL. GitHub closes the issue at a merge when
# any of the three is named after one of these words (retro-012 decision 1; #575 review).
closing_re='\b(close[sd]?|fix(e[sd])?|resolve[sd]?)[[:space:]]*:?[[:space:]]+((https?://github\.com/)?[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/issues/[0-9]+|[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+#[0-9]+|#[0-9]+)'

# closing_keyword <pr json>: names each closing phrase in the title, the body and the commit messages, and where it is.
# The line says none, or lists the phrases with their place and the words to remove them.
closing_keyword() {
  local found=() where text hit joined item
  for where in title body "commit message"; do
    case "$where" in
      title) text=$(printf '%s' "$1" | jq_lf -r '.title // ""') ;;
      body) text=$(printf '%s' "$1" | jq_lf -r '.body // ""') ;;
      *) text=$(printf '%s' "$1" | jq_lf -r '.commits[]? | (.messageHeadline // ""), (.messageBody // "")') ;;
    esac
    while IFS= read -r hit; do
      if [ -n "$hit" ]; then found+=("$hit in $where"); fi
    done < <(grep -oiE "$closing_re" <<<"$text" || true)
  done
  if [ "${#found[@]}" -eq 0 ]; then
    echo "closing keyword: none"
    return
  fi
  joined=""
  for item in "${found[@]}"; do
    joined+="${joined:+, }$item"
  done
  if [ "${#found[@]}" -gt 1 ]; then
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
closing_keyword "$json"
printf '%s' "$json" | jq_lf -r '.files[]? | [.path, (.additions // 0), (.deletions // 0)] | @tsv' | t4_verdict
printf '%s' "$json" | jq_lf -r '"files:", (.files[]? | "  " + .path)'
