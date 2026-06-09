// Package fhir is the release-neutral core of go-radx's HL7 FHIR support. It holds
// what FHIR R4 (4.0.1) and R5 (5.0.0) have in common: the Resource interface and
// the release Registry that the generated resource trees in the fhir/r4 and fhir/r5
// sub-packages register into, the FHIR primitive types and their parallel _element
// extension arrays, code and binding parsing, the Decimal lexical type (a twin of
// dicom.Decimal), and OperationOutcome. The release-typed resources themselves live
// in fhir/r4 and fhir/r5, not here: a FHIR client or server role is scoped to one
// fixed release, so this package is the machinery both releases share rather than a
// third, release-blurring resource model.
//
// See docs/reference/fhir.md and the conformance statement in
// docs/conformance/fhir.md.
//
// Stability: experimental. Pre-1.0; the API may change between v0.x releases.
package fhir
