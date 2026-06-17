# HL7 v2 feature-parity matrix

Parity of `hl7v2/` against the python-hl7 documented public surface (the project's declared parity floor,
PRD §6.2) and, as a secondary yardstick, the HAPI HL7v2 message/trigger-event catalogue. Evidence is
file:symbol against main as of 2026-06-10. Tests count toward MET.

## Summary

**python-hl7 floor (primary): 32 features — 31 MET, 0 PARTIAL, 0 NOT-MET, 1 N-A.** The floor is fully
met: every parsing entry point, the format-sniff predicates, the full container model, accessor
extract/assign semantics, escape/unescape with caller-defined app maps, ACK construction, batch/file
protocols including the one-call `split_file` flatten, charset-decode on parse, the MLLP client/server
pair, the cross-implementation MLLP interop CI leg (issue #114), and the CLI sender all have shipped,
tested go-radx equivalents. The four former PARTIALs (boolean format predicates, a `split_file` helper,
charset-decode on parse, and custom `\Z\` escape maps) shipped together as the size S floor close-out.

**HAPI catalogue (stretch): expected and confirmed large breadth gap.** go-radx types 5 message families
(ADT, ORM, OMG, ORU, ACK) with a radiology-scoped trigger subset; HAPI v2.5 alone ships ~195
typed message structures across ~30 families, per-version from 2.1 to 2.8.1. The generic six-level tree
parses everything, so the gap is typed-view breadth, not parseability.

Top gaps by size:

1. Trigger-event catalogue breadth vs HAPI — SIU, MDM, DFT, BAR, VXU, QBP/RSP, MFN, pharmacy families
   have no typed views (size M per family; L for HAPI-equivalent breadth, which needs codegen).
2. Per-version typed structures (2.1-2.8.1) as HAPI does — go-radx is single-layout v2.5.1,
   version-tolerant (size L; deliberate v1 posture, not an accident).
3. ~~Cross-implementation MLLP interop in CI~~ — closed by issue #114: the `interop:mllp` CI leg
   provisions a pinned python-hl7 peer container and gates both send directions.

## python-hl7 floor (primary)

### Parsing and format detection

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| `hl7.parse()` | api.html `parse` | MET | `Parse` `hl7v2/parse.go:66`; `hl7v2/parse_test.go` | - | Six-level tree incl. subcomponents; truncation is a typed failure |
| `hl7.parse_batch()` | api.html `parse_batch` | MET | `ParseBatch` `hl7v2/container.go:45`; `hl7v2/container_test.go` | - | Headerless batch (bare MSH sequence) accepted, matching python-hl7 |
| `hl7.parse_file()` | api.html `parse_file` | MET | `ParseFile` `hl7v2/container.go:62` | - | File with no inner BHS wraps a single headerless batch |
| `hl7.parse_hl7()` auto-detect | api.html `parse_hl7` | MET | `ParseAny` `hl7v2/container.go:82`, `leadingSegmentID` `:339` | - | Dispatches MSH/BHS/FHS to Message/Batch/File via `Container` interface |
| `ishl7`/`isbatch`/`isfile` predicates | api.html helpers | MET | `IsHL7`/`IsBatch`/`IsFile` `hl7v2/sniff.go:11,46,67`; `hl7v2/sniff_test.go` | - | Cheap non-parsing sniffs matching python-hl7's second-MSH/BHS/FHS logic exactly |
| `split_file()` | api.html `split_file` | MET | `SplitFile` `hl7v2/sniff.go:96`; `hl7v2/sniff_test.go` | - | One call flattens a batch/file to messages, dropping FHS/BHS/FTS/BTS framing, `\r`-terminated |
| `encoding=` charset decode | api.html `parse(lines, encoding=...)` | MET | `WithCharset` `hl7v2/parse.go:30` decodes via `golang.org/x/text/encoding` before parse; `hl7v2/charset_test.go` | - | ASCII delimiters decode safely first; Latin-1 and Shift-JIS round-trip in tests |
| `Factory` custom containers | api.html `Factory` | N-A | - | - | Subclass-injection pattern; Go uses concrete typed structs by design |

### Container model and indexing

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Message/Segment/Field/Repetition/Component hierarchy | api.html containers | MET | `Message`/`Segment`/`Field`/`Repetition`/`Component` `hl7v2/parse.go`, `hl7v2/segment.go` | - | go-radx adds an explicit sixth subcomponent level |
| `str(message)` round-trip | index.html | MET | `MarshalText`/`String` `hl7v2/segment.go:46,55`; byte-exact corpus round-trip `hl7v2/corpus_test.go` | - | Containers round-trip too (`Batch.MarshalText` `hl7v2/container.go:282`) |
| `Message.segment()` / `Message.segments()` | api.html | MET | `Message.Segment` `hl7v2/segment.go:19`, `AllSegments` `:30` | - | |
| List-style 0-based indexing | accessors.html | MET | Tree fields are plain Go slices (0-based) | - | |
| Callable 1-based indexing (`Sequence.__call__`) | accessors.html | MET | `Message.Get` 1-based path `hl7v2/accessor.go:170` | - | Deliberate divergence: one convention per layer, never mixed on one object |
| `hl7.NULL` (`""`) explicit-null handling | api.html constants | MET | `TestGetExplicitNullVersusAbsence` `hl7v2/accessor_test.go:150` | - | Exceeds python-hl7: null is *distinguished* from absence, not conflated |

### Accessors, extract/assign, escaping

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| `Accessor` namedtuple + `parse_key` + `key` | api.html `Accessor` | MET | `Accessor`, `ParseAccessor` `hl7v2/accessor.go:29`, `String` `:138`; `hl7v2/accessor_styles_test.go` | - | Accepts `PID-5-1-2`, `PID.5.1.2`, and `PID.F5.R1.C2` styles |
| `extract_field` fallback rules | accessors.html | MET | `Message.Get` + `resolveLeaf` `hl7v2/accessor.go:170,212`, `defaultIndex` `:240` | - | Deeper/shallower-tree rules implemented; absent optionals read `("", nil)` |
| `assign_field` | api.html | MET | `Message.Set` `hl7v2/accessor.go:271` + `grow*` `:322-353` | - | Grows structure to path; never invents a segment; rejects MSH-1/MSH-2 |
| Auto-unescape on extract / escape on assign | accessors.html | MET | `Get` unescapes, `Set` escapes (see `hl7v2/accessor.go`); MSH-1/2 verbatim | - | Delimiter bytes in values cannot forge structure |
| `escape()` / `unescape()` | api.html | MET | `Escape` `hl7v2/escape.go:68`, `Unescape` `:151`; `hl7v2/escape_test.go` | - | Full §2.10 incl. `\Xdd\` hex, `\H\`/`\N\`, `\.br\`; derived from message delimiters |
| `app_map` custom `\Z\` sequences | api.html `escape(field, app_map)` | MET | `WithAppMap` `hl7v2/escape.go:46` on `Escape`/`Unescape`; `hl7v2/escape_appmap_test.go` | - | Caller map decodes/encodes site-defined sequences; takes precedence over the §2.10 decline-and-note path |

### ACK and utilities

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| `create_ack(ack_code, message_id, application, facility)` | api.html (spec 2.9.2) | MET | `BuildACK` `hl7v2/build.go:258`; options `WithControlIDSource`/`WithACKSendingApplication`/`WithACKSendingFacility` `:191-225`; `hl7v2/build_test.go` | - | Field-swap verified against python-hl7's `create_ack`; adds enhanced-mode codes |
| `generate_message_control_id()` | api.html | MET | `defaultControlID` `hl7v2/build.go:237`, injectable via `WithControlIDSource` `:191` | - | Never derives the ID from message content |
| `parse_datetime()` (DTM) | api.html | MET | `ParseDTM` `hl7v2/composite.go:323` incl. TZ offset `indexTZSign` `:378`; `hl7v2/composite_test.go` | - | Exceeds python-hl7: precision is preserved, not zero-filled |
| ACK code semantics | api.html | MET | `AckCode` + `IsPositive`/`IsError`/`IsReject` `hl7v2/ack.go:24-40` | - | Typed Table 0008 enum, original + enhanced modes |

### MLLP transport and CLI

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| `MLLPClient` (blocking, context manager) | api.html `hl7.client` | MET | `Client`, `NewClient` `hl7v2/mllp_client.go:90`, `Send`/`SendRaw`/`Close` `:127-187` | - | Adds dial/read timeouts, max-frame cap, TLS (`WithClientTLS` `:65`) |
| asyncio server (`start_hl7_server`) | api.html `hl7.mllp` | MET | Context-aware `Server`, `NewServer` `hl7v2/mllp_server.go:168`, `ListenAndServe` `:195`, `Shutdown` `:345` | - | Goroutine-per-connection is the idiomatic Go equivalent of asyncio |
| asyncio client (`open_hl7_connection`, stream reader/writer) | api.html `hl7.mllp` | MET | `Client.Send(ctx, ...)` honours cancellation `hl7v2/mllp_client.go:127` | - | Single client covers both blocking and cancellable use |
| `InvalidBlockError` + `limit` | api.html `hl7.mllp` | MET | `FrameError` `hl7v2/mllp.go:32`, `DefaultMaxFrameSize` `:26`, bounded `ReadFrame` `:89`; `FuzzReadFrame` `hl7v2/mllp_frame_test.go:187` | - | Frame size capped both sides (hostile-input guard); fuzz target ships |
| `mllp_send` CLI | mllp.html | MET | `radx hl7 send` `cmd/radx/internal/command/hl7.go:36` (file or stdin, host/port/timeout/max-frame) | - | Exceeds python-hl7: `radx hl7 listen` receiver also ships (`:152`) |
| Cross-implementation MLLP interop | (project gate, not python-hl7 doc) | MET | `interop:mllp` CI leg: `hl7v2/mllp_peer_main_test.go` provisions the pinned python-hl7 peer container; `TestInteropMLLPPeer` (go-radx sends) + `TestInteropMLLPPeerSender` (`mllp_send` sends) | - | Wired by issue #114; `RADX_HL7_MLLP_PEER` substitutes an external peer; loopback go-go gate also runs in CI |

### Batch and file protocols

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| `Batch` with `header`/`trailer` (BHS/BTS) | api.html `Batch` | MET | `Batch{Header, Trailer, Messages}` `hl7v2/container.go:22` | - | Both-or-neither header/trailer rule enforced (`*ParseError`) |
| `File` with `header`/`trailer` (FHS/FTS) | api.html `File` | MET | `File{Header, Trailer, Batches}` `hl7v2/container.go:33` | - | Byte-exact round-trip against `testdata/hl7v2/batch.hl7`, `file.hl7` fixtures |

### go-radx beyond the floor

Not python-hl7 features, listed for context: typed segments (`MSH`/`PID`/`PV1`/`EVN`/`ORC`/`OBR`/`OBX`/
`MSA`/`ERR`, `hl7v2/typed.go`), typed composites (`XPN`/`XCN`/`XAD`/`CX`/`CWE`/`HD`/`DTM`,
`hl7v2/composite.go`), typed message lenses (`hl7v2/lens.go`, `hl7v2/orm.go`), message construction
(`NewMessage`/`SetMSH`/`AppendSegment`, `hl7v2/build.go:13-34`), MLLP TLS on both client and server,
loopback-by-default server bind, and the explicit-null/absence distinction. python-hl7 has none of these.

## HAPI HL7v2 catalogue (secondary stretch)

HAPI ships per-version typed message structures for HL7 2.1 through 2.8.1; the v2.5 message package alone
contains ~195 structure classes. go-radx's deliberate v1 posture (docs/conformance/hl7v2.md) is a
radiology-scoped typed subset over a universal generic tree, so NOT-MET rows below are expected; sizing
matters most. "Generic tree" means the message parses losslessly and is fully reachable via
`Message.Get`/`Segment` — only the typed view is absent.

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Generic message access (`GenericMessage`) | HAPI base docs | MET | Six-level tree, `Message.Get`/`AllSegments` `hl7v2/accessor.go:170`, `hl7v2/segment.go:30` | - | The generic tree is the universal fallback for every untyped structure |
| Pipe parser | HAPI `PipeParser` | MET | `Parse` `hl7v2/parse.go:66` | - | |
| XML encoding (`XMLParser`) | HAPI `XMLParser` | NOT-MET | Explicit out of scope, docs/conformance/hl7v2.md "Out of scope" | L | v2.xml rarely used in radiology traffic; deliberate exclusion |
| Per-version typed structures 2.1-2.8.1 | HAPI version JavaDocs | PARTIAL | v2.5.1 layouts, version-tolerant parse of 2.3-2.8.1 (`VersionID` MSH-12 `hl7v2/typed.go:24`, not enforced) | L | HAPI-style per-version codegen is a different architecture |
| ADT family (HAPI v2.5: 17 structures, A01-A61 + AXX) | HAPI v25 message pkg | PARTIAL | `AsADT` `hl7v2/lens.go:18` accepts any ADT trigger; conformance-scoped to A01/A02/A03/A04/A08 | M | Lens checks only MSH-9.1 = "ADT", so other triggers get the same typed view |
| ORM/ORR order family | HAPI v25 (ORx: ~28 structures) | PARTIAL | `AsORM` `hl7v2/orm.go:18` (`ORM^O01`), `Orders()` `:35` yields ORC+OBR groups | M | ORR (order response) untyped; generic tree only |
| OMG (general clinical order) | HAPI v25 OMx | MET | Dedicated `AsOMG`/`OMG` lens `hl7v2/orm.go:39,50`; `AsORM` still admits the `OMG` code; `TestAsOMG` against the `omg-o19` fixture `hl7v2/orm_test.go:22` | - | `OMG^O19` scope; the ORC+OBR grouping is shared with `ORM.Orders` |
| OML/OMB/OMD/OMI/OMN/OMP/OMS | HAPI v25 OMx (9 structures) | NOT-MET | Generic tree only | M | OML^O21 (lab order) is the most-requested next family |
| ORU result family (R01...) | HAPI v25 | PARTIAL | `AsORU` `hl7v2/lens.go:47`, `Results()` `:62` (OBR+OBX groups) | S | R01 conformance-scoped; lens accepts any ORU trigger |
| ACK | HAPI v25 | MET | `AsACK` `hl7v2/lens.go:98`, `BuildACK` `hl7v2/build.go:258` | - | |
| SIU scheduling | HAPI v25 | NOT-MET | Generic tree only; listed out of scope docs/conformance/hl7v2.md | M | Typed lens over existing tree ~M per family |
| MDM document management | HAPI v25 | NOT-MET | Generic tree only; out of scope | M | |
| DFT / BAR billing | HAPI v25 | NOT-MET | Generic tree only; out of scope | M | |
| VXU immunization | HAPI v25 | NOT-MET | Generic tree only; out of scope | M | Low relevance to radiology workflows |
| QRY/QBP/RSP query family | HAPI v25 (~13 structures) | NOT-MET | Generic tree only | L | Needs query-grammar segments (QPD, RCP, QAK) typed first |
| MFN master files | HAPI v25 (~16 structures) | NOT-MET | Generic tree only | L | |
| Pharmacy (RDE/RAS/RDS/RRx...) | HAPI v25 (~12 structures) | NOT-MET | Generic tree only | L | Out of domain for go-radx's radiology focus |
| Conformance-profile validation | HAPI conformance tools | NOT-MET | Explicit out of scope docs/conformance/hl7v2.md | L | go-radx validates well-formedness + typed shape, not declared profiles |
| HL7-over-HTTP (HoH) transport | HAPI HoH module | NOT-MET | MLLP only (`hl7v2/mllp*.go`) | M | No equivalent planned; DICOMweb fills the HTTP niche in this stack |

## Methodology

Sources fetched 2026-06-10:

- python-hl7 docs: https://python-hl7.readthedocs.io/en/latest/ (index, api.html, accessors.html) via
  WebFetch. ctx7 could not resolve python-hl7 (both queries returned only the PHP `/senaranya/hl7`
  library), so readthedocs was the documented fallback.
- HAPI HL7v2: https://hapifhir.github.io/hapi-hl7v2/ (index) and the v2.5 message package summary
  (ca/uhn/hl7v2/model/v25/message/package-summary.html) via WebFetch. Family counts ("~195 structures")
  are from the fetched package summary and are approximate.
- go-radx evidence: direct file reads and grep over `hl7v2/` and `cmd/radx/internal/command/` at the
  working tree (main, post-fdf7b54). docs/conformance/hl7v2.md was used as a map; every claim cited
  here was re-verified against code or tests.

Not verified / caveats:

- The four former-PARTIAL floor rows (sniff predicates, `split_file`, charset decode, app-map escaping)
  were closed with shipped code and tests run under `go test -race ./hl7v2/...`. The remaining MET rows
  cite symbols and existing test files rather than fresh side-by-side runs against python-hl7.
- python-hl7's exact `extract_field` edge-case behaviour was compared at documentation level, not by
  running both libraries side by side; go-radx's own conformance statement claims rule-for-rule parity
  and its accessor tests exercise the deep/shallow-path rules.
- HAPI per-version structure counts for versions other than 2.5 were not enumerated.
- The cross-implementation MLLP interop leg is wired in CI (issue #114): the `interop:mllp` leg
  provisions a pinned python-hl7 peer container and gates both send directions, so MLLP wire-compat
  with python-hl7 is a CI-proven gate. hl7apy wire-compat remains untested.
