# DICOM data model and Part 10

The `dicom` package is the foundation of go-radx. Every other subsystem reads it: `dimse` transports datasets over the
network, `dicomweb` serializes them to HTTP, and `convert` maps them to FHIR. This reference defines the public API and
behaviour of the data model (datasets, elements, tags, value representations, sequences), Part 10 file input and output,
the supported transfer syntaxes, the pixel-data pipeline, the DICOM Structured Report content-item model, and the
PS3.15 Basic Application Level Confidentiality Profile for de-identification.

The package is the M1 deliverable in the implementation sequence and a ground-up rewrite of the prototype's DICOM core.
The prototype was audited as prototype-level (`docs/prd/go-radx-prd.md` §2.2): it dropped `SQ` sequences, never wrote
the file-meta group length, mis-declared transfer syntax on write, accepted truncated files as complete, and fed
attacker-controlled lengths straight into `make([]byte, n)`. The behaviour specified here closes each of those defects,
and the worked examples and conformance section state exactly what is in scope for v1.

This document is the contract the implementation conforms to. Where it must commit an API detail the PRD left open, it
makes a sensible choice and records it; those choices are collected at the end.

## Scope

In scope for v1, seeded from the `pydicom` parity floor (`docs/prd/go-radx-prd.md` §6.2):

- The data model: `DataSet`, `Element`, `Tag`, `VR`, the value types, `Sequence` (`SQ`) and `Item`.
- `UID` parsing, validation, and generation under a configured organisation root, and the named SOP-identifier types
  `SOPClassUID` and `SOPInstanceUID` reused by `dimse`, `dicomweb`, and `convert`.
- The datetime value representations `DA`, `TM`, `DT`, with timezone offset and fractional-second handling.
- The lexical-preserving `Decimal` type backing `DS` and `IS`, shared with FHIR `decimal`.
- `PersonName` with three `=`-delimited component groups, each five `^`-delimited components.
- `SpecificCharacterSet` decoding, including ISO 2022 code extensions and UTF-8 / GB18030 / GBK.
- Part 10 read and write: the 128-byte preamble, the `DICM` prefix, `FileMeta` (Explicit VR Little Endian) with an
  auto-recomputed group length, and the main dataset.
- The four uncompressed transfer syntaxes, read and write: Implicit VR LE, Explicit VR LE, Explicit VR BE (retired),
  and Deflated Explicit VR LE.
- The pixel pipeline: native (contiguous `OB`/`OW`), RLE, and encapsulated fragmented frames; decoding for every
  supported compressed transfer syntax and transcoding/encoding where a codec exists, behind the optional-CGo build
  tag (RLE and JPEG 2000 lossless first), with transcoding off by default.
- The DICOM Structured Report content-item model: `ContentItem`, `ConceptNameCode`, the SR value-type and
  relationship-type vocabularies, and SR document read and build. This is the data layer the `convert` SR-to-FHIR leg
  reads (PRD §5.1 step 6); the supported SR SOP classes are declared in `docs/conformance/dicom.md`.
- The PS3.15 Basic Application Level Confidentiality Profile for de-identification, with documented scope and limits.

DICOM-JSON is part of the broader `pydicom` floor but is documented in its own reference. DICOMDIR file-sets ship in
the `dicom` package (`FileSet`, `OpenFileSet`, `FileSetBuilder`) with their conformance scope declared in
`docs/conformance/dicom.md`; the API detail lives in the package godoc. This file covers the binary data model, the
Structured Report content-item model, and Part 10.

## The data model

### Tag

A `Tag` is the 32-bit `(group, element)` identifier of a data element, written `(gggg,eeee)`. It is a named type, never
a bare `uint32` (PRD §8.2, glossary naming rule 1).

```go
type Tag uint32

func NewTag(group, element uint16) Tag
func (t Tag) Group() uint16
func (t Tag) Element() uint16
func (t Tag) String() string // "(0010,0010)"

// IsPrivate reports whether the group is odd (private data).
func (t Tag) IsPrivate() bool

// IsPrivateCreator reports a private-creator tag: odd group, element in 0x0010..0x00FF.
func (t Tag) IsPrivateCreator() bool

// IsGroupLength reports element == 0x0000.
func (t Tag) IsGroupLength() bool
```

Dictionary lookup resolves a tag to its keyword, VR, and value multiplicity. It must resolve repeating groups by mask,
not by exact match. The prototype looked up tags exact-only and replaced the repeating `x` with `0`, so the overlay tag
`(6002,3000)` resolved as unknown and silently degraded to `UN` under Implicit VR (Codex DCM-012). `Lookup` masks the
variable nibbles of `60xx`/`50xx` (and `5xxx` curve groups) before falling back to unknown.

```go
type TagInfo struct {
    Keyword string // e.g. "PatientName"
    VR      VR     // dictionary VR; ambiguous VRs carry both, see VR
    VM      string // value multiplicity, e.g. "1", "1-n", "2"
    Name    string // human-readable name from PS3.6
}

// Lookup resolves a tag through the standard dictionary (~5,189 entries), resolving 60xx/50xx
// repeating groups by mask. ok is false for genuinely unknown tags.
func Lookup(t Tag) (info TagInfo, ok bool)

// LookupKeywordTag resolves a known keyword to its canonical tag for the common case where the
// keyword is a compile-time literal, e.g. LookupKeywordTag("PatientName") == (0010,0010). It
// panics only on a keyword that is not in the standard dictionary — a programmer error in the
// literal, never reachable from external input. Use LookupKeyword for runtime/dynamic keywords.
func LookupKeywordTag(keyword string) Tag

// LookupKeyword resolves a keyword to its canonical tag for dynamic input, returning ok == false
// for an unknown keyword instead of panicking.
func LookupKeyword(keyword string) (Tag, bool)
```

For the common attributes, the dictionary generator also emits one exported `Tag` constant per keyword, so callers can
reference a tag by name without a lookup call. These constants are generator output, not hand-written, and stay in step
with the dictionary:

```go
const (
    TagPatientID         Tag = 0x00100020
    TagPatientName       Tag = 0x00100010
    TagStudyInstanceUID  Tag = 0x0020000D
    TagStudyDate         Tag = 0x00080020
    TagStudyDescription  Tag = 0x00081030
    TagModalitiesInStudy Tag = 0x00080061
    TagSeriesInstanceUID Tag = 0x0020000E
    TagSOPClassUID       Tag = 0x00080016
    TagSOPInstanceUID    Tag = 0x00080018
    // ... one const per dictionary keyword, generated from PS3.6.
)
```

These constants and `LookupKeywordTag` are the idiom every reference doc uses (`dimse`, `dicomweb`, `convert`, `cli`):
a literal keyword resolves to a `Tag` through the generated constant or through `LookupKeywordTag`; only genuinely
dynamic keyword input goes through `LookupKeyword`.

