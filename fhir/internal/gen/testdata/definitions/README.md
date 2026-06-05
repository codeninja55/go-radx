# Vendored HL7 FHIR definition bundles

This directory holds the pinned, checksum-verified HL7 FHIR `StructureDefinition`, `ValueSet`, and `CodeSystem`
bundles the FHIR generator reads. They are committed reference data so generation is reproducible offline and in CI
without trusting a live HL7 endpoint; generation itself never touches the network.

## Layout

```
definitions/
├── .gitattributes      # treats the bundles as binary so git performs no EOL normalisation
├── r4/                 # HL7 FHIR R4 4.0.1 bundle (see r4/SOURCE.md)
└── r5/                 # HL7 FHIR R5 5.0.0 bundle (see r5/SOURCE.md)
```

Each release directory holds the three bundle files the generator reads — `profiles-types.json` (datatypes),
`profiles-resources.json` (resources), and `valuesets.json` (value sets and code systems) — alongside a `SHA256SUMS`
manifest and a `SOURCE.md` recording provenance, license, and refresh instructions.

| Release | Version | Build | Resources | Source |
| --- | --- | --- | --- | --- |
| R4 | 4.0.1 | 9346c8cc45 (2019-11-01) | 148 | `r4/SOURCE.md` |
| R5 | 5.0.0 | 2aecd53 (2023-03-26) | 162 | `r5/SOURCE.md` |

## Integrity

Each `SHA256SUMS` records a SHA-256 per bundle file. The generator's loader (`fhir/internal/gen/loader`) verifies the
manifest before parsing and fails closed on any mismatch, a missing file, or an unpinned bundle. Verify a bundle on
the command line with:

```bash
shasum -a 256 -c r4/SHA256SUMS  # or r5/SHA256SUMS
```

## Refreshing

Refresh a bundle deliberately when bumping a pinned FHIR release, never at generate time. The refresh-only mise tasks
re-download from the canonical HL7 archive, verify the release version, and re-record the manifest:

```bash
mise run fhir:refresh-r4   # tools/fhir-definitions/refresh.sh r4
mise run fhir:refresh-r5   # tools/fhir-definitions/refresh.sh r5
```

After refreshing, update the matching `SOURCE.md` (version, build, date) and commit the bundle change separately from
any generator change.
