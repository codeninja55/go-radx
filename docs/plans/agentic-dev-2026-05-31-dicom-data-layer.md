# DICOM data layer (M1) implementation plan

> **For agentic workers:** REQUIRED: Use agentic-dev:subagent-driven-development (if subagents available) or
> agentic-dev:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `dicom` package, the M1 foundation every other go-radx subsystem reads, as a ground-up rewrite
of the audited prototype, conforming exactly to the committed public API in `docs/reference/dicom.md` and the v1 subset
in `docs/conformance/dicom.md`, with each of the sixteen Codex defects (DCM-001..DCM-016) guarded by a regression test.

**Architecture:** A single flat Go package `dicom` (one importable surface, per the reference doc's `dicom.Tag`,
`dicom.VR`, `dicom.ReadFile` idioms), split into focused files by responsibility. The core type system (Tag, VR, UID,
value types, Decimal, PersonName, DataSet/Element) is built first with no I/O dependencies, then the transfer-syntax
aware Part 10 reader and writer are layered on top, then sequences, character sets, datetimes, the pixel pipeline
(pure-Go native and RLE always; JPEG-family decode behind an optional CGo build tag), and finally the PS3.15 Basic
Profile de-identifier. The reader is built on a bounded reader so every length is validated against bytes actually
remaining before any allocation; truncation is a failure, never a graceful EOF.

**Tech Stack:** Go 1.26.3, module `github.com/codeninja55/go-radx`; standard library only for the core
(`encoding/binary`, `math/big`, `iter`, `crypto/rand`); `golang.org/x/text/encoding` for ISO 2022, GB18030, and GBK
character sets; optional CGo (OpenJPEG, CharLS) behind a `//go:build cgo && radx_codecs` tag for JPEG-family pixel
decode; `go.uber.org/zap` for diagnostics. Verification gates: `dciodvfy` (dcmtk) on written files and `pydicom`
round-trips against vendored fixtures.

---

## How to use this plan

Read this section once before starting; it states the conventions every task follows.

**Test-first, always.** Each task is a strict TDD cycle: write the failing test, run it and confirm it fails for the
right reason, write the minimal implementation, run it and confirm it passes, then commit. Do not write implementation
before its test. See `agentic-dev:test-driven-development`.

**Canonical names are mandatory.** Use the exact Go identifiers fixed in `UBIQUITOUS_LANGUAGE.md`: `Tag`, `VR`, `UID`,
`SOPClassUID`, `SOPInstanceUID`, `TransferSyntax`, `DataSet` (capital S), `Element`, `Sequence`, `Item`, `Decimal`,
`PersonName`, `SpecificCharacterSet`, `UIDGenerator`, `PixelData`, `BasicOffsetTable`, `FileMeta`, `ContentItem`,
`ConceptNameCode`, `ReferencedSOPInstance`, `Profile`. Never reintroduce the prototype's aliases (`dataset`, bare
`uint32` tags, `GenerateUID` free function).

**Honour the committed API.** The signatures in `docs/reference/dicom.md` are the contract. Do not invent new public
API. Where this plan shows a signature, it is copied from the reference doc; if you find a genuine gap, stop and surface
it rather than guessing (see "Open questions" at the end).

**Bounds-check every length.** This package parses hostile input. Every 32-bit length read from the wire is validated
against the bytes remaining in a bounded reader before any `make([]byte, n)` (PRD §9.3, Codex DCM-004). Each parsing
task includes a hostile-length regression test.

**Diagnostics carry no PHI.** Errors name the element by keyword plus `(gggg,eeee)`, the VR by name, the UID by
registered name — never the patient value (PRD §8.2, §9.1, Codex none-specific but a standing rule).

**Commit conventionally and often.** Each commit message follows `<type>(dicom): <description>` (for example
`feat(dicom): add Tag named type with group/element accessors`). Source and its test commit together; generated
dictionaries, fixtures, and tooling commit separately.

**Codex defect traceability.** Tasks that close a specific audited defect cite it inline (for example "guards Codex
DCM-005"). The defect catalogue is in `/tmp/go-radx-codex-reviews/dicom.md`; a defect is not closed until its named
regression test passes.

**Porting note — representation changes from the prototype.** The prototype at `/Users/codeninja/vcs/go-radx/dicom`
modelled `Tag` as `struct{Group, Element uint16}` and `UID` as a struct. The committed reference API is `type Tag uint32`
and `type UID string`. This is a deliberate rewrite, not a regression: port the *logic* (parsing, validation, dictionary
data) but adopt the committed scalar representations. The innolitics-derived tag dictionary generator and the vendored
fixture corpus in the prototype's `testdata/dicom/` are reusable and are pulled in by Increment 0.

---

## Increment overview (dependency-ordered)

- **Increment 0 — Test harness and fixtures.** Vendor a small licensed pydicom/dcmtk fixture set into `testdata/`,
  add a `mise`/`Makefile` test target, and wire the `dciodvfy` and pydicom round-trip gates (skipped gracefully when the
  tools are absent). No production code; everything below depends on it.
- **Increment 1 — Core type system.** `Tag` + generated dictionary, `VR`, `UID`/`SOPClassUID`/`SOPInstanceUID`/
  `UIDGenerator`, the value types, `Decimal`, `PersonName`, and `DataSet`/`Element`. Pure data; no I/O. **Fully expanded
  into bite-sized TDD tasks below.**
- **Increment 2 — Part 10 read/write (uncompressed).** Preamble, `FileMeta` with auto group length, Explicit and
  Implicit VR LE, Explicit VR BE (retired), Deflated Explicit VR LE; bounded reader; truncation-is-failure. *Outlined.*
- **Increment 3 — Sequences.** `SQ`/`Item`, defined and undefined length, bounded depth and size. *Outlined.*
- **Increment 4 — Specific Character Set.** `SpecificCharacterSet` decode/encode incl. ISO 2022, UTF-8, GB18030, GBK.
  *Outlined.*
- **Increment 5 — Datetime VRs.** `DA`/`TM`/`DT`, strict default + lenient opt-in, leap-second handling. *Outlined.*
- **Increment 6 — Pixel data pipeline.** Native + RLE pure Go; encapsulated fragment parsing; JPEG-family decode and
  RLE/JPEG2000-lossless encode behind the optional CGo build tag; transcoding off by default. *Outlined.*
- **Increment 7 — PS3.15 Basic Profile de-identification.** Recursive Table E.1-1 action set, consistent UID remap,
  dates removed by default, fail-closed on burned-in pixel PHI. *Outlined.*
- **Increment 8 — Structured Report content-item model.** `ContentItem`, `ConceptNameCode`, `ValueType`,
  `RelationshipType`, `ReferencedSOPInstance`, and `ParseSR`/`BuildSR` over the SR content tree for the Basic Text,
  Enhanced, and Comprehensive SR SOP classes. *Outlined.*

Increments 2 through 8 are outlined here (scope, key signatures from the reference doc, dependencies, verification gate)
and are expanded into bite-sized TDD tasks when reached.

---

## Increment 0 — Test harness and fixtures

**Scope:** Stand up the test infrastructure every later increment relies on: a vendored fixture corpus with license
attribution, a `mise` task to run the package tests, and the two conformance gates (`dciodvfy` on written files, pydicom
round-trip). The gates must skip cleanly with a clear message when the external tool is not installed, so the suite is
green on a developer machine without dcmtk/pydicom but fails loudly in CI where they are present.

**Files:**
- Create: `testdata/dicom/README.md` (fixture provenance + license attribution)
- Create: `testdata/dicom/*.dcm` (vendored fixtures — copied from the prototype corpus)
- Create: `testdata/dicom/LICENSE-pydicom.txt`, `testdata/dicom/LICENSE-gdcm.txt`, `testdata/dicom/LICENSE-dcmtk.txt`
- Create: `tools/dicom-conformance/dciodvfy.sh` (wrapper that no-ops with exit 0 + skip message when `dciodvfy` absent)
- Create: `tools/dicom-conformance/pydicom_roundtrip.py` (reads a `.dcm`, re-saves, asserts byte-stable for canonical
  Explicit VR LE input)
- Modify: `mise.toml` (add `[tasks."test:dicom"]`, `[tasks."dicom:dciodvfy"]`, `[tasks."dicom:pydicom"]`)
- Create: `Makefile` (mirror targets for non-mise users, per the task brief)

- [ ] **Step 1: Vendor the fixture corpus**

The prototype's corpus at `/Users/codeninja/vcs/go-radx/dicom/../testdata/dicom/` (and the broader pydicom/gdcm/dcmtk
test files) is the source. Copy a small, purposeful subset, not the whole corpus, chosen so each later increment has
exactly the fixtures it needs:

```bash
mkdir -p testdata/dicom
# Canonical small explicit-VR-LE file for round-trip stability tests (Increment 2).
cp /Users/codeninja/vcs/go-radx/testdata/dicom/liver.dcm                 testdata/dicom/
# Implicit VR LE + nested sequences for SQ tests (Increment 3).
cp /Users/codeninja/vcs/go-radx/testdata/dicom/MR2_UNCI.dcm              testdata/dicom/
# Retired Explicit VR Big Endian for byte-order tests (Increment 2, DCM-002).
cp /Users/codeninja/vcs/go-radx/testdata/dicom/SC_rgb_expb.dcm           testdata/dicom/
# RLE-encapsulated for pure-Go pixel decode (Increment 6).
cp /Users/codeninja/vcs/go-radx/testdata/dicom/liver_rle.dcm            testdata/dicom/
# JPEG2000 for CGo-gated decode + ErrCodecUnavailable nocgo test (Increment 6).
cp /Users/codeninja/vcs/go-radx/testdata/dicom/liver_j2k.dcm           testdata/dicom/
# Overlays + repeating group 60xx for dictionary-mask test (Increment 1, DCM-012).
cp /Users/codeninja/vcs/go-radx/testdata/dicom/MR-SIEMENS-DICOM-WithOverlays.dcm testdata/dicom/
# explicit_VR-UN.dcm for ambiguous/UN VR resolution.
cp /Users/codeninja/vcs/go-radx/testdata/dicom/explicit_VR-UN.dcm      testdata/dicom/
```

Then write `testdata/dicom/README.md` documenting each file's origin (pydicom `pydicom/data/test_files`, GDCM
`gdcmData`, or dcmtk), the transfer syntax it exercises, and which increment/defect uses it. Copy the upstream license
texts into the three `LICENSE-*.txt` files. The pydicom test files are MIT-licensed; GDCM is BSD-3; dcmtk fixtures carry
the dcmtk license — reproduce each verbatim with a one-line note on which fixtures it covers.

- [ ] **Step 2: Add a truncated-fixture generator helper**

Increment 2 needs a deliberately truncated file to prove truncation-is-failure (DCM-003). Generate it from a good
fixture at test time rather than committing a corrupt binary, but commit the generator as a `testdata` helper:

```go
// testdata/dicom/gen/truncate.go — run with: go run ./testdata/dicom/gen -in liver.dcm -bytes 512
// Writes liver.truncated.dcm = first N bytes of a valid file, for the DCM-003 regression.
```

Keep this minimal; full code is written when Increment 2 is reached.

- [ ] **Step 3: Add the mise/Make test targets**

Add to `mise.toml`:

```toml
[tasks."test:dicom"]
description = "Run the dicom package test suite (race + coverage)"
run = "go test -race -cover ./dicom/..."

[tasks."dicom:dciodvfy"]
description = "Validate written DICOM files with dcmtk dciodvfy (skips if absent)"
run = "tools/dicom-conformance/dciodvfy.sh"

[tasks."dicom:pydicom"]
description = "Round-trip vendored fixtures through pydicom (skips if absent)"
run = "python3 tools/dicom-conformance/pydicom_roundtrip.py testdata/dicom"
```

Mirror these three in a `Makefile` (`test-dicom`, `dicom-dciodvfy`, `dicom-pydicom`) so a non-mise contributor can run
them. The gate scripts must detect a missing tool and exit 0 with a clear `SKIP: dciodvfy not installed` message, never
a false failure on a bare machine. But in CI (`CI=true`) a missing tool is a hard failure so the gate cannot silently
rot.

- [ ] **Step 4: Verify the harness**

Run: `mise run test:dicom`
Expected: PASS with `no test files` for `dicom` (package is still empty) — confirms the task wiring works.

Run: `mise run dicom:pydicom`
Expected on this machine (pydicom not installed): `SKIP: pydicom not installed` and exit 0.

- [ ] **Step 5: Commit (fixtures, tooling, and config as separate atomic commits)**

```bash
git add testdata/dicom/*.dcm testdata/dicom/README.md testdata/dicom/LICENSE-*.txt
git commit -m "test(dicom): vendor licensed pydicom/gdcm/dcmtk fixture corpus"
git add tools/dicom-conformance/ testdata/dicom/gen/
git commit -m "build(dicom): add dciodvfy and pydicom round-trip conformance gates"
git add mise.toml Makefile
git commit -m "build(dicom): add test:dicom and conformance mise/make targets"
```

**Verification gate:** `mise run test:dicom` is green; both conformance gates run and skip cleanly without their tools.

---

## Increment 1 — Core type system (fully expanded)

**Scope:** The pure-data foundation: `Tag` and the generated dictionary, `VR`, `UID` family and `UIDGenerator`, the
`Value` interface and concrete value types, the lexical-preserving `Decimal`, `PersonName`, and `DataSet`/`Element`. No
file or network I/O. This increment closes Codex DCM-007 (padding), DCM-008 (UID root), DCM-009 (single UID validation
path), DCM-011 (PersonName component model), DCM-012 (repeating-group dictionary mask), and DCM-016 (deep clone, no
shared mutable value objects, documented concurrency).

**File structure:**
- `dicom/tag.go` + `dicom/tag_test.go` — `Tag`, `NewTag`, accessors, predicates, `String`.
- `dicom/dictionary.go` + `dicom/dictionary_test.go` — `TagInfo`, `Lookup` (with mask), `LookupKeyword`,
  `LookupKeywordTag`.