Diagnostics render a tag as its keyword plus `(gggg,eeee)`, never bare hex (PRD §8.2, glossary rule 4). The error and
log helpers in the package format tags through `Lookup` so a reader sees `PatientName (0010,0010)`, not `0x00100010`.

### VR

A `VR` is the two-letter Value Representation from PS3.5 Table 6.2-1. The floor is the 34 standard VRs plus the 4
ambiguous placeholders (PRD §6.2, glossary entry). The ambiguous VRs (`US or SS`, `OB or OW`, `US or OW`,
`US or SS or OW`) are parse-time placeholders the reader resolves from context; they never appear on the wire.

```go
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
    VROD           // Other Double (64-bit)
    VROF           // Other Float
    VROL           // Other Long
    VROV           // Other Very Long (64-bit)
    VROW           // Other Word
    VRPN           // Person Name
    VRSH           // Short String
    VRSL           // Signed Long
    VRSQ           // Sequence of Items
    VRSS           // Signed Short
    VRST           // Short Text
    VRSV           // Signed Very Long (64-bit)
    VRTM           // Time
    VRUC           // Unlimited Characters
    VRUI           // Unique Identifier (UID)
    VRUL           // Unsigned Long
    VRUN           // Unknown
    VRUR           // URI/URL
    VRUS           // Unsigned Short
    VRUT           // Unlimited Text
    VRUV           // Unsigned Very Long (64-bit)
)

func (vr VR) String() string // "PN", "SQ", ...

// Is32BitLength reports whether the VR uses the 4-byte explicit-VR length form
// (OB OW OD OF OL OV SQ UC UR UT UN); the rest use the 2-byte form.
func (vr VR) Is32BitLength() bool

// PadByte returns the value-field pad byte: NULL (0x00) for UI, otherwise SPACE (0x20).
// Binary VRs are not padded (their natural length is even).
func (vr VR) PadByte() (byte, bool)
```

Padding is centralised on the VR, not duplicated per value type. The whole character value field must be even length
(PS3.5 §6.2); `UI` pads with `NULL`, every other string VR pads with `SPACE`. The prototype padded only `UI`, so
odd-length `AE`/`CS`/`DA`/`DS`/`IS`/`LO`/`PN` values were written with odd value lengths (Codex DCM-007). The writer
computes value length after padding, applying the pad byte to the last value of a multi-valued element.

### DataSet and Element

A `DataSet` is an ordered, `Tag`-keyed collection of `Element` values (note the capital `S`, per PRD §8.1 and the
glossary). An `Element` is the atomic `(Tag, VR, length, Value)` unit.

```go
type DataSet struct {
    // internal: ordered tag map; not exported
}

type Element struct {
    Tag   Tag
    VR    VR
    Value Value
}
```

`DataSet` preserves insertion/ascending-tag order so a round-trip is byte-stable for canonical input. Accessors are
typed, never positional:

```go
func NewDataSet() *DataSet

// Get returns the element at t. ok is false if absent.
func (ds *DataSet) Get(t Tag) (Element, bool)

// Set inserts or replaces the element for its tag.
func (ds *DataSet) Set(e Element)

// SetString is a convenience mutator: it looks up t's dictionary VR and inserts (or replaces)
// a text element carrying vals. It is shorthand for Set(Element{Tag: t, VR: <dict VR>, ...}).
func (ds *DataSet) SetString(t Tag, vals ...string)

// SetEmpty inserts (or replaces) a zero-length element at t under its dictionary VR. It is the
// idiom for declaring a return-key in a query identifier dataset (the C-FIND "universal match").
func (ds *DataSet) SetEmpty(t Tag)

// Delete removes the element at t; it is not an error if absent.
func (ds *DataSet) Delete(t Tag)

// Len returns the number of elements at this level (sequence items are not counted).
func (ds *DataSet) Len() int

// All iterates elements in ascending tag order (Go 1.23+ iterator).
func (ds *DataSet) All() iter.Seq[Element]

// Clone returns a deep copy: sequences, items, and value slices are copied, not aliased.
func (ds *DataSet) Clone() *DataSet
```

`DataSet` is **not** safe for concurrent mutation. A `*DataSet` may be read concurrently only if no goroutine mutates
it; callers that share a dataset across goroutines must synchronise externally or pass independent `Clone()` copies.
This is documented rather than locked because the hot path is single-threaded parse and write, and a mutex on every
access would cost more than it buys. `Clone` performs a genuine deep copy so de-identification and conversion never
alias the source dataset's value slices — the prototype's "copy" reused the same value objects, allowing cross-dataset
contamination (Codex DCM-016).

Typed convenience accessors return the model types rather than raw bytes, so callers do not parse value fields by hand:

```go
func (ds *DataSet) GetString(t Tag) (string, bool)        // first value of a text VR, charset-decoded
func (ds *DataSet) GetStrings(t Tag) ([]string, bool)     // all backslash-separated values
func (ds *DataSet) GetUID(t Tag) (UID, bool)
func (ds *DataSet) GetInt(t Tag) (int64, bool)            // IS/SS/US/SL/UL/SV/UV
func (ds *DataSet) GetDecimal(t Tag) (Decimal, bool)      // DS
func (ds *DataSet) GetSequence(t Tag) (*Sequence, bool)   // SQ
func (ds *DataSet) GetPersonName(t Tag) (PersonName, bool)
```

### Value

`Value` is the interface every element value implements. It exposes the on-wire VR and the encoded length, and never
exposes a mutable internal slice without copying.

```go
type Value interface {
    VR() VR
    // EncodedLen returns the even value-field length for the given byte order
    // and VR encoding (after padding). It never panics.
    EncodedLen(bo binary.ByteOrder) uint32
}
```

The concrete value types map each VR class to a Go-native representation:

| Value type | VRs | Go representation |
|------------|-----|-------------------|
| `Strings` | AE AS CS LO SH UC UR UT ST LT DA TM DT PN UI | `[]string`, charset-aware |
| `Ints` | SS US SL UL SV UV | `[]int64` |
| `Floats` | FL FD OF OD | `[]float64` |
| `Decimals` | DS IS | `[]Decimal` (lexical-preserving) |
| `Tags` | AT | `[]Tag` |
| `Bytes` | OB OW OL OV UN | `[]byte` (length-bounded) |
| `Sequence` | SQ | `*Sequence` |
| `PixelData` | the `(7FE0,0010)` element | native or encapsulated, see pixel pipeline |

Construct values through helpers that validate VR and length; the zero element is invalid by construction so an
uninitialised value cannot be written:

