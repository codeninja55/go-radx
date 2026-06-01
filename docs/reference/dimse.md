# DIMSE networking

The `dimse` package implements the DICOM Message Service Element and its transport layer — the
[DICOM Upper Layer protocol](https://dicom.nema.org/medical/dicom/current/output/chtml/part08/PS3.8.html) (DUL, PS3.8)
and the DIMSE-C / DIMSE-N services (PS3.7). It is the network plane of go-radx: it carries DICOM datasets, defined by
the `dicom` package, between Application Entities over TCP, with optional TLS.

This document is the normative public API contract for `dimse`. The implementation conforms to it. It commits the type
surface, the negotiation model, the PS3.8 DUL state machine, the service-class operations, the streaming query contract,
the typed status model, and the conformance scope and limits. Behaviour is grounded in `pynetdicom` as the parity
reference (PRD §6.2) and in the Codex audit of the prototype (`docs/prd/go-radx-prd.md` §2.2, §12), whose defects
this contract exists to prevent recurring.

## Scope

In scope for v1:

- Application Entity and `AETitle` value type.
- Association lifecycle: A-ASSOCIATE negotiation, A-RELEASE, A-ABORT, A-P-ABORT, with `context.Context` cancellation.
- Presentation-context negotiation: max PDU length, SCP/SCU role selection, asynchronous-operations window, user
  identity (types 1–5), SOP-class extended and common-extended negotiation, presentation-context presets.
- The PS3.8 Table 9-10 DUL finite state machine: 13 states (including release-collision Sta9–Sta12), 19 events, 28
  actions.
- PDU and PDV encode/decode with hostile-input hardening.
- DIMSE-C as both SCU and SCP: C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE, plus C-CANCEL (driven by `context.Context`).
- DIMSE-N **SCU only** in v1: MPPS (N-CREATE / N-SET) and Storage Commitment (N-ACTION / N-EVENT-REPORT).
- The streaming multi-response query/retrieve contract as `iter.Seq2[Status, *dicom.DataSet]`.
- Typed `Status`, `Priority`, and `QueryLevel`.
- TLS 1.2+ (preferring 1.3) with peer-certificate verification and a documented mutual-TLS option.

Out of scope for v1 (architected-for, deferred — PRD §3.2, §5.1):

- The **SCP / server sides** of MPPS and Storage Commitment.
- The remaining DIMSE-N services (N-GET, N-DELETE as standalone SCU operations beyond the MPPS/Storage-Commitment
  flows), Print Management, and the Unified Procedure Step (UPS) service classes.
- Non-DICOM transports.

The `dimse` package reuses `dicom.TransferSyntax`, `dicom.UID`, `dicom.DataSet`, `dicom.SOPClassUID`, and
`dicom.SOPInstanceUID` rather than re-declaring them; those SOP types are owned by the `dicom` package
(`UBIQUITOUS_LANGUAGE.md`, cross-standard collision table). The package never declares a bare, ambiguous `Context`
type: presentation context is `dimse.PresentationContext`, distinct from Go's `context.Context`.

## Overview of the layers

DIMSE networking is three stacked layers, each a clear responsibility boundary (PRD §8.2, single-responsibility):

1. **DUL (PS3.8)** — the state machine and PDU codec. Owns the socket, the ARTIM timer, PDU framing, and the
   13-state lifecycle. Knows nothing about DICOM messages, only PDUs.
2. **ACSE / association** — negotiates the association (`A_ASSOCIATE`), tracks accepted presentation contexts, exposes
   `Association` to callers. Drives the DUL with A-ASSOCIATE / A-RELEASE / A-ABORT primitives.
3. **DIMSE message layer (PS3.7)** — builds and parses Command Sets and conditional Data Sets, fragments them into
   PDVs honouring the command-last and dataset-last boundaries, and dispatches typed C/N service operations.

The prototype conflated these and shipped them broken; the rewrite keeps them separate (PRD §12, verdict: REWRITE
`dimse/dul`, `dimse/dimse`, `dimse/scp`; PORT-WITH-FIXES `dimse/pdu`, `dimse/scu`).

## Core value types

```go
// AETitle is a DICOM Application Entity Title: 1–16 characters, default character repertoire, no leading/trailing
// padding semantics beyond the 16-byte field. It is a named type, never a bare string (PRD §8.2, glossary rule 1).
type AETitle string

// ParseAETitle validates length (1..16) and the allowed character set, returning a typed error on violation.
func ParseAETitle(s string) (AETitle, error)

func (a AETitle) String() string
func (a AETitle) Valid() bool
```

```go
// Priority is the DIMSE message priority. The zero value is MEDIUM (0x0000) — the footgun the enum guards
// (glossary, DIMSE section): callers who forget to set priority get MEDIUM, which is intentional and documented.
type Priority uint16

const (
	PriorityMedium Priority = 0x0000 // zero value
	PriorityHigh   Priority = 0x0001
	PriorityLow    Priority = 0x0002
)
```

```go
// QueryLevel is the Query/Retrieve Level (0008,0052) for C-FIND/C-GET/C-MOVE. It maps to DICOMweb resource paths.
type QueryLevel uint8

const (
	QueryLevelPatient QueryLevel = iota
	QueryLevelStudy
	QueryLevelSeries
	QueryLevelImage // SOP Instance level
)

// DICOM keyword as written into (0008,0052): "PATIENT", "STUDY", "SERIES", "IMAGE".
func (l QueryLevel) String() string
func ParseQueryLevel(s string) (QueryLevel, error)
```

## Typed status

DIMSE status is a 16-bit value whose categorisation (Success / Pending / Warning / Cancel / Failure) is defined per
service class (PS3.7 Annex C plus PS3.4 service-class annexes). go-radx returns it as a typed `Status` whose category
methods read in English and whose `String()` renders the registered meaning by name, never bare hex (PRD §8.2, rule 2).

A `Status` carries only its 16-bit code. The category and the human-readable meaning are **derived**, never stored as
settable fields, so a caller can never author a status whose category contradicts its code (PRD §8.1). Construct a
`Status` from a named constant (`StatusStoreSuccess`, `StatusStoreCannotUnderstand`, and the rest of the per-service
table) or from `NewStatus`, which binds the raw code to the service class that decides how to categorise it.

```go
// Status is a DIMSE status value (0000,0900). It wraps only the wire code; Category() and Meaning() are derived
// against the service class the status was constructed with, so the same numeric code reads correctly in context
// (e.g. 0xB000 is a Storage warning, not a generic value).
type Status struct {
	Code uint16
}

// NewStatus binds a raw status code to a service class, fixing how Category()/Meaning() resolve it. Prefer the named
// constants below; reach for NewStatus only when handling a code received from the wire.
func NewStatus(code uint16, sc ServiceClass) Status

// ServiceClass selects the per-class categorisation table (general, Verification, Storage, FIND, MOVE, GET, Worklist,
// Procedure Step, Storage Commitment) used to resolve a status code's category and meaning.
type ServiceClass uint8

const (
	ServiceClassGeneral ServiceClass = iota
	ServiceClassVerification
	ServiceClassStorage
	ServiceClassFind
	ServiceClassMove
	ServiceClassGet
	ServiceClassWorklist
	ServiceClassProcedureStep
	ServiceClassStorageCommitment
)

type StatusCategory uint8

const (
	StatusCategoryUnknown StatusCategory = iota
	StatusCategorySuccess
	StatusCategoryPending
	StatusCategoryWarning
	StatusCategoryCancel
	StatusCategoryFailure
)

// Category resolves the Success/Pending/Warning/Cancel/Failure class from the code and its service class.
func (s Status) Category() StatusCategory

// Meaning returns the registered meaning, e.g. "Refused: Out of Resources". Empty for bare Success/Cancel.
func (s Status) Meaning() string

func (s Status) IsSuccess() bool // Category() == StatusCategorySuccess
func (s Status) IsPending() bool // Category() == StatusCategoryPending (a continuing C-FIND/C-GET/C-MOVE match)
func (s Status) IsWarning() bool // Category() == StatusCategoryWarning
func (s Status) IsCancel() bool  // Category() == StatusCategoryCancel
func (s Status) IsFailure() bool // Category() == StatusCategoryFailure

// String renders "0xC000 Failure: Unable to Process" — name and class, the §8.2 legibility rule.
func (s Status) String() string
```

Named status constants give callers (and SCP handlers) values they cannot mis-categorise. Each constant pairs a code
with the service class that defines its meaning, so `StatusStoreCannotUnderstand.Category()` is always `Failure`:

```go
const (
	// General / shared
	StatusSuccess Status = /* 0x0000, ServiceClassGeneral */
	StatusCancel  Status = /* 0xFE00, ServiceClassGeneral */

	// Storage service class
	StatusStoreSuccess         Status = /* 0x0000, ServiceClassStorage */
	StatusStoreOutOfResources  Status = /* 0xA700, ServiceClassStorage */
	StatusStoreDataSetMismatch Status = /* 0xA900, ServiceClassStorage */
	StatusStoreCannotUnderstand Status = /* 0xC000, ServiceClassStorage */

	// Query/Retrieve FIND
	StatusFindPending Status = /* 0xFF00, ServiceClassFind */
	StatusFindSuccess Status = /* 0x0000, ServiceClassFind */

	// Verification
	StatusEchoSuccess Status = /* 0x0000, ServiceClassVerification */
)
```

Status resolution is service-class aware, mirroring `pynetdicom/status.py`. The categorisation tables ported are:
general (PS3.7 Annex C), Verification, Storage (including the ranged `A700–A7FF` out-of-resources, `A900–A9FF`
data-set-mismatch, and `C000–CFFF` cannot-understand bands), Query/Retrieve FIND, MOVE, and GET, Modality Worklist,
and the two N-service tables needed in v1 (Procedure Step / MPPS and Storage Commitment). A code with no registered
meaning in the active service class resolves to category `StatusCategoryUnknown` with the raw code preserved; it is
never silently coerced to success.

Selected named codes (illustrative, not exhaustive — the full tables live in the implementation and the Conformance
Statement):

| Code | Category | Meaning |
|------|----------|---------|
| `0x0000` | Success | (none) |
| `0xFF00` | Pending | Matches/sub-operations continuing |
| `0xFF01` | Pending | Continuing, warning (optional keys not supported) |
| `0xFE00` | Cancel | Operation terminated by C-CANCEL |
| `0xB000` | Warning | Sub-operations complete, one or more failures (C-GET/C-MOVE) |
| `0xA700`–`0xA7FF` | Failure | Refused: out of resources |
| `0xA801` | Failure | Move destination unknown |
| `0xC000`–`0xCFFF` | Failure | Unable to process / cannot understand |

## Application Entity

An `AE` is the local DICOM network endpoint. It is the factory for outbound associations (SCU) and inbound listeners
(SCP). It carries no global mutable state (PRD §9.4); every knob is per-`AE` and is set with functional options.

```go
// AE is a local DICOM Application Entity. Construct one per local endpoint. Safe for concurrent use.
type AE struct { /* unexported */ }

// NewAE constructs an AE with the given local AE Title and options. Zero options yield safe defaults
// (see the timeout table below). Returns a typed error if the title is invalid.
func NewAE(title AETitle, opts ...AEOption) (*AE, error)

type AEOption func(*aeConfig)

// Timeouts mirror pynetdicom defaults; all are also overridable per-association via context deadlines.
func WithACSETimeout(d time.Duration) AEOption        // default 30s — association negotiation/release
func WithDIMSETimeout(d time.Duration) AEOption       // default 30s — awaiting a DIMSE response
func WithNetworkTimeout(d time.Duration) AEOption     // default 60s — idle association
func WithConnectionTimeout(d time.Duration) AEOption  // default none — TCP connect
func WithMaxPDULength(n MaxPDULength) AEOption         // default 16382; 0 = unlimited (see negotiation)
func WithImplementationClassUID(uid dicom.UID) AEOption
func WithImplementationVersionName(name string) AEOption
func WithTLS(cfg *tls.Config) AEOption                // see "Transport security"

// WithStoreHandler registers the sink for instances received as C-GET sub-operations (the requestor stores them
// itself). Required before calling Association.Get.
func WithStoreHandler(h StoreHandler) AEOption

// WithCommitmentHandler registers the sink for the Storage Commitment N-EVENT-REPORT result.
func WithCommitmentHandler(h CommitmentHandler) AEOption
```

### Establishing an outbound association (SCU)

```go
// Associate opens an A-ASSOCIATE-RQ to the peer and blocks until accepted, rejected, aborted, or ctx is cancelled.
// On success it returns an established Association; on rejection/abort it returns a typed *AssociationError.
func (ae *AE) Associate(
	ctx context.Context,
	addr string, // "host:port"
	called AETitle,
	contexts []PresentationContext,
	opts ...AssociateOption,
) (*Association, error)

type AssociateOption func(*associateConfig)

func WithRoleSelection(sel ...RoleSelection) AssociateOption
func WithAsyncOps(invoked, performed uint16) AssociateOption // window negotiation
func WithUserIdentity(id UserIdentity) AssociateOption
func WithExtendedNegotiation(items ...SOPClassExtendedNegotiation) AssociateOption
func WithCommonExtendedNegotiation(items ...SOPClassCommonExtendedNegotiation) AssociateOption
```

### Serving inbound associations (SCP)

```go
// Server is an embeddable DIMSE SCP. It binds to loopback by default (PRD §9.1); a non-loopback bind is explicit.
type Server struct { /* unexported */ }

// NewServer builds an SCP for the AE, advertising the supported presentation contexts and dispatching to handlers.
func NewServer(ae *AE, supported []PresentationContext, h Handler, opts ...ServerOption) *Server

// ListenAndServe binds and serves until ctx is cancelled or Shutdown is called. Default bind is 127.0.0.1.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error

// Shutdown stops accepting, closes active association connections, and waits for in-flight handlers, bounded by ctx.
// (The prototype's Shutdown left handlers blocked in ReadPDU; this contract closes connections first — DIMSE-014.)
func (s *Server) Shutdown(ctx context.Context) error

type ServerOption func(*serverConfig)

func WithMaxAssociations(n int) ServerOption       // capacity is acquired before spawning the handler (Codex DIMSE-013)
func WithRequireCalledAETitle(t AETitle) ServerOption
func WithRequireCallingAETitles(t ...AETitle) ServerOption
```

## Negotiation primitives

These are the user-information sub-items negotiated inside A-ASSOCIATE (PS3.7 Annex D, PS3.8). They mirror the
`pynetdicom/pdu_primitives.py` primitives and are exposed as typed Go structs so callers never assemble raw sub-items.

```go
// MaxPDULength is the negotiated largest acceptable P-DATA-TF PDU. 0 means "no maximum specified" (unlimited),
// per PS3.8 Annex D.1 — represented internally as unlimited, never as a literal allocation size. A non-zero value
// that cannot fit a PDV header is rejected with a typed error (Codex DIMSE-005, DIMSE-012).
type MaxPDULength uint32

// RoleSelection is SCP/SCU role-selection negotiation (PS3.7 D.3.3.4) for one abstract syntax. Required to act as an
// SCP for C-GET (the requestor must accept the SCP role for the Storage SOP Classes used by sub-operations).
type RoleSelection struct {
	AbstractSyntax dicom.SOPClassUID
	SCURole        bool
	SCPRole        bool
}

// AsyncOps is asynchronous-operations-window negotiation (PS3.7 D.3.3.3). pynetdicom negotiates but never delivers
// real async; go-radx negotiates the window AND delivers genuine concurrency via goroutines (PRD §6.3).
type AsyncOps struct {
	MaxOperationsInvoked   uint16 // 0 = unlimited
	MaxOperationsPerformed uint16 // 0 = unlimited
}

// UserIdentity is user-identity negotiation (PS3.7 D.3.3.7). Type is one of the five standard forms.
type UserIdentity struct {
	Type                     UserIdentityType
	PrimaryField             []byte // username, Kerberos ticket, SAML assertion, or JWT
	SecondaryField           []byte // passcode, for UserIdentityUsernamePasscode only
	PositiveResponseRequested bool
}

type UserIdentityType uint8

const (
	UserIdentityUsername         UserIdentityType = 1 // username, UTF-8
	UserIdentityUsernamePasscode UserIdentityType = 2 // username + passcode, UTF-8
	UserIdentityKerberos         UserIdentityType = 3 // Kerberos service ticket
	UserIdentitySAML             UserIdentityType = 4 // SAML assertion
	UserIdentityJWT              UserIdentityType = 5 // JSON Web Token
)

// SOPClassExtendedNegotiation is per-SOP-class extended negotiation (PS3.7 D.3.3.5): an opaque service-class
// application-information blob keyed by SOP Class UID.
type SOPClassExtendedNegotiation struct {
	SOPClassUID            dicom.SOPClassUID
	ServiceClassAppInfo    []byte
}

// SOPClassCommonExtendedNegotiation is common-extended negotiation (PS3.7 D.3.3.6).
type SOPClassCommonExtendedNegotiation struct {
	SOPClassUID            dicom.SOPClassUID
	ServiceClassUID        dicom.UID
	RelatedGeneralSOPClasses []dicom.SOPClassUID
}
```

Secrets carried in `UserIdentity` (passcodes, tokens) are never logged and never written to any catalogue (PRD §9.8).

## Presentation contexts and presets

A `PresentationContext` pairs one Abstract Syntax (a SOP Class) with one or more proposed Transfer Syntaxes, keyed by an
odd context ID (PS3.8 9.3.2.2). On the acceptor side it carries the negotiation result and the single accepted transfer
syntax.

```go
// PresentationContext is a proposed or accepted presentation context. ID must be odd (PS3.8 9.3.2.2).
type PresentationContext struct {
	ID              uint8                  // odd, 1..255
	AbstractSyntax  dicom.SOPClassUID      // the SOP Class proposed
	TransferSyntaxes []dicom.TransferSyntax // proposed list (RQ) or single accepted (AC)
	// Result is meaningful on an accepted context (acceptance, or a rejection reason).
	Result          ContextResult
}

type ContextResult uint8

const (
	ContextAccepted ContextResult = iota
	ContextUserRejected
	ContextNoReason
	ContextAbstractSyntaxNotSupported
	ContextTransferSyntaxesNotSupported
)

// NewPresentationContext builds a proposal for one SOP Class with the given transfer syntaxes. Default transfer-syntax
// set when none given is the four uncompressed/deflated syntaxes (see below).
func NewPresentationContext(id uint8, abstract dicom.SOPClassUID, ts ...dicom.TransferSyntax) PresentationContext
```

### Default transfer syntaxes

When a context is built without an explicit transfer-syntax list, the proposed set is the four uncompressed/deflated
syntaxes the core library reads and writes (PRD §6.2 DICOM floor). The order is the proposal preference: the acceptor
may pick the first acceptable, so Explicit VR Little Endian leads. This literal is identical to the one in
`docs/conformance/dicom.md`:

```go
var DefaultTransferSyntaxes = []dicom.TransferSyntax{
	dicom.ExplicitVRLittleEndian,         // 1.2.840.10008.1.2.1
	dicom.ImplicitVRLittleEndian,         // 1.2.840.10008.1.2
	dicom.DeflatedExplicitVRLittleEndian, // 1.2.840.10008.1.2.1.99
	dicom.ExplicitVRBigEndian,            // 1.2.840.10008.1.2.2 (retired)
}
```

The full registered transfer-syntax set recognised for negotiation tracks the `dicom` package's transfer-syntax
registry (mirroring the `pynetdicom` floor). go-radx **decodes** every supported compressed syntax it negotiates;
**encoding/transcoding** is available only where a codec exists (RLE and JPEG 2000 lossless first), behind the optional
CGo build tag and off by default (explicit opt-in — PRD §7.3, see `docs/conformance/dicom.md` for the per-syntax
decode-only vs decode+encode matrix). Negotiation may still accept a JPEG/JPEG-LS/JPEG 2000/HTJ2K context whose codec
is not compiled in; pixel decode then degrades to a typed "codec unavailable" error rather than a panic.

### Presets

Presets are curated bundles of presentation contexts for common roles — a go-radx helper concept, not a standards term
(glossary). They mirror the `pynetdicom/presentation.py` exports. Each is a function returning a fresh slice so callers
may mutate without sharing state:

The context counts these functions return are the go-radx conformance subset, defined authoritatively in
`docs/conformance/dicom.md` (the source of truth for scope). The `pynetdicom` parity floor (120 selected / 170 all
Storage, 13 Q/R) is cited only as the upstream reference these presets filter from.

```go
func VerificationContexts() []PresentationContext        // 1 context (Verification SOP Class)
func StorageContexts() []PresentationContext             // 36 contexts (validated radiology Storage set)
func AllStorageContexts() []PresentationContext          // 170 contexts (every registered Storage SOP Class)
func QueryRetrieveContexts() []PresentationContext       // 6 contexts (Patient Root + Study Root Q/R models)
func BasicWorklistContexts() []PresentationContext       // 1 context (Modality Worklist Information Model — FIND)
func ModalityPerformedContexts() []PresentationContext   // 1 context (MPPS SOP Class, for the MPPS SCU)
func StorageCommitmentContexts() []PresentationContext   // 1 context (Storage Commitment Push Model SOP Class)
```

`StorageContexts()` returns the 36-class validated radiology set — round-trip-tested and interop-verified —
intentionally narrower than the 120-class `pynetdicom` selected-Storage floor. `AllStorageContexts()` proposes all 170
registered Storage SOP Classes for transport-only use. Because the A-ASSOCIATE-RQ has a 128-context limit, a single
`AllStorageContexts()` proposal must be split across associations; the curated `StorageContexts()` set stays well under
the limit. The accepted side always returns a single transfer syntax per context; rejected contexts still encode exactly
one (insignificant) transfer-syntax sub-item, which the prototype omitted (Codex DIMSE-008).

## The DUL state machine (PS3.8 Table 9-10)

The DICOM Upper Layer is a faithful, table-driven implementation of PS3.8 Table 9-10. The prototype shipped a two-state
release model that omitted the release-collision states and silently closed the socket on unexpected PDUs (Codex
DIMSE-010, DIMSE-011); this contract replaces it with the full machine.

```go
// State is a DUL state, Sta1..Sta13 (PS3.8 Table 9-10).
type State uint8

const (
	Sta1  State = iota + 1 // Idle
	Sta2                   // Transport connection open (awaiting A-ASSOCIATE-RQ)
	Sta3                   // Awaiting local A-ASSOCIATE response
	Sta4                   // Awaiting transport connection open to complete
	Sta5                   // Awaiting A-ASSOCIATE-AC or A-ASSOCIATE-RJ
	Sta6                   // Association established, ready for data transfer
	Sta7                   // Awaiting A-RELEASE-RP
	Sta8                   // Awaiting local A-RELEASE response
	Sta9                   // Release collision, requestor side: awaiting local A-RELEASE response
	Sta10                  // Release collision, acceptor side: awaiting A-RELEASE-RP
	Sta11                  // Release collision, requestor side: awaiting A-RELEASE-RP
	Sta12                  // Release collision, acceptor side: awaiting local A-RELEASE response
	Sta13                  // Awaiting transport connection close
)
```

The machine recognises all **19 events** (Evt1–Evt19): the A-ASSOCIATE request and the transport-connect confirmation;
the four association PDUs received (AC, RJ, RQ, plus the transport-connection indication); the accept/reject response
primitives; P-DATA request and P-DATA-TF received; the A-RELEASE request, A-RELEASE-RQ, A-RELEASE-RP, and A-RELEASE
response; the A-ABORT request and A-ABORT PDU; transport-closed indication; ARTIM-timer expiry; and crucially **Evt19,
unrecognised or invalid PDU received**, which the prototype handled by silently closing the socket.

It performs all **28 actions** in four families: association establishment (AE-1 … AE-8), data transfer (DT-1, DT-2),
association release (AR-1 … AR-10, the family that carries the Sta9–Sta12 release-collision transitions), and abort
(AA-1 … AA-8). AA-3 distinguishes a user-initiated A-ABORT indication from a provider-initiated A-P-ABORT indication;
AA-8 is the "send A-ABORT (provider source), issue A-P-ABORT, start ARTIM" action triggered by Evt19. Unexpected or
invalid PDUs are turned into A-ABORT with the correct provider source and reason before the connection closes, never a
silent TCP close (Codex DIMSE-011).

The state machine is internal; callers do not drive it directly. It is exposed for observability through the
`EVT_FSM_TRANSITION` event (see Handlers) so operators can trace lifecycle without reading raw PDUs.

## PDU and PDV

PDUs (A-ASSOCIATE-RQ/AC/RJ, P-DATA-TF, A-RELEASE-RQ/RP, A-ABORT) are encoded and decoded with mandatory bounds checks.
A DIMSE message (PS3.7 6.3) is a Command Set optionally followed by a Data Set; each is fragmented into Presentation
Data Values (PDVs) inside P-DATA-TF PDUs. Every PDV carries a one-byte message-control header whose two low bits are the
**command bit** (bit 0: 1 = command, 0 = dataset) and the **last-fragment bit** (bit 1).

The prototype's foundational DIMSE defect was mishandling these two bits, which was the concrete root cause of the
Orthanc aborts (PRD §2.2, Codex DIMSE-001 / DIMSE-002). This contract fixes it precisely:

- The **final command fragment is always marked last** (`0x03`), independently of whether a dataset follows.
  Command-last and dataset-last are separate boundaries (Codex DIMSE-001).
- The reassembler is a **state machine**: it collects command fragments until command-last, decodes
  `CommandDataSetType (0000,0800)`, and only then waits for dataset-last when a dataset is actually present (Codex
  DIMSE-002). It never dispatches a C-STORE/C-FIND/C-GET with a `nil` dataset that a compliant peer is still sending.
- Datasets are decoded with the **negotiated transfer syntax** of the presentation context, not hard-coded Implicit VR
  Little Endian (Codex DIMSE-003).
- Command Sets are built in **increasing tag order** with Command Group Length `(0000,0000)` computed last, and each
  command element uses its **dictionary VR** — Move Destination `(0000,0600)` is VR `AE`, not `UI` (Codex DIMSE-006,
  DIMSE-007).

PDV decoding rejects `length < 2` before subtracting the header, rejects item lengths exceeding the bytes remaining in
the PDU body, and reads from a bounded reader so a hostile P-DATA-TF cannot trigger a multi-gigabyte allocation (Codex
DIMSE-004; PRD §9.3). Fragmentation against a negotiated max PDU of `0` is treated as unlimited with a configured local
send cap, never a negative slice bound (Codex DIMSE-005).

The PDU/PDV codec is not part of the everyday public surface; callers use the service operations below. It is documented
here because its correctness is the load-bearing fix for the prototype's interoperability failures.

## Association and service operations

An established `Association` exposes the DIMSE-C and (SCU) DIMSE-N operations. All take a `context.Context`; cancelling
it sends a C-CANCEL for an in-flight query/retrieve and otherwise aborts the operation. Every operation guards against
being called on an unestablished or released association and returns a typed error rather than panicking (DIMSE-017).

```go
type Association struct { /* unexported */ }

// State reports the current DUL state (for observability).
func (a *Association) State() State

// AcceptedContexts returns the presentation contexts the peer accepted, each with its single transfer syntax.
func (a *Association) AcceptedContexts() []PresentationContext

// Release performs a graceful A-RELEASE (handling release collision per Sta9–Sta12). Bounded by ctx.
func (a *Association) Release(ctx context.Context) error

// Abort sends a user-initiated A-ABORT.
func (a *Association) Abort(ctx context.Context) error
```

### C-ECHO

```go
// Echo sends a C-ECHO (Verification) and returns the peer's status.
func (a *Association) Echo(ctx context.Context) (Status, error)
```

### C-STORE

```go
// Store transmits one dataset via C-STORE. The presentation context is selected by the dataset's SOP Class UID, and the
// dataset is encoded in that context's negotiated transfer syntax; if no accepted context matches, Store returns a typed
// error and transmits nothing. It never reports success on work it did not do (PRD §9.2 fail-closed rule, which the
// prototype's `store` violated).
//
// Limitation: selection is by SOP Class only. The dicom.DataSet does not carry its source transfer syntax (that lives on
// the File Meta group, separate from the dataset), so Store cannot yet prefer a context whose accepted transfer syntax
// matches the dataset's own. Transfer-syntax-faithful selection — required to send compressed pixel data as-is and avoid
// transcoding that would corrupt it — is deferred to a later increment. Today Store assumes the dataset is encodable in
// the context's negotiated transfer syntax.
func (a *Association) Store(ctx context.Context, ds *dicom.DataSet, opts ...StoreOption) (Status, error)

type StoreOption func(*storeConfig)

func WithStorePriority(p Priority) StoreOption
// For sub-operation stores (e.g. inside a C-MOVE/C-GET origin) the move originator can be propagated.
func WithMoveOriginator(aet AETitle, msgID uint16) StoreOption
```

### C-FIND, C-GET, C-MOVE — the streaming query contract

Query and retrieve operations produce **multiple responses**: zero or more `Pending` statuses (each carrying a matching
or sub-operation dataset) followed by a single terminal `Success`, `Warning`, `Cancel`, or `Failure`. go-radx exposes
this as a Go 1.23+ iterator, not a callback (PRD §8.1), so callers consume responses with `range` and can stop early.
Stopping (or cancelling `ctx`) sends a C-CANCEL.

These signatures extend the PRD §8.1 committed form `Find(ctx, q, lvl) iter.Seq2[Status, *dicom.DataSet]` with a
functional-options variadic (`opts ...QueryOption`) and clearer parameter names. Parameter names are illustrative; the
type-level shape (receiver, value parameters, `iter.Seq2[Status, *dicom.DataSet]` return) is the committed contract.

```go
// Find issues a C-FIND and yields (Status, identifier) for each response. The terminal status yields a nil dataset.
// Breaking out of the range loop, or cancelling ctx, sends a C-CANCEL for the operation's Message ID.
func (a *Association) Find(
	ctx context.Context,
	query *dicom.DataSet,
	level QueryLevel,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet]

// Get retrieves matching instances to THIS AE over the same association (C-GET). Each yielded value is a sub-operation
// status; received instances are delivered to the StoreHandler registered on the AE via WithStoreHandler (C-GET
// requires the requestor to accept the SCP role for the relevant Storage SOP Classes — see RoleSelection).
func (a *Association) Get(
	ctx context.Context,
	query *dicom.DataSet,
	level QueryLevel,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet]

// Move retrieves matching instances to a separate destination AE (C-MOVE). dest is the Move Destination AE Title.
func (a *Association) Move(
	ctx context.Context,
	query *dicom.DataSet,
	level QueryLevel,
	dest AETitle,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet]

type QueryOption func(*queryConfig)

func WithQueryPriority(p Priority) QueryOption
// Override the query/retrieve Information Model SOP Class (default chosen from level + operation).
func WithQueryModel(sopClass dicom.SOPClassUID) QueryOption
```

Two behaviours are guaranteed and were defects in the prototype:

1. The requested `QueryLevel` is **always written into `(0008,0052)`** of the identifier before sending; the prototype
   accepted a level argument and silently dropped it (Codex DIMSE-015).
2. The terminal status is reported faithfully. A `Failure` or `Cancel` is surfaced as a non-pending `Status` (and, for
   an operation that produced no usable results, an accompanying error); partial success is not laundered into success
   (PRD §9.2).

Iteration semantics: the iterator yields exactly one terminal status as its final element. If the iterator's `error`
needs to be inspected after the loop (transport failure, abort), use the trailing-error variant where a final yield of a
failure `Status` is paired with a retrievable error via the association:

```go
// LastError returns the transport/protocol error, if any, that terminated the most recent query/retrieve iterator.
// It is nil when the operation completed with a clean terminal DIMSE status.
func (a *Association) LastError() error
```

`LastError()` is set only for transport or protocol faults (a dropped connection, an A-ABORT, a malformed PDU) that end
the iteration before a clean terminal DIMSE status; a terminal `Failure`/`Cancel` `Status` is in-band, not an error. It
is scoped to the most recent iterator on that `Association` and must be read immediately after the `range` loop ends,
before starting another query. An `Association` is **not** safe for concurrent queries: do not run `Find`/`Get`/`Move`
iterators on the same association from multiple goroutines, because `LastError()` is per-association, not per-call.
Concurrency is achieved by opening one association per goroutine (the AE is concurrency-safe).

### C-CANCEL

There is no standalone `Cancel` method on the everyday path: cancellation is expressed by cancelling the `ctx` passed to
`Find`/`Get`/`Move`, or by `break`ing out of the range loop. Both send a C-CANCEL for that operation's Message ID on the
correct presentation context. A low-level escape hatch exists for advanced callers:

```go
// Cancel sends a C-CANCEL for an in-flight operation identified by its Message ID. Prefer ctx cancellation.
func (a *Association) Cancel(ctx context.Context, msgID uint16) error
```

### DIMSE-N SCU: MPPS and Storage Commitment

v1 ships two N-service SCU flows as typed operations rather than raw N-CREATE/N-SET/N-ACTION calls, so the workflow legs
read in domain terms (PRD §5.1, glossary DIMSE-N).

```go
// MPPS is the Modality Performed Procedure Step SCU (N-CREATE then N-SET). It reports procedure start and completion.
type MPPS struct{ assoc *Association }

func (a *Association) MPPS() *MPPS

// Create issues N-CREATE for a new MPPS instance (status "IN PROGRESS"), returning the created SOP Instance UID.
func (m *MPPS) Create(ctx context.Context, attrs *dicom.DataSet) (dicom.SOPInstanceUID, Status, error)

// Set issues N-SET to update an MPPS instance (typically to "COMPLETED" or "DISCONTINUED").
func (m *MPPS) Set(ctx context.Context, instance dicom.SOPInstanceUID, attrs *dicom.DataSet) (Status, error)
```

```go
// StorageCommitment is the Storage Commitment Push Model SCU. It requests commitment via N-ACTION and receives the
// result via an N-EVENT-REPORT from the peer. v1 receives the report on the SAME association: after Request returns,
// the SCU keeps the association open and the result is delivered to the AE's registered CommitmentHandler (see
// WithCommitmentHandler). Receiving the report on a later, peer-initiated association is the deferred SCP-side path
// (scope: N-services SCU only) and is not a v1 guarantee.
type StorageCommitment struct{ assoc *Association }

func (a *Association) StorageCommitment() *StorageCommitment

// Request sends the N-ACTION (action type 1) listing the referenced SOP instances to commit. transactionUID identifies
// the request so the later N-EVENT-REPORT can be correlated.
func (sc *StorageCommitment) Request(
	ctx context.Context,
	transactionUID dicom.UID,
	instances []dicom.ReferencedSOPInstance,
) (Status, error)

// ReferencedSOPInstance is owned by the dicom package (dicom.ReferencedSOPInstance): a (SOP Class UID, SOP Instance
// UID) pair, deliberately NOT named Reference (that noun is FHIR's). dimse and dicomweb reuse the one dicom type
// rather than redeclaring it.

// StorageCommitmentResult is delivered to the AE's CommitmentHandler when the N-EVENT-REPORT arrives.
type StorageCommitmentResult struct {
	TransactionUID dicom.UID
	Successful     []dicom.ReferencedSOPInstance
	Failed         []FailedSOPInstance
}

// FailedSOPInstance is a referenced instance plus a DIMSE failure reason code (0008,1197). Same shape as the
// dicomweb STOW-RS failed-instance entry.
type FailedSOPInstance struct {
	dicom.ReferencedSOPInstance
	FailureReason uint16
}
```

## SCP handlers and the event model

The SCP dispatches inbound operations to a `Handler`. Following `pynetdicom`'s split, events are either **intervention
events** — the service requests an SCP must answer with a status (C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE, and the
N-services) — or **notification events**, fire-and-forget lifecycle signals (`EVT_ESTABLISHED`, `EVT_RELEASED`,
`EVT_ABORTED`, `EVT_ACCEPTED`, `EVT_REJECTED`, `EVT_FSM_TRANSITION`, `EVT_PDU_RECV/SENT`, `EVT_DIMSE_RECV/SENT`).

go-radx models intervention events as interface methods returning typed status, so a handler cannot forget to answer:

```go
// Handler answers inbound DIMSE-C operations. Implement only the methods for services you support; unimplemented
// services are rejected at negotiation or answered with "SOP Class not supported". A handler returning success on work
// it did not store is a defect (PRD §9.2 fail-closed).
type Handler interface {
	// Echo answers a C-ECHO. Return StatusEchoSuccess unless the SCP is degraded.
	Echo(ctx context.Context, info OpInfo) Status

	// Store receives one dataset (C-STORE). Persisting it before returning success is the handler's responsibility;
	// returning success without storing violates the honest-failure rule.
	Store(ctx context.Context, ds *dicom.DataSet, info OpInfo) Status

	// Find yields (Status, match) pairs for a C-FIND query, mirroring the SCU iterator. The runtime sends one
	// Pending response per yielded match and a terminal status when the iterator ends.
	Find(ctx context.Context, query *dicom.DataSet, level QueryLevel, info OpInfo) iter.Seq2[Status, *dicom.DataSet]

	// Get/Move resolve matches and drive C-STORE sub-operations to the requestor (Get) or a named destination (Move).
	Get(ctx context.Context, query *dicom.DataSet, level QueryLevel, info OpInfo) iter.Seq2[Status, *dicom.DataSet]
	Move(
		ctx context.Context,
		query *dicom.DataSet,
		level QueryLevel,
		dest AETitle,
		info OpInfo,
	) iter.Seq2[Status, *dicom.DataSet]
}

// OpInfo carries the per-operation context an SCP needs: the AE Titles, the presentation context, the negotiated
// transfer syntax, and the Message ID. It is the structured, no-PHI diagnostic context (PRD §8.2, §9.1).
type OpInfo struct {
	CallingAETitle  AETitle
	CalledAETitle   AETitle
	PresentationID  uint8
	TransferSyntax  dicom.TransferSyntax
	MessageID       uint16
	SOPClassUID     dicom.SOPClassUID
}
```

Handlers may implement narrower interfaces (interface segregation, PRD §8.2): a store-only SCP implements
`StoreHandler`, a worklist SCP implements `FindHandler`. The dispatcher type-asserts for each. The same `StoreHandler`
is the sink for instances received as C-GET sub-operations on the SCU side (registered via `WithStoreHandler`).

```go
// StoreHandler receives a single dataset, as a C-STORE SCP and as the C-GET sub-operation sink on the requestor.
type StoreHandler interface {
	Store(ctx context.Context, ds *dicom.DataSet, info OpInfo) Status
}

// FindHandler answers a C-FIND query, mirroring the SCU iterator.
type FindHandler interface {
	Find(ctx context.Context, query *dicom.DataSet, level QueryLevel, info OpInfo) iter.Seq2[Status, *dicom.DataSet]
}

// CommitmentHandler receives the Storage Commitment N-EVENT-REPORT result on the SCU. Registered with
// WithCommitmentHandler; invoked once per correlated transaction.
type CommitmentHandler interface {
	Commitment(ctx context.Context, result StorageCommitmentResult)
}
```

Notification events are observed by registering callbacks on the `Server` (or `AE`) for diagnostics and metrics; they
never block the protocol.

When the SCP's C-GET / C-MOVE handler drives sub-operation C-STOREs, each sub-operation gets a **real, distinct Message
ID**, and each C-STORE-RSP is read through the same reassembly loop as a normal response — the prototype used
`MessageID: 0` and read exactly one P-DATA-TF, which miscounted failures and hung against compliant peers (Codex
DIMSE-016).

## Error model

Errors are values (PRD §8.2). Three typed error categories cover the protocol surface, all compatible with
`errors.Is` / `errors.As`:

```go
// AssociationError reports a failed or refused association: A-ASSOCIATE-RJ (with source/reason/result) or an A-ABORT
// during establishment. Rendered with the rejection reason by name, never bare codes.
type AssociationError struct {
	Kind   AssociationErrorKind // Rejected, Aborted, ProviderAborted, Timeout
	Source uint8                // PS3.8 rejection/abort source
	Reason uint8                // PS3.8 reason
}
func (e *AssociationError) Error() string

// AbortError reports an A-ABORT (user) or A-P-ABORT (provider) on an established association.
type AbortError struct {
	Provider bool // true => A-P-ABORT (provider-initiated)
	Source   uint8
	Reason   uint8
}
func (e *AbortError) Error() string

// ProtocolError reports malformed PDUs, length-limit violations, or unexpected PDUs for the current state.
type ProtocolError struct {
	State State
	// Detail names the violated constraint (e.g. "PDV item length below header size") without PHI.
	Detail string
}
func (e *ProtocolError) Error() string
```

A DIMSE `Status` of `Failure` category is **not** an `error` by itself — it is data the caller inspects with
`status.IsFailure()`. Transport, association, and protocol faults are `error`s. This separation lets a caller
distinguish "the peer answered, and said no" (a `Status`) from "the conversation broke" (an `error`). Truncated or
short reads mid-PDU surface as `ProtocolError` wrapping `io.ErrUnexpectedEOF`, never as a clean completion (PRD §9.2).

No operation panics on malformed network input; all length and dimension math is checked before allocation (PRD §9.3).

## Transport security (TLS)

DIMSE-TLS is configured per `AE` and applies to both SCU connections and the SCP listener.

```go
// WithTLS attaches a *tls.Config. The library defaults to TLS 1.2+ (preferring 1.3) and verifies peer certificates;
// it never sets InsecureSkipVerify outside an explicitly flagged test mode (PRD §9.7).
func WithTLS(cfg *tls.Config) AEOption
```

If `cfg.MinVersion` is unset, the library raises it to `tls.VersionTLS12`. Mutual TLS is enabled by supplying client
certificates in the `*tls.Config` (SCU) and requiring client certificates on the SCP side (`ClientAuth:
RequireAndVerifyClientCert`); this is the documented mTLS option (PRD §9.7). Credentials come from environment or
files, never hard-coded, and are never logged (PRD §9.8).

## Worked examples

### C-ECHO (verification)

```go
calling, err := dimse.ParseAETitle("RADX-SCU") // validates length 1..16 and the allowed repertoire
if err != nil {
	log.Fatal(err)
}
called, err := dimse.ParseAETitle("ORTHANC")
if err != nil {
	log.Fatal(err)
}

ae, err := dimse.NewAE(calling)
if err != nil {
	log.Fatal(err)
}

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

assoc, err := ae.Associate(ctx, "pacs.example.org:104", called, dimse.VerificationContexts())
if err != nil {
	log.Fatalf("association failed: %v", err)
}
defer assoc.Release(ctx)

status, err := assoc.Echo(ctx)
if err != nil {
	log.Fatalf("c-echo transport error: %v", err)
}
if !status.IsSuccess() {
	log.Fatalf("c-echo rejected: %s", status) // e.g. "0x0122 Failure: Refused: SOP Class Not Supported"
}
fmt.Println("verification succeeded")
```

### C-STORE (storage)

```go
ds, err := dicom.ReadFile("CT.dcm") // from the dicom package
if err != nil {
	log.Fatal(err)
}

assoc, err := ae.Associate(ctx, "pacs.example.org:104", called, dimse.StorageContexts())
if err != nil {
	log.Fatal(err)
}
defer assoc.Release(ctx)

status, err := assoc.Store(ctx, ds)
if err != nil {
	log.Fatalf("c-store failed: %v", err) // typed error if no accepted context matched the SOP Class
}
if !status.IsSuccess() && !status.IsWarning() {
	log.Fatalf("peer refused storage: %s", status)
}
```

### C-FIND (study-level query) with the streaming iterator

```go
query := dicom.NewDataSet()
query.SetString(dicom.TagPatientID, "12345")
query.SetEmpty(dicom.TagStudyInstanceUID)   // request these keys back
query.SetEmpty(dicom.TagStudyDescription)

assoc, err := ae.Associate(ctx, "pacs.example.org:104", called, dimse.QueryRetrieveContexts())
if err != nil {
	log.Fatal(err)
}
defer assoc.Release(ctx)

for status, match := range assoc.Find(ctx, query, dimse.QueryLevelStudy) {
	switch {
	case status.IsPending():
		uid, _ := match.GetString(dicom.TagStudyInstanceUID)
		fmt.Println("match:", uid)
	case status.IsSuccess():
		fmt.Println("query complete")
	case status.IsFailure():
		log.Printf("query failed: %s", status)
	}
}
if err := assoc.LastError(); err != nil {
	log.Fatalf("c-find transport error: %v", err)
}
```

Cancelling `ctx` (or `break`ing out of the loop) mid-iteration sends a C-CANCEL for the operation.

### Serving a Storage SCP

Handlers return named status constants, never `Status` struct literals — a literal with a `Category` field would let a
handler author a status that contradicts its code, which the typed model exists to prevent (PRD §8.1).

```go
type fileStore struct{ dir string }

func (f *fileStore) Echo(ctx context.Context, info dimse.OpInfo) dimse.Status {
	return dimse.StatusEchoSuccess
}

func (f *fileStore) Store(ctx context.Context, ds *dicom.DataSet, info dimse.OpInfo) dimse.Status {
	// Write a Part 10 file using the transfer syntax the context was negotiated with.
	path := filepath.Join(f.dir, info.SOPClassUID.String()+".dcm")
	if err := ds.WriteFile(path, info.TransferSyntax); err != nil {
		return dimse.StatusStoreCannotUnderstand
	}
	return dimse.StatusStoreSuccess
}

func main() {
	title, err := dimse.ParseAETitle("RADX-SCP")
	if err != nil {
		log.Fatal(err)
	}
	ae, err := dimse.NewAE(title)
	if err != nil {
		log.Fatal(err)
	}
	srv := dimse.NewServer(ae, dimse.AllStorageContexts(), &fileStore{dir: "/var/dicom"})
	ctx := context.Background()
	// Default bind is loopback; pass an explicit address to bind elsewhere.
	if err := srv.ListenAndServe(ctx, "127.0.0.1:11112"); err != nil {
		log.Fatal(err)
	}
}
```

## Conformance scope and limits

The authoritative, versioned scope lives in the DIMSE Conformance Statement (`docs/conformance/`); this section states
the v1 boundary so the API contract is self-contained.

What v1 conforms to:

- **DIMSE-C**: C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE — both SCU and SCP — with C-CANCEL and the streaming
  multi-response contract.
- **DIMSE-N (SCU only)**: MPPS (N-CREATE / N-SET) and Storage Commitment Push Model (N-ACTION / N-EVENT-REPORT).
- **Negotiation**: max PDU length, SCP/SCU role selection, asynchronous-operations window (with genuine concurrency),
  user identity types 1–5, SOP-class extended and common-extended negotiation.
- **DUL**: PS3.8 Table 9-10 in full — 13 states (including Sta9–Sta12), 19 events, 28 actions.
- **Transfer syntaxes**: the four defaults negotiated and exercised end-to-end; the full registered set recognised for
  negotiation; every supported compressed syntax **decoded**; **encode/transcode** only where a codec exists (RLE and
  JPEG 2000 lossless first), behind the optional CGo build tag and off by default (PRD §7.3, conformance matrix).
- **Transport**: plain TCP and TLS 1.2+ with peer verification and optional mutual TLS.
- **Interoperability**: verified in CI against Orthanc and dcm4chee-arc (PRD §11.1).

Explicit limits (deferred, architected-for — PRD §3.2, §5.1):

- No SCP / server side of MPPS or Storage Commitment.
- No standalone N-GET / N-DELETE service, no Print Management, no Unified Procedure Step (UPS).
- No private SOP-class business logic; private abstract syntaxes negotiate generically.
- Compressed transfer-syntax *codec availability* is a `dicom`-package build concern, not a DIMSE guarantee. Decode of
  every supported compressed syntax is in scope; encode/transcode is gated on a compiled-in codec behind the optional
  CGo build tag (off by default). A context may be negotiated whose pixel data cannot be decoded without the optional
  codecs, in which case decode returns a typed "codec unavailable" error.

## See also

- [DICOM data model](dicom.md) — the `DataSet`, `TransferSyntax`, `UID`, and SOP types this package transports.
- [DICOMweb](dicomweb.md) — RESTful counterparts of C-STORE (STOW-RS), C-FIND (QIDO-RS), and C-GET/C-MOVE (WADO-RS).
- `docs/prd/go-radx-prd.md` §6.2 (parity floor), §8.1 (API commitments), §8.2 (design principles), §9 (NFRs).
- `UBIQUITOUS_LANGUAGE.md` — DIMSE section and the cross-standard collision table.
