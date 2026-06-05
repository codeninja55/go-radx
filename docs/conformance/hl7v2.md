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

This statement enumerates *scope* — which message types, segments, datatypes, transport modes, and acknowledgement
behaviours are in or out. **API shape is defined solely in `docs/reference/hl7v2.md`.** The API reference fixes every
type, signature, and field definition; this statement does not restate them. Where this statement mentions a type or
function by name, the authoritative definition is in the reference.

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

> **Implementation status: PARTIAL.** This list is the v1 target. Of it, the parse tree, the typed segments and
> composites, encoding-character derivation and `DTM` precision, Chapter 2 §2.10 escape/unescape, the typed `ORM` view,
> and the `AckCode` enum with its predicates ship today. The typed `ADT`/`OMG`/`ORU`/`ACK` views, batch/file container
> parsing, MLLP transport, and `BuildACK` are **NOT YET SHIPPED**; the per-section banners below mark exactly which
> surface each describes.

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

The versions go-radx interoperates with across the version-tolerant range are `2.3`, `2.3.1`, `2.4`, `2.5`, `2.5.1`,
`2.6`, `2.7`, and `2.8.1`. The typed `Version` value that surfaces the declared `MSH-12` is defined in the API
reference; this statement only fixes the supported set.

## Public API

Per PRD §8.1, typed segments with named Go fields are the **primary** API; callers never index. Per PRD §8.2 and the
glossary naming rules, named types replace bare primitives (`MessageType`, `TriggerEvent`, `AckCode`,
`EncodingCharacters`), and there is exactly one indexing convention.

**The full public API — every type, signature, and field definition — is defined solely in
`docs/reference/hl7v2.md`.** This section records only the scope decisions that the API encodes; it intentionally does
not restate signatures, so the two documents can never drift apart. Where this statement names an identifier, the
reference is authoritative for its shape.

### Indexing convention (one, unambiguous)

The generic tree is Go-native and **0-based**, like every other Go slice. The string-key `Accessor` mirrors the HL7
**1-based** `SEG[n]-Fn-Rn-Cn-Sn` spec convention so a developer reading the HL7 standard can transcribe a path
directly. These two conventions never mix: tree navigation is 0-based Go, accessor keys are 1-based HL7. The reference
library's footgun — a 0-based container hidden under a 1-based callable accessor on the same object — does not exist
here. The `Accessor` segment-instance field is `SegmentNum` (1 selects the first instance); `Message.Get` takes a
1-based path string and returns the unescaped leaf value with an error channel. See the API reference for the exact
signatures.

### Generic tree and typed layers in scope

Two access layers are in scope:

- The generic six-level tree (`Message`, `Segment`, `Field`, `Repetition`, `Component`, subcomponent) navigated with
  0-based Go slices, with `Message.Segment`/`Message.AllSegments` for typed-or-untyped lookup and `Message.Get` for the
  1-based accessor path.
- The typed segment and message views layered over it (the primary API).

