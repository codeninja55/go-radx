# Walking skeleton (M2) implementation plan

> **For agentic workers:** REQUIRED: Use agentic-dev:subagent-driven-development (if subagents available) or
> agentic-dev:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the thinnest correct end-to-end path through every go-radx subsystem (PRD §13 milestone M2): one DIMSE
**C-ECHO + C-STORE** as both SCU and SCP, one DICOMweb **STOW-RS** store and **WADO-RS** read (client + server), parse
one HL7 v2 **ORM** and emit one FHIR **ServiceRequest**, produce one FHIR **DiagnosticReport**, and one DICOM→FHIR
**ImagingStudy** conversion — all built on the completed `dicom` package (M1) and conforming exactly to the committed
public API in `docs/reference/{dimse,dicomweb,hl7v2,fhir,convert}.md`. The DIMSE leg is interop-gated against Orthanc
and dcm4chee-arc; the DICOMweb leg against Orthanc. The DUL rewrite plus the PDV layer must fix the prototype's
foundational defect (the last-fragment bit on the final command PDV) and guard it with a named regression test.

**Architecture:** M2 fills the subsystem packages the PRD §7.1 layout commits to. The `dimse` package is built in
three stacked layers, each a clear single-responsibility boundary (PRD §8.2, dimse.md "Overview of the layers"):
`dimse/pdu` (PDU/PDV codec, owns no socket), `dimse/dul` (the PS3.8 Table 9-10 state machine, owns the socket and ARTIM
timer, knows only PDUs), and `dimse/acse` (association negotiation) plus the DIMSE message layer (command-set
build/parse and PDV fragmentation) under the root `dimse` package. The SCU and SCP service operations (`Echo`, `Store`,
and the `Handler` interface) sit on top. `dicomweb`, `hl7v2`, `fhir` (a hand-written minimal R5 slice), `convert`, and
`server` are then layered, each reusing the `dicom` data model (`*dicom.DataSet`, `dicom.UID`, `dicom.TransferSyntax`,
`dicom.ReferencedSOPInstance`) rather than redeclaring it. Every leg ends green: unit tests with `-race`, a clean
`golangci-lint`, and — for the DIMSE and DICOMweb legs — a passing interop container test.

**Tech Stack:** Go 1.26.3, module `github.com/codeninja55/go-radx`; standard library for the wire codecs
(`encoding/binary`, `net`, `net/http`, `mime/multipart`, `encoding/json`, `crypto/tls`, `context`, `iter`);
`go.uber.org/zap` for structured, no-PHI diagnostics; `testcontainers-go` for the Orthanc / dcm4chee-arc interop gates
(already a dependency per PRD §11.1). No CGo in any M2 leg — the four uncompressed/deflated transfer syntaxes are the
only ones the skeleton negotiates and exercises end-to-end (dimse.md "Default transfer syntaxes").

---

## How to use this plan

Read this section once before starting; it states the conventions every task follows.

**Test-first, always.** Each task is a strict TDD cycle: write the failing test, run it and confirm it fails for the
right reason, write the minimal implementation, run it and confirm it passes, then commit. Do not write implementation
before its test. See `agentic-dev:test-driven-development`.

