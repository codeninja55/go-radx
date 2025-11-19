# Implementation Tasks

## 1. Investigation Setup

- [ ] 1.1 Install Wireshark for network packet capture
- [ ] 1.2 Configure Wireshark DICOM dissector for PDU analysis
- [ ] 1.3 Set up local Orthanc instance with verbose logging enabled
- [ ] 1.4 Create test scripts to reproduce each failure scenario independently
- [ ] 1.5 Document baseline environment (Orthanc version, configuration, test data)

## 2. C-STORE Abort PDU Investigation

- [ ] 2.1 Capture Wireshark packets for failing C-STORE operation
- [ ] 2.2 Extract Orthanc server logs for C-STORE rejection
- [ ] 2.3 Analyze A-ASSOCIATE-RQ and A-ASSOCIATE-AC presentation contexts
- [ ] 2.4 Verify negotiated transfer syntax matches dataset encoding
- [ ] 2.5 Compare PDU structure with DICOM Part 7 specification
- [ ] 2.6 Test with both real DICOM files and synthetic datasets
- [ ] 2.7 Identify specific PDU/encoding issue causing Abort
- [ ] 2.8 Document root cause with packet capture evidence

## 3. C-FIND Abort PDU Investigation

- [ ] 3.1 Capture Wireshark packets for failing C-FIND operation
- [ ] 3.2 Extract Orthanc server logs for C-FIND rejection
- [ ] 3.3 Analyze C-FIND-RQ command dataset encoding
- [ ] 3.4 Verify query keys and matching keys format
- [ ] 3.5 Compare with pynetdicom C-FIND implementation
- [ ] 3.6 Identify specific issue causing Abort
- [ ] 3.7 Document root cause with packet capture evidence

## 4. C-GET Abort PDU Investigation

- [ ] 4.1 Capture Wireshark packets for failing C-GET operation
- [ ] 4.2 Extract Orthanc server logs for C-GET rejection
- [ ] 4.3 Analyze C-GET-RQ command dataset encoding
- [ ] 4.4 Verify SOP Class UID and instance retrieval parameters
- [ ] 4.5 Compare with reference implementation
- [ ] 4.6 Identify specific issue causing Abort
- [ ] 4.7 Document root cause with packet capture evidence

## 5. C-MOVE and SCP I/O Timeout Investigation

- [ ] 5.1 Capture Wireshark packets for C-MOVE operation up to timeout
- [ ] 5.2 Capture Wireshark packets for SCP receive operation up to timeout
- [ ] 5.3 Extract Orthanc server logs for timeout scenarios
- [ ] 5.4 Analyze if Orthanc sends expected response PDUs
- [ ] 5.5 Check for missing acknowledgments or state machine issues
- [ ] 5.6 Verify timeout values and socket configuration
- [ ] 5.7 Identify root cause of communication breakdown
- [ ] 5.8 Document findings with packet capture evidence

## 6. Comparative Analysis

- [ ] 6.1 Set up pynetdicom test scripts for same operations
- [ ] 6.2 Capture Wireshark packets from successful pynetdicom operations
- [ ] 6.3 Perform side-by-side PDU comparison (go-radx vs pynetdicom)
- [ ] 6.4 Document all structural differences in PDU encoding
- [ ] 6.5 Identify protocol behavior differences
- [ ] 6.6 Create detailed comparison report

## 7. Root Cause Documentation

- [ ] 7.1 Create root-cause-analysis.md with all findings
- [ ] 7.2 Include annotated packet captures (exported as images/PDFs)
- [ ] 7.3 Include relevant Orthanc log excerpts
- [ ] 7.4 Map each error to specific DICOM Part 7 specification violations
- [ ] 7.5 Provide PDU hex dumps with annotations
- [ ] 7.6 Document environmental factors (versions, configurations)

## 8. Fix Implementation

- [ ] 8.1 Create fix proposals based on root cause analysis
- [ ] 8.2 Implement fixes for Abort PDU issues
- [ ] 8.3 Implement fixes for I/O timeout issues
- [ ] 8.4 Update transfer syntax negotiation if needed
- [ ] 8.5 Update PDU encoding if needed
- [ ] 8.6 Update presentation context handling if needed
- [ ] 8.7 Ensure all unit tests still pass after fixes

## 9. Validation

- [ ] 9.1 Re-enable TestOrthancIntegration_CStore and verify pass
- [ ] 9.2 Re-enable TestOrthancIntegration_CFind and verify pass
- [ ] 9.3 Re-enable TestOrthancIntegration_CGet and verify pass
- [ ] 9.4 Re-enable TestOrthancIntegration_CMove and verify pass
- [ ] 9.5 Re-enable TestOrthancIntegration_SCPReceive and verify pass
- [ ] 9.6 Run full integration test suite (all tests passing)
- [ ] 9.7 Verify no regressions in unit tests
- [ ] 9.8 Test against multiple Orthanc versions if possible

## 10. Documentation

- [ ] 10.1 Update dimse/README.md with Orthanc compatibility notes
- [ ] 10.2 Create debugging guide for future DIMSE protocol issues
- [ ] 10.3 Document known limitations or workarounds
- [ ] 10.4 Add packet capture examples to documentation
- [ ] 10.5 Update CI/CD documentation for debugging DIMSE failures

## 11. Cleanup

- [ ] 11.1 Remove all t.Skip() calls from integration tests
- [ ] 11.2 Update test comments to reflect resolution
- [ ] 11.3 Archive investigation artifacts (packet captures, logs)
- [ ] 11.4 Update CHANGELOG.md with bug fixes
- [ ] 11.5 Close related GitHub issues if any
