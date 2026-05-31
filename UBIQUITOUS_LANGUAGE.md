# Ubiquitous language — go-radx

This glossary fixes the shared, precise vocabulary for go-radx across the three standards it implements — DICOM
(NEMA PS3), HL7 v2.x, and HL7 FHIR — and names the canonical Go identifier to use in code and docs for each concept. It
exists because the same word means different things in each standard ("Study", "Instance", "Reference", "Observation",
"Report"), and silently aliasing them is a primary source of conversion bugs.

Definitions are grounded in the standards and the reference libraries (`pydicom`, `pynetdicom`, `python-hl7`,
`fhir.resources`); canonical names align with the PRD's API commitments (`docs/prd/go-radx-prd.md` §8.1, §8.2).

## Naming rules

1. **Named types over bare primitives.** `Tag`, `UID`, `VR`, `TransferSyntax`, `AETitle`, `Decimal`, `Status`,
   `QueryLevel`, `Priority` are distinct Go types, never `string`/`uint32`. A signature should read without the docs.
2. **Each standard keeps its own nouns.** There is no shared, cross-standard `Study`, `Instance`, `Reference`,
   `Observation`, `Report`, or `Order` type. The DICOM side stays UID-keyed dataset access; the FHIR side is a generated
   resource; the HL7 side is a typed segment/message.
3. **Conversions use `convert.<Source>To<Target>`** with each standard's canonical noun, so direction and both type
   spaces are unambiguous from the signature alone: `DICOMToImagingStudy`, `SRToDiagnosticReport` /
   `DiagnosticReportToSR`, `ORUToDiagnosticReport`, `ORMToServiceRequest`, `ADTToPatient`, `ADTToEncounter`.
   Bidirectional mappings are two named functions, not a direction flag.
4. **Diagnostics name concepts** (a tag as keyword + `(gggg,eeee)`, VRs/UIDs/DIMSE statuses by name), never raw
   hex/codes, and never carry PHI by default (PRD §8.2, §9.1).

## Cross-standard collisions (read this first)

These concepts exist in more than one standard with different structure. Keep them as distinct types; bridge them only
through `convert/`.

| Concept | DICOM | HL7 v2 | FHIR | go-radx canonical |
|---------|-------|--------|------|-------------------|
| Study | Study Instance UID `(0020,000D)`-keyed dataset grouping | — | `ImagingStudy` resource (UID in `.identifier`) | DICOM: `StudyInstanceUID`; FHIR: `fhir.ImagingStudy`; `convert.DICOMToImagingStudy` |
| Series | Series Instance UID `(0020,000E)` grouping | — | `ImagingStudySeries.uid` | DICOM: `SeriesInstanceUID`; FHIR: `ImagingStudySeries` |
| Instance | SOP Instance (UID `(0008,0018)` + SOP Class UID) | — | `ImagingStudySeriesInstance`; also any "resource instance" | always qualify: `SOPInstance` (DICOM) vs "resource instance" (FHIR); never bare `Instance` |
| Patient | patient-module attributes on every dataset | `PID` segment | `Patient` resource | DICOM attrs / `hl7v2.PID` / `fhir.Patient`; `convert.ADTToPatient` |
| Encounter | loose visit attrs (e.g. AdmissionID) | `PV1` + `ADT` events | `Encounter` resource | `fhir.Encounter`; `convert.ADTToEncounter` (FHIR-only noun) |
| Observation | SR content item (`ContentItem`, `ValueType`) | `OBX` segment | `Observation` resource (`value[x]`) | `dicom` SR `ContentItem` / `hl7v2.OBX` / `fhir.Observation` |
| Report | Structured Report (SR) document tree | `ORU` message | `DiagnosticReport` resource | `StructuredReport` / `ORU` / `fhir.DiagnosticReport` |
| Order | — | `ORM`/`OMG` message (`ORC`+`OBR`) | `ServiceRequest` resource | `ORM` / `fhir.ServiceRequest`; `convert.ORMToServiceRequest` (no FHIR `Order`) |
| Reference | Referenced SOP Instance (UID pair in a sequence) | reference pointers | `Reference` datatype (URL/Identifier) | `fhir.Reference` vs `ReferencedSOPInstance`; never name a DICOM helper `Reference` |
| Identifier | UID (ISO-OID) or attr (PatientID, AccessionNumber) | `CX` (ID + assigning authority) | `Identifier` datatype (system+value) | `dicom.UID` / `hl7v2.CX` / `fhir.Identifier`; DICOM UID → FHIR `Identifier` (`urn:dicom:uid`/`urn:oid`), not a `Reference.reference` |
| UID | ISO-OID string (Study/Series/SOP/Class/TS) | — | stored in `ImagingStudy.*.uid`, or as `Identifier` | `dicom.UID`; distinct from FHIR `Resource.id` |
| Element | Data Element = `(Tag, VR, len, value)` | Field (rough analogue) | `Element` base type (id + extensions) | DICOM: `dicom.Element`/`Tag`; FHIR: `fhir.Element` |
| Context | Application Context UID; Presentation Context | — | — | `dimse.PresentationContext`; not Go `context.Context` |

