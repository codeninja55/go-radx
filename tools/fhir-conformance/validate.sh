#!/usr/bin/env bash
#
# Conformance gate: validate go-radx's FHIR R4 (4.0.1) and R5 (5.0.0) marshalling with
# the official HL7 FHIR validator (validator_cli.jar from hapifhir/org.hl7.fhir.core). It
# does NOT validate a borrowed example corpus: per release it marshals go-radx-generated
# instances of the radiology + clinical workflow set (Patient, Encounter, ServiceRequest,
# ImagingStudy, Observation, DiagnosticReport, OperationOutcome, CapabilityStatement, and
# a Bundle that references them) and validates that JSON against the matching FHIR version,
# so a validator error reflects a real conformance defect in the generated code (PRD §11.1,
# M6a acceptance gate). The R4 and R5 fixtures are distinct because the release type spaces
# never mix and several resources differ on the wire (no CodeableReference in R4, R4
# ImagingStudy.modality is a Coding, R4 Encounter.class is a single Coding).
#
# The gate also validates a DELIBERATELY-INVALID negative fixture against each release and
# requires the validator to REJECT it. That proves the gate actually bites: a validator that
# silently passed everything would still fail here.
#
# Reproducibility: the validator is pinned. The jar version and its SHA-256 are recorded
# in tools/versions (fhir-validator.* keys); this script verifies the jar's checksum
# before use and downloads the pinned release asset only when no local jar is supplied.
# Generation reads only committed, checksum-verified inputs.
#
# Behaviour when Java or the validator jar is unavailable:
#   - On a developer machine: print a SKIP message and exit 0, so the suite is green
#     without a JDK or the (large) validator jar.
#   - In CI (CI=true): exit non-zero, so a missing gate cannot silently rot (PRD §11.1).
#
# Local override: set FHIR_VALIDATOR_JAR to an already-downloaded validator_cli.jar to
# skip the download; its checksum is still verified against tools/versions.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

VERSIONS_FILE="tools/versions"
NEGATIVE_FIXTURE="tools/fhir-conformance/negative/invalid-observation.json"

# read_pin extracts a `key=value` pin from tools/versions, failing closed if absent so a
# missing pin is a loud error, never a silent unpinned run.
read_pin() {
  local key="$1" value
  value="$(grep -E "^${key}=" "$VERSIONS_FILE" | head -1 | cut -d= -f2- | tr -d '[:space:]')"
  if [ -z "$value" ]; then
    echo "FAIL: pin '$key' not found in $VERSIONS_FILE" >&2
    exit 1
  fi
  printf '%s' "$value"
}

skip_or_fail() {
  local msg="$1"
  if [ "${CI:-}" = "true" ]; then
    echo "FAIL: $msg" >&2
    echo "FAIL: CI=true requires the HL7 FHIR validator; the conformance gate must run in CI." >&2
    exit 1
  fi
  echo "SKIP: $msg"
  exit 0
}

if ! command -v java >/dev/null 2>&1; then
  skip_or_fail "java not installed (install a JDK 17+ to run the FHIR conformance gate)"
fi

VALIDATOR_VERSION="$(read_pin fhir-validator.version)"
VALIDATOR_SHA256="$(read_pin fhir-validator.sha256)"
VALIDATOR_URL="https://github.com/hapifhir/org.hl7.fhir.core/releases/download/${VALIDATOR_VERSION}/validator_cli.jar"

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

# sha256_of prints the SHA-256 of a file using whichever tool the platform ships
# (shasum on macOS, sha256sum on Linux runners).
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

# Resolve the jar: a caller-supplied local jar, or a download of the pinned release
# asset. Either way the checksum is verified against the pin before the jar is trusted.
jar="${FHIR_VALIDATOR_JAR:-}"
if [ -z "$jar" ]; then
  if ! command -v curl >/dev/null 2>&1; then
    skip_or_fail "curl not installed and FHIR_VALIDATOR_JAR unset; cannot obtain the pinned validator jar"
  fi
  jar="${work_dir}/validator_cli.jar"
  echo "Downloading pinned validator_cli.jar ${VALIDATOR_VERSION} ..."
  if ! curl -sSL -f -o "$jar" "$VALIDATOR_URL"; then
    skip_or_fail "could not download $VALIDATOR_URL"
  fi
fi

if [ ! -f "$jar" ]; then
  echo "FAIL: validator jar not found at '$jar'" >&2
  exit 1
fi

got_sha="$(sha256_of "$jar")"
if [ "$got_sha" != "$VALIDATOR_SHA256" ]; then
  echo "FAIL: validator jar SHA-256 mismatch" >&2
  echo "  expected (tools/versions): $VALIDATOR_SHA256" >&2
  echo "  got:                       $got_sha" >&2
  echo "FAIL: refusing to run an unverified validator; bumping the pin is a deliberate change." >&2
  exit 1
fi
echo "Verified validator_cli.jar ${VALIDATOR_VERSION} (sha256 ${got_sha})."

# run_validator runs the validator over a path at a given FHIR version; it returns the
# validator's exit code (0 = no errors, non-zero = validation errors found). `-tx n/a`
# disables the external terminology server (tx.fhir.org) so the gate is deterministic and
# needs no network at validation time: the structural, cardinality, and invariant checks
# the workflow set and the negative fixture rely on do not require terminology expansion.
run_validator() {
  local path="$1" version="$2"
  java -jar "$jar" "$path" -version "$version" -level errors -tx n/a
}

# validate_release marshals one release's go-radx workflow set and validates it against the
# matching FHIR version, then asserts the negative fixture is REJECTED at that version. It
# sets the shared `status` to 1 on any failure rather than exiting, so both releases are
# always reported in one run.
validate_release() {
  local release="$1" version="$2" fixtures_pkg="$3"
  local fixtures_dir="${work_dir}/fixtures-${release}"

  echo "Marshalling go-radx ${release} workflow-set instances ..."
  go run "$fixtures_pkg" "$fixtures_dir"

  echo "Validating the ${release} workflow set against FHIR ${version} ..."
  if ! run_validator "$fixtures_dir" "$version"; then
    echo "FAIL: the HL7 FHIR validator reported errors on the go-radx ${release} workflow set." >&2
    status=1
  fi

  # The negative fixture MUST be rejected. If the validator passes it, the gate is not
  # actually validating and is therefore worthless — fail loudly.
  echo "Validating the negative fixture against ${version} (must be REJECTED) ..."
  if run_validator "$NEGATIVE_FIXTURE" "$version" >/dev/null 2>&1; then
    echo "FAIL: the negative fixture $NEGATIVE_FIXTURE passed ${release} validation; the gate is not biting." >&2
    status=1
  else
    echo "OK: the validator rejected the negative fixture at ${version} as expected (the gate bites)."
  fi
}

status=0
validate_release "r4" "4.0.1" "./tools/fhir-conformance/fixtures-r4"
validate_release "r5" "5.0.0" "./tools/fhir-conformance/fixtures"

if [ "$status" -ne 0 ]; then
  exit "$status"
fi
echo "conformance: the FHIR R4 and R5 workflow sets validated cleanly and the negative fixture was rejected at each version."