```go
func NewStrings(vr VR, vals ...string) Value
func NewInts(vr VR, vals ...int64) Value
func NewFloats(vr VR, vals ...float64) Value
func NewDecimals(vr VR, vals ...Decimal) Value
func NewBytes(vr VR, b []byte) Value   // copies b; the value owns its bytes
func NewSequenceValue(s *Sequence) Value
```

`NewBytes` copies its input so the value owns its bytes and a later mutation of the caller's slice cannot reach into a
stored element. Read accessors that hand back slices return copies for the same reason.

### Sequence and Item

VR `SQ` is an ordered list of `Item` values, each a nested `DataSet`. Sequences nest arbitrarily and may be encoded with
a defined length or an undefined length terminated by a Sequence Delimitation Item. The item tag is `(FFFE,E000)`, the
item delimiter `(FFFE,E00D)`, and the sequence delimiter `(FFFE,E0DD)`.

```go
type Sequence struct {
    // internal: ordered slice of items
}

type Item struct {
    DataSet *DataSet
}

func NewSequence(items ...*DataSet) *Sequence
func (s *Sequence) Items() iter.Seq[Item]
func (s *Sequence) Append(ds *DataSet)
func (s *Sequence) Len() int
```

Sequences are first-class, not dropped. The prototype replaced `SQ` values with empty bytes and never represented nested
datasets (Codex DCM-005), which lost clinically significant data and, worse, hid PHI from de-identification. The reader
parses defined and undefined-length items recursively, honouring both encodings, and the writer round-trips them without
loss. Recursion is bounded: a configurable maximum nesting depth (default 64) guards against a maliciously deep
sequence, returning a typed error rather than overflowing the stack.

### UID

A `UID` is a dotted-numeric ISO OID of at most 64 characters (VR `UI`). It identifies SOP Classes and Instances,
Studies, Series, and transfer syntaxes. Validation follows PS3.5 §9.1: components are non-empty numeric runs separated
by single dots, with no leading zero in a multi-digit component (a single `0` is allowed) and no trailing dot.

```go
type UID string

// Parse validates s per PS3.5 sec 9.1 and returns the trimmed UID.
// It rejects empty components ("1..2"), leading-zero components ("1.02"),
// trailing dots, non-numeric characters, and lengths over 64 characters.
func ParseUID(s string) (UID, error)

func (u UID) Validate() error
func (u UID) IsValid() bool
func (u UID) String() string

// Name returns the registered name for a known UID (~490 entries: SOP classes,
// transfer syntaxes, well-known instances) or the UID itself if unregistered.
func (u UID) Name() string
```

The single validation path is `ParseUID`/`Validate`. The prototype's writer used a weaker local check that accepted
consecutive dots and leading zeros, diverging from the package validator (Codex DCM-009); here the writer rejects an
invalid UID through the same path before emitting any bytes.

UID generation mints a new UID under a configured organisation root. This is a deliberate fail-closed design: the
prototype generated under PixelMed's registered root `1.2.826.0.1.3680043.10`, which mislabels go-radx output as another
organisation's (Codex DCM-008, PS3.5 §9.1/§9.2). go-radx ships **no** default registered root.

```go
type UIDGenerator struct {
    // internal: root, entropy source
}

// NewUIDGenerator returns a generator that mints UIDs under root. root must be a
// valid UID prefix of at most 54 characters (leaving room for a >= 9-digit suffix
// within the 64-character limit). It returns an error if root is empty or invalid.
func NewUIDGenerator(root UID) (*UIDGenerator, error)

// NewRandomUIDGenerator returns a generator using the ISO/IEC 9834-8 UUID-derived
// root "2.25.", which requires no organisational registration. Suffixes are the
// integer form of a random UUID. This is the recommended default when no
// organisation root is configured.
func NewRandomUIDGenerator() *UIDGenerator

// Generate returns a fresh UID under the generator's root, <= 64 characters,
// using a cryptographically random suffix.
func (g *UIDGenerator) Generate() UID
```

UID minting is modelled as a `UIDGenerator` value (`Generate`) rather than a bare `GenerateUID` function so the
organisation root and entropy source are explicit, injectable state rather than global configuration — the `Generate`
method is the canonical minting operation the glossary refers to, and `NewProfile` takes a `*UIDGenerator` so the
caller controls the root used for de-identification remapping.

The `54`-character prefix cap and the `2.25.` UUID fallback follow `pydicom`'s `generate_uid` (`uid.py`): a prefix is
appended with a random suffix sized to fill the 64-character field, and a `None` prefix produces a `2.25.`-rooted UID
from a UUID. Generated UIDs are length-bounded to 64 characters including any NULL padding the writer adds.

#### SOP identifier types

Two named types distinguish the SOP roles a `UID` plays, so a function signature states whether it expects a SOP Class
or a SOP Instance rather than taking a bare `UID`. They are the canonical types `dimse`, `dicomweb`, `convert`, and
`servers` reuse (PRD §8.2, glossary), and `FileMeta` carries them:

```go
type SOPClassUID UID
type SOPInstanceUID UID
```

Both inherit `UID`'s validation through conversion: `UID(sopClassUID).Validate()` and `UID(sopClassUID).Name()` work as
expected. Defining them in `dicom` — the M1 foundation every other subsystem reads — means `dimse`
`PresentationContext.AbstractSyntax`, `dicomweb` resource paths, and `ReferencedSOPInstance` all share one type rather
than re-declaring it.

### Decimal

`Decimal` is the lexical-preserving numeric type shared by FHIR `decimal` and DICOM `DS`/`IS` (PRD §8.1, glossary). It
carries the source string so a value read from a file serialises back byte-identically, beating both `fhir.resources`
(lossy on serialise) and the prototype (mapped to `float64`, also lossy). It performs no in-place arithmetic; conversion
to a Go numeric is explicit and may report inexactness.

```go
type Decimal struct {
    // internal: preserved source lexical form + parsed value
}

// ParseDecimal validates s as a DICOM DS/IS or FHIR decimal lexical form and
// preserves it verbatim. DS is limited to 16 bytes per value (PS3.5); IS to a
// signed integer in [-2^31, 2^31 - 1] expressed without a fractional part.
func ParseDecimal(s string) (Decimal, error)

func (d Decimal) String() string                // the preserved lexical form
// Float64 returns the value as a float64. ok is false only when the lexical form has no finite
// float64 representation (non-finite scale); a representable-but-rounded value returns ok == true.
// This matches fhir.md and the PRD §8.1 contract for the shared type.
func (d Decimal) Float64() (f float64, ok bool)
// Exact reports whether the float64 from Float64 represents d without rounding loss.
func (d Decimal) Exact() bool
// BigFloat returns a *big.Float with precision sufficient for the lexical form, for callers that
// need exactness or their own rounding.
func (d Decimal) BigFloat() *big.Float
func (d Decimal) Int64() (n int64, ok bool)     // ok is false if d is not integral
func (d Decimal) MarshalJSON() ([]byte, error)  // emits the preserved lexical form, unquoted for FHIR decimal
func (d *Decimal) UnmarshalJSON(b []byte) error
```

