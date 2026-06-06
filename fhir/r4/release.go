// This file is HAND-WRITTEN, not generated. It owns the R4 release's own
// fhir.Registry instance and the release-scoped entry points that read it. The
// generated registry.go and validation_descriptors.go register every R4 resource's
// factory and descriptor into Registry below at package init; this file exposes the
// decode and validate functions a consumer calls.
//
// A FHIR JSON resource carries a "resourceType" but not its release, so a single
// resourceType-keyed registry shared between R4 and R5 is inherently ambiguous and
// collides at init when both release packages are imported. Each release owning its
// own Registry resolves that: r4.UnmarshalResource always returns R4 types, and an R4
// Bundle's generated decode dispatches its contained resources through Registry, so a
// dual-release build (importing both fhir/r4 and fhir/r5) no longer panics.

package r4

import "github.com/codeninja55/go-radx/fhir"

// Registry is the R4 release's resourceType-keyed factory, validation, and summary
// metadata. The generated init() functions in registry.go and
// validation_descriptors.go are its only writers, populating it before main runs; it
// is read-only in practice thereafter. It is exported so the generated registration
// code in this package can reach it across files and so a consumer that wants an
// explicit release-scoped handle can hold one.
var Registry = fhir.NewRegistry()

// UnmarshalResource decodes FHIR JSON into the concrete R4 resource named by its
// "resourceType". The returned Resource holds an R4 dynamic type (for example
// *r4.Patient). A payload whose "resourceType" is absent, empty, or not an R4 resource
// returns fhir.ErrUnknownResourceType and a nil Resource; dispatch fails closed rather
// than guessing a type.
func UnmarshalResource(data []byte) (fhir.Resource, error) {
	return Registry.UnmarshalResource(data)
}

// UnmarshalResourceSlice decodes a JSON array of FHIR resource objects into R4
// resources, dispatching each element through UnmarshalResource. It backs the decode of
// a repeating resource-typed R4 field (DomainResource.contained). A JSON null yields a
// nil slice; any element whose resourceType is absent, empty, or not an R4 resource
// fails the whole decode with fhir.ErrUnknownResourceType.
func UnmarshalResourceSlice(data []byte) ([]fhir.Resource, error) {
	return Registry.UnmarshalResourceSlice(data)
}

// Unmarshal decodes FHIR JSON into the concrete R4 resource type T, verifying the
// payload's "resourceType" matches T before decoding. It is fhir.Unmarshal pinned to
// the R4 release for symmetry with the other release-scoped entry points; because
// fhir.Unmarshal resolves T by its static type rather than the registry, it does not
// depend on R4 registration.
func Unmarshal[T fhir.Resource](data []byte) (T, error) {
	return fhir.Unmarshal[T](data)
}

// Validate runs go-radx's structural and binding-level validation over an R4 resource
// and returns the issues as an *fhir.OperationOutcome, consuming the R4 validation
// descriptors registered in Registry. A resource whose type has no registered R4
// descriptor is reported as a single unvalidated-type issue rather than silently
// passing.
func Validate(r fhir.Resource) *fhir.OperationOutcome {
	return Registry.Validate(r)
}

// MarshalSummary serializes an R4 resource under the given summary mode, consuming the
// R4 summary descriptors registered in Registry. A resource whose type has no
// registered R4 summary descriptor is returned in full rather than filtered.
func MarshalSummary(r fhir.Resource, mode fhir.SummaryMode) ([]byte, error) {
	return Registry.MarshalSummary(r, mode)
}
