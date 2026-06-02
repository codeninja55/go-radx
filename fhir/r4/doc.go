// Package r4 holds the HL7 FHIR R4 (4.0.1) resources, backbone elements, and
// datatypes as type-safe Go structs (for example r4.Patient, r4.ImagingStudy,
// r4.ServiceRequest, r4.DiagnosticReport, r4.Reference, r4.Identifier). It is one
// of two release sub-packages under fhir, deliberately distinct from fhir/r5 so
// the two release type spaces never mix; the release-agnostic machinery (the
// Resource interface, Unmarshal[T]/As[T], Decimal, and the sentinel errors) lives
// in the root fhir package. The R4 FHIR Element base component is generated here
// as r4.Element and is never conflated with a DICOM data element, and the FHIR
// Reference datatype is never used for a DICOM referenced-SOP UID pair.
//
// This package is produced by the build-time generator in fhir/internal/gen and
// is never hand-edited; `go generate ./fhir/...` reproduces it byte-for-byte from
// the pinned StructureDefinition bundle. R4 generation lands in milestone M6b;
// until then this package is an empty shell.
//
// See docs/reference/fhir.md for the public API.
package r4
