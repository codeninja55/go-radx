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
Part 8). DICOMweb (WADO-RS / STOW-RS / QIDO-RS) has its own conformance statement; the two share the
`dicom.TransferSyntax` and SOP Class UID vocabulary but negotiate transport differently.

## Scope summary

In scope for v1:

- The radiology-first **Storage** SOP Class set as both Storage SCU and Storage SCP.
- **Verification** (C-ECHO) as SCU and SCP.
- **Query/Retrieve** (C-FIND, C-GET, C-MOVE) under the Patient Root and Study Root information models, as SCU and SCP.
- **Modality Worklist** (C-FIND) as SCU, with a reference Modality Worklist SCP so the leg is testable end to end.
- **Modality Performed Procedure Step (MPPS)** as SCU only (N-CREATE, N-SET).
- **Storage Commitment Push Model** as SCU only (N-ACTION, N-EVENT-REPORT).
- Uncompressed transfer syntaxes for read and write; compressed transfer syntaxes for transport always, and for pixel
  decode/encode only where a pure-Go or optional-CGo codec is built in.
- Association negotiation: presentation-context negotiation, maximum PDU length, SCP/SCU role selection, asynchronous
  operations window, user identity (types 1–5), SOP Class extended and common extended negotiation, and DIMSE-TLS.

Out of scope for v1 (deferred, designed-for but not implemented — PRD §3.2, §5.1):

- The **SCP/server side** of MPPS and Storage Commitment. v1's N-services are SCU-only.
- All other DIMSE-N services as either role: N-GET, N-DELETE outside the two SCU flows above, Print Management
  (N-CREATE/N-SET/N-GET/N-ACTION/N-DELETE/N-EVENT-REPORT), Unified Procedure Step (UPS), RT Machine Verification, Media
  Creation Management, Display System, Application Event Logging, Instance Availability Notification, Storage
  Management, and Substance Administration.
- Non-radiology Query/Retrieve models: Color Palette, Hanging Protocol, Defined Procedure Protocol, Implant Template,
  Protocol Approval, Relevant Patient Information, Repository Query, and the Composite Instance Root / without-bulkdata
  retrieve variants.
- DICOMDIR file-sets (the `dicom.FileSet` type is named but not implemented in v1).
- Compressed-codec encode where no codec exists; the request returns a typed `dicom.ErrCodecUnavailable`.

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
var DefaultTransferSyntaxes = []dicom.TransferSyntax{
    ExplicitVRLittleEndian, ImplicitVRLittleEndian, DeflatedExplicitVRLittleEndian, ExplicitVRBigEndian,
}