## DICOM — data model (package `dicom`)

- **Element** (`Element`) — atomic `(Tag, VR, Value Length, Value)` unit of a dataset.
- **Tag** (`Tag`) — 32-bit `(group, element)` identifier, written `(gggg,eeee)`; odd groups are private.
- **VR** (`VR`) — two-letter Value Representation (PS3.5 Table 6.2-1): 34 standard + 4 ambiguous (the ambiguous ones are
  parse-time placeholders, not on-wire VRs).
- **VM** (`VM`) — Value Multiplicity: dictionary-defined count of values in one element (not FHIR cardinality).
- **DataSet** (`DataSet`) — ordered Tag-keyed collection of Elements (note the capital S, per §8.1).
- **Sequence** (`Sequence`) — VR `SQ`: an ordered list of Items, each a nested `DataSet`; nests arbitrarily; defined or
  undefined length. (The prototype dropped these — a foundational defect.)
- **Item** (`Item`) — one nested `DataSet` in an `SQ`, or a fragment container in encapsulated Pixel Data; item tag
  `(FFFE,E000)`, delimiter `(FFFE,E00D)`.
- **Transfer Syntax** (`TransferSyntax`) — UID-identified encoding (byte order, implicit/explicit VR, compression). The
  single `dicom.TransferSyntax` type is reused by `dimse` and `dicomweb`.
- **UID** (`UID`) — dotted-numeric ISO OID, ≤64 chars (VR `UI`); identifies SOP Classes/Instances, Studies, Series,
  transfer syntaxes.
- **SOP Class / SOP Instance** (`SOPClassUID` / `SOPInstanceUID`) — an IOD + its DIMSE services / one concrete object.
- **IOD** / **Module** — abstract data model (PS3.3) / reusable attribute grouping. Qualify "Module" as "DICOM/IOD
  Module" to avoid clashing with Go modules (§7.4).
- **Pixel Data** (`PixelData`) — `(7FE0,0010)`; native (contiguous OB/OW) or encapsulated (Basic/Extended Offset Table +
  per-frame fragments). All length math is bounds-checked (§9.3).
- **Basic Offset Table** (`BasicOffsetTable`) — first Item of encapsulated Pixel Data; 32-bit per-frame offsets.
- **File Meta Information** (`FileMeta`) — group-0002 elements (always Explicit VR LE); group length `(0002,0000)`
  auto-recomputed on write.
- **Preamble / DICM prefix** (`Preamble`) — 128-byte preamble + `DICM` magic starting a Part 10 file; truncation is an
  error (§9.2).
