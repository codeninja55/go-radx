# go-radx — product requirements document

## Document control

| Field | Value |
|-------|-------|
| Status | Draft for review (revised after five-dimension adversarial review) |
| Owner | Andru (Architect) |
| Authoring date | 2026-05-31 |
| Module | `github.com/codeninja55/go-radx` (multi-module; see §7.4) |
| Toolchain | Go 1.26.3 (latest 1.26). No v1 deliverable depends on Go 1.27 (see §10.1) |
| Supersedes | The prototype at `vcs/go-radx` and its four archived OpenSpec proposals (`vcs/_archive/go-radx/2026-05-31/`) |

This PRD defines *what* go-radx is and *how we will know it is correct*. Detailed public API contracts live in the
reference documentation (`docs/reference/`); the implementation sequence lives in per-subsystem plans (`docs/specs/`,
`docs/plans/`). This document is the umbrella those artifacts conform to. Section 8.1 commits the load-bearing API
shapes so the developer-experience thesis is reviewable here, not deferred entirely to the reference docs.

## 1. Executive summary

go-radx is a type-safe, idiomatic Go library — with a flagship CLI and embeddable servers — for the three
interoperability standards used in radiology: DICOM (NEMA PS3), HL7 v2.x, and HL7 FHIR (R4 4.0.1 and R5 5.0.0, with R6
designed-for but deferred). It is a ground-up re-foundation of an existing prototype, retaining the same module path and
selectively porting the parts that are sound.

The project exists because the established open-source toolkits force a hard trade. The Python references (`pydicom`,
`pynetdicom`, `python-hl7`, `fhir.resources`) are feature-rich but dynamically typed and easy to misuse; the Java
references (`dcm4che`, HAPI FHIR) are production-grade but heavyweight. go-radx targets the gap: **the feature surface of
the Python libraries, the production discipline of the Java libraries, and the compile-time safety and developer
ergonomics of modern Go.**

The first release (v1) delivers the radiology workflow end-to-end — order, worklist, acquisition, storage, retrieval,
procedure-step status, storage confirmation, reporting, and results. It is built **walking-skeleton-first**: a thin but
correct path touching every subsystem is delivered before any leg is deepened (§5.2, §13). Correctness is gated in CI by
live interoperability against reference implementations (Orthanc, dcm4chee-arc, the HL7 FHIR validator, dcmtk's
`dciodvfy`), plus basic safety checks and comparative performance benchmarks (§11) — not by self-asserted unit tests
alone.

## 2. Problem statement and motivation

### 2.1 The pain we are solving

Andru's frustrations with the Python references, confirmed by direct source analysis, are:

1. **No type safety.** `pydicom` resolves element keywords dynamically; `pynetdicom` passes datasets and string UIDs
   unchecked; `python-hl7` is entirely stringly-typed and positional (with a 0-based container under a 1-based accessor —
   a real footgun); `fhir.resources` stores required value-set bindings as *metadata only* and never enforces them.
   Misuse surfaces at runtime, against patient data.
2. **Poor developer ergonomics.** Positional accessors, reflection-based attribute access, global mutable configuration,
   and warning-versus-exception ambiguity make the libraries hard to use correctly.

### 2.2 Why a re-foundation, not a patch

An independent, adversarial Codex (`gpt-5.5`) audit of the existing `vcs/go-radx` code found it **prototype-level, not
production-ready, across all four subsystems**, with foundational defects:

- **DICOM core** drops `SQ` sequences (replacing them with empty bytes), never writes the file-meta group length,
  mis-declares transfer syntax on write, accepts truncated files as complete, and feeds attacker-controlled lengths
  straight into `make([]byte, n)`.
- **DIMSE** never sets the last-fragment bit on the final command PDV (the concrete root cause of the Orthanc aborts),
  always decodes datasets as Implicit VR LE regardless of negotiated transfer syntax, and ships a DUL state machine
  missing the PS3.8 release-collision states (Sta9–Sta12).
- **FHIR** ships an `UnmarshalResource[T]` that is not type-safe (never checks `resourceType`), a generator that emits
  every choice branch as required and produces empty backbone structs, required-field validation that treats a valid
  `false` as missing, and maps `decimal` to `float64`.
- **CLI** has a `modify` that is a **no-op reporting success**, a `store` that exits zero on failed transfers, banners
  that corrupt machine-readable stdout, PHI written to logs, and a PHI-bearing SQLite catalogue created by default on an
  all-interfaces bind.

The CHANGELOG claimed much of this "complete"; the archived roadmap honestly marked it "in progress." This PRD states
current status from evidence. A re-foundation that ports the salvageable shells and rewrites the broken cores is lower
risk than a literal rewrite (discards what works) or an incremental patch (builds on broken foundations).

## 3. Goals and non-goals

### 3.1 Goals

