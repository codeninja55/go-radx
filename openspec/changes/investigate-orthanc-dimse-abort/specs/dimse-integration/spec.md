# DIMSE Integration Testing

## ADDED Requirements

### Requirement: Orthanc PACS Integration Testing

The DIMSE implementation SHALL be validated against Orthanc PACS server to ensure protocol-level interoperability
with industry-standard DICOM systems.

#### Scenario: C-ECHO operation succeeds

- **WHEN** SCU client performs C-ECHO operation against Orthanc
- **THEN** association is established successfully
- **AND** C-ECHO-RQ is sent and C-ECHO-RSP is received
- **AND** operation completes without errors

#### Scenario: C-STORE operation succeeds

- **WHEN** SCU client performs C-STORE operation to send DICOM instance to Orthanc
- **THEN** association is established with correct presentation context
- **AND** transfer syntax is negotiated correctly
- **AND** DICOM dataset is encoded using negotiated transfer syntax
- **AND** C-STORE-RQ is sent and C-STORE-RSP with success status (0x0000) is received
- **AND** Orthanc confirms instance storage via REST API

#### Scenario: C-FIND operation succeeds

- **WHEN** SCU client performs C-FIND query against Orthanc
- **THEN** association is established with correct presentation context
- **AND** C-FIND-RQ is sent with properly formatted query dataset
- **AND** C-FIND-RSP is received with matching results
- **AND** final C-FIND-RSP with success status (0x0000) indicates query completion

#### Scenario: C-GET operation succeeds

- **WHEN** SCU client performs C-GET to retrieve instances from Orthanc
- **THEN** association is established with correct presentation contexts
- **AND** C-GET-RQ is sent with retrieval parameters
- **AND** Orthanc sends C-STORE sub-operations for matching instances
- **AND** SCU receives and acknowledges instances via C-STORE-RSP
- **AND** final C-GET-RSP with success status (0x0000) indicates completion

#### Scenario: C-MOVE operation succeeds

- **WHEN** SCU client performs C-MOVE to request instance transfer
- **THEN** association is established with correct presentation context
- **AND** C-MOVE-RQ is sent with destination AE title
- **AND** C-MOVE-RSP is received indicating move progress
- **AND** operation completes without timeout errors
- **AND** final C-MOVE-RSP with success status (0x0000) indicates completion

#### Scenario: SCP receives C-STORE from Orthanc

- **WHEN** SCP server listens for incoming associations from Orthanc
- **AND** DICOM instance is sent to Orthanc via REST API triggering C-STORE forward
- **THEN** SCP accepts association from Orthanc
- **AND** SCP receives C-STORE-RQ with DICOM instance
- **AND** SCP successfully decodes received dataset
- **AND** SCP sends C-STORE-RSP with success status (0x0000)
- **AND** operation completes without timeout errors

### Requirement: Transfer Syntax Negotiation Compliance

The DIMSE implementation SHALL correctly negotiate and apply transfer syntaxes according to DICOM Part 7
specification.

#### Scenario: Implicit VR Little Endian transfer syntax

- **WHEN** presentation context negotiates Implicit VR Little Endian (1.2.840.10008.1.2)
- **THEN** dataset SHALL be encoded using Implicit VR format
- **AND** all data elements use 4-byte length fields
- **AND** byte order is little-endian
- **AND** VR is not present in encoded stream

#### Scenario: Explicit VR Little Endian transfer syntax

- **WHEN** presentation context negotiates Explicit VR Little Endian (1.2.840.10008.1.2.1)
- **THEN** dataset SHALL be encoded using Explicit VR format
- **AND** data elements include 2-byte VR field
- **AND** length field is 2 bytes for short VRs, 4 bytes for long VRs
- **AND** byte order is little-endian

#### Scenario: Transfer syntax mismatch detection

- **WHEN** negotiated transfer syntax differs from dataset encoding
- **THEN** PACS server SHALL reject operation with Abort PDU or error status
- **AND** go-radx SHALL detect and report the rejection
- **AND** error message SHALL indicate transfer syntax mismatch

### Requirement: PDU Structure Compliance

The DIMSE implementation SHALL encode and decode PDUs according to DICOM Part 8 specification.

#### Scenario: A-ASSOCIATE-RQ structure is valid

- **WHEN** SCU initiates association
- **THEN** A-ASSOCIATE-RQ SHALL contain valid Application Context Name
- **AND** Presentation Contexts SHALL have odd-numbered IDs
- **AND** Each Presentation Context SHALL have Abstract Syntax UID
- **AND** Each Presentation Context SHALL have one or more Transfer Syntax UIDs
- **AND** Maximum PDU Length SHALL be specified
- **AND** Implementation Class UID SHALL be present

#### Scenario: P-DATA-TF fragmentation is correct

- **WHEN** DIMSE message exceeds maximum PDU length
- **THEN** message SHALL be fragmented into multiple P-DATA-TF PDUs
- **AND** each fragment SHALL have correct Message Control Header
- **AND** command fragments SHALL be marked with 0x01
- **AND** dataset fragments SHALL be marked with 0x00 or 0x02 (last)
- **AND** final fragment SHALL have last fragment bit set

#### Scenario: Command set encoding is Implicit VR

- **WHEN** DIMSE command set is encoded
- **THEN** command set SHALL use Implicit VR Little Endian encoding
- **AND** command set SHALL NOT use Explicit VR regardless of dataset transfer syntax
- **AND** all DIMSE command attributes SHALL be present and valid

### Requirement: Error Recovery and Diagnostics

The DIMSE implementation SHALL provide clear diagnostics when protocol errors occur during Orthanc integration.

#### Scenario: Abort PDU received from Orthanc

- **WHEN** Orthanc sends A-ABORT PDU
- **THEN** go-radx SHALL capture abort reason and source
- **AND** error message SHALL include abort reason code
- **AND** error message SHALL include abort source (UL service-user or service-provider)
- **AND** operation SHALL fail gracefully without panic

#### Scenario: I/O timeout during DIMSE operation

- **WHEN** expected PDU is not received within timeout period
- **THEN** operation SHALL fail with timeout error
- **AND** error message SHALL indicate which PDU was expected
- **AND** error message SHALL include timeout duration
- **AND** association SHALL be closed cleanly

#### Scenario: Network packet capture support

- **WHEN** debugging protocol issues
- **THEN** developers SHALL be able to capture network packets with Wireshark
- **AND** DICOM dissector SHALL decode PDUs correctly
- **AND** all PDU structures SHALL be inspectable in packet capture
- **AND** transfer syntax encoding SHALL be verifiable via packet analysis
