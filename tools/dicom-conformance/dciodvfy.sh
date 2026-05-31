#!/usr/bin/env bash
#
# Validate DICOM files with dcmtk's dciodvfy.
#
# Behaviour when dciodvfy is not installed:
#   - On a developer machine: print a SKIP message and exit 0, so the suite is
#     green without dcmtk.
#   - In CI (CI=true): exit non-zero, so a missing gate cannot silently rot
#     (PRD §11.1).
#
# Arguments: zero or more paths to .dcm files or directories. With no
# arguments, the vendored fixture corpus is validated.

set -euo pipefail

FIXTURE_DIR="testdata/dicom"

if ! command -v dciodvfy >/dev/null 2>&1; then
  msg="SKIP: dciodvfy not installed (install dcmtk to run the DICOM IOD validation gate)"
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

status=0
for f in "${targets[@]}"; do
  echo "dciodvfy: $f"
  if ! dciodvfy "$f"; then
    status=1
  fi
done

exit "$status"