- Type-safe, idiomatic Go APIs for DICOM, HL7 v2.x, and FHIR R4/R5 that make invalid states unrepresentable at the
  serialization boundary where Go's type system allows, and return errors as values everywhere else.
- Legible, opinionated APIs and human-readable DICOM/HL7/FHIR diagnostics: signatures and error messages a developer can
  read without the spec, with strong safe defaults over endless configuration (§8.2).
- A `radx` CLI with dcmtk-class breadth and ergonomics, serving practitioners and operators.
- Embeddable, composable servers (DIMSE SCP, DICOMweb, FHIR REST, HL7 v2 MLLP) with pluggable storage and auth, plus
  thin reference daemons that run out of the box.
- Production-grade non-functional behaviour: PHI safety, honest failure reporting, hostile-input hardening, concurrency
  safety, transport security, supply-chain integrity, observability, and no global mutable state.
- Conformance verified against reference implementations and official validators in CI; basic safety defaults checked in
  CI; and performance benchmarked against the reference libraries (§11).

### 3.2 Non-goals (v1)

- **Not** a full PACS/archive product competing with dcm4chee-arc or Orthanc. go-radx ships embeddable pieces and thin
  reference daemons; a full archive is a separate product built *on top* of the library.
- **Not** an image viewer or rendering/display pipeline beyond pixel decode/encode.
- **No** AI/ML inference or clinical decision support.
- **No** proprietary/private SOP-class business logic (private tags are parsed generically).
- **Deferred** past v1 (architected-for, not implemented): full SMART on FHIR, US Core IG profile validation, FHIR
  Subscriptions, FHIR R6, FHIR R4B (4.3.0) and STU3, the **SCP/server sides** of MPPS, Storage Commitment, Print
  Management and UPS (the v1 N-services are SCU-only — see §5.1), UPS-RS, and the legacy WADO-URI interface.

## 4. Target users and primary audience

Target users: healthcare software developers, radiology workflow automation engineers, medical imaging application
developers, PACS/RIS integrators, and interoperability specialists.

**Primary audience (the tie-breaker for design conflicts): library consumers.** When a clean Go API conflicts with CLI
simplicity or server flexibility, the library API wins by default. The `radx` CLI is the flagship first-party consumer
that proves the API; the servers are library features. Consequence enforced in §7.4: library consumers must never be
forced to pull the CLI/TUI dependency graph.

## 5. Product scope

### 5.1 v1 — the radiology workflow vertical, with the loop closed

v1 delivers every step of the radiology workflow end-to-end, including the status-feedback and storage-confirmation legs
that make "complete" honest:

| # | Workflow step | go-radx surface in v1 |
|---|---------------|------------------------|
| 1 | Order placed | HL7 v2 `ORM`/`OMG` parse + build; FHIR `ServiceRequest` |
| 2 | Worklist query | DIMSE `C-FIND` Modality Worklist — **both SCU and a reference MWL SCP** so the leg is testable end-to-end |
| 3 | Acquisition | DICOM data layer: Part 10, `SQ`, transfer syntaxes, value types, UID generation |
| 3.5 | Procedure-step status | **MPPS SCU** (`N-CREATE`/`N-SET`) — reports procedure start/completion |
| 4 | Storage | DIMSE `C-STORE` (SCU + SCP); DICOMweb `STOW-RS` (client + server) |
| 4.5 | Storage confirmation | **Storage Commitment SCU** (`N-ACTION` + `N-EVENT-REPORT`) |
| 5 | Viewing | DIMSE `C-GET`/`C-MOVE`/`C-ECHO`; DICOMweb `WADO-RS` + `QIDO-RS` (client + server) |
| 6 | Report | FHIR `DiagnosticReport` + `ImagingStudy`; DICOM→FHIR ImagingStudy (R4+R5); DICOM SR ↔ FHIR DiagnosticReport/Observation (bidirectional) |
| 7 | Results | HL7 v2 `ORU` over MLLP (client + server, ACK/NACK); ORU↔FHIR; `ADT`→FHIR Patient/Encounter |

The v1 DIMSE-N surface is **SCU-only** for MPPS and Storage Commitment; all other N-services and the SCP/server sides of
these two are deferred (§3.2).

### 5.2 Sequencing — walking skeleton, then deepen

The dependency graph forces the DICOM data layer first (DIMSE, DICOMweb, and DICOM→FHIR all read datasets). v1 is then
assembled as a *thin, correct* end-to-end slice touching every subsystem (the walking skeleton, milestone M2), and each
leg is deepened afterward. The term "walking skeleton" in this document means **only** that first cross-subsystem slice;
per-subsystem depth milestones are named explicitly in §13 and never reuse the term.

### 5.3 FHIR release scope (closed)