- **Undefined Length** (`UndefinedLength`) — value length `0xFFFFFFFF`, delimiter-terminated; never a literal allocation
  size.
- **Specific Character Set** (`SpecificCharacterSet`) — `(0008,0005)`; encodings for customizable text VRs (ISO 2022,
  UTF-8/GB18030/GBK).
- **PersonName** (`PersonName`) — VR `PN`: up to three `=`-delimited component groups × five `^`-delimited components.
- **Decimal String / Integer String** (`Decimal`) — VRs `DS`/`IS`; lexical-preserving `Decimal` type (shared with FHIR
  `decimal`), beating the prototype's `float64`.
- **Private Creator / Private Block** (`PrivateBlock`) — vendor-reserved 256-element block in an odd group; parsed
  generically (no private SOP-class logic, §3.2).
- **Repeating Group** (`RepeatingGroup`) — `60xx`/`50xx`; dictionary lookup masks the variable bytes.
- **Implicit vs Explicit VR** (`ExplicitVR`) — encoding style set by the Transfer Syntax; the reader must honour the
  negotiated syntax (the prototype always used Implicit VR LE — a defect).
- **generate_uid** (`GenerateUID`) — mint a new UID under the org root; reused by PS3.15 UID remapping.
- **Keyword** (`Keyword`) — human-readable element name (e.g. `PatientName`) from the data dictionary (~5,189 entries);
  resolved from a static dictionary, not reflection.
- **File-set / DICOMDIR** (`FileSet`) — Media Storage Directory + referenced files.

## DIMSE — networking (package `dimse`)

- **Application Entity / AE Title** (`AE` / `AETitle`) — a DICOM network endpoint / its 1–16 char name (Calling vs
  Called). `AETitle` is a named type, never a bare string.
- **Association** (`Association`) — negotiated, stateful connection between two AEs (A-ASSOCIATE → A-RELEASE/A-ABORT).
- **Presentation Context** (`PresentationContext`) — odd-ID-keyed pairing of one Abstract Syntax (a SOP Class) with
  negotiated Transfer Syntax(es).
- **Abstract Syntax** (`AbstractSyntax`, a `SOPClassUID`) — the SOP Class proposed in a presentation context.
- **ACSE / DUL** — association control service element / DICOM Upper Layer (PS3.8). The DUL **state machine**
  (`StateMachine`/DUL FSM) is PS3.8 Table 9-10: 13 states (incl. release-collision Sta9–Sta12), 19 events, 28 actions.
- **PDU** (`PDU`) — Protocol Data Unit (A-ASSOCIATE-RQ/AC/RJ, P-DATA-TF, A-RELEASE-RQ/RP, A-ABORT).
- **PDV** (`PDV` / `PresentationDataValue`) — command/dataset fragment inside a P-DATA-TF, with a message-control header
  whose last-fragment bit the prototype failed to set (root cause of Orthanc aborts).
- **Max PDU Length** (`MaxPDULength`) — negotiated largest acceptable P-DATA-TF; 0 = unlimited.
- **SCU / SCP** — Service Class User (invoker/client) / Provider (performer/server). v1 N-services are SCU-only.
- **C-services** — `CEcho`, `CStore`, `Find`, `Get`, `Move`, `Cancel`. `Association.Find(ctx, q, lvl)` returns
  `iter.Seq2[Status, *dicom.DataSet]` (§8.1); C-MOVE carries a `MoveDestination` AE Title.
- **N-services** — N-CREATE/SET/GET/DELETE/ACTION/EVENT-REPORT. v1: **MPPS** (`MPPS`, N-CREATE/N-SET) and **Storage
  Commitment** (`StorageCommitment`, N-ACTION/N-EVENT-REPORT), SCU only.
