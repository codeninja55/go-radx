# DIMSE Testing Infrastructure

This document describes the comprehensive testing strategy for the DIMSE implementation, including fuzz testing for robustness and integration testing against real PACS systems.

## Overview

The testing infrastructure consists of three layers:

1. **Unit Tests**: Standard Go tests covering individual components
2. **Fuzz Tests**: Property-based testing for protocol robustness and security
3. **Integration Tests**: Real-world testing against Orthanc PACS

## Fuzz Testing

Fuzz testing uses Go's native fuzzing framework (Go 1.18+) to discover bugs, crashes, and security issues by feeding random/malformed input to protocol decoders and state machines.

### Why Fuzz Testing?

- **Security**: Discovers buffer overflows, DoS vulnerabilities, resource exhaustion
- **Robustness**: Tests edge cases that manual testing might miss
- **Protocol Conformance**: Validates proper handling of malformed DICOM/DIMSE messages
- **Crash Prevention**: Ensures code never panics on unexpected input

### Fuzz Test Coverage

#### PDU Layer (9 fuzz functions)

Location: `dimse/pdu/fuzz_test.go`

- `FuzzAssociateRQDecode`: A-ASSOCIATE-RQ PDU decoder
- `FuzzAssociateACDecode`: A-ASSOCIATE-AC PDU decoder
- `FuzzAssociateRJDecode`: A-ASSOCIATE-RJ PDU decoder
- `FuzzDataTFDecode`: P-DATA-TF PDU decoder
- `FuzzReleaseRQDecode`: A-RELEASE-RQ PDU decoder
- `FuzzReleaseRPDecode`: A-RELEASE-RP PDU decoder
- `FuzzAbortDecode`: A-ABORT PDU decoder
- `FuzzPDUType`: PDU type identification with invalid types
- `FuzzPDUSizeLimit`: PDU size limit enforcement (DoS protection)

**Key Security Findings**: PDU decoder now validates size limits to prevent DoS attacks via memory exhaustion (fixed in `dimse/pdu/data.go`).

#### DIMSE Layer (6 fuzz functions)

Location: `dimse/dimse/fuzz_test.go`

- `FuzzCommandSetFromDataSet`: DIMSE command decoding from malformed datasets
- `FuzzCommandSetRoundTrip`: Encode/decode round-trip consistency
- `FuzzMessageEncoding`: Message encoding with invalid parameters
- `FuzzReassemblerAddPDU`: Message reassembly from fragmented PDUs
- `FuzzStatusCodeHandling`: Status code validation
- `FuzzMessageFragmentation`: Message fragmentation at various PDU sizes

#### State Machine (6 fuzz functions)

Location: `dimse/dul/fuzz_test.go`

- `FuzzStateMachineEventSequence`: Random event sequences
- `FuzzStateMachineInvalidEvents`: Out-of-range event values
- `FuzzStateMachineStates`: All state/event combinations
- `FuzzStateMachineConcurrent`: Concurrent event processing (race detection)
- `FuzzStateMachineTransitionInvariants`: State transition invariant validation
- `FuzzStateMachineIdempotency`: Repeated event consistency

### Running Fuzz Tests

Run a single fuzz test:
```bash
cd dimse/pdu
go test -fuzz=FuzzAssociateRQDecode -fuzztime=30s
```

Run all tests in a package (not fuzzing, just validation):
```bash
cd dimse/pdu
go test -v
```

Run fuzz tests for extended periods (CI/CD):
```bash
go test -fuzz=FuzzAssociateRQDecode -fuzztime=5m
```

Run all fuzz tests with the provided script:
```bash
# Quick run (30s per test, default)
./scripts/run-fuzz-tests.sh

# Extended run (5m per test)
./scripts/run-fuzz-tests.sh 5m

# Custom configuration
FUZZ_TIME=10m VERBOSE=true ./scripts/run-fuzz-tests.sh
```

### Interpreting Fuzz Test Results

**Success**: Test runs without crashes/panics
```
fuzz: elapsed: 30s, execs: 152000 (5066/sec), new interesting: 12 (total: 25)
PASS
```

**Failure**: Test found a crash/panic
```
--- FAIL: FuzzExample (0.42s)
    --- FAIL: FuzzExample/8a3c2d1f (0.00s)
        panic: runtime error: index out of range
```

When a failure occurs:
1. Go saves the failing input to `testdata/fuzz/`
2. The test will replay that input on future runs
3. Fix the bug, then rerun to verify

## Integration Testing

Integration tests validate the DIMSE implementation against a real PACS system (Orthanc) using Docker containers.

### Why Integration Testing?

- **Real-World Validation**: Tests against actual PACS behavior
- **End-to-End Coverage**: Validates complete workflows
- **Interoperability**: Ensures compliance with real DICOM implementations
- **Network Protocol Testing**: Tests TCP connection handling, timeouts, association lifecycle

### Orthanc Container Setup

Location: `dimse/integration/orthanc/`

