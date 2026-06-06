// Package fhir is go-radx's type-safe implementation of HL7 FHIR R4 (4.0.1) and
// R5 (5.0.0). The resources and datatypes for each release live in their own
// release sub-package (fhir/r4 and fhir/r5) as distinct type spaces, so a consumer
// never mixes the two by accident; this root package holds only the
// release-agnostic machinery — the Resource interface, the checked
// Unmarshal[T]/As[T] identity API, the Decimal primitive that preserves lexical
// fidelity, the sentinel error types, and the Registry type that backs
// resourceType-dispatched decode, validation, and summary serialization. The
// Registry is release-scoped: each release package owns its own instance keyed by
// resourceType, because a FHIR JSON resource carries no release marker and so a
// single registry shared between R4 and R5 would be ambiguous and collide at init.
// It serializes JSON only in v1, enforces choice-type mutual exclusion at the
// serialization boundary, and round-trips the FHIR-JSON primitive-extension
// (_field) mechanic.
//
// See docs/reference/fhir.md for the public API and docs/conformance/fhir.md for
// the supported releases and resources.
//
// Stability: experimental. Pre-1.0; the API may change between v0.x releases.
package fhir

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Resource is the base unit of FHIR exchange. ResourceType returns the FHIR
// discriminator (for example "Patient"), which is a compile-time constant per
// type, not a mutable field.
type Resource interface {
	ResourceType() string
}

var (
	// ErrResourceTypeMismatch is returned by Unmarshal[T] when the payload's
	// resourceType does not match T.
	ErrResourceTypeMismatch = newSentinel("resourceType does not match target type")

	// ErrUnknownResourceType is returned by UnmarshalResource when resourceType
	// is absent or not in the registry.
	ErrUnknownResourceType = newSentinel("unknown resourceType")

	// ErrUnknownCode is returned by ParseXxx and by strict decode of a required
	// binding when a code is outside the bound value set.
	ErrUnknownCode = newSentinel("code not in required value set")
)

// Unmarshal decodes FHIR JSON into the concrete resource type T, verifying the
// payload's embedded "resourceType" matches T before fully decoding. A Patient
// payload decoded as *r5.Observation returns ErrResourceTypeMismatch and the zero
// value of T; it never silently succeeds against the wrong type. This is the
// checked-decode contract the prototype lacked: its UnmarshalResource[T] decoded
// the bytes into T without ever comparing the discriminator (Codex FHIR-003).
//
// A payload whose "resourceType" is absent or empty also fails, with
// ErrResourceTypeMismatch, because a resource with no discriminator cannot be
// asserted to be a T. The error names both discriminators (the payload's and T's),
// never any patient value.
func Unmarshal[T Resource](data []byte) (T, error) {
	var zero T

	payloadType, err := peekResourceType(data)
	if err != nil {
		return zero, err
	}

	// T is an interface constrained to Resource; its concrete dynamic type is a
	// pointer to a generated resource struct (for example *r5.Patient). Allocate a
	// fresh, decode-ready value of that concrete type so the discriminator can be
	// checked against a real ResourceType() before any field decode happens. An
	// instantiation that is not a pointer to a struct (the bare Resource interface, a
	// value type) cannot be allocated this way; newConcrete reports that as an error
	// rather than panicking, honouring the never-panic-on-misuse contract.
	target, err := newConcrete[T]()
	if err != nil {
		return zero, err
	}
	wantType := target.ResourceType()
	if payloadType != wantType {
		return zero, fmt.Errorf("fhir: %w: payload %q, want %q",
			ErrResourceTypeMismatch, payloadType, wantType)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return zero, fmt.Errorf("fhir: decode %s: %w", wantType, mapDecodeError(err))
	}
	return target, nil
}

// As is a checked downcast from the Resource interface to the concrete type T. It
// returns (value, true) when r's dynamic type is T and r is not a nil pointer, and
// (zero, false) otherwise, so a caller never panics at a polymorphic boundary
// (Bundle.entry.resource, contained) and never receives a non-nil ok alongside a nil
// pointer it would dereference. A nil interface and a typed-nil pointer both fail
// closed.
//
//	patient, ok := fhir.As[*r5.Patient](entry.Resource)
func As[T Resource](r Resource) (T, bool) {
	var zero T
	if r == nil || isNilResource(r) {
		return zero, false
	}
	t, ok := r.(T)
	return t, ok
}

// UnmarshalResource decodes FHIR JSON into the concrete type named by its
// "resourceType", dispatching through this registry's init-populated factory map. The
// returned Resource holds the dynamic type (for example *r5.Patient). A payload
// whose "resourceType" is absent, empty, or not registered returns
// ErrUnknownResourceType and a nil Resource; dispatch fails closed rather than
// guessing a type. Because the registry is release-scoped, the returned dynamic type
// is always of the release that owns this Registry.
func (reg *Registry) UnmarshalResource(data []byte) (Resource, error) {
	resourceType, err := peekResourceType(data)
	if err != nil {
		return nil, err
	}

	factory, ok := reg.lookupFactory(resourceType)
	if !ok {
		return nil, fmt.Errorf("fhir: %w: %q", ErrUnknownResourceType, resourceType)
	}

	r := factory()
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("fhir: decode %s: %w", resourceType, mapDecodeError(err))
	}
	return r, nil
}

