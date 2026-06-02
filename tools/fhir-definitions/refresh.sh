#!/usr/bin/env bash
#
# Refresh-only download of the vendored HL7 FHIR StructureDefinition bundle.
#
# This is the ONE deliberate network fetch in the FHIR generator: it re-downloads
# the official HL7 definitions archive, extracts the three bundle files the
# generator reads, copies them into the vendored definitions directory, and
# re-records the SHA-256 manifest. Run it deliberately when bumping the pinned
# FHIR release — NEVER at generate time. Generation reads only the committed,
# checksum-verified bundle and never touches the network.
#
# Usage:
#   tools/fhir-definitions/refresh.sh [r5|r4]
#
# Default release is r5.

set -euo pipefail

RELEASE="${1:-r5}"

case "$RELEASE" in
  r5)
    FHIR_URL="https://hl7.org/fhir/R5/definitions.json.zip"
    EXPECT_VERSION="5.0.0"
    ;;
  r4)
    FHIR_URL="https://hl7.org/fhir/R4/definitions.json.zip"
    EXPECT_VERSION="4.0.1"
    ;;
  *)
    echo "FAIL: unknown release '$RELEASE' (want r5 or r4)" >&2
    exit 1
    ;;
esac

REPO_ROOT="$(git rev-parse --show-toplevel)"
DEST_DIR="${REPO_ROOT}/fhir/internal/gen/testdata/definitions/${RELEASE}"
REQUIRED_FILES=("profiles-types.json" "profiles-resources.json" "valuesets.json")

for cmd in curl unzip shasum git; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "FAIL: required command '$cmd' not installed" >&2
    exit 1
  fi
done

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Downloading $FHIR_URL ..."
curl -sSL -f -o "${WORK_DIR}/definitions.json.zip" "$FHIR_URL"

echo "Extracting ..."
unzip -o -q "${WORK_DIR}/definitions.json.zip" -d "${WORK_DIR}/unzipped"

# Fail closed unless the archive is the expected pinned release version.
if [ -f "${WORK_DIR}/unzipped/version.info" ]; then
  GOT_VERSION="$(grep -E '^FhirVersion=' "${WORK_DIR}/unzipped/version.info" | head -1 | cut -d= -f2 | tr -d '[:space:]')"
  if [ "$GOT_VERSION" != "$EXPECT_VERSION" ]; then
    echo "FAIL: downloaded FHIR version '$GOT_VERSION' != expected '$EXPECT_VERSION'" >&2
    echo "FAIL: bumping the pinned release is a deliberate, reviewed change; edit EXPECT_VERSION here first." >&2
    exit 1
  fi
  echo "Verified FHIR version $GOT_VERSION."
else
  echo "FAIL: archive has no version.info; refusing to vendor an unverifiable bundle" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"
for f in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "${WORK_DIR}/unzipped/${f}" ]; then
    echo "FAIL: archive missing required file ${f}" >&2
    exit 1
  fi
  cp "${WORK_DIR}/unzipped/${f}" "${DEST_DIR}/${f}"
done

echo "Recording SHA256SUMS ..."
(
  cd "$DEST_DIR"
  shasum -a 256 "${REQUIRED_FILES[@]}" | sort -k2 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)

echo
echo "Refreshed ${RELEASE} bundle in ${DEST_DIR}."
echo "Review the diff and update SOURCE.md (version / build / date) before committing."
