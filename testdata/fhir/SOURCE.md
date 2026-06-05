# FHIR test corpus

This directory holds the FHIR conformance corpus: a small, purposeful set of FHIR resource instances used by the M6a
conformance gate (`tools/fhir-conformance/validate.sh`) and the corpus harness in `fhir/r5/corpus_test.go`. Fixtures
are test corpora only, vendored with upstream license attribution per PRD §5.3 and §11.1.

## Synthetic, fictitious data only

Per the project PHI rule (PRD §9.1), every instance in this corpus is **authored by go-radx with entirely synthetic,
fictitious data**. Patient names follow a `TESTPATIENT^<given>` scheme; the medical record number (`MRN0001234`),
identifiers, dates, and the `example.org` / `go-radx.test` systems are invented and do not identify any real person.
No fixture content or filename encodes real Protected Health Information.

These instances are **go-radx originals**, not copies of any upstream corpus. They are *shaped after* the canonical
resource structures documented in the [HL7 FHIR R5 specification](https://www.hl7.org/fhir/R5/) examples, which is the
project's reference for canonical resource shapes. The HL7 FHIR specification and its examples are published under the
Creative Commons "No Rights Reserved" (CC0) public-domain dedication; the notice is vendored as `LICENSE-hl7-fhir.txt`
to attribute that reference. Rather than redistribute HL7's example JSON, go-radx authored equivalent synthetic
instances with fictitious data and uses the HL7 examples only as the structural/shape reference.

## How the corpus is produced

The committed `r5/*.json` files are the output of `tools/fhir-conformance/fixtures`, a small program that constructs a
fully-populated go-radx-generated instance of each workflow resource and marshals it through the generated
`MarshalJSON` (the canonical wire form, with the primitive `_field` siblings trailing). Regenerate the corpus with:

```bash
go run ./tools/fhir-conformance/fixtures testdata/fhir/r5
```

Validating go-radx's **own** marshalling — rather than a borrowed example file — is the point: a validator error
reflects a real conformance defect in the generated code, not a thin or hand-edited fixture.

## Provenance and coverage

The corpus covers the radiology + clinical **workflow set** the M6a gate commits to. Each instance is fully populated
(choice types, primitive values, references) so the validator exercises real structure. The Bundle is a `collection`
that references the other workflow resources by `fullUrl`.

| File | Resource | Authorship | Shape reference | Exercises |
| --- | --- | --- | --- | --- |
| `r5/Patient.json` | `Patient` | go-radx synthetic | HL7 FHIR R5 Patient example | identifier, name, gender, birthDate, narrative |
| `r5/Encounter.json` | `Encounter` | go-radx synthetic | HL7 FHIR R5 Encounter example | status, class CodeableConcept, subject reference |
| `r5/ServiceRequest.json` | `ServiceRequest` | go-radx synthetic | HL7 FHIR R5 ServiceRequest example | status, intent, `code` CodeableReference, subject/encounter references |
| `r5/ImagingStudy.json` | `ImagingStudy` | go-radx synthetic | HL7 FHIR R5 ImagingStudy example | status, modality, series + instance backbone, DICOM UID identifiers |
| `r5/Observation.json` | `Observation` | go-radx synthetic | HL7 FHIR R5 Observation (body weight) example | status, code, `value[x]` Quantity choice, effective choice |
| `r5/DiagnosticReport.json` | `DiagnosticReport` | go-radx synthetic | HL7 FHIR R5 DiagnosticReport example | status, code, study/result references |
| `r5/OperationOutcome.json` | `OperationOutcome` | go-radx synthetic | HL7 FHIR R5 OperationOutcome example | issue with severity + code |
| `r5/CapabilityStatement.json` | `CapabilityStatement` | go-radx synthetic | HL7 FHIR R5 CapabilityStatement example | kind=instance + implementation (cpb-14), fhirVersion, rest |
| `r5/Bundle.json` | `Bundle` | go-radx synthetic | HL7 FHIR R5 Bundle example | `collection` type bundling the workflow set by fullUrl |

## How the gate consumes the corpus

The conformance gate does not read these committed files directly: it regenerates the same workflow set into a temp
directory and runs the official HL7 FHIR validator over it (see `tools/fhir-conformance/validate.sh`). The committed
corpus here is the inspectable, version-controlled evidence of that output, and is exercised by `fhir/r5/corpus_test.go`
in the unit suite: every instance is decoded, structurally round-tripped, and passed through go-radx's own `Validate`
with no errors. (The Bundle's full decode round-trip is the known polymorphic-resource decode gap — see the corpus
harness comment — so it is checked on the marshal side; the validator gate is the authoritative marshal-side check.)

## Upstream sources

- HL7 FHIR R5 specification and examples (shape reference): <https://www.hl7.org/fhir/R5/>. The canonical resource
  structures these synthetic instances follow are demonstrated by the per-resource examples in the R5 specification.
- HL7 FHIR validator (`validator_cli.jar`, hapifhir/org.hl7.fhir.core): pinned in `tools/versions`
  (`fhir-validator.*` keys, version 6.9.9 with a recorded SHA-256).

## License attribution

- `LICENSE-hl7-fhir.txt` — the Creative Commons CC0 1.0 Universal public-domain dedication HL7 applies to the FHIR
  specification and its examples, attributed as the shape reference for this corpus. No HL7 example content is
  redistributed here; the instances are go-radx synthetic originals shaped after HL7's canonical structures.