- `dicom/tag_values.go` (generated) + `dicom/gen/gentags/main.go` (generator) — the ~5,189-entry dictionary and the
  `Tag*` constants.
- `dicom/vr.go` + `dicom/vr_test.go` — `VR`, the 34+4 constants, `String`, `Is32BitLength`, `PadByte`.
- `dicom/uid.go` + `dicom/uid_test.go` — `UID`, `ParseUID`, `Validate`, `Name`, `SOPClassUID`/`SOPInstanceUID`.
- `dicom/uid_generator.go` + `dicom/uid_generator_test.go` — `UIDGenerator`, `NewUIDGenerator`,
  `NewRandomUIDGenerator`, `Generate`.
- `dicom/decimal.go` + `dicom/decimal_test.go` — `Decimal` and its conversions.
- `dicom/person_name.go` + `dicom/person_name_test.go` — `PersonName`, `NameComponents`.
- `dicom/value.go` + `dicom/value_test.go` — `Value` interface, `Strings`/`Ints`/`Floats`/`Decimals`/`Tags`/`Bytes`
  constructors and `EncodedLen` with VR-correct padding.
- `dicom/dataset.go` + `dicom/dataset_test.go` — `DataSet`, `Element`, accessors, `All`, `Clone`, typed getters.

---

### Task 1.1: `Tag` named type

**Files:**
- Create: `dicom/tag.go`
- Test: `dicom/tag_test.go`

- [ ] **Step 1: Write the failing test**

```go
package dicom

import "testing"

func TestTagAccessorsAndString(t *testing.T) {
	tests := []struct {
		name           string
		group, element uint16
		wantString     string
	}{
		{"patient name", 0x0010, 0x0010, "(0010,0010)"},
		{"study instance uid", 0x0020, 0x000D, "(0020,000D)"},
		{"pixel data", 0x7FE0, 0x0010, "(7FE0,0010)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := NewTag(tc.group, tc.element)
			if got := tag.Group(); got != tc.group {
				t.Errorf("Group() = %#04x, want %#04x", got, tc.group)
			}
			if got := tag.Element(); got != tc.element {
				t.Errorf("Element() = %#04x, want %#04x", got, tc.element)
			}
			if got := tag.String(); got != tc.wantString {
				t.Errorf("String() = %q, want %q", got, tc.wantString)
			}
		})
	}
}

func TestTagPredicates(t *testing.T) {
	if !NewTag(0x0009, 0x0010).IsPrivate() {
		t.Error("odd group should be private")
	}
	if NewTag(0x0010, 0x0010).IsPrivate() {
		t.Error("even group should not be private")
	}
	if !NewTag(0x0009, 0x0010).IsPrivateCreator() {
		t.Error("(0009,0010) should be a private creator tag")
	}
	if NewTag(0x0009, 0x1001).IsPrivateCreator() {
		t.Error("(0009,1001) is a private data tag, not a creator")
	}
	if !NewTag(0x0010, 0x0000).IsGroupLength() {
		t.Error("element 0x0000 should be a group-length tag")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestTag -v`
Expected: FAIL — `undefined: NewTag` (compile error).

- [ ] **Step 3: Write minimal implementation**

```go
// Package dicom is the go-radx DICOM data model and Part 10 file format.
package dicom

import "fmt"

// Tag is the 32-bit (group, element) identifier of a data element, written
// (gggg,eeee). Odd groups are private. It is a named type, never a bare uint32.
type Tag uint32

// NewTag composes a Tag from its group and element halves.
func NewTag(group, element uint16) Tag {
	return Tag(uint32(group)<<16 | uint32(element))
}

// Group returns the high 16 bits.
func (t Tag) Group() uint16 { return uint16(t >> 16) }

// Element returns the low 16 bits.
func (t Tag) Element() uint16 { return uint16(t) }

// String renders the canonical (gggg,eeee) form.
func (t Tag) String() string {
	return fmt.Sprintf("(%04X,%04X)", t.Group(), t.Element())
}

// IsPrivate reports whether the group is odd (private data).
func (t Tag) IsPrivate() bool { return t.Group()%2 == 1 }

// IsPrivateCreator reports a private-creator tag: odd group, element in 0x0010..0x00FF.
func (t Tag) IsPrivateCreator() bool {
	return t.IsPrivate() && t.Element() >= 0x0010 && t.Element() <= 0x00FF
}

// IsGroupLength reports element == 0x0000.
func (t Tag) IsGroupLength() bool { return t.Element() == 0x0000 }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestTag -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/tag.go dicom/tag_test.go
git commit -m "feat(dicom): add Tag named type with accessors and predicates"
```

---

### Task 1.2: `VR` enum, `String`, `Is32BitLength`, `PadByte`

**Files:**
- Create: `dicom/vr.go`
- Test: `dicom/vr_test.go`

This task centralises padding on the VR so no value type reimplements it — the fix for Codex DCM-007 (the prototype
padded only `UI`, emitting odd-length value fields for `AE`/`CS`/`DA`/`DS`/`IS`/`LO`/`PN`).

- [ ] **Step 1: Write the failing test**

```go
package dicom

import "testing"

func TestVRString(t *testing.T) {
	tests := map[VR]string{
		VRAE: "AE", VRPN: "PN", VRSQ: "SQ", VRUI: "UI",
		VROB: "OB", VROV: "OV", VRUV: "UV", VRUN: "UN",
	}
	for vr, want := range tests {
		if got := vr.String(); got != want {
			t.Errorf("VR(%d).String() = %q, want %q", vr, got, want)
		}
	}
}

func TestVRStandardCount(t *testing.T) {
	// 34 standard VRs are enumerated as iota constants VRAE..VRUV.
	if int(VRUV)+1 != 34 {
		t.Errorf("expected 34 standard VRs (VRAE..VRUV), last index = %d", VRUV)
	}
}

func TestVRIs32BitLength(t *testing.T) {
	long := []VR{VROB, VROW, VROD, VROF, VROL, VROV, VRSQ, VRUC, VRUR, VRUT, VRUN}
	for _, vr := range long {
		if !vr.Is32BitLength() {
			t.Errorf("%s should use the 4-byte explicit-VR length form", vr)
		}
	}
	short := []VR{VRAE, VRCS, VRDA, VRPN, VRUS, VRSS, VRUI, VRDS, VRIS}
	for _, vr := range short {
		if vr.Is32BitLength() {
			t.Errorf("%s should use the 2-byte length form", vr)
		}
	}
}

func TestVRPadByte(t *testing.T) {
	// UI pads with NULL; other string VRs pad with SPACE; binary VRs are not padded.
	if b, ok := VRUI.PadByte(); !ok || b != 0x00 {
		t.Errorf("VRUI pad = (%#x,%v), want (0x00,true)", b, ok)
	}
	for _, vr := range []VR{VRAE, VRCS, VRDA, VRDS, VRIS, VRLO, VRPN} {
		if b, ok := vr.PadByte(); !ok || b != 0x20 {
			t.Errorf("%s pad = (%#x,%v), want (0x20,true)", vr, b, ok)
		}
	}
	for _, vr := range []VR{VRUS, VRFL, VROB, VRSQ} {
		if _, ok := vr.PadByte(); ok {
			t.Errorf("%s should not declare a pad byte", vr)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestVR -v`
Expected: FAIL — `undefined: VR` and the constants.

- [ ] **Step 3: Write minimal implementation**

Copy the full constant block and `String` table from the reference doc `docs/reference/dicom.md` (the 34 standard VRs in
the exact `iota` order `VRAE..VRUV`). Then add the four ambiguous placeholders as constants *after* `VRUV` (so the
standard count stays 34) and the three methods:

```go
package dicom

// VR is the two-letter Value Representation from PS3.5 Table 6.2-1.
type VR uint8

const (
	VRAE VR = iota // Application Entity
	VRAS           // Age String
	VRAT           // Attribute Tag
	VRCS           // Code String
	VRDA           // Date
	VRDS           // Decimal String
	VRDT           // Date Time
	VRFL           // Floating Point Single
	VRFD           // Floating Point Double
	VRIS           // Integer String
	VRLO           // Long String
	VRLT           // Long Text
	VROB           // Other Byte
	VROD           // Other Double
	VROF           // Other Float
	VROL           // Other Long
	VROV           // Other Very Long
	VROW           // Other Word
	VRPN           // Person Name
	VRSH           // Short String
	VRSL           // Signed Long
	VRSQ           // Sequence of Items
	VRSS           // Signed Short
	VRST           // Short Text
	VRSV           // Signed Very Long
	VRTM           // Time
	VRUC           // Unlimited Characters
	VRUI           // Unique Identifier
	VRUL           // Unsigned Long
	VRUN           // Unknown
	VRUR           // URI/URL
	VRUS           // Unsigned Short
	VRUT           // Unlimited Text
	VRUV           // Unsigned Very Long
)

// Ambiguous parse-time placeholders (PS3.6 dictionary VRs the reader resolves from
// context). They never appear on the wire.
const (
	VRUSorSS VR = iota + 34 // "US or SS"
	VRUSorOW                // "US or OW"
	VROBorOW                // "OB or OW"
	VRUSorSSorOW            // "US or SS or OW"
)

var vrNames = [...]string{
	VRAE: "AE", VRAS: "AS", VRAT: "AT", VRCS: "CS", VRDA: "DA", VRDS: "DS",
	VRDT: "DT", VRFL: "FL", VRFD: "FD", VRIS: "IS", VRLO: "LO", VRLT: "LT",
	VROB: "OB", VROD: "OD", VROF: "OF", VROL: "OL", VROV: "OV", VROW: "OW",
	VRPN: "PN", VRSH: "SH", VRSL: "SL", VRSQ: "SQ", VRSS: "SS", VRST: "ST",
	VRSV: "SV", VRTM: "TM", VRUC: "UC", VRUI: "UI", VRUL: "UL", VRUN: "UN",
	VRUR: "UR", VRUS: "US", VRUT: "UT", VRUV: "UV",
	VRUSorSS: "US or SS", VRUSorOW: "US or OW", VROBorOW: "OB or OW",
	VRUSorSSorOW: "US or SS or OW",
}

func (vr VR) String() string {
	if int(vr) < len(vrNames) && vrNames[vr] != "" {
		return vrNames[vr]
	}
	return "??"
}

// Is32BitLength reports whether the VR uses the 4-byte explicit-VR length form.
func (vr VR) Is32BitLength() bool {
	switch vr {
	case VROB, VROW, VROD, VROF, VROL, VROV, VRSQ, VRUC, VRUR, VRUT, VRUN:
		return true
	default:
		return false
	}
}

// PadByte returns the value-field pad byte: NULL (0x00) for UI, SPACE (0x20) for
// other string VRs. ok is false for binary VRs (their natural length is even).
func (vr VR) PadByte() (byte, bool) {
	switch vr {
	case VRUI:
		return 0x00, true
	case VRAE, VRAS, VRCS, VRDA, VRDS, VRDT, VRIS, VRLO, VRLT,
		VRPN, VRSH, VRST, VRTM, VRUC, VRUR, VRUT:
		return 0x20, true
	default:
		return 0, false
	}
}
```

**Note — DS/IS pad with SPACE.** `VRDS` and `VRIS` are in the SPACE pad set above even though their values are modelled
as `Decimals` (Task 1.10), not `Strings`. They are character VRs on the wire, so PS3.5 §6.2 requires SPACE padding of the
value field, and `Decimals.EncodedLen` relies on this to produce an even length (Codex DCM-007). This is more complete
than the reference doc's `PadByte` prose, which lists only UI and "other string VRs" — do not "simplify" DS/IS back out
of the SPACE set.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestVR -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/vr.go dicom/vr_test.go
git commit -m "feat(dicom): add VR enum with 34 standard + 4 ambiguous and centralised padding"
```

---

### Task 1.3: `UID` parse, validate, and name

**Files:**
- Create: `dicom/uid.go`
- Test: `dicom/uid_test.go`

This is the single validation path the writer must reuse (Codex DCM-009: the prototype had a weaker local check
accepting `1..2` and `1.02`).

- [ ] **Step 1: Write the failing test**

```go
package dicom

import (
	"errors"
	"strings"
	"testing"
)

func TestParseUIDValid(t *testing.T) {
	valid := []string{
		"1.2.840.10008.1.2.1",
		"1.2.840.10008.5.1.4.1.1.2",
		"0",          // single zero component is allowed
		"1.0.2",      // single zero between dots is allowed
		"2.25.123456789",
	}
	for _, s := range valid {
		if _, err := ParseUID(s); err != nil {
			t.Errorf("ParseUID(%q) unexpected error: %v", s, err)
		}
	}
}

func TestParseUIDInvalid(t *testing.T) {
	invalid := []string{
		"",            // empty
		"1..2",        // empty component (DCM-009)
		"1.02",        // leading zero in multi-digit component (DCM-009)
		"1.2.",        // trailing dot
		".1.2",        // leading dot
		"1.2.a",       // non-numeric
		strings.Repeat("1", 65), // over 64 characters
	}
	for _, s := range invalid {
		if _, err := ParseUID(s); err == nil {
			t.Errorf("ParseUID(%q) = nil error, want rejection", s)
		}
	}
}

func TestUIDName(t *testing.T) {
	if got := UID("1.2.840.10008.1.2.1").Name(); got != "Explicit VR Little Endian" {
		t.Errorf("Name() = %q, want Explicit VR Little Endian", got)
	}
	// Unregistered UID returns itself.
	if got := UID("1.2.3.4.5").Name(); got != "1.2.3.4.5" {
		t.Errorf("Name() = %q, want the UID itself", got)
	}
}