Generate from the official HL7 StructureDefinitions for **R4 (4.0.1)** and **R5 (5.0.0)** — and *only* those for v1.
STU3 is out of scope; R4B (4.3.0) is deferred (open only as a future addition). The generator is architected so R6 can
be added when normative; R6 is not implemented in v1. All resources for R4 and R5 are generated and compile-tested; the
radiology + clinical workflow set is conformance-tested in v1. (`fhir.resources` serves R4-era content only via its R4B
sub-package — from v7.0.0 there is no separate R4 package, since it treats R4B as overlapping R4. go-radx is not bound by
that and generates R4 4.0.1 directly because it remains the most deployed release; US Core runs on R4.)

## 6. Conformance model

### 6.1 Definition

"Conformant" means **100% conformant to an explicitly declared, versioned subset, verified against reference
validators** — not "implements all of every standard." Each standard gets a published Conformance Statement
(`docs/conformance/`) enumerating exactly which SOP Classes, transfer syntaxes, resources/profiles, and message types are
supported. The Conformance Statement is the single source of truth for scope; growth is a deliberate, reviewed change.
The subset is seeded from the Python parity floor (§6.2), extended radiology-first, with dc4che and HAPI FHIR as
production-design references.

### 6.2 Parity floor (evidence-based, from source analysis of the reference libraries)

**DICOM data model (`pydicom` floor).** 34 standard VRs + 4 ambiguous (per DICOM PS3.5 Table 6.2-1, including the 64-bit
OD/OL/OV/SV/UV and the UC/UR string VRs); uncompressed transfer syntaxes read *and* write (Implicit VR LE, Explicit VR
LE, Explicit VR BE retired, Deflated); Part 10 with preamble, file-meta (Explicit VR LE) and auto-recomputed group
length; nested `SQ` with defined and undefined lengths; private creators and private-block access; repeating groups
(`60xx`/`50xx`); deferred/partial reads; standard data dictionary (~5,189 entries), private dictionary (~382 creators /
~10,996 element entries), UID dictionary (~490); Specific Character Set including ISO 2022 code extensions plus
UTF-8/GB18030/GBK; `PersonName` with three component groups; UID generation; native + RLE + encapsulated pixel data with
a pluggable codec architecture; DICOM-JSON; DICOMDIR file-sets.

**DIMSE networking (`pynetdicom` floor).** DIMSE-C (C-ECHO/C-STORE/C-FIND/C-GET/C-MOVE) as SCU and SCP, with C-CANCEL and
the streaming multi-response contract for query/retrieve; DIMSE-N (N-CREATE/SET/GET/DELETE/ACTION/EVENT-REPORT) — full
suite in the floor, with v1 implementing MPPS-SCU and Storage-Commitment-SCU and deferring the rest (§5.1, §3.2);
presentation-context presets (Storage 120 selected / 170 all, Query-Retrieve 13, Verification 1, Print Management 11,
Basic Worklist, and ~20 more); transfer-syntax negotiation (4 default; the full registered set is 45 in pynetdicom / 43
in pydicom); association negotiation (max PDU, SCP/SCU role selection, async-ops window, user identity types 1–5, SOP
class extended + common negotiation); a faithful PS3.8 Table 9-10 DUL state machine (13 states, 19 events, 28 actions,
including the Sta9–Sta12 release-collision states the prototype omitted); an event-handler model with integer-or-dataset
status; TLS; and the reference CLI apps (echoscu/scp, storescu/scp, findscu, getscu, movescu, qrscp).

**HL7 v2 (`python-hl7` floor).** Six-level parse tree (message/segment/field/repetition/component/subcomponent);
**Batch (BHS/BTS) and File (FHS/FTS) containers with batch/file parsing**; delimiter and encoding-character derivation
from MSH-1/MSH-2 with defaults; both prefixed and numeric accessor keys with segment-instance indexing; message
construction; `create_ack`; escape/unescape per the HL7 v2 base standard Chapter 2 §2.10 (separator, hex, highlight,
rich-text, and application-defined sequences); DTM parsing; MLLP framing with blocking client and async client/server.

**FHIR (`fhir.resources` floor).** Typed models for every resource and backbone element of R4 and R5; choice types as N
separate scalar fields (max cardinality 1 each) with a mutual-exclusivity + required validator; primitive types with the
`_field` extension sibling round-tripped in JSON; cardinality/required validation; `resourceType` integrity on
polymorphic fields; canonical element ordering; `contained`/`Bundle.entry.resource` polymorphic dispatch to concrete
types; **OperationOutcome modelling and transaction/searchset/document/message Bundle semantics**; **intra-Bundle and
contained reference-integrity resolution**; **`_summary` mode (full/true/false/text/data) for bandwidth-constrained
serialization** (a working prototype feature, carried forward); JSON serialization (XML/YAML optional, deferrable).

### 6.3 Where go-radx exceeds the references (the type-safety thesis)

First-class v1 requirements — the reason the project exists. The concrete API shapes that realise these are committed in
§8.1; the claims below are scoped honestly:

