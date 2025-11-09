#!/usr/bin/env bash
#
# download_r5_schemas.sh
#
# Downloads FHIR R5 schemas from HL7 official site and updates the README.
#
# Usage:
#   ./fhir/scripts/download_r5_schemas.sh [--version VERSION]
#
# Options:
#   --version VERSION   Specify FHIR R5 version (default: 5.0.0)
#
# Examples:
#   ./fhir/scripts/download_r5_schemas.sh
#   ./fhir/scripts/download_r5_schemas.sh --version 5.0.0
#

set -euo pipefail

# Script directory (resolve symlinks)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SCHEMAS_DIR="${PROJECT_ROOT}/fhir_schemas/r5"

# Default values
FHIR_VERSION="${FHIR_VERSION:-5.0.0}"
BASE_URL="https://hl7.org/fhir/R5"
SCHEMA_ZIP="fhir.schema.json.zip"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            FHIR_VERSION="$2"
            shift 2
            ;;
        -h|--help)
            grep '^#' "$0" | cut -c 3-
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}" >&2
            exit 1
            ;;
    esac
done

# Function to log messages
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Verify prerequisites
if ! command_exists curl; then
    log_error "curl is not installed. Please install curl and try again."
    exit 1
fi

if ! command_exists unzip; then
    log_error "unzip is not installed. Please install unzip and try again."
    exit 1
fi

# Create schemas directory if it doesn't exist
mkdir -p "${SCHEMAS_DIR}"

log_info "Downloading FHIR R5 schemas (version ${FHIR_VERSION})..."
log_info "Source: ${BASE_URL}/${SCHEMA_ZIP}"

# Download schemas
cd "${SCHEMAS_DIR}"

if curl -L -f -o "${SCHEMA_ZIP}" "${BASE_URL}/${SCHEMA_ZIP}"; then
    log_info "Download successful"
else
    log_error "Failed to download schemas from ${BASE_URL}/${SCHEMA_ZIP}"
    exit 1
fi

# Extract schemas
log_info "Extracting schemas..."
if unzip -o "${SCHEMA_ZIP}"; then
    log_info "Extraction successful"
else
    log_error "Failed to extract ${SCHEMA_ZIP}"
    exit 1
fi

# Clean up zip file
rm -f "${SCHEMA_ZIP}"

# Get current date
DOWNLOAD_DATE=$(date +"%B %d, %Y")

# Update README
log_info "Updating README.md..."
cat > README.md << EOF
# FHIR R5 Schemas

This directory contains FHIR R5 StructureDefinition files used for code generation.

## Files

- **profiles-resources.json** - FHIR R5 resource definitions (146 resources)
- **profiles-types.json** - FHIR R5 complex type definitions
- **profiles-others.json** - Other FHIR profiles (extensions, etc.)
- **valuesets.json** - FHIR value sets for coded elements
- **search-parameters.json** - FHIR search parameter definitions
- **conceptmaps.json** - FHIR concept maps
- **dataelements.json** - FHIR data element definitions

## Source

- **FHIR Version**: R5 (${FHIR_VERSION})
- **Source URL**: ${BASE_URL}/
- **Download Date**: ${DOWNLOAD_DATE}
- **Schema Bundle**: ${BASE_URL}/${SCHEMA_ZIP}

## Usage

These schemas are used by the FHIR code generator to produce type-safe Go structs:

\`\`\`bash
# Generate R5 resources
./fhir/scripts/gen/bin/fhirgen -version r5 \\
  -input fhir_schemas/r5/profiles-resources.json \\
  -output fhir/r5/resources \\
  -verbose

# Generate R5 complex types
./fhir/scripts/gen/bin/fhirgen -version r5 \\
  -input fhir_schemas/r5/profiles-types.json \\
  -output fhir/r5/types \\
  -verbose
\`\`\`

## Updating Schemas

To update to a newer R5 version:

\`\`\`bash
# Run the download script
./fhir/scripts/download_r5_schemas.sh

# Or specify a specific version
./fhir/scripts/download_r5_schemas.sh --version 5.0.0
\`\`\`

## Version Control

These schema files are tracked in version control to ensure:
- Reproducible builds across environments
- Ability to regenerate code with identical schemas
- Clear documentation of which FHIR version is being used
EOF

log_info "README.md updated"

# Verify critical files exist
REQUIRED_FILES=("profiles-resources.json" "profiles-types.json")
for file in "${REQUIRED_FILES[@]}"; do
    if [[ ! -f "${file}" ]]; then
        log_error "Required file ${file} was not downloaded"
        exit 1
    fi
done

# Show summary
log_info "✓ FHIR R5 schemas successfully downloaded and extracted"
log_info "  Version: ${FHIR_VERSION}"
log_info "  Location: ${SCHEMAS_DIR}"
log_info "  Files:"
for file in profiles-*.json valuesets.json search-parameters.json conceptmaps.json dataelements.json; do
    if [[ -f "${file}" ]]; then
        SIZE=$(du -h "${file}" | cut -f1)
        log_info "    - ${file} (${SIZE})"
    fi
done

log_info ""
log_info "Next steps:"
log_info "  1. Rebuild the FHIR code generator: cd fhir/scripts/gen && go build -o bin/fhirgen ."
log_info "  2. Regenerate R5 resources: ./fhir/scripts/gen/bin/fhirgen -version r5 -input fhir_schemas/r5/profiles-resources.json -output fhir/r5/resources -verbose"
log_info "  3. Regenerate R5 types: ./fhir/scripts/gen/bin/fhirgen -version r5 -input fhir_schemas/r5/profiles-types.json -output fhir/r5/types -verbose"
