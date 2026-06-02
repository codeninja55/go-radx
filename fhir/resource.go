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

import "errors"

// Resource is the base unit of FHIR exchange. ResourceType returns the FHIR
// discriminator (for example "Patient"), which is a compile-time constant per
// type, not a mutable field.
type Resource interface {
	ResourceType() string
}

var (
	// ErrResourceTypeMismatch is returned by Unmarshal[T] when the payload's
	// resourceType does not match T.
	ErrResourceTypeMismatch = errors.New("fhir: resourceType does not match target type")

	// ErrUnknownResourceType is returned by UnmarshalResource when resourceType
	// is absent or not in the registry.
	ErrUnknownResourceType = errors.New("fhir: unknown resourceType")

	// ErrUnknownCode is returned by ParseXxx and by strict decode of a required
	// binding when a code is outside the bound value set.
	ErrUnknownCode = errors.New("fhir: code not in required value set")
)
