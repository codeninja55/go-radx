// Package convert bridges the three standards go-radx implements — DICOM (NEMA
// PS3), HL7 v2.x, and HL7 FHIR (R4 4.0.1 and R5 5.0.0). It exists because the same
// clinical concept is modelled differently in each standard, and silently aliasing
// those models is a primary source of conversion bugs. Every converter keeps each
// standard's own nouns and bridges between them explicitly, following the
// convention convert.<Source>To<Target><Release> (for example DICOMToImagingStudyR5,
// ORMToServiceRequestR5, SRToDiagnosticReportR5), where the FHIR release suffix is
// part of the function name because the R4 and R5 resource models live in the
// distinct fhir/r4 and fhir/r5 sub-packages. The DICOM side stays UID-keyed
// *dicom.DataSet access, the FHIR side is a generated release-typed resource, and
// the HL7 side is a typed segment or message; there is no shared cross-standard
// type.
//
// See docs/reference/convert.md and the cross-standard table in
// UBIQUITOUS_LANGUAGE.md.
//
// Stability: experimental. Pre-1.0; the API may change between v0.x releases.
package convert
