#!/bin/sh
# Watches the versions Dependabot does not see: the Go of go.mod, which every
# workflow builds with, and the golangci-lint pinned in ci.yml. When a newer
# one is out, it opens one issue, or updates the one already open; when all
# is up to date again, it closes it. News never fails the run.
#
#   scripts/versions.sh             # in the Versions workflow, with GH_TOKEN
#   DRY_RUN=1 scripts/versions.sh   # prints what it would do
#
# GO_LATEST and LINT_LATEST, when set, stand for the latest releases, so the
# script can be tried without the network.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
title="Versions to raise"

go_have=$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' "$root/go.mod")
lint_have=$(sed -n 's/^ *version: \(v[0-9][0-9.]*\).*/\1/p' "$root/.github/workflows/ci.yml")

if [ -z "$go_have" ] || [ -z "$lint_have" ]; then
  echo "cannot read the versions in use: go '$go_have', golangci-lint '$lint_have'" >&2
  exit 1 # the files changed shape: this script must follow
fi

go_last=${GO_LATEST:-$(curl -fsSL 'https://go.dev/dl/?mode=json' |
  jq -r '[.[] | select(.stable)][0].version' | sed 's/^go//')}
lint_last=${LINT_LATEST:-$(gh release view --repo golangci/golangci-lint --json tagName -q .tagName)}

body=""
if [ "$go_have" != "$go_last" ]; then
  body="$body- Go: $go_have in go.mod, $go_last released. Raise the go line of go.mod; every workflow follows it.
"
fi
if [ "$lint_have" != "$lint_last" ]; then
  body="$body- golangci-lint: $lint_have in .github/workflows/ci.yml, $lint_last released. Raise the version of the lint job.
"
fi

echo "Go $go_have (latest $go_last), golangci-lint $lint_have (latest $lint_last)"

open=""
if [ -z "${DRY_RUN:-}" ]; then
  open=$(gh issue list --state open --search "in:title \"$title\"" --json number,title \
    -q ".[] | select(.title == \"$title\") | .number" | head -n 1)
fi

run() {
  if [ -n "${DRY_RUN:-}" ]; then
    echo "would run: $*"
  else
    "$@"
  fi
}

if [ -n "$body" ]; then
  body="Newer releases than the ones in use, seen on $(date -u +%Y-%m-%d):

$body
Opened by the Versions workflow (scripts/versions.sh), which closes this issue once all is up to date."
  if [ -n "$open" ]; then
    run gh issue edit "$open" --body "$body"
  else
    run gh issue create --title "$title" --body "$body"
  fi
elif [ -n "$open" ]; then
  run gh issue close "$open" --comment "All up to date: Go $go_have, golangci-lint $lint_have."
fi
