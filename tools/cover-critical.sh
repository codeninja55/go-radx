#!/usr/bin/env bash
#
# Critical-path coverage gate. The aggregate 80% floor (tools/cover-check.sh, PRD §11.4) is a floor
# on the whole module; this gate adds a per-package 90% target on the critical paths — the parsers,
# codecs, transport, and conversion core that handle untrusted input or carry conversion correctness,
# where a coverage hole is most consequential. The critical-path set is the standing contract fixed in
# docs/conformance/cross-cutting.md ("Coverage targets and critical-path enumeration").
#
# It reads the SAME merged profile cover-check.sh reads — the single
# `go test -race -covermode=atomic -coverpkg=./... ./...` run the lint-test job already produces — and
# computes per-package UNION coverage (every test binary's contribution to that package's own
# statements, generated files excluded). Measuring from the merged profile, not a per-package
# `go test -cover`, credits cross-package exercise (e.g. the convert suite exercising dicom) the same
# way the 80% floor does, and avoids a second expensive race run.
#
# Honesty over a green dashboard. Raising every critical-path package to 90% is a larger test effort
# than one increment; rather than silently exclude the packages still short of 90% or lower the bar,
# the gate splits the set in two:
#
#   * ENFORCED — packages at or above 90% today. The gate FAILS if any drops below 90%. This is the
#     real, biting 90% contract.
#   * RATCHET  — packages still being brought up to 90% (a documented TODO, mirrored in
#     cross-cutting.md). Each carries a recorded baseline; the gate FAILS if one regresses more than
#     the jitter tolerance below its baseline (see RATCHET_TOLERANCE below), so a TODO package can only
#     move toward 90%, never meaningfully backslide. When a RATCHET package reaches 90% the gate prints
#     a PROMOTE notice: move it into ENFORCED (and update the doc) so the 90% contract binds it from
#     then on.
#
# So the gate enforces 90% on the ENFORCED subset now and ratchets the rest upward; it never lowers
# the 90% bar and never hides a short package.
#
# Ratchet jitter tolerance. Union coverage from `go test -race -coverpkg=./...` is NOT bit-stable
# run-to-run: which tests happen to exercise cross-package code shifts slightly under -race timing, so
# a per-package number can wobble by a fraction of a point between identical runs. A RATCHET baseline
# pinned to the exact measured value would then false-fail on that noise (it once failed on a 0.1pp
# drop). So a RATCHET package fails only when it falls more than RATCHET_TOLERANCE (0.5 percentage
# points) below its baseline — enough to absorb measurement jitter, still tight enough to catch a real
# regression (>0.5pp). This tolerance applies ONLY to the per-package no-regression baseline. The 90%
# target itself gets NO tolerance: it is a fixed bar, and ENFORCED packages must be >= 90.0 exactly.
#
# Inputs:
#   $1            profile path  (default: coverage.out — the profile cover:check already wrote)
#   CRIT_TARGET   critical-path target percent (default: 90.0)
#
# Exit 0 when every ENFORCED package meets the target and no RATCHET package regressed beyond the
# tolerance; exit 1 (naming the offending packages) otherwise, or if the profile is missing.

set -euo pipefail

PROFILE="${1:-coverage.out}"
TARGET="${CRIT_TARGET:-90.0}"
RATCHET_TOLERANCE="0.5"

if [ ! -f "$PROFILE" ]; then
  echo "FAIL: coverage profile '$PROFILE' not found; run the coverage test step (mise run cover) first." >&2
  exit 1
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
MODULE="$(awk '/^module /{print $2; exit}' "$REPO_ROOT/go.mod")"

# --- Critical-path enumeration (cross-cutting.md "Coverage targets and critical-path enumeration") ---
#
# The untrusted-input parsers/codecs/transport plus the conversion correctness core. The generated
# FHIR R4/R5 trees (fhir/r4, fhir/r5) are deliberately EXCLUDED — they are byte-for-byte generated and
# gated by the regeneration test, not by unit coverage — as are pure glue/auth-helper packages
# (dicomweb/auth/*, server, logging). "fhir" here is the hand-written validate/decode core (the
# top-level package: validate.go, resource.go, primitive.go, binding.go, ...), not the generated tree.
#
# ENFORCED: at or below this list the 90% target BITES. RATCHET: TODO packages with their recorded
# baseline; the gate fails on a regression below the baseline and prints a promote notice at >= target.
# Baselines were measured on the merged -race -coverpkg=./... profile with generated files excluded;
# re-measure with `CRIT_TARGET=90 tools/cover-critical.sh <profile>` (it prints every package's number)
# and bump a baseline only when raising it, never to paper over a regression.
ENFORCED_PKGS=(
  "dimse/dul"
)
# pkg=baseline_percent. Baselines are the EXACT measured union coverage (one decimal, the precision
# `go tool cover` reports), NOT rounded down: a baseline below the measured value would let coverage
# backslide within the rounding gap without failing, which contradicts the "fail on any regression
# below the recorded baseline" contract. Mirrored in docs/conformance/cross-cutting.md ("Coverage
# targets and critical-path enumeration"). Re-measure with `CRIT_TARGET=90 tools/cover-critical.sh`.
RATCHET_PKGS=(
  "dicom=85.8"
  "dimse=83.8"
  "dimse/acse=83.4"
  "dimse/pdu=89.7"
  "dicomweb=80.1"
  "hl7v2=89.9"
  "convert=81.8"
  "fhir=88.7"
)