### PersonName

`PersonName` is VR `PN`: up to three `=`-delimited component groups (alphabetic, ideographic, phonetic), each holding up
to five `^`-delimited components (family, given, middle, prefix, suffix). The structure mirrors `pydicom`'s `PersonName`
(`valuerep.py`) and PS3.5: each component group is independently capped at 64 characters per component.

```go
type PersonName struct {
    Alphabetic  NameComponents
    Ideographic NameComponents // empty if absent
    Phonetic    NameComponents // empty if absent
}

type NameComponents struct {
    FamilyName string
    GivenName  string
    MiddleName string
    Prefix     string
    Suffix     string
}

// ParsePersonName splits s on "=" into up to three groups, each on "^" into up to
// five components, trimming the standard pad. It errors on more than three groups
// or more than five components in a group.
func ParsePersonName(s string) (PersonName, error)

// String renders the canonical "=" / "^" form, dropping trailing empty components
// and trailing empty groups (so "Doe^John" not "Doe^John^^^==").
func (p PersonName) String() string
```

The prototype had no component-group model and treated `PN` as a plain Go string (Codex DCM-011), which made
ideographic/phonetic names and per-component de-identification impossible.

### SpecificCharacterSet

`SpecificCharacterSet` is element `(0008,0005)`. It governs the decoding of the customisable text VRs (`SH`, `LO`, `ST`,
`LT`, `PN`, `UC`, `UT`) and supports the default repertoire, ISO 2022 code extensions, and the stand-alone encodings
UTF-8 (`ISO_IR 192`), GB18030, and GBK. The mapping from defined terms to Go decoders follows `pydicom`'s `charset.py`
table.

```go
type SpecificCharacterSet struct {
    // internal: ordered defined terms from (0008,0005), and the resolved decoders
}

// NewSpecificCharacterSet resolves the value-multiplicity-N defined terms of
// (0008,0005) into a decode/encode pipeline. An empty set means the default
// repertoire (ISO_IR 6 / ASCII). Unknown defined terms return a typed error
// rather than silently mojibake-ing.
func NewSpecificCharacterSet(definedTerms ...string) (*SpecificCharacterSet, error)

// Decode converts raw value-field bytes to a Go string, applying ISO 2022 G0/G1
// code-element switching for multi-valued character sets.
func (c *SpecificCharacterSet) Decode(b []byte) (string, error)

// Encode converts a Go string back to value-field bytes under the active set.
func (c *SpecificCharacterSet) Encode(s string) ([]byte, error)
```

The reader stores raw value-field bytes alongside the decoded string and decodes lazily through the dataset's resolved
character set. The prototype ignored `(0008,0005)` and treated bytes as Go strings (Codex DCM-011), corrupting any
non-ASCII text.

### DICOM date and time

