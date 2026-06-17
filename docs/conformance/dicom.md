# DICOM conformance statement

| Field | Value |
|-------|-------|
| Standard | DICOM (NEMA PS3, 2024 edition referenced) |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | `v1` (tracks the go-radx v1 release line) |
| Status | Authored docs-first; the implementation conforms to this statement |
| Scope authority | This document is the single source of truth for DICOM scope (PRD §6.1) |

This is a DICOM Conformance Statement in the sense of PRD §6.1: go-radx is "100% conformant to an explicitly
declared, versioned subset, verified against reference validators" — not "implements all of DICOM." It declares
exactly which SOP Classes, presentation-context presets, transfer syntaxes, and association-negotiation features the
`dicom` and `dimse` packages support in v1, and the SCU/SCP role for each service. Anything not listed here is out of
scope for v1; growth is a deliberate, reviewed change to this statement, not a silent capability drift.

The subset is seeded from the `pynetdicom` parity floor (PRD §6.2), enumerated by the Codex audit at
`/tmp/go-radx-codex-reviews/parity-pynetdicom.md`, and extended radiology-first. Counts in this document trace back to
that audit and to `pynetdicom/sop_class.py` and `pynetdicom/presentation.py`. Where go-radx supports fewer SOP Classes
than the full `pynetdicom` preset, that is stated explicitly with the reason.

This statement covers the DICOM data layer (Part 10 files, datasets, transfer syntaxes) and DIMSE networking (Part 7,
Part 8). DICOMweb (WADO-RS / STOW-RS / QIDO-RS) has its own [conformance statement](./dicomweb.md); the two share the
`dicom.TransferSyntax` and SOP Class UID vocabulary but negotiate transport differently.

## Scope summary

In scope for v1:

- The radiology-first **Storage** SOP Class set as both Storage SCU and Storage SCP.
- The **DICOM Structured Report (SR) document model** — the content-item tree (`dicom.ContentItem`, its `ValueType`
  vocabulary, `ConceptNameCode`, and relationship types) plus SR-document read and build — for the Basic Text,
  Enhanced, and Comprehensive SR SOP Classes. These are validated, IOD-aware targets, not opaque transport, because the
  convert layer bridges them to FHIR `DiagnosticReport`/`Observation` (PRD §5.1 step 6). The data-layer contract is
  defined in `docs/reference/dicom.md`.
- **Verification** (C-ECHO) as SCU and SCP.
- **Query/Retrieve** (C-FIND, C-GET, C-MOVE) under the Patient Root and Study Root information models, as SCU and SCP.
- **Modality Worklist** (C-FIND) as SCU, with a reference Modality Worklist SCP so the leg is testable end to end.
- **Modality Performed Procedure Step (MPPS)** as SCU and SCP (N-CREATE, N-SET; `MPPSProvider`).
- **Storage Commitment Push Model** as SCU and SCP (N-ACTION, N-EVENT-REPORT; `StorageCommitmentProvider`, with the
  SCP reporting on the same association).
- Uncompressed transfer syntaxes for read and write. The recognised compressed transfer syntaxes (RLE, the JPEG
  families, JPEG 2000, HTJ2K) also read and write at the Part 10 / dataset level: the main dataset parses in full
  and the encapsulated pixel stream is retained verbatim and re-emitted byte-identically. Pixel decode/encode is a
  separate concern, available only where a pure-Go or optional-CGo codec is built in.