- **Generated Go enums for `required`-strength value-set bindings** — a defined string type + const set + a validating
  `UnmarshalJSON`/`Parse` with an explicit unknown-code policy per binding strength. This gives binding safety at the
  JSON boundary (not at every literal assignment, which Go cannot enforce), strictly better than `fhir.resources`'
  metadata-only approach.
- **A lexical-preserving `Decimal` type** for FHIR `decimal` and DICOM DS/IS — unifies what `pydicom` already does per
  type (`original_string`) across both standards, and beats `fhir.resources` (lossy on serialize) and the prototype
  (`float64`). API committed in §8.1.
- **A genuinely type-safe resource API** — an internal `resourceType→factory` registry returning the `fhir.Resource`
  interface, plus package-level generic functions `Unmarshal[T]`/`As[T]` that verify the embedded `resourceType` matches
  `T`. This replaces the prototype's unchecked `UnmarshalResource[T]`; the win is the *check*, not a "typed registry"
  (which Go cannot provide — §8.1).
- **Typed choice-type accessors** — generated `Value()`/`SetValueX()` per choice group that enforce mutual exclusion at
  the API boundary, so the design is an ergonomic win over `fhir.resources`, not merely a more verbose clone (§8.1).
- **Typed HL7 v2 segments and message types** (MSH/PID/PV1/EVN/ORC/OBR/OBX/MSA/ERR/…, and ADT/ORM/ORU/ACK) with named
  Go fields as the primary API, enums for common HL7 tables, and one unambiguous indexing convention (§8.1).
- **A PS3.15 Basic Application Level Confidentiality Profile** (PS3.15 Annex E / §E.1) for de-identification: Table
  E.1-1 actions applied recursively through sequence items, consistent UID remapping with a stable map, and
  de-identification-method metadata. Its scope and limits (notably whether burned-in pixel/overlay cleaning is
  performed) are clearly documented so the consumer can judge whether it meets their de-identification needs — go-radx
  provides the capability, not a compliance guarantee. Neither `pydicom` nor `pynetdicom` ships a built-in profile.
- **Typed DIMSE status**, `context.Context`-based cancellation/timeouts, goroutine-and-channel concurrency (real async
  ops, which pynetdicom negotiates but does not provide), and per-instance configuration instead of global mutable
  state.
- **Human-readable diagnostics** — logs and errors that name tags, VRs, UIDs, and DIMSE statuses instead of emitting raw
  hex or numeric codes, with actionable context and no PHI by default (§8.2). The reference libraries' diagnostics are
  raw, and `pynetdicom`'s are PHI-leaking by default.

## 7. Architecture overview

### 7.1 Package layout

```
go-radx/
├── dicom/        # DICOM data model: Part 10 I/O, datasets, VR/value types, SQ, tags, UID, datetime, pixel, anonymize
├── dimse/        # DIMSE networking: DUL FSM, PDU, ACSE association, SCU, SCP, C/N services
├── dicomweb/     # DICOMweb: WADO-RS, STOW-RS, QIDO-RS — client and server (greenfield; measured vs dc4che + the standard)
├── hl7v2/        # HL7 v2.x: typed segments/messages, generic tree, batch/file, MLLP client/server (greenfield)
├── fhir/         # FHIR R4/R5: root (Resource, Unmarshal/As, Decimal) + generated r4/ and r5/ release sub-packages
├── convert/      # Cross-standard: DICOM→FHIR ImagingStudy, SR↔DiagnosticReport, HL7v2↔FHIR
├── server/       # Embeddable server building blocks + pluggable storage/auth interfaces
└── cmd/radx/     # The flagship CLI and thin reference daemons (separate Go module — §7.4)
```

### 7.2 Servers — embeddable components plus thin daemons

Each server role (DIMSE SCP including a Modality Worklist SCP, DICOMweb, FHIR REST, MLLP) is a composable package
exposing handler interfaces with **pluggable storage, auth, and persistence**. Thin reference daemons wire sane defaults
(filesystem object store + SQLite catalogue) so the CLI and users get something runnable immediately. "Production-ready"
means the primitives are concurrency-safe, observable (OTel + structured logging), shut down gracefully, and the
reference daemons are deployable and loopback-bound by default (§9.1) — it does *not* mean a full PACS.

### 7.3 Codec strategy (explicit decision)

Compressed transfer-syntax codecs (JPEG, JPEG-LS, JPEG 2000, HTJ2K) are the source of the prototype's build fragility.
Strategy: **pure-Go decoders where they exist; optional CGo behind a build tag for the rest, never load-bearing.** The
core library builds and passes its non-pixel tests with CGo disabled; compressed pixel support degrades to a clear,
typed "codec unavailable" error rather than a build failure.

### 7.4 Module structure

`cmd/radx` is a **separate Go module** (`github.com/codeninja55/go-radx/cmd/radx`) so library consumers importing the
core packages do not inherit the CLI's Kong/TUI dependency graph. This directly serves the §4 primary-audience
tie-breaker. The root module depends only on what the library needs.