The three datetime VRs are `DA` (`YYYYMMDD`), `TM` (`HHMMSS.FFFFFF`), and `DT`
(`YYYYMMDDHHMMSS.FFFFFF&ZZXX`). Each preserves its source lexical form (like `pydicom`'s `original_string`) so a
round-trip is byte-stable, and each parses to a Go `time.Time` where representable.

```go
type DA struct{ /* preserves source; resolves to a date */ }
type TM struct{ /* preserves source; variable precision */ }
type DT struct{ /* preserves source; offset + fractional seconds */ }

// ParseDA requires exactly 8 digits in strict mode (the default). In lenient mode
// it also accepts the legacy YYYY and YYYYMM partial forms.
func ParseDA(s string, opts ...DateOption) (DA, error)
func ParseTM(s string) (TM, error)
func ParseDT(s string) (DT, error)

func (d DA) String() string
func (d DA) Time() (time.Time, bool)
func (t DT) Time() (time.Time, bool) // carries the parsed UTC offset
```

Three behaviours follow PS3.5 §6.2 and correct prototype defects (Codex DCM-010):

- `DA` is `YYYYMMDD` and requires 8 digits in strict mode. The prototype accepted `YYYY` and `YYYYMM` unconditionally,
  silently treating partial dates as valid clinical metadata. Partial-date acceptance is opt-in through `DateOption`.
- `TM`/`DT` seconds range over `00`–`60`. The leap-second value `60` is valid in DICOM but not in Go's `time` package,
  so `ParseTM`/`ParseDT` accept it and `Time()` normalises `60` to `59` while the preserved string keeps `60`. The
  prototype rejected `60` outright.
- `DT` parses both the fractional-second component (1–6 digits) and the `&ZZXX` UTC offset; `Time()` returns a
  zone-aware value. `TM` and `DT` preserve their fractional precision exactly rather than zero-filling.

### Structured Report content items

A DICOM Structured Report (SR) encodes its content as a tree of content items rooted at the Content Sequence
`(0040,A730)`. Each item carries a concept name, a value typed by its `ValueType (0040,A040)`, and a relationship to its
parent. go-radx models this tree as first-class data so the `convert` SR-to-FHIR leg has a documented data layer to read
and build (PRD §5.1 step 6). The supported SR SOP classes (Basic Text SR, Enhanced SR, Comprehensive SR) are declared
in `docs/conformance/dicom.md`.

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

// ConceptNameCode is a coded concept (a Code Sequence item): the (code, scheme, meaning) triple
// from PS3.3 used for ConceptNameCodeSequence (0040,A043) and coded values.
type ConceptNameCode struct {
    CodeValue             string // (0008,0100)
    CodingSchemeDesignator string // (0008,0102), e.g. "DCM", "SCT", "LN"
    CodeMeaning           string // (0008,0104)
}

// ContentItem is one node of the SR content tree.
type ContentItem struct {
    ValueType    ValueType
    Relationship RelationshipType // relationship to the parent; root carries the zero value
    ConceptName  ConceptNameCode  // (0040,A043) what this item measures or states

    // Value fields; only the field matching ValueType is populated.
    Text        string          // TEXT
    Code        ConceptNameCode  // CODE: the coded value (0040,A168)
    MeasuredValue Decimal        // NUM: numeric value (0040,A30A); units in MeasurementUnits
    MeasurementUnits ConceptNameCode // NUM: units of measurement (0040,08EA)
    PersonName  PersonName       // PNAME
    DateTime    DT               // DATE/TIME/DATETIME, preserved lexical form
    UID         UID              // UIDREF
    Referenced  []ReferencedSOPInstance // COMPOSITE/IMAGE referenced instances

    Children []ContentItem // nested content (CONTAINS and other relationships)
}

// ReferencedSOPInstance pairs a referenced SOP Class with its SOP Instance. It is the single
// shape reused by dimse and dicomweb; SR COMPOSITE/IMAGE items reference instances through it.
type ReferencedSOPInstance struct {
    SOPClassUID    SOPClassUID
    SOPInstanceUID SOPInstanceUID
}
```

An SR document is read from a parsed `DataSet` and built back into one through entry points on the package:

```go
// ParseSR reads the SR content tree from ds, starting at the root content item and recursing
// through the Content Sequence (0040,A730). It returns a typed error if ds is not an SR IOD
// (its SOP Class is not a supported SR SOP class) or the tree is malformed.
func ParseSR(ds *DataSet) (*ContentItem, error)

// BuildSR encodes root into the Content Sequence of ds, setting ValueType, RelationshipType,
// ConceptNameCodeSequence, and the value attributes for each node recursively.
func BuildSR(ds *DataSet, root *ContentItem) error
```

The tree is bounded by the same sequence-depth cap as any other `SQ` nesting (default 64), so a maliciously deep SR
returns a typed `LimitExceededError` rather than overflowing the stack.

## Part 10 file format

A Part 10 file is a 128-byte preamble, the `DICM` magic, the File Meta Information group (group `0002`, always Explicit
VR Little Endian), and the main dataset encoded in the transfer syntax named by `(0002,0010)`.

### Reading

```go
// ReadFile opens path and parses it as a Part 10 file.
func ReadFile(path string, opts ...ReadOption) (*File, error)

// Read parses a Part 10 stream. The reader honours the negotiated/declared
// transfer syntax for the main dataset; it does not assume Implicit VR LE.
func Read(r io.Reader, opts ...ReadOption) (*File, error)

type File struct {
    Preamble [128]byte
    Meta     *FileMeta
    DataSet  *DataSet
}

type FileMeta struct {
    // Typed access to the required group-0002 elements.
    MediaStorageSOPClassUID    SOPClassUID
    MediaStorageSOPInstanceUID SOPInstanceUID
    TransferSyntaxUID          TransferSyntax
    ImplementationClassUID     UID
    // ... plus the raw group-0002 DataSet for any optional elements.
    Elements *DataSet
}
```

`ReadOption` is the functional-options entry point; there is no global mutable config (PRD §8.1, §9.4):

```go
func WithMaxElementLen(n uint32) ReadOption   // cap a single element's value field
func WithMaxSequenceDepth(n int) ReadOption   // cap SQ nesting (default 64)
func WithLenientDates() ReadOption            // accept partial DA forms
func WithStopAtPixelData() ReadOption         // defer/skip the pixel-data element for partial reads
func WithDefaultCharacterSet(cs ...string) ReadOption
```

Two safety behaviours are mandatory and tested as regressions (PRD §9.2, §9.3):

- **Truncation is a failure.** The reader accepts EOF only at a clean top-level tag boundary before any bytes of the
  next element are consumed. EOF inside a value, item, sequence, or pixel fragment propagates as `io.ErrUnexpectedEOF`.
  The prototype treated a wrapped `io.EOF` from value/sequence parsing as a normal end of dataset, accepting truncated
  files as complete (Codex DCM-003) — in medical imaging that is data corruption, not a graceful end.
- **Length fields are untrusted.** Value Length is encoded data, not an allocation request. Every 32-bit length is
  validated against bytes actually remaining in a bounded reader before any `make([]byte, n)`. Negative, overflowing,
  and underflowing conversions are rejected before allocation; an element claiming more bytes than remain is a typed
  error. The prototype converted attacker-controlled 32-bit lengths straight to `int` and passed them to
  `make([]byte, n)`, enabling multi-gigabyte allocations and 32-bit overflow from a hostile file (Codex DCM-004).

### Writing

```go
// WriteFile encodes f to path in f.Meta.TransferSyntaxUID.
func WriteFile(path string, f *File, opts ...WriteOption) error

// Write encodes f to w. The encoder is selected from the declared transfer syntax;
// an unsupported transfer syntax is rejected before any bytes are written.
func Write(w io.Writer, f *File, opts ...WriteOption) error
```

A `*DataSet` exposes a convenience writer used in the worked examples and the consuming references (`dimse`,
`dicomweb`). It synthesises a Part 10 `File`, deriving the file-meta SOP Class/Instance UIDs from the dataset's
`(0008,0016)`/`(0008,0018)` elements and writing in the supplied transfer syntax:

```go
// WriteFile wraps ds in a minimal Part 10 File and writes it to path in ts. The
// File Meta MediaStorageSOPClassUID/MediaStorageSOPInstanceUID are taken from the
// dataset's SOPClassUID (0008,0016) and SOPInstanceUID (0008,0018); it returns a
// typed error if either is absent or ts is unsupported.
func (ds *DataSet) WriteFile(path string, ts TransferSyntax, opts ...WriteOption) error
```

The writer is transfer-syntax-faithful. It selects a byte-order- and VR-aware encoder from the declared transfer syntax
and, for Deflated Explicit VR LE, deflates the main dataset after the file-meta group. The prototype only distinguished
explicit from implicit VR, always emitted Little Endian, and never deflated — so declaring Explicit VR BE or Deflated
while writing uncompressed Little Endian produced corrupt files (Codex DCM-002, PS3.5 Annex A).

The transport subsystems carry a dataset as a **bare element stream** — no Part 10 preamble and no file-meta group —
because the framing lives outside the dataset (P-DATA-TF fragments for `dimse`, multipart parts for `dicomweb`). The
package exposes that stream codec so the subsystems reuse the one transfer-syntax-faithful encoder rather than
redeclaring their own (the prototype's DIMSE layer hand-rolled an Explicit/Implicit VR encoder that ignored byte order,
sequences, and the active character set):

```go
// EncodeDataSet writes ds as a bare element stream in ts (PS3.7 §6.3.1): the dataset elements in
// ascending tag order, no preamble or file-meta group. Deflated Explicit VR LE deflates the
// stream. An encapsulated or empty transfer syntax is rejected before any bytes are written.
func EncodeDataSet(w io.Writer, ds *DataSet, ts TransferSyntax) error

// DecodeDataSet reads a bare element stream from r in ts, the inverse of EncodeDataSet. A clean
// EOF at a top-level tag boundary ends the dataset; truncation surfaces as io.ErrUnexpectedEOF.
// Every wire length is bounds-checked against a bounded reader before allocation (PRD §9.3).
func DecodeDataSet(r io.Reader, ts TransferSyntax, opts ...ReadOption) (*DataSet, error)
```

The File Meta Information group is always written Explicit VR Little Endian, and `(0002,0000)` File Meta Information
Group Length is recomputed and written first. The encoder serialises the group-0002 elements to a buffer, counts the
bytes following the group-length element through the last group-0002 element, and prepends the `UL` group-length with
that count (PS3.10 Table 7.1-1, where group length is Type 1). The prototype started file-meta at `(0002,0001)` and
never wrote the group length at all (Codex DCM-001), producing files that strict readers reject.

## Transfer syntaxes

`TransferSyntax` is the UID-identified encoding: byte order, implicit-versus-explicit VR, and compression. It is the
single `dicom.TransferSyntax` type reused by `dimse` and `dicomweb` (glossary). v1 reads and writes the four
uncompressed transfer syntaxes; compressed syntaxes are recognised for transport and pixel decoding but the main dataset
is never written in a compressed syntax (compression applies only to the pixel-data element).

```go
type TransferSyntax UID

const (
    ImplicitVRLittleEndian         TransferSyntax = "1.2.840.10008.1.2"
    ExplicitVRLittleEndian         TransferSyntax = "1.2.840.10008.1.2.1"
    DeflatedExplicitVRLittleEndian TransferSyntax = "1.2.840.10008.1.2.1.99"
    ExplicitVRBigEndian            TransferSyntax = "1.2.840.10008.1.2.2" // retired, read+write
    RLELossless                    TransferSyntax = "1.2.840.10008.1.2.5"
    JPEGBaseline8Bit               TransferSyntax = "1.2.840.10008.1.2.4.50"
    // ... JPEG-LS, JPEG 2000, HTJ2K UIDs registered for transport/decode.
)

func (ts TransferSyntax) IsImplicitVR() bool
func (ts TransferSyntax) IsBigEndian() bool
func (ts TransferSyntax) IsDeflated() bool
func (ts TransferSyntax) IsEncapsulated() bool // compressed pixel data in fragments
func (ts TransferSyntax) Name() string         // registered name, e.g. "Explicit VR Little Endian"
```

## Pixel data pipeline

Pixel Data is element `(7FE0,0010)`. It is either **native**, a single contiguous `OB`/`OW` value under an
uncompressed transfer syntax, or **encapsulated**, fragmented `(FFFE,E000)` items under a compressed transfer syntax,
optionally preceded by a Basic Offset Table and followed by a `(FFFE,E0DD)` Sequence Delimitation Item.

```go
type PixelData struct {
    // Geometry resolved from the dataset's image-pixel module.
    Rows, Columns     uint16
    SamplesPerPixel   uint16
    BitsAllocated     uint16
    BitsStored        uint16
    PixelRepresentation uint16
    NumberOfFrames    int
    TransferSyntax    TransferSyntax
    // internal: native bytes or encapsulated fragments
}

// Frames iterates decoded frames. For native data it slices the contiguous buffer;
// for encapsulated data it decodes each fragment group through the codec for the
// transfer syntax. It returns a typed CodecUnavailableError when no codec is built in.
func (p *PixelData) Frames() iter.Seq2[Frame, error]

type Frame struct {
    Index int
    Pixels []byte // decoded, one frame's worth; native byte order resolved
}

// BasicOffsetTable returns the first item of encapsulated pixel data (32-bit
// per-frame offsets), if present.
func (p *PixelData) BasicOffsetTable() (BasicOffsetTable, bool)
```

Encapsulated pixel data is parsed as a bounded stream of fragments, not accumulated into one unbounded byte blob. Each
fragment requires a valid `(FFFE,E000)` item header with an even length, validated against the bytes remaining in the
bounded reader; a truncated trailing header is an error, not a silent `break`. The prototype rebuilt fragments into a
single unbounded slice and accepted truncated headers (Codex DCM-006), turning malformed input into partial images. The
Basic Offset Table is parsed per `pydicom`'s `parse_basic_offsets` (`encaps.py`): the first item must be `(FFFE,E000)`,
and per-frame fragment grouping uses the offsets where present.

Native and RLE decoding/encoding are pure Go. RLE Lossless (`1.2.840.10008.1.2.5`) is implemented per PS3.5 Annex G with
bounds-checked segment math.

### Codec strategy (optional CGo)

Compressed JPEG-family codecs (JPEG baseline/extended, JPEG-LS, JPEG 2000, HTJ2K) are the source of the prototype's
build fragility, so go-radx follows the PRD §7.3 decision: **pure-Go where it exists; optional CGo behind a build tag
for the rest, never load-bearing.** The core library builds and passes its non-pixel tests with CGo disabled.

The two directions are scoped differently:

- **Decode** is supported for every compressed transfer syntax in scope. With the codec build tag enabled the pipeline
  decodes the JPEG family; native and RLE Lossless decode in pure Go regardless of build tag.
- **Encode/transcode** is supported only where an encoder exists, starting with RLE and JPEG 2000 lossless, and is gated
  behind the same optional-CGo build tag. Transcoding is **off by default**: re-encoding pixel data to a different
  transfer syntax is an explicit opt-in (for example the CLI's `radx store --transcode` flag), never automatic, because
  silent re-compression of clinical pixels is a data-integrity hazard.

```go
// Codec decodes, and where an encoder exists encodes, one transfer syntax's pixel frames.
type Codec interface {
    TransferSyntax() TransferSyntax
    Decode(frame []byte, geom PixelGeometry) ([]byte, error)
    // Encode returns ErrEncodeUnsupported for a decode-only codec.
    Encode(frame []byte, geom PixelGeometry) ([]byte, error)
    CanEncode() bool
}

// RegisterCodec makes c available to the pixel pipeline. Build-tagged CGo codecs
// register themselves in init; with CGo disabled, no JPEG-family codec registers.
func RegisterCodec(c Codec)

// ErrCodecUnavailable is returned by Frames when no codec is registered for an
// encapsulated transfer syntax. It names the missing transfer syntax.
var ErrCodecUnavailable = errors.New("dicom: codec unavailable for transfer syntax")

// ErrEncodeUnsupported is returned when a transcode is requested for a transfer
// syntax whose registered codec is decode-only. It names the transfer syntax.
var ErrEncodeUnsupported = errors.New("dicom: encode unsupported for transfer syntax")
```

`docs/conformance/dicom.md` marks each transfer syntax decode-only versus decode+encode so the supported direction is
unambiguous. With CGo disabled, requesting frames from a JPEG 2000 instance returns `ErrCodecUnavailable` naming the
transfer syntax as a clear, typed failure, not a build break or a silent partial image. CGo codecs are hardened against
hostile input per PRD §9.3 (correcting Codex DCM-014/DCM-015): every C allocation result is checked, image dimensions
are capped before allocation, sizes are converted through checked `size_t`/`int`, output is never truncated by an
unchecked `C.int(size)` cast, and crash/fuzz tests run in subprocesses with timeouts so a hang fails CI rather than
being skipped.

## De-identification — PS3.15 Basic Application Level Confidentiality Profile

The `dicom` package implements the PS3.15 Annex E Basic Application Level Confidentiality Profile (the "Basic Profile").
Neither `pydicom` nor `pynetdicom` ships a built-in profile; this is a deliberate go-radx capability (PRD §6.3). It
applies the PS3.15 Table E.1-1 actions recursively through sequence items, remaps UIDs consistently through a stable
map, and records the required de-identification metadata.

```go
// Profile applies a configured set of PS3.15 Annex E options to a dataset.
type Profile struct {
    // internal: action table, option flags, UID remap, dummy-value policy
}

type ProfileOption func(*Profile)

func WithRetainPatientCharacteristics() ProfileOption // PS3.15 option: retain age/sex/etc.

// WithRetainLongitudinalTemporalInformation opts in to the PS3.15 "Retain Longitudinal Temporal
// Information" sub-option: instead of the default removal/zeroing, dates and times are kept either
// as-is (DateModeKeep) or shifted by one consistent per-study offset (DateModeShift). It is
// opt-in precisely because retaining temporal data weakens de-identification.
func WithRetainLongitudinalTemporalInformation(mode DateMode) ProfileOption

func WithRetainDeviceIdentity() ProfileOption
func WithRetainUIDs() ProfileOption                   // skip UID remapping (off by default)
func WithDummyValues(replacements map[Tag]string) ProfileOption

// WithRetainSafePrivate preserves private attributes whose private creator is on the
// supplied allow-list (PS3.15 "Retain Safe Private" sub-option). Without it the Basic
// Profile removes all private tags, since go-radx implements no private SOP-class logic.
func WithRetainSafePrivate(creators ...string) ProfileOption

// WithAllowBurnedInPixelData accepts the residual risk of burned-in identifying pixel
// data: Deidentify will not return ErrBurnedInPixelData even when BurnedInAnnotation is
// "YES". The caller asserts it has handled the pixels by other means; the profile never
// itself removes burned-in pixel text in v1.
func WithAllowBurnedInPixelData() ProfileOption

// NewProfile builds the Basic Profile with the given options. The UID remap is
// seeded from g so Study/Series/SOP UIDs are rewritten consistently and a given
// source UID always maps to the same replacement within one run.
func NewProfile(g *UIDGenerator, opts ...ProfileOption) *Profile

// Deidentify returns a de-identified deep copy of ds. The source is never mutated.
// It walks every level, including sequence items, applies the Table E.1-1 action
// for each attribute, remaps UIDs through the stable map, and sets the
// de-identification metadata: PatientIdentityRemoved (0012,0062) = "YES" and
// DeidentificationMethod (0012,0063) describing the applied profile and options.
func (p *Profile) Deidentify(ds *DataSet) (*DataSet, error)
```

The implementation corrects the prototype's de-identification, which was a sparse, top-level-only action table that
claimed PS3.15 compliance while leaving PHI inside sequence items and doing no pixel cleaning (Codex DCM-013). Here:

- The full Table E.1-1 action set (remove `X`, replace with dummy `D`, replace with zero-length `Z`, clean `C`, remap
  UID `U`, keep `K`) is applied to every matched attribute at every nesting level, recursing through `SQ` items.
- UID remapping is consistent within a run: the same source UID always yields the same replacement, preserving the
  Study/Series/Instance reference graph after de-identification. New UIDs are minted by the supplied `UIDGenerator`, so
  the caller controls the organisation root.
- Dates and times are **removed or zeroed by default**: the Basic Profile applies the `X`/`Z` action to date and time
  attributes so no temporal data survives unless the caller opts in. `WithRetainLongitudinalTemporalInformation` is the
  only way to keep them, and it applies a single consistent per-study shift (or keeps them verbatim).
- `PatientIdentityRemoved (0012,0062)` is set to `YES` and `DeidentificationMethod (0012,0063)` records the applied
  profile and retained options.

### Documented scope and limits

go-radx provides the de-identification **capability**, not a compliance guarantee (PRD §6.3, §9.1). The consumer owns
their de-identification policy and must judge whether this profile meets their needs. Explicit limits for v1:

- **Burned-in pixel and overlay cleaning is not performed.** The Clean Pixel Data (`C`) and overlay actions detect and
  flag attributes but do **not** remove identifying text rendered into pixel or overlay data. `BurnedInAnnotation`
  `(0028,0301)` is read and surfaced; if it is `YES` (or absent and the modality is known to burn in annotations),
  `Deidentify` returns a typed `ErrBurnedInPixelData` unless the caller explicitly accepts the residual risk through an
  option. Treating burned-in PHI as cleaned would be the most dangerous possible silent failure, so the default is
  fail-closed.
- **The optional sub-profiles of PS3.15** beyond the Basic Profile (Clean Recognizable Visual Features, Clean Graphics,
  Clean Structured Content, Clean Descriptors) are not implemented in v1; their options on `Profile` are reserved.
- **Private tags** are removed by default (the Basic Profile's safe action) because go-radx implements no private
  SOP-class logic (PRD §3.2). `WithRetainSafePrivate` can preserve creators on an allow-list the caller supplies.

The profile is tested against an attribute checklist as a feature, not gated as a compliance bar (PRD §11.2).

## Behaviour and error model

Errors are values, never panics on malformed input (PRD §9.2, §9.3). The package returns typed errors a caller matches
with `errors.Is`/`errors.As`:

```go
// ErrTruncated wraps io.ErrUnexpectedEOF; the message names the element or offset.
var ErrTruncated = errors.New("dicom: truncated input")

// LimitExceededError is returned when a length, depth, or count cap is hit.
type LimitExceededError struct {
    Tag    Tag    // the offending element, if known
    Limit  uint64
    Actual uint64
    Kind   string // "element-length", "sequence-depth", ...
}

// ValueError reports a value that does not conform to its VR (bad UID, bad date,
// odd binary length, over-long PN component). It names the tag and VR.
type ValueError struct {
    Tag Tag
    VR  VR
    Msg string
}
```

Every error carries the context needed to act — which element (by keyword and `(gggg,eeee)`), which VR, which byte
offset or file — while honouring the no-PHI-by-default rule: diagnostics name identifiers and structure, never patient
values (PRD §8.2, §9.1). For example, a bad patient name reports `invalid PN at PatientName (0010,0010): component 1
exceeds 64 characters`, not the name itself.

Three invariants the model guarantees:

- A successfully read `*File` is structurally complete: no truncation was tolerated, all sequences closed, all lengths
  validated against bytes present.
- A successful `Write` produced bytes that re-read to an equal dataset under the same transfer syntax (round-trip
  stability), and the file-meta group length is exact.
- `Deidentify` never mutates its input and never reports success while leaving a Table E.1-1-listed attribute in place
  at any nesting level; burned-in pixel PHI is fail-closed.

## Worked examples

### Read a file, inspect typed values

```go
package main

import (
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/dicom"
)

func main() {
    f, err := dicom.ReadFile("study.dcm")
    if err != nil {
        log.Fatalf("read: %v", err) // truncation or hostile length surfaces here, not silently
    }

    if name, ok := f.DataSet.GetPersonName(dicom.LookupKeywordTag("PatientName")); ok {
        fmt.Println("family:", name.Alphabetic.FamilyName)
    }
    if uid, ok := f.DataSet.GetUID(dicom.LookupKeywordTag("StudyInstanceUID")); ok {
        fmt.Println("study:", uid)
    }
    fmt.Println("transfer syntax:", f.Meta.TransferSyntaxUID.Name())
}
```

`LookupKeywordTag` is the convenience for a compile-time literal keyword: it returns the bare `Tag` and panics only on a
keyword that is not in the standard dictionary, which is a programmer typo, never reachable from input. For a
keyword that comes from runtime input, use `LookupKeyword`, which returns `(Tag, bool)`. The generated `dicom.Tag*`
constants are an alternative for literals that avoids the call entirely (`dicom.TagPatientName`).

### Build and write a dataset

```go
ds := dicom.NewDataSet()
ds.Set(dicom.Element{
    Tag:   dicom.LookupKeywordTag("PatientName"),
    VR:    dicom.VRPN,
    Value: dicom.NewStrings(dicom.VRPN, "Doe^Jane"),
})

gen, _ := dicom.NewUIDGenerator("1.2.826.0.1.3680043.2.1143") // your registered root
study := gen.Generate()
ds.Set(dicom.Element{
    Tag:   dicom.LookupKeywordTag("StudyInstanceUID"),
    VR:    dicom.VRUI,
    Value: dicom.NewStrings(dicom.VRUI, string(study)),
})

f := &dicom.File{
    Meta: &dicom.FileMeta{
        TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
        MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7", // Secondary Capture
        MediaStorageSOPInstanceUID: dicom.SOPInstanceUID(gen.Generate()),
    },
    DataSet: ds,
}
if err := dicom.WriteFile("out.dcm", f); err != nil {
    log.Fatal(err) // group length is recomputed; transfer syntax is honoured
}
```

`MediaStorageSOPClassUID` accepts the untyped string constant directly because it is a named type over `UID`; the
generated `gen.Generate()` returns a `UID`, so it is converted to `SOPInstanceUID` explicitly.

### De-identify, preserving the reference graph

```go
gen := dicom.NewRandomUIDGenerator() // 2.25. UUID root; no registration needed
prof := dicom.NewProfile(gen,
    // Dates are removed by default; this opts in to a consistent per-study shift instead.
    dicom.WithRetainLongitudinalTemporalInformation(dicom.DateModeShift),
)

clean, err := prof.Deidentify(f.DataSet)
if err != nil {
    // e.g. ErrBurnedInPixelData when (0028,0301) == "YES" and no override given
    log.Fatalf("de-identify: %v", err)
}
// Study/Series/SOP UIDs in clean are consistently remapped; (0012,0062) == "YES".
_ = clean
```

## Conformance scope and limits

go-radx is conformant to an explicitly declared subset verified against reference validators (PRD §6.1); it does not
implement all of DICOM. For the `dicom` package, v1 conformance is:

- **Encoding.** Read and write the four uncompressed transfer syntaxes: Implicit VR LE, Explicit VR LE, Explicit VR BE
  (retired), and Deflated Explicit VR LE. Defined- and undefined-length sequences and items round-trip without loss.
- **Value representations.** The 34 standard VRs plus the 4 ambiguous parse-time placeholders, with VR-correct padding
  and length, including the 64-bit `OD`/`OL`/`OV`/`SV`/`UV` and the `UC`/`UR` string VRs.
- **Part 10.** 128-byte preamble, `DICM` prefix, Explicit-VR-LE file-meta with an auto-recomputed group length, and
  rejection of truncated input.
- **Dictionaries.** The standard data dictionary (~5,189 entries), repeating-group resolution for `60xx`/`50xx`, the
  private dictionary for private-creator block lookup, and the UID dictionary (~490 entries). Private tags are parsed
  generically; no private SOP-class logic.
- **Text.** `SpecificCharacterSet` for the default repertoire, ISO 2022 code extensions, UTF-8, GB18030, and GBK.
  `PersonName` with three component groups.
- **Pixel data.** Native (`OB`/`OW`) and RLE Lossless in pure Go, read and write. Decode for every supported compressed
  transfer syntax through optional CGo codecs behind a build tag; with CGo disabled these return `ErrCodecUnavailable`.
  Encoding/transcoding is supported where a codec exists (RLE and JPEG 2000 lossless first), behind the same build tag,
  and is off by default (explicit opt-in). `docs/conformance/dicom.md` marks each transfer syntax decode-only versus
  decode+encode.
- **Structured Report.** `ParseSR`/`BuildSR` over the `ContentItem` tree (CONTAINER/TEXT/CODE/NUM/PNAME and the other
  value types) for the SR SOP classes declared in `docs/conformance/dicom.md` (Basic Text SR, Enhanced SR, Comprehensive
  SR). This is the data layer the `convert` SR-to-FHIR leg reads.
- **De-identification.** PS3.15 Annex E Basic Profile with recursive traversal, consistent UID remapping, and
  de-identification metadata. Dates and times are removed/zeroed unless the caller opts in through
  `WithRetainLongitudinalTemporalInformation`. Burned-in pixel/overlay cleaning and the optional clean sub-profiles are
  **not** performed (fail-closed on detected burned-in PHI).

Validation is gated in CI by `dciodvfy` (dcmtk) and round-trips against vendored `pydicom` fixtures (PRD §11.1). The
published Conformance Statement (`docs/conformance/`) is the single source of truth for the supported SOP classes and
transfer syntaxes; growth beyond this subset is a deliberate, reviewed change.

## See also

- `docs/prd/go-radx-prd.md` — the umbrella PRD (parity floor §6.2, API commitments §8.1, design principles §8.2,
  NFRs §9).
- `UBIQUITOUS_LANGUAGE.md` — canonical Go names and the cross-standard collision rules.
- The `dimse` reference — how datasets are negotiated and transported over the network.
- The `dicomweb` reference — how datasets are serialised to WADO-RS / STOW-RS / QIDO-RS.
