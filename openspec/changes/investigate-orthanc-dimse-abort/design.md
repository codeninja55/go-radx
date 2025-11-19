# Design: Orthanc DIMSE Investigation Approach

## Context

The DIMSE protocol implementation passes all unit tests and successfully performs C-ECHO operations against Orthanc,
proving that basic association establishment, PDU encoding, and state machine logic are fundamentally sound. However,
more complex DIMSE operations (C-STORE, C-FIND, C-GET, C-MOVE) fail with either Abort PDU errors or I/O timeouts.

**Key Observations:**
- C-ECHO succeeds → Association establishment works correctly
- Unit tests pass → PDU encoding/decoding logic is structurally sound
- C-STORE/C-FIND/C-GET fail with Abort PDU → Orthanc rejects operations for protocol reasons
- C-MOVE/SCP receive fail with I/O timeout → Communication breakdown without explicit rejection

**Recent Implementation Changes:**
- Added transfer syntax-aware dataset encoding (commit 9ca1ec4)
- Implemented `encodeDataSetByTransferSyntax()` to handle Implicit vs Explicit VR
- Modified `Message.Encode()` to pass negotiated transfer syntax
- Updated SCU client and SCP server to propagate transfer syntax from presentation context

**Hypothesis:**
The Abort PDU and timeout errors likely stem from subtle protocol violations in:
1. Presentation context negotiation (wrong SOP Classes or transfer syntaxes)
2. Dataset encoding (transfer syntax mismatch or VR errors)
3. DIMSE command set structure (missing or malformed attributes)
4. PDU fragmentation or control headers
5. State machine transitions not matching Orthanc's expectations

## Goals / Non-Goals

**Goals:**
- Identify exact protocol violations causing Orthanc to send Abort PDU
- Determine root cause of I/O timeout errors in C-MOVE and SCP operations
- Fix identified issues to achieve 100% Orthanc integration test pass rate
- Document debugging methodology for future DIMSE protocol investigations
- Validate fixes work against both Orthanc and potentially other PACS systems

**Non-Goals:**
- Supporting Orthanc-specific non-standard protocol extensions
- Implementing workarounds that compromise DICOM standard compliance
- Optimizing performance of DIMSE operations (investigation phase only)
- Adding new DIMSE features beyond fixing existing broken ones
- Creating Orthanc-specific code paths (fixes must be standard-compliant)

## Decisions

### Decision 1: Use Wireshark for Packet Capture

**Rationale:**
- Industry-standard tool with native DICOM protocol dissector
- Can decode PDUs, presentation contexts, transfer syntaxes, and dataset structure
- Provides side-by-side comparison capability with reference implementations
- Generates exportable packet captures for documentation
- Free, open-source, and widely available

**Alternatives Considered:**
- Raw tcpdump with manual hex analysis → Too time-consuming, error-prone
- Custom Go packet sniffer → Unnecessary engineering overhead
- Orthanc logs alone → Insufficient low-level PDU detail

**Decision:** Use Wireshark with DICOM dissector enabled.

### Decision 2: Comparative Analysis with pynetdicom

**Rationale:**
- pynetdicom is a mature, well-tested DIMSE implementation
- Known to work correctly with Orthanc
- Python codebase is readable and well-documented
- Can perform identical operations and compare packet captures
- Provides ground truth for correct protocol behavior