// Presentation-context presets: curated bundles for the roles this statement declares (a go-radx helper concept,
// not a standards term). Each returns proposals keyed by odd presentation-context IDs over DefaultTransferSyntaxes.
func VerificationContexts() []dimse.PresentationContext      // 1 context  — Verification SOP Class
func StorageContexts() []dimse.PresentationContext           // radiology-first Storage set (see table below)
func AllStorageContexts() []dimse.PresentationContext        // every registered Storage SOP Class (transport only)
func QueryRetrieveContexts() []dimse.PresentationContext     // Patient Root + Study Root C-FIND/C-GET/C-MOVE
func BasicWorklistContexts() []dimse.PresentationContext     // Modality Worklist Information Model — FIND
func ModalityPerformedContexts() []dimse.PresentationContext // MPPS SOP Class (MPPS SCU)
func StorageCommitmentContexts() []dimse.PresentationContext // Storage Commitment Push Model SOP Class
```

The verb-level entry points that drive each service are `Association.Echo`, `Association.Store`, `Association.Find`,
`Association.Get`, `Association.Move` (SCU); the `dimse.Handler` interface methods `Echo`/`Store`/`Find`/`Get`/`Move`
(SCP); and `Association.MPPS()` / `Association.StorageCommitment()` for the two N-service SCU flows.

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
**supported, validated** Storage set, and exposes the rest only for negotiation and opaque transport via
`AllStorageContexts()`.

The distinction is deliberate. A SOP Class in the supported set is round-trip-tested (parse, write, `dciodvfy`, and
Orthanc/dcm4chee-arc interop). A SOP Class reachable only through `AllStorageContexts()` is accepted on the wire and
stored or forwarded byte-for-byte, but go-radx makes no IOD-semantic claim about it — it is "transport-only."

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

The three SR SOP Classes are the SR documents the convert layer bridges to FHIR `DiagnosticReport`/`Observation` (PRD
§5.1 step 6); they appear here so the SR side of that conversion has a validated wire path. Secondary Capture is
included so non-radiology-native content (rendered reports, screenshots) has a conformant store target. The supported
set is intentionally narrower than the 120-Class `pynetdicom` selection: it is the radiology workflow's actual SOP
Classes plus the SR and encapsulated-document classes the convert layer needs, not the long tail of ophthalmic,
dermatologic, RT-treatment, and waveform classes that v1 does not validate.

`AllStorageContexts()` proposes every registered Storage SOP Class (the 170-Class transport set) for consumers who
need a forwarding store; instances of unsupported classes are stored and retrieved verbatim with no IOD-level claim.

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

`QueryRetrieveContexts()` returns the `pynetdicom`-equivalent Q/R preset filtered to these six models. The
Query/Retrieve Levels supported are PATIENT, STUDY, SERIES, and IMAGE (`dimse.QueryLevel`); the SCU always writes the
requested level into `(0008,0052)` before sending, and C-FIND/C-GET/C-MOVE expose results as
`iter.Seq2[Status, *dicom.DataSet]` iterators with C-CANCEL on early iterator exit (PRD §8.1, glossary).

### Modality Worklist

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Modality Worklist Information Model — FIND | `1.2.840.10008.5.1.4.31` | Yes | Yes (reference) |

Worklist is a C-FIND with the Modality Worklist information model. v1 ships both the SCU and a reference Modality
Worklist SCP (PRD §5.1 step 2) so the order-to-worklist leg is testable end to end. Exposed by
`BasicWorklistContexts()`, one context, matching the single-Class `pynetdicom` Basic Worklist preset.

### Modality Performed Procedure Step (MPPS) — SCU only

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Modality Performed Procedure Step | `1.2.840.10008.3.1.2.3.3` | Yes (N-CREATE, N-SET) | No (deferred) |

MPPS reports procedure-step start and completion (PRD §5.1 step 3.5). v1 implements the SCU only: `Association.MPPS()`
returns an `*MPPS` with `Create` (N-CREATE, status "IN PROGRESS") and `Set` (N-SET, "COMPLETED"/"DISCONTINUED"). The
SCP side is deferred. The N-services used are N-CREATE and N-SET; no other MPPS N-services are in scope. Exposed for
negotiation by `ModalityPerformedContexts()`.

### Storage Commitment — SCU only

| SOP Class | UID | SCU | SCP |
|-----------|-----|-----|-----|
| Storage Commitment Push Model | `1.2.840.10008.1.20.1` | Yes (N-ACTION, N-EVENT-REPORT) | No (deferred) |

Storage Commitment confirms that an archive has taken custody of stored instances (PRD §5.1 step 4.5). v1 implements
the SCU only: `Association.StorageCommitment().Request(...)` issues the N-ACTION request and receives the asynchronous
N-EVENT-REPORT result as a `StorageCommitmentResult`. The SCP side is deferred. Exposed for negotiation by
`StorageCommitmentContexts()`.

### Presentation-context preset summary

| Preset (`dimse` helper) | Role | Context count |
|-------------------------|------|---------------|
| `VerificationContexts()` | Verification SCU/SCP | 1 |
| `StorageContexts()` | Storage SCU/SCP (validated radiology set) | 36 (table above) |
| `AllStorageContexts()` | Storage transport-only | 170 (registered Storage set) |
| `QueryRetrieveContexts()` | Patient Root + Study Root Q/R | 6 |
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
forwarded), but pixel-frame decode and encode are available only when an optional CGo codec is built in. With CGo
disabled, requesting frames from one of these instances returns the typed `dicom.ErrCodecUnavailable` naming the
transfer syntax — a clear failure, never a build break or a silent partial image (PRD §7.3).

| Transfer syntax | UID | Negotiate / transport | Pixel decode | Pixel encode |
|-----------------|-----|-----------------------|--------------|--------------|
| JPEG Baseline (8-bit) | `1.2.840.10008.1.2.4.50` | Always | CGo only | CGo only |
| JPEG Extended (12-bit) | `1.2.840.10008.1.2.4.51` | Always | CGo only | CGo only |
| JPEG Lossless, Non-Hierarchical | `1.2.840.10008.1.2.4.57` | Always | CGo only | CGo only |
| JPEG Lossless SV1 | `1.2.840.10008.1.2.4.70` | Always | CGo only | CGo only |
| JPEG-LS Lossless | `1.2.840.10008.1.2.4.80` | Always | CGo only | CGo only |
| JPEG-LS Near-Lossless | `1.2.840.10008.1.2.4.81` | Always | CGo only | CGo only |
| JPEG 2000 Lossless | `1.2.840.10008.1.2.4.90` | Always | CGo only | CGo only |
| JPEG 2000 | `1.2.840.10008.1.2.4.91` | Always | CGo only | CGo only |
| HTJ2K Lossless | `1.2.840.10008.1.2.4.201` | Always | CGo only | CGo only |
| HTJ2K | `1.2.840.10008.1.2.4.203` | Always | CGo only | CGo only |

The codec set built behind the CGo build tag is finalised in the M8 hardening milestone; this statement is updated when
a codec moves from "negotiate/transport only" to "decode" or "encode" available. Codecs that `pynetdicom` registers but
go-radx does not target for v1 pixel handling (JPIP, MPEG-2/4, HEVC/H.265, JPEG XL, SMPTE ST 2110, multi-component JPEG
2000) are negotiable for transport but have no go-radx pixel codec.

Selecting a codec is explicit: go-radx never transcodes pixel data unless asked (PRD §8.2 opinionated default). Reading
a file preserves its transfer syntax; conversion is a deliberate `Transcode`-style call.

## Association negotiation

go-radx negotiates associations through `dimse.AE`, `AE.Associate(...)`, and `dimse.NewServer(...)`. The negotiation
features below match the `pynetdicom` floor (PRD §6.2), with the documented async-ops caveat.

| Feature | Support | Notes |
|---------|---------|-------|
| Presentation-context negotiation | Yes | Odd-ID-keyed; per-context accepted transfer syntax; rejection reason codes |
| Maximum PDU length | Yes | `WithMaxPDULength(n)`; default 16382; 0 means "no maximum specified" |
| SCP/SCU role selection | Yes | `WithRoleSelection(...)`; required for C-GET (requestor accepts the Storage SCP role) |
| Asynchronous operations window | Yes (negotiated) | `WithAsyncOps(...)`; acceptor windows to (1,1), synchronous |
| User identity negotiation | Yes (types 1–5) | `WithUserIdentity(...)`: username, passcode, Kerberos, SAML, JWT |
| SOP Class extended negotiation | Yes | `WithExtendedNegotiation(...)`: per-SOP-class service-class application info |
| SOP Class common extended negotiation | Yes | `WithCommonExtendedNegotiation(...)` |
| DIMSE-TLS | Yes | TLS 1.2+ (prefer 1.3), peer verification on by default; mutual-TLS optional (PRD §9.7) |

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

## Behaviour and error model

DICOM behaviour in go-radx follows the PRD's honest-failure and hostile-input rules; the conformance-relevant
guarantees are:

- **DIMSE status is typed.** `dimse.Status` wraps the 16-bit code with `IsSuccess`/`IsPending`/`IsWarning`/`IsFailure`
  and renders by name and class, never as bare hex (PRD §8.2). Pending statuses drive the query/retrieve iterators; a
  terminal failure or warning ends them with the carried status.
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
  (ophthalmic, dermatologic, RT treatment objects, waveforms, raw data) are negotiable for transport via
  `AllStorageContexts()` but carry no IOD-semantic guarantee in v1.
- **N-services are SCU-only.** MPPS and Storage Commitment have no SCP side in v1; every other DIMSE-N service is out of
  scope entirely. The N-service SCP roles, Print Management, UPS, and RT Machine Verification are deferred (PRD §3.2,
  §5.1).
- **Query/Retrieve is Patient Root and Study Root only.** The Patient/Study Only model and the composite-instance,
  without-bulkdata, and repository retrieve variants are out of scope.
- **Compressed pixel decode/encode is optional and codec-gated.** Without the CGo build tag, JPEG-family pixel access
  returns `dicom.ErrCodecUnavailable`; transport and dataset parsing of those instances still work. The exact built-in
  codec set is finalised at M8 and reflected here.
- **DICOMDIR file-sets are not implemented in v1.** The `dicom.FileSet` type is named in the glossary but deferred.
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