- **Priority** (`Priority`) — typed enum (MEDIUM=0x0000 is the footgun the enum guards).
- **Query/Retrieve Level** (`QueryLevel`) — PATIENT/STUDY/SERIES/IMAGE; maps to DICOMweb resource paths.
- **DIMSE Status** (`Status`) — 16-bit outcome (Success/Pending/Warning/Cancel/Failure) with `IsSuccess()` etc.; render
  by name, not hex.
- **A-ABORT vs A-P-ABORT** — user-requested vs provider-initiated abrupt termination; surfaced as typed errors.
- **Presentation-context presets** — curated role bundles (Storage 120/170, Query-Retrieve 13, Verification 1,
  Print 11, Basic Worklist); a go-radx helper concept, not a standards term.

## DICOMweb (package `dicomweb`)

- **WADO-RS** — RESTful retrieval (objects, metadata, rendered, frames, bulkdata); web counterpart of C-GET/C-MOVE.
  v1 implements WADO-RS, not the deferred legacy WADO-URI.
- **STOW-RS** — RESTful store (HTTP POST of instances); web counterpart of C-STORE.
- **QIDO-RS** — RESTful search (query parameters → `application/dicom+json`); web counterpart of C-FIND.
- **Study/Series/Instance web resources** — `/studies/{StudyInstanceUID}/series/{SeriesInstanceUID}/instances/{SOPInstanceUID}`;
  correspond to DIMSE `QueryLevel` STUDY/SERIES/IMAGE.
- **Frames** — `/instances/{uid}/frames/{frameList}`; a *pixel* frame, not a DUL/PDV transport fragment.
- **Bulkdata** (`BulkDataURI`) — large binary values fetched separately via WADO-RS.
- **DICOM JSON / multipart-related** — tag-keyed `application/dicom+json` (PS3.18 Annex F) vs multipart bundling;
  distinct schema from FHIR JSON — `convert/` bridges; never cross-feed serializers.

## HL7 v2 (package `hl7v2`)

- **Message** (`Message`) — segments beginning with MSH; root of the six-level tree. Typed message types (ADT/ORM/ORU/
  ACK) layer over a generic tree; the tree is Go-native 0-based, with string-key accessors mirroring the HL7 1-based spec.
- **Message Type / Trigger Event** (`MessageType` / `TriggerEvent`) — MSH-9 composite `code^trigger^structure`
  (e.g. `ORU^R01^ORU_R01`); the trigger event is only MSH-9.2.
- **Segment** (`Segment`) — three-char-ID line; typed structs (`MSH`, `PID`, `PV1`, `OBR`, `OBX`, `ORC`, `MSA`) are the
  primary API alongside a generic `Segment`.
- **Field / Repetition / Component / Subcomponent** — levels 3–6; absent optionals read as empty, not error.
- **Encoding Characters** (`EncodingCharacters`) — field sep (MSH-1) + component/repetition/escape/subcomponent (MSH-2),
  derived per message (defaults `| ^ ~ \ &`); hardcoding them is a bug for non-standard senders.