## 8. Functional requirements

Detailed, testable functional requirements live per subsystem in the reference documentation. At the PRD level, each
subsystem must meet its §6.2 parity floor plus the §6.3 exceed-parity requirements and the §8.1 API commitments, and
satisfy the §9 NFRs. The v1 cross-standard conversions are: DICOM→FHIR `ImagingStudy` (R4+R5), DICOM SR ↔ FHIR
`DiagnosticReport`+`Observation` (bidirectional), HL7 v2 `ORU` ↔ FHIR `DiagnosticReport`/`Observation`, and HL7 v2 `ADT`
→ FHIR `Patient`/`Encounter`.

The `radx` CLI carries forward the preserved command set and an explicit exit-code taxonomy as the v1 floor: commands
`echo`, `store`, `scp`, `dump`, `modify`, `organize`, `lookup`, `catalogue` (plus the new `hl7`, `dicomweb`, and
`convert` groups); output formats human/JSON/CSV with machine output uncontaminated by banners (banners and diagnostics
to stderr); and exit codes `0` success, `1` general, `2` usage, `3` DICOM/parse, `4` network, `5` file I/O. The §9.2
failure-reporting rules govern every command.

### 8.1 API shape commitments (normative)

These illustrative-but-binding signatures are the contract the reference docs must conform to. They encode the
type-safety thesis and resolve the Go-language constraints (notably: **no generic methods in v1** — interface and struct
methods cannot declare type parameters, so all type-parameterised dispatch is via package-level generic functions).

```go
// FHIR resource identity + type-safe access. Registry returns the interface; generics give the static type.
type Resource interface{ ResourceType() string }
func Unmarshal[T Resource](data []byte) (T, error) // verifies embedded resourceType matches T, else error
func As[T Resource](r Resource) (T, bool)           // checked downcast; call site: fhir.As[*r5.Patient](entry.Resource)

// Choice types: typed accessor + setters; mutual exclusion enforced at the boundary.
type ObservationValue interface{ isObservationValue() } // sealed-ish; implemented by Quantity, CodeableConcept, ...
func (o *Observation) Value() (ObservationValue, bool)
func (o *Observation) SetValueQuantity(q Quantity)       // clears the other value[x] siblings

// Required-binding enums: defined type + const set + validating parse with explicit unknown policy.
type AdministrativeGender string
const (GenderMale AdministrativeGender = "male"; GenderFemale = "female"; GenderOther = "other"; GenderUnknown = "unknown")
func ParseAdministrativeGender(s string) (AdministrativeGender, error)

// Lexical-preserving decimal (FHIR decimal + DICOM DS/IS). Carries source string; explicit conversion; no in-place math.
type Decimal struct{ /* preserves source lexical form */ }
func (d Decimal) String() string
func (d Decimal) Float64() (float64, bool)
func (d Decimal) MarshalJSON() ([]byte, error) // emits the preserved lexical form

// HL7 v2: typed segments with named Go fields are the PRIMARY API (callers never index).
// The generic tree uses Go-native 0-based slices; any string-key accessor mirrors the HL7 1-based spec convention.
type PID struct{ PatientName XPN; PatientID CX /* ... */ }
func (m *Message) PID() (PID, bool)
func (m *Message) Segment(id string) (Segment, bool)

// DIMSE typed status; query/retrieve return Go 1.23+ iterators, not callbacks.
type Status struct{ Code uint16 }
func (s Status) IsSuccess() bool; func (s Status) IsPending() bool; func (s Status) IsWarning() bool; func (s Status) IsFailure() bool
func (a *Association) Find(ctx context.Context, q *dicom.DataSet, lvl QueryLevel) iter.Seq2[Status, *dicom.DataSet]

// No global config: decoders/encoders take functional options; zero values are sane defaults.
func NewReader(r io.Reader, opts ...ReadOption) *Reader
```

### 8.2 Design principles — legibility, simplicity, and opinionated defaults

Three rules bind both the public API surface and the implementation, in priority order:

1. **Legible call signatures.** A signature should be understandable at a glance without the docs: named types instead of
   bare primitives (`TransferSyntax`, `AETitle`, `Tag` — not `string`/`uint32`); options or config structs (or functional
   options) instead of long positional parameter lists; verbs that read as English (`Store`, `Query`, `Transcode`); and
   errors returned as values with messages a caller can act on. If a signature needs a comment to be understood, the
   signature is wrong.
2. **Legible DICOM diagnostics.** Logs and errors render DICOM (and HL7/FHIR) concepts in human terms, never bare numbers:
   a tag as its keyword plus `(gggg,eeee)`, VRs by name, UIDs and SOP/transfer-syntax UIDs by their registered names, and
   DIMSE statuses by name and class (success/pending/warning/failure). Every error carries the context needed to act —
   which element, which association or SOP instance, which file or byte offset — while honouring the no-PHI-by-default
   rule (§9.1): identifiers and structure, not patient values. Diagnostics are structured (zap) and human-readable at
   once.
