# HL7 FHIR R5 definition bundle (vendored)

This directory holds the pinned, checksum-verified HL7 FHIR R5 `StructureDefinition` and `ValueSet` bundles the
generator reads. The bundle is committed reference data so generation is reproducible offline and in CI without
trusting a live HL7 endpoint; generation itself never touches the network.

## Provenance

- FHIR release: R5 `5.0.0` (`buildId` 2aecd53, build date 2023-03-26)
- Source URL: <https://hl7.org/fhir/R5/definitions.json.zip>
- Vendored on: 2026-06-02

The download archive `definitions.json.zip` unzips to several bundles; only the three the generator needs are
vendored here:

- `profiles-types.json` — complex and primitive datatype `StructureDefinition`s (`Period`, `Reference`, `Identifier`,
  `CodeableConcept`, `Quantity`, `HumanName`, `Coding`, `CodeableReference`, and the rest).
- `profiles-resources.json` — resource `StructureDefinition`s (`Patient`, `Observation`, `ServiceRequest`,
  `ImagingStudy`, `DiagnosticReport`, `Bundle`, `OperationOutcome`, `Encounter`, and the rest — 162 entries).
- `valuesets.json` — the `ValueSet` and `CodeSystem` bundle the required-binding enums enumerate codes from
  (`administrative-gender` and the rest).

## License and attribution

The HL7 FHIR specification, including these `StructureDefinition`, `ValueSet`, and `CodeSystem` artifacts, is published
by HL7 under the Creative Commons "No Rights Reserved" (CC0 1.0) public domain dedication. The bundle is committed
verbatim as reference data; no artifact is modified. Some value sets reference external terminologies (SNOMED CT,
LOINC, DICOM, UCUM) that carry their own third-party use terms; those codes are referenced by URL here, not
redistributed as code lists. See <https://hl7.org/fhir/R5/license.html>.

## Integrity

`SHA256SUMS` records a SHA-256 per file. The generator's loader (Increment 1) verifies these checksums before parsing
and fails closed on any mismatch, a missing file, or an unpinned bundle. Verify on the command line with:

```bash
shasum -a 256 -c SHA256SUMS
```

## Refreshing

Refresh the bundle deliberately when bumping the pinned FHIR release, never at generate time. Run the refresh-only
mise task, which re-downloads and re-checksums:

```bash
mise run fhir:refresh-r5
```
