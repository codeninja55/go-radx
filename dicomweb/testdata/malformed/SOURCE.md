# Malformed DICOMweb corpus

This directory holds a small, purposeful set of deliberately **malformed** inputs for the two `dicomweb` parsers that
sit on the trust boundary of a hostile request: the DICOM-JSON decoder (`UnmarshalJSON`, PS3.18 Annex F) and the
`multipart/related` reader (`NewMultipartReader` / `NextPart`). It is the seed corpus for the package fuzz targets
(`parser_fuzz_test.go`: `FuzzUnmarshalJSON`, `FuzzMultipartReader`, `FuzzParseMediaType`). Each file exercises one class
of decode or framing fault the parser must survive without panicking, hanging, exhausting memory, or leaking data.

## Synthetic, fictitious data only

Per the project PHI rule (PRD §9.1), every file here is **authored by go-radx with entirely synthetic, fictitious
data**. The only patient-shaped token present is the recognisable synthetic sentinel `ZZZ^SENTINEL` (and the obviously
fake UID `1.2.3`); they are markers, not real Protected Health Information, and identify no real person. No file content
or filename encodes real PHI. These are **go-radx originals**, hand-authored to be invalid in a specific, documented
way, shaped after the DICOM-JSON and multipart/related forms in
[DICOM PS3.18](https://dicom.nema.org/medical/dicom/current/output/html/part18.html).

## JSON fault classes (`json/`)

Each DICOM-JSON file is invalid in exactly one documented way, so a fuzz crash points at a known class rather than an
ambiguous parse error.

| File | Fault class | What it exercises |
| --- | --- | --- |
| `truncated-mid-value.json` | Truncation | A document cut off inside a `Value` array; the decoder must report `io.ErrUnexpectedEOF` via `*TruncatedError`, never panic. |
| `truncated-mid-object.json` | Truncation | A document cut off after the `vr` key; same truncation contract. |
| `null-value.json` | Present-but-null payload | `"Value": null` where Annex F omits the key for an empty element; the decoder must reject it. |
| `multiple-payload-forms.json` | Conflicting payload | Both `Value` and `InlineBinary` present; Annex F permits at most one, so decode must error. |
| `binary-vr-value-array.json` | Payload-form mismatch | A binary VR (OB) carrying a `Value` array instead of `InlineBinary`/`BulkDataURI`; decode must error. |
| `bad-base64.json` | Invalid encoding | `InlineBinary` that is not valid base64; decode must error, never panic on the decode step. |
| `short-tag-key.json` | Malformed key | A tag key that is not eight hex digits; decode must error and the message must not echo the key. |
| `unknown-vr.json` | Unknown VR | A `vr` token outside the 34 real VRs; decode must error without echoing the token. |
| `us-out-of-range.json` | Range violation | A US value above 65535; decode must reject it rather than wrap on a later binary write. |
| `fractional-integer.json` | Type confusion | A fractional number for an integer VR; decode must reject it rather than truncate. |
| `ow-odd-length.json` | Width violation | An OW payload whose decoded byte length is not a multiple of two; decode must error. |
| `array-not-object.json` | Shape confusion | A top-level JSON array where a tag-keyed object is expected; decode must error. |
| `pn-embedded-equals.json` | Delimiter smuggling | A PN component group carrying the `=` group separator; decode must reject it so the name cannot be re-split into the wrong groups. |
| `deeply-nested-sq.json` | Deep SQ nesting | Twelve levels of nested SQ; a stack-depth probe the fuzzer runs against a tightened `WithMaxJSONDepth`, expecting `*LimitExceededError`. |

## Multipart fault classes (`multipart/`)

Each multipart file is a media-type line, a newline, then the body the reader sees, so a fuzz seed reconstructs both
inputs `NewMultipartReader` (the media type) and `NextPart` (the body) take.

| File | Fault class | What it exercises |
| --- | --- | --- |
| `unparseable-media-type.txt` | Media-type fault | A string `mime.ParseMediaType` cannot parse; the constructor must return a typed error. |
| `wrong-media-type.txt` | Wrong type | A non-`multipart/related` media type; the constructor must reject it with `ErrUnsupported`. |
| `missing-boundary.txt` | Missing boundary | `multipart/related` with no `boundary` parameter; the constructor must reject it. |
| `truncated-part.txt` | Truncation | A body that ends before its closing boundary; `NextPart`/drain must surface `*TruncatedError`. |
| `malformed-part-header.txt` | Framing fault | A part header the parser cannot frame; `NextPart` must return `*MalformedPartError` without echoing raw bytes. |
| `boundary-never-appears.txt` | Boundary mismatch | A declared boundary that never appears in the body; iteration must terminate, not loop. |
| `many-empty-parts.txt` | Part-count cap | Forty empty parts against the fuzz target's tightened `MaxParts`; the cap must trip with `*LimitExceededError`. |
| `oversized-part.txt` | Per-part byte cap | One 8 KiB part against the fuzz target's tightened `MaxPartBytes`; the cap must trip before the body is fully read. |

## How the corpus is consumed

The fuzz targets seed from this directory plus a handful of inline edge cases, so the fuzzer starts in the failure space
the parsers must survive and then mutates outward. Replaying these seeds under `go test ./dicomweb/...` (no fuzzing
build) runs every seed once, so a seed that would crash a parser is caught without a fuzzing run. Each target holds the
recursion, part-count, and per-part-size caps low so a hostile input trips its `*LimitExceededError` before it can drive
the run to OOM or a hang.
