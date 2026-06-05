#!/usr/bin/env bash
#
# Coverage floor gate: fail if aggregate statement coverage of the root module falls below the floor
# the PRD fixes (§11.4). The race-test step writes a single merged profile with -coverpkg=./... so
# every statement in the module is in the denominator regardless of which test binary exercises it;
# this script reads that profile, extracts the `total:` line `go tool cover -func` prints, and
# compares it to the floor.
#
# Inputs:
#   $1            profile path  (default: coverage.out)
#   COVER_FLOOR   floor percent (default: 80.0)
#
# Exit 0 when total >= floor; exit 1 (printing total and floor) otherwise, or if the profile is
# missing or yields no total line.

set -euo pipefail

PROFILE="${1:-coverage.out}"
FLOOR="${COVER_FLOOR:-80.0}"

if [ ! -f "$PROFILE" ]; then
  echo "FAIL: coverage profile '$PROFILE' not found; run the coverage test step first." >&2
  exit 1
fi

# `go tool cover -func` prints one row per function and a trailing `total:` row whose last column is
# the aggregate percentage with a trailing %. Take that last field of the last line and strip the %.
total_field="$(go tool cover -func="$PROFILE" | tail -1 | awk '{print $NF}')"
total="${total_field%\%}"

if [ -z "$total" ]; then
  echo "FAIL: could not parse a total coverage percentage from '$PROFILE'." >&2
  exit 1
fi

# Compare as decimals with awk; bash arithmetic is integer-only and these are fractional percents.
if awk -v t="$total" -v f="$FLOOR" 'BEGIN { exit !(t < f) }'; then
  echo "FAIL: aggregate coverage ${total}% is below the ${FLOOR}% floor (PRD §11.4)." >&2
  exit 1
fi

echo "OK: aggregate coverage ${total}% meets the ${FLOOR}% floor (PRD §11.4)."
