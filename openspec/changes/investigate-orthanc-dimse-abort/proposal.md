# Investigation: Orthanc DIMSE Abort PDU and Timeout Issues

## Why

DIMSE integration tests against Orthanc PACS server are failing with two distinct error patterns that prevent
validation of the DIMSE protocol implementation against a real-world DICOM server. This blocks production readiness
of the DIMSE networking capabilities and prevents verification that the implementation interoperates correctly with
standard PACS systems.

**Current State:**
- C-ECHO operations succeed (proves basic association establishment works)
- All unit tests pass (proves encoding/decoding logic is sound)
- C-STORE, C-FIND, C-GET fail with "Abort PDU" errors from Orthanc
- C-MOVE and SCP receive operations fail with I/O timeout errors
- Five integration tests are currently skipped to unblock CI/CD

**Problem Impact:**
- Cannot verify DIMSE implementation against industry-standard PACS
- Potential protocol-level incompatibilities remain undetected
- Risk of interoperability failures in production deployments
- Reduced confidence in transfer syntax negotiation and dataset encoding

## What Changes

This investigation will:
- Identify root cause of Abort PDU errors (C-STORE, C-FIND, C-GET)
- Identify root cause of I/O timeout errors (C-MOVE, SCP receive)
- Determine if issues stem from:
  - Incorrect PDU encoding/formatting
  - Transfer syntax negotiation problems
  - Dataset encoding issues (Implicit vs Explicit VR)
  - Presentation context configuration
  - Protocol state machine errors
- Document findings with network packet captures and Orthanc logs
- Propose specific fixes to achieve Orthanc compatibility
- Re-enable all skipped integration tests once fixes are validated

**Investigation Approach:**
1. Network packet capture (Wireshark) of successful vs failing operations
2. Orthanc server log analysis to understand rejection reasons
3. DICOM Part 7 specification review for protocol compliance
4. Comparison with reference implementations (pynetdicom, dcm4che)
5. Validation of PDU structure, transfer syntax encoding, and presentation contexts

## Impact

**Affected Components:**
- dimse/integration/orthanc/integration_test.go - Five tests currently skipped
- dimse/scu/client.go - SCU client implementation (potential fixes needed)
- dimse/scp/server.go - SCP server implementation (potential fixes needed)
- dimse/dimse/message.go - DIMSE message encoding (transfer syntax handling)
- dimse/pdu/*.go - PDU encoding and state machine
- dimse/dul/*.go - DUL association management

**Code Changes:**
- Investigation findings will inform specific bug fixes
- May require changes to transfer syntax selection logic
- May require PDU encoding corrections
- May require presentation context negotiation adjustments
- All changes must maintain backward compatibility with unit tests

**Testing Impact:**
- Five currently skipped integration tests will be re-enabled
- Additional test cases may be added based on findings
- Network capture and log analysis tools added to CI/CD for future debugging

**Documentation:**
- Root cause analysis document
- Protocol debugging guide for future DIMSE work
- Orthanc compatibility notes
- Known limitations and workarounds
