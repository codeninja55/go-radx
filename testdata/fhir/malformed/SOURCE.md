# Malformed FHIR corpus

This directory holds a small, purposeful set of deliberately **malformed** FHIR JSON instances. It is the first
contribution to the Phase 4 hostile-input gate and the seed corpus for the FHIR fuzz targets
(`fhir/r5/fuzz_test.go`: `FuzzUnmarshalResource`, `FuzzValidate`). Each instance exercises one class of decode/validate
fault the decode and in-process `Validate` surface must survive without panicking or leaking data.

## Synthetic, fictitious data only

Per the project PHI rule (PRD §9.1), every instance here is **authored by go-radx with entirely synthetic, fictitious
data**. The only patient-shaped tokens present are the recognisable synthetic sentinels reused from the clean corpus
(`TESTPATIENT`, `Synthetic`); they are markers, not real Protected Health Information, and identify no real person. No
fixture content or filename encodes real PHI.

These are **go-radx originals**, not copies of any upstream corpus. They are hand-authored to be invalid in a specific,
documented way, shaped after the canonical resource structures in the
[HL7 FHIR R5 specification](https://www.hl7.org/fhir/R5/). The HL7 FHIR specification and its examples are published
under the Creative Commons "No Rights Reserved" (CC0) public-domain dedication; the notice is vendored as
`../LICENSE-hl7-fhir.txt`, attributing that reference for the whole `testdata/fhir` tree.

## Fault classes

Each file is invalid in exactly one documented way, so a gate failure points at a known class rather than an ambiguous
parse error.

| File | Fault class | What it exercises |
| --- | --- | --- |
| `truncated-patient.json` | Truncation | A valid Patient cut off mid-array; the decoder must report `io.ErrUnexpectedEOF`, never panic. |
| `wrong-type-active.json` | Type confusion | `active` is a string and `name[0].family` is a number where the schema expects a boolean and a string; the typed decode must error. |
| `two-choice-branches.json` | Choice-group violation | Both `deceasedBoolean` and `deceasedDateTime` set; `Validate` must report the mutual-exclusion violation (Codex FHIR-001). |
| `unknown-resourcetype.json` | Unknown discriminator | A `resourceType` not in the registry; dispatch must fail closed with `ErrUnknownResourceType`. |
| `missing-resourcetype.json` | Absent discriminator | No `resourceType` key; dispatch must fail closed rather than guess a type. |
| `array-where-object.json` | Shape confusion | A scalar/array supplied where an object (CodeableConcept) is expected; the decode must error. |
| `syntax-error.json` | Mid-buffer syntax fault | A doubled comma — a genuine syntax error inside the buffer that must NOT be misreported as truncation. |
| `deeply-nested.json` | Deep nesting | 300 levels of nested Bundle/`entry.resource` polymorphic decode; a stack-depth probe for the recursive decode path. |

## How the corpus is consumed

The FHIR fuzz targets seed from this directory (and the clean `../r5` corpus) so the fuzzer starts in the failure space
the decode/validate surface must survive, then mutates outward. Replaying these seeds under `go test ./fhir/...` (no
`-fuzz` flag) is a regression gate on its own: a seed that crashes the decoder fails the build without a fuzzing run.
The bounded CI fuzz job (`mise run fuzz`) runs the same targets for a fixed budget; a panic, crasher, or hang is a
failure.

The contracts these instances pin:

- **Never panic.** Arbitrary, truncated, wrong-typed, or deeply nested input yields a typed error, never a crash
  (PRD §9.3).
- **Never leak PHI.** No `Validate` issue diagnostic or expression echoes a payload value; messages name elements,
  paths, types, and codes only (PRD §9.1).
- **Truncation is `io.ErrUnexpectedEOF`.** A payload that ends mid-value is matchable with
  `errors.Is(err, io.ErrUnexpectedEOF)`, distinct from a structural syntax fault.

## License attribution

- `../LICENSE-hl7-fhir.txt` — the Creative Commons CC0 1.0 Universal public-domain dedication HL7 applies to the FHIR
  specification and its examples, attributed as the shape reference for this corpus. No HL7 example content is
  redistributed; these instances are go-radx synthetic originals shaped after HL7's canonical structures and then
  deliberately broken.