- **DICOMDIR file-sets** (PS3.10 §8; PS3.3 Annex F): load and query an existing file-set through the glossary-named
  `dicom.FileSet` (`OpenFileSet`, record hierarchy, `Find`/`FindValues`, member `Load`), and create and write a new
  one from Part 10 files with `dicom.FileSetBuilder` (Patient/Study/Series/Instance records, conformant generated
  File IDs, offset-linked Directory Record Sequence). The DICOMDIR must be encoded in Explicit VR Little Endian
  (PS3.10 §8.6); any other transfer syntax, cyclic or out-of-range record offsets, and root-escaping Referenced File
  IDs fail closed with typed errors. Referenced File IDs are read permissively but traversal-safe: lowercase or
  over-long components common in real-world file-sets are accepted (matching pydicom's read behaviour), while
  strictly conformant PS3.10 §8.2/§8.5 File IDs are enforced on write.
- Association negotiation: presentation-context negotiation, maximum PDU length, SCP/SCU role selection, the
  asynchronous operations window, user identity (types 1–5), SOP Class extended and common extended negotiation, and
  DIMSE-TLS (TLS 1.2+ with peer verification and optional mutual-TLS). The asynchronous-operations window is negotiated
  but the acceptor windows to (1,1); concurrency is delivered through Go goroutines, not the DICOM async-ops mechanism
  (see the negotiation table below).

Out of scope for v1 (deferred, designed-for but not implemented — PRD §3.2, §5.1):

- The MPPS Retrieve (N-GET) and Notification (N-EVENT-REPORT) SOP classes, and the separate-association Storage
  Commitment report leg (the SCP opening a new association back to the SCU to report later). The MPPS and Storage
  Commitment SCP sides themselves now ship (`MPPSProvider`, `StorageCommitmentProvider`).
- All other DIMSE-N services as either role: N-GET, N-DELETE outside the two SCU flows above, Print Management
  (N-CREATE/N-SET/N-GET/N-ACTION/N-DELETE/N-EVENT-REPORT), Unified Procedure Step (UPS), RT Machine Verification, Media
  Creation Management, Display System, Application Event Logging, Instance Availability Notification, Storage
  Management, and Substance Administration.
- Non-radiology Query/Retrieve models: Color Palette, Hanging Protocol, Defined Procedure Protocol, Implant Template,
  Protocol Approval, Relevant Patient Information, Repository Query, and the Composite Instance Root / without-bulkdata
  retrieve variants.
- Compressed-codec encode where no codec exists; the request returns a typed `dicom.ErrCodecUnavailable`.
- Updating an existing DICOMDIR file-set in place: removing or re-staging instances and PS3.11 media application
  profile enforcement are deferred; v1 file-set writing is create-from-scratch.

## Go API entry points

This statement is realised through the public API committed in the reference docs (`docs/reference/dicom.md`,
`docs/reference/dimse.md`) and the glossary. The load-bearing identifiers a consumer touches to exercise the declared
scope are below; they exist so the conformance scope is reachable through named types, never bare strings.

```go
// Transfer syntaxes are named dicom.TransferSyntax constants, reused by dimse and dicomweb (glossary rule 1).
const (
    ImplicitVRLittleEndian         dicom.TransferSyntax = "1.2.840.10008.1.2"
    ExplicitVRLittleEndian         dicom.TransferSyntax = "1.2.840.10008.1.2.1"
    DeflatedExplicitVRLittleEndian dicom.TransferSyntax = "1.2.840.10008.1.2.1.99"
    ExplicitVRBigEndian            dicom.TransferSyntax = "1.2.840.10008.1.2.2" // retired, read + write
    RLELossless                    dicom.TransferSyntax = "1.2.840.10008.1.2.5"
)

// DefaultTransferSyntaxes is the proposed list when a presentation context is built without an explicit list.
// The element order is the proposal preference: the acceptor may pick the first transfer syntax it supports, so
// Explicit VR Little Endian is listed first as the most interoperable default. This order is identical in
// docs/reference/dimse.md.
var DefaultTransferSyntaxes = []dicom.TransferSyntax{
    ExplicitVRLittleEndian, ImplicitVRLittleEndian, DeflatedExplicitVRLittleEndian, ExplicitVRBigEndian,
}

// Presentation-context presets: curated bundles for the roles this statement declares (a go-radx helper concept,
// not a standards term). Each returns proposals keyed by odd presentation-context IDs over DefaultTransferSyntaxes.
func VerificationContexts() []dimse.PresentationContext      // 1 context  — Verification SOP Class
func StorageContexts() []dimse.PresentationContext           // radiology-first Storage set (see table below)
func QueryRetrieveContexts() []dimse.PresentationContext     // Patient Root + Study Root C-FIND/C-GET/C-MOVE
func QueryRetrieveWithStorageContexts() []dimse.PresentationContext // Q/R + validated Storage, one ID sequence (C-GET SCU)
func BasicWorklistContexts() []dimse.PresentationContext     // Modality Worklist Information Model — FIND
func ModalityPerformedContexts() []dimse.PresentationContext // MPPS SOP Class (MPPS SCU and SCP)
func StorageCommitmentContexts() []dimse.PresentationContext // Storage Commitment Push Model SOP Class (SCU and SCP)
```

The verb-level entry points that drive each service are `Association.Echo`, `Association.Store`, `Association.Find`,
`Association.Get`, `Association.Move` (SCU); the `dimse.Handler` interface methods `Echo`/`Store`/`Find`/`Get`/`Move`
(SCP); `Association.MPPS()` / `Association.StorageCommitment()` for the two N-service SCU flows; and `MPPSProvider`
/ `StorageCommitmentProvider` (registered as the `dimse.Server` handler) for their SCP sides.

`Find`/`Get`/`Move` extend the PRD §8.1 `Find(ctx, query, level)` form with a trailing functional-options variadic
(`opts ...dimse.QueryOption`) — a deliberate, recorded extension that keeps the type-level shape (receiver, value
parameters, `iter.Seq2[Status, *dicom.DataSet]` return). The worked examples below call the PRD-committed form without
options; the full signature with `opts ...dimse.QueryOption` is documented in `docs/reference/dimse.md`.

## SOP Classes

### Verification

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Verification | `1.2.840.10008.1.1` | Yes | Yes |

C-ECHO is the simplest end-to-end check and the first leg of the walking skeleton (PRD M2). Exposed by
`VerificationContexts()`; one presentation context, matching the `pynetdicom` Verification preset.

### Storage (radiology-first set)

go-radx supports a curated, radiology-first Storage SOP Class set as both **Storage SCU** (C-STORE invoker) and
**Storage SCP** (C-STORE provider, e.g. the reference store daemon). The full `pynetdicom` selected-Storage preset is
120 SOP Classes (`StoragePresentationContexts`) and the all-Storage preset is 170
(`AllStoragePresentationContexts`); go-radx declares conformance for the radiology-relevant subset below as the
**supported, validated** Storage set.

The distinction is deliberate. A SOP Class in the supported set is round-trip-tested (parse, write, `dciodvfy`, and
Orthanc/dcm4chee-arc interop). A transport-only forwarding preset that proposes the full registered Storage set
(`AllStorageContexts()`) — accepted on the wire and stored or forwarded byte-for-byte, with no IOD-semantic claim — is
**NOT YET SHIPPED**; the validated `StorageContexts()` set below is the only Storage preset v1 exposes.

Supported (validated) Storage SOP Classes:

| Category | SOP Class | UID |
|----------|-----------|-----|
| CR / DX | Computed Radiography Image Storage | `1.2.840.10008.5.1.4.1.1.1` |
| CR / DX | Digital X-Ray Image Storage — For Presentation | `1.2.840.10008.5.1.4.1.1.1.1` |
| CR / DX | Digital X-Ray Image Storage — For Processing | `1.2.840.10008.5.1.4.1.1.1.1.1` |
| Mammo | Digital Mammography X-Ray Image Storage — For Presentation | `1.2.840.10008.5.1.4.1.1.1.2` |
| Mammo | Digital Mammography X-Ray Image Storage — For Processing | `1.2.840.10008.5.1.4.1.1.1.2.1` |
| Mammo | Breast Tomosynthesis Image Storage | `1.2.840.10008.5.1.4.1.1.13.1.3` |
| CT | CT Image Storage | `1.2.840.10008.5.1.4.1.1.2` |
| CT | Enhanced CT Image Storage | `1.2.840.10008.5.1.4.1.1.2.1` |
| CT | Legacy Converted Enhanced CT Image Storage | `1.2.840.10008.5.1.4.1.1.2.2` |
| MR | MR Image Storage | `1.2.840.10008.5.1.4.1.1.4` |
| MR | Enhanced MR Image Storage | `1.2.840.10008.5.1.4.1.1.4.1` |
| MR | Enhanced MR Color Image Storage | `1.2.840.10008.5.1.4.1.1.4.3` |
| MR | Legacy Converted Enhanced MR Image Storage | `1.2.840.10008.5.1.4.1.1.4.4` |
| US | Ultrasound Image Storage | `1.2.840.10008.5.1.4.1.1.6.1` |
| US | Ultrasound Multi-frame Image Storage | `1.2.840.10008.5.1.4.1.1.3.1` |
| XA / XRF | X-Ray Angiographic Image Storage | `1.2.840.10008.5.1.4.1.1.12.1` |
| XA / XRF | Enhanced XA Image Storage | `1.2.840.10008.5.1.4.1.1.12.1.1` |
| XA / XRF | X-Ray Radiofluoroscopic Image Storage | `1.2.840.10008.5.1.4.1.1.12.2` |
| XA / XRF | Enhanced XRF Image Storage | `1.2.840.10008.5.1.4.1.1.12.2.1` |
| NM / PET | Nuclear Medicine Image Storage | `1.2.840.10008.5.1.4.1.1.20` |
| NM / PET | Positron Emission Tomography Image Storage | `1.2.840.10008.5.1.4.1.1.128` |
| NM / PET | Enhanced PET Image Storage | `1.2.840.10008.5.1.4.1.1.130` |
| SC | Secondary Capture Image Storage | `1.2.840.10008.5.1.4.1.1.7` |
| SC | Multi-frame Single Bit Secondary Capture Image Storage | `1.2.840.10008.5.1.4.1.1.7.1` |
| SC | Multi-frame Grayscale Byte Secondary Capture Image Storage | `1.2.840.10008.5.1.4.1.1.7.2` |
| SC | Multi-frame Grayscale Word Secondary Capture Image Storage | `1.2.840.10008.5.1.4.1.1.7.3` |
| SC | Multi-frame True Color Secondary Capture Image Storage | `1.2.840.10008.5.1.4.1.1.7.4` |
| Derived | Segmentation Storage | `1.2.840.10008.5.1.4.1.1.66.4` |
| Derived | Parametric Map Storage | `1.2.840.10008.5.1.4.1.1.30` |
| Presentation | Grayscale Softcopy Presentation State Storage | `1.2.840.10008.5.1.4.1.1.11.1` |
| Presentation | Color Softcopy Presentation State Storage | `1.2.840.10008.5.1.4.1.1.11.2` |
| SR | Basic Text SR Storage | `1.2.840.10008.5.1.4.1.1.88.11` |
| SR | Enhanced SR Storage | `1.2.840.10008.5.1.4.1.1.88.22` |
| SR | Comprehensive SR Storage | `1.2.840.10008.5.1.4.1.1.88.33` |
| SR | Key Object Selection Document Storage | `1.2.840.10008.5.1.4.1.1.88.59` |
| Encapsulated | Encapsulated PDF Storage | `1.2.840.10008.5.1.4.1.1.104.1` |

The three SR SOP Classes — Basic Text SR, Enhanced SR, and Comprehensive SR — are validated as IOD-aware SR
documents, not as opaque transport. v1 parses and builds their content-item tree (`dicom.ContentItem`, the `ValueType`
vocabulary, `ConceptNameCode`, and relationship types, defined in `docs/reference/dicom.md`); the convert layer bridges
that tree to FHIR `DiagnosticReport`/`Observation` (PRD §5.1 step 6). Round-trip validation therefore covers the SR
document structure, not merely byte-for-byte storage. Secondary Capture is included so non-radiology-native content
(rendered reports, screenshots) has a conformant store target. The supported set is intentionally narrower than the
120-Class `pynetdicom` selection: it is the radiology workflow's actual SOP Classes plus the SR and
encapsulated-document classes the convert layer needs, not the long tail of ophthalmic, dermatologic, RT-treatment, and
waveform classes that v1 does not validate.

A transport-only forwarding preset that proposes every registered Storage SOP Class (the 170-Class transport set) for
consumers who need a forwarding store — `AllStorageContexts()`, where instances of unsupported classes would be stored
and retrieved verbatim with no IOD-level claim — is **NOT YET SHIPPED** in v1.

### Query/Retrieve

go-radx supports the Patient Root and Study Root information models for all three retrieve verbs, as SCU and SCP. The
Patient/Study Only model and the Composite Instance Root / without-bulkdata / repository variants are out of v1 scope.

| Information model | Service | SOP Class UID | SCU | SCP |
|-------------------|---------|---------------|-----|-----|
| Patient Root | C-FIND | `1.2.840.10008.5.1.4.1.2.1.1` | Yes | Yes |
| Patient Root | C-MOVE | `1.2.840.10008.5.1.4.1.2.1.2` | Yes | Yes |
| Patient Root | C-GET | `1.2.840.10008.5.1.4.1.2.1.3` | Yes | Yes |
| Study Root | C-FIND | `1.2.840.10008.5.1.4.1.2.2.1` | Yes | Yes |
| Study Root | C-MOVE | `1.2.840.10008.5.1.4.1.2.2.2` | Yes | Yes |
| Study Root | C-GET | `1.2.840.10008.5.1.4.1.2.2.3` | Yes | Yes |

`QueryRetrieveContexts()` returns 6 contexts (Patient Root + Study Root C-FIND/C-GET/C-MOVE), filtered from the
`pynetdicom` 13-Class Q/R floor down to the two information models go-radx validates; the 13-Class floor is the upstream
reference, not go-radx's preset count. The Query/Retrieve Levels supported are PATIENT, STUDY, SERIES, and IMAGE
(`dimse.QueryLevel`); the SCU always writes the requested level into `(0008,0052)` before sending, and
C-FIND/C-GET/C-MOVE expose results as `iter.Seq2[Status, *dicom.DataSet]` iterators with C-CANCEL on early iterator exit
(PRD §8.1, glossary).

### Modality Worklist

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Modality Worklist Information Model — FIND | `1.2.840.10008.5.1.4.31` | Yes | Yes (reference) |

Worklist is a C-FIND with the Modality Worklist information model. v1 ships both the SCU and a reference Modality
Worklist SCP (PRD §5.1 step 2) so the order-to-worklist leg is testable end to end. Exposed by
`BasicWorklistContexts()`, one context, matching the single-Class `pynetdicom` Basic Worklist preset.

### Modality Performed Procedure Step (MPPS) — SCU and SCP

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Modality Performed Procedure Step | `1.2.840.10008.3.1.2.3.3` | Yes (N-CREATE, N-SET) | Yes (N-CREATE, N-SET; `MPPSProvider`) |

MPPS reports procedure-step start and completion (PRD §5.1 step 3.5). The SCU `Association.MPPS()` returns an `*MPPS`
with `Create` (N-CREATE, status "IN PROGRESS") and `Set` (N-SET, "COMPLETED"/"DISCONTINUED"). The SCP `MPPSProvider`
(an `NCreateHandler` and `NSetHandler` registered as the `dimse.Server` handler) opens the step on N-CREATE — after
checking the Affected SOP Instance UID and an `IN PROGRESS` Performed Procedure Step Status are present — and advances
it on N-SET, rejecting an N-SET against an unknown step or one already in a final state. Persistence is delegated to an
`MPPSStore` (with a `MemoryMPPSStore` default). The N-services used are N-CREATE and N-SET; the MPPS Retrieve (N-GET)
and Notification (N-EVENT-REPORT) SOP classes are not in scope. Exposed for negotiation by `ModalityPerformedContexts()`.

### Storage Commitment — SCU and SCP

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Storage Commitment Push Model | `1.2.840.10008.1.20.1` | Yes (N-ACTION, N-EVENT-REPORT) | Yes (N-ACTION, N-EVENT-REPORT; `StorageCommitmentProvider`) |

Storage Commitment confirms that an archive has taken custody of stored instances (PRD §5.1 step 4.5). The SCU
`Association.StorageCommitment().Request(...)` issues the N-ACTION request and receives the asynchronous
N-EVENT-REPORT result as a `StorageCommitmentResult`. The SCP `StorageCommitmentProvider` (an `NActionHandler`
registered as the `dimse.Server` handler) answers the N-ACTION (action type 1), decides per-instance commitment
through a `CommitmentDecider` hook, then reports the outcome to the SCU with an N-EVENT-REPORT on the same association
(event type 1 complete, event type 2 failures exist). The separate-association report leg (the SCP opening a new
association to report later) is deferred. Exposed for negotiation by `StorageCommitmentContexts()`.

### Presentation-context preset summary

| Preset (`dimse` helper) | Role | Context count |
|-------------------------|------|---------------|
| `VerificationContexts()` | Verification SCU/SCP | 1 |
| `StorageContexts()` | Storage SCU/SCP (validated radiology set) | 36 (table above) |
| `AllStorageContexts()` | Storage transport-only | NOT YET SHIPPED (would be 170, the registered Storage set) |
| `QueryRetrieveContexts()` | Patient Root + Study Root Q/R | 6 |
| `QueryRetrieveWithStorageContexts()` | Q/R + validated Storage under one ID sequence (same-association C-GET) | 42 |
| `BasicWorklistContexts()` | Modality Worklist FIND | 1 |
| `ModalityPerformedContexts()` | MPPS SCU | 1 |
| `StorageCommitmentContexts()` | Storage Commitment SCU | 1 |

Presets the `pynetdicom` floor exports but go-radx does not ship as v1 presets: Print Management (11), Unified
Procedure Step (5), RT Machine Verification (2), Implant Template (9), Non-Patient Object (9), Color Palette (3),
Hanging Protocol (3), Defined Procedure Protocol (3), Relevant Patient Information (3), Substance Administration (2),
and the remaining single-Class management presets. These are out of v1 scope (PRD §3.2).

## Transfer syntaxes

A transfer syntax is the UID-identified encoding: byte order, implicit-versus-explicit VR, and compression. go-radx
divides transfer syntaxes into three tiers, and this distinction is the heart of the codec strategy (PRD §7.3).

### Tier 1 — uncompressed: read and write, always available

These are pure Go, require no build tags, and are always available. The dataset codec reads and writes all four.
The recognised encapsulated syntaxes in Tiers 2 and 3 are also readable and writable at the dataset level — their
main dataset is Explicit VR LE (PS3.5 A.4) and the `(7FE0,0010)` fragment stream is retained verbatim, undecoded —
so `dicom.Read`/`Write` round-trip a compressed file byte-identically regardless of which pixel codecs are built in.
The retained stream is structurally validated on read (item tags only, defined even item lengths, a zero-length
Sequence Delimitation Item) and its accumulated size is bounded by the same `WithMaxElementLen` cap that bounds a
native pixel value, so many small fragments cannot grow memory without limit. `dicom.ReadPixelData` and
`ReadPixelDataFrom` share this Read path: the whole dataset is parsed — a malformed element after the pixel data
fails the call, and memory holds the full dataset, not just the elements up to the pixels — before the pixel element
is bound to its geometry. An unrecognised or private transfer syntax is rejected fail-closed.

For large objects, `ReadFile` with `WithDeferredValues(threshold)` (pydicom's `defer_size` analogue, PRD §6.2) skips
any element value larger than the threshold and records its byte window instead, so memory stays bounded; the value
loads from the source file on first access — transparently through the dataset accessors and the write path, or
explicitly via `(*DeferredValue).Load`. Loads re-validate the recorded window against the file (a shrunk, replaced,
or no-longer-parseable source is a typed `*DeferredLoadError`, never a panic), and a deferred encapsulated pixel
stream is re-parsed through the same structural validator the read used. The option is `ReadFile`-only: `Read`,
`DecodeDataSet`, and Deflated Explicit VR LE reject it fail-closed because their sources cannot be re-read at a
recorded offset. The default read path is unchanged: without the option every value is materialised exactly as
before.

| Transfer syntax | UID | Read | Write |
|-----------------|-----|------|-------|
| Implicit VR Little Endian | `1.2.840.10008.1.2` | Yes | Yes |
| Explicit VR Little Endian | `1.2.840.10008.1.2.1` | Yes | Yes |
| Deflated Explicit VR Little Endian | `1.2.840.10008.1.2.1.99` | Yes | Yes |
| Explicit VR Big Endian (retired) | `1.2.840.10008.1.2.2` | Yes | Yes |

Explicit VR Little Endian is the default for new files and the mandatory file-meta encoding. The reader honours the
transfer syntax actually in effect rather than assuming Implicit VR LE; the prototype's always-Implicit decode was a
foundational defect (PRD §2.2).

### Tier 2 — RLE Lossless: read and write, always available, pure Go

| Transfer syntax | UID | Pixel decode | Pixel encode |
|-----------------|-----|--------------|--------------|
| RLE Lossless | `1.2.840.10008.1.2.5` | Yes (pure Go) | Yes (pure Go) |

RLE is encapsulated pixel data but uses a simple run-length scheme implementable in pure Go, so it is always available
without CGo.

### Tier 3 — compressed JPEG family: negotiated always, pixel codec optional via CGo

These are accepted at the dataset and association level always (the encapsulated bytes are parsed, stored, and
forwarded). Pixel-frame **decode** is provided by optional CGo codecs, each behind its own build tag so a deployment
links only the C libraries it needs. Pixel-frame **encode** (and therefore transcode) is supported only where a go-radx
encoder exists and is pixel-exact: RLE Lossless first (Tier 2, pure Go), then JPEG 2000 Lossless (OpenJPEG) and JPEG-LS
Lossless (CharLS). With the relevant tag disabled, requesting decoded frames from one of these instances returns the
typed `dicom.ErrCodecUnavailable` naming the transfer syntax: a clear failure, never a build break or a silent
partial image (PRD §7.3).

The CGo codecs and their build tags and backing libraries:

- `dicom_openjpeg` — OpenJPEG (`libopenjp2`): JPEG 2000 and High-Throughput JPEG 2000.
- `dicom_libjpeg` — libjpeg-turbo (`libturbojpeg`): JPEG Baseline and Extended (lossy DCT processes), and JPEG Lossless / Lossless SV1 (predictive lossless, decode-only).
- `dicom_charls` — CharLS (`charls`): JPEG-LS Lossless and Near-Lossless.

The "Pixel decode" column names the build tag that enables decode (empty cell with `ErrCodecUnavailable` until that tag
is set). "Pixel encode" marks each syntax decode-only versus decode+encode.

| Transfer syntax | UID | Negotiate / transport | Pixel decode | Pixel encode |
|-----------------|-----|-----------------------|--------------|--------------|
| JPEG Baseline (8-bit) | `1.2.840.10008.1.2.4.50` | Always | `dicom_libjpeg` | No (decode-only) |
| JPEG Extended (12-bit) | `1.2.840.10008.1.2.4.51` | Always | `dicom_libjpeg` | No (decode-only) |
| JPEG Lossless, Non-Hierarchical | `1.2.840.10008.1.2.4.57` | Always | `dicom_libjpeg` | No (decode-only) |
| JPEG Lossless SV1 | `1.2.840.10008.1.2.4.70` | Always | `dicom_libjpeg` | No (decode-only) |
| JPEG-LS Lossless | `1.2.840.10008.1.2.4.80` | Always | `dicom_charls` | `dicom_charls` (lossless) |
| JPEG-LS Near-Lossless | `1.2.840.10008.1.2.4.81` | Always | `dicom_charls` | No (decode-only) |
| JPEG 2000 Lossless | `1.2.840.10008.1.2.4.90` | Always | `dicom_openjpeg` | `dicom_openjpeg` (lossless) |
| JPEG 2000 | `1.2.840.10008.1.2.4.91` | Always | `dicom_openjpeg` | No (decode-only) |
| HTJ2K Lossless | `1.2.840.10008.1.2.4.201` | Always | `dicom_openjpeg` | No (decode-only) |
| HTJ2K with RPCL Options | `1.2.840.10008.1.2.4.202` | Always | `dicom_openjpeg` | No (decode-only) |
| HTJ2K | `1.2.840.10008.1.2.4.203` | Always | `dicom_openjpeg` | No (decode-only) |

The JPEG Lossless processes `1.2.840.10008.1.2.4.57` (Process 14) and `1.2.840.10008.1.2.4.70` (Process 14, SV1) are the
predictive lossless JPEG processes. libjpeg-turbo 3.x decodes them at 2..16-bit precision through its lossless decode
path, so under `dicom_libjpeg` a `.57`/`.70` instance decodes; without the tag it degrades to `ErrCodecUnavailable`.
Decode is predictor-agnostic (the predictor selection value is read from the codestream), so the same codec serves both
the general Process 14 (`.57`) and the SV1 (`.70`) forms. Re-encoding to a lossless JPEG syntax is deferred: the codec
is decode-only, consistent with the lossy JPEG syntaxes.

The codec set built behind the CGo build tags is finalised in the M8 hardening milestone; this statement is updated when
a codec moves from decode-only to encode-capable. Codecs that `pynetdicom` registers but go-radx does not target for v1
pixel handling (JPIP, MPEG-2/4, HEVC/H.265, JPEG XL, SMPTE ST 2110, multi-component JPEG 2000) are negotiable for
transport but have no go-radx pixel codec.

Selecting a codec is explicit, and **transcoding is off by default**: go-radx never re-encodes pixel data unless the
consumer opts in (PRD §8.2 opinionated default). Reading a file preserves its transfer syntax — encapsulated pixel
data is retained verbatim, never decoded on the read path. Transcoding is the deliberate, opt-in
`dicom.Transcode` call, written back to the dataset through `File.SetPixelData`, and surfaced on the CLI as
`radx store --transcode-to` (see `docs/reference/cli.md`). The seam reconciles the Image Pixel attributes with the
decoded bytes: planar configuration follows the interleaved decoder output, the photometric interpretation follows
the decoded colour model (the JPEG 2000 transform terms decode to RGB, `YBR_FULL_422` decodes to `YBR_FULL`, and a
colour term whose decoded layout cannot be determined fails closed), the frame count follows the actual frames, and
the stale offset-table and total-length elements (`(7FE0,0001)`/`(7FE0,0002)`/`(7FE0,0003)`) are removed. Lossy
bookkeeping is preserved per PS3.3 C.7.6.1.1.5: a lossy source syntax (`TransferSyntax.IsLossy`) forces
`(0028,2110)` to `01`, and the ratio/method attributes are never invented.

Separately from transcoding, two explicit colour helpers operate on a decoded `Frame` without touching the transfer
syntax. `dicom.ApplyColorLUT` expands a `PALETTE COLOR` frame to interleaved RGB through the Red/Green/Blue Palette
Color Lookup Tables (the non-segmented path, PS3.3 C.7.9 and C.7.6.3.1.5; segmented LUT data is out of scope).
`dicom.ConvertColorSpace` converts between `YBR_FULL` and `RGB`, and from `YBR_FULL_422` to either, using the
PS3.3 C.7.6.3.1.2 full-range equations and handling `PlanarConfiguration` 0 and 1. The limited-range
`YBR_PARTIAL_*` terms and the JPEG-2000-internal ICT/RCT transforms are out of scope: the latter are applied inside
the J2K codec so decoded frames are already RGB.

### Performance baseline

The Part 10 dataset decode and per-transfer-syntax codec hot paths carry committed benchmark baselines so a regression
is a visible, reviewable change rather than silent drift (PRD §9.3, minimise allocations in hot paths). The baselines
are recorded as `go test -bench` output that `benchstat` consumes directly.

The default-build baseline, [`benchmarks/dicom-baseline.txt`](benchmarks/dicom-baseline.txt), is the pure-Go build
(`CGO_ENABLED=0`, no codec build tags). It covers `BenchmarkReadFile` (the full Part 10 decode), the bare-dataset
`BenchmarkDecodeDataSet` and `BenchmarkEncodeDataSet`, and the pure-Go RLE Lossless `BenchmarkRLECodecDecode` and
`BenchmarkRLECodecEncode`. `BenchmarkReadFile` is the load-bearing measurement: it records the Part 10 decode
allocation profile, where the per-decode `B/op` is roughly twice the on-disk file size, so the known Part 10 decode
allocation cost is defended against silent growth.

The CGo-codecs baseline, [`benchmarks/dicom-codecs-baseline.txt`](benchmarks/dicom-codecs-baseline.txt), is recorded
with the `dicom_openjpeg dicom_libjpeg dicom_charls` build tags (the CI `codecs` job native-library matrix). It covers
per-transfer-syntax decode, and lossless encode where a codec is encode-capable, across the compressed fixtures:
JPEG-LS via CharLS, JPEG Baseline / Extended / Lossless via libjpeg-turbo, and JPEG 2000 and HTJ2K via OpenJPEG. The
codec benchmarks that require a C library live behind the matching build tag, so the default build compiles and runs
the pure-Go benchmarks unchanged.

Regenerate a baseline with the command recorded in each file's header, and compare a candidate run with
`benchstat <baseline>.txt <candidate>.txt`.

## Association negotiation

go-radx negotiates associations through `dimse.AE`, `AE.Associate(...)`, and `dimse.NewServer(...)`. Every feature in
the table ships today, negotiated against the `pynetdicom` floor (PRD §6.2). The asynchronous-operations window is
negotiated and echoed honestly but the acceptor windows to (1,1): go-radx delivers concurrency through Go goroutines and
`context.Context`, not the DICOM async-ops mechanism, so the negotiated window is the truthful (1,1), not real
asynchronous delivery.

| Feature | Support | Notes |
|---------|---------|-------|
| Presentation-context negotiation | Yes | Odd-ID-keyed; per-context accepted transfer syntax; rejection reason codes |
| Maximum PDU length | Yes | `WithMaxPDULength(n)`; default `dimse.MaxPDULength` (16382); 0 = "no maximum specified" |
| SCP/SCU role selection | Yes | `WithRoleSelection(...)`; required for C-GET (requestor accepts the Storage SCP role) |
| Asynchronous operations window | Yes (negotiated, window 1) | `WithAsyncOps(invoked, performed)`; acceptor echoes (1,1) — concurrency via goroutines, not DICOM async-ops |
| User identity negotiation | Yes | `WithUserIdentity(...)`: username, passcode, Kerberos, SAML, JWT (types 1–5); acceptor seam `WithAuthenticator(...)` |
| SOP Class extended negotiation | Yes | `WithExtendedNegotiation(...)`: per-SOP-class service-class application info |
| SOP Class common extended negotiation | Yes | `WithCommonExtendedNegotiation(...)` |
| DIMSE-TLS | Yes | `WithTLS(cfg)`; TLS 1.2+ (prefer 1.3), peer verification on by default, optional mutual-TLS (PRD §9.7) |

### DUL state machine

The association is driven by a faithful PS3.8 Table 9-10 DICOM Upper Layer state machine: 13 states including the
release-collision states Sta9–Sta12 that the prototype omitted, 19 events, and 28 actions. Malformed or unexpected
PDUs are turned into a provider-sourced A-ABORT (via Evt19 / action AA-8) with the correct reason before the
connection closes, rather than a panic or a half-open association. A-ABORT (user) and A-P-ABORT (provider) are
distinguished and surfaced as a typed `dimse.AbortError`; association rejection surfaces as `dimse.AssociationError`
carrying source, reason, and result. This is a parity-floor requirement and the fix for the prototype's Orthanc aborts
(PRD §2.2, §6.2).

### Calling/Called AE Title enforcement

The reference SCP can require a specific Called AE Title (`WithRequireCalledAETitle`) and an allow-list of Calling AE
Titles (`WithRequireCallingAETitles`). `AETitle` is a validated named type (1–16 characters, default repertoire),
never a bare string.

### User identity negotiation

User identity is the only association-level authentication DICOM defines (PS3.7 D.3.3.7). An SCU presents an identity
with `WithUserIdentity(...)` — username, username and passcode, Kerberos service ticket, SAML assertion, or JSON Web
Token (types 1–5). The acceptor authenticates it through `WithAuthenticator(...)`, the same negotiation seam
`WithAssociationAuthorizer(...)` uses: the authenticator runs after the Calling-AE check and before presentation-context
matching, and a non-nil error rejects the association with an A-ASSOCIATE-RJ before any service runs. When the SCU asks
for a positive response, the authenticator's returned bytes are echoed back as the user-identity server response,
readable via `Association.UserIdentityResponse()`. Identity fields hold secrets (passcodes, tokens) that are never
logged or written to any catalogue (PRD §9.8).

### Negotiation hardening

The acceptor rejects a malformed or non-conformant A-ASSOCIATE-RQ deterministically before any policy or service runs:
an unsupported DUL protocol version is refused with a service-provider-ACSE A-ASSOCIATE-RJ (protocol-version-not-
supported); an RQ proposing more than the 128-presentation-context limit (PS3.8 §7.1.1.13) is refused with a
service-provider-ACSE rejection; and a Called or Calling AE Title field carrying a non-conformant character or an
over-length value is refused with a service-user rejection naming the bad title (PS3.5 VR AE, PS3.8 Table 9-21).

## Behaviour and error model

DICOM behaviour in go-radx follows the PRD's honest-failure and hostile-input rules; the conformance-relevant
guarantees are:

- **DIMSE status is typed.** `dimse.Status` is `struct { Code uint16 }`: the 16-bit status code is the only stored
  field. Its category and human-readable meaning are derived, not settable — `(Status) Category() StatusCategory` and
  `(Status) Meaning() string` resolve them from the per-service-class status table, and the predicates
  `IsSuccess`/`IsPending`/`IsWarning`/`IsFailure`/`IsCancel` classify the code so a status renders by name and class,
  never as bare hex (PRD §8.2). A status is constructed from named constants (for example `dimse.StatusStoreSuccess`,
  `dimse.StatusStoreCannotUnderstand`) or via `dimse.NewStatus(code uint16, sc dimse.ServiceClass) Status`; there are no
  caller-settable category or meaning fields, so a handler cannot author a status whose category contradicts its code.
  SCP handlers return these named constants rather than struct literals. Pending statuses drive the query/retrieve
  iterators; a terminal failure or warning ends them with the carried status.
- **Fail-closed on partial capability.** A C-STORE SCP handler that has not persisted a dataset must not return
  success; an unsupported SOP Class is rejected at negotiation or answered "SOP Class not supported", never silently
  accepted (PRD §9.2). An N-service SCP request (MPPS or Storage Commitment as provider) is out of scope and rejected
  at negotiation, not stubbed to success.
- **Truncation is failure.** A Part 10 reader distinguishes a clean dataset-boundary EOF from a short read mid-value
  and propagates `io.ErrUnexpectedEOF`; a truncated file is never accepted as complete (PRD §9.2). Missing file-meta,
  absent `DICM` magic, or a truncated preamble is an error.
- **Bounds-checked length math.** Every value, sequence, PDU, and pixel-frame length is validated against the bytes
  actually remaining in a bounded reader before allocation; attacker-controlled lengths never reach
  `make([]byte, n)` unchecked (PRD §9.3). Undefined length (`0xFFFFFFFF`) is delimiter-terminated, never an allocation
  size.
- **Codec-unavailable is typed, not fatal.** Compressed pixel decode/encode without a built-in codec returns
  `dicom.ErrCodecUnavailable` naming the transfer syntax; the rest of the dataset and the wire transport are
  unaffected (PRD §7.3).
- **No PHI in diagnostics by default.** Errors and logs name the offending element (keyword plus `(gggg,eeee)`), VR,
  SOP Class/transfer-syntax UID by registered name, and DIMSE status by name — identifiers and structure, not patient
  values (PRD §8.2, §9.1). Servers and the reference daemon bind to loopback unless an explicit non-loopback bind is
  configured.

### Fuzzing posture

The untrusted-binary surfaces are exercised by Go native fuzz targets that guard the hostile-input guarantees above:
on a malformed or truncated input the reader must return rather than panic or over-allocate. The targets discard the
returned error and assert only the survival property (no panic, and under the scheduled harness no hang or runaway
allocation); the typed-error contract for each rejection is asserted by the unit tests. The data-layer targets live in
`dicom/fuzz_test.go`:

- `FuzzRead` drives `dicom.Read`, the Part 10 / dataset reader.
- `FuzzReadPixelDataFrom` drives `dicom.ReadPixelDataFrom`, including the encapsulated fragment-stream path.

The DIMSE PDU targets live in `dimse/pdu/`: `FuzzReadPDU` (whole-PDU dispatch), `FuzzDecodeAssociateAC`
(A-ASSOCIATE-AC sub-item length math), and `FuzzDecodePDV` (presentation-data-value framing). Each target ships a
version-controlled seed corpus under its package's `testdata/fuzz/<FuzzName>/` directory: minimal, well-formed objects
with synthetic identifiers alongside hostile regression seeds (truncated, oversized-length), so a normal `go test`
replays both as regression cases. The targets are intended to run in CI with `hang=fail` so a hang or a crash fails the
build; that scheduled fuzz job is wired separately.

## De-identification conformance (PS3.15 Basic Application Level Confidentiality Profile, Table E.1-1)

go-radx implements the PS3.15 Annex E Basic Application Level Confidentiality Profile through `dicom.Profile`
(`NewProfile` / `Deidentify`). The profile applies a fixed action to each confidentiality attribute wherever it appears,
recursing into every sequence item, and writes the PS3.15 de-identification metadata (`PatientIdentityRemoved`
(0012,0062) = `YES`, `DeidentificationMethod` (0012,0063), and `DeidentificationMethodCodeSequence` (0012,0064)). The
covered attribute set and its actions are enumerated below so the conformance claim is checkable, not generic.

The covered set is exactly **213 attributes**, the keyword table in `dicom/deidentify_actions.go`. The
`tools/conformance-drift` gate asserts this documented count matches `len(basicProfileKeywordActions)` in that
source, so the statement cannot drift from the code.

### Action codes

The profile models the Table E.1-1 action vocabulary as `dicom.deidAction`. go-radx applies these codes:

| Code | Action | go-radx behaviour |
|------|--------|-------------------|
| X | Remove | The attribute is deleted at every nesting level. |
| Z | Replace with zero length | The value is replaced with a zero-length value of the same VR. |
| D | Replace with non-zero dummy | The value is replaced with a VR-valid, non-empty placeholder (overridable per tag via `WithDummyValues`). |
| C | Clean | go-radx cannot parse free text for residual identity, so C collapses to Z: identity is removed, benign content is not preserved (documented v1 limit). |
| U | Replace UID | The UID is remapped through a stable per-call map so the reference graph stays consistent within one dataset. |
| K | Keep | The attribute is retained unchanged. |

### Action breakdown

The 213 covered attributes resolve to these actions under the default (strictest) profile:

| Action | Count |
|--------|-------|
| X (remove) | 149 |
| Z (replace with zero length) | 28 |
| U (replace UID) | 18 |
| C (clean → collapses to Z in v1) | 7 |
| K (keep) | 7 |
| D (replace with dummy) | 4 |
| Total | 213 |

### Attributes by action

**D — replace with dummy (4):** `PatientName` (0010,0010), `PatientID` (0010,0020), `VerifyingObserverName`
(0040,A075), `VerifyingObserverSequence` (0040,A073).

**C — clean, collapses to Z in v1 (7):** `StudyDescription` (0008,1030), `SeriesDescription` (0008,103E),
`ProtocolName` (0018,1030), `RequestedProcedureDescription` (0032,1060), `PerformedProcedureStepDescription`
(0040,0254), `DerivationDescription` (0008,2111), `AcquisitionDeviceProcessingDescription` (0018,1400).

**U — replace UID (18):** `InstanceCreatorUID`, `SOPInstanceUID`, `StudyInstanceUID`, `SeriesInstanceUID`,
`FrameOfReferenceUID`, `SynchronizationFrameOfReferenceUID`, `ReferencedFrameOfReferenceUID`, `ConcatenationUID`,
`DimensionOrganizationUID`, `PaletteColorLookupTableUID`, `ReferencedSOPInstanceUID`, `StorageMediaFileSetUID`,
`FiducialUID`, `IrradiationEventUID`, `TargetUID`, `TransactionUID`, `DeviceUID`, `FailedSOPInstanceUIDList`.

**K — keep (7):** `BodyPartExamined` (0018,0015), `ImageType` (0008,0008), `DerivationCodeSequence` (0008,9215),
`ContentSequence` (0040,A730), and the three de-identification-metadata attributes the profile itself writes —
`PatientIdentityRemoved`, `DeidentificationMethod`, `DeidentificationMethodCodeSequence`. Keeping the metadata
attributes prevents the table walk from removing the markers the profile must set.

**Z — replace with zero length (28):** `PatientBirthDate`, `PatientSex`, `PatientSexNeutered`, `AccessionNumber`,
`StudyID`, `StudyDate`, `StudyTime`, `ReferringPhysicianName`, `SeriesDate`, `SeriesTime`, `PerformingPhysicianName`,
`OperatorsName`, `RequestAttributesSequence`, `PerformedProcedureStepStartDate`, `PerformedProcedureStepStartTime`,
`PerformedProcedureStepEndDate`, `PerformedProcedureStepEndTime`, `AcquisitionDate`, `AcquisitionTime`,
`AcquisitionDateTime`, `ContentDate`, `ContentTime`, `InstanceCreationDate`, `InstanceCreationTime`,
`ReferencedImageSequence`, `SourceImageSequence`, `ReferencedPerformedProcedureStepSequence`, `ContentCreatorName`.

**X — remove (149):** the remaining covered attributes, removed at every nesting level. Grouped by module as the
source table organises them:

- Patient identity and demographics: `IssuerOfPatientID`, `TypeOfPatientID`, `PatientBirthTime`,
  `PatientInsurancePlanCodeSequence`, `PatientPrimaryLanguageCodeSequence`, `OtherPatientIDs`,
  `OtherPatientIDsSequence`, `OtherPatientNames`, `PatientBirthName`, `PatientAge`, `PatientSize`, `PatientWeight`,
  `PatientAddress`,
  `InsurancePlanIdentification`, `PatientMotherBirthName`, `MilitaryRank`, `BranchOfService`, `MedicalRecordLocator`,
  `MedicalAlerts`, `Allergies`, `CountryOfResidence`, `RegionOfResidence`, `PatientTelephoneNumbers`, `EthnicGroup`,
  `Occupation`, `SmokingStatus`, `AdditionalPatientHistory`, `PregnancyStatus`, `LastMenstrualDate`,
  `PatientReligiousPreference`, `PatientComments`, `PatientSpeciesDescription`, `PatientBreedDescription`,
  `BreedRegistrationNumber`, `ResponsiblePerson`, `ResponsibleOrganization`, `PatientState`, `CurrentPatientLocation`,
  `PatientInstitutionResidence`, `ConfidentialityConstraintOnPatientDataDescription`.
- Study, visit, and ordering identity: `IssuerOfAccessionNumberSequence`, `StudyIDIssuer`, `ReferringPhysicianAddress`,
  `ReferringPhysicianTelephoneNumbers`, `ReferringPhysicianIdentificationSequence`, `ConsultingPhysicianName`,
  `ConsultingPhysicianIdentificationSequence`, `PhysiciansOfRecord`, `PhysiciansOfRecordIdentificationSequence`,
  `NameOfPhysiciansReadingStudy`, `PhysiciansReadingStudyIdentificationSequence`, `RequestingPhysician`,
  `RequestingService`, `AdmittingDiagnosesDescription`, `AdmittingDiagnosesCodeSequence`, `PatientHospitalDiscussion`,
  `AdmissionID`, `IssuerOfAdmissionID`, `IssuerOfAdmissionIDSequence`, `ServiceEpisodeID`, `ServiceEpisodeDescription`,
  `IssuerOfServiceEpisodeID`, `IssuerOfServiceEpisodeIDSequence`, `ReferencedPatientAliasSequence`, `AdmittingDate`,
  `AdmittingTime`, `DischargeDiagnosisDescription`, `PerformedProcedureStepID`, `CommentsOnThePerformedProcedureStep`,
  `ScheduledProcedureStepStartDate`, `ScheduledProcedureStepStartTime`, `ScheduledProcedureStepEndDate`,
  `ScheduledProcedureStepEndTime`, `ScheduledPerformingPhysicianName`, `ScheduledStationName`,
  `ScheduledStationAETitle`, `ScheduledProcedureStepDescription`, `RequestedProcedureID`,
  `FillerOrderNumberImagingServiceRequest`,
  `PlacerOrderNumberImagingServiceRequest`, `OrderCallbackPhoneNumber`, `OrderEnteredBy`, `OrderEntererLocation`.
- Series and operator identity: `PerformingPhysicianIdentificationSequence`, `OperatorIdentificationSequence`.
- Equipment and institution identity: `InstitutionName`, `InstitutionAddress`, `InstitutionCodeSequence`,
  `InstitutionalDepartmentName`, `StationName`, `DeviceSerialNumber`, `DeviceLabel`, `GantryID`, `PlateID`,
  `DetectorID`, `CassetteID`, `SourceManufacturer`.
- Image, acquisition, and overlay: `OverlayDate`, `OverlayTime`, `DateOfLastCalibration`, `TimeOfLastCalibration`,
  `AcquisitionComments`, `ImageComments`, `FrameComments`, `ImagePresentationComments`.
- SOP common and provenance: `TimezoneOffsetFromUTC`, `DigitalSignaturesSequence`, `DigitalSignatureUID`,
  `MACParametersSequence`, `DataSetTrailingPadding`, `ContributionDescription`, `ModifiedAttributesSequence`,
  `OriginalAttributesSequence`.
- Content, person, and free text: `PersonName`, `PersonAddress`, `PersonTelephoneNumbers`,
  `PersonIdentificationCodeSequence`, `VerifyingObserverIdentificationCodeSequence`, `VerifyingOrganization`,
  `AuthorObserverSequence`, `ParticipantSequence`, `CustodialOrganizationSequence`,
  `ContentCreatorIdentificationCodeSequence`, `NameOfPhysiciansReadingStudyCodeSequence`, `TextComments`, `TextString`,
  `TelephoneNumberTrial`, `DistributionName`, `DistributionAddress`, `NamesOfIntendedRecipientsOfResults`,
  `IntendedRecipientsOfResultsIdentificationSequence`, `ImpressionsTrial`, `ResultsComments`,
  `InterpretationApproverSequence`, `InterpretationAuthor`, `InterpretationDiagnosisDescription`,
  `InterpretationIDIssuer`, `InterpretationRecorder`, `InterpretationTranscriber`, `InterpretationText`, `ReviewerName`,
  `DataSetName`, `ArbitraryText`.
- Specimen and protocol-context identity: `BarcodeValue`, `SpecimenAccessionNumber`, `SpecimenIdentifier`,
  `SlideIdentifier`, `DischargeDate`, `DischargeTime`.

### Supported sub-options

The default profile (zero-value options) is the strictest: nothing retained, UIDs remapped, dates removed or zeroed,
all private tags removed, burned-in pixel data fail-closed. Each sub-option below is an explicit opt-in that weakens
de-identification, and selecting it appends the matching PS3.15 Context Group 7050 code to
`DeidentificationMethodCodeSequence`:

| Sub-option | `dicom.ProfileOption` | CG 7050 code | Effect |
|------------|-----------------------|--------------|--------|
| Retain Patient Characteristics | `WithRetainPatientCharacteristics()` | 113108 | Keeps age, sex, size, weight, smoking and pregnancy status. |
| Retain Longitudinal Temporal Information | `WithRetainLongitudinalTemporalInformation(mode)` | 113106 / 113107 | Keeps dates/times verbatim (`DateModeKeep`) or shifts them by one per-study offset (`DateModeShift`). |
| Retain Device Identity | `WithRetainDeviceIdentity()` | 113109 | Keeps station name, device serial number, institution name and related device/institution attributes. |
| Retain UIDs | `WithRetainUIDs()` | 113110 | Skips UID remapping, leaving Study/Series/SOP and referenced UIDs in place. |
| Retain Safe Private | `WithRetainSafePrivate(creators...)` | 113111 | Keeps private attributes whose private creator is on the supplied allow-list; all other private tags are removed. |

The Basic Profile code `113100` ("Basic Application Confidentiality Profile") is always written; the sub-option codes
are appended only when the matching option is set.

In addition to the PS3.15 sub-options, `WithAudit(f dicom.AuditFunc)` registers the optional data-modification audit
hook (PRD §9.5): each successful `Deidentify` emits one `dicom.AuditEvent` listing the applied changes as
(tag, action) pairs. The event carries tag coordinates and action names only — no attribute values and, unlike the
server-side audit events (which name stored objects by their PHI-adjacent UIDs), no UIDs, original or replacement; a
PHI-sentinel test enforces value absence. The default is no hook, and the disabled cost is a nil comparison; the
consumer owns the sink and retention.

### Documented limits

- **Burned-in pixel data is fail-closed.** If a dataset declares `BurnedInAnnotation` (0028,0301) = `YES`, `Deidentify`
  returns `dicom.ErrBurnedInPixelData` before doing any work and never reports a complete de-identification while
  identifying pixel text remains. The profile never itself removes or cleans burned-in pixel text. A caller that has
  handled the pixels by other means can accept the residual risk with `WithAllowBurnedInPixelData()`.
- **C (clean) is not content-preserving.** Clean attributes collapse to Z (zero length): identity is removed, but the
  benign clinical content a true PS3.15 clean would retain is not preserved. This is a v1 limit, applied in the safe
  direction.
- **Private tags are removed wholesale by default.** go-radx implements no private-SOP-class logic to judge which
  private attributes are safe, so the Basic Profile removes all private tags unless `WithRetainSafePrivate` allow-lists
  specific private creators (PRD §3.2).
- **UID remap is per-call and structural.** Each `Deidentify` call mints replacements through a fresh map keyed by
  source UID, preserving the reference graph within one dataset; UIDs are not correlated across separate calls unless
  the caller reuses identifiers. UID remapping requires a `UIDGenerator`; without one the call fails closed
  (`errNoUIDGenerator`) unless `WithRetainUIDs` is set.
- **Date shifting preserves precision, not absolute dates.** Under `DateModeShift` the per-study offset is derived from
  `StudyInstanceUID`, so intervals within a study are preserved while absolute dates are obscured; time-only (TM) values
  are left unchanged because they have no date to anchor a day-granular shift.
- **Out of Table E.1-1 scope for v1.** Attributes the standard's Table E.1-1 lists but go-radx does not yet model are
  treated as Keep (K) by default, the same as any attribute absent from the table. Curve/overlay comment data beyond the
  overlay date/time attributes above, and option columns other than the five sub-options listed, are not covered in v1.

## Worked examples

### C-ECHO against a reference PACS

```go
package main

import (
    "context"
    "log"

    "github.com/codeninja55/go-radx/dimse"
)

func main() {
    callingAET, _ := dimse.ParseAETitle("RADX-SCU")
    calledAET, _ := dimse.ParseAETitle("ORTHANC")

    ae, err := dimse.NewAE(callingAET)
    if err != nil {
        log.Fatalf("create AE: %v", err)
    }

    ctx := context.Background()
    assoc, err := ae.Associate(ctx, "localhost:4242", calledAET, dimse.VerificationContexts())
    if err != nil {
        // Typed: *dimse.AssociationError (rejection) or *dimse.AbortError (abort), with source/reason by name.
        log.Fatalf("associate: %v", err)
    }
    defer assoc.Release(ctx)

    status, err := assoc.Echo(ctx)
    if err != nil {
        log.Fatalf("c-echo: %v", err)
    }
    if !status.IsSuccess() {
        log.Fatalf("c-echo returned non-success status: %s", status)
    }
    log.Printf("c-echo OK: %s", status)
}
```

### Storing one CT instance

```go
func storeCT(ctx context.Context, ae *dimse.AE, target string, called dimse.AETitle, ds *dicom.DataSet) error {
    // StorageContexts() proposes the validated radiology Storage set over the default transfer syntaxes.
    assoc, err := ae.Associate(ctx, target, called, dimse.StorageContexts())
    if err != nil {
        return err
    }
    defer assoc.Release(ctx)

    status, err := assoc.Store(ctx, ds)
    if err != nil {
        return err // network or encode failure, mapped to the CLI exit-code taxonomy upstream
    }
    if !status.IsSuccess() {
        // Honest failure: a non-success C-STORE status is an error, never reported as a successful store.
        return fmt.Errorf("c-store failed: %s", status)
    }
    return nil
}
```

### Study-level C-FIND with the result iterator

```go
func findStudies(ctx context.Context, assoc *dimse.Association, query *dicom.DataSet) error {
    // Find returns iter.Seq2[Status, *dicom.DataSet]; each Pending status carries one match. Breaking the loop
    // early issues C-CANCEL. QueryLevelStudy is written into (0008,0052) before sending.
    for status, match := range assoc.Find(ctx, query, dimse.QueryLevelStudy) {
        switch {
        case status.IsPending():
            handleMatch(match) // accumulate study-level identifiers
        case status.IsSuccess():
            return nil // terminal success, iteration ends
        case status.IsFailure(), status.IsWarning():
            return fmt.Errorf("c-find terminated: %s", status)
        }
    }
    return nil
}
```

### Reporting a procedure step with MPPS (SCU)

```go
func reportProcedure(ctx context.Context, assoc *dimse.Association, begin, end *dicom.DataSet) error {
    mpps := assoc.MPPS()

    instanceUID, status, err := mpps.Create(ctx, begin) // N-CREATE, status "IN PROGRESS"
    if err != nil {
        return err
    }
    if !status.IsSuccess() {
        return fmt.Errorf("mpps n-create failed: %s", status)
    }

    status, err = mpps.Set(ctx, instanceUID, end) // N-SET, status "COMPLETED"
    if err != nil {
        return err
    }
    if !status.IsSuccess() {
        return fmt.Errorf("mpps n-set failed: %s", status)
    }
    return nil
}
```

## Conformance scope and limits

This statement is the v1 DICOM scope contract. To restate the boundaries precisely:

- **Verified against reference validators.** Conformance to the declared subset is gated in CI by `dciodvfy` (dcmtk)
  on written files, round-trips against vendored `pydicom`/`pynetdicom` corpora, and DIMSE interop against Orthanc and
  dcm4chee-arc (PRD §11.1). A SOP Class in the supported Storage table is exercised against these; a transport-only
  class is not.
- **Supported Storage is the radiology subset, not the 120-Class `pynetdicom` selection.** Non-radiology image classes
  (ophthalmic, dermatologic, RT treatment objects, waveforms, raw data) would be negotiable for transport via the
  transport-only `AllStorageContexts()` forwarding preset, which is **NOT YET SHIPPED** in v1; only the validated
  `StorageContexts()` set is exposed.
- **N-services are SCU-only.** MPPS and Storage Commitment have no SCP side in v1; every other DIMSE-N service is out of
  scope entirely. The N-service SCP roles, Print Management, UPS, and RT Machine Verification are deferred (PRD §3.2,
  §5.1).
- **Query/Retrieve is Patient Root and Study Root only.** The Patient/Study Only model and the composite-instance,
  without-bulkdata, and repository retrieve variants are out of scope.
- **Compressed pixel decode/encode is optional and codec-gated.** Without the CGo build tag, JPEG-family pixel access
  returns `dicom.ErrCodecUnavailable`; transport and dataset parsing of those instances still work. The exact built-in
  codec set is finalised at M8 and reflected here.
- **DICOMDIR file-sets are create-and-read, not update-in-place.** `OpenFileSet` loads and queries an existing
  file-set; `FileSetBuilder` builds and writes a new one. Removing or re-staging instances in an existing file-set is
  not implemented, leaf records are typed IMAGE or SR DOCUMENT only, and PS3.11 media application profiles are not
  enforced on members.
- **Async operations are negotiated, not delivered.** Like `pynetdicom`, go-radx negotiates the async-ops window but
  the acceptor windows to (1,1); concurrency is delivered through Go goroutines and `context.Context`, not the DICOM
  async-ops mechanism.
- **No private SOP-class business logic.** Private tags and creators are parsed generically; no proprietary SOP Class
  semantics are implemented (PRD §3.2).

Changes to any item above are changes to this versioned statement and are reviewed deliberately, never introduced as a
silent capability change (PRD §6.1, risk "conformance-subset creep" §14).

## References

- DICOM PS3.3 (IODs), PS3.4 (Service Class Specifications), PS3.5 (Data Structures and Encoding), PS3.6 (Data
  Dictionary), PS3.7 (Message Exchange), PS3.8 (Network Communication Support), PS3.15 Annex E (de-identification).
- go-radx PRD §6.1 (conformance definition), §6.2 (parity floor), §5.1 (workflow scope), §7.3 (codec strategy),
  §8.1–§8.2 (API and design commitments), §9 (NFRs).
- go-radx reference docs: `docs/reference/dicom.md`, `docs/reference/dimse.md`.
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
- Parity audit: `pynetdicom` SOP Class tables (`sop_class.py`) and presentation-context presets (`presentation.py`),
  enumerated in the Codex review at `parity-pynetdicom.md`.