func TestUIDValidateErrorIsTyped(t *testing.T) {
	_, err := ParseUID("1..2")
	if err == nil {
		t.Fatal("want error")
	}
	var ve *ValueError
	if !errors.As(err, &ve) {
		t.Errorf("want *ValueError, got %T", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestUID -v` and `go test ./dicom/ -run TestParseUID -v`
Expected: FAIL — `undefined: ParseUID`, `UID`, `ValueError`.

- [ ] **Step 3: Write minimal implementation**

```go
package dicom

import "fmt"

// UID is a dotted-numeric ISO OID of at most 64 characters (VR UI). It identifies
// SOP Classes and Instances, Studies, Series, and transfer syntaxes.
type UID string

// ParseUID validates s per PS3.5 sec 9.1 and returns the trimmed UID. It rejects
// empty components ("1..2"), leading-zero multi-digit components ("1.02"), leading
// and trailing dots, non-numeric characters, and lengths over 64 characters.
func ParseUID(s string) (UID, error) {
	u := UID(s)
	if err := u.Validate(); err != nil {
		return "", err
	}
	return u, nil
}

// Validate reports a typed ValueError when u is not a conformant UID.
func (u UID) Validate() error {
	s := string(u)
	if s == "" {
		return &ValueError{VR: VRUI, Msg: "UID is empty"}
	}
	if len(s) > 64 {
		return &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID exceeds 64 characters (%d)", len(s))}
	}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			comp := s[start:i]
			if comp == "" {
				return &ValueError{VR: VRUI, Msg: "UID has an empty component"}
			}
			if len(comp) > 1 && comp[0] == '0' {
				return &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID component %q has a leading zero", comp)}
			}
			for j := 0; j < len(comp); j++ {
				if comp[j] < '0' || comp[j] > '9' {
					return &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID component %q is not numeric", comp)}
				}
			}
			start = i + 1
		}
	}
	return nil
}

// IsValid reports whether u passes Validate.
func (u UID) IsValid() bool { return u.Validate() == nil }

// String returns the raw UID text.
func (u UID) String() string { return string(u) }

// Name returns the registered name for a known UID or the UID itself if unregistered.
func (u UID) Name() string {
	if name, ok := uidNames[u]; ok {
		return name
	}
	return string(u)
}
```

Also add the `ValueError` type in `dicom/errors.go` (it is reused throughout; create the file now).

**Ordering note — keep every task compiling.** `keywordFor` is the diagnostic that resolves a tag to its keyword
through the dictionary, but `Lookup`/`TagInfo` do not exist until Task 1.5. To keep the package compiling at the end of
*every* task (so each `go test` fails only for its intended reason, never `undefined: Lookup`), `keywordFor` ships
**self-contained** here (it returns `""` for now) and Task 1.5 enriches it to call `Lookup` once the dictionary lands.
Do not call `Lookup` from this file yet.

```go
package dicom

import "fmt"

// ValueError reports a value that does not conform to its VR (bad UID, bad date,
// odd binary length, over-long PN component). It names the tag and VR; it never
// carries the offending PHI value (PRD §9.1).
type ValueError struct {
	Tag Tag
	VR  VR
	Msg string
}

func (e *ValueError) Error() string {
	if e.Tag != 0 {
		return fmt.Sprintf("dicom: invalid %s at %s %s: %s", e.VR, keywordFor(e.Tag), e.Tag, e.Msg)
	}
	return fmt.Sprintf("dicom: invalid %s: %s", e.VR, e.Msg)
}

// keywordFor renders a tag's keyword for diagnostics; falls back to "" if unknown.
// Task 1.5 enriches this to resolve the keyword through Lookup once the dictionary
// exists; until then it is deliberately empty so this file compiles standalone.
func keywordFor(Tag) string { return "" }
```

The `uidNames` map is supplied by the generated UID dictionary (Task 1.4 generator output). For this task, seed a tiny
hand-written stub map in `dicom/uid.go` with just the transfer-syntax UIDs the test needs, marked with a `TODO:` to be
superseded by the generated dictionary:

```go
// TODO: superseded by the generated uid_values.go dictionary (Task 1.4).
var uidNames = map[UID]string{
	"1.2.840.10008.1.2.1": "Explicit VR Little Endian",
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run 'TestUID|TestParseUID' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/uid.go dicom/uid_test.go dicom/errors.go
git commit -m "feat(dicom): add UID type with single PS3.5 sec9.1 validation path"
```

---

### Task 1.4: Generated tag and UID dictionaries

**Files:**
- Create: `dicom/dictionary.go` (the `TagInfo` and `repeatingEntry` type definitions + the `dictLen` helper; the
  `Lookup`/`LookupKeyword`/`LookupKeywordTag` functions are added by Task 1.5)
- Create: `dicom/gen/gentags/main.go` (generator; ports the prototype's
  `/Users/codeninja/vcs/go-radx/dicom/tag/generate_tag_values.py` logic to Go, reading the innolitics JSON)
- Create: `dicom/tag_values.go` (generated: `Tag*` constants, `tagDict map[Tag]TagInfo`, `repeatingDict` slice)
- Create: `dicom/uid_values.go` (generated: `uidNames` map, ~490 entries)
- Test: `dicom/tag_values_test.go`

The dictionary is generated, never hand-edited. Reuse the innolitics dataset and license attribution from the prototype
generator. The generator emits one `Tag*` constant per keyword (`TagPatientName Tag = 0x00100010`, …) plus the
`tagDict` used by `Lookup`. The license header (innolitics MIT) must be reproduced in the generated file.

**Ordering note — keep every task compiling.** The generated `tag_values.go` declares `tagDict map[Tag]TagInfo` and
`repeatingDict []repeatingEntry`, so the `TagInfo` and `repeatingEntry` types must exist *before* the generated file
references them. This task therefore creates `dicom/dictionary.go` with those two type definitions (and the `dictLen`
helper the test calls) in Step 3, then generates the data against them. Task 1.5 appends the lookup functions
(`Lookup`, `LookupKeyword`, `LookupKeywordTag`, the `keywordTag` index) to the same `dictionary.go` and enriches
`keywordFor`. After this task the package compiles; the `Lookup`-using tests live in Task 1.5.

- [ ] **Step 1: Write the failing test**

```go
package dicom

import "testing"

func TestGeneratedTagConstants(t *testing.T) {
	tests := map[string]struct {
		got  Tag
		want Tag
	}{
		"PatientID":         {TagPatientID, 0x00100020},
		"PatientName":       {TagPatientName, 0x00100010},
		"StudyInstanceUID":  {TagStudyInstanceUID, 0x0020000D},
		"SeriesInstanceUID": {TagSeriesInstanceUID, 0x0020000E},
		"SOPClassUID":       {TagSOPClassUID, 0x00080016},
		"SOPInstanceUID":    {TagSOPInstanceUID, 0x00080018},
		"PixelData":         {TagPixelData, 0x7FE00010},
	}
	for name, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("Tag%s = %#08x, want %#08x", name, uint32(tc.got), uint32(tc.want))
		}
	}
}

func TestDictionaryEntryCount(t *testing.T) {
	// The standard data dictionary is ~5,189 entries (reference doc, conformance).
	if n := dictLen(); n < 5000 {
		t.Errorf("dictionary has %d entries, expected ~5,189", n)
	}
}