- **Escape Sequence** (`Escape`) — `\F\ \R\ \S\ \T\ \E\ \Xdd..\ \.br\` per Chapter 2 §2.10 (separator/highlight/hex/
  rich-text/app-defined; inline charset switches out of floor).
- **MLLP** (`MLLP`) — TCP framing: `0x0B` … `0x1C 0x0D`. go-radx adds context cancellation and a max-frame cap (§9.3).
- **ACK / AckCode** (`Acknowledgment` / `AckCode`) — there is no "NACK" message; a negative ack is an ACK with MSA-1 =
  AE/AR (or enhanced CE/CR). AA/AE/AR is HL7 Table 0008; modelled as a typed enum.
- **Batch (BHS/BTS) / File (FHS/FTS)** (`Batch` / `File`) — optional bulk/file containers; header+trailer present
  together or not at all. "File" here is the HL7 batch container, not a `.dcm` or OS file.
- **Composite datatypes** — `XPN` (person name), `XAD` (address), `CX` (identifier + assigning authority), `CWE` (coded,
  supersedes CE), `HD` (hierarchic designator), `DTM` (variable-precision timestamp — preserve precision, don't
  zero-fill). Typed structs (python-hl7 leaves them stringly-typed).
- **Accessor** (`Accessor`) — 1-based `SEG-Fn-Rn-Cn-Sn` path; typed segment structs are the primary API so callers avoid
  it.
- **Message Control ID** (`MessageControlID`) — MSH-10; locally unique, not a global UID.

## FHIR (package `fhir`)

- **Resource** (`Resource` interface, `ResourceType() string`) — base unit of exchange; `resourceType` discriminator.
- **DomainResource** — Resource + narrative/extensions/contained (most clinical resources).
- **Element / Backbone Element** (`Element` / `BackboneElement`) — base component (id + extensions) / inline nested
  structure (e.g. `ObservationComponent`). FHIR `Element` ≠ DICOM Data Element.
- **Primitive Type** — atomic value; maps to Go scalars except `decimal` → `Decimal` (lexical fidelity).
- **Primitive Extension (`_field`)** — sibling `_`-prefixed JSON object carrying a primitive's id/extensions; must
  round-trip with its value (the subtlest FHIR-JSON mechanic; floor requirement).
- **Complex Datatype** — reusable Element-derived struct (`Coding`, `CodeableConcept`, `Quantity`, `Reference`,
  `Identifier`, `HumanName`, `Period`).
- **Choice Type (`[x]`)** — polymorphic element (one of several types); committed as a sealed value interface with
  `Value()` getter and `SetValueX()` setters enforcing mutual exclusion (§8.1).
- **Cardinality** — `min..max` (0..1 pointer/optional, 1..1 required-validated, `*` slice); required is presence, so a
  valid `false`/`0` must not read as missing.
- **Value Set Binding (strength)** — required/extensible/preferred/example; only **required** strength becomes a closed
  Go enum (defined string type + const set + `ParseXxx`), e.g. `AdministrativeGender`.
- **code** — controlled-token primitive; typed enum under a required binding, else string.
- **Reference** (`Reference`) — URL/Identifier pointer to a resource; `fhir.As[*Patient](ref.Resource)` is the checked
  downcast. Not a DICOM referenced-SOP UID pair.
- **contained / Bundle / Bundle.type** — inline resources / collection container / processing semantics
  (transaction|searchset|document|message|batch|collection). FHIR "transaction" ≠ DICOM/HL7 transaction.
- **OperationOutcome** — structured operation result (issues with severity); parallel to a DIMSE `Status` and an HL7 ACK,
  but its own type.
- **Profile** — constraint `StructureDefinition` (e.g. US Core, deferred); "FHIR profile" ≠ "PS3.15 de-identification
  profile" — always disambiguate.
- **resourceType** — JSON discriminator; verified by `Unmarshal[T]`/`As[T]` (the prototype never checked it).
- **ImagingStudy / Observation / DiagnosticReport / ServiceRequest / Patient / Encounter** — generated resources; see the
  cross-standard table for their DICOM/HL7 counterparts and the `convert/` mappings.

## Go-API canonical types (PRD §8.1)

`Tag`, `UID`, `VR`, `TransferSyntax`, `DataSet`, `Decimal`, `PersonName`, `AETitle`, `Association`,
`PresentationContext`, `Status`, `QueryLevel`, `Priority` (DICOM/DIMSE); `Resource` interface with package-level
`Unmarshal[T]` / `As[T]`; sealed choice-value interfaces with `Value()`/`SetValueX()`; generated enums (e.g.
`AdministrativeGender` + `ParseAdministrativeGender`); typed HL7 v2 segments (`MSH`, `PID`, …) and composite datatypes
(`XPN`, `CX`, …). No type-parameterised methods (Go has no generic methods in 1.26) — dispatch is via package-level
generic functions.
