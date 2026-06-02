# HL7 v2 depth (M5) implementation plan

> **For agentic workers:** REQUIRED: Use agentic-dev:subagent-driven-development (if subagents available) or
> agentic-dev:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Take the `hl7v2` package from the thin M2 ORM slice (walking-skeleton Increment 10) to the full HL7 v2 parity
floor (PRD §6.2, "HL7 v2 floor") and the scope the conformance statement governs (`docs/conformance/hl7v2.md`). M5 adds
the complete typed-segment set (`EVN`, `PV1`, `OBX`, `MSA`, `ERR` on top of the existing `MSH`/`PID`/`ORC`/`OBR`) and
the missing composites (`XAD`, plus the extra fields of `CWE`/`XPN`/`PID`); the typed message types `ADT` (with its
A01/A02/A03/A04/A08 trigger events), `ORU^R01` (results, grouped `OBR`+`OBX`), and the `ORM`/`OMG` order types beyond
the thin slice; Chapter 2 §2.10 escape/unescape; the full 1-based `Accessor` with prefixed keys, segment-instance
indexing, future-proofed resolution, and `Set`; spec-correct `BuildACK` with the `AckCode` enum (AA/AE/AR plus
enhanced-mode CA/CE/CR); MLLP transport (the `0x0B…0x1C0x0D` framing, a blocking client, and a loopback-default,
goroutine-tracked server); and the batch/file containers (`BHS`/`BTS`, `FHS`/`FTS`). Every increment ends green: unit
tests with `-race`, a clean `golangci-lint`, and the named hostile-input/round-trip regressions passing.

**Architecture:** M5 **extends** the existing `hl7v2` package; it does not rebuild it. The generic six-level parse tree
(`Message`/`Segment`/`Field`/`Repetition`/`Component`/subcomponent), `EncodingCharacters`/`DeriveEncoding`, the MSH
offset-quirk handling, `MarshalText` byte-exact round-trip, and the `iter.Seq[OrderGroup]` shape are already in place
and correct (Inc 10, files `parse.go`/`encoding.go`/`segment.go`/`composite.go`/`typed.go`/`orm.go`/`errors.go`). The
M5 work layers on top in dependency order: composites and typed segments first (pure, no I/O), then the message types
that lens over them, then escape/unescape and the full accessor (which the typed segments and `Set` depend on), then
`BuildACK` (depends on construction + `AckCode`), then the MLLP wire codec (`WriteFrame`/`ReadFrame`), then the client,
then the server (modelled on `dimse.Server`'s `sync.WaitGroup` + loopback-default + context-cancelled `Shutdown`
discipline), and finally the batch/file containers and the `Container`/`ParseAny` dispatch. The package stays
single-package `hl7v2` (no sub-packages) — the reference doc commits `hl7v2.Client`/`hl7v2.Server`/`hl7v2.WriteFrame`
at the package root, mirroring `net/http`.

**Tech stack:** Go 1.26.3, module `github.com/codeninja55/go-radx`; standard library only for the wire and transport
codecs (`bytes`, `strings`, `strconv`, `io`, `net`, `crypto/tls`, `context`, `iter`, `time`); `go.uber.org/zap` for the
server's structured, no-PHI diagnostics (consistent with `dimse.Server`). No CGo, no new third-party dependency. The
optional interop gate (see Open questions) may add a `//go:build interop` test path; it adds no production dependency.

---

## How to use this plan

Read this section once before starting; it states the conventions every task follows.

**Test-first, always.** Each task is a strict TDD cycle: write the failing test, run it and confirm it fails for the
right reason, write the minimal implementation, run it and confirm it passes, then commit. Do not write implementation
before its test. See `agentic-dev:test-driven-development`.