**Canonical names are mandatory.** Use the exact Go identifiers fixed in `UBIQUITOUS_LANGUAGE.md` and the reference
docs: `AETitle`, `Association`, `PresentationContext`, `AbstractSyntax`, `Status`, `ServiceClass`, `Priority`,
`QueryLevel`, `MaxPDULength`, `PDU`, `PDV` / `PresentationDataValue`, `State` (`Sta1..Sta13`), `AE`, `Server`,
`Handler`, `StoreHandler`, `OpInfo` (DIMSE); `ResourcePath`, `StoreResponse`, `StoredInstance`, `SearchQuery`,
`MultipartReader`/`MultipartWriter` (DICOMweb); `Message`, `Segment`, `Field`, `EncodingCharacters`, `ORM`,
`OrderGroup`, `ORC`, `OBR` (HL7 v2); `Resource`, `ServiceRequest`, `DiagnosticReport`, `ImagingStudy`, `Reference`,
`Identifier`, `Decimal` (FHIR). Never reintroduce the prototype's aliases: the DUL events are `Evt1..Evt19` (PS3.8),
**not** the prototype's `AE1..AE19`; presentation context is `dimse.PresentationContext`, never a bare `Context` that
collides with `context.Context`; a DICOM referenced instance is `dicom.ReferencedSOPInstance`, never a `Reference`
(that noun is FHIR's, per the cross-standard collision table).

**Honour the committed API.** The signatures in the reference docs are the contract. Do not invent new public API.
Where this plan shows a signature, it is copied from the reference doc; if you find a genuine gap, stop and surface it
rather than guessing (see "Open questions" at the end).

**Bounds-check every length.** `dimse` and `dicomweb` parse hostile network input. Every length read from the wire is
validated against the bytes remaining in a bounded reader before any `make([]byte, n)` (PRD §9.3; dimse.md "PDU and
PDV"; dicomweb.md "Hostile-input caps"). A PDV item length below the 2-byte header is rejected **before** the
subtraction (Codex DIMSE-004 — the prototype computed `dataLength := length - 2` with no `length >= 2` guard). Each
parsing task includes a hostile-length regression test.

**Diagnostics carry no PHI.** Errors and logs name the concept — a tag by keyword plus `(gggg,eeee)`, a VR by name, a
UID and SOP/transfer-syntax UID by registered name, a DIMSE `Status` by name and class, an HL7 locus by `SEG-Fn`
accessor, a FHIR locus by element path — never the patient value (PRD §8.2, §9.1; every reference doc's error model).
DIMSE `OpInfo` is the structured no-PHI diagnostic context for an SCP operation; pass it, never raw datasets, to the
logger.

**Status is data, errors are faults.** A DIMSE `Status` of `Failure` category is **not** a Go `error` — it is data the
caller inspects with `status.IsFailure()`. Transport, association, and protocol faults are `error`s
(`*AssociationError`, `*AbortError`, `*ProtocolError`). This separation lets a caller distinguish "the peer answered,
and said no" from "the conversation broke" (dimse.md "Error model"). Never launder a `Failure` status into success, and
never report success on work not done (PRD §9.2 fail-closed; the rule the prototype's `store` violated).

**Commit conventionally and often.** Each commit message follows `<type>(<pkg>): <description>` (for example
`feat(dimse/pdu): add P-DATA-TF PDV encode with last-fragment bit`). Source and its test commit together; generated
data, fixtures, and tooling commit separately (per the project Atomic Commit Strategy).

**Codex defect traceability.** Tasks that close a specific audited defect cite it inline (for example "guards Codex
DIMSE-001"). A defect is not closed until its named regression test passes. The DIMSE defects this milestone must fix,
from dimse.md and PRD §2.2/§12:

| Defect | What the prototype did wrong | Where M2 fixes it |
|--------|------------------------------|-------------------|
| DIMSE-001 | final command PDV not marked last (`0x03`) independently of a following dataset — the concrete Orthanc-abort root cause (PRD line 60) | Increment 5 (DIMSE message layer), regression test |
| DIMSE-002 | reassembler did not gate on command-last before deciding a dataset follows | Increment 5 reassembler state machine |
| DIMSE-003 | dataset decoded as Implicit VR LE regardless of the negotiated transfer syntax | Increment 5 |
| DIMSE-004 | PDV `length - 2` with no `length >= 2` guard (underflow → giant allocation) | Increment 1 PDV decode |
| DIMSE-005 | max-PDU `0` treated as a literal allocation size, not "unlimited" | Increment 1 / Increment 5 fragmentation |
| DIMSE-006/007 | command set not in increasing tag order; Move Destination given VR `UI` not `AE`; group length not computed last | Increment 5 (C-STORE command set) |
| DIMSE-008 | rejected presentation context omitted its (insignificant) transfer-syntax sub-item | Increment 2 PDU associate codec |
| DIMSE-010/011 | two-state release model; silent socket close on unexpected/invalid PDU instead of A-ABORT | Increment 2 DUL rewrite (Sta9–Sta12, Evt19 → AA-8) |
| DIMSE-013/014 | SCP accepted before capacity acquired; Shutdown left handlers blocked in ReadPDU | Increments 6/7 (SCP) |
| DIMSE-017 | operations panicked on an unestablished/released association | Increments 4–7 (guarded with typed error) |

The C-FIND/C-GET/C-MOVE-specific defects (DIMSE-015, DIMSE-016) are **out of M2 scope** (those services are M3) and
are not addressed here.

**Salvage map (port-with-fixes vs rewrite).** The read-only prototype lives at `/Users/codeninja/vcs/go-radx/dimse`.
Per PRD §12:

| Verdict | Prototype source | M2 increment |
|---------|------------------|--------------|
| **PORT-WITH-FIXES** | `dimse/pdu` (`pdu.go`, `data.go`, `associate.go`, `release.go`, `abort.go`) | Increments 1–2 |
| **PORT-WITH-FIXES** | `dimse/scu` (`client.go` — its C-ECHO/C-STORE structure) | Increments 4–5 |
| **PORT-WITH-FIXES** | `dimse/integration` (the Orthanc integration tests, ported first as a regression net) | Increment 0 / Increment 12 |
| **REWRITE** | `dimse/dul` (`statemachine.go` — only 9 states, misnamed `AE1..AE19` events, ~14 actions) | Increment 2 |
| **REWRITE** | `dimse/dimse` (`message.go` — the PDV last-fragment-bit defect) | Increment 5 |
| **REWRITE** | `dimse/scp` (`server.go`) | Increments 6–7 |

Port the *logic and wire layouts* (PDU byte structure, sub-item encoding, association negotiation) but adopt the
committed types: `AETitle` (named type, not bare string), `dicom.TransferSyntax` / `dicom.SOPClassUID` (reused from
`dicom`, not redeclared as strings), the `Status`/`ServiceClass` typed status model, and `iter.Seq2` where the
reference doc commits it. The prototype's `MessageControlCommand`/`MessageControlLastFragment` constants are correct
values (`0x01`/`0x02`) but the prototype conflated `MessageControlLastFragment` and `MessageControlDatasetLast` as the
same `0x02` and decided the boundaries wrongly — the rewrite separates command-last from dataset-last cleanly.

**The reference docs ARE the specs.** Read the cited section before implementing each increment. The conformance gates
for DIMSE and DICOMweb live in the "Conformance scope and limits" sections of `docs/reference/dimse.md` and
`docs/reference/dicomweb.md` respectively (there is no separate `docs/conformance/dimse.md`); the DICOM transfer-syntax
floor is in `docs/conformance/dicom.md`.

---

## Increment overview (dependency-ordered)

The DIMSE leg leads, in strict dependency order (PDU → DUL → ACSE → C-ECHO → C-STORE), because it is the densest and
its correctness is the load-bearing fix for the prototype's interop failures. The remaining legs follow.

- **Increment 0 — Package scaffolding and interop harness.** Create the M2 package skeletons
  (`dimse/{pdu,dul,acse}`, root `dimse`, `dicomweb`, `hl7v2`, `fhir/r5`, `convert`, `server`), wire `mise`/`Makefile`
  test and interop targets, and port the prototype's Orthanc integration test as a (initially skipped) regression net.
  No production code beyond package declarations. *Outlined.*
- **Increment 1 — `dimse/pdu` PDU and PDV encode/decode.** The wire codec foundation: PDU header framing, the P-DATA-TF
  PDU, and the PDV message-control-header bits with bounds-checked decode. **Fully expanded into bite-sized TDD tasks
  below.** Port-with-fixes from `dimse/pdu`. Gate: unit + race + lint + a fuzz/hostile-input target.
- **Increment 2 — `dimse/dul` DUL state machine (rewrite) + association PDU codec.** The full PS3.8 Table 9-10 machine
  (13 states incl. Sta9–Sta12, 19 events Evt1–Evt19, 28 actions AE/DT/AR/AA), the ARTIM timer, the socket owner, and
  the A-ASSOCIATE-RQ/AC/RJ, A-RELEASE-RQ/RP, A-ABORT PDU codecs. *Outlined.*
- **Increment 3 — `dimse/acse` association establishment + presentation-context negotiation.** `AETitle`, `AE`,
  `PresentationContext`, the default transfer-syntax set, `VerificationContexts()` / `StorageContexts()`, and the
  `Associate` / accept negotiation that drives the DUL. *Outlined.*
- **Increment 4 — DIMSE typed status + C-ECHO (SCU + SCP).** `Status` / `ServiceClass` / `StatusCategory`, the named
  status constants, `Association.Echo`, and the SCP `Handler.Echo` dispatch. *Outlined.*
- **Increment 5 — DIMSE message layer + C-STORE (SCU + SCP).** Command-set build/parse, PDV fragmentation with the
  command-last/dataset-last fix and the reassembler state machine, `Association.Store`, and `Handler.Store`. This is
  where DIMSE-001/002/003/006/007 are closed. *Outlined.*
- **Increment 6 — `dimse.Server` SCP scaffolding.** `NewServer`, `ListenAndServe` (loopback default), `Shutdown`
  (closes connections first), capacity acquired before spawning the handler. *Outlined.*
- **Increment 7 — DIMSE interop gate (C-ECHO + C-STORE against Orthanc / dcm4chee-arc).** The ported integration test,
  un-skipped, proving SCU↔real-PACS and the last-fragment-bit fix end-to-end. *Outlined.*
- **Increment 8 — `dicomweb` DICOM-JSON + multipart/related codecs.** `MarshalJSON`/`UnmarshalJSON` over a
  `*dicom.DataSet` (PS3.18 Annex F) and the bounded `MultipartReader`/`MultipartWriter`. *Outlined.*
- **Increment 9 — DICOMweb STOW-RS store + WADO-RS read (client + thin server).** `ResourcePath`, `Client.Store` /
  `Client.RetrieveInstance`, the `StoreBackend`/`RetrieveBackend` server, plus the Orthanc interop gate. *Outlined.*
- **Increment 10 — `hl7v2` ORM parse (thin).** The generic parse tree, `EncodingCharacters` derivation, `Parse`, and
  just enough typed `ORM`/`ORC`/`OBR`/`PID` to feed the converter. *Outlined.*
- **Increment 11 — `fhir/r5` minimal hand-written resources.** Hand-written minimal `ServiceRequest`,
  `DiagnosticReport`, `ImagingStudy` (plus the `Reference`/`Identifier`/`CodeableConcept` datatypes and the root
  `fhir.Resource`/`Decimal`) — only the fields the skeleton needs, **not** the generator. *Outlined.*
- **Increment 12 — `convert` slice + end-to-end skeleton test.** `ORMToServiceRequestR5`, a minimal
  `SR/DiagnosticReport` producer, `DICOMToImagingStudyR5`, the `UIDIdentifierR5` helper and `Report` model, and the
  single end-to-end test that drives DICOM → DIMSE store → DICOMweb store/read → HL7 ORM → ServiceRequest →
  DiagnosticReport → ImagingStudy, proving the architecture. *Outlined.*

Increments 2 through 12 are outlined here (goal, files, key tests, reference-doc section, ports-vs-rewrite note, and
verification gate) and are expanded into bite-sized TDD tasks when reached, exactly as M1 fully expanded Increment 1
and outlined the rest.

---

## Increment 0 — Package scaffolding and interop harness

**Scope:** Stand up every M2 package as a compilable skeleton and wire the test/interop targets every later increment
relies on. Port the prototype's Orthanc integration test as a regression net (skipped until the DIMSE leg lands), so
the interop gate exists before the rewrite begins (PRD §12: "the Orthanc integration tests are ported first as a
regression net before the DIMSE rewrite"). No production logic.

**Reference-doc section:** PRD §7.1 (package layout), §11.1 (interop gates), §13 (M2 scope); dimse.md "Worked examples".

**Ports vs rewrite:** Port `dimse/integration/orthanc` and `integration_test.go` from the prototype (adapt to the new
package paths and the committed `dimse.AE`/`Associate`/`Echo`/`Store` API once those exist; until then keep the test
behind a `//go:build interop` tag and `t.Skip` so the suite stays green).

**Files:**
- Create: `dimse/pdu/doc.go`, `dimse/dul/doc.go`, `dimse/acse/doc.go`, `dimse/doc.go` (replace the placeholder),
  `dicomweb/doc.go` (replace), `hl7v2/doc.go` (replace), `fhir/doc.go` + `fhir/r5/doc.go`, `convert/doc.go` (replace),
  `server/doc.go` (replace) — each a one-paragraph package comment matching the reference doc's opening.
- Create: `dimse/integration/orthanc/orthanc.go` (testcontainers helper, ported), `dimse/integration/interop_test.go`
  (ported, `//go:build interop`, skipped).
- Modify: `mise.toml` — add `[tasks."test:dimse"]`, `[tasks."test:dicomweb"]`, `[tasks."test:hl7v2"]`,
  `[tasks."test:fhir"]`, `[tasks."test:convert"]`, `[tasks."test:skeleton"]` (race + cover), and
  `[tasks."interop:dimse"]` / `[tasks."interop:dicomweb"]` (run the `interop`-tagged tests with the containers).
- Modify: `Makefile` — mirror the targets for non-mise users.

**Key tests:** `go test ./dimse/... ./dicomweb/... ./hl7v2/... ./fhir/... ./convert/... ./server/...` compiles and
reports `no test files` (confirms wiring); `mise run interop:dimse` runs and the ported test `t.Skip`s cleanly with a
clear message when the DIMSE API does not yet exist or `interop` is not tagged.

**Verification gate:** `go build ./...` is green; the six `test:*` targets run; the interop targets run and skip
cleanly. Commit scaffolding, ported integration test, and the mise/make config as three separate atomic commits.

---

## Increment 1 — `dimse/pdu` PDU and PDV encode/decode (fully expanded)

**Scope:** The wire codec at the bottom of the DIMSE stack: the 6-byte PDU header framing shared by every PDU type, the
P-DATA-TF PDU carrying one or more Presentation Data Values, and the PDV one-byte message-control header whose command
bit (bit 0) and last-fragment bit (bit 1) are the load-bearing fix for the prototype's Orthanc aborts. This increment
delivers the encode/decode primitives and their bounds checks; the *fragmentation policy* that decides which PDV is
marked last lives in the DIMSE message layer (Increment 5), but the bit semantics and the underflow guard are fixed
here. This increment closes Codex DIMSE-004 (PDV length underflow) and lays the typed foundation that DIMSE-001/002
build on.

**Reference-doc section:** dimse.md "PDU and PDV" (the normative bit rules and the bounds-check requirements); PRD §9.3
(hostile-input hardening). The prototype sources to port-with-fixes are `/Users/codeninja/vcs/go-radx/dimse/pdu/pdu.go`
(`ReadPDU`, `writePDUHeader`, the PDU-type constants) and `/Users/codeninja/vcs/go-radx/dimse/pdu/data.go` (`DataTF`,
`PresentationDataValue`, the message-control constants, `encodePresentationDataValue`/`decodePresentationDataValue`).

**Porting note — representation changes from the prototype.** The prototype's `DataTF.Decode` read items "until EOF"
from a raw `io.Reader` with the body pre-read into a slice; this increment reads from an explicit **bounded reader**
seeded with the declared PDU-body length. **Two distinct guards are required, not one** (the lesson from the `dicom`
bounded reader, DCM-003/DCM-004): (a) a PDV item length exceeding the body *remaining* is a truncation, and (b) a PDV
payload or declared PDU length exceeding an **absolute `MaxPDULength` ceiling** is rejected before allocation. The
remaining-bytes check alone is insufficient because the bounded reader's remaining count is seeded from the declared
length read off the wire, which is attacker-controlled — so the prototype's `MaxPDULength = 16777215` (16 MiB) cap is
**kept** as the absolute allocation ceiling (enforced in `readHeader` and again before the PDV `make`), with the
negotiated Maximum Length enforced later at the association layer. The `MessageControlCommand`/
`MessageControlLastFragment` constant values (`0x01`/`0x02`) are kept; the prototype's ambiguous
`MessageControlDataset`/`MessageControlDatasetLast` aliases are dropped in favour of composing the two real bits
(command bit + last-fragment bit). A `DataTF` body with zero PDV items is a malformed PDU, rejected (PS3.8 §9.3.5), not
an empty success. The `pdu` package depends only on the standard library (no `dicom` types in this increment — the PDU
layer is pure bytes); it never imports `dul`, `acse`, or the root `dimse` (acyclic layering, dimse.md "Overview of the
layers"). It returns a local `*PDUError`.

**Transcription corrections (from execution):** the plan's Task 1.4 `errors.go` block declared an unused `import "fmt"`
(the `Error()` method uses string concatenation) — omit it; `fmt` is used in `data.go`, not `errors.go`. Increment 0
already created `dimse/pdu/doc.go` with the package doc comment, so `pdu.go` (Task 1.1) starts at `package pdu` without
re-declaring it (two package-level doc comments would trip `go vet`/lint).

**File structure:**
- `dimse/pdu/pdu.go` + `dimse/pdu/pdu_test.go` — PDU-type constants, `PDUType`, the 6-byte header read/write, and the
  bounded-reader helper.
- `dimse/pdu/bounded_reader.go` + `dimse/pdu/bounded_reader_test.go` — a small `boundedReader` that fails closed when a
  read would exceed the declared remaining bytes (mirrors the `dicom` package's bounded-reader discipline).
- `dimse/pdu/data.go` + `dimse/pdu/data_test.go` — `DataTF`, `PresentationDataValue`, the message-control bit
  constants and accessors, and the bounds-checked PDV encode/decode.
- `dimse/pdu/errors.go` — the `ProtocolError`-shaped typed error this package returns for malformed PDUs (the root
  `dimse.ProtocolError` is defined later; this package returns a local typed error the root wraps, to keep the layering
  acyclic — see the ordering note in Task 1.4).

---

### Task 1.1: PDU type constants and the 6-byte header

**Files:**
- Create: `dimse/pdu/pdu.go`
- Test: `dimse/pdu/pdu_test.go`

- [ ] **Step 1: Write the failing test**

```go
package pdu

import (
	"bytes"
	"testing"
)

func TestPDUTypeString(t *testing.T) {
	tests := map[PDUType]string{
		PDUTypeAssociateRQ: "A-ASSOCIATE-RQ",
		PDUTypeAssociateAC: "A-ASSOCIATE-AC",
		PDUTypeAssociateRJ: "A-ASSOCIATE-RJ",
		PDUTypeData:        "P-DATA-TF",
		PDUTypeReleaseRQ:   "A-RELEASE-RQ",
		PDUTypeReleaseRP:   "A-RELEASE-RP",
		PDUTypeAbort:       "A-ABORT",
	}
	for pt, want := range tests {
		if got := pt.String(); got != want {
			t.Errorf("PDUType(%#02x).String() = %q, want %q", byte(pt), got, want)
		}
	}
}

func TestWriteReadHeaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeHeader(&buf, PDUTypeData, 0x12345678); err != nil {
		t.Fatalf("writeHeader: %v", err)
	}
	// PDU header is 6 bytes: type(1) reserved(1) length(4 big-endian).
	if got := buf.Len(); got != 6 {
		t.Fatalf("header length = %d, want 6", got)
	}
	want := []byte{0x04, 0x00, 0x12, 0x34, 0x56, 0x78}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("header bytes = % x, want % x", buf.Bytes(), want)
	}
	pt, length, err := readHeader(&buf)
	if err != nil {
		t.Fatalf("readHeader: %v", err)
	}
	if pt != PDUTypeData || length != 0x12345678 {
		t.Errorf("readHeader = (%s, %#x), want (P-DATA-TF, 0x12345678)", pt, length)
	}
}

func TestReadHeaderRejectsUnknownType(t *testing.T) {
	r := bytes.NewReader([]byte{0x09, 0x00, 0x00, 0x00, 0x00, 0x00})
	if _, _, err := readHeader(r); err == nil {
		t.Error("readHeader should reject an unknown PDU type 0x09")
	}
}

func TestReadHeaderRejectsTruncated(t *testing.T) {
	// A header that ends mid-length must be io.ErrUnexpectedEOF, never a clean read.
	r := bytes.NewReader([]byte{0x04, 0x00, 0x12})
	if _, _, err := readHeader(r); err == nil {
		t.Error("readHeader should reject a truncated header")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/pdu/ -run TestPDU -v` and `go test ./dimse/pdu/ -run TestWriteReadHeader -v`
Expected: FAIL — `undefined: PDUType`, `writeHeader`, `readHeader`.

- [ ] **Step 3: Write minimal implementation**

```go
// Package pdu implements the DICOM Upper Layer Protocol Data Units (PS3.8 §9.3):
// the 6-byte PDU header framing, the P-DATA-TF PDU and its Presentation Data
// Values, and the association/release/abort PDUs. It owns no socket and knows
// nothing about DICOM messages — only PDU bytes (dimse.md "Overview of the
// layers"). All length math is bounds-checked before allocation (PRD §9.3).
package pdu

import (
	"encoding/binary"
	"fmt"
	"io"
)

// PDUType is the one-byte PDU type discriminator (PS3.8 §9.3.1, Table 9-11).
type PDUType byte

const (
	PDUTypeAssociateRQ PDUType = 0x01
	PDUTypeAssociateAC PDUType = 0x02
	PDUTypeAssociateRJ PDUType = 0x03
	PDUTypeData        PDUType = 0x04 // P-DATA-TF
	PDUTypeReleaseRQ   PDUType = 0x05
	PDUTypeReleaseRP   PDUType = 0x06
	PDUTypeAbort       PDUType = 0x07
)

var pduTypeNames = map[PDUType]string{
	PDUTypeAssociateRQ: "A-ASSOCIATE-RQ",
	PDUTypeAssociateAC: "A-ASSOCIATE-AC",
	PDUTypeAssociateRJ: "A-ASSOCIATE-RJ",
	PDUTypeData:        "P-DATA-TF",
	PDUTypeReleaseRQ:   "A-RELEASE-RQ",
	PDUTypeReleaseRP:   "A-RELEASE-RP",
	PDUTypeAbort:       "A-ABORT",
}

// String renders the registered PDU name, never bare hex (PRD §8.2).
func (pt PDUType) String() string {
	if name, ok := pduTypeNames[pt]; ok {
		return name
	}
	return fmt.Sprintf("unknown-PDU(0x%02X)", byte(pt))
}

func (pt PDUType) valid() bool {
	_, ok := pduTypeNames[pt]
	return ok
}

// writeHeader writes the PDU type, the reserved byte (0x00), and the big-endian
// 4-byte body length (PS3.8 §9.3.1). The body follows; this writes only the header.
func writeHeader(w io.Writer, pt PDUType, length uint32) error {
	var h [6]byte
	h[0] = byte(pt)
	h[1] = 0x00
	binary.BigEndian.PutUint32(h[2:], length)
	_, err := w.Write(h[:])
	if err != nil {
		return fmt.Errorf("pdu: write %s header: %w", pt, err)
	}
	return nil
}

// readHeader reads and validates a 6-byte PDU header, returning the type and the
// declared body length. An unknown type is rejected; a short read surfaces as
// io.ErrUnexpectedEOF (truncation is failure, PRD §9.2).
func readHeader(r io.Reader) (PDUType, uint32, error) {
	var h [6]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		if err == io.EOF {
			return 0, 0, err // clean EOF at a PDU boundary
		}
		return 0, 0, fmt.Errorf("pdu: read header: %w", err)
	}
	pt := PDUType(h[0])
	if !pt.valid() {
		return 0, 0, fmt.Errorf("pdu: unrecognised PDU type 0x%02X", h[0])
	}
	length := binary.BigEndian.Uint32(h[2:])
	return pt, length, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/pdu/ -run 'TestPDU|TestWriteReadHeader|TestReadHeader' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/pdu/pdu.go dimse/pdu/pdu_test.go
git commit -m "feat(dimse/pdu): add PDU type constants and 6-byte header codec"
```

---

### Task 1.2: Bounded reader for PDU bodies

**Files:**
- Create: `dimse/pdu/bounded_reader.go`
- Test: `dimse/pdu/bounded_reader_test.go`

The PDV decoder must never allocate against a length that exceeds the bytes actually remaining in the PDU body. This
task provides the bounded reader the decoder reads through, so a hostile P-DATA-TF declaring a multi-gigabyte PDV item
length is rejected against real bytes, not against a constant ceiling (Codex DIMSE-004; PRD §9.3). It mirrors the
discipline of the `dicom` package's `boundedReader`.

- [ ] **Step 1: Write the failing test**

```go
package pdu

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestBoundedReaderEnforcesRemaining(t *testing.T) {
	br := newBoundedReader(bytes.NewReader([]byte("abcdef")), 4)
	got := make([]byte, 4)
	if _, err := io.ReadFull(br, got); err != nil {
		t.Fatalf("ReadFull within bound: %v", err)
	}
	if string(got) != "abcd" {
		t.Errorf("read %q, want abcd", got)
	}
	// A further read past the declared bound is EOF, not the underlying "ef".
	if _, err := br.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Errorf("read past bound = %v, want io.EOF", err)
	}
}

func TestBoundedReaderRejectsOverLongAllocation(t *testing.T) {
	br := newBoundedReader(bytes.NewReader([]byte("ab")), 2)
	// A caller asking whether it may allocate n must consult Remaining first.
	if br.Remaining() != 2 {
		t.Fatalf("Remaining() = %d, want 2", br.Remaining())
	}
	if br.CanRead(3) {
		t.Error("CanRead(3) should be false when only 2 bytes remain")
	}
	if !br.CanRead(2) {
		t.Error("CanRead(2) should be true when 2 bytes remain")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/pdu/ -run TestBoundedReader -v`
Expected: FAIL — `undefined: newBoundedReader`.

- [ ] **Step 3: Write minimal implementation**

```go
package pdu

import "io"

// boundedReader caps reads at a declared number of remaining bytes so a length
// read from the wire can be validated against bytes actually present before any
// allocation (PRD §9.3). It fails closed: reads past the bound return io.EOF.
type boundedReader struct {
	r         io.Reader
	remaining int64
}

func newBoundedReader(r io.Reader, n int64) *boundedReader {
	return &boundedReader{r: r, remaining: n}
}

func (b *boundedReader) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.r.Read(p)
	b.remaining -= int64(n)
	return n, err
}

// Remaining reports the bytes still readable within the bound.
func (b *boundedReader) Remaining() int64 { return b.remaining }

// CanRead reports whether n bytes may be read without exceeding the bound. The
// PDV decoder calls this before make([]byte, n) (Codex DIMSE-004).
func (b *boundedReader) CanRead(n int64) bool { return n >= 0 && n <= b.remaining }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/pdu/ -run TestBoundedReader -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/pdu/bounded_reader.go dimse/pdu/bounded_reader_test.go
git commit -m "feat(dimse/pdu): add bounded reader for fail-closed PDU body decode"
```

---

### Task 1.3: PDV message-control header bits and accessors

**Files:**
- Create: `dimse/pdu/data.go`
- Test: `dimse/pdu/data_test.go`

This task fixes the prototype's conflation of the command-last and dataset-last boundaries by modelling the two real
bits independently. Every PDV carries a one-byte message-control header: **bit 0** is the command bit (1 = command,
0 = dataset) and **bit 1** is the last-fragment bit (dimse.md "PDU and PDV"). The four legal headers are therefore
`0x00` (dataset, more follow), `0x01` (command, more follow), `0x02` (dataset, last), and `0x03` (command, last). The
prototype defined `MessageControlDataset = 0x00` and `MessageControlDatasetLast = 0x02` as if dataset-last and
command-last were the same bit, which made it impossible to express "this is the final command fragment **and** a
dataset follows" — the `0x03`-then-dataset case Orthanc requires (Codex DIMSE-001, fully fixed in Increment 5).

- [ ] **Step 1: Write the failing test**

```go
package pdu

import "testing"

func TestPDVControlBits(t *testing.T) {
	cases := []struct {
		header           byte
		isCommand, isLast bool
	}{
		{0x00, false, false}, // dataset, more fragments
		{0x01, true, false},  // command, more fragments
		{0x02, false, true},  // dataset, last
		{0x03, true, true},   // command, last (the DIMSE-001 case)
	}
	for _, c := range cases {
		pdv := PresentationDataValue{MessageControlHeader: c.header}
		if pdv.IsCommand() != c.isCommand {
			t.Errorf("header %#02x IsCommand() = %v, want %v", c.header, pdv.IsCommand(), c.isCommand)
		}
		if pdv.IsLastFragment() != c.isLast {
			t.Errorf("header %#02x IsLastFragment() = %v, want %v", c.header, pdv.IsLastFragment(), c.isLast)
		}
	}
}

func TestMakeControlHeader(t *testing.T) {
	// A final command fragment is 0x03 regardless of whether a dataset follows.
	if got := MakeControlHeader(true, true); got != 0x03 {
		t.Errorf("MakeControlHeader(command=true, last=true) = %#02x, want 0x03", got)
	}
	if got := MakeControlHeader(false, true); got != 0x02 {
		t.Errorf("MakeControlHeader(command=false, last=true) = %#02x, want 0x02", got)
	}
	if got := MakeControlHeader(true, false); got != 0x01 {
		t.Errorf("MakeControlHeader(command=true, last=false) = %#02x, want 0x01", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/pdu/ -run 'TestPDVControlBits|TestMakeControlHeader' -v`
Expected: FAIL — `undefined: PresentationDataValue`, `MakeControlHeader`.

- [ ] **Step 3: Write minimal implementation**

```go
package pdu

// Message-control header bits (PS3.8 §9.3.5.1). Bit 0 is the command/dataset bit;
// bit 1 is the last-fragment bit. They are independent: a final command fragment
// is 0x03 whether or not a dataset follows (Codex DIMSE-001).
const (
	controlCommandBit byte = 0x01 // bit 0: 1 = command, 0 = dataset
	controlLastBit    byte = 0x02 // bit 1: 1 = last fragment of this command/dataset
)

// PresentationDataValue is one PDV inside a P-DATA-TF PDU: a presentation context
// ID, a one-byte message-control header, and the fragment payload.
type PresentationDataValue struct {
	PresentationContextID uint8
	MessageControlHeader  byte
	Data                  []byte
}

// IsCommand reports whether the PDV carries command-set bytes (bit 0 set).
func (p PresentationDataValue) IsCommand() bool { return p.MessageControlHeader&controlCommandBit != 0 }

// IsLastFragment reports whether this is the last fragment of its command or
// dataset (bit 1 set).
func (p PresentationDataValue) IsLastFragment() bool {
	return p.MessageControlHeader&controlLastBit != 0
}

// MakeControlHeader composes a message-control header from the two independent
// bits. The DIMSE message layer (Increment 5) uses this so the final command
// fragment is always 0x03 and the final dataset fragment 0x02.
func MakeControlHeader(command, last bool) byte {
	var h byte
	if command {
		h |= controlCommandBit
	}
	if last {
		h |= controlLastBit
	}
	return h
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/pdu/ -run 'TestPDVControlBits|TestMakeControlHeader' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/pdu/data.go dimse/pdu/data_test.go
git commit -m "feat(dimse/pdu): model PDV command and last-fragment bits independently"
```

---

### Task 1.4: PDV encode and bounds-checked decode

**Files:**
- Modify: `dimse/pdu/data.go` (append the PDV item codec)
- Create: `dimse/pdu/errors.go`
- Test: `dimse/pdu/data_test.go` (append)

This is the core hostile-input fix. A PDV item on the wire is a 4-byte big-endian item length, then the 1-byte
presentation-context ID, then the 1-byte message-control header, then the payload; the item length counts the two
header bytes plus the payload. The prototype decoded `dataLength := length - 2` with no check that `length >= 2`, so a
PDV declaring item length `0` or `1` underflowed `uint32` to a near-4-GB `make([]byte, dataLength)` (Codex DIMSE-004).
This task rejects `length < 2` before the subtraction and validates the payload length against the bounded reader's
remaining bytes before allocating.

**Ordering note — keep the layering acyclic.** The root `dimse` package defines the public `ProtocolError` (dimse.md
"Error model"). The `pdu` package must not import `dimse` (that would invert the dependency). So `pdu` returns its own
`*PDUError` here (in `errors.go`), and the root `dimse` package wraps or translates it into `*ProtocolError` when it
surfaces to callers. Do not reference `dimse.ProtocolError` from this package.

- [ ] **Step 1: Write the failing test (append to `data_test.go`)**

```go
func TestPDVEncodeDecodeRoundTrip(t *testing.T) {
	pdv := PresentationDataValue{
		PresentationContextID: 1,
		MessageControlHeader:  MakeControlHeader(true, true), // 0x03
		Data:                  []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	var buf bytes.Buffer
	if err := encodePDV(&buf, pdv); err != nil {
		t.Fatalf("encodePDV: %v", err)
	}
	// item length = 2 header bytes + 4 payload = 6, big-endian.
	want := []byte{0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0xDE, 0xAD, 0xBE, 0xEF}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("encodePDV bytes = % x, want % x", buf.Bytes(), want)
	}
	br := newBoundedReader(&buf, int64(buf.Len()))
	got, err := decodePDV(br)
	if err != nil {
		t.Fatalf("decodePDV: %v", err)
	}
	if got.PresentationContextID != 1 || got.MessageControlHeader != 0x03 ||
		!bytes.Equal(got.Data, pdv.Data) {
		t.Errorf("decodePDV = %+v, want round-trip of %+v", got, pdv)
	}
}

// TestPDVDecodeRejectsUnderflowLength guards Codex DIMSE-004: an item length below
// the 2-byte header must be rejected before the length-2 subtraction, never
// underflow into a giant allocation.
func TestPDVDecodeRejectsUnderflowLength(t *testing.T) {
	for _, badLen := range []uint32{0, 1} {
		var raw bytes.Buffer
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], badLen)
		raw.Write(hdr[:])
		br := newBoundedReader(&raw, int64(raw.Len()))
		_, err := decodePDV(br)
		if err == nil {
			t.Fatalf("decodePDV(item length %d) = nil error, want rejection", badLen)
		}
		var pe *PDUError
		if !errors.As(err, &pe) {
			t.Errorf("decodePDV(item length %d) error = %T, want *PDUError", badLen, err)
		}
	}
}

// TestPDVDecodeRejectsLengthBeyondBody guards against a PDV item length larger than
// the bytes remaining in the PDU body.
func TestPDVDecodeRejectsLengthBeyondBody(t *testing.T) {
	var raw bytes.Buffer
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], 1000) // claims 1000 bytes...
	raw.Write(hdr[:])
	raw.Write([]byte{0x01, 0x03, 0x00}) // ...but only 3 follow
	br := newBoundedReader(&raw, int64(raw.Len()))
	if _, err := decodePDV(br); err == nil {
		t.Error("decodePDV should reject an item length exceeding the bytes remaining")
	}
}
```

Add the imports `"encoding/binary"` and `"errors"` to the test file's import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/pdu/ -run TestPDV -v`
Expected: FAIL — `undefined: encodePDV`, `decodePDV`, `PDUError`.

- [ ] **Step 3: Write minimal implementation**

`dimse/pdu/errors.go`:

```go
package pdu

import "fmt"

// PDUError reports a malformed PDU or PDV: a length-limit violation, an underflow
// item length, or a truncated read. It names the violated constraint without PHI.
// The root dimse package translates it into a public *ProtocolError; pdu must not
// import dimse (acyclic layering, dimse.md "Overview of the layers").
type PDUError struct {
	Detail string
}

func (e *PDUError) Error() string { return "dimse/pdu: " + e.Detail }
```

Append to `dimse/pdu/data.go`:

```go
import (
	"encoding/binary"
	"fmt"
	"io"
)

// pdvHeaderLen is the PDV item-header size counted inside the item length: the
// 1-byte presentation-context ID plus the 1-byte message-control header.
const pdvHeaderLen = 2

// encodePDV writes one PDV item: a 4-byte big-endian item length (header + payload),
// the context ID, the message-control header, then the payload.
func encodePDV(w io.Writer, pdv PresentationDataValue) error {
	itemLen := uint32(pdvHeaderLen + len(pdv.Data))
	var hdr [6]byte
	binary.BigEndian.PutUint32(hdr[0:4], itemLen)
	hdr[4] = pdv.PresentationContextID
	hdr[5] = pdv.MessageControlHeader
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("pdu: write PDV header: %w", err)
	}
	if _, err := w.Write(pdv.Data); err != nil {
		return fmt.Errorf("pdu: write PDV payload: %w", err)
	}
	return nil
}

// decodePDV reads one PDV item from a bounded reader. It rejects an item length
// below the 2-byte header BEFORE subtracting (Codex DIMSE-004) and validates the
// payload length against the bytes remaining before allocation (PRD §9.3).
func decodePDV(br *boundedReader) (PresentationDataValue, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(br, lenBuf[:]); err != nil {
		return PresentationDataValue{}, err
	}
	itemLen := binary.BigEndian.Uint32(lenBuf[:])
	if itemLen < pdvHeaderLen {
		return PresentationDataValue{}, &PDUError{
			Detail: fmt.Sprintf("PDV item length %d below header size %d", itemLen, pdvHeaderLen),
		}
	}
	payloadLen := int64(itemLen - pdvHeaderLen)
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return PresentationDataValue{}, err
	}
	if !br.CanRead(payloadLen) {
		return PresentationDataValue{}, &PDUError{
			Detail: fmt.Sprintf("PDV payload length %d exceeds %d bytes remaining in PDU body",
				payloadLen, br.Remaining()),
		}
	}
	data := make([]byte, payloadLen)
	if _, err := io.ReadFull(br, data); err != nil {
		return PresentationDataValue{}, err
	}
	return PresentationDataValue{
		PresentationContextID: hdr[0],
		MessageControlHeader:  hdr[1],
		Data:                  data,
	}, nil
}
```

(If `data.go` already imports `fmt`/`io` from earlier tasks, merge rather than duplicate the import block.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/pdu/ -run TestPDV -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/pdu/data.go dimse/pdu/errors.go dimse/pdu/data_test.go
git commit -m "feat(dimse/pdu): add bounds-checked PDV encode/decode (guards DIMSE-004)"
```

---

### Task 1.5: P-DATA-TF PDU encode and decode

**Files:**
- Modify: `dimse/pdu/data.go` (append `DataTF`)
- Test: `dimse/pdu/data_test.go` (append)

A P-DATA-TF PDU is the 6-byte header (type `0x04`) followed by one or more PDV items. This task assembles the PDV codec
from Task 1.4 with the header from Task 1.1 into the full PDU, decoding through a bounded reader seeded with the
declared body length so the sum of PDV item lengths cannot exceed the PDU body.

- [ ] **Step 1: Write the failing test (append to `data_test.go`)**

```go
func TestDataTFRoundTrip(t *testing.T) {
	in := &DataTF{Items: []PresentationDataValue{
		{PresentationContextID: 1, MessageControlHeader: 0x01, Data: []byte("cmd")},
		{PresentationContextID: 1, MessageControlHeader: 0x03, Data: []byte("end")},
	}}
	var buf bytes.Buffer
	if err := in.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if PDUType(buf.Bytes()[0]) != PDUTypeData {
		t.Fatalf("first byte = %#02x, want P-DATA-TF type 0x04", buf.Bytes()[0])
	}
	pt, length, err := readHeader(&buf)
	if err != nil || pt != PDUTypeData {
		t.Fatalf("readHeader = (%s, %d), %v", pt, length, err)
	}
	out := &DataTF{}
	if err := out.Decode(newBoundedReader(&buf, int64(length))); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out.Items) != 2 || string(out.Items[0].Data) != "cmd" || string(out.Items[1].Data) != "end" {
		t.Errorf("Decode items = %+v, want round-trip", out.Items)
	}
	if !out.Items[1].IsLastFragment() {
		t.Error("last item should be marked last-fragment")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/pdu/ -run TestDataTF -v`
Expected: FAIL — `undefined: DataTF` / `(*DataTF).Encode` / `(*DataTF).Decode`.

- [ ] **Step 3: Write minimal implementation (append to `data.go`)**

```go
import "bytes"

// DataTF is a P-DATA-TF PDU: one or more PDV items carrying command and dataset
// fragments (PS3.8 §9.3.5).
type DataTF struct {
	Items []PresentationDataValue
}

// Encode writes the P-DATA-TF PDU: the 6-byte header (with the summed item length)
// followed by each PDV item.
func (p *DataTF) Encode(w io.Writer) error {
	var body bytes.Buffer
	for _, item := range p.Items {
		if err := encodePDV(&body, item); err != nil {
			return err
		}
	}
	if err := writeHeader(w, PDUTypeData, uint32(body.Len())); err != nil {
		return err
	}
	_, err := w.Write(body.Bytes())
	return err
}

// Decode reads PDV items from a bounded reader seeded with the PDU body length, so
// the items cannot collectively exceed the declared body (PRD §9.3).
func (p *DataTF) Decode(br *boundedReader) error {
	for br.Remaining() > 0 {
		item, err := decodePDV(br)
		if err != nil {
			return err
		}
		p.Items = append(p.Items, item)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/pdu/ -run TestDataTF -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/pdu/data.go dimse/pdu/data_test.go
git commit -m "feat(dimse/pdu): add P-DATA-TF PDU encode/decode over bounded body"
```

---

### Task 1.6: Hostile-input fuzz target for the PDV/PDU decoder

**Files:**
- Create: `dimse/pdu/fuzz_test.go`

PRD §9.3 requires fuzz/crash tests for binary parsers and they may not be skipped. This task adds a Go native fuzz
target over `decodePDV` / `DataTF.Decode` proving no malformed P-DATA-TF input panics or hangs, complementing the
explicit underflow regression in Task 1.4. Port the prototype's `dimse/pdu/fuzz_test.go` corpus seeds.

- [ ] **Step 1: Write the fuzz target**

```go
package pdu

import (
	"bytes"
	"testing"
)

func FuzzDecodePDV(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0xDE, 0xAD, 0xBE, 0xEF})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00}) // underflow item length
	f.Add([]byte{0x00, 0x00, 0x00, 0x01}) // underflow item length
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF}) // huge declared length, no body
	f.Fuzz(func(t *testing.T, data []byte) {
		br := newBoundedReader(bytes.NewReader(data), int64(len(data)))
		// Must never panic; an error is the acceptable outcome for malformed input.
		_, _ = decodePDV(br)
	})
}
```

- [ ] **Step 2: Run the fuzz target briefly**

Run: `go test ./dimse/pdu/ -run FuzzDecodePDV -fuzz FuzzDecodePDV -fuzztime 20s`
Expected: PASS with no crash; then run the seed corpus as a normal test: `go test ./dimse/pdu/ -run FuzzDecodePDV`.

- [ ] **Step 3: Commit**

```bash
git add dimse/pdu/fuzz_test.go
git commit -m "test(dimse/pdu): add hostile-input fuzz target for PDV decode"
```

---

**Increment 1 verification gate:** `go test -race ./dimse/pdu/...` is green; `golangci-lint run ./dimse/pdu/...` is
clean; the underflow regression (`TestPDVDecodeRejectsUnderflowLength`) and the fuzz seed corpus pass. The `pdu`
package compiles standalone and imports neither `dul` nor the root `dimse` (acyclic layering verified by `go build`).

---

## Increment 2 — `dimse/dul` DUL state machine (rewrite) + association PDU codec

**Goal:** Replace the prototype's broken DUL with a faithful, table-driven PS3.8 Table 9-10 implementation: the 13
states `Sta1..Sta13` **including the release-collision states Sta9–Sta12** the prototype omitted, the 19 events
`Evt1..Evt19` (named per the standard, not the prototype's `AE1..AE19` mislabelling), and all 28 actions in the four
families — association establishment (AE-1…AE-8), data transfer (DT-1, DT-2), association release (AR-1…AR-10, carrying
the Sta9–Sta12 transitions), and abort (AA-1…AA-8). Crucially, **Evt19 (unrecognised or invalid PDU received) drives
AA-8** (send A-ABORT with provider source, issue A-P-ABORT, start ARTIM) rather than the prototype's silent socket
close (Codex DIMSE-010/011). The DUL owns the socket, the ARTIM timer, and PDU framing via `dimse/pdu`; it knows
nothing about DICOM messages. This increment also ports-with-fixes the association/release/abort PDU codecs
(A-ASSOCIATE-RQ/AC/RJ from `associate.go`, A-RELEASE-RQ/RP from `release.go`, A-ABORT from `abort.go`), fixing
DIMSE-008 (a rejected presentation context must still encode exactly one insignificant transfer-syntax sub-item).

**Files:** `dimse/dul/state.go` (the `State` enum + `String`), `dimse/dul/event.go` (`Evt1..Evt19`),
`dimse/dul/action.go` (the 28 action identifiers), `dimse/dul/statemachine.go` (the transition table keyed by
`(State, Evt)` → `(Action, State)`, table-driven so it is auditable against PS3.8 Table 9-10), `dimse/dul/artim.go`
(the ARTIM timer), `dimse/dul/connection.go` (the socket owner, `context.Context`-aware read/write of PDUs through
`dimse/pdu`), plus the associate/release/abort PDU codecs `dimse/pdu/associate.go`, `dimse/pdu/release.go`,
`dimse/pdu/abort.go` (these live in the `pdu` package, ported-with-fixes, but are exercised by `dul`). Tests:
`statemachine_test.go` (a table-driven test asserting every documented PS3.8 transition, the release-collision paths,
and that Evt19 in any data-transfer/awaiting state yields AA-8 — the DIMSE-011 regression), `associate_test.go`
(round-trip of A-ASSOCIATE-RQ/AC with presentation contexts; the DIMSE-008 rejected-context-keeps-one-TS regression),
fuzz targets over each association PDU decoder.

**Key tests:** every `(state, event)` pair in PS3.8 Table 9-10 maps to the documented `(action, next-state)`; an
unexpected PDU in `Sta6` produces AA-8 not a silent close (DIMSE-011 regression); a release collision drives
Sta7→Sta9/Sta8→Sta12 correctly (the states the prototype lacked); A-ASSOCIATE-RQ/AC round-trips with the four default
transfer syntaxes; a rejected presentation context in an A-ASSOCIATE-AC encodes exactly one transfer-syntax sub-item
(DIMSE-008 regression).

**Reference-doc section:** dimse.md "The DUL state machine (PS3.8 Table 9-10)" and "PDU and PDV" (the rejected-context
sub-item rule); PS3.8 Table 9-10 itself; PRD §9.3 (hostile PDU input). **Rewrite** of `dimse/dul`; **port-with-fixes**
of the `dimse/pdu` associate/release/abort codecs.

**Verification gate:** `go test -race ./dimse/dul/... ./dimse/pdu/...` green; lint clean; the DIMSE-011 (Evt19→AA-8)
and DIMSE-008 (rejected-context TS sub-item) regressions pass; association-PDU fuzz targets run without panic. No
interop yet (no end-to-end socket flow until Increment 3).

---

## Increment 3 — `dimse/acse` association establishment + presentation-context negotiation

**Goal:** Build the ACSE layer that negotiates an association and exposes the `Association` callers use: the `AETitle`
value type with `ParseAETitle` (length 1–16, allowed repertoire), the `AE` factory with its functional-option config
and the pynetdicom-default timeouts (`WithACSETimeout` 30s, `WithDIMSETimeout` 30s, `WithNetworkTimeout` 60s,
`WithMaxPDULength` 16382), the `PresentationContext` type and `NewPresentationContext`, the `DefaultTransferSyntaxes`
literal (the four uncompressed/deflated syntaxes, Explicit VR LE leading), the `MaxPDULength` type (with `0` = unlimited
handling, not a literal allocation size — Codex DIMSE-005), and the two presets the skeleton needs:
`VerificationContexts()` (1 context) and `StorageContexts()` (the validated radiology Storage set). `AE.Associate`
drives the DUL (Increment 2) through A-ASSOCIATE-RQ → accepted/rejected/aborted, returning an established `*Association`
or a typed `*AssociationError`; the acceptor-side negotiation (matching proposed contexts against supported ones) is
also built here so the SCP (Increment 6) can reuse it.

**Files:** `dimse/aetitle.go`, `dimse/ae.go` (the `AE`, `aeConfig`, `AEOption`s), `dimse/association.go` (the
`Association` type, `State()`, `AcceptedContexts()`, `Release()`, `Abort()`, the unestablished-association guard for
DIMSE-017), `dimse/presentation.go` (`PresentationContext`, `ContextResult`, `NewPresentationContext`,
`DefaultTransferSyntaxes`, `MaxPDULength`), `dimse/presets.go` (`VerificationContexts`, `StorageContexts`), and
`dimse/acse/negotiate.go` (the propose/accept matching logic driving the DUL). Tests: `aetitle_test.go` (1–16 length
and repertoire validation), `presets_test.go` (the context counts match dimse.md: Verification = 1, Storage = the
validated set), `association_test.go` (Associate/Release round-trip against an in-process loopback DUL pair; a
double-Release or operation-on-released returns a typed error not a panic — DIMSE-017), `negotiate_test.go` (an
abstract syntax not supported is rejected with the correct `ContextResult`; max-PDU `0` negotiates as unlimited).

**Key tests:** `ParseAETitle("")` and a 17-char title fail; `DefaultTransferSyntaxes` is exactly the four committed
UIDs in the committed order; an in-process Associate→Release completes via the DUL and leaves both sides in Sta1; an
operation on a released association returns `*ProtocolError`/`*AssociationError`, never panics (DIMSE-017 regression);
`MaxPDULength(0)` is treated as unlimited with the configured local send cap (DIMSE-005 regression).

**Reference-doc section:** dimse.md "Core value types" (`AETitle`), "Application Entity", "Negotiation primitives"
(`MaxPDULength`), "Presentation contexts and presets", "Default transfer syntaxes". **New** (the ACSE layer the
prototype conflated into `dul`); it reuses the **ported** association PDU codecs from Increment 2.

**Verification gate:** `go test -race ./dimse/... ./dimse/acse/...` green (in-process loopback, no real PACS yet); lint
clean; DIMSE-005 and DIMSE-017 regressions pass.

---

## Increment 4 — DIMSE typed status + C-ECHO (SCU + SCP)

**Goal:** The typed `Status` model and the first complete service operation. Build `Status` (wrapping only the 16-bit
code, with `Category()`/`Meaning()` derived against the `ServiceClass` it was constructed with — never settable fields,
so a caller cannot author a status whose category contradicts its code), the `ServiceClass` and `StatusCategory` enums,
the per-class categorisation tables (general and Verification suffice for C-ECHO; Storage is added in Increment 5), and
the named constants (`StatusSuccess`, `StatusEchoSuccess`). Then `Association.Echo(ctx)` (SCU: build the C-ECHO-RQ
command set, send via the message layer, await the C-ECHO-RSP, return the peer's `Status`) and the SCP side: the
`Handler` interface's `Echo` method, the `OpInfo` no-PHI context, and the server dispatch that answers a received
C-ECHO-RQ with `Handler.Echo`'s status. The DIMSE message layer's command-set encode is first exercised here, but the
C-ECHO command set carries no dataset, so the full fragmentation fix is proven in Increment 5.

**Files:** `dimse/command.go` (the minimal `CommandSet`/`CommandField` foundation + Implicit-VR command-dataset encode),
`dimse/status.go` (+ the categorisation tables), `dimse/status_test.go`, `dimse/echo.go` (`Association.Echo`),
`dimse/handler.go` (the `Handler`/`StoreHandler`/`OpInfo` types), `dimse/echo_test.go`.

**Command-set ordering (reviewer M2 — pinned, not executor's discretion):** C-ECHO sends a C-ECHO-RQ command set, so
this increment cannot build only on Increments 1–3; its **first tasks** build the minimal `CommandSet`/`CommandField`
constants and Implicit-VR command-dataset encode (a C-ECHO-RQ/RSP carries no dataset, so no fragmentation is exercised
here). The load-bearing C-STORE command-set fixes land in Increment 5 where the C-STORE command set needs them:
DIMSE-006/007 — command elements built in increasing tag order, Command Group Length `(0000,0000)` encoded last, and
Move Destination written with VR `AE`. Increment 4's encode need only produce a well-formed C-ECHO command set.

**Key tests:** `StatusEchoSuccess.IsSuccess()` is true and `StatusStoreCannotUnderstand.Category()` is `Failure`
(category derived from code+service class); `Status.String()` renders "0x0000 Success" style, never bare hex; an
in-process SCU↔SCP C-ECHO over the loopback association returns `StatusEchoSuccess`; an unknown status code resolves to
`StatusCategoryUnknown` with the code preserved, never coerced to success.

**Reference-doc section:** dimse.md "Typed status", "### C-ECHO", "SCP handlers and the event model", and the worked
example "### C-ECHO (verification)". **New** (typed status; the prototype used bare `0x0000` comparisons);
**rewrite** of the C-ECHO path (the prototype's was in `dimse/scu` and `dimse/scp`, both REWRITE verdicts, though the
SCU request structure is a useful port reference).

**Verification gate:** `go test -race ./dimse/...` green; lint clean; in-process C-ECHO passes both directions. (The
real-PACS C-ECHO is folded into the Increment 7 interop gate.)

---

## Increment 5 — DIMSE message layer + C-STORE (SCU + SCP)

**Goal:** The DIMSE message layer (PS3.7) and C-STORE — where the prototype's foundational defects are closed. Build
the `CommandSet` (the command elements built in **increasing tag order** with Command Group Length `(0000,0000)`
computed **last**, each element using its **dictionary VR** so Move Destination `(0000,0600)` is VR `AE` not `UI` —
Codex DIMSE-006/007), the command-set encode/decode (always Implicit VR LE per PS3.7), and the PDV fragmentation that
**marks the final command fragment last (`0x03`) independently of whether a dataset follows** (the
`pdu.MakeControlHeader(command, last)` from Increment 1.3) — closing Codex DIMSE-001, the concrete Orthanc-abort root
cause recorded in PRD line 60. Build the reassembler as a **state machine** that collects command fragments until
command-last, decodes `CommandDataSetType (0000,0800)`, and only then waits for dataset-last when a dataset is actually
present (Codex DIMSE-002), decoding the dataset with the **negotiated transfer syntax** of the presentation context,
not hard-coded Implicit VR LE (Codex DIMSE-003). Then `Association.Store(ctx, ds, opts...)` (SCU: select the
presentation context from the dataset's SOP Class UID and transfer syntax; if none matches, return a typed error and
transmit nothing — never report success on work not done, PRD §9.2) and the SCP `Handler.Store` dispatch with the
fail-closed rule (a handler returning success without storing is a defect).

**Files:** `dimse/command.go` (`CommandSet`, `CommandField` constants, `CommandDataSetType`, encode/decode in tag
order with group length last), `dimse/command_test.go`, `dimse/message.go` (the fragmentation and reassembler state
machine), `dimse/message_test.go`, `dimse/store.go` (`Association.Store`, `StoreOption`), `dimse/store_test.go`. Port
the *structure* of the prototype's `dimse/dimse/message.go` and `dimse/scu/client.go` `Store`, but **rewrite** the
fragmentation and reassembly per the fixes above.

**Key tests (the load-bearing regressions):**
- `TestCommandLastBitSetIndependentOfDataset` — encoding a C-STORE-RQ command set followed by a dataset produces a
  final command PDV with header `0x03` (command + last) *and then* dataset PDVs; the command-last bit does not wait
  for the dataset (Codex DIMSE-001 regression, the named regression PRD line 60 requires).
- `TestReassemblerGatesOnCommandLast` — the reassembler does not treat a dataset as present until command-last is seen
  and `CommandDataSetType` says a dataset follows (Codex DIMSE-002).
- `TestDatasetDecodedWithNegotiatedTransferSyntax` — a dataset sent under Explicit VR LE is decoded as Explicit VR LE,
  not Implicit (Codex DIMSE-003).
- `TestCommandSetTagOrderAndGroupLength` — command elements are in increasing tag order and `(0000,0000)` is computed
  last; `(0000,0600)` Move Destination uses VR `AE` (Codex DIMSE-006/007).
- `TestStoreNoMatchingContextTransmitsNothing` — `Store` with no accepted context for the SOP Class returns a typed
  error and sends no PDU (PRD §9.2 fail-closed).
- In-process SCU↔SCP C-STORE of a vendored `.dcm` round-trips and the SCP persists it, returning `StatusStoreSuccess`.

**Reference-doc section:** dimse.md "PDU and PDV" (the four fix bullets), "### C-STORE", "SCP handlers and the event
model", worked examples "### C-STORE (storage)" and "### Serving a Storage SCP". **Rewrite** of `dimse/dimse` (the
message layer); the SCU `Store` and SCP `Store` are rewrites (REWRITE verdict for `dimse/scp`, the SCU `Store`
structure ported from `dimse/scu`).

**Verification gate:** `go test -race ./dimse/...` green; lint clean; all five named DIMSE regressions pass; in-process
C-STORE round-trips a real fixture. This increment's correctness is proven end-to-end against a real PACS in
Increment 7.

---

## Increment 6 — `dimse.Server` SCP scaffolding

**Goal:** The embeddable SCP server that hosts the `Handler` (Increment 4/5). Build `NewServer(ae, supported, h,
opts...)`, `ListenAndServe(ctx, addr)` binding to **loopback by default** (PRD §9.1; a non-loopback bind is explicit),
and `Shutdown(ctx)` that **stops accepting, closes active association connections first, then waits for in-flight
handlers** — fixing the prototype's Shutdown that left handlers blocked in `ReadPDU` (Codex DIMSE-014). Capacity from
`WithMaxAssociations(n)` is **acquired before spawning the handler goroutine**, not after (Codex DIMSE-013). Each
inbound association runs the acceptor-side DUL/ACSE negotiation (Increment 3) and dispatches received C-ECHO-RQ /
C-STORE-RQ to the registered handler; `WithRequireCalledAETitle` / `WithRequireCallingAETitles` enforce the AE-title
checks. No fire-and-forget goroutines (PRD §9.4).

**Files:** `dimse/server.go` (`Server`, `NewServer`, `ListenAndServe`, `Shutdown`, the `ServerOption`s),
`dimse/server_test.go`, `dimse/dispatch.go` (the per-association accept loop and DIMSE dispatch). **Rewrite** of
`dimse/scp` (the prototype's `server.go`).

**Key tests:** a server bound with no explicit address listens on `127.0.0.1` (bind-default test, PRD §11.2); an
in-process SCU runs C-ECHO and C-STORE against the `Server` and the registered `Handler` answers; `Shutdown` with a
deadline returns after closing connections even while a slow handler is mid-store (DIMSE-014 regression);
`WithMaxAssociations(1)` refuses a second concurrent association before spawning its handler (DIMSE-013 regression);
a wrong Called AE Title is rejected at negotiation.

**Reference-doc section:** dimse.md "Serving inbound associations (SCP)", "SCP handlers and the event model", worked
example "### Serving a Storage SCP"; PRD §9.1 (loopback), §9.4 (concurrency). **Rewrite**.

**Verification gate:** `go test -race ./dimse/...` green; lint clean; bind-default, DIMSE-013, and DIMSE-014
regressions pass.

---

## Increment 7 — DIMSE interop gate (C-ECHO + C-STORE against Orthanc / dcm4chee-arc)

**Goal:** Un-skip the ported integration test (Increment 0) and prove the full DIMSE leg end-to-end against real PACS,
closing the loop on the prototype's Orthanc aborts. The go-radx SCU performs a C-ECHO and then a C-STORE of a vendored
`.dcm` against an Orthanc container and a dcm4chee-arc container (both via testcontainers, PRD §11.1); the go-radx
`Server` SCP receives a C-STORE from the reference toolkit (or from a second go-radx AE) and persists it. This is the
acceptance proof that the last-fragment-bit fix (Increment 5) works against a compliant peer that the prototype
aborted.

**Files:** `dimse/integration/interop_test.go` (the ported test, now active, `//go:build interop`),
`dimse/integration/orthanc/orthanc.go` and a `dcm4chee/` sibling (testcontainers helpers). No new production code —
this increment exercises Increments 1–6.

**Key tests:** `TestInteropCEchoOrthanc` (SCU C-ECHO returns `StatusEchoSuccess`); `TestInteropCStoreOrthanc` (SCU
C-STORE returns `StatusStoreSuccess` and the instance is retrievable from Orthanc); the same against dcm4chee-arc;
`TestInteropSCPReceivesCStore` (a reference SCU stores to the go-radx `Server`). The store test is the named regression
proving the Orthanc-abort defect is fixed.

**Reference-doc section:** dimse.md "Conformance scope and limits" (the interop bullet); PRD §11.1, §13 M2
("Interop-gated"). The interop tests are **ported** from `dimse/integration`.

**Verification gate:** `mise run interop:dimse` green against both Orthanc and dcm4chee-arc containers; the suite must
not be skipped in CI (PRD §11.1, merge-blocking). This is the M2 DIMSE-leg acceptance gate.

---

## Increment 8 — `dicomweb` DICOM-JSON + multipart/related codecs

**Goal:** The two wire codecs the STOW/WADO legs need. Build the DICOM-JSON model (PS3.18 Annex F) as a codec over
`*dicom.DataSet` — `MarshalJSON`/`UnmarshalJSON` keyed by eight-hex-digit tag strings, each value an object carrying
`vr` and one of `Value` / `BulkDataURI` / `InlineBinary`, with PN rendered as the Alphabetic/Ideographic/Phonetic
component-group form and `SQ` nesting DataSet objects — distinct from FHIR JSON (never cross-feed serializers, per the
glossary). Build the bounded `MultipartReader`/`MultipartWriter` for `multipart/related` framing with the part-count
and per-part-size caps (defaults: 10,000 parts, 256 MiB per part) so a hostile peer cannot exhaust memory. M2 needs
only the slice of DICOM-JSON the STOW store response and WADO metadata exercise (instance-level objects; deep `SQ`
nesting and `BulkDataURI` are exercised minimally).

**Files:** `dicomweb/json.go` (`MarshalJSON`, `UnmarshalJSON`, `JSONOption`), `dicomweb/json_test.go`,
`dicomweb/multipart.go` (`MultipartReader`, `MultipartWriter`, the caps), `dicomweb/multipart_test.go`,
`dicomweb/errors.go` (the sentinel errors `ErrLimitExceeded`, `ErrNotAcceptable`, `ErrUnsupported`,
`ErrInvalidResource`). **New** (greenfield; no Python parity floor for DICOMweb, dicomweb.md "Scope").

**Key tests:** a `DataSet` with a PN, a UI, and a US element round-trips through DICOM-JSON byte-stably; the
multipart writer/reader round-trips two `application/dicom` parts; a multipart body declaring more than the part cap
returns `ErrLimitExceeded` before reading them all (PRD §9.3 hostile-input); a body that ends mid-part returns
`io.ErrUnexpectedEOF` (truncation is failure, PRD §9.2).

**Reference-doc section:** dicomweb.md "DICOM JSON and multipart/related", "Hostile-input caps", "Behaviour and error
model". **New.**

**Verification gate:** `go test -race ./dicomweb/...` green; lint clean; the part-cap and truncation regressions pass.

---

## Increment 9 — DICOMweb STOW-RS store + WADO-RS read (client + thin server)

**Goal:** The thinnest STOW-RS + WADO-RS slice. Build `ResourcePath` (`NewStudy`/`NewSeries`/`NewInstance`, `Level()`,
`Path()` validating each UID before interpolation), the `Client` (`NewClient(baseURL, opts...)`, `WithBearerToken`
never logged, `WithMaxResponseBytes`), and `Client.Store(ctx, instances...)` — POST `multipart/related` of
`application/dicom` parts, then parse the `application/dicom+json` `StoreResponse` distinguishing `Referenced` from
`Failed` and fail-closed with a non-nil error when any instance failed (PRD §9.2). `Client.RetrieveInstance(ctx, p)`
issues a `multipart/related` GET and parses the `application/dicom` part to a `*dicom.DataSet`. Then the embeddable
server: the `StoreBackend` and `RetrieveBackend` interfaces, and `NewServer(opts...)` wiring them into an
`http.Handler` (unimplemented services return `501` with a typed problem document, never a `200` no-op, PRD §9.2),
bound to loopback by default. `StoreResponse` reuses `dicom.ReferencedSOPInstance` / `dicom.FailedSOPInstance` via the
`StoredInstance` embedding (never a bare `Reference`).

**Files:** `dicomweb/resource.go` (`ResourcePath`, `BulkDataURI`), `dicomweb/client.go` (`Client`, `ClientOption`,
`Store`, `RetrieveInstance`), `dicomweb/store_response.go` (`StoreResponse`, `StoredInstance`, `IsComplete`),
`dicomweb/server.go` (`Server`, `StoreBackend`, `RetrieveBackend`, `NewServer`, `Handler()`),
`dicomweb/negotiation.go` (the `Accept`/`Content-Type` handling for the two media types), tests for each, plus
`dicomweb/integration/interop_test.go` (Orthanc STOW then WADO round-trip, `//go:build interop`). **New.**

**Key tests:** `ResourcePath.Path()` rejects an invalid UID; `Client.Store` of one instance to an in-process `Server`
returns a complete `StoreResponse`; a backend reporting one failed instance yields a non-nil error and a
`StoreResponse` with a populated `Failed` (fail-closed, PRD §9.2); `RetrieveInstance` round-trips the stored dataset;
an unimplemented QIDO request returns `501` with `ErrUnsupported`, not a `200`; the server binds to loopback by
default (PRD §11.2). Interop: STOW a vendored `.dcm` to Orthanc, then WADO-RS retrieve it and assert the dataset
matches.

**Reference-doc section:** dicomweb.md "Resource model", "Client API" (WADO-RS retrieval, STOW-RS storage), "Server
API", "Content negotiation", "Conformance scope and limits"; PRD §9.1, §9.2, §11.1. **New.**

**Verification gate:** `go test -race ./dicomweb/...` green; lint clean; fail-closed and loopback regressions pass;
`mise run interop:dicomweb` green against Orthanc (the M2 DICOMweb-leg acceptance gate, PRD §11.1).

---

## Increment 10 — `hl7v2` ORM parse (thin)

**Goal:** Just enough HL7 v2 to parse one ORM message and feed the converter. Build the generic six-level parse tree
(`Message`, `Segment`, `Field`, `Repetition`, `Component`), `EncodingCharacters` with `DeriveEncoding` from
`MSH-1`/`MSH-2` (default-fill a short `MSH-2`, never a package global — the non-standard-sender footgun), `Parse(b,
opts...)` (lenient on `\r`/`\n`/`\r\n` terminators, strict that the first segment is `MSH`, truncation is failure),
the minimal typed segments the ORM→ServiceRequest converter reads (`MSH`, `PID`, `ORC`, `OBR` with their composite
datatypes `CX`, `CWE`, `HD`, `DTM`, `XPN` to the depth `convert` needs), and the `ORM` typed message with
`Orders() iter.Seq[OrderGroup]` yielding each `ORC` with its following `OBR`(s). The full typed-segment set, MLLP,
ACK, and batch/file are **out of M2 scope** (M5) and are not built here.

**Files:** `hl7v2/parse.go` (`Parse`, `ParseOption`, the tree types), `hl7v2/encoding.go` (`EncodingCharacters`,
`DefaultEncoding`, `DeriveEncoding`), `hl7v2/segment.go` (the generic `Segment`, `Message.Segment`/`AllSegments`),
`hl7v2/composite.go` (`CX`, `CWE`, `HD`, `DTM`, `XPN` — only the fields `convert` reads), `hl7v2/typed.go`
(`MSH`/`PID`/`ORC`/`OBR` parse-from-generic), `hl7v2/orm.go` (`ORM`, `OrderGroup`, `AsORM`, `Orders`),
`hl7v2/errors.go` (`ParseError`, `SegmentError`), tests for each. **New** (greenfield; parity floor is `python-hl7`).

**Key tests:** the canonical ORM in dimse/hl7v2 docs parses; `DeriveEncoding("MSH|^")` fills repetition/escape/
subcomponent from defaults; a body not starting with `MSH` returns `*ParseError`; an ORM with one `ORC`+`OBR` yields
one `OrderGroup` from `Orders()`; an MLLP stream that ends mid-frame is not in scope but a body that ends mid-segment
returns `io.ErrUnexpectedEOF` (truncation is failure, PRD §9.2); round-trip `Parse` → `MarshalText` is byte-exact for
the fixture.

**Reference-doc section:** hl7v2.md "The generic tree", "Accessor", "Encoding characters", "Typed segments",
"Message types" (the `ORM`/`OrderGroup` slice), "Parsing entry points"; PRD §6.2 (HL7 floor). **New.** Note: M2 builds
only the ORM-feeding slice; the conformance statement for hl7v2 (`docs/conformance/hl7v2.md`) governs the full scope
reached in M5.

**Verification gate:** `go test -race ./hl7v2/...` green; lint clean; the fixture round-trip and truncation
regressions pass.

---

## Increment 11 — `fhir/r5` minimal hand-written resources

**Goal:** Hand-written minimal R5 resources — **only** `ServiceRequest`, `DiagnosticReport`, and `ImagingStudy`, with
just the fields the skeleton converters populate — plus the shared datatypes (`Reference`, `Identifier`,
`CodeableConcept`, `Coding`) and the root release-agnostic machinery the `convert` package consumes (`fhir.Resource`
interface, `fhir.Decimal`). This is explicitly **not** the FHIR generator (M6a) and not the full resource set: it is
the smallest correct slice that lets `convert` emit valid JSON. Each resource implements `fhir.Resource`
(`ResourceType()` constant) and `MarshalJSON` emitting `"resourceType":"..."`. The `Decimal` type is the
lexical-preserving type shared with `dicom` (reuse the M1 `dicom.Decimal` semantics; `fhir.Decimal` is its FHIR-side
twin per the reference docs — confirm whether to alias or re-implement, see Open questions).

**Files:** `fhir/resource.go` (the `Resource` interface, the sentinel errors `ErrResourceTypeMismatch` etc. — only
what `convert` needs), `fhir/decimal.go` (`Decimal`, mirroring the committed signature shared across dicom.md/fhir.md),
`fhir/r5/datatypes.go` (`Reference`, `Identifier`, `CodeableConcept`, `Coding` — the committed shapes from fhir.md),
`fhir/r5/service_request.go` (`ServiceRequest` with `Identifier`, `Status`, `Intent`, `Code`, `Subject`, `AuthoredOn`,
`Requester` — the fields the ORM converter sets), `fhir/r5/diagnostic_report.go` (`DiagnosticReport` with `Identifier`,
`Status`, `Code`, `Category`, `Subject`, `EffectiveDateTime`, `Conclusion`, `Result`), `fhir/r5/imaging_study.go`
(`ImagingStudy` with `Identifier`, `Status`, `Subject`, `Started`, `NumberOfSeries`, `NumberOfInstances`,
`Description`, `Modality`, `Series[]` with `Uid`/`Number`/`Modality`/`Instance[]`), tests asserting JSON shape against
small golden files. **Hand-written minimal** (the generator is M6a; do not build it).

**Key tests:** `json.Marshal(&r5.ServiceRequest{...})` emits `{"resourceType":"ServiceRequest",...}`; an `ImagingStudy`
with one series and one instance marshals to the expected nested JSON; `fhir.Decimal("1.500").MarshalJSON()` emits the
unquoted `1.500` (lexical fidelity); `ResourceType()` is the constant string per resource.

**Reference-doc section:** fhir.md "The Resource interface", "Complex datatypes" (`Reference`/`Identifier`), "The
Decimal primitive", "Release selection in code"; convert.md (the exact fields each converter writes — the field lists
above are taken from the convert.md mapping tables); PRD §8.1. **Hand-written minimal.**

**Verification gate:** `go test -race ./fhir/...` green; lint clean; the golden-JSON shapes match. (No FHIR-validator
gate in M2 for the hand-written slice; the validator gate arrives with the generator in M6a — confirm in Open
questions whether the M2 hand-written resources should pass the validator now.)

---

## Increment 12 — `convert` slice + end-to-end skeleton test

**Goal:** The three converters the skeleton needs, plus the single end-to-end test that proves the architecture.
Build the shared `Option`/`Report`/`LossError` model and the `UIDIdentifierR5(uid)` helper (the absolute identity
rule: a DICOM UID becomes a FHIR `Identifier` with `system "urn:dicom:uid"`, value `"urn:oid:" + uid`, **never** a
`Reference.reference` URL); `ORMToServiceRequestR5(msg)` (read one `hl7v2.OrderGroup` via the typed API, map ORC/OBR to
`*r5.ServiceRequest` per the convert.md table, single-order-per-call, `ErrUnsupportedSource` on a multi-order message);
a minimal `SRToDiagnosticReportR5` or a direct `DiagnosticReport` producer from the DICOM SR content items (M1's
`dicom.ContentItem`) sufficient to emit one `*r5.DiagnosticReport`; and `DICOMToImagingStudyR5(instances)` (group by
Series/SOP Instance UID, recompute counts, map per the convert.md table, `subject` as a logical `Reference.identifier`
absent `WithSubjectR5`, defaults recorded in `Report.Defaulted`). Then the **end-to-end skeleton test**: read a
vendored `.dcm`, C-STORE it via the in-process DIMSE SCU→SCP, STOW it to and WADO-read it from the in-process DICOMweb
server, parse a vendored ORM and emit a `ServiceRequest`, produce a `DiagnosticReport`, and convert the DICOM instance
to an `ImagingStudy` — asserting each leg's output, proving every subsystem connects (PRD §13 M2: "Proves the
architecture before depth").

**Files:** `convert/options.go` (`Option`, `WithUIDRoot`, `WithStrictLoss`, `WithSubjectR5`), `convert/report.go`
(`Report`, `DroppedField`, `LossError`, the sentinel errors `ErrMissingIdentifier`/`ErrUnsupportedSource`/
`ErrMalformedSource`), `convert/identity.go` (`UIDIdentifierR5`), `convert/orm_servicerequest.go`
(`ORMToServiceRequestR5`), `convert/sr_diagnosticreport.go` (the minimal `SRToDiagnosticReportR5`/DiagnosticReport
producer), `convert/dicom_imagingstudy.go` (`DICOMToImagingStudyR5`), tests for each, and
`convert/skeleton_e2e_test.go` (the end-to-end proof, plus an `//go:build interop` variant that runs the DIMSE/DICOMweb
legs against the real containers). **New** (the `convert` package is new; reuses `dicom`/`hl7v2`/`fhir`).

**Key tests:** `UIDIdentifierR5("1.2.3")` yields `{System:"urn:dicom:uid", Value:"urn:oid:1.2.3"}`;
`ORMToServiceRequestR5` of the canonical ORM yields a `ServiceRequest` with the ORC/OBR identifiers and `intent`
defaulted to `order` (recorded in `Report.Defaulted`); a multi-order ORM returns `ErrUnsupportedSource` (fail-closed,
convert.md single-order limit); `DICOMToImagingStudyR5` of two instances of one study recomputes `numberOfSeries`/
`numberOfInstances` and never fabricates a `Reference.reference` URL for the subject; the end-to-end test drives all
six legs and asserts each output.

**Reference-doc section:** convert.md "Identity handling" (the absolute UID→Identifier rule), "DICOMToImagingStudy",
"ORMToServiceRequest", "SRToDiagnosticReport", "The conversion report and the error model"; UBIQUITOUS_LANGUAGE.md
(cross-standard collisions); PRD §13 M2. **New.**

**Verification gate:** `go test -race ./convert/...` green; lint clean; the identity rule and fail-closed regressions
pass; `mise run test:skeleton` (the in-process end-to-end test) green; the `interop` variant green against the
containers. This is the **M2 acceptance gate**: the architecture is proven thin and end-to-end.

---

## Open questions and resolutions

Resolved against the committed specs by the orchestrator before execution. One remains for the architect.

1. **`fhir.Decimal` vs `dicom.Decimal` — RESOLVED (defer to M6a).** The M2 hand-written slice
   (`ServiceRequest`/`DiagnosticReport`/`ImagingStudy` minimal fields) uses no FHIR `decimal`, so M2 introduces **no**
   `Decimal` type in `fhir`. Increment 11 must not define one. The `dicom.Decimal`/`fhir.Decimal` unification is decided
   at M6a, when the generator needs `decimal` pervasively; the leading option is a type alias `type Decimal =
   dicom.Decimal` in the `fhir` root (intra-library coupling is acceptable — PRD §7.4 constrains only the CLI module
   graph), but the choice is deferred so M2 makes no premature commitment.

2. **ACSE package name — RESOLVED (confirmed).** `dimse/acse` holds the internal negotiation logic; the public
   `AE`/`Association`/`PresentationContext` types live in the **root** `dimse` package (matching the reference doc's
   `dimse.AE`/`dimse.Association`/`dimse.PresentationContext` call sites). The acyclic three-layer split is
   `dimse/pdu` → `dimse/dul` → `dimse/acse` → root `dimse`.

3. **FHIR validator on the M2 slice — RESOLVED (no validator gate in M2).** Per PRD §13 the FHIR validator gate is M6a.
   M2's hand-written resources need only be well-formed; the gate is golden-JSON shape tests, not the validator.

4. **dcm4chee-arc in the M2 DIMSE interop gate — RESOLVED by the architect: require BOTH at M2.** The M2 DIMSE
   (Increment 7) and DICOMweb (Increment 9) interop gates are green only when C-ECHO + C-STORE / STOW + WADO pass
   against **both** Orthanc **and** dcm4chee-arc, per PRD §11.1's dual-PACS rule applied at M2 (not deferred to M3).
   Increment 0 stands up both testcontainers; if dcm4chee-arc startup proves flaky, harden the container fixture rather
   than narrow the gate. dcm4chee is NOT marked deferrable.

5. **`docs/conformance/dimse.md` / `dicomweb.md` do not exist — RESOLVED (confirmed).** Only `dicom.md`, `fhir.md`,
   `hl7v2.md` exist in `docs/conformance/`. M2 uses the "Conformance scope and limits" sections inside
   `docs/reference/dimse.md` and `docs/reference/dicomweb.md` as the gate. The formal DIMSE/DICOMweb conformance
   statements are authored in **M8** per PRD §13, not M2.

### Reviewer corrections to apply when expanding the named increment

An independent plan review (verified against the committed reference docs and the prototype source) confirmed the DIMSE
legs (Increments 0–7) are execution-ready and found no Critical issues. Apply these corrections when expanding the
later legs, so the outlines do not drift from the committed API shapes:

- **Increment 9 (DICOMweb store, H1):** the store error model must use the committed `*StoreError` type, not a generic
  error. `Client.Store(ctx, ...*dicom.DataSet) (*StoreResponse, error)` returns a non-nil `*StoreResponse` **and** a
  non-nil `*StoreError` on partial failure (dicomweb.md §"Storing"); `StoreResponse.IsComplete()` is true only when
  `Failed` is empty. Name `StoreError` and an `IsComplete()` regression test explicitly.
- **Increment 11 (FHIR resources, H2):** pin each field to its R5 datatype per convert.md's release-gated tables —
  `ServiceRequest.code` is **`CodeableReference`** (R5), `ImagingStudy.modality` and `series.modality` are
  **`CodeableConcept`** (R5), `series.bodySite` is **`CodeableReference`** (R5). Add `CodeableReference` to
  `fhir/r5/datatypes.go` (the outline currently lists only `CodeableConcept`/`Coding`).
- **Increment 12 (DiagnosticReport, H3):** M1 *does* deliver `dicom.ContentItem` (`dicom/sr_parse.go` `ParseSR`), but
  the committed `SRToDiagnosticReportR5` also returns `[]*r5.Observation`, which the thin M2 FHIR slice omits.
  **Scope M2's DiagnosticReport to a narrative-only producer** (no SR content-item walk, no `Observation`); defer the
  full `SRToDiagnosticReportR5` (+ `Observation`, via `dicom.ContentItem`) to **M7** (Conversion). Remove the "or" hedge
  in the outline and pin narrative-only.
- **Increment 12 (options, M1):** `WithSubjectR5(ref r5.Reference) Option` depends on `fhir/r5.Reference` (built in
  Increment 11); note the dependency so the executor does not stub `Reference`.

### Architecture decision (from execution): DUL state-machine ownership

Surfaced at Increment 3 and decided by the architect. The DUL state machine is owned by the **caller** — the ACSE
layer for the association lifecycle, and the DIMSE message layer (Increment 5) and the SCP (Increment 6) for their
phases — NOT by `dul.Conn`. The `dul` package provides three composable pieces: `StateMachine` (the PS3.8 Table 9-10
table + `Apply`), `Conn` (pure context-aware PDU framing/I/O — no embedded FSM), and `DriveInbound(ctx, conn, *StateMachine)`,
the one hardened inbound path that reads a PDU, maps it (io.EOF→Evt17, malformed→Evt19, recognised→its event), applies
it to the caller's machine, and SENDS the provider/user A-ABORT with the correct reason on a fault action. Increments
5 and 6 MUST route their inbound reads through `dul.DriveInbound` against their own `StateMachine` so the abort-send and
the clean-close/malformed distinction are never reimplemented (the original split left the hardening in dead `Conn` code
while the live path failed to send the AA-8 A-ABORT — do not recreate that).
