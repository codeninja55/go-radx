# HL7 v2 messaging and MLLP

The `hl7v2` package parses, constructs, and exchanges HL7 v2.x messages. It is a greenfield package in go-radx —
there was no prototype to port — so its public API is designed from scratch against the parity floor in the product
requirements document (`docs/prd/go-radx-prd.md` §6.2, "HL7 v2 floor") and the type-safety thesis in §6.3. The
reference library that fixes the floor is `python-hl7`; go-radx matches its capability surface and then improves on its
two documented footguns: stringly-typed access and a 0-based container hidden under a 1-based accessor.

This page is the contract the implementation conforms to. It defines the public types, their behaviour, the error
model, and the conformance scope. Where a signature here disagrees with the implementation, the implementation is
wrong.

## Overview and scope

HL7 v2 is a pipe-delimited messaging standard for clinical events: admissions (`ADT`), orders (`ORM`/`OMG`), and
results (`ORU`). A message is a sequence of segments, each a delimiter-separated line; the delimiters themselves are
declared in the first two fields of the leading `MSH`/`BHS`/`FHS` segment. The package provides two layers over this:

1. A generic six-level parse tree — message, segment, field, repetition, component, subcomponent — that round-trips
   any conformant message byte-for-byte and gives untyped access to every position.
2. Typed segments (`MSH`, `EVN`, `PID`, `PV1`, `OBR`, `OBX`, `ORC`, `MSA`, `ERR`) and typed message types (`ADT`,
   `ORM`, `ORU`, `ACK`) with named Go fields, backed by typed composite datatypes (`XPN`, `XAD`, `CX`, `CWE`, `HD`,
   `DTM`).
   The typed layer is the **primary** API: callers read `msg.PID().PatientName.Family`, never `seg[5][0][0]`.

The package also covers the batch protocol (`BHS`/`BTS` batches, `FHS`/`FTS` files), escape/unescape per HL7 v2
Chapter 2 §2.10, encoding-character derivation, and a Minimal Lower Layer Protocol (MLLP) client and server with
framing, acknowledgement, context cancellation, and a configurable maximum frame size.

In scope for v1, matching the parity floor:

- The six-level parse tree with byte-exact round-trip.
- Delimiter and encoding-character derivation from `MSH-1`/`MSH-2` with standard defaults.
- Both numeric and prefixed accessor keys with segment-instance indexing.
- Message construction and a spec-correct `BuildACK` (HL7 §2.9.2).
- Escape/unescape: separator, highlight, hex, rich-text, and application-defined sequences (§2.10).
- `DTM` parsing that preserves variable precision.
- Batch (`BHS`/`BTS`) and File (`FHS`/`FTS`) containers, with batch and file parsing.
- MLLP framing with a blocking client and a concurrent client and server.

Out of scope for v1 (architected-for, deferred — consistent with the PRD non-goals):

