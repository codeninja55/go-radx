#!/usr/bin/env bash
# run-fuzz-tests.sh - Run all fuzz tests across DIMSE packages
#
# This script runs fuzz tests for the DIMSE implementation, suitable for
# CI/CD environments or local extended testing.
#
# Usage:
#   ./scripts/run-fuzz-tests.sh [duration]
#
# Arguments:
#   duration - Fuzzing duration per test (default: 30s)
#              Examples: 10s, 1m, 5m
#
# Environment variables:
#   FUZZ_TIME     - Override default fuzz duration
#   PARALLEL      - Number of parallel workers (default: auto)
#   VERBOSE       - Enable verbose output (default: false)

set -euo pipefail

# Configuration
FUZZ_TIME="${1:-${FUZZ_TIME:-30s}}"
PARALLEL="${PARALLEL:-}"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_ROOT"

log_info "Starting fuzz tests with duration: $FUZZ_TIME"
log_info "Project root: $PROJECT_ROOT"

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
declare -a FAILED_TEST_NAMES

# Function to run fuzz test
run_fuzz_test() {
    local package=$1
    local test_name=$2

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    log_info "Running $test_name in $package..."

    # Build command
    local cmd="go test"
    if [[ "$VERBOSE" == "true" ]]; then
        cmd="$cmd -v"
    fi
    cmd="$cmd -fuzz=$test_name -fuzztime=$FUZZ_TIME"
    if [[ -n "$PARALLEL" ]]; then
        cmd="$cmd -parallel=$PARALLEL"
    fi

    # Run test and capture result
    if (cd "$package" && eval "$cmd"); then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_info "✓ $test_name passed"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_TEST_NAMES+=("$package:$test_name")
        log_error "✗ $test_name failed"
    fi

    echo ""
}

# PDU Layer Fuzz Tests
log_info "========== PDU Layer Fuzz Tests =========="
PDU_PKG="dimse/pdu"
run_fuzz_test "$PDU_PKG" "FuzzAssociateRQDecode"
run_fuzz_test "$PDU_PKG" "FuzzAssociateACDecode"
run_fuzz_test "$PDU_PKG" "FuzzAssociateRJDecode"
run_fuzz_test "$PDU_PKG" "FuzzDataTFDecode"
run_fuzz_test "$PDU_PKG" "FuzzReleaseRQDecode"
run_fuzz_test "$PDU_PKG" "FuzzReleaseRPDecode"
run_fuzz_test "$PDU_PKG" "FuzzAbortDecode"
run_fuzz_test "$PDU_PKG" "FuzzPDUType"
run_fuzz_test "$PDU_PKG" "FuzzPDUSizeLimit"

# DIMSE Layer Fuzz Tests
log_info "========== DIMSE Layer Fuzz Tests =========="
DIMSE_PKG="dimse/dimse"
run_fuzz_test "$DIMSE_PKG" "FuzzCommandSetFromDataSet"
run_fuzz_test "$DIMSE_PKG" "FuzzCommandSetRoundTrip"
run_fuzz_test "$DIMSE_PKG" "FuzzMessageEncoding"
run_fuzz_test "$DIMSE_PKG" "FuzzReassemblerAddPDU"
run_fuzz_test "$DIMSE_PKG" "FuzzStatusCodeHandling"
run_fuzz_test "$DIMSE_PKG" "FuzzMessageFragmentation"

# DUL/State Machine Fuzz Tests
log_info "========== State Machine Fuzz Tests =========="
DUL_PKG="dimse/dul"
run_fuzz_test "$DUL_PKG" "FuzzStateMachineEventSequence"
run_fuzz_test "$DUL_PKG" "FuzzStateMachineInvalidEvents"
run_fuzz_test "$DUL_PKG" "FuzzStateMachineStates"
run_fuzz_test "$DUL_PKG" "FuzzStateMachineConcurrent"
run_fuzz_test "$DUL_PKG" "FuzzStateMachineTransitionInvariants"
run_fuzz_test "$DUL_PKG" "FuzzStateMachineIdempotency"

# Summary
log_info "=========================================="
log_info "Fuzz Test Summary:"
log_info "  Total:  $TOTAL_TESTS"
log_info "  Passed: $PASSED_TESTS"
log_info "  Failed: $FAILED_TESTS"

if [[ $FAILED_TESTS -gt 0 ]]; then
    log_error "Failed tests:"
    for test in "${FAILED_TEST_NAMES[@]}"; do
        log_error "  - $test"
    done
    exit 1
fi

log_info "All fuzz tests passed!"
exit 0
