#!/bin/sh
# The deep run of the tests: the comparisons with an exhaustive search and
# the walks over random problems, on many more problems than an ordinary
# run, and on other ones every week. Each of those tests draws its problems
# from seeds (planning.DeepSeeds): DEEP_SEEDS of them, from DEEP_FROM, which
# this script takes from the week of the year so that a week never plays the
# problems of the week before. A failure names its seed, and the command
# printed below plays that one again.
#
# When it fails, it opens one issue, or updates the one already open; when
# it passes again, it closes it.
#
#   scripts/deep.sh                    # in the Deep workflow, with GH_TOKEN
#   DRY_RUN=1 scripts/deep.sh          # runs the tests, and prints what it would do with the issue
#   DEEP_SEEDS=50 DRY_RUN=1 scripts/deep.sh
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
title="Deep tests failed"

week=$(date -u +%G%V) # such as 202647
seeds=${DEEP_SEEDS:-2000}
from=${DEEP_FROM:-$(( (week % 1000) * 100000 ))}
log=$(mktemp)

echo "Deep run: $seeds problems per test, from seed $from (week $week)"

status=0
(cd "$root" && DEEP_SEEDS=$seeds DEEP_FROM=$from go test -count=1 -timeout 300m ./...) >"$log" 2>&1 || status=$?

# The failures, and nothing of the thousands of subtests that passed.
grep -v -e '^=== ' -e '--- PASS' -e '^PASS$' "$log" | tail -n 200 || true

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

if [ "$status" -ne 0 ]; then
  failed=$(grep -o -e '--- FAIL: [^ ]*' "$log" | sed 's/--- FAIL: //' | sort -u | head -n 30)
  body="The deep run of $(date -u +%Y-%m-%d) failed: $seeds problems per test, from seed $from.

Failing:

\`\`\`
$failed
\`\`\`

To play one again, with the seed its message names:

\`\`\`sh
DEEP_FROM=<seed> DEEP_SEEDS=1 go test -run '<the test>' ./planning/
\`\`\`

The whole output is in the run of the Deep workflow. Opened by scripts/deep.sh, which closes this issue once a deep run passes."
  if [ -n "$open" ]; then
    run gh issue edit "$open" --body "$body"
  else
    run gh issue create --title "$title" --body "$body"
  fi

  rm -f "$log"
  exit "$status"
fi

if [ -n "$open" ]; then
  run gh issue close "$open" --comment "The deep run of $(date -u +%Y-%m-%d) passed: $seeds problems per test, from seed $from."
fi

rm -f "$log"
echo "Deep run passed."
