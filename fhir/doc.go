// Package fhir is go-radx's type-safe implementation of HL7 FHIR R4 (4.0.1) and
// R5 (5.0.0). The resources and datatypes for each release live in their own
// release sub-package (fhir/r4 and fhir/r5) as distinct type spaces, so a consumer
// never mixes the two by accident; this root package holds only the
// release-agnostic machinery — the Resource interface, the checked
// Unmarshal[T]/As[T] identity API, the Decimal primitive that preserves lexical
// fidelity, the sentinel error types, and the resourceType→factory registry. It
// serializes JSON only in v1, enforces choice-type mutual exclusion at the
// serialization boundary, and round-trips the FHIR-JSON primitive-extension
// (_field) mechanic.
//
// See docs/reference/fhir.md for the public API and docs/conformance/fhir.md for
// the supported releases and resources.
package fhir