// UnmarshalResourceSlice decodes a JSON array of FHIR resource objects, dispatching
// each element through this registry's UnmarshalResource. It backs the decode of a
// repeating resource-typed field (DomainResource.contained), where the standard codec
// cannot unmarshal a JSON object into the fhir.Resource interface. A JSON null yields
// a nil slice; any element whose resourceType is absent, empty, or unregistered fails
// the whole decode with ErrUnknownResourceType rather than skipping the element, so a
// partial slice is never returned. The element index is named in the error so a
// caller can locate the offending entry without exposing any element value.
func (reg *Registry) UnmarshalResourceSlice(data []byte) ([]Resource, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return nil, fmt.Errorf("fhir: decode resource array: %w", mapDecodeError(err))
	}
	if raws == nil {
		return nil, nil
	}
	out := make([]Resource, 0, len(raws))
	for i, raw := range raws {
		r, err := reg.UnmarshalResource(raw)
		if err != nil {
			return nil, fmt.Errorf("fhir: decode resource array element %d: %w", i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// UnmarshalResource decodes FHIR JSON through the root package's default registry. It
// is the release-agnostic counterpart to (*Registry).UnmarshalResource; a consumer
// that wants a specific release decodes through that release's r4.UnmarshalResource or
// r5.UnmarshalResource so the dynamic type is unambiguous.
func UnmarshalResource(data []byte) (Resource, error) {
	return defaultRegistry.UnmarshalResource(data)
}

// UnmarshalResourceSlice decodes a JSON array of FHIR resources through the root
// package's default registry. It is the release-agnostic counterpart to
// (*Registry).UnmarshalResourceSlice.
func UnmarshalResourceSlice(data []byte) ([]Resource, error) {
	return defaultRegistry.UnmarshalResourceSlice(data)
}

// discriminator is the minimal envelope used to peek a payload's "resourceType"
// before committing to a concrete decode, so the checked path reads only the one
// key it needs.
type discriminator struct {
	ResourceType string `json:"resourceType"`
}

// peekResourceType reads only the "resourceType" key from a FHIR JSON payload. An
// absent or empty discriminator is reported as ErrUnknownResourceType, matching the
// "reject a payload whose resourceType is absent, empty, or unknown" rule, so both
// the checked and the registry path treat a discriminator-less payload identically.
func peekResourceType(data []byte) (string, error) {
	var d discriminator
	if err := json.Unmarshal(data, &d); err != nil {
		return "", fmt.Errorf("fhir: read resourceType: %w", mapDecodeError(err))
	}
	if d.ResourceType == "" {
		return "", fmt.Errorf("fhir: %w: payload has no resourceType", ErrUnknownResourceType)
	}
	return d.ResourceType, nil
}

// newConcrete constructs a fresh, non-nil value of the concrete type behind
// interface T. Every generated resource type instantiates T as a pointer to a struct
// (*r5.Patient, ...); allocating the pointed-to struct yields a decode-ready value
// whose ResourceType() returns the type's constant discriminator. The one reflect
// allocation runs once per Unmarshal call, off any hot path, and keeps Unmarshal
// independent of the registry so a type need not be registered to be decoded by its
// static type.
//
// An instantiation whose concrete type is not a pointer (Unmarshal called with the
// bare Resource interface, or a value-receiver Resource) cannot be allocated into a
// decode target. newConcrete returns an error for that misuse rather than a nil or
// non-pointer value that would later panic or fail a json decode; no generated type
// triggers this path.
func newConcrete[T Resource]() (T, error) {
	var zero T
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Pointer {
		return zero, fmt.Errorf("fhir: Unmarshal target %s is not a pointer resource type; "+
			"instantiate Unmarshal with a concrete pointer type such as *r5.Patient", t)
	}
	return reflect.New(t.Elem()).Interface().(T), nil
}

// isNilResource reports whether r holds a typed-nil pointer, so a checked downcast of
// a Resource whose dynamic value is (for example) a nil *r5.Patient fails closed
// instead of handing back a nil the caller would dereference. A non-pointer dynamic
// value is never "nil" in this sense.
func isNilResource(r Resource) bool {
	v := reflect.ValueOf(r)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// errUnexpectedEndOfJSON is the message encoding/json attaches to the *json.SyntaxError
// it returns when a buffer ends before the JSON value it carries is closed — the
// signature of a truncated payload. It is matched verbatim (rather than by offset)
// because a trailing-garbage error ("invalid character … after top-level value") also
// sits at the end of the buffer yet is a structural fault, not truncation. This message
// has been stable across Go releases; mapping keys off it, with io.EOF/io.ErrUnexpectedEOF
// (the Decoder path) covered separately.
const errUnexpectedEndOfJSON = "unexpected end of JSON input"

// mapDecodeError normalises the error encoding/json returns for a payload that ends
// mid-value to the io.ErrUnexpectedEOF sentinel, so a truncated FHIR resource — a stream
// cut short, a partial network read — is matchable with
// errors.Is(err, io.ErrUnexpectedEOF) regardless of which decode entry point produced it.
// json.Unmarshal reports a buffer that runs out before the value closes as a
// *json.SyntaxError carrying errUnexpectedEndOfJSON; json.Decoder.Decode reports the same
// condition as io.ErrUnexpectedEOF (mid-value) or io.EOF (no value at all). All three are
// folded to io.ErrUnexpectedEOF here while the original error is preserved in the chain
// (wrapped with %w) so callers keep the stdlib diagnostic and gain the sentinel. A
// genuine syntax fault that is not mere truncation — a stray brace after a complete value,
// a bad token mid-buffer — is left untouched. The error carries no payload bytes, so no
// PHI leaks.
func mapDecodeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}
		return fmt.Errorf("%w: %w", io.ErrUnexpectedEOF, err)
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) && syntaxErr.Error() == errUnexpectedEndOfJSON {
		return fmt.Errorf("%w: %w", io.ErrUnexpectedEOF, err)
	}
	return err
}
