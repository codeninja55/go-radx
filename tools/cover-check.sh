#!/usr/bin/env bash
#
# Coverage floor gate: fail if aggregate statement coverage of the root module falls below the floor
# the PRD fixes (§11.4). The race-test step writes a single merged profile with -coverpkg=./... so
# every statement in the module is in the denominator regardless of which test binary exercises it;
# this script reads that profile, extracts the `total:` line `go tool cover -func` prints, and
# compares it to the floor.
#
# Generated code is excluded from the floor. Every byte under fhir/r4 and fhir/r5 is produced by the
# FHIR code generator and never hand-edited; it carries the standard `// Code generated ... DO NOT
# EDIT.` banner and is gated by the byte-for-byte regeneration test, not by unit coverage. The floor
# measures hand-written code, so generated files are filtered out of the profile before the total is
# computed. This excludes the bulk-generated resource tree (tens of thousands of mechanical
# MarshalJSON/UnmarshalJSON statements) without lowering the 80% floor on the code people write.
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

REPO_ROOT="$(git rev-parse --show-toplevel)"

# Build the set of generated files (those carrying the standard "Code generated ... DO NOT EDIT."
# banner) as the module-qualified paths the profile uses, so their coverage blocks can be filtered
# out. The module path prefix is read from go.mod so the filter matches the profile's file column.
MODULE="$(awk '/^module /{print $2; exit}' "$REPO_ROOT/go.mod")"
GENERATED_LIST="$(mktemp)"
trap 'rm -f "$GENERATED_LIST" "$FILTERED_PROFILE"' EXIT
# grep -rl finds files containing the banner; map each repo-relative path to its module-qualified
# form (the form the cover profile's first column uses).
( cd "$REPO_ROOT" && grep -rlI --include='*.go' 'Code generated .* DO NOT EDIT\.' . 2>/dev/null \
  | sed "s#^\./#${MODULE}/#" ) | sort -u > "$GENERATED_LIST"

# Filter generated-file blocks out of the profile. A profile line is "<file>:<span> <stmts> <count>";
# its file column is the substring before the first colon, matched against the generated set.
FILTERED_PROFILE="$(mktemp)"
awk -v genlist="$GENERATED_LIST" '
  BEGIN { while ((getline f < genlist) > 0) gen[f] = 1 }
  NR == 1 && $0 ~ /^mode:/ { print; next }
  { file = $1; sub(/:.*/, "", file); if (!(file in gen)) print }
' "$PROFILE" > "$FILTERED_PROFILE"

# `go tool cover -func` prints one row per function and a trailing `total:` row whose last column is
# the aggregate percentage with a trailing %. Take that last field of the last line and strip the %.
total_field="$(go tool cover -func="$FILTERED_PROFILE" | tail -1 | awk '{print $NF}')"
total="${total_field%\%}"

if [ -z "$total" ]; then
  echo "FAIL: could not parse a total coverage percentage from '$PROFILE'." >&2
  exit 1
fi

# Compare as decimals with awk; bash arithmetic is integer-only and these are fractional percents.
if awk -v t="$total" -v f="$FLOOR" 'BEGIN { exit !(t < f) }'; then
  echo "FAIL: aggregate coverage ${total}% (hand-written code, generated files excluded) is below the ${FLOOR}% floor (PRD §11.4)." >&2
  exit 1
fi

echo "OK: aggregate coverage ${total}% (hand-written code, generated files excluded) meets the ${FLOOR}% floor (PRD §11.4)."
