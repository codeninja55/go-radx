# go-radx documentation

`go-radx` is a Go library for medical imaging and healthcare interoperability standards: FHIR R4/R5, DICOM, DICOMweb,
HL7 v2.x, and DIMSE networking. The project is pre-1.0 and every public package is experimental; exported types,
signatures, and behaviour can change between any two `v0.x` releases.

This site collects the two document sets that define what the library does and how far you can rely on it.

## Conformance statements

The conformance statements are the versioned scope contracts. Each declares exactly which subset of a standard the
library supports and how that subset is verified, rather than claiming to implement the whole standard. A statement
marked **NOT YET SHIPPED** is a scaffold for a surface that is not yet implemented; do not cite it as a conformance
basis.

- [Cross-cutting posture](conformance/cross-cutting.md) — supply chain, interop determinism, build layout, coverage,
  concurrency, and the conformance-drift methodology that keeps these statements honest.
- [DICOM](conformance/dicom.md) — the data layer and DIMSE networking scope (authoritative for DIMSE today).
- [DIMSE](conformance/dimse.md) — the network-plane statement (scaffold).
- [DICOMweb](conformance/dicomweb.md) — WADO-RS, STOW-RS, and QIDO-RS scope (scaffold).
- [HL7 v2](conformance/hl7v2.md) — the messaging scope.
- [FHIR](conformance/fhir.md) — the FHIR R4/R5 resource scope.
- [Cross-standard conversion](conformance/convert.md) — the DICOM/HL7 v2/FHIR mapping scope (scaffold).
- [CLI and server](conformance/cli-server.md) — the operator-facing surface scope (scaffold).

## API reference

The reference docs describe the public API of each package — the types, functions, and entry points you call.

- [DICOM](reference/dicom.md)
- [DIMSE](reference/dimse.md)
- [DICOMweb](reference/dicomweb.md)
- [HL7 v2](reference/hl7v2.md)
- [FHIR](reference/fhir.md)
- [Cross-standard conversion](reference/convert.md)
- [CLI](reference/cli.md)
- [Servers](reference/servers.md)