`EncodingCharacters` is derived per message from `MSH-1`/`MSH-2`, never hardcoded (PRD §6.2 floor; glossary). The
standard defaults (`|`, `^`, `~`, `\`, `&`) apply only when the sender omits trailing delimiters; a sender using
non-standard delimiters round-trips correctly.

### Typed message types in scope

> **Implementation status: PARTIAL.** Only the typed `ORM` view (with `OrderGroup` and `Orders()`) ships today. The
> `ADT`, `OMG`, `ORU`, and `ACK` typed views and their `As*` constructors described below are NOT YET SHIPPED; until
> they land, those message types parse into the generic six-level tree but have no dedicated typed view.

`MessageType` is the `MSH-9` composite `code^trigger^structure` (e.g. `ORU^R01^ORU_R01`). The trigger event is only
`MSH-9.2` (glossary). The in-scope message views are `ADT`, `ORM`, `OMG`, `ORU`, and `ACK`, each obtained from a parsed
`*Message` via the `As*` view functions in the reference. `ORU.Results()` and `ORM.Orders()` expose the within-message
grouping as iterators of `ResultGroup` (one `OBR` with its following `OBX` rows) and `OrderGroup` (one `ORC` with its
following `OBR` requests) respectively; both group types are defined in the API reference.

### Typed segments in scope

Typed segment structs are the primary read/write surface. Absent optional fields read as the datatype zero value, not
an error (glossary: "absent optionals read as empty, not error"). The typed segments are `MSH`, `EVN`, `PID`, `PV1`,
`ORC`, `OBR`, `OBX`, `MSA`, and `ERR`, plus the batch/file headers and trailers. Field shapes use the typed composite
datatypes, never bare strings. Two scope-load-bearing field placements are fixed here and match the reference:

- `PID.PatientID` is `CX` at **PID-3** — the patient identifier list (first repetition), with `PID.AllPatientIDs` for
  the remaining repetitions. PID-3 is the patient identifier list; the retired external `PID-2` is not modelled.
- `OBX.Value` is `[]string` at OBX-5 (raw repetitions, interpreted per `OBX.ValueType` at OBX-2).

### Composite datatypes in scope

Typed composites are returned by the typed segment accessors. The reference library leaves these stringly-typed;
go-radx models them (PRD §6.3). The in-scope composites are `XPN` (person name: `Family`, `Given`, `Middle`, `Suffix`,
`Prefix`, `Degree`, `NameTypeCode`), `XAD` (address), `CX` (identifier with assigning authority), `CWE` (coded with
exceptions, with `Code`/`Text` plus coding-system and alternate fields; supersedes the retired `CE`), `HD` (hierarchic
designator), and `DTM` (variable-precision timestamp). Exact field lists and component numbers are in the API
reference.

`DTM` **preserves precision** (glossary: "preserve precision, don't zero-fill"). A date-only `19720101` does not
silently become midnight; the precision is retained so it round-trips and so a consumer can tell a true midnight from
an unspecified time. The `Precision` enum and the resolving accessors are in the reference.

### Acknowledgement (ACK) in scope

> **Implementation status: PARTIAL.** The `AckCode` typed enum and its `IsPositive()` / `IsError()` / `IsReject()`
> predicates ship today. `BuildACK` and the typed `ACK` message view are NOT YET SHIPPED; the spec-correct response
> construction described below is the planned design.

There is no "NACK" message in HL7 (glossary). A negative acknowledgement is an `ACK` whose `MSA-1` carries a rejecting
code. go-radx models `AckCode` as a typed enum over HL7 Table 0008, covering both original-mode (`AA`/`AE`/`AR`) and
enhanced-mode (`CA`/`CE`/`CR`) acknowledgement codes, with the predicates `IsPositive()` (AA or CA), `IsError()` (AE or
CE), and `IsReject()` (AR or CR). `BuildACK` constructs the spec-correct `ACK` response per HL7 §2.9.2: it swaps
sender/receiver application and facility, echoes the inbound control ID into `MSA-2`, mints a fresh `MSH-10` for the
`ACK`, and sets `MSA-1` to the chosen code. Signatures are in the API reference.

### MLLP transport in scope

> **Implementation status: NOT YET SHIPPED.** The MLLP `Client`, `Server`, and `Handler` described below are not yet
> implemented; the `hl7v2` package currently has no transport layer. The behaviour in this section is the planned
> design.

MLLP (Minimal Lower Layer Protocol) frames each message as `0x0B` `<message>` `0x1C` `0x0D` over TCP (glossary).
go-radx adds `context` cancellation and a configurable maximum frame length to guard against a hostile or runaway peer
(PRD §9.3, §9.4). Both client and server are in scope (PRD §5.1, §6.2 floor): a blocking `Client` (constructed with
`NewClient`) that sends a message and blocks for the acknowledgement frame, and a `context`-aware `Server` (constructed
with `NewServer`) that invokes a `Handler` once per inbound framed message. The `Handler.Handle(ctx, m)` method decides
the acknowledgement, so a consumer can reject (`AR`) or report an error (`AE`) deliberately rather than have the server
auto-build a reply. Returning a non-nil error from the handler is logged without PHI and closes that connection. The
server binds to loopback unless an explicit non-loopback address is supplied (PRD §9.1), and supports a configurable
maximum frame length, read timeouts, and TLS with peer verification (PRD §9.7). Type and option signatures are in the
API reference.

### Batch and file containers in scope

> **Implementation status: NOT YET SHIPPED.** The `Batch` and `File` container types and the `ParseBatch` / `ParseFile`
> functions described below are not yet implemented. The behaviour in this section is the planned design.

`Batch` (`BHS`/`BTS`) and `File` (`FHS`/`FTS`) are optional bulk containers. The headers and trailers are present
together or not at all — a header without its trailer (or vice versa) is a malformed container, matching the reference
library's "both or neither" rule. "File" here is the HL7 batch-protocol container, not an OS or `.dcm` file (glossary).
`ParseBatch`/`ParseFile` parse them and the containers re-render through `MarshalText`; signatures are in the API
reference.

## Supported message types and trigger events

> **Implementation status: PARTIAL.** Of the message types in the table below, only `ORM` has a shipped typed view
> today. `ADT`, `OMG`, `ORU`, and `ACK` are NOT YET SHIPPED as typed views; their rows describe the planned scope.

The following message types are typed and conformance-tested in v1. Other trigger events of these message types parse
into the generic tree but do not get a dedicated typed view.

| Message type | Trigger events in scope | Typed view | Notes |
|--------------|-------------------------|------------|-------|
| `ADT` | `A01`, `A02`, `A03`, `A04`, `A08` | `ADT` | Feeds `ADTToPatient` / `ADTToEncounter` |
| `ORM` | `O01` general order | `ORM` | Feeds `convert.ORMToServiceRequest`; carries `ORC`+`OBR` groups |
| `OMG` | `O19` general clinical order (imaging) | `OMG` | Radiology order variant; `ORC`+`OBR` groups |
| `ORU` | `R01` unsolicited observation result | `ORU` | Feeds `convert.ORUToDiagnosticReport`; `OBR`+`OBX` groups |
| `ACK` | general acknowledgement | `ACK` | Built by `BuildACK`; `MSA`(+`ERR`) |

The `ORM`/`OMG` split follows the standard's migration of imaging orders toward `OMG^O19`; go-radx accepts both so it
interoperates with senders on either convention. Both `ORM` and `OMG` may carry several `ORC`+`OBR` order groups; the
typed `Orders()` iterator yields each `OrderGroup` in order.

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
rule (PRD §8.2, §9.1). The typed error set (`ParseError`, `AccessorError`, `FramingError`, `LimitExceededError`,
`SegmentError`) is defined in the API reference.

### Truncation is a failure, not a success

A message that ends mid-value — a frame whose bytes stop inside a field rather than at a clean segment boundary —
produces `io.ErrUnexpectedEOF` (wrapped in a `ParseError`). A message that ends cleanly at a segment terminator is a
complete parse. The parser distinguishes a clean record-boundary EOF from a short read (PRD §9.2 rule b). Accepting a
truncated message as complete is a defect, and the regression test for this ships with the parser.

### Generic tree fallthrough

A typed segment accessor never fails because the sender included extra fields; it surfaces only the fields go-radx
models. Unmodelled fields, repetitions beyond the modelled set, and entire segments go-radx does not type are always
reachable through `Message.Segment`, `Message.AllSegments`, and `Message.Get`. A `Get` against an absent optional
field returns `("", nil)` rather than an error, matching the standard's optional-field semantics. A `Get` whose path
runs past a leaf node (asking for a component of a value that has none) returns an `*AccessorError`, because that is a
malformed request, not an absent optional.

### Encoding characters and escaping

> **Implementation status: SHIPPED.** `Escape`, `Unescape`, and the `UnescapeNote` channel ship today; `Encoding`,
> `DefaultEncoding`, and `DeriveEncoding` ship alongside them.

Encoding characters are derived from `MSH-1` and `MSH-2` on every parse; they are never hardcoded, so a sender using
non-standard delimiters round-trips correctly. `Escape` and `Unescape` implement HL7 Chapter 2 §2.10 against the escape
table **derived from the message's encoding characters**, never the static defaults: the field, repetition, component,
subcomponent, and escape separators (`\F\`, `\R\`, `\S\`, `\T\`, `\E\`), hex data (`\Xdd...\`), highlight (`\H\`/`\N\`),
rich-text formatting (`\.br\` and the formatting commands), and application-defined sequences (`\Zxxx\`). Highlight,
formatting, and application-defined sequences carry no character data and decode to the empty string. A malformed
sequence — an unterminated escape, a non-hex `\X\` body, or an empty `\\` — is preserved verbatim so a value is
never silently corrupted. `MSH-1` and `MSH-2` are themselves never unescaped, because they *define* the escape
mechanism.
Inline character-set switching inside escape sequences (`\Cxxyy\`, `\Mxxyyzz\`) is out of scope for v1; `Unescape`
preserves the raw sequence and reports it through the `UnescapeNote` slice it returns rather than silently losing it,
matching the reference library's documented limitation.

### Concurrency

`Parse` and the typed accessors are pure and safe to call concurrently on distinct messages. A `*Message` is not safe
for concurrent mutation. The MLLP `Server` handles each connection on its own goroutine; the `Handler` must be safe for
concurrent invocation. All network operations honour `context` cancellation, and the server shuts down gracefully when
its context is cancelled (PRD §9.4).

## Worked usage examples

The signatures used below are defined in `docs/reference/hl7v2.md`. These examples illustrate scope, not the API shape.

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

    oru, ok := hl7v2.AsORU(msg)
    if !ok {
        log.Fatal("not an ORU^R01 message")
    }

    if pid, ok := oru.PID(); ok {
        fmt.Println("family name:", pid.PatientName.Family) // DOE
        fmt.Println("dob precision:", pid.BirthDate.Precision()) // PrecisionDay
    }
    for result := range oru.Results() {
        for _, obx := range result.Observations {
            fmt.Println("observation:", obx.Value)
        }
    }
}
```

### Receive over MLLP and acknowledge

```go
func serve(ctx context.Context) error {
    // The Handler decides the acknowledgement; BuildACK swaps sender/receiver,
    // echoes the inbound MSH-10 into MSA-2, and mints a fresh ACK control ID.
    srv, err := hl7v2.NewServer("127.0.0.1:2575", hl7v2.HandlerFunc(
        func(ctx context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
            return m.BuildACK(hl7v2.AckAccept)
        }),
    )
    if err != nil {
        return err
    }
    return srv.Serve(ctx) // returns on ctx cancellation
}
```

### Send and inspect the acknowledgement

```go
func send(ctx context.Context, msg *hl7v2.Message) error {
    client, err := hl7v2.NewClient("127.0.0.1:2575")
    if err != nil {
        return err
    }
    defer client.Close()

    ack, err := client.Send(ctx, msg) // blocks for the ACK frame
    if err != nil {
        return err
    }
    typedAck, ok := hl7v2.AsACK(ack)
    if !ok {
        return fmt.Errorf("peer reply was not an ACK")
    }
    msa, ok := typedAck.MSA()
    if !ok {
        return fmt.Errorf("ACK has no MSA segment")
    }
    if msa.AckCode.IsReject() || msa.AckCode.IsError() {
        return fmt.Errorf("peer rejected message: %s %s", msa.AckCode, msa.TextMessage)
    }
    return nil
}
```

## Conformance scope and limits

In scope (v1):

- Six-level lossless parse tree for any well-formed `MSH`-led message; byte-for-byte round-trip via `MarshalText`.
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

- HL7 v2 API reference (the single normative API source): `../reference/hl7v2.md`
- Ubiquitous language (HL7 v2 section): `../../UBIQUITOUS_LANGUAGE.md`
- Product requirements (parity floor §6.2, API commitments §8.1): `../prd/go-radx-prd.md`
- Cross-standard conversions (`ORU`/`ORM`/`ADT` to FHIR): `../reference/convert.md`
