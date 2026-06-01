// Package r5 holds the HL7 FHIR R5 (5.0.0) resources, backbone elements, and
// datatypes as type-safe Go structs (for example r5.Patient, r5.ImagingStudy,
// r5.ServiceRequest, r5.DiagnosticReport, r5.Reference, r5.Identifier). It is one
// of two release sub-packages under fhir, deliberately distinct from fhir/r4 so
// the two release type spaces never mix; the release-agnostic machinery (the
// Resource interface, Unmarshal[T]/As[T], Decimal, and the sentinel errors) lives
// in the root fhir package. The R5 FHIR Element base component is generated here
// as r5.Element and is never conflated with a DICOM data element, and the FHIR
// Reference datatype is never used for a DICOM referenced-SOP UID pair.
//
// See docs/reference/fhir.md for the public API.
package r5