- Inline character-set switching escapes (`\Cxxyy\`, `\Mxxyyzz\`); these decode to a typed "unsupported escape" notice,
  not silent loss. `python-hl7` also declines these.
- A complete typed model of every segment and field in every HL7 version. The typed segments above are the radiology
  workflow set; any other segment is reachable through the generic tree.
- Message-structure validation against conformance profiles (segment grammar, required-segment checks). The package
  validates syntax and datatype shape, not message-profile conformance.

## The one indexing convention

This is the single most important rule in the package, and the footgun go-radx exists to remove. `python-hl7` stores
its tree in 0-based Python lists but exposes a `__call__` accessor that is 1-based, so `seg[0]` and `seg(1)` return the
same element and mixing the two corrupts data silently.

go-radx splits the two cleanly and never overlaps them:

- **The generic tree is Go-native and 0-based.** `Segment.Fields`, `Field.Repetitions`, and the rest are ordinary Go
  slices. `seg.Fields[0]` is the segment name (`"PID"`), `seg.Fields[5]` is field five. You index them with normal Go
  semantics, because they are normal Go slices.
- **The string-key `Accessor` is 1-based and mirrors the HL7 spec.** When you write `msg.Get("PID-5-1-2")` you are using
  the HL7 numbering exactly as it appears in the standard and in segment documentation: field 5, repetition 1,
  component 2. The accessor translates spec-numbering to slice-numbering internally; you never see the 0-based index.

So: indices into Go slices are 0-based; string accessor keys are 1-based HL7 spec numbering. There is no third
convention and no overlap between them. Typed segment fields (the primary API) sidestep indexing entirely.

## Public API

### Parsing entry points

```go
// ParseOption configures a parse: maximum sizes, an application-defined escape
// map, and strictness toggles. Pass none for the standard behaviour.
type ParseOption func(*parseConfig)

// Parse decodes a single HL7 v2 message from b. b may use \r, \n, or \r\n
// segment terminators; the canonical form uses \r. Encoding characters are
// derived from MSH-1/MSH-2 (see EncodingCharacters).
func Parse(b []byte, opts ...ParseOption) (*Message, error)

// ParseBatch decodes a BHS/BTS batch (or a bare sequence of messages with no
// BHS/BTS) into a Batch.
func ParseBatch(b []byte, opts ...ParseOption) (*Batch, error)

// ParseFile decodes an FHS/FTS file (or a bare batch) into a File.
func ParseFile(b []byte, opts ...ParseOption) (*File, error)

// ParseAny dispatches on the leading segment: MSH -> Message, BHS or a multi-
// message body -> Batch, FHS -> File. It returns one of *Message, *Batch, or
// *File in the Container interface.
func ParseAny(b []byte, opts ...ParseOption) (Container, error)
```

`Parse` is strict about structure but lenient about line endings, matching how real interfaces emit messages. A body
whose first segment is not `MSH`, `BHS`, or `FHS` is a `*ParseError`. Encoding characters come from the message itself,
never from a global default, so a sender using non-standard delimiters round-trips correctly.

### The generic tree

The six levels are concrete structs, not an untyped recursive `[]any`. Each carries the `EncodingCharacters` in force
so that `String`/`MarshalText` re-render with the original delimiters and `Get`/`Set` can escape and unescape
correctly.

```go
// Container is implemented by Message, Batch, and File so ParseAny can return
// any of them and callers can render any of them.
type Container interface {
    encoding.TextMarshaler
    Encoding() EncodingCharacters
}

// Message is the root of the six-level tree: an ordered list of Segments, the
// first of which is MSH.
type Message struct {
    Segments []Segment
    Enc      EncodingCharacters
}

type Segment struct {
    Fields []Field // Fields[0] is the segment ID, e.g. "PID"
}

type Field struct {
    Repetitions []Repetition
}

type Repetition struct {
    Components []Component
}

type Component struct {
    Subcomponents []string // leaf level; values are unescaped on access via Get
}
```

`Segment.ID() string` returns `Fields[0]` rendered. `Message.Segment(id string) (Segment, bool)` returns the first
segment with the given three-character ID; `Message.AllSegments(id string) []Segment` returns every instance. These
report absence as `false` / an empty slice, never an error, because an absent optional segment is normal.

### Accessor — the 1-based string path

```go
// Accessor is a parsed SEG[n]-Fn-Rn-Cn-Sn path using 1-based HL7 spec numbering.
// SegmentNum selects which instance of a repeated segment (1 = first).
type Accessor struct {
    Segment      string // three-character segment ID, e.g. "PID"
    SegmentNum   int    // 1-based segment instance; 1 if omitted
    Field        int    // 1-based HL7 field number; 0 if not in the path
    Repetition   int
    Component    int
    Subcomponent int
}

// ParseAccessor parses both styles: "PID-5-1-2" / "PID.5.1.2" (numeric) and
// "PID.F5.R1.C2" (prefixed). A trailing segment-instance index is written
// "PID2-5" to select the second PID.
func ParseAccessor(key string) (Accessor, error)

func (a Accessor) String() string // canonical numeric form, e.g. "PID-5-1-2"

// Get resolves an accessor key against the message and returns the unescaped
// leaf value. An absent optional position returns "" with no error; a path that
// runs past a leaf returns *AccessorError.
func (m *Message) Get(key string) (string, error)

// Set assigns value at the accessor key, growing the tree as needed, and
// escapes value using the message's EncodingCharacters. The target segment must
// exist (Set never invents a segment).
func (m *Message) Set(key, value string) error
```

`Get` follows the HL7 "future-proofed" resolution rules: if the tree is deeper than the path, it descends the first
child to a leaf; if the path is deeper than the tree but every extra step is index 1, it returns the leaf reached.
This matches `python-hl7`'s `extract_field` so vendored fixtures round-trip.

`MSH-1` and `MSH-2` are returned verbatim (not unescaped), because they *are* the delimiters; unescaping them would
be circular.

### Encoding characters

```go
// EncodingCharacters carries the five HL7 delimiters in force for a message.
// Field is MSH-1; the remaining four are the characters of MSH-2 in the spec
// order component/repetition/escape/subcomponent.
type EncodingCharacters struct {
    Field        byte // MSH-1, default '|'
    Component    byte // MSH-2[0], default '^'
    Repetition   byte // MSH-2[1], default '~'
    Escape       byte // MSH-2[2], default '\'
    Subcomponent byte // MSH-2[3], default '&'
}

// DefaultEncoding returns the standard delimiters: | ^ ~ \ &
func DefaultEncoding() EncodingCharacters

// DeriveEncoding reads the delimiters from a raw message header. It reads
// MSH-1 as the byte at offset 3 and MSH-2 as the bytes up to the next MSH-1.
// Missing trailing characters fall back to the standard defaults (a header of
// "MSH|^~\&" is complete; "MSH|^" fills repetition/escape/subcomponent from
// the defaults). The segment terminator is always '\r'.
func DeriveEncoding(header []byte) (EncodingCharacters, error)
```

Hardcoding `| ^ ~ \ &` is a defect for non-standard senders, so every render and every `Set`/`Get` consults the
`EncodingCharacters` carried by the tree. Derivation tolerates a short `MSH-2` exactly as `python-hl7` does: it fills
missing positions from the defaults rather than erroring, because many real senders omit the rarely-used subcomponent
and repetition characters.

### Escape and unescape (Chapter 2 §2.10)

```go
// Escape encodes value for embedding in a field, replacing delimiter, escape,
// and non-printable bytes with their HL7 escape sequences. appMap supplies
// application-defined sequences (§2.10.7); pass nil for none.
func Escape(value string, enc EncodingCharacters, appMap map[byte]string) string

// Unescape decodes HL7 escape sequences in value back to their literal bytes.
// It handles separator (\F\ \R\ \S\ \T\ \E\), highlight (\H\ \N\), hex
// (\Xdd..\), and rich-text (\.br\ \.sp\ \.in\ ...) sequences, plus any
// application-defined sequences in appMap. Inline character-set switches
// (\Cxxyy\, \Mxxyyzz\) are reported via the returned UnescapeNotes, not
// silently dropped.
func Unescape(value string, enc EncodingCharacters, appMap map[string]string) (string, UnescapeNotes)
```

The escape table is derived from the in-force `EncodingCharacters`, not assumed: `\F\` maps to the actual field
separator byte, `\S\` to the actual component separator, and so on. This is why `Escape`/`Unescape` take the encoding
rather than reading a package global. The §2.10 sequence set covered is the floor set — separator, highlight, hex,
rich-text, and application-defined — matching `python-hl7`; the two inline-charset escapes are surfaced as
`UnescapeNotes` so a caller can decide policy rather than losing data.

### Composite datatypes

Composite datatypes are typed structs with named fields. `python-hl7` leaves these as positional strings, which is
exactly the misuse go-radx removes. Each parses from a `Repetition`/`Field` and renders back with the in-force
encoding.

```go
// XPN — extended person name (PID-5, OBR-32, ...).
type XPN struct {
    Family     string // XPN-1
    Given      string // XPN-2
    Middle     string // XPN-3 (second/further given names)
    Suffix     string // XPN-4
    Prefix     string // XPN-5
    Degree     string // XPN-6
    NameTypeCode string // XPN-7, e.g. "L" legal
}

// XAD — extended address (PID-11, ...).
type XAD struct {
    Street     string // XAD-1
    OtherDesignation string // XAD-2
    City       string // XAD-3
    State      string // XAD-4
    Zip        string // XAD-5
    Country    string // XAD-6
}

// CX — extended composite ID with check digit (PID-3, identifier + authority).
type CX struct {
    ID                 string // CX-1
    CheckDigit         string // CX-2
    AssigningAuthority HD     // CX-4
    IdentifierTypeCode string // CX-5, e.g. "MR" medical record
}

// CWE — coded with exceptions (supersedes the retired CE).
type CWE struct {
    Code            string // CWE-1
    Text            string // CWE-2
    CodingSystem    string // CWE-3
    AltCode         string // CWE-4
    AltText         string // CWE-5
    AltCodingSystem string // CWE-6
}

// HD — hierarchic designator (namespace + universal ID).
type HD struct {
    NamespaceID     string // HD-1
    UniversalID     string // HD-2
    UniversalIDType string // HD-3, e.g. "ISO"
}

// DTM — variable-precision HL7 timestamp. Precision is preserved exactly: a
// value of "2026" is year precision, "202605" is month precision. Parsing does
// not zero-fill, and rendering re-emits at the original precision.
type DTM struct {
    // unexported fields preserve the source lexical form and precision
}

func ParseDTM(s string) (DTM, error)
func (d DTM) String() string             // re-emits at the original precision
func (d DTM) Time() (time.Time, Precision, bool) // resolved time + how precise
func (d DTM) Precision() Precision
```

`DTM` mirrors go-radx's lexical-preserving philosophy (the same reason `dicom`/`fhir` share a `Decimal` type): a
timestamp that arrives as month precision must not silently become midnight on the first of the month. The `Precision`
enum (`PrecisionYear`, `PrecisionMonth`, `PrecisionDay`, `PrecisionMinute`, `PrecisionSecond`, `PrecisionFraction`)
tells the caller how much of the returned `time.Time` is real.

### Typed segments

Typed segments are the primary read API. Each is a view derived from a generic `Segment`; the conversion validates the
segment ID and parses the composite datatypes.

```go
// MSH — message header. MSH-1 and MSH-2 are the encoding characters and are
// exposed through Encoding(), not as string fields.
type MSH struct {
    SendingApplication   HD          // MSH-3
    SendingFacility      HD          // MSH-4
    ReceivingApplication HD          // MSH-5
    ReceivingFacility    HD          // MSH-6
    DateTime             DTM         // MSH-7
    MessageType          MessageType // MSH-9
    ControlID            string      // MSH-10 (locally unique, not a UID)
    ProcessingID         string      // MSH-11, e.g. "P" production
    VersionID            string      // MSH-12, e.g. "2.5"
}

type PID struct {
    SetID         string // PID-1
    PatientID     CX     // PID-3 (first repetition; AllPatientIDs for the rest)
    AllPatientIDs []CX   // PID-3 full repetition list
    PatientName   XPN    // PID-5
    BirthDate     DTM    // PID-7
    Sex           string // PID-8
    Address       XAD    // PID-11
}

type PV1 struct {
    SetID          string // PV1-1
    PatientClass   string // PV1-2, e.g. "I" inpatient, "O" outpatient
    AssignedLocation string // PV1-3 (PL, rendered)
    AttendingDoctor  XPN    // PV1-7
    VisitNumber      CX     // PV1-19
}

type OBR struct {
    SetID              string // OBR-1
    PlacerOrderNumber  string // OBR-2
    FillerOrderNumber  string // OBR-3
    UniversalServiceID CWE    // OBR-4 (the ordered procedure)
    ObservationDateTime DTM   // OBR-7
}

type OBX struct {
    SetID         string   // OBX-1
    ValueType     string   // OBX-2, e.g. "NM", "ST", "CWE", "SN"
    ObservationID CWE      // OBX-3 (what was observed)
    Value         []string // OBX-5 (raw repetitions; interpret per ValueType)
    Units         CWE      // OBX-6
    ReferenceRange string  // OBX-7
    AbnormalFlags []string // OBX-8
    ResultStatus  string   // OBX-11, e.g. "F" final
}

type ORC struct {
    OrderControl      string // ORC-1, e.g. "NW" new, "OK" accepted
    PlacerOrderNumber string // ORC-2
    FillerOrderNumber string // ORC-3
    OrderStatus       string // ORC-5
}

type MSA struct {
    AckCode   AckCode // MSA-1
    ControlID string  // MSA-2 (the control ID of the message being acked)
    TextMessage string // MSA-3
}

type ERR struct {
    // ERR-2 location and ERR-3/ERR-4 codes; rendered for diagnostics
    Location string
    Code     CWE
    Severity string // ERR-4, e.g. "E" error, "W" warning
}

// EVN — event type. Carried by ADT messages and read by convert.ADTToEncounter.
type EVN struct {
    EventTypeCode string // EVN-1, e.g. "A01" (deprecated mirror of MSH-9.2)
    RecordedDateTime DTM // EVN-2
    EventReasonCode  string // EVN-4
}
```

Every typed segment offers a parse-from-generic constructor and an into-generic renderer so the typed and generic
layers compose:

```go
func ParsePID(s Segment) (PID, error)        // validates s.ID() == "PID"
func (p PID) Segment(enc EncodingCharacters) Segment // builds a generic Segment
```

Accessing a typed segment from a message returns the value plus a presence flag, never an error for a simply-absent
segment:

```go
func (m *Message) MSH() (MSH, bool)
func (m *Message) EVN() (EVN, bool)
func (m *Message) PID() (PID, bool)
func (m *Message) PV1() (PV1, bool)
func (m *Message) AllOBX() []OBX // every OBX in order
```

### Message types

The message types layer named accessors over a `*Message`. They do not copy the tree; they are typed lenses, so a
`*Message` parsed once can be viewed as the appropriate type.

```go
// MessageType is the MSH-9 composite: code ^ trigger event ^ structure.
type MessageType struct {
    Code         string // MSH-9.1, e.g. "ORU"
    TriggerEvent string // MSH-9.2, e.g. "R01"
    Structure    string // MSH-9.3, e.g. "ORU_R01"
}

// ResultGroup is one OBR with the OBX rows that follow it, the grouping an ORU
// message expresses through segment order. ORU.Results yields these in order.
type ResultGroup struct {
    Order        OBR
    Observations []OBX
}

// OrderGroup is one ORC with the OBR requests that follow it, the grouping an
// ORM/OMG message expresses through segment order. ORM.Orders yields these.
type OrderGroup struct {
    Common   ORC
    Requests []OBR
}

// ADT — admission/discharge/transfer. Trigger event in MSH-9.2 (A01, A04, ...).
type ADT struct{ *Message }
func (a ADT) Event() string         // MSH-9.2
func (a ADT) EVN() (EVN, bool)
func (a ADT) PID() (PID, bool)
func (a ADT) PV1() (PV1, bool)

// ORM — order message (ORC + OBR groups).
type ORM struct{ *Message }
func (o ORM) Orders() iter.Seq[OrderGroup] // each ORC with its following OBR(s)

// ORU — observation result. Iterates result groups (OBR with its OBX rows).
type ORU struct{ *Message }
func (o ORU) PID() (PID, bool)
func (o ORU) Results() iter.Seq[ResultGroup] // each OBR with its OBX rows

// ACK — acknowledgement (MSH + MSA, optional ERR).
type ACK struct{ *Message }
func (a ACK) MSA() (MSA, bool)
func (a ACK) Errors() []ERR

// AsADT / AsORU / ... verify MSH-9 and return the typed view; the bool is false
// if MSH-9.1 does not match.
func AsADT(m *Message) (ADT, bool)
func AsORM(m *Message) (ORM, bool)
func AsORU(m *Message) (ORU, bool)
func AsACK(m *Message) (ACK, bool)
```

The result and order groups expose the within-message grouping that the flat segment list hides: an `ORU` carries one
or more `OBR` segments, each followed by its `OBX` rows, and `Results()` yields them already grouped. This is the
Go 1.23+ iterator shape committed for streaming responses elsewhere in go-radx (PRD §8.1).

### Construction and acknowledgement

```go
// NewMessage starts an empty message with the given encoding characters and an
// MSH whose MSH-1/MSH-2 reflect that encoding.
func NewMessage(enc EncodingCharacters) *Message

// SetMSH replaces the MSH from a typed MSH value.
func (m *Message) SetMSH(h MSH)

// AppendSegment appends a generic segment (built from a typed segment via its
// Segment method, or constructed directly).
func (m *Message) AppendSegment(s Segment)

// BuildACK constructs a spec-correct acknowledgement for m per HL7 §2.9.2:
// sender/receiver applications and facilities are swapped from the source MSH,
// MSH-9 becomes ACK^<trigger>^ACK, a fresh control ID is generated, and MSA-2
// echoes the source MSH-10. opts override the defaults (control ID, sending
// application/facility, text message).
func (m *Message) BuildACK(code AckCode, opts ...ACKOption) (*Message, error)
```

`BuildACK` follows the same field-swap logic `python-hl7`'s `create_ack` uses — the part of the floor most easily got
wrong — but returns a typed error rather than raising if the source has no `MSH`.

### Acknowledgement codes

There is no NACK message in HL7 v2. A negative acknowledgement is an `ACK` whose `MSA-1` carries a rejecting code. The
codes are HL7 Table 0008, modelled as a closed enum with a validating parser, exactly the required-binding enum
pattern committed for FHIR (PRD §8.1).

```go
type AckCode string

const (
    AckAccept AckCode = "AA" // Application Accept
    AckError  AckCode = "AE" // Application Error
    AckReject AckCode = "AR" // Application Reject
    // Enhanced acknowledgement mode:
    AckCommitAccept AckCode = "CA"
    AckCommitError  AckCode = "CE"
    AckCommitReject AckCode = "CR"
)

func ParseAckCode(s string) (AckCode, error) // unknown code -> *ParseError

func (c AckCode) IsPositive() bool // AA or CA
func (c AckCode) IsError() bool    // AE or CE
func (c AckCode) IsReject() bool   // AR or CR
```

### MLLP transport

MLLP frames each message between a start byte and an end sequence over a stream: `0x0B <message> 0x1C 0x0D`. The
package provides a blocking client, a concurrent client, and a server, all `context.Context`-aware and all enforcing a
configurable maximum frame size. The framing primitives are exported so callers can wrap their own transport.

```go
// MLLP framing constants.
const (
    StartBlock   = 0x0B // <VT>
    EndBlock     = 0x1C // <FS>
    CarriageReturn = 0x0D // <CR>
)

// DefaultMaxFrameSize bounds a single MLLP frame to defend against a peer that
// never sends the end sequence. Callers raise it for large batches.
const DefaultMaxFrameSize = 16 << 20 // 16 MiB

// WriteFrame wraps payload in an MLLP frame and writes it to w.
func WriteFrame(w io.Writer, payload []byte) error

// ReadFrame reads one MLLP frame from r, returning the unwrapped payload. It
// stops at the end sequence or maxFrame bytes, whichever comes first. A stream
// that ends mid-frame returns io.ErrUnexpectedEOF; a frame that does not begin
// with StartBlock returns *FramingError; exceeding maxFrame returns
// *LimitExceededError.
func ReadFrame(ctx context.Context, r io.Reader, maxFrame int) ([]byte, error)

// Client is a blocking MLLP client. Send transmits one message and blocks for
// the acknowledgement frame.
type Client struct {
    // configured via NewClient options
}

func NewClient(addr string, opts ...ClientOption) (*Client, error)
func (c *Client) Send(ctx context.Context, m *Message) (*Message, error) // returns the ACK
func (c *Client) SendRaw(ctx context.Context, frame []byte) ([]byte, error)
func (c *Client) Close() error

// Handler processes one inbound message and returns the reply (typically an
// ACK). Returning an error closes the connection; return a built ACK with a
// rejecting AckCode to NAK at the application level instead.
type Handler interface {
    Handle(ctx context.Context, m *Message) (*Message, error)
}

type HandlerFunc func(ctx context.Context, m *Message) (*Message, error)

// Server serves MLLP on a listener, one goroutine per connection, framing each
// message and writing the Handler's reply. It binds to loopback unless an
// explicit non-loopback address is given (PRD §9.1).
type Server struct {
    Handler     Handler
    MaxFrameSize int // 0 means DefaultMaxFrameSize
    // TLS, timeouts, and logger configured via options
}

func NewServer(addr string, h Handler, opts ...ServerOption) (*Server, error)
func (s *Server) Serve(ctx context.Context) error // returns when ctx is done
func (s *Server) Shutdown(ctx context.Context) error
```

Each connection runs in its own goroutine; the server reads a frame, decodes the message, calls the handler, and
writes the reply frame. Cancelling the `Serve` context stops accepting and drains in-flight connections within the
`Shutdown` deadline. There is no global state and no fire-and-forget goroutine, per the concurrency NFR (PRD §9.4).

## Behaviour and error model

### Errors as values, typed and human-readable

Every failure is a typed error value, never a panic and never a logged warning that the caller cannot observe — the
warning/exception ambiguity in `python-hl7` is one of the documented pains this package removes. The error types are:

```go
type ParseError struct { Offset int; Reason string }       // malformed input at a byte offset
type AccessorError struct { Key string; Reason string }    // bad accessor key or path past a leaf
type FramingError struct { Reason string }                 // MLLP frame violated the start/end protocol
type LimitExceededError struct { Limit int; Kind string }  // a configured cap was exceeded
type SegmentError struct { Segment string; Reason string } // typed-segment parse failed (wrong ID, bad composite)
```

All implement `error`; callers use `errors.As` to discriminate. Diagnostics name concepts, not raw bytes: a parse
error reports the segment and field position and the offending delimiter by name, not a hex dump, and never includes a
field value (which could be PHI) at default verbosity — only structure and identifiers (PRD §8.2, §9.1).

### Truncation and limits are failures

Honest failure reporting (PRD §9.2) is mandatory here because HL7 arrives over the network:

- A message body that ends inside a value — for example an MLLP stream that closes after the start block but before
  the end sequence — returns `io.ErrUnexpectedEOF`. A clean end at a frame boundary is a normal end of stream. The two
  are never conflated.
- A frame that exceeds `maxFrame` returns `*LimitExceededError`; the reader does not allocate the whole frame first and
  then check. This defends against a peer that opens a frame and never closes it.
- A `MalformedBatch`/`MalformedFile` condition — a second `BHS` inside a batch, a trailer without a header — is a
  `*ParseError` with the specific reason, matching `python-hl7`'s `MalformedBatchException`/`MalformedFileException`
  boundaries.

### Absence is not an error

An absent optional segment, field, repetition, component, or subcomponent reads as the empty value with a `false`
presence flag, never an error. This is HL7's own rule (an absent optional value is legal) and it keeps the typed
accessors ergonomic: `if pid, ok := msg.PID(); ok { ... }` rather than error handling for routine optionality. The HL7
null value `""` (a present-but-explicitly-empty field) is distinguished from absence: `Get` returns the literal `""`
quote pair when the sender encoded an explicit null, and the empty string when the position is simply absent.

### Round-trip fidelity

`Parse` followed by `MarshalText` reproduces the input byte-for-byte for any conformant message, including the original
delimiters, repetition structure, and trailing segment terminator. This is a tested invariant against the vendored
`python-hl7` corpus (PRD §11.1). Construction (`NewMessage` + `AppendSegment`) produces canonical output: `\r`
terminators and the configured `EncodingCharacters`.

## Worked examples

### Parse a result message and read it typed

```go
package main

import (
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/hl7v2"
)

func main() {
    raw := []byte(
        "MSH|^~\\&|GHH LAB|ELAB-3|GHH OE|BLDG4|200202150930||ORU^R01|CNTRL-3456|P|2.4\r" +
            "PID|||555-44-4444||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
            "OBR|1|845439^GHH OE|1045813^GHH LAB|1554-5^GLUCOSE\r" +
            "OBX|1|SN|1554-5^GLUCOSE^POST 12H CFST||^182|mg/dl|70_105|H|||F\r",
    )

    msg, err := hl7v2.Parse(raw)
    if err != nil {
        log.Fatalf("parse: %v", err)
    }

    oru, ok := hl7v2.AsORU(msg)
    if !ok {
        log.Fatal("not an ORU^R01 message")
    }

    if pid, ok := oru.PID(); ok {
        fmt.Printf("patient: %s, %s (DOB %s)\n",
            pid.PatientName.Family, pid.PatientName.Given, pid.BirthDate)
    }

    for result := range oru.Results() {
        for _, obx := range result.Observations {
            fmt.Printf("  %s = %v %s [%s] status=%s\n",
                obx.ObservationID.Text, obx.Value, obx.Units.Code,
                obx.AbnormalFlags, obx.ResultStatus)
        }
    }
}
```

### Use the generic tree and the 1-based accessor

```go
// The typed API covers the common case; the generic tree reaches anything.
msg, _ := hl7v2.Parse(raw)

// 1-based HL7 spec numbering through the accessor:
family, _ := msg.Get("PID-5-1-1") // patient family name
fmt.Println(family)               // EVERYWOMAN

// 0-based Go slices through the tree (note: never mix the two numberings):
pid, _ := msg.Segment("PID")
fmt.Println(pid.Fields[5].Repetitions[0].Components[0].Subcomponents[0]) // EVERYWOMAN
```

### Build and reply with an acknowledgement

```go
msg, err := hl7v2.Parse(raw)
if err != nil {
    return err
}

// Positive acknowledgement (AA): sender/receiver fields are swapped from the
// source MSH, MSH-9 becomes ACK, MSA-2 echoes the source control ID.
ack, err := msg.BuildACK(hl7v2.AckAccept)
if err != nil {
    return err
}
out, _ := ack.MarshalText()
fmt.Printf("%s", out)

// Negative acknowledgement is an ACK with a rejecting code, not a separate
// message type:
nak, _ := msg.BuildACK(hl7v2.AckError,
    hl7v2.WithACKText("OBX-5 failed datatype validation"))
_ = nak
```

### Run an MLLP server and a client

```go
// Server: acknowledge every well-formed message, NAK on handler error.
srv, err := hl7v2.NewServer("127.0.0.1:2575", hl7v2.HandlerFunc(
    func(ctx context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
        // ... persist or forward m ...
        return m.BuildACK(hl7v2.AckAccept)
    }),
)
if err != nil {
    log.Fatal(err)
}
go func() { _ = srv.Serve(ctx) }() // returns when ctx is cancelled

// Client: send one message and inspect the acknowledgement.
client, err := hl7v2.NewClient("127.0.0.1:2575")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

ack, err := client.Send(ctx, msg) // blocks for the ACK frame
if err != nil {
    log.Fatalf("send: %v", err)
}
typedAck, _ := hl7v2.AsACK(ack)
if msa, ok := typedAck.MSA(); ok && !msa.AckCode.IsPositive() {
    log.Fatalf("rejected: %s (%s)", msa.AckCode, msa.TextMessage)
}
```

### Parse a batch

```go
batch, err := hl7v2.ParseBatch(rawBatch)
if err != nil {
    return err
}
for _, m := range batch.Messages {
    if oru, ok := hl7v2.AsORU(m); ok {
        // ... process each result message ...
        _ = oru
    }
}
```

## Conformance scope and limits

### What v1 conforms to

The package meets the HL7 v2 parity floor (PRD §6.2) and is verified by byte-exact round-trip against the vendored
`python-hl7` corpus and by interop on the MLLP results leg of the workflow (PRD §11.1, §13 milestone M5). Conformance
is to the HL7 v2 base standard Chapter 2 encoding and Chapter 2 §2.10 escaping, not to a specific minor version's full
segment dictionary. This page is the single normative source for the `hl7v2` public API shape — every type, signature,
and field definition. The Conformance Statement (`docs/conformance/hl7v2.md`) is the source of truth for the supported
*scope* (the exact message types, segments, and MLLP modes) and defers all API shape to this page.

Covered:

- Six-level parse tree with byte-exact round-trip and both accessor styles.
- Encoding-character derivation with default-fill for short `MSH-2`.
- Escape/unescape for the §2.10 floor set (separator, highlight, hex, rich-text, application-defined).
- Variable-precision `DTM` parsing without zero-fill.
- Typed segments `MSH`, `EVN`, `PID`, `PV1`, `OBR`, `OBX`, `ORC`, `MSA`, `ERR` and typed messages `ADT`, `ORM`, `ORU`,
  `ACK`.
- Composite datatypes `XPN`, `XAD`, `CX`, `CWE`, `HD`, `DTM`.
- Batch (`BHS`/`BTS`) and File (`FHS`/`FTS`) containers.
- MLLP client and server with framing, acknowledgement, context cancellation, and a maximum frame cap.

### Documented limits

- **Inline character-set switching** (`\Cxxyy\`, `\Mxxyyzz\`) is not decoded; it is surfaced as an `UnescapeNotes`
  entry so the caller is never silently given lossy text. `python-hl7` declines these too; go-radx makes the decline
  observable.
- **Message-profile validation** (required-segment grammar, conformance profiles) is out of scope. The package
  validates encoding syntax and datatype shape; structural conformance to a profile is a separate concern.
- **Segment coverage beyond the typed set** is via the generic tree. Any segment not in the typed list parses, renders,
  and round-trips, but has no named-field accessor in v1.
- **Character encoding.** Bytes are decoded as UTF-8 by default; `MSH-18` (character set) is read and reported, but
  legacy single-byte and multi-byte code-page transcoding beyond UTF-8/ASCII is not performed in v1. This matches the
  floor (`python-hl7` works on decoded strings and leaves transcoding to the caller).
- **No global state.** Delimiters, caps, and timeouts are per-instance; there is no package-level mutable
  configuration to set, consistent with PRD §8.2 and §9.4.

### Security and PHI posture

External input is untrusted. Frame and body size are bounded before allocation; malformed input yields a typed error,
never a panic (PRD §9.3). At default verbosity, diagnostics carry structure and identifiers — segment IDs, field
positions, accessor keys, control IDs — but not field values, because HL7 fields routinely contain PHI (patient names,
identifiers, results). Surfacing values requires opt-in verbosity (PRD §9.1). The MLLP server binds to loopback unless
an explicit non-loopback address is supplied, and supports TLS with peer verification through server options
(PRD §9.7).