**Extend, never rebuild.** The M2 slice is complete and green. Each task below names the existing `hl7v2` symbols it
extends (for example "extends `composite.go`'s `parseCWE` to the full six-component CWE", "adds `PV1`/`OBX`/`EVN`/`MSA`/
`ERR` alongside the existing `MSH`/`PID`/`ORC`/`OBR` in `typed.go`"). YOU MUST NOT rewrite the parse tree, the encoding
derivation, the MSH offset-quirk handling, or the round-trip renderer. If a change to existing M2 code seems necessary
beyond pure addition, stop and surface it (see Open questions) rather than rewriting — this is the project's
"never throw away the old implementation without permission" rule.

**Canonical names are mandatory.** Use the exact Go identifiers fixed in `docs/reference/hl7v2.md` and
`UBIQUITOUS_LANGUAGE.md`: `Message`, `Segment`, `Field`, `Repetition`, `Component`, `EncodingCharacters`, `Accessor`,
`MessageType`, `TriggerEvent`, `AckCode`, `MSH`, `EVN`, `PID`, `PV1`, `ORC`, `OBR`, `OBX`, `MSA`, `ERR`, `XPN`, `XAD`,
`CX`, `CWE`, `HD`, `DTM`, `Precision`, `ADT`, `ORM`, `OMG`, `ORU`, `ACK`, `ResultGroup`, `OrderGroup`, `Batch`, `File`,
`Container`, `Client`, `Server`, `Handler`, `HandlerFunc`. Never invent `NACK` as a type — a negative acknowledgement
is an `ACK` whose `MSA-1` carries a rejecting `AckCode` (glossary; reference doc "Acknowledgement codes"). The MLLP
constants are `StartBlock` (`0x0B`), `EndBlock` (`0x1C`), `CarriageReturn` (`0x0D`); the cap is `DefaultMaxFrameSize`
(`16 << 20`).

**Honour the committed API.** The signatures in `docs/reference/hl7v2.md` are the contract; that doc is the single
normative source for API shape, and `docs/conformance/hl7v2.md` is the single source for scope. Where this plan shows a
signature, it is copied from the reference doc. Do not invent new public API; if you find a genuine gap, stop and
surface it (see Open questions) rather than guessing.

**Bounds-check every length; HL7 arrives over the network.** The MLLP increments parse hostile peer input. Frame and
body size are bounded **before** allocation: `ReadFrame` stops at `EndBlock` or `maxFrame` bytes, whichever comes
first, and never buffers an unbounded frame to "check the length afterwards" (PRD §9.3; reference doc "MLLP
transport"). A stream that ends mid-frame returns `io.ErrUnexpectedEOF`; a frame that does not begin with `StartBlock`
returns `*FramingError`; exceeding `maxFrame` returns `*LimitExceededError`. The parser increments carry the same
discipline already in place: a body that ends mid-segment is `io.ErrUnexpectedEOF` wrapped in a `*ParseError`
(truncation is failure, PRD §9.2). Each parsing/framing task includes a hostile-length regression test, and the
fuzz/hostile-input run must not panic, OOM, or hang (PRD §9.3, §11.2).

**Diagnostics carry no PHI.** Errors and logs name the concept — a segment ID, a field position by `SEG-Fn` accessor,
a delimiter by name, an `AckCode` by name and class, an MLLP framing fault by protocol position — never a field
*value*,
because HL7 fields routinely carry PHI (patient names, identifiers, results). This is already the contract of
`ParseError`/`SegmentError`; the new `AccessorError`, `FramingError`, and `LimitExceededError` follow it, and the MLLP
`Server` logs handler faults without the message body (PRD §8.2, §9.1; reference doc "Behaviour and error model").

**Absence is not an error.** An absent optional segment, field, repetition, component, or subcomponent reads as the
empty value with a `false` presence flag, never an error (HL7's own rule). The new typed segments and message accessors
keep this: `if pv1, ok := msg.PV1(); ok { … }`. The HL7 explicit-null `""` (a present-but-empty field) is
distinguished from absence — `Get` returns the literal `""` quote pair for an explicit null and the empty string for
an absent position (reference doc "Absence is not an error").

**Round-trip fidelity is a tested invariant.** `Parse` followed by `MarshalText` reproduces a conformant message
byte-for-byte. M5 must not regress this; every new typed segment's `Segment(enc)` renderer and the batch/file
containers' `MarshalText` are tested for round-trip against fixtures (PRD §11.1; reference doc "Round-trip fidelity").

**Concurrency discipline.** No global mutable state — delimiters, caps, and timeouts are per-instance, exactly as the
M2 `parseConfig` already is. The MLLP `Server` runs one goroutine per connection, tracks them with a `sync.WaitGroup`,
binds to loopback unless an explicit non-loopback address is given, and drains in-flight connections on a
context-cancelled `Shutdown` — the same model as `dimse.Server` (`server.go`); reuse that discipline, do not invent a
new one. No fire-and-forget goroutine (PRD §9.4).

**Commit conventionally and often.** Each commit follows `<type>(hl7v2): <description>` (for example
`feat(hl7v2): add MLLP frame codec with hostile-frame caps`). Source and its test commit together; fixtures and tooling
commit separately (project Atomic Commit Strategy).

**The reference docs ARE the specs.** Read the cited `docs/reference/hl7v2.md` section before implementing each
increment. The HL7 base standard Chapter 2 (encoding) and Chapter 2 §2.10 (escaping) are the underlying authority;
`python-hl7` is the parity floor (verify its behaviour from its source/docs, do not trust it from memory), and HAPI
(Java) is a quality reference for the segment/datatype layouts and `BuildACK` field-swap.

---

## Increment overview (dependency-ordered)

The pure, no-I/O work leads (composites, typed segments, message types, escape, accessor), because the transport and
acknowledgement layers depend on it. MLLP follows once there is a `*Message` to frame and a `BuildACK` to reply with.
Batch/file closes the parser surface last.

- **Increment 0 — Test corpus and harness.** Vendor a small `python-hl7`-derived fixture corpus under
  `testdata/hl7v2/` (ADT, ORU, ORM, ACK, a batch, a file, a non-standard-delimiter message) with license attribution,
  add the `test:hl7v2` target's M5 coverage, and stand up the MLLP loopback self-test scaffolding (no real container).
  No production code. *Outlined.*
- **Increment 1 — Composite datatypes to full depth.** Extend `composite.go`: add `XAD`; extend `CWE` to its six
  modelled components and `XPN` to `Degree`; add the typed-segment renderers' shared component-builder. Pure, tested in
  isolation. *Fully expanded into bite-sized TDD tasks below.*
- **Increment 2 — The full typed-segment set + into-generic renderers.** Add `EVN`, `PV1`, `OBX`, `MSA`, `ERR` typed
  segments and their `ParseXxx`/`(x) Segment(enc)` pairs; extend `PID` with `Address` (PID-11), `MSH` and `ORC`/`OBR`
  unchanged in shape; add `Message.EVN()`/`PV1()`/`AllOBX()` accessors. *Outlined.*
- **Increment 3 — The `ADT` message type and trigger events.** `ADT` typed lens with
  `Event()`/`EVN()`/`PID()`/`PV1()`, `AsADT`, and the A01/A02/A03/A04/A08 trigger-event scope. Feeds (does not build)
  `convert.ADTToPatientR5`/`ADTToEncounterR5` (M7). *Outlined.*
- **Increment 4 — The `ORU` message type (results).** `ORU` typed lens with `PID()` and
  `Results() iter.Seq[ResultGroup]` (each `OBR` with its following `OBX` rows), `ResultGroup`, `AsORU`. Mirrors the
  existing `ORM.Orders()` grouping. Feeds (does not build) `convert.ORUToDiagnosticReportR5` (M7). *Outlined.*
- **Increment 5 — Escape/unescape (Chapter 2 §2.10).** `Escape`/`Unescape` over the §2.10 floor set (separator,
  highlight, hex, rich-text, application-defined), the `UnescapeNotes` surface for the declined inline-charset escapes,
  and the encoding-derived escape table. *Outlined.*
- **Increment 6 — The full `Accessor` (prefixed keys, segment-instance, future-proofed resolution, `Set`).** Extend
  the existing numeric-only `Get` to the committed `Accessor`/`ParseAccessor`, the prefixed `PID.F5.R1.C2` form, the
  `PID2-5` segment-instance form, future-proofed descend-past/-short resolution, unescape-on-read, and `Message.Set`
  (escape-on-write, never invents a segment). Adds `AccessorError`. *Outlined.*
- **Increment 7 — Construction + `AckCode` + `BuildACK`.** `NewMessage`/`SetMSH`/`AppendSegment`, the `AckCode` enum
  with `ParseAckCode`/`IsPositive`/`IsError`/`IsReject`, the `ACK` typed lens (`MSA()`/`Errors()`, `AsACK`), and the
  spec-correct `BuildACK` (§2.9.2 field-swap) with `ACKOption`s. *Outlined.*
- **Increment 8 — MLLP frame codec.** `StartBlock`/`EndBlock`/`CarriageReturn`/`DefaultMaxFrameSize`, `WriteFrame`,
  `ReadFrame` (bounded, context-aware), `FramingError`, `LimitExceededError`. The hostile-frame floor. *Outlined.*
- **Increment 9 — The MLLP client.** `Client`, `NewClient`, `ClientOption`s, `Send` (blocks for the ACK frame),
  `SendRaw`, `Close`. Blocking, context-aware, loopback-tolerant. *Outlined.*
- **Increment 10 — The MLLP server.** `Server`, `NewServer`, `ServerOption`s, `Handler`/`HandlerFunc`, `Serve`,
  `Shutdown` — loopback-default bind, one goroutine per connection tracked by `sync.WaitGroup`, context-cancelled
  graceful drain, TLS option, modelled on `dimse.Server`. Plus the MLLP loopback self-test (client↔server round-trip).
  *Outlined.*
- **Increment 11 — Batch and file containers + `ParseAny`.** `Batch` (`BHS`/`BTS`), `File` (`FHS`/`FTS`), the
  both-or-neither header/trailer rule, `ParseBatch`/`ParseFile`/`ParseAny`, the `Container` interface, and
  `MalformedBatch`/`MalformedFile` as typed `*ParseError`s. *Outlined.*

Increment 1 is fully expanded into bite-sized TDD tasks below; Increments 2–11 are outlined (goal, files, key tests,
reference-doc section, extends-note, verification gate) and expanded into bite-sized TDD tasks when reached, exactly as
the walking-skeleton plan fully expanded its Increment 1 and outlined the rest.

---

## Increment 0 — Test corpus and harness

**Goal:** Stand up the fixtures and harness M5 verifies against, before any production code. Vendor a small corpus of
real-shaped HL7 v2 messages under `testdata/hl7v2/` (a `python-hl7`-derived set with license attribution per the §5.3
test-corpus vendoring decision): an `ADT^A01`, an `ADT^A03`, an `ORU^R01` with multiple `OBR`+`OBX` groups, an
`ORM^O01`, an `OMG^O19`, an `ACK`, a `BHS`/`BTS` batch, an `FHS`/`FTS` file, a message using non-standard delimiters
(`MSH|^~\&` replaced), and a message with §2.10 escape sequences. Add `test:hl7v2` coverage (already present in
`mise.toml`) and the MLLP loopback self-test scaffolding directory. No production code beyond what already exists.

**Files:** `testdata/hl7v2/*.hl7` (the vendored corpus), `testdata/hl7v2/LICENSE-python-hl7.txt` (attribution),
`testdata/hl7v2/README.md` (provenance and what each fixture exercises). No `.go` production files. **New corpus**
(the `hl7v2` package code already exists from M2).

**Key tests:** none yet — this increment lands fixtures and attribution only. The existing `go test ./hl7v2/...` must
stay green (the corpus must not be imported by a test that does not yet exist).

**Reference-doc section:** hl7v2.md "Worked examples" (the example messages are the shape templates); PRD §5.3 (test-
corpus vendoring with license attribution), §11.1 (round-trip against the `python-hl7` corpus). **New corpus.**

**Verification gate:** `go test -race ./hl7v2/...` still green; the corpus files are present and license-attributed;
`gofmt`/lint unaffected (no new `.go`). Confirm the chosen `python-hl7` fixtures' license permits vendoring before
committing (see Open questions on corpus provenance).

---

## Increment 1 — Composite datatypes to full depth (fully expanded)

**Goal:** Bring the composite datatypes from the M2 convert-feeding subset up to the full reference shapes, so the
typed segments in Increment 2 have complete composites to read and render. The M2 `composite.go` already models `HD`
(3 components), `CX` (ID/CheckDigit/AssigningAuthority/IdentifierTypeCode), `CWE` (Code/Text/CodingSystem only), `XPN`
(through NameTypeCode), and `DTM`. M5 adds `XAD` (extended address), extends `CWE` to its six modelled components and
`XPN` to `Degree`, and adds the shared component renderer the typed-segment `Segment(enc)` methods (Increment 2) need
to build a `Repetition` back from a composite. All work is pure (no I/O) and tested in isolation. Round-trip
(`parse → render → parse`) is the load-bearing invariant for each composite.

This increment **extends** `composite.go`; it does not touch the parse tree or the existing `parseHD`/`parseCX`/`DTM`.

### Task 1.1: Extend `CWE` to its six modelled components

**Files:** `hl7v2/composite.go` (extend the `CWE` struct and `parseCWE`), `hl7v2/composite_test.go`.

Add `AltCode` (CWE-4), `AltText` (CWE-5), and `AltCodingSystem` (CWE-6) to the existing `CWE` struct (which has
`Code`/`Text`/`CodingSystem`) and read them in `parseCWE` via the existing `r.component(n)` workhorse. The reference
doc fixes these exact six fields.

- [ ] Write the failing test: `parseCWE` of a repetition `36643-5^CHEST XRAY^LN^36643^CHEST X-RAY^L` populates all six
  fields; a three-component CWE leaves the alternate fields `""` (absence is empty, not error).
- [ ] Confirm it fails (the alternate fields do not yet exist).
- [ ] Extend the struct and `parseCWE` minimally.
- [ ] Confirm green; commit `feat(hl7v2): extend CWE to its six modelled components`.

### Task 1.2: Extend `XPN` with `Degree` (XPN-6)

**Files:** `hl7v2/composite.go` (extend `XPN`/`parseXPN`), `hl7v2/composite_test.go`.

The M2 `XPN` models Family/Given/Middle/Suffix/Prefix and NameTypeCode (XPN-7) but skips `Degree` (XPN-6). The
reference doc lists `Degree string // XPN-6` between Prefix and NameTypeCode.

- [ ] Write the failing test: `parseXPN` of `DOE^JOHN^A^JR^DR^PHD^L` reads `Degree == "PHD"` and `NameTypeCode == "L"`
  (proving the component numbering does not shift).
- [ ] Confirm it fails.
- [ ] Add `Degree` reading `r.component(6)`; keep `NameTypeCode` at `r.component(7)`.
- [ ] Confirm green; commit `feat(hl7v2): add XPN Degree component`.

### Task 1.3: Add the `XAD` extended-address composite

**Files:** `hl7v2/composite.go` (`XAD` struct + `parseXAD`), `hl7v2/composite_test.go`.

Add `XAD` with `Street` (XAD-1), `OtherDesignation` (XAD-2), `City` (XAD-3), `State` (XAD-4), `Zip` (XAD-5), `Country`
(XAD-6), reading via `r.component(n)` exactly as `parseHD`/`parseCWE` do.

- [ ] Write the failing test: `parseXAD` of `123 MAIN ST^APT 4^METROPOLIS^NY^10001^USA` populates all six fields; an
  absent address yields the zero `XAD`.
- [ ] Confirm it fails.
- [ ] Add the struct and `parseXAD`.
- [ ] Confirm green; commit `feat(hl7v2): add XAD extended-address composite`.

### Task 1.4: Add the shared composite-to-Repetition renderer

**Files:** `hl7v2/composite.go` (a small `componentsToRepetition([]string) Repetition` helper, or per-composite
`(c CWE) repetition(enc) Repetition`-style builders — pick the simpler that the Increment 2 renderers consume),
`hl7v2/composite_test.go`.

The typed-segment `Segment(enc)` renderers in Increment 2 must build a generic `Repetition`/`Field` back from a typed
composite so a constructed message round-trips. Add the minimal shared builder that turns an ordered list of component
strings into a `Repetition` (trailing-empty-component trimming matching how `MarshalText` already renders), so each
composite gets a small `repetition()` method without duplicating the slice-building logic.

- [ ] Write the failing test: building a `Repetition` from a `CX` with `ID:"PATID1234"`,
  `AssigningAuthority:HD{NamespaceID:"HOSP"}`, `IdentifierTypeCode:"MR"` and rendering it through the existing
  `Repetition.render` reproduces `PATID1234^^^HOSP^MR` (proving component gaps are preserved); a `CWE{Code:"NM"}`
  renders as `NM` with no trailing carets.
- [ ] Confirm it fails.
- [ ] Add the shared builder and the per-composite `repetition()` methods for `CX`/`CWE`/`HD`/`XPN`/`XAD`.
- [ ] Confirm green; commit `feat(hl7v2): add typed-composite to-Repetition renderers`.

### Task 1.5: Composite round-trip property test

**Files:** `hl7v2/composite_test.go`.

A table-driven round-trip: for each composite, `parse(render(value)) == value` and `render(parse(raw)) == raw` for a
set of realistic and edge (empty, trailing-empty-component, subcomponent-bearing) inputs. This is the composite-level
guard for the package-level round-trip invariant.

- [ ] Write the table test covering `HD`/`CX`/`CWE`/`XPN`/`XAD` round-trips including empty and gappy values.
- [ ] Confirm it passes against the Task 1.1–1.4 implementation (or fails and is fixed if a renderer drops a gap).
- [ ] Commit `test(hl7v2): add composite round-trip property test`.

**Verification gate:** `go test -race ./hl7v2/...` green; `golangci-lint` clean; the composite round-trip property
test passes; no change to `parse.go`/`encoding.go`/`segment.go` (extension only).

---

## Increment 2 — The full typed-segment set + into-generic renderers

**Goal:** Add the typed segments the conformance statement lists beyond the M2 set — `EVN`, `PV1`, `OBX`, `MSA`, `ERR`
— each with a `ParseXxx(s Segment) (Xxx, error)` parse-from-generic constructor (validating the segment ID, parsing
composites to the reference-doc field shapes) and a `(x Xxx) Segment(enc EncodingCharacters) Segment` into-generic
renderer (so constructed messages round-trip). Extend `PID` with `Address XAD` (PID-11). Add the `Message` accessors
`EVN()`/`PV1()` (value + presence flag) and `AllOBX()` (every `OBX` in order). The field shapes are fixed by the
reference doc: `OBX.Value` is `[]string` at OBX-5 interpreted per `OBX.ValueType` (OBX-2); `MSA.AckCode` is the typed
`AckCode` (defined in Increment 7 — until then `MSA` reads MSA-1 as a string and is upgraded to `AckCode` in Increment
7, OR Increment 7 is reordered before this — see Open questions on the MSA/AckCode ordering); `ERR` carries
Location/Code(`CWE`)/Severity. This **extends** `typed.go` alongside the existing `MSH`/`PID`/`ORC`/`OBR`.

**Files:** `hl7v2/typed.go` (add `EVN`/`PV1`/`OBX`/`MSA`/`ERR` structs, `ParseEVN`/`ParsePV1`/`ParseOBX`/`ParseMSA`/
`ParseERR`, the `(x) Segment(enc)` renderers, extend `PID` with `Address`, add `Message.EVN()`/`PV1()`/`AllOBX()`),
`hl7v2/typed_test.go`. **Extends** (`typed.go` exists from M2; add to it, do not rewrite `ParseMSH`/`ParsePID`/
`ParseORC`/`ParseOBR`).

**Key tests:** `ParsePV1` of `PV1|1|I|ICU^101^A||||1234^DOE^JANE^^^^DR|||||||||||V123` reads `PatientClass=="I"` and the
attending-doctor `XPN`; `ParseOBX` of an `OBX|1|NM|...||182|mg/dl|70-105|H|||F` reads `ValueType=="NM"`,
`Value==["182"]`, `AbnormalFlags==["H"]`, `ResultStatus=="F"`; `ParseOBX` of a repeated OBX-5 (`a~b`) reads
`Value==["a","b"]`; `ParseERR`
reads the CWE code and severity; `ParseEVN` reads the event-type code and recorded `DTM`; `ParsePID` now reads PID-11
into `Address`; a wrong-ID segment returns `*SegmentError`; each typed segment's `Segment(enc)` renders back to a line
that re-parses equal (typed round-trip); `Message.AllOBX()` returns every OBX in document order.

**Reference-doc section:** hl7v2.md "Typed segments" (the `PV1`/`OBX`/`MSA`/`ERR`/`EVN` field lists and the
`ParsePID`/`(p PID) Segment` constructor/renderer pair), "Composite datatypes"; conformance hl7v2.md "Supported typed
segments", "Two scope-load-bearing field placements" (`OBX.Value` is `[]string` at OBX-5). **Extends.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the typed round-trip and wrong-ID `*SegmentError`
regressions pass; the M2 `MSH`/`PID`/`ORC`/`OBR` tests still pass unchanged.

---

## Increment 3 — The `ADT` message type and trigger events

**Goal:** Add the `ADT` typed message lens over `*Message` (no tree copy, exactly like the existing `ORM`):
`ADT struct{ *Message }`, `func (a ADT) Event() string` (MSH-9.2), `func (a ADT) EVN() (EVN, bool)`,
`func (a ADT) PID() (PID, bool)`, `func (a ADT) PV1() (PV1, bool)`, and `func AsADT(m *Message) (ADT, bool)` verifying
MSH-9.1 is `ADT`. The in-scope trigger events are A01/A02/A03/A04/A08 (conformance table); other ADT triggers parse
into the generic tree but get no special handling — `AsADT` accepts any `ADT^Axx` (the trigger-event scope governs the
*tested* set and the downstream converter, not a parse-time reject). This feeds — but does **not** build —
`convert.ADTToPatientR5`/`ADTToEncounterR5`, which are M7 (Conversion); M5 delivers only the typed `ADT` view those
converters consume.

**Files:** `hl7v2/adt.go` (`ADT`, `AsADT`, the accessors — new file mirroring `orm.go`), `hl7v2/adt_test.go`.
**New file, extends the message-types layer** (`orm.go` is the template; do not modify it).

**Key tests:** `AsADT` of an `ADT^A01` returns `(ADT, true)`; of an `ORU^R01` returns `(_, false)`; `a.Event()` reads
`A01` from MSH-9.2; `a.EVN()`/`a.PID()`/`a.PV1()` return the typed segments with presence flags; an `ADT^A03` with no
PV1 reports `PV1()` absent as `(_, false)`, not an error; the vendored `ADT^A01` and `ADT^A03` fixtures parse and lens.

**Reference-doc section:** hl7v2.md "Message types" (the `ADT` lens, `AsADT`), "Typed segments" (`EVN`/`PID`/`PV1`);
conformance hl7v2.md "Supported message types and trigger events" (ADT A01/A02/A03/A04/A08), "Typed message types in
scope". **New file.** Note: the ADT→Patient/Encounter converters are M7; M5 builds only the `hl7v2.ADT` view (see Open
questions on the M7 converter dependency).

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the ADT fixture lens tests pass; `AsADT` on a
non-ADT returns `false` (not an error, not a panic).

---

## Increment 4 — The `ORU` message type (results)

**Goal:** Add the `ORU` typed message lens, mirroring `ORM.Orders()`: `ORU struct{ *Message }`,
`func (o ORU) PID() (PID, bool)`, `func (o ORU) Results() iter.Seq[ResultGroup]` (each `OBR` with its following `OBX`
rows, grouped by segment order), the `ResultGroup struct{ Order OBR; Observations []OBX }`, and
`func AsORU(m *Message) (ORU, bool)` verifying MSH-9.1 is `ORU`. The grouping logic is the OBR/OBX analogue of the
existing ORC/OBR grouping in `orm.go`'s `Orders()` — reuse that iterator shape (a Go 1.23+ `iter.Seq`), do not invent
a new one. Feeds — but does **not** build — `convert.ORUToDiagnosticReportR5` (M7).

**Files:** `hl7v2/oru.go` (`ORU`, `ResultGroup`, `AsORU`, `Results`, `PID` — new file mirroring `orm.go`),
`hl7v2/oru_test.go`. **New file, extends the message-types layer** (`orm.go`'s `Orders()` is the template).

**Key tests:** `AsORU` of an `ORU^R01` returns `(ORU, true)`; of an `ORM^O01` returns `(_, false)`; the vendored
multi-group `ORU^R01` yields the correct number of `ResultGroup`s, each with its `OBR` and the `OBX` rows that follow
it (and not the next group's); an OBX appearing before the first OBR is not yielded (a well-formed ORU opens each result
with an OBR); a trailing OBR with no following OBX yields a `ResultGroup` with an empty `Observations` slice;
`o.PID()` reads the patient. Match the existing `ORM.Orders()` boundary behaviour exactly.

**Reference-doc section:** hl7v2.md "Message types" (`ORU` lens, `ResultGroup`, `AsORU`, the worked ORU example that
iterates `Results()`), "Typed segments" (`OBR`/`OBX`); conformance hl7v2.md "Supported message types" (ORU^R01).
**New file.** Note: the ORU→DiagnosticReport converter is M7.

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the ORU grouping regressions (multi-group,
OBX-before-OBR, trailing-OBR) pass; the worked example in hl7v2.md compiles and runs.

---

## Increment 5 — Escape/unescape (Chapter 2 §2.10)

**Goal:** Implement the §2.10 escape mechanism the accessor (`Get`/`Set`, Increment 6) depends on:
`func Escape(value string, enc EncodingCharacters, appMap map[byte]string) string` and
`func Unescape(value string, enc EncodingCharacters, appMap map[string]string) (string, UnescapeNotes)`. The escape
table is **derived from the in-force `EncodingCharacters`** (not hardcoded `| ^ ~ \ &`): `\F\`↔field separator,
`\S\`↔component, `\T\`↔subcomponent, `\R\`↔repetition, `\E\`↔escape char. Cover the §2.10 floor set: separator,
highlight (`\H\`/`\N\`), hex (`\Xdd…\`), rich-text (`\.br\`/`\.sp\`/`\.in\`), and application-defined sequences. The
two inline character-set escapes (`\Cxxyy\`, `\Mxxyyzz\`) are **not decoded**; they are surfaced as `UnescapeNotes`
entries (never silently dropped), making the decline observable — go-radx's documented improvement over `python-hl7`.

**Files:** `hl7v2/escape.go` (`Escape`, `Unescape`, `UnescapeNotes` and its note type), `hl7v2/escape_test.go`.
**New file** (greenfield; parity floor is `python-hl7`'s escape handling — verify its sequence set from source).

**Key tests:** `Escape("a|b^c", DefaultEncoding(), nil)` produces `a\F\b\S\c`; with a non-standard `EncodingCharacters`
the escape table tracks the actual delimiters (the non-standard-sender guard); `Unescape("a\\F\\b", enc, nil)` recovers
`a|b`; `\Xc3a9\` hex decodes to its bytes; `\H\bold\N\` highlight sequences are handled; an unknown `\.foo\` rich-text
command is preserved or noted per the reference's policy; `\Cxxyy\` yields an `UnescapeNotes` entry and the literal
text untouched (declined, not lost); `Escape`∘`Unescape` round-trips printable text; a malformed `\X` with odd hex
digits is reported, not silently mangled (lexical-fidelity guard, PRD §9.2).

**Reference-doc section:** hl7v2.md "Escape and unescape (Chapter 2 §2.10)", "Documented limits" (inline charset
escapes surfaced via `UnescapeNotes`); conformance hl7v2.md "Encoding characters and escaping"; PRD §6.2 (HL7 floor:
"escape/unescape per the HL7 v2 base standard Chapter 2 §2.10"). **New file.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the encoding-derived escape table, the
`UnescapeNotes`-not-silent-loss, and the malformed-hex regressions pass.

---

## Increment 6 — The full `Accessor` (prefixed keys, segment-instance, future-proofed resolution, `Set`)

**Goal:** Promote the M2 numeric-only `Get` (which already handles `PID-5-1-2`/`PID.5.1.2` and absence-as-empty) to the
full committed accessor surface. Add the `Accessor` struct (`Segment`/`SegmentNum`/`Field`/`Repetition`/`Component`/
`Subcomponent`, all 1-based HL7 spec numbering), `ParseAccessor(key)` accepting **both** the numeric `PID-5-1-2` and the
prefixed `PID.F5.R1.C2` forms plus the segment-instance form `PID2-5` (second PID), `(a Accessor) String()` (canonical
numeric form), the future-proofed resolution rules (descend the first child to a leaf when the tree is deeper than the
path; return the leaf reached when the path is deeper but every extra step is index 1 — matching `python-hl7`'s
`extract_field` so vendored fixtures round-trip), unescape-on-read (via Increment 5; `MSH-1`/`MSH-2` returned verbatim,
never unescaped, because they *are* the delimiters), explicit-null `""` distinguished from absence, and
`Message.Set(key, value)` (grows the tree as needed, escapes via the in-force encoding, never invents a segment — the
target segment must exist). Add `AccessorError` (bad key, or a path that runs past a leaf). This **extends** `Get` in
`segment.go`; reuse `resolveSegment`/`resolveField` as the resolution core, do not rewrite them.

**Files:** `hl7v2/accessor.go` (`Accessor`, `ParseAccessor`, `String`, the prefixed/segment-instance parsing),
`hl7v2/segment.go` (extend `Get` to use the new `Accessor` + unescape + future-proofed resolution; add `Set`),
`hl7v2/errors.go` (add `AccessorError`), `hl7v2/accessor_test.go`. **Extends** (`Get` and `resolveSegment` exist from
M2).

**Key tests:** `ParseAccessor("PID-5-1-2")` and `ParseAccessor("PID.F5.R1.C2")` produce equal `Accessor`s;
`ParseAccessor("PID2-5")` sets `SegmentNum==2`; `a.String()` round-trips to canonical numeric; `Get("PID-5-1-1")` on the
fixture returns the unescaped family name; `Get` of an absent optional returns `("", nil)`; `Get` of a path past a leaf
returns `*AccessorError`; an explicit-null field returns the literal `""` quote pair, distinct from absence; `Get` of
`MSH-1`/`MSH-2` returns the delimiters verbatim (not unescaped); future-proofed resolution: a path one level deeper than
the tree with index 1 returns the leaf, deeper with index >1 returns `*AccessorError`; `Set("PID-8", "F")` mutates and
re-renders; `Set` against an absent segment returns `*AccessorError` (never invents a segment); `Set` of a value
containing a delimiter escapes it so a subsequent `Get` recovers the original.

**Reference-doc section:** hl7v2.md "The one indexing convention", "Accessor — the 1-based string path" (both
`ParseAccessor` styles, the `PID2-5` form, future-proofed resolution, `MSH-1`/`MSH-2` verbatim, `Set` never invents a
segment),
"Absence is not an error" (explicit-null vs absence); conformance hl7v2.md "Indexing convention", "Generic tree
fallthrough". **Extends.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; both accessor styles, segment-instance, future-
proofed-vs-`*AccessorError`, explicit-null-vs-absence, `MSH-1`-verbatim, and `Set`-escapes regressions pass; the M2
numeric `Get` tests still pass unchanged.

---

## Increment 7 — Construction + `AckCode` + `BuildACK`

**Goal:** Add message construction and acknowledgement. Construction: `NewMessage(enc) *Message` (empty message with an
MSH whose MSH-1/MSH-2 reflect `enc`), `(m *Message) SetMSH(h MSH)` (replace MSH from a typed value),
`(m *Message) AppendSegment(s Segment)`. The `AckCode` enum over HL7 Table 0008: `AckAccept`/`AckError`/`AckReject`
(`AA`/`AE`/`AR`, original mode) and `AckCommitAccept`/`AckCommitError`/`AckCommitReject` (`CA`/`CE`/`CR`, enhanced
mode), with `ParseAckCode(s) (AckCode, error)` (unknown → `*ParseError`) and the predicates `IsPositive()` (AA/CA),
`IsError()` (AE/CE), `IsReject()` (AR/CR). Upgrade `MSA.AckCode` to the typed `AckCode` (Increment 2 left it a string
pending this).
The `ACK` typed lens: `ACK struct{ *Message }`, `(a ACK) MSA() (MSA, bool)`, `(a ACK) Errors() []ERR`, `AsACK`. And
`(m *Message) BuildACK(code AckCode, opts ...ACKOption) (*Message, error)` — spec-correct per HL7 §2.9.2: swap
sending/receiving applications and facilities from the source MSH, set MSH-9 to `ACK^<trigger>^ACK`, mint a fresh
control ID, echo the source MSH-10 into MSA-2, set MSA-1 to `code`; a source with no MSH returns a typed error (not a
panic). `ACKOption`s override control ID, sending application/facility, and `WithACKText` (MSA-3). Verify the field-swap
against `python-hl7`'s `create_ack` and HAPI's ACK builder from source — this is the part of the floor most easily got
wrong.

**Files:** `hl7v2/build.go` (`NewMessage`, `SetMSH`, `AppendSegment`, `BuildACK`, `ACKOption`, `WithACKText` etc.),
`hl7v2/ack.go` (`AckCode` + constants + `ParseAckCode` + predicates, `ACK`, `AsACK`, `MSA()`, `Errors()`),
`hl7v2/build_test.go`, `hl7v2/ack_test.go`. **New files; extends** `typed.go`'s `MSA` (the `AckCode` field type) and
the message-types layer.

**Key tests:** `ParseAckCode("AA")` is `AckAccept`; `ParseAckCode("ZZ")` returns `*ParseError`;
`AckAccept.IsPositive()` is true, `AckError.IsError()` is true, `AckReject.IsReject()` is true, and
`AckCommitAccept.IsPositive()` is true; `BuildACK(AckAccept)` on the vendored ORU swaps MSH-3↔MSH-5 and MSH-4↔MSH-6,
sets MSH-9 to `ACK^R01^ACK`, mints a control ID distinct from the source, and sets MSA-2 to the source MSH-10 and MSA-1
to `AA`; `BuildACK(AckError, WithACKText("…"))` sets MSA-1 to `AE` and MSA-3 to the text; `BuildACK` on a message whose
MSH is missing returns a typed error (no panic);
`NewMessage(DefaultEncoding())` + `AppendSegment` produces canonical `\r`-terminated output that re-parses; `AsACK` of
the built ACK lenses `MSA()` with the right `AckCode`.

**Reference-doc section:** hl7v2.md "Construction and acknowledgement"
(`NewMessage`/`SetMSH`/`AppendSegment`/`BuildACK`),
"Acknowledgement codes" (the `AckCode` enum and predicates), "Message types" (`ACK` lens), the worked "Build and reply"
example; conformance hl7v2.md "Acknowledgement (ACK) in scope" (§2.9.2 field-swap, original + enhanced mode); PRD §6.2
(`create_ack`). **New files.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the field-swap, fresh-control-ID, MSA-2-echo,
missing-MSH-typed-error, and `ParseAckCode`-unknown regressions pass; the built ACK round-trips.

---

## Increment 8 — MLLP frame codec

**Goal:** The MLLP wire codec the client and server build on. Constants `StartBlock` (`0x0B`), `EndBlock` (`0x1C`),
`CarriageReturn` (`0x0D`), `DefaultMaxFrameSize` (`16 << 20`). `WriteFrame(w io.Writer, payload []byte) error` wraps
`payload` as `0x0B <payload> 0x1C 0x0D`. `ReadFrame(ctx context.Context, r io.Reader, maxFrame int) ([]byte, error)`
reads one frame, returns the unwrapped payload, stops at `EndBlock` or `maxFrame` bytes (whichever comes first,
**never** buffering an unbounded frame to measure it afterward), honours `ctx` cancellation between reads. A stream
that ends
mid-frame returns `io.ErrUnexpectedEOF`; a frame not beginning with `StartBlock` returns `*FramingError`; exceeding
`maxFrame` returns `*LimitExceededError`. This is the hostile-frame floor — the regression that defends against a peer
that opens a frame and never closes it.

**Files:** `hl7v2/mllp.go` (the constants, `WriteFrame`, `ReadFrame`), `hl7v2/errors.go` (add `FramingError`,
`LimitExceededError`), `hl7v2/mllp_test.go`, `hl7v2/mllp_fuzz_test.go` (the hostile-input fuzz target). **New file;
extends** `errors.go`.

**Key tests:** `WriteFrame` then `ReadFrame` round-trips a payload byte-for-byte; `ReadFrame` of bytes not starting with
`0x0B` returns `*FramingError`; a stream that stops after `0x0B` before `0x1C 0x0D` returns `io.ErrUnexpectedEOF`; a
frame longer than `maxFrame` returns `*LimitExceededError` **without** allocating the whole frame first (assert the
reader stops early, e.g. via a counting reader); a cancelled `ctx` aborts a blocked read with `ctx.Err()`; the fuzz
target over random/truncated/oversized framing never panics, OOMs, or hangs (PRD §9.3, §11.2 — fuzz hang is CI
failure, not skippable).

**Reference-doc section:** hl7v2.md "MLLP transport" (the framing constants, `WriteFrame`/`ReadFrame`,
`DefaultMaxFrameSize`, the truncation/framing/limit error contract), "Truncation and limits are failures"; conformance
hl7v2.md "MLLP transport in scope"; PRD §9.3 (max MLLP frame length, typed limit error, no panic on malformed input).
**New file.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the framing-error,
mid-frame-`io.ErrUnexpectedEOF`,
`*LimitExceededError`-before-allocation, and context-cancellation regressions pass; `go test -run=Fuzz` / a bounded
`go test -fuzz` smoke does not panic/OOM/hang.

---

## Increment 9 — The MLLP client

**Goal:** The blocking MLLP client. `Client` (configured via
`NewClient(addr string, opts ...ClientOption) (*Client, error)`),
`(c *Client) Send(ctx context.Context, m *Message) (*Message, error)` — frames `m` with `WriteFrame`, blocks for the
acknowledgement frame, reads it with `ReadFrame`, parses it to a `*Message`, and returns it (the caller lenses it via
`AsACK` and inspects `MSA-1`). `(c *Client) SendRaw(ctx context.Context, frame []byte) ([]byte, error)` for callers that
own their own framing. `(c *Client) Close() error`. All operations honour `ctx` cancellation and the configured max
frame size; `ClientOption`s set the dial timeout, read timeout, max frame size, and TLS config. No global state; the
client owns its connection.

**Files:** `hl7v2/client.go` (`Client`, `NewClient`, `ClientOption`s, `Send`, `SendRaw`, `Close`),
`hl7v2/client_test.go`. **New file** (depends on Increment 8 `WriteFrame`/`ReadFrame` and Increment 7 `BuildACK` for
the test's reply shape).

**Key tests:** against an in-process `net.Pipe` or a stub listener that echoes a `BuildACK`, `Send` transmits a message
and returns the parsed ACK with the expected `AckCode`; a cancelled `ctx` aborts a blocked `Send` with `ctx.Err()`; a
peer that returns an oversized frame surfaces `*LimitExceededError`; a peer that closes mid-reply surfaces
`io.ErrUnexpectedEOF`; `SendRaw` round-trips raw frames; `Close` is idempotent and releases the connection. The full
client↔server loopback round-trip is exercised in Increment 10.

**Reference-doc section:** hl7v2.md "MLLP transport" (`Client`/`NewClient`/`Send`/`SendRaw`/`Close`), the worked "Run an
MLLP server and a client" example; conformance hl7v2.md "MLLP transport in scope" (blocking client, configurable max
frame, read timeout, TLS); PRD §9.4 (context cancellation), §9.7 (TLS peer verification). **New file.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the cancellation, oversized-frame, and
mid-reply-truncation regressions pass.

---

## Increment 10 — The MLLP server

**Goal:** The concurrent, context-aware MLLP server, modelled on `dimse.Server`. `Handler` interface
(`Handle(ctx, *Message) (*Message, error)`) and `HandlerFunc` adapter; returning an error closes the connection (logged
without PHI), and returning a built ACK with a rejecting `AckCode` NAKs at the application level. `Server` with a
`Handler`, a `MaxFrameSize` (0 = `DefaultMaxFrameSize`), and TLS/timeout/logger via `ServerOption`s.
`NewServer(addr string, h Handler, opts ...ServerOption) (*Server, error)`.
`(s *Server) Serve(ctx context.Context) error`
— one goroutine per connection, reads a frame, decodes the message, calls the handler, writes the reply frame; tracks
goroutines with a `sync.WaitGroup`; binds to **loopback** unless an explicit non-loopback address is given (PRD §9.1;
reuse `dimse.Server`'s `loopbackAddr` discipline). `(s *Server) Shutdown(ctx context.Context) error` — stops
accepting, cancels in-flight handlers, drains within the deadline, returns `ctx.Err()` if handlers do not finish
(re-runnable
bounded join, NOT a `sync.Once`-gated rubber-stamp — the exact `dimse.Server` Shutdown lesson). Then the **MLLP
loopback self-test**: a `Server` and `Client` on `127.0.0.1:0`, the client `Send`s a message and the server replies
`BuildACK`,
proving the framing+client+server+ACK path end-to-end without any external dependency (the interop strategy — see Open
questions).

**Files:** `hl7v2/server.go` (`Server`, `NewServer`, `ServerOption`s, `Handler`, `HandlerFunc`, `Serve`, `Shutdown`),
`hl7v2/server_test.go` (incl. the bind-default test and the loopback self-test). **New file** (model on
`dimse/server.go`;
reuse its `sync.WaitGroup` + capacity + `loopbackAddr` + re-runnable-`Shutdown` patterns — do not invent a new
concurrency model).

**Key tests:** a server bound with `":2575"` / `""` listens on `127.0.0.1` (bind-default test, PRD §11.2); an explicit
`0.0.0.0:port` binds the named interface; the loopback self-test (`Client.Send` ↔ `Server` + `BuildACK`) returns a
positive ACK; a handler returning an error closes that connection without leaking PHI to the log; a handler returning
an `AckError` ACK NAKs at the application level (the client sees `MSA-1==AE`); `Serve` returns when its context is
cancelled; `Shutdown` drains in-flight handlers and returns `nil` once they finish, `ctx.Err()` on deadline; a
second `Shutdown` after a deadline waits again and returns the real outcome (no false `nil`); an oversized inbound frame
is rejected with `*LimitExceededError` and the connection closed, not OOM.

**Reference-doc section:** hl7v2.md "MLLP transport" (`Server`/`NewServer`/`Serve`/`Shutdown`/`Handler`/`HandlerFunc`,
loopback default, one goroutine per connection, graceful drain, no global state, TLS with peer verification), the
worked server example; conformance hl7v2.md "MLLP transport in scope", "Concurrency"; PRD §9.1 (loopback default,
no-PHI logging), §9.4 (no fire-and-forget goroutine, context cancellation), §9.7 (TLS). **New file.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the bind-default, loopback-self-test,
NAK-via-handler, graceful-`Shutdown`, second-`Shutdown`-no-false-nil, and oversized-frame regressions pass. This
increment proves the MLLP leg end-to-end in-process (PRD §13 M5 interop scope — see Open questions on whether an
external HL7 interop gate is required).

---

## Increment 11 — Batch and file containers + `ParseAny`

**Goal:** The optional bulk containers. `Batch` (`BHS`/`BTS`) holding `Messages []*Message` and its encoding; `File`
(`FHS`/`FTS`) holding `Batches []*Batch`. The both-or-neither header/trailer rule (matching `python-hl7`'s
`MalformedBatchException`/`MalformedFileException`): a header without its trailer, a trailer without its header, or a
second `BHS` inside a batch is a `*ParseError` with a specific reason. `ParseBatch(b, opts...) (*Batch, error)` (parses
a `BHS`/`BTS` batch **or** a bare sequence of `MSH`-led messages with no `BHS`/`BTS`);
`ParseFile(b, opts...) (*File, error)`
(an `FHS`/`FTS` file or a bare batch); `ParseAny(b, opts...) (Container, error)` dispatching on the leading segment
(`MSH`→`*Message`, `BHS` or a multi-message body→`*Batch`, `FHS`→`*File`). The `Container` interface
(`encoding.TextMarshaler` + `Encoding() EncodingCharacters`), implemented by `Message`, `Batch`, and `File`, so any
container renders and reports its encoding. Containers re-render through `MarshalText` byte-exactly. The `BHS`/`FHS`
headers reuse the existing MSH offset-quirk handling for delimiter derivation (their first two fields declare the
delimiters exactly as MSH-1/MSH-2 do).

**Files:** `hl7v2/batch.go` (`Batch`, `File`, `Container`, `ParseBatch`, `ParseFile`, `ParseAny`, the
`MalformedBatch`/`MalformedFile` `*ParseError` reasons, the `BHS`/`BTS`/`FHS`/`FTS` typed headers if the reference
commits them), `hl7v2/batch_test.go`. **New file; extends** the parse layer (reuse `splitSegments`/`DeriveEncoding`/
`parseMSHSegment` for the `BHS`/`FHS` delimiter derivation — do not duplicate the offset-quirk logic).

**Key tests:** `ParseBatch` of the vendored `BHS`/`BTS` fixture yields the right `Messages` count and round-trips
byte-exactly via `MarshalText`; `ParseBatch` of a bare two-message body (no `BHS`) yields a two-message `Batch`;
`ParseFile` of the `FHS`/`FTS` fixture yields the batches and round-trips; a `BHS` with no `BTS` returns a `*ParseError`
(both-or-neither); a stray `BTS` with no `BHS` returns a `*ParseError`; a second `BHS` inside a batch returns a
`*ParseError`; `ParseAny` dispatches `MSH`→`*Message`, `BHS`→`*Batch`, `FHS`→`*File`; all three satisfy `Container`
and `Encoding()` reports the derived delimiters; a truncated container (ends mid-segment) returns
`io.ErrUnexpectedEOF`.

**Reference-doc section:** hl7v2.md "Parsing entry points" (`ParseBatch`/`ParseFile`/`ParseAny`), "The generic tree"
(`Container`), "Truncation and limits are failures" (`MalformedBatch`/`MalformedFile`), the worked "Parse a batch"
example; conformance hl7v2.md "Batch and file containers in scope" (both-or-neither rule); PRD §6.2 (HL7 floor: "Batch
(BHS/BTS) and File (FHS/FTS) containers with batch/file parsing"). **New file.**

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the both-or-neither, second-`BHS`, bare-body,
`ParseAny`-dispatch, and container-truncation regressions pass; the batch/file round-trip is byte-exact. This is the
**M5 acceptance gate**: the HL7 v2 package meets the §6.2 floor and the `docs/conformance/hl7v2.md` scope, and
`mise run test:hl7v2` is green.

---

## Open questions and resolutions

Resolve these against the committed specs (and with the architect where flagged) before or during execution.

1. **MLLP interop strategy — OPEN, for the architect.** HL7 v2 interop is harder than DICOM: there is no ubiquitous,
   reliable HL7 receiver testcontainer equivalent to Orthanc/dcm4chee-arc, and PRD §11.1 names the HL7 leg as "interop
   on the MLLP results leg" without naming a peer image. Three options, in increasing cost:
   (a) **MLLP loopback self-test only** (Increment 10) — the go-radx `Client` sends to the go-radx `Server`, proving
   the full framing+ACK path in-process with no external dependency. This is deterministic and CI-cheap but does not
   prove
   cross-implementation interop.
   (b) **A `python-hl7`/`hl7apy` MLLP receiver under `//go:build interop`** — containerize a tiny Python MLLP server
   (python-hl7's `hl7.mllp` or `hl7apy`'s MLLP) the go-radx `Client` sends to, and a go-radx `Server` the Python client
   sends to, proving both directions against the parity-floor library. Costs a container image and a flake budget.
   (c) **A HAPI (Java) MLLP endpoint** — the quality reference; heavier image, strongest interop signal.
   The plan defaults to **(a) as the merge gate** with **(b) recommended as a `//go:build interop` add-on** mirroring
   the DIMSE interop pattern, but the architect should confirm whether (b) is required to call M5 "done" or whether the
   loopback self-test plus byte-exact round-trip against the vendored `python-hl7` corpus suffices for v1. The
   walking-skeleton DIMSE precedent required real-peer interop; HL7 may not have an equivalently turnkey peer.

2. **`convert.ADTToPatientR5`/`ADTToEncounterR5` and `ORUToDiagnosticReportR5` — RESOLVED (M7, not M5).** These
   converters are committed in `docs/reference/convert.md` and consume the typed `hl7v2.ADT`/`hl7v2.ORU` views,
   `PID`/`PV1`/`EVN`/`OBX` segments, and `XPN`/`XAD`/`CWE`/`DTM` composites this milestone builds. **M5 delivers only
   the `hl7v2` typed views; the converters are M7 (Conversion).** Increments 3 and 4 must build the `ADT`/`ORU` lenses
   to
   the depth those converters read (verified against the convert.md mapping tables for ADT→Patient/Encounter and
   ORU→DiagnosticReport) so M7 has no `hl7v2` gap, but must NOT add anything to the `convert` package. If a convert.md
   mapping needs an `hl7v2` field not in the reference doc's typed-segment list, stop and surface it.

3. **`MSA.AckCode` field type vs increment ordering — OPEN (executor's call, default given).** Increment 2 adds the
   `MSA` typed segment, but the `AckCode` enum lands in Increment 7. Two clean options: (a) Increment 2 types
   `MSA.AckCode` as a `string` and Increment 7 changes it to the `AckCode` type when the enum exists (a one-line type
   change plus a `ParseAckCode` call in `ParseMSA`); or (b) move the `AckCode` enum (a pure, dependency-free enum) ahead
   of Increment 2 so `MSA.AckCode` is typed from the start. **Default: (b)** — pull the `AckCode` enum out of Increment
   7 into a tiny pre-Increment-2 task, because the reference doc fixes `MSA.AckCode` as the typed enum and a
   string→enum field change mid-plan is a small but avoidable churn. Surface to the architect if (a) is preferred to
   keep Increment
   7 cohesive.

4. **Vendored `python-hl7` corpus provenance and license — OPEN, verify before committing.** Increment 0 vendors a
   fixture corpus. Confirm the exact `python-hl7` (and/or `hl7apy`) example messages chosen are license-compatible with
   vendoring as test data (PRD §5.3: "test corpora only, with license attribution"). Prefer messages from the
   `python-hl7` test suite / docs (verify the repository license from source, do not assume). If license-compatible
   fixtures are scarce, author equivalent synthetic messages with clearly fictitious PHI and document them as go-radx
   originals — the round-trip invariant does not require the corpus to be `python-hl7`'s, only that it be realistic.

5. **`OMG^O19` typed view — RESOLVED (reuse `ORM`, no separate type beyond the existing `AsORM`).** The conformance
   table lists `OMG^O19` as a typed view feeding `convert.ORMToServiceRequest`, and the existing `AsORM` already accepts
   both `ORM` and `OMG` MSH-9.1 codes (the M2 `orm.go` switch). M5 does **not** add a separate `OMG` lens type; the
   `ORM` lens with its `ORC`+`OBR` `Orders()` iterator covers both, exactly as M2 built it and as the reference doc's
   message-types section implies. If a future scope needs OMG-specific segments (e.g. an `OMG`-only `OBR` field), that
   is a reviewed conformance change, not M5.

6. **`Set` growth semantics for repetitions/components — verify against `python-hl7`.** Increment 6's `Set` "grows the
   tree as needed" but "never invents a segment". Confirm from `python-hl7` source how far growth extends: growing
   missing fields/repetitions/components within an existing segment is in scope; the reference doc says the target
   *segment* must exist. Pin the exact behaviour (grow fields up to the addressed index; pad intervening positions with
   empty values) with a regression test so it matches the parity floor and the round-trip stays byte-stable.

---

## Verification summary (the M5 gate)

M5 is "done" when, in dependency order, every increment's gate is green and the package-level invariants hold:

- `mise run test:hl7v2` (`go test -race -cover ./hl7v2/...`) is green; `golangci-lint` is clean.
- Byte-exact `Parse`→`MarshalText` round-trip holds for every fixture in the vendored corpus, including the batch and
  file containers and the non-standard-delimiter message (PRD §11.1; reference doc "Round-trip fidelity").
- The hostile-input run (malformed messages and malformed MLLP frames under a memory cap) does not panic, OOM, or hang;
  the MLLP fuzz target is active and not skipped (PRD §9.3, §11.2).
- The PHI-default sanity holds: no field value appears in any error or server log at default verbosity (PRD §9.1).
- The bind-default test passes: the MLLP `Server` listens on loopback unless an explicit non-loopback address is given
  (PRD §11.2).
- The MLLP loopback self-test passes (client↔server↔`BuildACK`); the external HL7 interop gate is added if the
  architect requires it per Open question 1.
- The conformance statement `docs/conformance/hl7v2.md` scope is fully reached: typed segments `MSH`/`EVN`/`PID`/`PV1`/
  `ORC`/`OBR`/`OBX`/`MSA`/`ERR` + batch/file headers; message types `ADT` (A01/A02/A03/A04/A08), `ORM^O01`, `OMG^O19`,
  `ORU^R01`, `ACK`; composites `XPN`/`XAD`/`CX`/`CWE`/`HD`/`DTM`; escape/unescape (§2.10 floor set with `UnescapeNotes`
  for the declined inline-charset escapes); both accessor styles with segment-instance and future-proofed resolution;
  `BuildACK` (original + enhanced mode); `Batch`/`File` with the both-or-neither rule; MLLP client and context-aware
  server with configurable max frame, read timeout, and TLS.
