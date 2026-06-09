# go-radx

`go-radx` is a Go library for medical imaging and healthcare interoperability standards. It is being
built toward type-safe, idiomatic implementations of DICOM (NEMA PS3), the DIMSE networking protocol,
DICOMweb (WADO-RS, STOW-RS, QIDO-RS), HL7 v2.x, and HL7 FHIR (R4 and R5), plus a cross-standard
conversion layer and embeddable server roles for the receiving side of a radiology workflow. The
implemented subset is still partial; the conformance statements under `docs/conformance/` declare the
versioned subset each package targets.

```bash
go get github.com/codeninja55/go-radx@main
```

The published semantic-version tags (`v0.10.x` and earlier) belong to the legacy codebase on the
`legacy-main` branch. Until the re-foundation is tagged, depend on `@main`.

## Quick start

Read a DICOM Part 10 file and look up elements by tag:

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
		log.Fatal(err)
	}

	if name, ok := f.DataSet.GetString(dicom.TagPatientName); ok {
		fmt.Println("Patient name:", name)
	}
	if modality, ok := f.DataSet.GetString(dicom.TagModality); ok {
		fmt.Println("Modality:", modality)
	}
}
```

The `radx` CLI wraps the same library for the common workflows — for example `radx dump study.dcm`
to inspect a file, or `radx echo PACS_HOST 11112 --called-ae ARCHIVE` to verify DIMSE connectivity.
See [`docs/reference/`](docs/reference/) for the per-package API and [`docs/reference/cli.md`](docs/reference/cli.md)
for the full command tree.

## Stability: pre-1.0, everything experimental

> **go-radx is pre-1.0. Every public package is experimental and the entire API surface is
> unstable.** Exported types, function signatures, and behaviour can change between any two `v0.x`
> releases without a deprecation cycle. Pin an exact version, and expect to adjust call sites when you
> upgrade. Do not depend on go-radx in production-critical paths until it reaches `v1.0.0`.

The reference docs under `docs/reference/` describe the API contract each package is being built
*toward* (the planned v1 surface); they do not imply that surface is frozen today. Each top-level
public package (`convert`, `dicom`, `dicomweb`, `dimse`, `fhir`, `hl7v2`, and `server`) carries a
one-line stability marker in its godoc — `experimental` while the API is still moving, or
`stabilising` once it is settling toward v1.

Earlier `v0.x` tags belong to the legacy codebase on the `legacy-main` branch and are not continued
here; the current history begins with the re-foundation. See `CHANGELOG.md`.

## Packages

| Package     | Scope                                                                       |
| ----------- | --------------------------------------------------------------------------- |
| `dicom`     | DICOM data model (NEMA PS3) and the Part 10 file format.                     |
| `dimse`     | DIMSE-C / DIMSE-N services and the DICOM Upper Layer transport (PS3.7/PS3.8).|
| `dicomweb`  | RESTful DICOM services — WADO-RS, STOW-RS, QIDO-RS (PS3.18).                 |
| `hl7v2`     | HL7 v2.x parse tree, typed segments and messages, and MLLP transport.       |
| `fhir`      | HL7 FHIR R4 (4.0.1) and R5 (5.0.0) release-typed resources, by generated sub-package. |
| `convert`   | Cross-standard conversion between DICOM, HL7 v2.x, and FHIR.                 |
| `server`    | Composition layer wiring the receiving-side server roles for a deployment.  |

## Documentation

- Reference: `docs/reference/`
- Conformance (supported subsets): `docs/conformance/`
- Ubiquitous language and the cross-standard model: `UBIQUITOUS_LANGUAGE.md`

## Contributing

See `CONTRIBUTING.md` for the development setup and local gates, `CODE_OF_CONDUCT.md` for community
expectations, and `SECURITY.md` for how to report a vulnerability.

## Health data

This library processes Protected Health Information (PHI). It never logs PHI by default. Never put
real PHI in code, tests, fixtures, or logs — use clearly synthetic sentinel values.

## License

MIT. See `LICENSE`.