3. **Elegant, simple to implement, opinionated where it matters.** Prefer the smallest correct design and one obvious way
   to do each thing over breadth of configuration. Be **opinionated** where it improves safety or clarity — strong sane
   defaults (no transcoding unless asked, loopback bind, errors not panics, typed enums over strings) and a single
   canonical API rather than several overlapping ones — and add a configuration knob only when a real use case proves it
   necessary. Opinionation serves the consumer; it is not rigidity for its own sake.

## 9. Non-functional requirements

These are the library's safe defaults and engineering standards, informed by the failures the prototype and the
references exhibited. The library ships sane defaults and the controls; the **policy** — how much PHI protection, what
regulatory posture, what retention — belongs to the consumer who integrates go-radx into their product. A few of these
are verified by basic CI sanity checks (§11.2); the rest are consumer-tunable, not mandated.

1. **PHI safety (safe defaults; governance is the consumer's).** The library does not log PHI by default — including in
   error messages, query/SQL text, and file paths — and does not emit PHI in telemetry (span attributes, metric labels)
   by default; verbosity that would surface PHI is opt-in. Servers and daemons bind to loopback by default; a
   non-loopback bind is explicit. Any PHI-bearing convenience store (e.g. the reference catalogue) is opt-in and offers a
   redacted mode. Beyond these defaults, **PHI governance — encryption at rest, retention/erasure, access control, and
   audit — is the integrating consumer's responsibility; go-radx provides the hooks and configuration, not the policy,
   and makes no compliance guarantee.**
2. **Honest failure reporting.** Never report success on failed work. Typed errors mapped to the §8 exit-code taxonomy;
   partial batch failure is non-zero by default. Two explicit rules the prototype broke: **(a) fail-closed on
   unimplemented/partial capability** — a path that cannot perform the requested mutation returns a typed error and
   writes nothing; a stub errors, never no-ops-and-reports-success; **(b) truncation and incompleteness are failures** —
   parsers distinguish a clean record-boundary EOF from a short read mid-value (propagate `io.ErrUnexpectedEOF`);
   accepting a truncated object as complete is a defect. Each rule carries a regression test.
3. **Hostile-input hardening.** All external input (files, network peers, HTTP bodies, MLLP frames) is untrusted.
   Checked integer conversions for all length/dimension math (reject negative, overflow, and underflow *before*
   allocation; require `length >= header-size` before subtraction); per-element, per-sequence, per-PDU, and
   per-pixel-frame caps validated against bytes actually remaining in a bounded reader; configurable caps on every
   network entry point (max request/body size, max multipart parts, max JSON/Bundle nesting depth and resource count,
   max MLLP frame length) with typed "limit exceeded" errors; no panics on malformed input. For CGo codecs: every C
   allocation result checked, dimensions capped before allocation, sizes converted through checked `size_t`/`int`, and
   ASAN/UBSAN builds in CI. **Fuzz/crash tests fail CI on hang (subprocess timeout = failure) and may not be skipped.**
4. **Concurrency safety.** No global mutable state. Concurrent-safe APIs or explicitly documented non-safe types.
   `context.Context`-driven cancellation/timeouts on all network and long-running operations. No fire-and-forget
   goroutines.
5. **Engineering quality (no imposed regulatory posture).** go-radx is a general-purpose library built to good
   engineering standards: deterministic and testable behaviour, explicit error handling, and validation against the
   published Conformance Statements. It makes **no medical-device certification claim and imposes no compliance
   machinery.** Whether it is used in regulated or SaMD contexts — and any resulting obligations (IEC 62304, ISO 13485,
   formal audit trails, de-identification governance) — is the consumer's decision and responsibility. For consumers who
   want an audit trail, the library exposes an optional data-modification hook they can wire to their own sink; it does
   not mandate a schema or a sink.
6. **Performance.** Minimise allocations in hot paths; stream large objects (pixel data, bulk transfers); benchmark
   performance-critical paths, including comparative benchmarks against the reference libraries for compression and
   transfer-syntax handling (§11.3).
7. **Transport security.** DIMSE-TLS and all HTTP servers default to TLS 1.2+ (prefer 1.3), verify peer certificates by
   default, never set `InsecureSkipVerify` outside an explicitly flagged test mode, and offer a documented mutual-TLS
   option.
8. **Secrets handling.** Credentials, keys, and tokens come from env/files per 12-factor, are never logged, and are
   never written to the PHI catalogue.
9. **Supply-chain integrity.** Pinned dependencies and `govulncheck` in CI. SBOM and signed releases are recommended for
   downstream consumers who need them, not mandated by the library.
10. **Observability.** Structured logging via `zap` — human-readable and machine-structured at once, rendering
    DICOM/HL7/FHIR concepts by name per §8.2 — and OpenTelemetry tracing/metrics, both subject to the §9.1 no-PHI rule. OTel is operator-controlled, opt-in, exports nowhere by default, and carries no PHI; this reconciles the
    archived `project.md` "no telemetry/data collection" privacy stance (which meant outbound vendor collection) with
    local operator observability. Neither `zap` nor a direct OTel dependency is wired today (OTel is only transitive);
    both are added as direct, wired dependencies in M0.
11. **Compatibility.** Semantic versioning; breaking changes only in major versions; deprecation before removal.

## 10. Engineering standards and toolchain

- **Go 1.26.3** (latest 1.26), pinned in `mise.toml` (currently `"latest"` — to be pinned to `"1.26"` in M0 so the
  toolchain does not silently jump to 1.27). Use generics and generic type aliases now.
- Uber Go Style Guide; KISS/YAGNI/SOLID/12-Factor; `gofmt`; 120-character lines; `errors.Is`/`As`; `any` over
  `interface{}`; small focused interfaces; no pointers to interfaces; factory functions and dependency injection over
  globals.
- `mise` task runner; `golangci-lint`; `govulncheck` (gated); pre-commit hooks are mandatory and never bypassed.
- The `radx` CLI uses the Kong parser foundation (the salvageable part of the prototype CLI).

### 10.1 Future toolchain considerations (no v1 dependency)

Go 1.27 is expected ~August 2026 and its accepted **generic methods** proposal (interface methods still may not declare
type parameters) would allow a later ergonomic refactor of some `As[T]`-style call sites into methods. **No v1
deliverable depends on Go 1.27**; v1 is designed, built, and shipped on Go 1.26.x assuming the no-generic-methods
constraint holds permanently (§8.1).

## 11. Verification and acceptance

### 11.1 Conformance and interop gates (merge-blocking)

A change is not "done" until the interop matrix is green: **DIMSE + DICOMweb** interop against **Orthanc** *and*
**dcm4chee-arc** (via `testcontainers`, already a dependency); **FHIR** output validated by the official **HL7 FHIR
validator**; **DICOM** files validated by **`dciodvfy`** (dcmtk); and **round-trip** against vendored `pydicom`/
`pynetdicom`/`python-hl7` fixtures and corpora (§5.3 vendoring decision: test corpora only, with license attribution).

### 11.2 Basic safety checks (CI)

Lightweight, sane-default checks — not a compliance regime (compliance and PHI governance are the consumer's, per §9.1
and §9.5):

- **PHI-default sanity test** — run representative commands and servers at default verbosity against fixtures with known
  PHI tokens; flag any token surfacing in stdout, stderr, logs, or telemetry, confirming the no-PHI-by-default behaviour.
- **Bind-default test** — assert servers and daemons listen on loopback unless explicitly opted out.
- **Hostile-input run** — execute a malformed-input corpus under a memory cap; must not OOM, panic, or hang.
- **`govulncheck`** in CI. SBOM and signed releases are recommended, not required.

The PS3.15 de-identification profile is tested like any feature (correctness against an attribute checklist), not gated
as a compliance bar; its documented scope and limits are what the consumer relies on.

### 11.3 Performance benchmarks (versus reference libraries)

Comparative benchmarks against the reference libraries are part of verification, tracked over time to catch regressions
and to substantiate the "Go performance" claim. The **priority area is compression and transfer syntax**: encode/decode
throughput and allocations for the supported transfer syntaxes (uncompressed Implicit/Explicit VR LE, Deflated, RLE, and
the JPEG / JPEG-LS / JPEG 2000 / HTJ2K codecs where available), benchmarked against `pydicom` (with its `pylibjpeg` /
`gdcm` plugins) for DICOM pixel pipelines and against `pynetdicom` for DIMSE C-STORE / transfer throughput. Secondary:
FHIR (de)serialization versus `fhir.resources`, and HL7 v2 parse throughput versus `python-hl7`. Benchmarks run on a
documented fixture set; results are published and reviewed per release (not merge-gated).

### 11.4 Other verification

Published Conformance Statements per standard; table-driven and golden-file unit tests; fuzzing for binary parsers,
HTTP/MLLP parsers, and codecs; coverage targets of 80% overall and 90% on critical paths; benchmark suites. Every bug
fix ships a regression test.

## 12. Salvage strategy — port versus rewrite

From the Codex audit:

| Verdict | Components |
|---------|-----------|
| **REWRITE** | `dimse/dul`, `dimse/dimse`, `dimse/scp`; DICOM root parser/writer, `dicom/value`, `dicom/pixel`, `dicom/anonymize`; the FHIR code generator; all `radx` command semantics |
| **PORT-WITH-FIXES** | `dimse/pdu`, `dimse/scu` (incl. its existing C-FIND/GET/MOVE paths), the Orthanc integration tests; `dicom/element`, `dicom/vr`, `dicom/tag`, `dicom/uid`, `dicom/datetime`; the FHIR hand-written core; the Kong CLI foundation |

Each verdict maps to the milestone that consumes it (§13). Porting rules: the FHIR generator is *rewritten* (its model
is broken) but the regeneration approach is kept; generated FHIR code is never hand-edited; the Orthanc integration tests
are ported *first* (M0) as a regression net before the DIMSE rewrite.

## 13. Milestones

Skeleton-first: M2 delivers a thin, correct slice of **every** subsystem before any leg is deepened; M3+ deepen legs and
can proceed in parallel once M2 is green.

1. **M0 — Foundations.** `git init`; pin `mise` to Go 1.26; scaffold the multi-module layout (root + `cmd/radx`); CI with
   testcontainers (Orthanc + dcm4chee-arc), the FHIR validator, and `dciodvfy`; wire `zap` + direct OTel; **port the
   Orthanc integration tests**; vendor test corpora; stand up the §11.2 basic safety checks and the §11.3 benchmark harness.
2. **M1 — DICOM data layer.** Correct Part 10 I/O, `SQ`, transfer syntaxes, value/VR types, tags, UID, datetime; gated
   by `dciodvfy` + pydicom round-trips. (Rewrites the DICOM core; ports element/vr/tag/uid/datetime with fixes.)
3. **M2 — Walking skeleton (thin, end-to-end).** The thinnest correct path through every subsystem: parse+validate one
   DICOM instance; one DIMSE C-ECHO + C-STORE (SCU+SCP, ports `dimse/pdu`+`dimse/scu`, rewrites `dimse/dul`); one
   DICOMweb STOW + WADO read; parse one HL7 `ORM` and emit one FHIR `ServiceRequest`; produce one FHIR
   `DiagnosticReport`; one DICOM→FHIR `ImagingStudy`. Interop-gated. Proves the architecture before depth.
4. **M3 — DIMSE depth.** C-FIND/C-GET/C-MOVE (extending the ported SCU paths), Modality Worklist SCU **and** reference
   SCP, MPPS-SCU, Storage-Commitment-SCU; full presentation-context presets and negotiation.
5. **M4 — DICOMweb depth.** Full WADO-RS/STOW-RS/QIDO-RS client and server.
6. **M5 — HL7 v2 depth.** Typed segments/messages, Batch/File, MLLP client/server with ACK.
7. **M6a — FHIR generator + R5 core.** Rewritten generator, primitives, choice types, validation, bundles, summary
   mode — R5 first, gated by the FHIR validator.
8. **M6b — FHIR R4.** Full R4 (4.0.1) generation; R4-conformance for the workflow set.
9. **M7 — Conversion + CLI + daemons.** DICOM↔FHIR, SR↔FHIR, HL7v2↔FHIR; dcmtk-parity CLI; thin reference daemons.
10. **M8 — Hardening + v1.** PS3.15 de-identification (with documented scope), fuzzing, performance benchmarks vs the
    reference libraries, transport security, conformance statements, docs.

## 14. Risks

- **Scope breadth.** v1 is the full workflow vertical across three standards. Mitigation: M2 walking-skeleton-first keeps
  an end-to-end path correct from early on; M3+ legs are independently shippable.
- **FHIR generator correctness** is the highest-leverage single component. Mitigation: split into M6a/M6b; golden tests
  against the FHIR validator with choice-type and primitive-extension cases first.
- **Codec/CGo fragility.** Mitigation: optional CGo behind a build tag; pure-Go core; clear degradation.
- **DICOMweb has no Python parity floor.** Mitigation: measured against dc4che and the DICOMweb standard directly.
- **De-identification correctness.** Mitigation: test the PS3.15 profile against an attribute checklist and document its
  scope and limits clearly; the library provides the capability, and the consumer owns their de-identification policy.
- **Conformance-subset creep.** Mitigation: the Conformance Statement is the single source of truth; growth is reviewed.

## 15. Open questions

None blocking. Resolved at review: FHIR releases = R4 4.0.1 + R5 (STU3 out, R4B deferred); DICOM SR ↔ FHIR and
MPPS-SCU + Storage-Commitment-SCU are in v1; test corpora are vendored with attribution. Remaining low-priority item:
confirm the shape of the optional data-modification audit hook (§9.5), if a consumer needs one, during M0.

## 16. Success metrics

- Green reference-interop matrix (Orthanc + dcm4chee-arc + FHIR validator + `dciodvfy`) and passing §11.2 basic safety
  checks on every CI run.
- Performance benchmarks against the reference libraries tracked per release with no unexplained regression — especially
  for compression and transfer-syntax encode/decode.
- The radiology workflow demonstrated end-to-end via the `radx` CLI against a reference PACS, including procedure-step
  status (MPPS) and storage confirmation.
- Published Conformance Statements for DICOM, HL7 v2, and FHIR.
- A public API a Go developer can use correctly without reading the DICOM/HL7/FHIR standards — the type system and docs
  carry the safety the reference libraries leave to runtime.