**Alternatives Considered:**
- dcm4che (Java) → More complex to set up, less readable
- DCMTK (C++) → Older codebase, harder to trace
- fo-dicom (C#) → Less mature, requires .NET runtime
- Reverse-engineer from DICOM spec alone → Spec ambiguities exist

**Decision:** Use pynetdicom as reference implementation for comparative packet analysis.

### Decision 3: Investigate Transfer Syntax Encoding First

**Rationale:**
- Recent changes introduced transfer syntax-aware encoding
- Transfer syntax mismatch is common cause of Abort PDU in DICOM
- C-ECHO doesn't send dataset (no transfer syntax), so C-ECHO success doesn't validate dataset encoding
- Dataset encoding impacts C-STORE, C-FIND, C-GET but not C-ECHO

**Hypothesis to Test:**
1. Are we negotiating the correct transfer syntax?
2. Is dataset encoded using the negotiated transfer syntax?
3. Is the VR encoding (Implicit vs Explicit) correct for the negotiated transfer syntax?
4. Are we properly encoding sequences and nested datasets?

**Investigation Approach:**
- Capture C-STORE packet with real DICOM file (known transfer syntax)
- Verify A-ASSOCIATE-AC presentation context acceptance
- Verify P-DATA-TF dataset encoding matches accepted transfer syntax
- Compare dataset encoding with file's original encoding

**Decision:** Prioritize transfer syntax validation in investigation order.

### Decision 4: Skip Investigation of C-ECHO (It Already Works)

**Rationale:**
- C-ECHO succeeds consistently
- C-ECHO only tests association and command set (no dataset)
- Problem is specific to operations with datasets
- Investigating C-ECHO provides no new information

**Decision:** Focus investigation on failing operations only.

### Decision 5: Document Findings Before Implementing Fixes

**Rationale:**
- Premature fixes may address symptoms without root cause
- Documentation ensures fixes are evidence-based
- Packet captures provide reproducible test cases
- Future protocol issues can reference this methodology

**Decision:** Complete root cause analysis document before implementing any code changes.

## Technical Approach

### Investigation Workflow

```
1. Set up environment
   ├── Orthanc with verbose logging
   ├── Wireshark with DICOM dissector
   └── pynetdicom for reference

2. For each failing operation (C-STORE, C-FIND, C-GET, C-MOVE, SCP):
   ├── Reproduce failure with go-radx
   ├── Capture packet trace
   ├── Extract Orthanc logs
   ├── Perform same operation with pynetdicom
   ├── Capture pynetdicom packet trace
   ├── Compare PDU structure side-by-side
   ├── Identify differences
   └── Map differences to DICOM Part 7 spec

3. Document root cause
   ├── Annotated packet captures
   ├── Specification references
   ├── Code locations responsible
   └── Proposed fixes

4. Implement fixes
   ├── Update relevant DIMSE code
   ├── Ensure unit tests still pass
   └── Validate fix with Orthanc

5. Re-enable integration tests
   └── Verify 100% pass rate
```

### Key Areas to Inspect

**A-ASSOCIATE-RQ (Association Request):**
- Application Context Name: `1.2.840.10008.3.1.1.1`
- Presentation Context IDs (odd numbers only)
- Abstract Syntax UIDs (SOP Class UIDs)
- Transfer Syntax UIDs (must include at least one Orthanc supports)
- Maximum PDU Length (typically 16384 or higher)
- Implementation Class UID and Version Name

**A-ASSOCIATE-AC (Association Accept):**
- Accepted presentation contexts (Result: 0x00 = acceptance)
- Accepted transfer syntax (exactly one per context)
- Rejected contexts and reasons
- Maximum PDU Length from Orthanc

**P-DATA-TF (Data Transfer):**
- Presentation Context ID matches accepted context
- Message Control Header (0x01 = command, 0x02 = dataset, 0x03 = last fragment)
- Command set encoding (always Implicit VR Little Endian per Part 7)
- Dataset encoding (must match accepted transfer syntax)
- Fragmentation and reassembly

**DIMSE Command Sets:**
- (0000,0002) Affected SOP Class UID
- (0000,0100) Command Field (e.g., 0x0001 = C-STORE-RQ)
- (0000,0110) Message ID
- (0000,0700) Priority
- (0000,0800) Data Set Type (0x0101 = dataset present, 0x0101 = no dataset)
- Operation-specific attributes

**Dataset Encoding:**
- Implicit VR: Tag (4 bytes) + Length (4 bytes) + Value
- Explicit VR: Tag (4 bytes) + VR (2 bytes) + Length (2 or 4 bytes) + Value
- Transfer syntax must match presentation context
- Byte order: Little Endian (unless Big Endian explicitly negotiated)

## Risks / Trade-offs

### Risk: Fixes May Break Other PACS Systems

**Mitigation:**
- Ensure all fixes comply with DICOM Part 7 specification
- Keep comprehensive unit test coverage
- Test against multiple PACS systems if available
- Avoid Orthanc-specific workarounds

### Risk: Investigation May Reveal Fundamental Architecture Issues

**Mitigation:**
- Start with smallest possible fixes
- Document alternatives if large refactor needed
- Separate investigation from implementation
- Get approval before major architectural changes

### Risk: Orthanc May Have Non-Standard Behavior

**Mitigation:**
- Cross-reference with DICOM conformance statement
- Test with other PACS systems (DCM4CHEE, Conquest, etc.)
- Prioritize standard compliance over Orthanc-specific compatibility
- Document any Orthanc quirks separately

### Trade-off: Investigation Time vs Quick Workarounds

**Decision:**
- Invest time in thorough investigation rather than quick hacks
- Proper root cause analysis prevents future regressions
- Documentation benefits long-term maintainability
- Workarounds create technical debt

## Migration Plan

### Phase 1: Investigation (No Code Changes)

1. Set up debugging environment
2. Collect packet captures and logs for all failing tests
3. Perform comparative analysis with pynetdicom
4. Document root cause for each failure
5. Create root-cause-analysis.md with findings
6. Get approval on proposed fixes

### Phase 2: Fix Implementation

1. Implement fixes based on approved root cause analysis
2. Run unit tests after each change (ensure no regressions)
3. Run failing integration tests after each fix
4. Document any behavioral changes
5. Update comments and documentation

### Phase 3: Validation and Re-enable Tests

1. Remove t.Skip() from TestOrthancIntegration_CStore
2. Verify test passes, if not return to Phase 2
3. Repeat for C-FIND, C-GET, C-MOVE, SCPReceive
4. Run full integration test suite
5. Verify no flaky tests (run multiple times)

### Phase 4: Documentation and Cleanup

1. Update dimse/README.md
2. Create debugging guide for future use
3. Archive packet captures and logs
4. Update CHANGELOG.md
5. Archive this OpenSpec change

### Rollback Plan

If fixes introduce regressions:
- Revert specific commit
- Re-apply t.Skip() to affected tests
- Revisit root cause analysis
- Adjust fix approach

## Open Questions

1. **Q:** Should we test against PACS systems other than Orthanc?
   - **A:** If accessible, yes, but Orthanc is sufficient for initial validation

2. **Q:** Do we need to update transfer syntax negotiation logic?
   - **A:** TBD - investigation will determine

3. **Q:** Should we add protocol validation tooling to CI/CD?
   - **A:** Yes if investigation reveals value (e.g., automated PDU structure validation)

4. **Q:** Are there known Orthanc version differences in DIMSE behavior?
   - **A:** TBD - may test multiple versions if issues arise

5. **Q:** Should we create helper scripts for packet capture automation?
   - **A:** Yes if investigation workflow is repetitive enough to benefit from automation