func TestGeneratedUIDNames(t *testing.T) {
	if got := UID("1.2.840.10008.1.2").Name(); got != "Implicit VR Little Endian" {
		t.Errorf("Name() = %q, want Implicit VR Little Endian", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run 'TestGenerated|TestDictionary' -v`
Expected: FAIL — `undefined: TagPatientID`, `dictLen`.

- [ ] **Step 3: Define the dictionary types, then write the generator and generate the dictionaries**

First create `dicom/dictionary.go` with the type definitions the generated file references plus the `dictLen` helper the
test calls (the lookup functions are added by Task 1.5):

```go
package dicom

//go:generate go run ./gen/gentags

// TagInfo is a dictionary entry resolved from a Tag.
type TagInfo struct {
	Keyword string // e.g. "PatientName"
	VR      VR     // dictionary VR; ambiguous VRs use the ambiguous placeholders
	VM      string // value multiplicity, e.g. "1", "1-n", "2"
	Name    string // human-readable name from PS3.6
}

// repeatingEntry holds a masked dictionary entry for a repeating group.
type repeatingEntry struct {
	mask, value uint32 // (tag & mask) == value matches
	info        TagInfo
}

// dictLen reports the exact-entry count, used by the dictionary coverage test.
func dictLen() int { return len(tagDict) }
```

Then port `generate_tag_values.py` to a Go generator at `dicom/gen/gentags/main.go`, pinned to the same innolitics
revision hash the prototype used (`7f4749d09ed3ef2fa70637d376d423a4b13523cd`, rev2024b) for reproducibility. It reads
the innolitics `attributes.json` and `sop_classes.json` / `uid_dictionary.json`, and writes:

- `dicom/tag_values.go` — for each non-repeating attribute, a `const Tag<Keyword> Tag = 0x<group><element>`; plus a
  `tagDict` of type `map[Tag]TagInfo` for exact entries and a `repeatingDict []repeatingEntry` (mask + value + TagInfo)
  for `60xx`/`50xx`/`5xxx` (Task 1.5 consumes the repeating slice).
- `dicom/uid_values.go` — the `uidNames map[UID]string` (~490 entries), superseding the stub from Task 1.3.

Both generated files begin with `// Code generated by gen/gentags; DO NOT EDIT.` and the innolitics MIT license block.

- [ ] **Step 4: Run test to verify it passes**

Run: `go generate ./dicom/... && go test ./dicom/ -run 'TestGenerated|TestDictionary' -v`
Expected: PASS.

- [ ] **Step 5: Commit (generator and generated output separately)**

```bash
git add dicom/gen/gentags/main.go dicom/dictionary.go
git commit -m "build(dicom): add tag/UID dictionary generator (innolitics rev2024b)"
git add dicom/tag_values.go dicom/uid_values.go dicom/tag_values_test.go
git commit -m "feat(dicom): generate tag constants and UID name dictionary"
```

---

### Task 1.5: Dictionary lookup with repeating-group mask

**Files:**
- Modify: `dicom/dictionary.go` (append `Lookup`, `LookupKeyword`, `LookupKeywordTag`, the `keywordTag` index; the
  `TagInfo`/`repeatingEntry` types already exist from Task 1.4)
- Modify: `dicom/errors.go` (enrich `keywordFor` to resolve through `Lookup`)
- Test: `dicom/dictionary_test.go`

This closes Codex DCM-012: the prototype looked up tags exact-only and replaced the repeating `x` with `0`, so the
overlay tag `(6002,3000)` resolved as unknown and degraded to `UN` under Implicit VR. `Lookup` must mask the variable
nibbles of `60xx`/`50xx`/`5xxx` before falling back to unknown.

- [ ] **Step 1: Write the failing test**

```go
package dicom

import "testing"

func TestLookupExact(t *testing.T) {
	info, ok := Lookup(TagPatientName)
	if !ok {
		t.Fatal("PatientName should be in the dictionary")
	}
	if info.Keyword != "PatientName" || info.VR != VRPN {
		t.Errorf("Lookup(PatientName) = %+v, want keyword PatientName VR PN", info)
	}
}

func TestLookupRepeatingGroupMask(t *testing.T) {
	// (6002,3000) is Overlay Data in the repeating 60xx group; it must resolve, not
	// degrade to unknown (Codex DCM-012).
	info, ok := Lookup(NewTag(0x6002, 0x3000))
	if !ok {
		t.Fatal("(6002,3000) should resolve via the 60xx mask")
	}
	if info.Keyword != "OverlayData" {
		t.Errorf("Lookup((6002,3000)).Keyword = %q, want OverlayData", info.Keyword)
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup(NewTag(0x0011, 0x1234)); ok {
		t.Error("a genuinely unknown private tag should return ok == false")
	}
}

func TestLookupKeyword(t *testing.T) {
	got, ok := LookupKeyword("StudyInstanceUID")
	if !ok || got != TagStudyInstanceUID {
		t.Errorf("LookupKeyword(StudyInstanceUID) = (%s,%v), want (%s,true)", got, ok, TagStudyInstanceUID)
	}
	if _, ok := LookupKeyword("NotARealKeyword"); ok {
		t.Error("unknown keyword should return ok == false, not panic")
	}
}

func TestLookupKeywordTagPanicsOnTypo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("LookupKeywordTag should panic on a keyword not in the dictionary")
		}
	}()
	_ = LookupKeywordTag("DefinitelyNotAKeyword")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestLookup -v`
Expected: FAIL — `undefined: Lookup`, `LookupKeyword`, `LookupKeywordTag` (the `TagInfo`/`repeatingEntry` types already
exist from Task 1.4, so they are not in the undefined list).

- [ ] **Step 3: Append the lookup functions to `dictionary.go`**

`TagInfo` and `repeatingEntry` were defined in Task 1.4; this step only appends the resolver functions and the reverse
index to the same file (do not re-declare the types). Add an `import "fmt"` to the file's existing import block — Task
1.4 created `dictionary.go` without imports, so introduce the import statement here:

```go
// Lookup resolves a tag through the standard dictionary, resolving 60xx/50xx/5xxx
// repeating groups by mask. ok is false for genuinely unknown tags.
func Lookup(t Tag) (TagInfo, bool) {
	if info, ok := tagDict[t]; ok {
		return info, true
	}
	for _, re := range repeatingDict {
		if uint32(t)&re.mask == re.value {
			return re.info, true
		}
	}
	return TagInfo{}, false
}

// keywordTag is the reverse index built once from tagDict at init.
var keywordTag = func() map[string]Tag {
	m := make(map[string]Tag, len(tagDict))
	for tag, info := range tagDict {
		if info.Keyword != "" {
			m[info.Keyword] = tag
		}
	}
	return m
}()

// LookupKeyword resolves a keyword to its canonical tag for dynamic input,
// returning ok == false for an unknown keyword.
func LookupKeyword(keyword string) (Tag, bool) {
	t, ok := keywordTag[keyword]
	return t, ok
}

// LookupKeywordTag resolves a compile-time-literal keyword to its tag. It panics
// only on a keyword not in the standard dictionary — a programmer error, never
// reachable from external input.
func LookupKeywordTag(keyword string) Tag {
	t, ok := keywordTag[keyword]
	if !ok {
		panic(fmt.Sprintf("dicom: LookupKeywordTag: unknown keyword %q", keyword))
	}
	return t
}
```

Now that `Lookup` exists, enrich `keywordFor` in `dicom/errors.go` so diagnostics render the tag keyword (it returned
`""` as a self-contained stub in Task 1.3):

```go
// keywordFor renders a tag's keyword for diagnostics; falls back to "" if unknown.
func keywordFor(t Tag) string {
	if info, ok := Lookup(t); ok {
		return info.Keyword
	}
	return ""
}
```

The generator (Task 1.4) must emit the `repeatingDict` slice with `OverlayData` masked at `0xFF00FFFF == 0x60003000`.
If the generated file does not yet include `repeatingDict`, extend the generator and regenerate.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestLookup -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/dictionary.go dicom/errors.go dicom/dictionary_test.go
git commit -m "feat(dicom): add dictionary Lookup with repeating-group mask resolution"
```

---

### Task 1.6: `Decimal` lexical-preserving type

**Files:**
- Create: `dicom/decimal.go`
- Test: `dicom/decimal_test.go`

`Decimal` carries the source string so DS/IS round-trip byte-identically (reference doc; PRD §8.1). It beats the
prototype's `float64` mapping (lossy).

- [ ] **Step 1: Write the failing test**

```go
package dicom

import (
	"math/big"
	"testing"
)

func TestDecimalPreservesLexicalForm(t *testing.T) {
	for _, s := range []string{"1.500", "-0.0", "3.14159265358979", "100", "+2.5"} {
		d, err := ParseDecimal(s)
		if err != nil {
			t.Fatalf("ParseDecimal(%q): %v", s, err)
		}
		if d.String() != s {
			t.Errorf("String() = %q, want preserved %q", d.String(), s)
		}
	}
}

func TestDecimalFloat64AndExact(t *testing.T) {
	d, _ := ParseDecimal("1.5")
	if f, ok := d.Float64(); !ok || f != 1.5 {
		t.Errorf("Float64() = (%v,%v), want (1.5,true)", f, ok)
	}
	if !d.Exact() {
		t.Error("1.5 is exactly representable as float64")
	}
	d2, _ := ParseDecimal("0.1")
	if _, ok := d2.Float64(); !ok {
		t.Error("0.1 should return ok == true (representable but rounded)")
	}
	if d2.Exact() {
		t.Error("0.1 is not exactly representable as float64")
	}
}

func TestDecimalBigFloat(t *testing.T) {
	d, _ := ParseDecimal("3.14159265358979")
	bf := d.BigFloat()
	want, _, _ := big.ParseFloat("3.14159265358979", 10, bf.Prec(), big.ToNearestEven)
	if bf.Cmp(want) != 0 {
		t.Errorf("BigFloat() = %v, want %v", bf, want)
	}
}

func TestDecimalDSLengthLimit(t *testing.T) {
	// DS is limited to 16 bytes per value (PS3.5).
	if _, err := ParseDecimal("12345678901234567"); err == nil {
		t.Error("a 17-byte DS value should be rejected")
	}
}

func TestDecimalInt64(t *testing.T) {
	d, _ := ParseDecimal("42")
	if n, ok := d.Int64(); !ok || n != 42 {
		t.Errorf("Int64() = (%d,%v), want (42,true)", n, ok)
	}
	d2, _ := ParseDecimal("4.2")
	if _, ok := d2.Int64(); ok {
		t.Error("4.2 is not integral")
	}
}

func TestDecimalJSONRoundTrip(t *testing.T) {
	d, _ := ParseDecimal("1.500")
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "1.500" {
		t.Errorf("MarshalJSON() = %q, want unquoted 1.500 (FHIR decimal)", b)
	}
	var d2 Decimal
	if err := d2.UnmarshalJSON([]byte("1.500")); err != nil {
		t.Fatal(err)
	}
	if d2.String() != "1.500" {
		t.Errorf("round-trip lost lexical form: %q", d2.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestDecimal -v`
Expected: FAIL — `undefined: ParseDecimal`, `Decimal`.

- [ ] **Step 3: Write minimal implementation**

```go
package dicom

import (
	"fmt"
	"math/big"
	"strings"
)

// maxDSLen is the PS3.5 byte cap for a single DS value.
const maxDSLen = 16

// Decimal is the lexical-preserving numeric type shared by FHIR decimal and DICOM
// DS/IS. It carries the source string so a value read from a file serialises back
// byte-identically. It performs no in-place arithmetic; conversion to a Go numeric
// is explicit and may report inexactness.
type Decimal struct {
	lexical string     // preserved source form
	val     *big.Float // parsed once on construction
}

// ParseDecimal validates s as a DICOM DS/IS or FHIR decimal lexical form and
// preserves it verbatim. DS is limited to 16 bytes per value (PS3.5).
func ParseDecimal(s string) (Decimal, error) {
	if s == "" {
		return Decimal{}, &ValueError{VR: VRDS, Msg: "decimal is empty"}
	}
	if len(s) > maxDSLen {
		return Decimal{}, &ValueError{VR: VRDS, Msg: fmt.Sprintf("DS value exceeds 16 bytes (%d)", len(s))}
	}
	// big.Float parses the standard decimal/exponent forms; reject anything it cannot.
	bf, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return Decimal{}, &ValueError{VR: VRDS, Msg: fmt.Sprintf("not a decimal: %q", s)}
	}
	return Decimal{lexical: s, val: bf}, nil
}

// String returns the preserved lexical form.
func (d Decimal) String() string { return d.lexical }

// Float64 returns the value as a float64. ok is false only when the lexical form has
// no finite float64 representation; a representable-but-rounded value returns ok == true.
func (d Decimal) Float64() (float64, bool) {
	if d.val == nil {
		return 0, false
	}
	f, acc := d.val.Float64()
	if (f == 0 && acc == big.Below) || f != f { // NaN guard
		return 0, false
	}
	if f > 1.7976931348623157e308 || f < -1.7976931348623157e308 {
		return 0, false
	}
	return f, true
}

// Exact reports whether the float64 from Float64 represents d without rounding loss.
func (d Decimal) Exact() bool {
	if d.val == nil {
		return false
	}
	_, acc := d.val.Float64()
	return acc == big.Exact
}

// BigFloat returns a copy of the parsed *big.Float with precision sufficient for the
// lexical form, for callers that need exactness or their own rounding.
func (d Decimal) BigFloat() *big.Float {
	if d.val == nil {
		return new(big.Float)
	}
	return new(big.Float).Copy(d.val)
}

// Int64 returns the integral value. ok is false if d is not integral.
func (d Decimal) Int64() (int64, bool) {
	if d.val == nil || !d.val.IsInt() {
		return 0, false
	}
	n, acc := d.val.Int64()
	if acc != big.Exact {
		return 0, false
	}
	return n, true
}

// MarshalJSON emits the preserved lexical form, unquoted, for FHIR decimal.
func (d Decimal) MarshalJSON() ([]byte, error) {
	if d.lexical == "" {
		return []byte("null"), nil
	}
	return []byte(d.lexical), nil
}

// UnmarshalJSON preserves the raw token's lexical form (trimming any quotes a lenient
// producer added).
func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	parsed, err := ParseDecimal(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
```

Note: `ParseDecimal` validates DS-shaped lexical forms here. The reference doc also names an IS sub-range (a signed
32-bit integer expressed without a fractional part); IS-specific range checking is layered in by the `Decimals` value
constructor and the writer when the VR is known to be `IS`, since `ParseDecimal` alone does not carry the VR.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestDecimal -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/decimal.go dicom/decimal_test.go
git commit -m "feat(dicom): add lexical-preserving Decimal for DS/IS and FHIR decimal"
```

---

### Task 1.7: `PersonName` component model

**Files:**
- Create: `dicom/person_name.go`
- Test: `dicom/person_name_test.go`

Closes Codex DCM-011 (the prototype treated `PN` as a plain string, making ideographic/phonetic names and per-component
de-identification impossible).

- [ ] **Step 1: Write the failing test**

```go
package dicom

import "testing"

func TestParsePersonNameAlphabetic(t *testing.T) {
	pn, err := ParsePersonName("Doe^John^Q^Dr^Jr")
	if err != nil {
		t.Fatal(err)
	}
	a := pn.Alphabetic
	if a.FamilyName != "Doe" || a.GivenName != "John" || a.MiddleName != "Q" ||
		a.Prefix != "Dr" || a.Suffix != "Jr" {
		t.Errorf("components = %+v", a)
	}
}

func TestParsePersonNameThreeGroups(t *testing.T) {
	// alphabetic=Yamada^Tarou, ideographic, phonetic.
	pn, err := ParsePersonName("Yamada^Tarou=山田^太郎=yamada^tarou")
	if err != nil {
		t.Fatal(err)
	}
	if pn.Alphabetic.FamilyName != "Yamada" {
		t.Errorf("alphabetic family = %q", pn.Alphabetic.FamilyName)
	}
	if pn.Ideographic.FamilyName == "" {
		t.Error("ideographic group should be populated")
	}
	if pn.Phonetic.FamilyName != "yamada" {
		t.Errorf("phonetic family = %q", pn.Phonetic.FamilyName)
	}
}

func TestParsePersonNameRejectsTooManyGroups(t *testing.T) {
	if _, err := ParsePersonName("a=b=c=d"); err == nil {
		t.Error("more than three component groups should error")
	}
}

func TestPersonNameStringDropsTrailingEmpties(t *testing.T) {
	pn, _ := ParsePersonName("Doe^John^^^")
	if got := pn.String(); got != "Doe^John" {
		t.Errorf("String() = %q, want Doe^John", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestPersonName -v` and `-run TestParsePersonName`
Expected: FAIL — `undefined: ParsePersonName`, `PersonName`.

- [ ] **Step 3: Write minimal implementation**

```go
package dicom

import (
	"fmt"
	"strings"
)

// maxPNComponent is the per-component PS3.5 character cap for PN.
const maxPNComponent = 64

// NameComponents holds the five ^-delimited components of one PersonName group.
type NameComponents struct {
	FamilyName string
	GivenName  string
	MiddleName string
	Prefix     string
	Suffix     string
}

// PersonName is VR PN: up to three =-delimited component groups (alphabetic,
// ideographic, phonetic), each holding up to five ^-delimited components.
type PersonName struct {
	Alphabetic  NameComponents
	Ideographic NameComponents // empty if absent
	Phonetic    NameComponents // empty if absent
}

// ParsePersonName splits s on "=" into up to three groups, each on "^" into up to
// five components, trimming the standard pad. It errors on more than three groups or
// more than five components in a group, or a component over 64 characters.
func ParsePersonName(s string) (PersonName, error) {
	groups := strings.Split(s, "=")
	if len(groups) > 3 {
		return PersonName{}, &ValueError{VR: VRPN, Msg: fmt.Sprintf("PN has %d component groups, max 3", len(groups))}
	}
	var pn PersonName
	dst := []*NameComponents{&pn.Alphabetic, &pn.Ideographic, &pn.Phonetic}
	for i, g := range groups {
		comps := strings.Split(g, "^")
		if len(comps) > 5 {
			return PersonName{}, &ValueError{VR: VRPN, Msg: fmt.Sprintf("PN group %d has %d components, max 5", i+1, len(comps))}
		}
		for _, c := range comps {
			if len(strings.TrimRight(c, " ")) > maxPNComponent {
				return PersonName{}, &ValueError{VR: VRPN, Msg: fmt.Sprintf("PN group %d component exceeds 64 characters", i+1)}
			}
		}
		fields := [5]*string{&dst[i].FamilyName, &dst[i].GivenName, &dst[i].MiddleName, &dst[i].Prefix, &dst[i].Suffix}
		for j, c := range comps {
			*fields[j] = strings.TrimRight(c, " ")
		}
	}
	return pn, nil
}

// String renders the canonical "=" / "^" form, dropping trailing empty components and
// trailing empty groups (so "Doe^John" not "Doe^John^^^==").
func (p PersonName) String() string {
	groups := [3]NameComponents{p.Alphabetic, p.Ideographic, p.Phonetic}
	rendered := make([]string, 0, 3)
	for _, g := range groups {
		comps := []string{g.FamilyName, g.GivenName, g.MiddleName, g.Prefix, g.Suffix}
		end := len(comps)
		for end > 0 && comps[end-1] == "" {
			end-- // drop trailing empty components
		}
		rendered = append(rendered, strings.Join(comps[:end], "^"))
	}
	end := len(rendered)
	for end > 0 && rendered[end-1] == "" {
		end-- // drop trailing empty groups
	}
	return strings.Join(rendered[:end], "=")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run 'TestPersonName|TestParsePersonName' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/person_name.go dicom/person_name_test.go
git commit -m "feat(dicom): add PersonName with three component groups"
```

---

### Task 1.8: `UIDGenerator` (fail-closed root, no default)

**Files:**
- Create: `dicom/uid_generator.go`
- Test: `dicom/uid_generator_test.go`

Closes Codex DCM-008: the prototype minted under PixelMed's registered root `1.2.826.0.1.3680043.10`, mislabelling
go-radx output. go-radx ships **no** default registered root; `NewUIDGenerator` fails closed on an empty/invalid root,
and `NewRandomUIDGenerator` uses the `2.25.` UUID-derived root that needs no registration.

- [ ] **Step 1: Write the failing test**

```go
package dicom

import (
	"strings"
	"testing"
)

func TestNewUIDGeneratorRejectsEmptyRoot(t *testing.T) {
	if _, err := NewUIDGenerator(""); err == nil {
		t.Error("empty root must fail closed (no default registered root, Codex DCM-008)")
	}
}

func TestNewUIDGeneratorRejectsOverlongRoot(t *testing.T) {
	// Root must be <= 54 chars to leave room for a >= 9-digit suffix within 64.
	long := UID("1." + strings.Repeat("2.", 27)) // > 54 chars
	if _, err := NewUIDGenerator(long); err == nil {
		t.Error("over-54-char root must be rejected")
	}
}

func TestGenerateUnderConfiguredRoot(t *testing.T) {
	g, err := NewUIDGenerator("1.2.826.0.1.3680043.2.1143")
	if err != nil {
		t.Fatal(err)
	}
	u := g.Generate()
	if !strings.HasPrefix(string(u), "1.2.826.0.1.3680043.2.1143.") {
		t.Errorf("Generate() = %q, want the configured root prefix", u)
	}
	if !u.IsValid() {
		t.Errorf("Generate() produced an invalid UID: %q", u)
	}
	if len(u) > 64 {
		t.Errorf("Generate() exceeded 64 chars: %d", len(u))
	}
}

func TestGenerateIsUnique(t *testing.T) {
	g, _ := NewUIDGenerator("1.2.826.0.1.3680043.2.1143")
	seen := make(map[UID]bool)
	for i := 0; i < 1000; i++ {
		u := g.Generate()
		if seen[u] {
			t.Fatalf("duplicate UID minted: %q", u)
		}
		seen[u] = true
	}
}

func TestRandomUIDGeneratorUses225Root(t *testing.T) {
	g := NewRandomUIDGenerator()
	u := g.Generate()
	if !strings.HasPrefix(string(u), "2.25.") {
		t.Errorf("Generate() = %q, want 2.25. UUID root", u)
	}
	if !u.IsValid() || len(u) > 64 {
		t.Errorf("invalid 2.25. UID: %q (len %d)", u, len(u))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run 'TestUIDGenerator|TestGenerate|TestNewUIDGenerator|TestRandomUID' -v`
Expected: FAIL — `undefined: NewUIDGenerator`, `UIDGenerator`, `NewRandomUIDGenerator`.

- [ ] **Step 3: Write minimal implementation**

Implement per the reference doc: `NewUIDGenerator(root UID)` validates the root through `UID.Validate` and the 54-char
cap; `Generate` appends a cryptographically random numeric suffix sized to fill the 64-char field;
`NewRandomUIDGenerator` roots at `2.25.` with the integer form of a random UUID (mirrors pydicom `generate_uid`). Use
`crypto/rand`. Full code follows the committed signatures.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run 'TestUIDGenerator|TestGenerate|TestNewUIDGenerator|TestRandomUID' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/uid_generator.go dicom/uid_generator_test.go
git commit -m "feat(dicom): add fail-closed UIDGenerator with no default registered root"
```

---

### Task 1.9: `SOPClassUID` / `SOPInstanceUID` named types

**Files:**
- Modify: `dicom/uid.go` (append the two named types)
- Test: `dicom/uid_test.go` (extend)

- [ ] **Step 1: Write the failing test**

```go
func TestSOPIdentifierTypesInheritValidation(t *testing.T) {
	sc := SOPClassUID("1.2.840.10008.5.1.4.1.1.2")
	if err := UID(sc).Validate(); err != nil {
		t.Errorf("SOPClassUID should validate through UID conversion: %v", err)
	}
	if UID(sc).Name() != "CT Image Storage" {
		t.Errorf("Name() = %q, want CT Image Storage", UID(sc).Name())
	}
	si := SOPInstanceUID("1..2") // invalid
	if UID(si).IsValid() {
		t.Error("invalid SOPInstanceUID should not validate")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestSOPIdentifierTypes -v`
Expected: FAIL — `undefined: SOPClassUID`, `SOPInstanceUID`.

- [ ] **Step 3: Write minimal implementation**

```go
// SOPClassUID identifies a SOP Class (an IOD + its DIMSE services). It is a named
// type over UID so a signature states the SOP role; dimse, dicomweb, and convert
// reuse it. Validation is inherited via conversion: UID(sc).Validate().
type SOPClassUID UID

// SOPInstanceUID identifies one concrete SOP Instance.
type SOPInstanceUID UID
```

(The CT Image Storage name resolves through the generated `uidNames` dictionary from Task 1.4.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestSOPIdentifierTypes -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/uid.go dicom/uid_test.go
git commit -m "feat(dicom): add SOPClassUID and SOPInstanceUID named types"
```

---

### Task 1.10: `Value` interface and `Strings`/`Ints`/`Floats`/`Decimals`/`Tags` values

**Files:**
- Create: `dicom/value.go`
- Test: `dicom/value_test.go`

`Value` exposes the on-wire VR and the encoded (padded, even) length; `EncodedLen` never panics. This is where VR-correct
padding (Task 1.2) is applied, closing the length half of Codex DCM-007.

**Scope note.** This task delivers the `Value` interface and the `Strings`, `Ints`, `Floats`, `Decimals`, and `Tags`
value types only. `NewBytes`/`Bytes` is deferred to Task 1.11. `NewSequenceValue` references the `Sequence` type, which
does not exist until Increment 3, so it is deferred there. The `Value` interface is intentionally open to extension:
later increments add `Bytes`, `Sequence`, and `PixelData` implementations without changing the interface.

- [ ] **Step 1: Write the failing test**

```go
package dicom

import (
	"encoding/binary"
	"testing"
)

func TestStringsValueEncodedLenIsEven(t *testing.T) {
	// "Doe^Jane" is 8 bytes (even). "ABC" is 3 -> padded to 4 with SPACE.
	tests := []struct {
		vr   VR
		vals []string
		want uint32
	}{
		{VRPN, []string{"Doe^Jane"}, 8},
		{VRCS, []string{"ABC"}, 4},                 // SPACE-padded
		{VRUI, []string{"1.2.840.10008.1.2"}, 18},  // 17 chars -> NULL-padded to 18
		{VRLO, []string{"a", "bb"}, 4},             // "a\bb" = 4 bytes, even
	}
	for _, tc := range tests {
		v := NewStrings(tc.vr, tc.vals...)
		if got := v.EncodedLen(binary.LittleEndian); got != tc.want {
			t.Errorf("NewStrings(%s,%v).EncodedLen = %d, want %d", tc.vr, tc.vals, got, tc.want)
		}
		if got := v.EncodedLen(binary.LittleEndian); got%2 != 0 {
			t.Errorf("%s encoded length %d is odd (Codex DCM-007)", tc.vr, got)
		}
	}
}

func TestIntsValueEncodedLen(t *testing.T) {
	if got := NewInts(VRUS, 1, 2, 3).EncodedLen(binary.LittleEndian); got != 6 {
		t.Errorf("US x3 EncodedLen = %d, want 6", got)
	}
	if got := NewInts(VRUL, 1).EncodedLen(binary.LittleEndian); got != 4 {
		t.Errorf("UL EncodedLen = %d, want 4", got)
	}
	if got := NewInts(VRSV, 1).EncodedLen(binary.LittleEndian); got != 8 {
		t.Errorf("SV EncodedLen = %d, want 8", got)
	}
}

func TestValueVR(t *testing.T) {
	if NewStrings(VRPN, "x").VR() != VRPN {
		t.Error("VR() should report PN")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run 'TestStringsValue|TestIntsValue|TestValueVR' -v`
Expected: FAIL — `undefined: NewStrings`, `Value`.

- [ ] **Step 3: Write minimal implementation**

```go
package dicom

import (
	"encoding/binary"
	"strings"
)

// Value is the interface every element value implements. It exposes the on-wire VR
// and the even, padded value-field length. EncodedLen never panics, and tolerates a
// nil byte order for zero-length values (used by SetEmpty). It is open to extension:
// later increments add Bytes, Sequence, and PixelData implementations.
type Value interface {
	VR() VR
	EncodedLen(bo binary.ByteOrder) uint32
}

// Strings is the value type for the text VRs (AE AS CS LO SH UC UR UT ST LT DA TM DT
// PN UI). Values are stored decoded; SpecificCharacterSet (Increment 4) governs the
// byte encoding.
type Strings struct {
	vr   VR
	vals []string
}

// NewStrings constructs a text value under vr.
func NewStrings(vr VR, vals ...string) Value {
	cp := make([]string, len(vals))
	copy(cp, vals)
	return &Strings{vr: vr, vals: cp}
}

func (v *Strings) VR() VR { return v.vr }

// Strings returns a copy of the value list.
func (v *Strings) Strings() []string {
	cp := make([]string, len(v.vals))
	copy(cp, v.vals)
	return cp
}

// EncodedLen joins the values with backslash and pads the whole field to an even
// length with the VR pad byte (Codex DCM-007: the entire character field is even).
func (v *Strings) EncodedLen(binary.ByteOrder) uint32 {
	if len(v.vals) == 0 {
		return 0
	}
	n := uint32(len(strings.Join(v.vals, `\`)))
	if n%2 == 1 {
		if _, ok := v.vr.PadByte(); ok {
			n++
		}
	}
	return n
}

// Ints is the value type for SS US SL UL SV UV.
type Ints struct {
	vr   VR
	vals []int64
}

// NewInts constructs an integer value under vr.
func NewInts(vr VR, vals ...int64) Value {
	cp := make([]int64, len(vals))
	copy(cp, vals)
	return &Ints{vr: vr, vals: cp}
}

func (v *Ints) VR() VR { return v.vr }

// Ints returns a copy of the value list.
func (v *Ints) Ints() []int64 {
	cp := make([]int64, len(v.vals))
	copy(cp, v.vals)
	return cp
}

func (v *Ints) EncodedLen(binary.ByteOrder) uint32 {
	return uint32(len(v.vals)) * uint32(intSize(v.vr))
}

// intSize is the per-element byte width of an integer VR.
func intSize(vr VR) int {
	switch vr {
	case VRSS, VRUS:
		return 2
	case VRSL, VRUL:
		return 4
	case VRSV, VRUV:
		return 8
	default:
		return 0
	}
}

// Floats is the value type for FL FD OF OD.
type Floats struct {
	vr   VR
	vals []float64
}

// NewFloats constructs a floating-point value under vr.
func NewFloats(vr VR, vals ...float64) Value {
	cp := make([]float64, len(vals))
	copy(cp, vals)
	return &Floats{vr: vr, vals: cp}
}

func (v *Floats) VR() VR { return v.vr }

// Floats returns a copy of the value list.
func (v *Floats) Floats() []float64 {
	cp := make([]float64, len(v.vals))
	copy(cp, v.vals)
	return cp
}

func (v *Floats) EncodedLen(binary.ByteOrder) uint32 {
	size := 4
	if v.vr == VRFD || v.vr == VROD {
		size = 8
	}
	return uint32(len(v.vals)) * uint32(size)
}

// Decimals is the value type for DS and IS, carrying lexical-preserving Decimals.
type Decimals struct {
	vr   VR
	vals []Decimal
}

// NewDecimals constructs a DS/IS value under vr.
func NewDecimals(vr VR, vals ...Decimal) Value {
	cp := make([]Decimal, len(vals))
	copy(cp, vals)
	return &Decimals{vr: vr, vals: cp}
}

func (v *Decimals) VR() VR { return v.vr }

// Decimals returns a copy of the value list.
func (v *Decimals) Decimals() []Decimal {
	cp := make([]Decimal, len(v.vals))
	copy(cp, v.vals)
	return cp
}

// EncodedLen joins the preserved lexical forms with backslash and pads to even with
// SPACE (DS/IS pad with SPACE per PS3.5; see Task 1.2 note).
func (v *Decimals) EncodedLen(binary.ByteOrder) uint32 {
	if len(v.vals) == 0 {
		return 0
	}
	parts := make([]string, len(v.vals))
	for i, d := range v.vals {
		parts[i] = d.String()
	}
	n := uint32(len(strings.Join(parts, `\`)))
	if n%2 == 1 {
		n++
	}
	return n
}

// Tags is the value type for AT (each value is a 4-byte tag).
type Tags struct {
	vals []Tag
}

// NewTags constructs an AT value.
func NewTags(vals ...Tag) Value {
	cp := make([]Tag, len(vals))
	copy(cp, vals)
	return &Tags{vals: cp}
}

func (v *Tags) VR() VR { return VRAT }

// Tags returns a copy of the value list.
func (v *Tags) Tags() []Tag {
	cp := make([]Tag, len(v.vals))
	copy(cp, v.vals)
	return cp
}

func (v *Tags) EncodedLen(binary.ByteOrder) uint32 { return uint32(len(v.vals)) * 4 }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run 'TestStringsValue|TestIntsValue|TestValueVR' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/value.go dicom/value_test.go
git commit -m "feat(dicom): add Value interface and string/int/float/decimal/tag values with even encoded length"
```

---

### Task 1.11: `Bytes` value with defensive copy

**Files:**
- Modify: `dicom/value.go` (add `Bytes` value + `NewBytes`)
- Test: `dicom/value_test.go` (extend)

`NewBytes` copies its input so a later mutation of the caller's slice cannot reach a stored element (part of Codex
DCM-016).

- [ ] **Step 1: Write the failing test**

```go
func TestNewBytesCopiesInput(t *testing.T) {
	src := []byte{1, 2, 3, 4}
	v := NewBytes(VROB, src)
	src[0] = 0xFF // mutate the caller's slice after construction
	bv, ok := v.(*Bytes)
	if !ok {
		t.Fatal("NewBytes should return *Bytes")
	}
	if got := bv.Bytes(); got[0] != 1 {
		t.Errorf("value aliased the caller's slice: got[0] = %#x, want 1 (Codex DCM-016)", got[0])
	}
}

func TestBytesAccessorReturnsCopy(t *testing.T) {
	v := NewBytes(VROB, []byte{1, 2, 3, 4})
	bv := v.(*Bytes)
	out := bv.Bytes()
	out[0] = 0xFF // mutate the returned slice
	if bv.Bytes()[0] != 1 {
		t.Error("Bytes() returned an internal slice that callers can mutate (Codex DCM-016)")
	}
}

func TestBytesEncodedLenEvenPadded(t *testing.T) {
	// Odd OB length pads to even with a trailing NULL.
	if got := NewBytes(VROB, []byte{1, 2, 3}).EncodedLen(binary.LittleEndian); got != 4 {
		t.Errorf("odd OB EncodedLen = %d, want 4 (even-padded)", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestBytes -v` and `-run TestNewBytes`
Expected: FAIL — `undefined: Bytes`, `NewBytes`.

- [ ] **Step 3: Write minimal implementation**

```go
// Bytes is the value type for OB OW OL OV UN (length-bounded raw bytes).
type Bytes struct {
	vr VR
	b  []byte // owned; never aliased to a caller slice
}

// NewBytes copies b so the value owns its bytes.
func NewBytes(vr VR, b []byte) Value {
	cp := make([]byte, len(b))
	copy(cp, b)
	return &Bytes{vr: vr, b: cp}
}

func (v *Bytes) VR() VR { return v.vr }

// Bytes returns a copy of the value field.
func (v *Bytes) Bytes() []byte {
	cp := make([]byte, len(v.b))
	copy(cp, v.b)
	return cp
}

// EncodedLen is the byte length padded up to even with a trailing NULL.
func (v *Bytes) EncodedLen(binary.ByteOrder) uint32 {
	n := uint32(len(v.b))
	if n%2 == 1 {
		n++
	}
	return n
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run 'TestBytes|TestNewBytes' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/value.go dicom/value_test.go
git commit -m "feat(dicom): add Bytes value with defensive copy on construct and access"
```

---

### Task 1.12: `DataSet` and `Element` — core accessors

**Files:**
- Create: `dicom/dataset.go`
- Test: `dicom/dataset_test.go`

`DataSet` preserves ascending-tag order. Document explicitly that it is not safe for concurrent mutation (Codex DCM-016
resolution: document rather than lock the single-threaded hot path).

- [ ] **Step 1: Write the failing test**

```go
package dicom

import (
	"slices"
	"testing"
)

func TestDataSetGetSetDelete(t *testing.T) {
	ds := NewDataSet()
	if _, ok := ds.Get(TagPatientName); ok {
		t.Error("empty dataset should return ok == false")
	}
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})
	e, ok := ds.Get(TagPatientName)
	if !ok || e.VR != VRPN {
		t.Errorf("Get(PatientName) = (%+v,%v)", e, ok)
	}
	if ds.Len() != 1 {
		t.Errorf("Len() = %d, want 1", ds.Len())
	}
	ds.Delete(TagPatientName)
	if _, ok := ds.Get(TagPatientName); ok {
		t.Error("Delete should remove the element")
	}
	ds.Delete(TagPatientName) // not an error when absent
}

func TestDataSetAllAscendingTagOrder(t *testing.T) {
	ds := NewDataSet()
	// Insert out of order; All must yield ascending.
	ds.Set(Element{Tag: TagPixelData, VR: VROW, Value: NewBytes(VROW, []byte{0, 0})})
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "a")})
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2")})
	var got []Tag
	for e := range ds.All() {
		got = append(got, e.Tag)
	}
	want := []Tag{TagPatientName, TagStudyInstanceUID, TagPixelData}
	if !slices.Equal(got, want) {
		t.Errorf("All() order = %v, want ascending %v", got, want)
	}
}

func TestDataSetSetReplaces(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "a")})
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "b")})
	if ds.Len() != 1 {
		t.Errorf("Set on same tag should replace, Len() = %d", ds.Len())
	}
	e, _ := ds.Get(TagPatientName)
	if s, _ := ds.GetString(TagPatientName); s != "b" {
		t.Errorf("replaced value = %q, want b (element VR %s)", s, e.VR)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestDataSet -v`
Expected: FAIL — `undefined: NewDataSet`, `DataSet`, `Element`.

- [ ] **Step 3: Write minimal implementation**

Implement `DataSet` over an internal `map[Tag]Element` plus a sorted-tag slice (or a single ordered structure), with the
doc comment stating it is not safe for concurrent mutation. `All` returns an `iter.Seq[Element]` in ascending tag order.
`GetString` is needed by the test — provide the minimal version (first value of the text element). Full accessor set
(`Get`, `Set`, `Delete`, `Len`, `All`) follows the reference doc signatures.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestDataSet -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/dataset.go dicom/dataset_test.go
git commit -m "feat(dicom): add DataSet and Element with ordered accessors (documented non-concurrent)"
```

---

### Task 1.13: `DataSet` convenience mutators — `SetString`, `SetEmpty`

**Files:**
- Modify: `dicom/dataset.go`
- Test: `dicom/dataset_test.go` (extend)

- [ ] **Step 1: Write the failing test**

```go
func TestSetStringUsesDictionaryVR(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagPatientName, "Doe^Jane")
	e, ok := ds.Get(TagPatientName)
	if !ok || e.VR != VRPN { // dictionary VR for PatientName is PN
		t.Errorf("SetString should use dictionary VR PN, got %s", e.VR)
	}
}

func TestSetEmptyDeclaresZeroLengthReturnKey(t *testing.T) {
	// The C-FIND universal-match idiom.
	ds := NewDataSet()
	ds.SetEmpty(TagStudyDescription)
	e, ok := ds.Get(TagStudyDescription)
	if !ok {
		t.Fatal("SetEmpty should insert the element")
	}
	if e.Value.EncodedLen(nil) != 0 {
		t.Errorf("SetEmpty value length = %d, want 0", e.Value.EncodedLen(nil))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run 'TestSetString|TestSetEmpty' -v`
Expected: FAIL — `ds.SetString undefined`.

- [ ] **Step 3: Write minimal implementation**

```go
// SetString looks up t's dictionary VR and inserts (or replaces) a text element
// carrying vals.
func (ds *DataSet) SetString(t Tag, vals ...string) {
	vr := dictVR(t)
	ds.Set(Element{Tag: t, VR: vr, Value: NewStrings(vr, vals...)})
}

// SetEmpty inserts (or replaces) a zero-length element at t under its dictionary VR.
func (ds *DataSet) SetEmpty(t Tag) {
	vr := dictVR(t)
	ds.Set(Element{Tag: t, VR: vr, Value: NewStrings(vr)})
}

// dictVR resolves the dictionary VR for t, defaulting to UN for unknown tags.
func dictVR(t Tag) VR {
	if info, ok := Lookup(t); ok {
		return info.VR
	}
	return VRUN
}
```

(`NewStrings(vr)` with no values produces a zero-length value; ensure `EncodedLen` returns 0 and tolerates a nil byte
order for the empty case.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run 'TestSetString|TestSetEmpty' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/dataset.go dicom/dataset_test.go
git commit -m "feat(dicom): add SetString and SetEmpty convenience mutators"
```

---

### Task 1.14: `DataSet.Clone` deep copy

**Files:**
- Modify: `dicom/dataset.go`
- Test: `dicom/dataset_test.go` (extend)

Closes Codex DCM-016: the prototype's "copy" reused the same value objects, allowing cross-dataset contamination. `Clone`
must deep-copy value slices.

- [ ] **Step 1: Write the failing test**

```go
func TestCloneIsDeep(t *testing.T) {
	src := NewDataSet()
	src.Set(Element{Tag: TagPixelData, VR: VROB, Value: NewBytes(VROB, []byte{1, 2, 3, 4})})

	clone := src.Clone()
	// Mutate the clone's element; the source must be untouched.
	clone.Set(Element{Tag: TagPixelData, VR: VROB, Value: NewBytes(VROB, []byte{9, 9})})

	se, _ := src.Get(TagPixelData)
	sb := se.Value.(*Bytes).Bytes()
	if len(sb) != 4 || sb[0] != 1 {
		t.Errorf("Clone aliased source value: source bytes = %v (Codex DCM-016)", sb)
	}
}

func TestCloneLengthMatches(t *testing.T) {
	src := NewDataSet()
	src.SetString(TagPatientName, "a")
	src.SetString(TagStudyInstanceUID, "1.2")
	if src.Clone().Len() != src.Len() {
		t.Error("Clone should preserve element count")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestClone -v`
Expected: FAIL — `src.Clone undefined`.

- [ ] **Step 3: Write minimal implementation**

`Clone` iterates `All`, deep-copying each value. For `Bytes` and `Strings`, construct a fresh value (the constructors
already copy). For `Sequence` values, recurse (Increment 3 extends this; for now sequences are not yet present, so a
straight value re-construction suffices and is extended when `Sequence` lands). Document that `Clone` is the safe path
for de-identification and conversion.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestClone -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/dataset.go dicom/dataset_test.go
git commit -m "feat(dicom): add DataSet.Clone deep copy guarding against value aliasing"
```

---

### Task 1.15: Typed convenience getters

**Files:**
- Modify: `dicom/dataset.go`
- Test: `dicom/dataset_test.go` (extend)

The reference doc commits `GetString`, `GetStrings`, `GetUID`, `GetInt`, `GetDecimal`, `GetPersonName` (and
`GetSequence`, deferred to Increment 3). They return model types, not raw bytes.

- [ ] **Step 1: Write the failing test**

```go
func TestTypedGetters(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.1.2.1")})
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})
	d, _ := ParseDecimal("1.5")
	ds.Set(Element{Tag: LookupKeywordTag("SliceThickness"), VR: VRDS, Value: NewDecimals(VRDS, d)})
	ds.Set(Element{Tag: LookupKeywordTag("Rows"), VR: VRUS, Value: NewInts(VRUS, 512)})

	if u, ok := ds.GetUID(TagStudyInstanceUID); !ok || u != "1.2.840.10008.1.2.1" {
		t.Errorf("GetUID = (%q,%v)", u, ok)
	}
	if pn, ok := ds.GetPersonName(TagPatientName); !ok || pn.Alphabetic.FamilyName != "Doe" {
		t.Errorf("GetPersonName = (%+v,%v)", pn, ok)
	}
	if dec, ok := ds.GetDecimal(LookupKeywordTag("SliceThickness")); !ok || dec.String() != "1.5" {
		t.Errorf("GetDecimal = (%v,%v)", dec, ok)
	}
	if n, ok := ds.GetInt(LookupKeywordTag("Rows")); !ok || n != 512 {
		t.Errorf("GetInt = (%d,%v)", n, ok)
	}
	if _, ok := ds.GetString(TagPatientID); ok {
		t.Error("absent tag should return ok == false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dicom/ -run TestTypedGetters -v`
Expected: FAIL — `ds.GetUID undefined`, etc.

- [ ] **Step 3: Write minimal implementation**

Each getter fetches the element, type-switches on its `Value`, and returns the model type with `ok` semantics. `GetString`
returns the first value of a text VR (charset decoding becomes real in Increment 4); `GetStrings` returns all values.

```go
// GetString returns the first value of a text VR element, charset-decoded.
func (ds *DataSet) GetString(t Tag) (string, bool) {
	vals, ok := ds.GetStrings(t)
	if !ok || len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// GetStrings returns all backslash-separated values of a text VR element.
func (ds *DataSet) GetStrings(t Tag) ([]string, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return nil, false
	}
	sv, ok := e.Value.(*Strings)
	if !ok {
		return nil, false
	}
	return sv.Strings(), true
}

// GetUID returns the first UI value parsed as a UID.
func (ds *DataSet) GetUID(t Tag) (UID, bool) {
	s, ok := ds.GetString(t)
	if !ok {
		return "", false
	}
	return UID(s), true
}

// GetInt returns the first integer value of an IS/SS/US/SL/UL/SV/UV element.
func (ds *DataSet) GetInt(t Tag) (int64, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return 0, false
	}
	switch v := e.Value.(type) {
	case *Ints:
		ns := v.Ints()
		if len(ns) == 0 {
			return 0, false
		}
		return ns[0], true
	case *Decimals: // IS is carried as a Decimal
		ds := v.Decimals()
		if len(ds) == 0 {
			return 0, false
		}
		return ds[0].Int64()
	default:
		return 0, false
	}
}

// GetDecimal returns the first DS value.
func (ds *DataSet) GetDecimal(t Tag) (Decimal, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return Decimal{}, false
	}
	dv, ok := e.Value.(*Decimals)
	if !ok {
		return Decimal{}, false
	}
	vals := dv.Decimals()
	if len(vals) == 0 {
		return Decimal{}, false
	}
	return vals[0], true
}

// GetPersonName returns the first PN value parsed into component groups.
func (ds *DataSet) GetPersonName(t Tag) (PersonName, bool) {
	s, ok := ds.GetString(t)
	if !ok {
		return PersonName{}, false
	}
	pn, err := ParsePersonName(s)
	if err != nil {
		return PersonName{}, false
	}
	return pn, true
}
```

`GetSequence` is deferred to Increment 3 (the `Sequence` type does not exist yet).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dicom/ -run TestTypedGetters -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dicom/dataset.go dicom/dataset_test.go
git commit -m "feat(dicom): add typed convenience getters (GetUID/GetInt/GetDecimal/GetPersonName)"
```

---

### Task 1.16: Increment 1 integration check

**Files:**
- Test: `dicom/dataset_test.go` (add an end-to-end build test)

- [ ] **Step 1: Write the failing test**

Mirror the reference doc's "Build a dataset" worked example as a test: build a `DataSet` with `PatientName`,
`StudyInstanceUID` minted by a `UIDGenerator`, read the values back through the typed getters, and assert they round-trip
through the type system (no I/O yet).

- [ ] **Step 2: Run it**

Run: `go test -race ./dicom/ -v`
Expected: all Increment 1 tests PASS under the race detector.

- [ ] **Step 3: Verify vet and lint**

Run: `go vet ./dicom/... && golangci-lint run ./dicom/...`
Expected: no findings.

- [ ] **Step 4: Commit**

```bash
git add dicom/dataset_test.go
git commit -m "test(dicom): add core type-system integration test for the build worked example"
```

**Verification gate for Increment 1:** `mise run test:dicom` green under `-race`; `go vet` and `golangci-lint` clean;
Codex DCM-007, DCM-008, DCM-009, DCM-011, DCM-012, DCM-016 each have a named passing regression test.

---

## Increment 2 — Part 10 read/write (uncompressed) — outline

**Scope:** The Part 10 container and the four uncompressed transfer-syntax codecs. Reading: 128-byte preamble, `DICM`
magic, the group-0002 `FileMeta` (always Explicit VR LE) with strict group-length boundary validation, then the main
dataset in the declared transfer syntax. Writing: emit preamble, `FileMeta` with an auto-recomputed `(0002,0000)` group
length written *first*, then the main dataset in the declared syntax. A bounded reader underlies all parsing so no length
reaches `make` unchecked.

**Key types/functions (from `docs/reference/dicom.md`):**

```go
type TransferSyntax UID
const (
	ImplicitVRLittleEndian         TransferSyntax = "1.2.840.10008.1.2"
	ExplicitVRLittleEndian         TransferSyntax = "1.2.840.10008.1.2.1"
	DeflatedExplicitVRLittleEndian TransferSyntax = "1.2.840.10008.1.2.1.99"
	ExplicitVRBigEndian            TransferSyntax = "1.2.840.10008.1.2.2"
)
func (ts TransferSyntax) IsImplicitVR() bool
func (ts TransferSyntax) IsBigEndian() bool
func (ts TransferSyntax) IsDeflated() bool

type File struct { Preamble [128]byte; Meta *FileMeta; DataSet *DataSet }
type FileMeta struct {
	MediaStorageSOPClassUID    SOPClassUID
	MediaStorageSOPInstanceUID SOPInstanceUID
	TransferSyntaxUID          TransferSyntax
	ImplementationClassUID     UID
	Elements                   *DataSet
}
func ReadFile(path string, opts ...ReadOption) (*File, error)
func Read(r io.Reader, opts ...ReadOption) (*File, error)
func WriteFile(path string, f *File, opts ...WriteOption) error
func Write(w io.Writer, f *File, opts ...WriteOption) error
func (ds *DataSet) WriteFile(path string, ts TransferSyntax, opts ...WriteOption) error
func WithMaxElementLen(n uint32) ReadOption
func WithMaxSequenceDepth(n int) ReadOption
func WithStopAtPixelData() ReadOption
```

**Dependencies:** Increment 1 (all types). Increment 3 (sequences) is read/written by the same element loop, so Increment
2 builds the loop with a clean extension point for `SQ` and lands flat (non-sequence) elements first.

**Tasks (to be expanded):** bounded reader with checked length conversion; `TransferSyntax` predicates; explicit-VR-LE
element read/write round-trip; implicit-VR-LE read/write; big-endian byte-order-aware codec; deflated dataset
read/write; `FileMeta` parse + group-length boundary validation; `FileMeta` write with auto-recomputed group length;
preamble + `DICM` handling; `DataSet.WriteFile` convenience deriving file-meta UIDs from `(0008,0016)`/`(0008,0018)`;
truncation-is-failure regression; hostile-length regression. The explicit-VR-LE writer round-trip task must include a
**byte-level pad assertion**: write an odd-length value for each character VR (`AE`/`CS`/`DA`/`DS`/`IS`/`LO`/`PN` with
SPACE, `UI` with NULL) and assert the emitted value field is even-length and ends in the correct pad byte, completing
the write half of DCM-007 (Increment 1 proved only the `EncodedLen`/`PadByte` length computation; this proves the bytes
are actually emitted padded).

**Codex defects guarded:** DCM-001 (group length written first, exact byte count), DCM-002 (transfer-syntax-faithful
encoder: byte order + deflate, reject unsupported before writing), DCM-003 (EOF only at a clean top-level tag boundary;
`io.ErrUnexpectedEOF` inside any value/item propagated), DCM-004 (every 32-bit length validated against bytes remaining
in a bounded reader before allocation), DCM-007 (write half: emitted character value fields are even and padded with the
correct byte), DCM-009 (writer rejects invalid UIDs through `ParseUID`, not a local check).

**Fixtures needed (Increment 0):** `liver.dcm` (Explicit VR LE round-trip), `SC_rgb_expb.dcm` (Big Endian),
`MR2_UNCI.dcm` (Implicit VR LE), plus a generated `liver.truncated.dcm` for DCM-003.

**Verification gate:** round-trip parse-encode-parse byte-stable for canonical Explicit VR LE input; `dciodvfy` passes
on written files (Increment 0 gate); pydicom reads every written file; truncated and hostile-length fixtures produce
typed errors, never a partial success.

---

## Increment 3 — Sequences (`SQ`/`Item`) — outline

**Scope:** First-class `SQ` parsing and writing: defined-length and undefined-length items terminated by the Item
Delimitation Item `(FFFE,E00D)` and Sequence Delimitation Item `(FFFE,E0DD)`, nested datasets to arbitrary depth, with a
bounded maximum nesting depth (default 64) returning a typed `LimitExceededError`.

**Key types/functions:**

```go
type Sequence struct { /* ordered items */ }
type Item struct { DataSet *DataSet }
func NewSequence(items ...*DataSet) *Sequence
func (s *Sequence) Items() iter.Seq[Item]
func (s *Sequence) Append(ds *DataSet)
func (s *Sequence) Len() int
func NewSequenceValue(s *Sequence) Value
func (ds *DataSet) GetSequence(t Tag) (*Sequence, bool)
type LimitExceededError struct { Tag Tag; Limit, Actual uint64; Kind string }
```

**Dependencies:** Increment 1 (`DataSet`, `Value`), Increment 2 (the element read/write loop, the bounded reader, the
depth-cap `ReadOption`). `DataSet.Clone` (Task 1.14) is extended here to deep-copy sequence items.

**Tasks (to be expanded):** `Sequence`/`Item` types and constructors; defined-length item parse; undefined-length item
parse with delimiter detection; recursive parse with depth cap + `LimitExceededError`; sequence write (both encodings);
`Clone` deep-copies items; round-trip with `MR2_UNCI.dcm` nested sequences; depth-bomb regression.

**Codex defects guarded:** DCM-005 (sequences are real nested datasets, never dropped/blanked), DCM-003 and DCM-004 again
inside item parsing (truncated items fail; item lengths validated against bytes remaining; undefined length is never an
allocation size).

**Verification gate:** round-trip of a deeply nested fixture is byte-stable; a synthetic depth-65 sequence yields
`LimitExceededError` with `Kind == "sequence-depth"`; pydicom agrees on the parsed sequence structure.

---

## Increment 4 — Specific Character Set — outline

**Scope:** `SpecificCharacterSet` `(0008,0005)` decode/encode for the customisable text VRs (`SH`, `LO`, `ST`, `LT`,
`PN`, `UC`, `UT`): default repertoire (ISO_IR 6), ISO 2022 code extensions with G0/G1 switching, and the stand-alone
encodings UTF-8 (`ISO_IR 192`), GB18030, and GBK. The reader stores raw value-field bytes alongside the decoded string
and decodes lazily through the dataset's resolved character set.

**Key types/functions:**

```go
type SpecificCharacterSet struct { /* ordered defined terms + resolved decoders */ }
func NewSpecificCharacterSet(definedTerms ...string) (*SpecificCharacterSet, error)
func (c *SpecificCharacterSet) Decode(b []byte) (string, error)
func (c *SpecificCharacterSet) Encode(s string) ([]byte, error)
func WithDefaultCharacterSet(cs ...string) ReadOption
```

**Dependencies:** Increment 1 (`Strings` values, `PersonName`), Increment 2 (the reader resolves `(0008,0005)` before
decoding text). `golang.org/x/text/encoding` provides the GB18030/GBK and ISO 2022 codecs.

**Tasks (to be expanded):** defined-term-to-decoder table (mirrors pydicom `charset.py`); default repertoire decode;
UTF-8 decode; GB18030/GBK decode; ISO 2022 G0/G1 switching for multi-valued sets; unknown defined term -> typed error
(no silent mojibake); encode path for write; integration into `GetString`/`GetStrings` and `PersonName` decoding.

**Codex defects guarded:** DCM-011 (the prototype ignored `(0008,0005)` and treated bytes as Go strings, corrupting
non-ASCII text).

**Verification gate:** a fixture with `ISO 2022` Japanese `PersonName` decodes to the correct ideographic/phonetic
components and re-encodes byte-stable; an unknown defined term returns a typed error.

---

## Increment 5 — Datetime VRs (`DA`/`TM`/`DT`) — outline

**Scope:** The three datetime VRs, each preserving its source lexical form (byte-stable round-trip) and resolving to a Go
`time.Time` where representable. Strict parsing by default; partial-date acceptance opt-in.

**Key types/functions:**

```go
type DA struct { /* preserves source */ }
type TM struct { /* preserves source; variable precision */ }
type DT struct { /* preserves source; offset + fractional seconds */ }
func ParseDA(s string, opts ...DateOption) (DA, error)
func ParseTM(s string) (TM, error)
func ParseDT(s string) (DT, error)
func (d DA) Time() (time.Time, bool)
func (t DT) Time() (time.Time, bool) // carries the parsed UTC offset
func WithLenientDates() ReadOption
```

**Dependencies:** Increment 1 (`Strings` values carry the lexical form). Largely a PORT-WITH-FIXES of the prototype's
`dicom/datetime` package, adopting the committed type names.

**Tasks (to be expanded):** strict `DA` requiring 8 digits; `DateOption` lenient mode accepting legacy `YYYY`/`YYYYMM`;
`TM` variable precision + leap-second `60` accepted and normalised to `59` on `Time()` while the string keeps `60`; `DT`
fractional seconds (1-6 digits) + `&ZZXX` offset, zone-aware `Time()`; preserve fractional precision (no zero-fill).

**Codex defects guarded:** DCM-010 (prototype accepted `YYYY`/`YYYYMM` unconditionally and rejected leap-second `60`;
strict default + opt-in lenient + leap-second acceptance correct all three).

**Verification gate:** table-driven tests for the PS3.5 §6.2 forms; `YYYY` rejected in strict mode, accepted in lenient;
`235960` parses and `Time()` normalises to `59`; the preserved string survives a round-trip unchanged.

---

## Increment 6 — Pixel data pipeline — outline

**Scope:** `(7FE0,0010)` decode and encode. Native (contiguous `OB`/`OW`) and RLE Lossless are pure Go, always available,
read and write. Encapsulated fragmented frames parse as a bounded stream with a Basic Offset Table. JPEG-family decode is
behind a `//go:build cgo && radx_codecs` tag; encode/transcode exists only where a codec does (RLE pure-Go first, JPEG
2000 lossless behind the tag) and is off by default. With CGo disabled, JPEG-family frame access returns a typed
`ErrCodecUnavailable` naming the transfer syntax.

**Key types/functions:**

```go
type PixelData struct {
	Rows, Columns, SamplesPerPixel, BitsAllocated, BitsStored, PixelRepresentation uint16
	NumberOfFrames int
	TransferSyntax TransferSyntax
}
func (p *PixelData) Frames() iter.Seq2[Frame, error]
func (p *PixelData) BasicOffsetTable() (BasicOffsetTable, bool)
type Frame struct { Index int; Pixels []byte }
type Codec interface {
	TransferSyntax() TransferSyntax
	Decode(frame []byte, geom PixelGeometry) ([]byte, error)
	Encode(frame []byte, geom PixelGeometry) ([]byte, error)
	CanEncode() bool
}
func RegisterCodec(c Codec)
var ErrCodecUnavailable = errors.New("dicom: codec unavailable for transfer syntax")
var ErrEncodeUnsupported = errors.New("dicom: encode unsupported for transfer syntax")
```

**Dependencies:** Increment 2 (transfer syntaxes, the bounded reader), Increment 3 (the encapsulated fragment items reuse
the bounded item parser). This is a REWRITE of the prototype's pixel package.

**Tasks (to be expanded):** `PixelData` geometry resolution from the image-pixel module; native frame slicing (byte-order
aware); RLE Lossless decode (PS3.5 Annex G, bounds-checked segments); RLE Lossless encode; encapsulated fragment
bounded-stream parse + Basic Offset Table (pydicom `parse_basic_offsets`); `Codec` registry; `ErrCodecUnavailable` on
nocgo JPEG access; CGo JPEG2000 decode/encode hardened (checked C allocations, dimension caps, checked `size_t`/`int`,
no `C.int(size)` truncation); subprocess fuzz/crash harness with timeouts; transcode opt-in only.

**Codex defects guarded:** DCM-006 (bounded fragment stream, truncated trailing header is an error not a `break`, even
item lengths validated; no single unbounded blob), DCM-014 (CGo hardening: checked allocations, dimension caps, checked
size conversions, no output truncation), DCM-015 (skipped JPEG2000 crash/fuzz tests moved into subprocesses with
timeouts; a hang fails CI, never skipped).

**Fixtures needed:** `liver_rle.dcm` (RLE pure Go), `liver_j2k.dcm` (JPEG2000: CGo decode + nocgo `ErrCodecUnavailable`),
plus malformed/truncated encapsulated fixtures generated for the fragment and fuzz regressions.

**Verification gate:** RLE round-trip pixel-exact in pure Go; `go test ./dicom/...` (CGo disabled) passes and JPEG2000
frame access returns `ErrCodecUnavailable`; `go test -tags 'radx_codecs' ./dicom/...` (CGo enabled) decodes JPEG2000;
the fuzz/crash subprocess suite fails CI on hang.

---

## Increment 7 — PS3.15 Basic Profile de-identification — outline

**Scope:** The PS3.15 Annex E Basic Application Level Confidentiality Profile. Apply the Table E.1-1 action set
(X remove, D dummy, Z zero-length, C clean, U remap-UID, K keep) to every matched attribute at every nesting level,
recursing through `SQ` items. UID remapping is consistent within a run (same source UID -> same replacement) via the
`UIDGenerator`. Dates and times are removed/zeroed by default; retaining them is opt-in and applies a single consistent
per-study shift or keeps them verbatim. Set `PatientIdentityRemoved (0012,0062) = YES` and `DeidentificationMethod
(0012,0063)`. Fail closed on detected burned-in pixel PHI.

**Key types/functions:**

```go
type Profile struct { /* action table, options, UID remap, dummy policy */ }
type ProfileOption func(*Profile)

// DateMode selects how the Retain Longitudinal Temporal Information sub-option keeps
// dates/times when opted in. It is defined here (used only by the de-id profile).
type DateMode uint8
const (
	DateModeKeep  DateMode = iota // keep dates/times verbatim
	DateModeShift                 // shift by one consistent per-study offset
)

func NewProfile(g *UIDGenerator, opts ...ProfileOption) *Profile
func (p *Profile) Deidentify(ds *DataSet) (*DataSet, error)
func WithRetainPatientCharacteristics() ProfileOption
func WithRetainLongitudinalTemporalInformation(mode DateMode) ProfileOption
func WithRetainDeviceIdentity() ProfileOption
func WithRetainUIDs() ProfileOption
func WithDummyValues(replacements map[Tag]string) ProfileOption
var ErrBurnedInPixelData = errors.New("dicom: burned-in pixel PHI not cleaned")
```

**Dependencies:** Increment 1 (`DataSet.Clone`, `UIDGenerator`, `Tag` dictionary), Increment 3 (recursion through `SQ`
items), Increment 5 (date/time attributes for the removal/shift policy). This is a REWRITE of the prototype's
`dicom/anonymize` package, which the audit flagged as a sparse, top-level-only table falsely claiming PS3.15 compliance.

**Tasks (to be expanded):** the full Table E.1-1 action table (generated or hand-curated from PS3.15 Annex E, not the
sparse prototype subset); recursive traversal applying the action at every level including `SQ` items; consistent UID
remap table seeded by the `UIDGenerator`; date/time removal by default + `WithRetainLongitudinalTemporalInformation`
shift/keep; private-tag removal by default + `WithRetainSafePrivate` allow-list; de-identification metadata
(`(0012,0062)`, `(0012,0063)`); burned-in detection reading `BurnedInAnnotation (0028,0301)` and fail-closed
`ErrBurnedInPixelData`; never-mutate-input invariant; the PS3.15 attribute-checklist feature test.

**Codex defects guarded:** DCM-013 (recursive into sequences so PHI inside items is acted on; no false PS3.15-compliance
claim; burned-in pixel PHI fail-closed rather than silently "cleaned").

**Verification gate:** a fixture with PHI nested in a sequence item is de-identified at every level (no residual
Table E.1-1 attribute survives); UID references stay internally consistent after remap; `(0012,0062) == "YES"`; a
fixture with `BurnedInAnnotation == "YES"` returns `ErrBurnedInPixelData` unless explicitly overridden; the source
dataset is never mutated (assert by comparing a pre-clone snapshot).

---

## Increment 8 — Structured Report content-item model — outline

**Scope:** The DICOM Structured Report (SR) content-item tree, a committed v1 deliverable (reference doc; conformance
declares Basic Text SR, Enhanced SR, and Comprehensive SR as IOD-aware supported SOP classes, not opaque transport). An
SR encodes its content as a tree rooted at the Content Sequence `(0040,A730)`; each node carries a concept name, a value
typed by its `ValueType (0040,A040)`, and a relationship to its parent `(0040,A010)`. go-radx models this tree as
first-class data so the `convert` SR-to-FHIR leg has a documented data layer to read and build (PRD §5.1 step 6).
`ParseSR` reads the tree from a parsed `DataSet`; `BuildSR` encodes a tree back into one.

**Key types/functions (from `docs/reference/dicom.md`):**

```go
// ValueType is the SR content-item value type from PS3.3 (the VR CS value of (0040,A040)).
type ValueType uint8
const (
	ValueTypeContainer ValueType = iota // CONTAINER
	ValueTypeText                       // TEXT
	ValueTypeCode                       // CODE
	ValueTypeNum                        // NUM (measured value + units)
	ValueTypePName                      // PNAME (person name)
	ValueTypeDate                       // DATE
	ValueTypeTime                       // TIME
	ValueTypeDateTime                   // DATETIME
	ValueTypeUIDRef                     // UIDREF
	ValueTypeComposite                  // COMPOSITE (referenced SOP instance)
	ValueTypeImage                      // IMAGE (referenced image)
	ValueTypeSCoord                     // SCOORD (spatial coordinates)
	ValueTypeSCoord3D                   // SCOORD3D
	ValueTypeTCoord                     // TCOORD (temporal coordinates)
	ValueTypeWaveform                   // WAVEFORM
)

// RelationshipType is the SR parent-child relationship from PS3.3 (the VR CS value of (0040,A010)).
type RelationshipType uint8
const (
	RelationshipContains          RelationshipType = iota // CONTAINS
	RelationshipHasObsContext                             // HAS OBS CONTEXT
	RelationshipHasConceptMod                             // HAS CONCEPT MOD
	RelationshipHasProperties                             // HAS PROPERTIES
	RelationshipHasAcqContext                             // HAS ACQ CONTEXT
	RelationshipInferredFrom                              // INFERRED FROM
	RelationshipSelectedFrom                              // SELECTED FROM
)

// ConceptNameCode is a coded concept (a Code Sequence item): the (code, scheme, meaning) triple.
type ConceptNameCode struct {
	CodeValue              string // (0008,0100)
	CodingSchemeDesignator string // (0008,0102), e.g. "DCM", "SCT", "LN"
	CodeMeaning            string // (0008,0104)
}

// ContentItem is one node of the SR content tree.
type ContentItem struct {
	ValueType    ValueType
	Relationship RelationshipType // relationship to the parent; root carries the zero value
	ConceptName  ConceptNameCode  // (0040,A043) what this item measures or states

	Text             string                  // TEXT
	Code             ConceptNameCode         // CODE: the coded value (0040,A168)
	MeasuredValue    Decimal                 // NUM: numeric value (0040,A30A)
	MeasurementUnits ConceptNameCode         // NUM: units (0040,08EA)
	PersonName       PersonName              // PNAME
	DateTime         DT                      // DATE/TIME/DATETIME, preserved lexical form
	UID              UID                     // UIDREF
	Referenced       []ReferencedSOPInstance // COMPOSITE/IMAGE referenced instances

	Children []ContentItem // nested content (CONTAINS and other relationships)
}

// ReferencedSOPInstance pairs a referenced SOP Class with its SOP Instance. It is the single shape
// reused by dimse and dicomweb; SR COMPOSITE/IMAGE items reference instances through it.
type ReferencedSOPInstance struct {
	SOPClassUID    SOPClassUID
	SOPInstanceUID SOPInstanceUID
}

func ParseSR(ds *DataSet) (*ContentItem, error)
func BuildSR(ds *DataSet, root *ContentItem) error
```

**Dependencies:** Increment 1 (`Decimal`, `PersonName`, `UID`, `SOPClassUID`/`SOPInstanceUID`, `DataSet`), Increment 3
(`Sequence`/`Item` — the content tree is a recursion over `SQ`, and `ReferencedSOPInstance` is read from referenced-SOP
sequences), and Increment 5 (`DT` for DATE/TIME/DATETIME items). `ReferencedSOPInstance` is a new shared type added in
this increment, reused later by `dimse` and `dicomweb`. This is new code, not a port (the prototype had no SR model).

**Tasks (to be expanded):** the `ValueType` and `RelationshipType` enums with their CS-string mappings (parse/format the
`(0040,A040)`/`(0040,A010)` defined terms); `ConceptNameCode` read/write against a Code Sequence item (`(0008,0100)`/
`(0008,0102)`/`(0008,0104)`); `ContentItem` and `ReferencedSOPInstance` types; `ParseSR` recursing the Content Sequence
`(0040,A730)`, rejecting a non-SR IOD (SOP Class not a supported SR class) with a typed error; per-`ValueType` value
extraction (TEXT/CODE/NUM/PNAME/DATETIME/UIDREF/COMPOSITE/IMAGE); `BuildSR` encoding the tree back into the Content
Sequence with `ValueType`, `RelationshipType`, `ConceptNameCodeSequence`, and value attributes per node; the
depth/size-cap reuse and malformed-tree regression.

**Depth and size cap:** the SR tree reuses the same sequence-depth cap as any other `SQ` nesting (default 64, from
Increment 3's `WithMaxSequenceDepth`), so a maliciously deep SR returns a typed `LimitExceededError` rather than
overflowing the stack — `ParseSR` recurses through `SQ` and inherits that bound.

**Codex defects guarded:** none specific to SR in the audit (the prototype shipped no SR model). The increment inherits
DCM-003/DCM-004 truncation and length bounds through the Increment 3 sequence parser it recurses over.

**Fixtures needed (Increment 0):** add a small Basic Text SR and a Comprehensive SR fixture (pydicom ships SR test files,
e.g. `reportsi.dcm`-style) to `testdata/dicom/` so `ParseSR` round-trips a real content tree.

**Verification gate:** `ParseSR` of an SR fixture yields the expected root `CONTAINER` with the correct nested
`ValueType`/`RelationshipType` and `ConceptNameCode` values; `BuildSR` of that tree re-encodes to a `DataSet` that
`ParseSR` reads back equal (tree round-trip); a non-SR dataset returns a typed error; a synthetic depth-65 SR yields
`LimitExceededError`; pydicom agrees on the parsed `(0040,A730)` structure.

---

## Cross-increment Codex defect traceability

Every audited defect is closed by a named regression test in the increment shown.

| Defect | Severity | Closed in | Guarded by |
|--------|----------|-----------|------------|
| DCM-001 | High | Increment 2 | FileMeta group-length-written-first, exact byte count |
| DCM-002 | High | Increment 2 | transfer-syntax-faithful encoder (byte order + deflate) |
| DCM-003 | High | Increment 2 (+3, +6) | truncation-is-failure regression at every level |
| DCM-004 | Critical | Increment 2 | hostile-length rejected before `make` in bounded reader |
| DCM-005 | High | Increment 3 | sequences are real nested datasets, round-trip |
| DCM-006 | High | Increment 6 | bounded fragment stream, truncated header is an error |
| DCM-007 | Medium | Increment 1 (length) + Increment 2 (emitted padded bytes) | VR-centralised even-length padding (1.2/1.10/1.11) + byte-level pad assertion in the writer round-trip |
| DCM-008 | High | Increment 1 (1.8) | fail-closed UIDGenerator, no default root |
| DCM-009 | Medium | Increment 1 (1.3) + 2 | single `ParseUID` validation path, writer reuses it |
| DCM-010 | Medium | Increment 5 | strict DA + lenient opt-in + leap-second |
| DCM-011 | Medium | Increment 1 (1.7) + 4 | PersonName components + SpecificCharacterSet |
| DCM-012 | Medium | Increment 1 (1.5) | repeating-group dictionary mask |
| DCM-013 | Critical | Increment 7 | recursive Table E.1-1 + burned-in fail-closed |
| DCM-014 | High | Increment 6 | CGo hardening (checked alloc/size, dimension caps) |
| DCM-015 | High | Increment 6 | subprocess fuzz/crash tests, hang fails CI |
| DCM-016 | Medium | Increment 1 (1.11, 1.12, 1.14) | deep Clone + copy-on-construct/access + documented concurrency |

---

## Open questions and spec gaps

These surfaced while writing the plan and should be resolved before or during execution. None blocks Increment 1.

1. **External validators are not installed locally.** Neither `dciodvfy` (dcmtk), `dcmdump`, nor `pydicom` is present on
   this machine. Increment 0 wires the gates to skip cleanly when absent (and to fail in CI where present). Confirm the
   CI image installs dcmtk and pydicom, and pin their versions, so the conformance gate is real in CI rather than always
   skipped. This is the one verification step I could not run end to end while planning.

2. **Tag representation change from the prototype is intentional but worth a sign-off.** The committed API is
   `type Tag uint32`; the prototype used `struct{Group, Element uint16}`. The plan ports the prototype's dictionary data
   and parsing logic but adopts the scalar `uint32`. This changes the public type and any code that referenced the
   struct fields. This is acceptable for a greenfield rewrite, but flagged so it is a conscious decision, not a drift.

3. **Innolitics dictionary provenance and licence in the canonical repo.** The prototype generated its ~5,189-entry
   dictionary from the innolitics dataset (MIT, pinned at rev `7f4749d09ed3ef2fa70637d376d423a4b13523cd`, rev2024b). The
   plan reuses that generator and pins the same revision. Confirm the innolitics MIT attribution is acceptable for the
   canonical repo's licence posture (the LICENSE is MIT, so this is almost certainly fine, but the attribution header
   must be preserved verbatim in the generated file).

4. **`EncodedLen(bo binary.ByteOrder)` for the empty/zero-length case.** Task 1.13 (`SetEmpty`) calls
   `EncodedLen(nil)`. The reference doc says `EncodedLen` "never panics", so the implementation must tolerate a nil byte
   order for the zero-length case. Confirmed against the doc; noting it because it is an easy panic to introduce.

5. **`PixelGeometry` and `BasicOffsetTable` exact shapes.** The reference doc names `PixelGeometry` (Codec parameter) and
   `BasicOffsetTable` (return of `PixelData.BasicOffsetTable`) but does not give their full field lists. Increment 6 will
   need to define them; the plan should commit their shapes when Increment 6 is expanded, deriving fields from the
   image-pixel module attributes the doc lists on `PixelData`.

6. **`WithRetainSafePrivate` option.** The de-identification scope section names `WithRetainSafePrivate` but the
   `ProfileOption` list does not include its signature. Increment 7 expansion should reconcile this — either it is an
   additional `ProfileOption` to add, or the prose is aspirational. Surface to the API owner.
