# HL7 v2 conformance statement

| Field | Value |
|-------|-------|
| Statement version | 1.0.0 (tracks go-radx v1) |
| Standard | HL7 v2.x (Health Level Seven version 2) |
| Package | `github.com/codeninja55/go-radx/hl7v2` |
| Base version posture | HL7 v2.5.1 layouts; version-tolerant parser (see [Version posture](#version-posture)) |
| Status | Normative for v1. Growth is a reviewed change to this file. |

This document is the single source of truth for what go-radx supports in HL7 v2. "Conformant" here means 100%
conformant to the explicitly declared subset enumerated below, verified against the `python-hl7` reference corpus and
round-trip fixtures (PRD §6.1, §11.1). It is not a claim to implement every trigger event or segment in the HL7 v2
standard. Anything not listed here is out of scope for v1; consult
[Conformance scope and limits](#conformance-scope-and-limits) before relying on a behaviour.

The companion API reference (`docs/reference/hl7v2.md`) is the full public API contract. This statement enumerates
*scope* — which message types, segments, datatypes, transport modes, and acknowledgement behaviours are in or out —
and fixes the load-bearing signatures that encode that scope.

## Overview and scope

HL7 v2 is the pipe-delimited messaging standard that still carries most order and result traffic between hospital
information systems. go-radx covers the radiology workflow legs that the PRD commits in §5.1: inbound and outbound
orders (`ORM`/`OMG`), results (`ORU`), patient administration events (`ADT`), and the acknowledgement (`ACK`) that
closes every exchange. The design goal (PRD §2.1, §8.2) is to replace the stringly-typed, positional, footgun-laden
ergonomics of the reference library (`python-hl7`) with typed Go segments, named fields, typed table enums, and a single
unambiguous indexing convention — while keeping a generic parse tree underneath for segments and fields go-radx
does not model.

In scope for v1:

- A lossless six-level parse tree (message, segment, field, repetition, component, subcomponent) for any well-formed
  HL7 v2 message, regardless of whether the segment is typed.
- Typed segment structs with named Go fields for the radiology workflow segments (`MSH`, `PID`, `PV1`, `EVN`, `ORC`,
  `OBR`, `OBX`, `MSA`, `ERR`, and the batch/file headers).
- Typed message types for `ADT`, `ORM`/`OMG`, `ORU`, and `ACK`, with their in-scope trigger events.
- Typed composite datatypes (`XPN`, `XAD`, `CX`, `CWE`, `HD`, `DTM`, and the others listed below).
- Encoding-character derivation from `MSH-1`/`MSH-2` with standard defaults, escape/unescape per HL7 Chapter 2 §2.10,
  and variable-precision `DTM` parsing that preserves precision.
- Batch (`BHS`/`BTS`) and File (`FHS`/`FTS`) container parsing and construction.
- MLLP transport: a blocking client and a `context`-aware server, both with a configurable maximum frame length.
- Acknowledgement construction (`BuildACK`) honouring original-mode and enhanced-mode acknowledgement codes.

Out of scope for v1 is listed explicitly in [Conformance scope and limits](#conformance-scope-and-limits). Notably:
HL7 v2 XML encoding, FHIR-based v2 representations, inline character-set switching inside escape sequences, the full
catalogue of trigger events beyond the radiology set, and message-level conformance-profile validation are all out of
scope.

## Version posture

go-radx does not bind to a single HL7 v2 minor version on the wire. HL7 v2 is backward-compatible by design: later
versions add fields and trigger events without renumbering existing ones. The posture is:

- **Parsing is version-tolerant.** The generic tree parses any `MSH`-led message and never rejects a message for
  carrying fields beyond a known version. Typed segment accessors return only the fields go-radx models; trailing
  unmodelled fields remain reachable through the generic tree
  (see [Generic tree fallthrough](#generic-tree-fallthrough)).
- **Typed segment layouts follow HL7 v2.5.1** field positions, because v2.5.1 is the most widely deployed release for
  radiology order/result traffic and is a strict superset (for the in-scope segments) of v2.3 and v2.4. A field that
  did not exist before v2.5.1 reads as empty when the sender used an earlier version, which is indistinguishable from a
  legitimately absent optional field — this is the standard HL7 contract, not a defect.
- **`MSH-12` (Version ID) is parsed and exposed but not enforced.** go-radx reads the declared version and makes it
  available, but does not reject a message whose layout disagrees with `MSH-12`. Senders frequently misdeclare it; the
  parser trusts the bytes, not the declaration.

The `Version` value is surfaced as a typed field so a consumer can branch on it:

```go
type Version string

const (
    V23  Version = "2.3"
    V231 Version = "2.3.1"
    V24  Version = "2.4"
    V25  Version = "2.5"
    V251 Version = "2.5.1"
    V26  Version = "2.6"
    V27  Version = "2.7"
    V281 Version = "2.8.1"
)
```

## Public API

The signatures below are the contract this statement fixes. Per PRD §8.1, typed segments with named Go fields are the
**primary** API; callers never index. Per PRD §8.2 and the glossary naming rules, named types replace bare primitives
(`MessageType`, `TriggerEvent`, `AckCode`, `EncodingCharacters`, `MessageControlID`), and there is exactly one indexing
convention.

### Indexing convention (one, unambiguous)

The generic tree is Go-native and **0-based**, like every other Go slice. The string-key `Accessor` mirrors the HL7
**1-based** `SEG[n]-Fn-Rn-Cn-Sn` spec convention so a developer reading the HL7 standard can transcribe a path
directly. These two conventions never mix: tree navigation is 0-based Go, accessor keys are 1-based HL7. The reference
library's footgun — a 0-based container hidden under a 1-based callable accessor on the same object — does not exist
here.

```go
// Accessor is the 1-based HL7 path SEG[n]-Fn-Rn-Cn-Sn (segment-instance, field, repetition, component, subcomponent).
// Typed segment structs are the primary API; reach for Accessor only for fields go-radx does not model.
type Accessor struct {
    Segment      string // three-character segment ID, e.g. "PID"
    SegmentNum   int    // 1-based segment instance; 1 selects the first PID
    Field        int    // 1-based field; for MSH, Field 1 is the field separator
    Repetition   int    // 1-based repetition
    Component    int    // 1-based component
    Subcomponent int    // 1-based subcomponent
}

func ParseAccessor(key string) (Accessor, error) // "PID.5.1.2", "PID-5-1-2", or "OBX2.5"
func (a Accessor) Key() string
```

### Parsing and the generic tree

```go
// Parse decodes one HL7 v2 message. Encoding characters are derived from MSH-1/MSH-2.
// A truncated message (ending mid-value, not at a segment boundary) is io.ErrUnexpectedEOF, never a silent success.
func Parse(data []byte, opts ...ParseOption) (*Message, error)

// Message is the root of the six-level tree. Typed segment accessors layer over it.
type Message struct{ /* segments, derived EncodingCharacters */ }

func (m *Message) Encoding() EncodingCharacters
func (m *Message) MessageType() (MessageType, bool) // parsed from MSH-9
func (m *Message) ControlID() (MessageControlID, bool)
func (m *Message) Version() (Version, bool)

// Generic navigation: 0-based Go slices for any segment, typed or not.
func (m *Message) Segment(id string) (Segment, bool) // first segment with this ID
func (m *Message) Segments(id string) []Segment      // all segments with this ID
func (m *Message) Get(a Accessor) (string, bool)     // unescaped leaf value at a 1-based path
func (m *Message) Marshal() ([]byte, error)          // re-serialize; round-trips Parse byte-for-byte

type Segment struct{ /* fields */ }
func (s Segment) ID() string
func (s Segment) Field(n int) (Field, bool) // 0-based Go index; field 0 is the segment ID
```

`EncodingCharacters` is derived per message, never hardcoded (PRD §6.2 floor; glossary). The defaults apply only when
`MSH-2` is the canonical `^~\&`:

```go
type EncodingCharacters struct {
    Field        byte // MSH-1, default '|'
    Component    byte // MSH-2[0], default '^'
    Repetition   byte // MSH-2[1], default '~'
    Escape       byte // MSH-2[2], default '\'
    Subcomponent byte // MSH-2[3], default '&'
}

func DefaultEncoding() EncodingCharacters // '|', '^', '~', '\', '&'
```

### Typed message types

`MessageType` is the `MSH-9` composite `code^trigger^structure` (e.g. `ORU^R01^ORU_R01`). The trigger event is only
`MSH-9.2` (glossary).

```go
type MessageType struct {
    Code      string       // MSH-9.1, e.g. "ORU"
    Trigger   TriggerEvent // MSH-9.2, e.g. "R01"
    Structure string       // MSH-9.3, e.g. "ORU_R01" (optional)
}

type TriggerEvent string

// Typed views over a parsed Message. Each returns false if the message is not of that type.
func (m *Message) ADT() (ADT, bool)
func (m *Message) ORM() (ORM, bool)
func (m *Message) OMG() (OMG, bool)
func (m *Message) ORU() (ORU, bool)
func (m *Message) ACK() (ACK, bool)
```

### Typed segments

Typed segment structs are the primary read/write surface. Absent optional fields read as the datatype zero value, not an
error (glossary: "absent optionals read as empty, not error").

```go
func (m *Message) MSH() (MSH, bool)
func (m *Message) PID() (PID, bool)
func (m *Message) PV1() (PV1, bool)
func (m *Message) EVN() (EVN, bool)
func (m *Message) ORC() []ORC // an order may carry several
func (m *Message) OBR() []OBR
func (m *Message) OBX() []OBX
func (m *Message) MSA() (MSA, bool)
func (m *Message) ERR() []ERR
```

Field shapes use the typed composite datatypes, never bare strings, for example:

```go
type PID struct {
    SetID              string // PID-1
    PatientID          CX     // PID-2 (external, retired post-2.3.1 but still seen)
    PatientIdentifiers []CX   // PID-3 patient identifier list
    PatientName        []XPN  // PID-5
    DateOfBirth        DTM    // PID-7
    AdministrativeSex  string // PID-8 (HL7 Table 0001)
    PatientAddress     []XAD  // PID-11
    // ... remaining v2.5.1 fields
}

type OBX struct {
    SetID            string // OBX-1
    ValueType        string // OBX-2 (HL7 Table 0125)
    ObservationID    CWE    // OBX-3
    ObservationSubID string // OBX-4
    ObservationValue []string
    Units            CWE    // OBX-6
    AbnormalFlags    []string
    ResultStatus     string // OBX-11 (HL7 Table 0085)
    // ... remaining v2.5.1 fields
}
```

### Composite datatypes

Typed composites are returned by the typed segment accessors. The reference library leaves these stringly-typed; go-radx
models them (PRD §6.3).

```go
type XPN struct { // person name
    FamilyName, GivenName, MiddleName, Suffix, Prefix, Degree string
    NameTypeCode                                              string // HL7 Table 0200
}
type XAD struct { // address
    StreetAddress, OtherDesignation, City, State, ZipCode, Country, AddressType string
}
type CX struct { // identifier with assigning authority
    ID                 string
    AssigningAuthority HD
    IdentifierTypeCode string // HL7 Table 0203
}
type CWE struct { // coded with exceptions (supersedes CE)
    Identifier, Text, CodingSystem          string
    AltIdentifier, AltText, AltCodingSystem string
}
type HD struct { // hierarchic designator
    NamespaceID, UniversalID, UniversalIDType string
}
```

`DTM` is a variable-precision timestamp that **preserves precision** (glossary: "preserve precision, don't zero-fill").
A date-only `19720101` does not silently become midnight; the precision is retained so it round-trips and so a consumer
can tell a true midnight from an unspecified time.

```go
type DTM struct{ /* preserves YYYY[MM[DD[HH[MM[SS[.S...]]]]]] precision and optional +/-HHMM offset */ }

func ParseDTM(s string) (DTM, error)
func (d DTM) String() string         // re-emits at the original precision
func (d DTM) Time() (time.Time, bool) // ok=false if precision is below a usable instant
func (d DTM) Precision() DTMPrecision  // Year | Month | Day | Hour | Minute | Second | Fraction
```

### Acknowledgement (ACK)

There is no "NACK" message in HL7 (glossary). A negative acknowledgement is an `ACK` whose `MSA-1` carries a rejecting
code. go-radx models `AckCode` as a typed enum over HL7 Table 0008, covering both original-mode (`AA`/`AE`/`AR`) and
enhanced-mode (`CA`/`CE`/`CR`) acknowledgement codes.

```go
type AckCode string

const (
    AckAA AckCode = "AA" // Application Accept (original mode)
    AckAE AckCode = "AE" // Application Error
    AckAR AckCode = "AR" // Application Reject
    AckCA AckCode = "CA" // Commit Accept (enhanced mode)
    AckCE AckCode = "CE" // Commit Error
    AckCR AckCode = "CR" // Commit Reject
)

func (c AckCode) IsAccept() bool // AA or CA
func (c AckCode) IsReject() bool // AR or CR
func (c AckCode) IsError() bool  // AE or CE

// BuildACK constructs the ACK response for an inbound message per HL7 §2.9.2:
// it swaps sender/receiver application and facility, echoes the inbound control ID into MSA-2,
// mints a fresh MSH-10 for the ACK, and sets MSA-1 to the chosen code.
func BuildACK(inbound *Message, code AckCode, opts ...ACKOption) (*Message, error)

type Acknowledgment struct {
    Code        AckCode          // MSA-1
    ControlID   MessageControlID // MSA-2, echoes the acknowledged message's MSH-10
    TextMessage string           // MSA-3 (optional)
    Errors      []ERR            // structured error detail (enhanced mode)
}
func (m *Message) Acknowledgment() (Acknowledgment, bool)
```

### MLLP transport

MLLP (Minimal Lower Layer Protocol) frames each message as `0x0B` `<message>` `0x1C` `0x0D` over TCP (glossary). go-radx
adds `context` cancellation and a configurable maximum frame length to guard against a hostile or runaway peer
(PRD §9.3, §9.4). Both client and server are in scope (PRD §5.1, §6.2 floor).

```go
// Client: blocking send-and-receive-ACK over one connection.
type MLLPClient struct{ /* ... */ }

func DialMLLP(ctx context.Context, addr string, opts ...MLLPOption) (*MLLPClient, error)
func (c *MLLPClient) Send(ctx context.Context, m *Message) (*Message, error) // returns the peer's ACK
func (c *MLLPClient) SendRaw(ctx context.Context, frame []byte) ([]byte, error)
func (c *MLLPClient) Close() error

// Server: one Handler invocation per inbound framed message; the returned Message is framed back as the ACK.
type Handler interface {
    HandleMessage(ctx context.Context, m *Message) (*Message, error)
}

type MLLPServer struct{ /* ... */ }

func NewMLLPServer(h Handler, opts ...MLLPOption) *MLLPServer
func (s *MLLPServer) Serve(ctx context.Context, ln net.Listener) error // returns when ctx is done or ln closes

// Options applying to both client and server.
func WithMaxFrameBytes(n int) MLLPOption  // typed "frame too large" error when exceeded; default 16 MiB
func WithReadTimeout(d time.Duration) MLLPOption
func WithTLS(cfg *tls.Config) MLLPOption  // TLS 1.2+, peer verification on by default (PRD §9.7)
func WithEncoding(name string) MLLPOption // character set for frame bytes; default UTF-8
```

The server binds wherever the caller's `net.Listener` binds; the reference daemon that wires it defaults to loopback
(PRD §9.1). The server does not auto-build an ACK — the `Handler` decides the acknowledgement, so a consumer can
reject (`AR`) or report an error (`AE`) deliberately. A `Handler` returning `(nil, nil)` sends no ACK (some integrations
suppress acknowledgements); returning a non-nil error is logged without PHI and aborts that frame's exchange.

### Batch and file containers

`Batch` (`BHS`/`BTS`) and `File` (`FHS`/`FTS`) are optional bulk containers. The headers and trailers are present
together or not at all — a header without its trailer (or vice versa) is a malformed container, matching the reference
library's "both or neither" rule. "File" here is the HL7 batch-protocol container, not an OS or `.dcm` file (glossary).

```go
type Batch struct {
    Header   *Segment // BHS, or nil for a bare batch
    Messages []*Message
    Trailer  *Segment // BTS, or nil for a bare batch
}
type File struct {
    Header  *Segment // FHS, or nil
    Batches []*Batch
    Trailer *Segment // FTS, or nil
}

func ParseBatch(data []byte, opts ...ParseOption) (*Batch, error)
func ParseFile(data []byte, opts ...ParseOption) (*File, error)
func (b *Batch) Marshal() ([]byte, error)
func (f *File) Marshal() ([]byte, error)
```

## Supported message types and trigger events

The following message types are typed and conformance-tested in v1. Other trigger events of these message types parse
into the generic tree but do not get a dedicated typed view.

| Message type | Trigger events in scope | Typed view | Notes |
|--------------|-------------------------|------------|-------|
| `ADT` | `A01`, `A02`, `A03`, `A04`, `A08` | `ADT` | Feeds `ADTToPatient` / `ADTToEncounter` |
| `ORM` | `O01` general order | `ORM` | Feeds `convert.ORMToServiceRequest`; carries `ORC`+`OBR` |
| `OMG` | `O19` general clinical order (imaging) | `OMG` | Radiology order variant; `ORC`+`OBR` |
| `ORU` | `R01` unsolicited observation result | `ORU` | Feeds `convert.ORUToDiagnosticReport`; carries `OBR`+`OBX` |
| `ACK` | general acknowledgement | `ACK` | Built by `BuildACK`; `MSA`(+`ERR`) |

The `ORM`/`OMG` split follows the standard's migration of imaging orders toward `OMG^O19`; go-radx accepts both so it
interoperates with senders on either convention.

## Supported typed segments

The segments below have typed structs with named v2.5.1 fields. Every other segment is reachable through the generic
tree (`Message.Segment`, `Message.Get`).

| Segment | Purpose | Used by |
|---------|---------|---------|
| `MSH` | Message header (encoding, type, control ID, version) | all messages |
| `EVN` | Event type | `ADT` |
| `PID` | Patient identification | `ADT`, `ORM`, `ORU` |
| `PV1` | Patient visit | `ADT`, `ORU` |
| `ORC` | Common order | `ORM`, `OMG`, `ORU` |
| `OBR` | Observation request | `ORM`, `OMG`, `ORU` |
| `OBX` | Observation result | `ORU` |
| `MSA` | Message acknowledgement | `ACK` |
| `ERR` | Error detail | `ACK` (enhanced mode) |
| `BHS`/`BTS` | Batch header/trailer | `Batch` |
| `FHS`/`FTS` | File header/trailer | `File` |

## Supported composite datatypes

`XPN`, `XAD`, `CX`, `CWE` (supersedes `CE`), `HD`, and `DTM` are typed. Subcomponents beyond the modelled fields remain
reachable through the generic tree. Datatypes not listed (for example `XCN`, `XON`, `EI`, `MSG`, `PT`, `TS`) parse into
the generic tree as raw component lists; they are not given dedicated structs in v1.

## Behaviour and error model

go-radx returns errors as values; it never panics on malformed input (PRD §9.3) and never reports success on failed
work (PRD §9.2). Errors are typed and checkable with `errors.Is`/`errors.As`, and diagnostics name the offending
location — segment ID, field number, byte offset — without emitting field *values*, honouring the no-PHI-by-default
rule (PRD §8.2, §9.1).

### Sentinel and typed errors

```go
var (
    ErrMissingMSH      = errors.New("hl7v2: message does not begin with MSH")
    ErrShortMSH        = errors.New("hl7v2: MSH too short to derive encoding characters")
    ErrFrameTooLarge   = errors.New("hl7v2: MLLP frame exceeds configured maximum")
    ErrInvalidBlock    = errors.New("hl7v2: MLLP block missing start byte 0x0B")
    ErrUnbalancedBatch = errors.New("hl7v2: batch/file header present without matching trailer")
)

// ParseError locates a fault without leaking PHI: it names structure, not values.
type ParseError struct {
    Segment string // e.g. "PID"
    Field   int    // 1-based; 0 if not field-specific
    Offset  int    // byte offset into the input
    Err     error
}
func (e *ParseError) Error() string
func (e *ParseError) Unwrap() error
```

### Truncation is a failure, not a success

A message that ends mid-value — a frame whose bytes stop inside a field rather than at a clean segment boundary —
produces `io.ErrUnexpectedEOF` (wrapped in a `ParseError`). A message that ends cleanly at a segment terminator is a
complete parse. The parser distinguishes a clean record-boundary EOF from a short read (PRD §9.2 rule b). Accepting a
truncated message as complete is a defect, and the regression test for this ships with the parser.

### Generic tree fallthrough

A typed segment accessor never fails because the sender included extra fields; it surfaces only the fields go-radx
models. Unmodelled fields, repetitions beyond the modelled set, and entire segments go-radx does not type are always
reachable through `Message.Segment`, `Message.Segments`, and `Message.Get(Accessor)`. A `Get` against an absent optional
field returns `("", false)` rather than an error, matching the standard's optional-field semantics. A `Get` whose path
runs past a leaf node (asking for a component of a value that has none) returns an error, because that is a malformed
request, not an absent optional.

### Encoding characters and escaping

Encoding characters are derived from `MSH-1` and `MSH-2` on every parse; they are never hardcoded, so a sender using
non-standard delimiters round-trips correctly. Escape and unescape implement HL7 Chapter 2 §2.10: the field,
repetition, component, subcomponent, and escape separators (`\F\`, `\R\`, `\S\`, `\T\`, `\E\`), hex data (`\Xdd...\`),
highlight (`\H\`/`\N\`), rich-text formatting (`\.br\` and the formatting commands), and application-defined sequences.
`MSH-1` and `MSH-2` are themselves never unescaped, because they *define* the escape mechanism. Inline character-set
switching inside escape sequences (ISO-IR code-page changes mid-field) is out of scope for v1, matching the reference
library's documented limitation.

### Concurrency

`Parse` and the typed accessors are pure and safe to call concurrently on distinct messages. A `*Message` is not safe
for concurrent mutation. `MLLPServer.Serve` handles each connection on its own goroutine; the `Handler` must be safe for
concurrent invocation. All network operations honour `context` cancellation, and the server shuts down gracefully when
its context is cancelled (PRD §9.4).

## Worked usage examples

### Parse a result message and read typed fields

```go
package main

import (
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/hl7v2"
)

func main() {
    raw := []byte(
        "MSH|^~\\&|LAB|HOSP|EMR|HOSP|20260531120000||ORU^R01^ORU_R01|MSG00001|P|2.5.1\r" +
            "PID|1||PATID1234^^^HOSP^MR||DOE^JOHN^A||19720101|M\r" +
            "OBR|1||ACC9001|36643-5^CHEST XRAY^LN\r" +
            "OBX|1|TX|36643-5^CHEST XRAY^LN||No acute findings.||||||F\r")

    msg, err := hl7v2.Parse(raw)
    if err != nil {
        log.Fatalf("parse: %v", err)
    }

    oru, ok := msg.ORU()
    if !ok {
        log.Fatal("not an ORU^R01 message")
    }

    if pid, ok := msg.PID(); ok && len(pid.PatientName) > 0 {
        fmt.Println("family name:", pid.PatientName[0].FamilyName) // DOE
        fmt.Println("dob precision:", pid.DateOfBirth.Precision())  // Day
    }
    for _, obx := range msg.OBX() {
        fmt.Println("observation:", obx.ObservationValue)
    }
    _ = oru
}
```

### Receive over MLLP and acknowledge

```go
type echoHandler struct{}

func (echoHandler) HandleMessage(ctx context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
    // BuildACK swaps sender/receiver, echoes the inbound MSH-10 into MSA-2, and mints a fresh ACK control ID.
    return hl7v2.BuildACK(m, hl7v2.AckAA)
}

func serve(ctx context.Context) error {
    ln, err := net.Listen("tcp", "127.0.0.1:2575") // loopback by default
    if err != nil {
        return err
    }
    srv := hl7v2.NewMLLPServer(echoHandler{}, hl7v2.WithMaxFrameBytes(8<<20))
    return srv.Serve(ctx, ln) // returns on ctx cancellation or listener close
}
```

### Send and inspect the acknowledgement

```go
func send(ctx context.Context, msg *hl7v2.Message) error {
    client, err := hl7v2.DialMLLP(ctx, "127.0.0.1:2575")
    if err != nil {
        return err
    }
    defer client.Close()

    ack, err := client.Send(ctx, msg)
    if err != nil {
        return err
    }
    a, ok := ack.Acknowledgment()
    if !ok {
        return fmt.Errorf("peer reply was not an ACK")
    }
    if a.Code.IsReject() || a.Code.IsError() {
        return fmt.Errorf("peer rejected message: %s %s", a.Code, a.TextMessage)
    }
    return nil
}
```

## Conformance scope and limits

In scope (v1):

- Six-level lossless parse tree for any well-formed `MSH`-led message; byte-for-byte round-trip via `Marshal`.
- Typed message types `ADT` (A01/A02/A03/A04/A08), `ORM^O01`, `OMG^O19`, `ORU^R01`, `ACK`.
- Typed segments `MSH`, `EVN`, `PID`, `PV1`, `ORC`, `OBR`, `OBX`, `MSA`, `ERR`, plus `BHS`/`BTS` and `FHS`/`FTS`.
- Typed composites `XPN`, `XAD`, `CX`, `CWE`, `HD`, `DTM`.
- Encoding-character derivation, escape/unescape per Chapter 2 §2.10, variable-precision `DTM`.
- `Batch`/`File` container parsing and construction with the both-or-neither header/trailer rule.
- MLLP client and `context`-aware server with configurable max frame length, read timeout, TLS, and encoding.
- `BuildACK` for original-mode and enhanced-mode acknowledgement codes.

Out of scope (v1) — parses into the generic tree where applicable, but no typed view, validation, or transform:

- Trigger events other than those listed (for example `ADT^A05`/`A11`, `ORM^O02`, scheduling `SIU`, `MDM`, `DFT`,
  `VXU`); they parse generically but get no typed accessor.
- Message-level conformance profile validation (HL7 conformance profiles / message-structure enforcement against a
  declared profile). go-radx validates well-formedness and typed-field shape, not profile conformance.
- HL7 v2 XML encoding and the FHIR-based representations of v2 messages.
- Inline character-set switching inside escape sequences (ISO-IR code-page changes mid-field).
- Z-segments are parsed generically; go-radx ships no typed Z-segment structs.
- Sequence-number protocol, continuation pointers (`DSC`), and original-mode batch acknowledgement orchestration beyond
  single-message `BuildACK`.
- `MSH-12` version *enforcement*: the version is read and exposed but layout is not validated against it.

go-radx provides the messaging capability and the typed safety; the consumer owns interface-specification conformance,
local table extensions, and any site-specific Z-segment semantics. Growth of this scope is a deliberate, reviewed change
to this conformance statement (PRD §6.1).

## See also

- HL7 v2 API reference: `../reference/hl7v2.md`
- Ubiquitous language (HL7 v2 section): `../../UBIQUITOUS_LANGUAGE.md`
- Product requirements (parity floor §6.2, API commitments §8.1): `../prd/go-radx-prd.md`
- Cross-standard conversions (`ORU`/`ORM`/`ADT` to FHIR): `../reference/convert.md`