# Build the generated-file exclusion set (module-qualified, the form the profile's file column uses),
# identical to cover-check.sh so the two gates agree on what counts as hand-written.
GENERATED_LIST="$(mktemp)"
trap 'rm -f "$GENERATED_LIST"' EXIT
( cd "$REPO_ROOT" && grep -rlI --include='*.go' 'Code generated .* DO NOT EDIT\.' . 2>/dev/null \
  | sed "s#^\./#${MODULE}/#" ) | sort -u > "$GENERATED_LIST"

# pkg_coverage prints the union statement-coverage percent (no % sign) of package $1's own files in the
# profile, generated files excluded, or the empty string if the package contributes no profiled
# statements. "Own files" = the file column begins with MODULE/pkg/ and the remainder names a file
# directly in that directory (no further "/"), so a subpackage is never folded into its parent.
pkg_coverage() {
  local pkg="$1"
  local prefix="${MODULE}/${pkg}/"
  local tmp
  tmp="$(mktemp)"
  {
    echo "mode: atomic"
    awk -v pre="$prefix" -v genf="$GENERATED_LIST" '
      BEGIN { while ((getline f < genf) > 0) gen[f] = 1 }
      NR == 1 { next }
      {
        file = $1; sub(/:.*/, "", file)
        if (index(file, pre) == 1) {
          rest = substr(file, length(pre) + 1)
          if (index(rest, "/") == 0 && !(file in gen)) print
        }
      }
    ' "$PROFILE"
  } > "$tmp"
  # A package with no profiled (non-generated) statements yields only the mode line; report empty.
  if [ "$(wc -l < "$tmp")" -le 1 ]; then
    rm -f "$tmp"
    echo ""
    return
  fi
  local field
  field="$(go tool cover -func="$tmp" | tail -1 | awk '{print $NF}')"
  rm -f "$tmp"
  echo "${field%\%}"
}

# below a b  -> exit 0 (true) when a < b, comparing as decimals (bash arithmetic is integer-only).
below() { awk -v a="$1" -v b="$2" 'BEGIN { exit !(a < b) }'; }

fail=0

echo "Critical-path coverage gate (target ${TARGET}%, generated files excluded):"

echo "  ENFORCED (90% bites):"
for pkg in "${ENFORCED_PKGS[@]}"; do
  cov="$(pkg_coverage "$pkg")"
  if [ -z "$cov" ]; then
    echo "    FAIL ${pkg}: no profiled statements found — is the package in the profile's -coverpkg set?" >&2
    fail=1
    continue
  fi
  if below "$cov" "$TARGET"; then
    echo "    FAIL ${pkg}: ${cov}% is below the ${TARGET}% critical-path target." >&2
    fail=1
  else
    echo "    ok   ${pkg}: ${cov}%"
  fi
done

echo "  RATCHET (TODO toward 90%; must not regress more than ${RATCHET_TOLERANCE}pp below baseline):"
for entry in "${RATCHET_PKGS[@]}"; do
  pkg="${entry%%=*}"
  base="${entry#*=}"
  # Floor = baseline minus the jitter tolerance; a regression FAILS only when current < floor, so
  # sub-0.5pp run-to-run noise is absorbed while a real >0.5pp drop still bites.
  floor="$(awk -v b="$base" -v t="$RATCHET_TOLERANCE" 'BEGIN { printf "%.4f", b - t }')"
  cov="$(pkg_coverage "$pkg")"
  if [ -z "$cov" ]; then
    echo "    FAIL ${pkg}: no profiled statements found — is the package in the profile's -coverpkg set?" >&2
    fail=1
    continue
  fi
  if below "$cov" "$floor"; then
    echo "    FAIL ${pkg}: ${cov}% regressed more than ${RATCHET_TOLERANCE}pp below its recorded baseline ${base}% — restore coverage or justify lowering the baseline." >&2
    fail=1
  elif ! below "$cov" "$TARGET"; then
    echo "    PROMOTE ${pkg}: ${cov}% now meets the ${TARGET}% target — move it into ENFORCED_PKGS and update cross-cutting.md." >&2
    fail=1
  else
    echo "    ok   ${pkg}: ${cov}% (within ${RATCHET_TOLERANCE}pp of baseline ${base}%, still below ${TARGET}% target)"
  fi
done

if [ "$fail" -ne 0 ]; then
  echo "FAIL: critical-path coverage gate failed (see lines above)." >&2
  exit 1
fi

echo "OK: every ENFORCED critical-path package meets ${TARGET}% and no RATCHET package regressed more than ${RATCHET_TOLERANCE}pp below its baseline."