The test infrastructure uses [testcontainers-go](https://github.com/testcontainers/testcontainers-go) to automatically:
1. Pull the Orthanc Docker image
2. Start an Orthanc container
3. Configure DICOM ports (4242) and HTTP REST API (8042)
4. Run tests
5. Clean up container

### Integration Test Coverage

#### TestOrthancIntegration_CEcho
Tests C-ECHO (verification) operation:
- Association establishment
- C-ECHO request/response
- Association release

#### TestOrthancIntegration_CStore
Tests C-STORE (image storage) operation:
- Creates test DICOM dataset
- Stores instance to Orthanc
- Verifies storage via REST API

#### TestOrthancIntegration_CFind
Tests C-FIND (query) operation:
- Stores test data
- Performs patient-level query
- Validates query results

#### TestOrthancIntegration_CGet
Tests C-GET (retrieve) operation:
- Stores test data
- Retrieves instances via C-GET
- Validates retrieved datasets

#### TestOrthancIntegration_SCPReceive
Tests receiving C-STORE as an SCP (Service Class Provider):
- Starts local SCP server
- Stores test data to Orthanc
- Configures Orthanc modality to send to local SCP
- Triggers C-STORE from Orthanc to SCP
- Validates SCP received correct data

#### TestOrthancIntegration_CMove
Tests C-MOVE operation with third-party destination:
- Starts local SCP as move destination
- Stores test data to Orthanc
- Configures SCP as modality in Orthanc
- Issues C-MOVE command to Orthanc
- Orthanc sends data to destination SCP
- Validates destination SCP received moved data

### Running Integration Tests

Integration tests are skipped in short mode by default:

```bash
# Skip integration tests (fast)
go test -short ./dimse/integration/orthanc/

# Run integration tests (requires Docker)
go test -v ./dimse/integration/orthanc/

# Run specific test
go test -v ./dimse/integration/orthanc/ -run TestOrthancIntegration_CEcho
```

### Requirements

- **Docker**: Must be running and accessible
- **Network Access**: Tests need to pull `orthancteam/orthanc:latest` image
- **Time**: Each test takes 30-120 seconds (container startup time)

### Test Timeout

Integration tests have generous timeouts (2 minutes per test) to account for:
- Docker image pull (first run)
- Container startup
- Orthanc initialization
- DICOM operations

## Continuous Integration

### Recommended CI Pipeline

A complete GitHub Actions workflow example is provided in `.github/workflows/fuzz-tests.yml.example`.

The workflow includes:
- **Pull Request Checks**: Quick 30s fuzz tests for fast feedback
- **Nightly Extended Tests**: 5m fuzz tests for thorough coverage
- **Manual Dispatch**: Configurable duration for ad-hoc testing
- **Security Focus**: Dedicated security-critical fuzz tests
- **Artifact Upload**: Automatic upload of failing test cases

To enable:
```bash
cp .github/workflows/fuzz-tests.yml.example .github/workflows/fuzz-tests.yml
```

Key features:
- Runs 21 fuzz tests across PDU, DIMSE, and DUL layers
- Configurable durations based on event type (PR vs scheduled)
- Uploads fuzz corpus on failure for reproduction
- Parallel execution for faster CI runs
- Integration with Docker for Orthanc tests

Example minimal CI configuration:
```yaml
name: Tests
on: [push, pull_request]
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -short ./...

  fuzz-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: cd dimse && ./scripts/run-fuzz-tests.sh 30s

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -v ./dimse/integration/orthanc/
```

## Test Data

### Fuzz Test Corpus

Fuzz tests seed with:
- Valid protocol messages
- Boundary cases (empty, truncated, oversized)
- Known invalid sequences
- Regression test cases from past bugs

Corpus data is stored in `testdata/fuzz/<FuzzTestName>/` and versioned in git.

### Integration Test Data

Integration tests use synthetic DICOM datasets:
- Minimal valid DICOM attributes
- Predictable UIDs for verification
- Patient names prefixed by test name (FindTest^Patient, GetTest^Patient)

## Troubleshooting

### Fuzz Tests Won't Run

**Error**: `testing: will not fuzz, -fuzz matches more than one fuzz test`

**Solution**: Specify the exact fuzz function name:
```bash
go test -fuzz=FuzzAssociateRQDecode
```

### Integration Tests Fail to Start Container

**Error**: `failed to start Orthanc container`

**Possible causes**:
1. Docker daemon not running: `docker ps` to verify
2. Network issues pulling image: Check internet connection
3. Port conflicts: Orthanc needs ports 4242 and 8042

**Solution**: Testcontainers uses random ports, so conflicts are rare. If issues persist, check Docker logs.

### Integration Tests Timeout

**Error**: `context deadline exceeded`

**Possible causes**:
1. Slow Docker on first run (image pull)
2. Resource constraints (CI environment)
3. Network issues

**Solution**: Increase timeout in test or run with `-timeout` flag:
```bash
go test -timeout 10m ./dimse/integration/orthanc/
```

## Best Practices

### Adding New Fuzz Tests

1. **Seed with valid inputs**: Start with known-good messages
2. **Test invariants**: Verify properties that must always hold
3. **Don't test external behavior**: Focus on crash prevention, not correctness
4. **Document findings**: Add TODO comments for discovered issues

Example template:
```go
func FuzzNewDecoder(f *testing.F) {
    // Seed with valid input
    f.Add(validBytes)
    f.Add([]byte{})  // empty

    f.Fuzz(func(t *testing.T, data []byte) {
        // Should never panic
        result, err := Decode(data)

        // If decode succeeded, verify invariants
        if err == nil && result != nil {
            // Invariant checks...
        }
    })
}
```

### Adding New Integration Tests

1. **Use `testing.Short()` guard**: Allow skipping in quick test runs
2. **Clean up resources**: Use `defer` for container teardown
3. **Verify via multiple methods**: Check results via both DIMSE and REST API
4. **Use unique identifiers**: Prevent test interference

## Performance

### Benchmark Tests

Performance benchmarks are provided for critical operations to track performance regressions and optimize hot paths.

#### Running Benchmarks

Run all benchmarks:
```bash
cd dimse/pdu
go test -bench=. -benchmem

cd ../dimse
go test -bench=. -benchmem
```

Run specific benchmarks:
```bash
# PDU encoding/decoding
go test -bench=BenchmarkAssociateRQ -benchmem

# Data transfer performance
go test -bench=BenchmarkDataTF -benchmem

# Message fragmentation
go test -bench=BenchmarkMessage_Fragmentation -benchmem
```

Compare performance across changes:
```bash
# Baseline
go test -bench=. -benchmem > old.txt

# After changes
go test -bench=. -benchmem > new.txt

# Compare (requires benchstat: go install golang.org/x/perf/cmd/benchstat@latest)
benchstat old.txt new.txt
```

#### Benchmark Coverage

**PDU Layer** (`dimse/pdu/bench_test.go`):
- `BenchmarkAssociateRQ_Encode/Decode`: Association negotiation performance
- `BenchmarkDataTF_Encode/Decode`: Data transfer with 1KB, 16KB, 64KB payloads
- `BenchmarkPDU_RoundTrip`: Full encode/decode cycle
- `BenchmarkPDUType_Identification`: Type checking overhead

**DIMSE Layer** (`dimse/dimse/bench_test.go`):
- `BenchmarkCommandSet_Encode/Decode`: Command message handling
- `BenchmarkMessage_Encode`: Message fragmentation at various PDU sizes
- `BenchmarkReassembler_AddPDU`: Message reassembly from fragments
- `BenchmarkMessage_Fragmentation`: Boundary condition testing
- `BenchmarkCommandSet_RoundTrip`: Full command lifecycle

#### Expected Performance

Typical performance on modern hardware (M4 Pro, example):
- **PDU Encode**: ~900 ns/op (~1M ops/sec)
- **Data Transfer (16KB)**: ~1.6 μs encode, ~136 ns decode
- **Command Encode**: ~324 ns/op (~3M ops/sec)
- **Message Reassembly**: ~3.6 μs/op

Throughput:
- **Small PDU (1KB)**: ~4.8 GB/s encode, ~7.5 GB/s decode
- **Medium PDU (16KB)**: ~10 GB/s encode, ~120 GB/s decode
- **Large PDU (64KB)**: ~10.7 GB/s encode, ~483 GB/s decode

### Fuzz Test Performance

- **PDU Layer**: ~5000 exec/sec (small messages)
- **DIMSE Layer**: ~300000 exec/sec (command encoding)
- **State Machine**: ~20000 exec/sec (concurrent testing slower)

### Integration Test Performance

- **First run**: 30-60 seconds (Docker image pull)
- **Subsequent runs**: 10-20 seconds per test
- **Parallel execution**: Not recommended (Docker overhead)

## Security Considerations

### Security Issues Fixed by Fuzzing

1. **PDU Size Limit Validation** (Fixed in dimse/pdu/data.go:107-110)
   - Issue: Decoder didn't enforce maximum PDU size
   - Impact: Potential DoS via memory exhaustion
   - Status: **FIXED** - Added validation in `decodePresentationDataValue()`
   - Implementation: Validates item length before memory allocation
   - Testing: Fuzz test `FuzzPDUSizeLimit` verifies the fix with 70k+ test cases

### Future Security Testing

- **Malformed DICOM encoding**: Test with corrupted VR/VL values
- **Association flood testing**: Test rapid association requests
- **Transfer syntax attack vectors**: Test unsupported transfer syntaxes
- **Resource exhaustion**: Test with extremely large datasets

## References

- [Go Fuzzing Tutorial](https://go.dev/doc/tutorial/fuzz)
- [Testcontainers Documentation](https://golang.testcontainers.org/)
- [DICOM Standard Part 8](https://dicom.nema.org/medical/dicom/current/output/html/part08.html)
- [Orthanc Book](https://book.orthanc-server.com/)
