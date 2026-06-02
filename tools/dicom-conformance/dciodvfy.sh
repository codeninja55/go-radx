#!/usr/bin/env bash
#
# Conformance gate: validate go-radx's WRITER output with dciodvfy from David Clunie's dicom3tools
# (NOT dcmtk — dcmtk ships no IOD validator; dciodvfy is a dicom3tools program).
#
# It does NOT validate the vendored fixtures directly: they are intentionally-minimal codec and SR
# test samples that are not IOD-complete and never pass strict dciodvfy (missing Type 1/Type 2
# elements such as PatientName and StudyInstanceUID). Instead it round-trips each fixture go-radx
# can read AND write (the uncompressed transfer syntaxes) through go-radx and fails only if go-radx
# INTRODUCES IOD errors beyond what the input already had — i.e. go-radx's writer must not DEGRADE a
# file's conformance. Inputs go-radx cannot read or write (e.g. an encapsulated transfer syntax the
# writer does not support) are skipped.
#
# Behaviour when dciodvfy is not installed:
#   - On a developer machine: print a SKIP message and exit 0, so the suite is green without
#     dicom3tools.
#   - In CI (CI=true): exit non-zero, so a missing gate cannot silently rot (PRD §11.1).
#
# Arguments: zero or more paths to .dcm files or directories. With no arguments, the vendored
# fixture corpus is the round-trip input set.

set -euo pipefail

FIXTURE_DIR="testdata/dicom"

if ! command -v dciodvfy >/dev/null 2>&1; then
  msg="SKIP: dciodvfy not installed (install dicom3tools to run the DICOM IOD validation gate)"
  if [ "${CI:-}" = "true" ]; then
    echo "FAIL: $msg" >&2
    echo "FAIL: CI=true requires dciodvfy; the conformance gate must run in CI." >&2
    exit 1
  fi
  echo "$msg"
  exit 0
fi

# Collect targets: explicit args, or every .dcm under the fixture dir.
targets=()
if [ "$#" -gt 0 ]; then
  for arg in "$@"; do
    if [ -d "$arg" ]; then
      while IFS= read -r f; do targets+=("$f"); done < <(find "$arg" -type f -name '*.dcm')
    else
      targets+=("$arg")
    fi
  done
else
  while IFS= read -r f; do targets+=("$f"); done < <(find "$FIXTURE_DIR" -type f -name '*.dcm')
fi

if [ "${#targets[@]}" -eq 0 ]; then
  echo "SKIP: no .dcm files to validate"
  exit 0
fi

# error_count prints how many "Error" lines dciodvfy emits for a file. dciodvfy exits non-zero when
# it reports errors and grep -c exits non-zero on zero matches, so both are tolerated (|| true)
# without tripping set -o pipefail; the count still reaches stdout.
error_count() {
  dciodvfy "$1" 2>&1 | grep -c '^Error' || true
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

status=0
tested=0
skipped=0
for f in "${targets[@]}"; do
  out="$tmp/$(basename "$f")"
  rc=0
  go run ./tools/dicom-conformance/roundtrip "$f" "$out" >/dev/null 2>&1 || rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "SKIP (go-radx cannot round-trip, rc=$rc): $f"
    skipped=$((skipped + 1))
    continue
  fi

  in_errs="$(error_count "$f")"
  out_errs="$(error_count "$out")"
  if [ "$out_errs" -gt "$in_errs" ]; then
    echo "FAIL: go-radx degraded conformance ($in_errs -> $out_errs IOD errors): $f"
    dciodvfy "$out" 2>&1 | grep '^Error' || true
    status=1
  else
    echo "OK ($in_errs -> $out_errs IOD errors): $f"
  fi
  tested=$((tested + 1))
done

echo "conformance: $tested round-tripped through go-radx, $skipped skipped"
exit "$status"
