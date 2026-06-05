package fhir

import (
	"encoding/json"
	"errors"
	"io"
	"testing"
)

// fakePatient and fakeObservation are minimal in-package Resource implementations.
// They exercise the release-agnostic identity machinery without importing a release
// package (which would create an import cycle: the release packages import the root
// fhir package). Their ResourceType methods return constants and do not read the
// receiver, mirroring the generated resources.
type fakePatient struct {
	ID string `json:"id,omitempty"`
}

func (*fakePatient) ResourceType() string { return "Patient" }

func (p *fakePatient) MarshalJSON() ([]byte, error) {
	type alias fakePatient
	return json.Marshal(struct {
		ResourceType string `json:"resourceType"`
		*alias
	}{ResourceType: "Patient", alias: (*alias)(p)})
}

type fakeObservation struct {
	ID string `json:"id,omitempty"`
}

func (*fakeObservation) ResourceType() string { return "Observation" }

// TestUnmarshalChecksResourceType is the FHIR-003 regression: Unmarshal[T] must
// compare the payload's resourceType against T and reject a mismatch instead of
// silently decoding the bytes into the wrong type, which is exactly what the
// prototype's UnmarshalResource[T] did.
func TestUnmarshalChecksResourceType(t *testing.T) {
	payload := []byte(`{"resourceType":"Patient","id":"pat-1"}`)

	got, err := Unmarshal[*fakePatient](payload)
	if err != nil {
		t.Fatalf("Unmarshal[*fakePatient] of a Patient payload: %v", err)
	}
	if got == nil || got.ID != "pat-1" {
		t.Fatalf("decoded patient = %+v, want id pat-1", got)
	}

	mismatch, err := Unmarshal[*fakeObservation](payload)
	if !errors.Is(err, ErrResourceTypeMismatch) {
		t.Fatalf("Unmarshal[*fakeObservation] of a Patient payload: err = %v, want ErrResourceTypeMismatch", err)
	}
	if mismatch != nil {
		t.Errorf("a type-mismatch decode returned %+v, want the zero value (nil)", mismatch)
	}
}

// TestUnmarshalRejectsMissingResourceType asserts a payload with no discriminator
// fails closed: there is no resourceType to assert T against, so the decode errors
// rather than producing a half-typed value.
func TestUnmarshalRejectsMissingResourceType(t *testing.T) {
	for _, payload := range []string{`{"id":"x"}`, `{"resourceType":""}`} {
		_, err := Unmarshal[*fakePatient]([]byte(payload))
		if !errors.Is(err, ErrUnknownResourceType) {
			t.Errorf("Unmarshal of %q: err = %v, want ErrUnknownResourceType", payload, err)
		}
	}
}

// TestUnmarshalReportsInvalidJSON asserts malformed JSON surfaces a decode error
// (a Go error, per the behaviour-and-error-model contract) rather than panicking.
func TestUnmarshalReportsInvalidJSON(t *testing.T) {
	if _, err := Unmarshal[*fakePatient]([]byte(`{not json`)); err == nil {
		t.Error("Unmarshal of malformed JSON should return an error")
	}
}

// TestUnmarshalTruncatedYieldsUnexpectedEOF is the truncation contract: a valid FHIR
// payload cut short at any byte must surface io.ErrUnexpectedEOF (matchable with
// errors.Is), not the stdlib's opaque "unexpected end of JSON input" syntax error nor a
// panic. A reader handling a partial network read or a clipped stream can then
// distinguish "the input ran out" from "the input is structurally wrong". Every decode
// entry point is checked because each calls a different json boundary (peekResourceType
// reads only the discriminator; the full decode reads the whole object). A genuinely
// malformed payload (a stray brace mid-buffer) must NOT be mapped to truncation, so the
// non-truncation cases are asserted to be plain errors.
func TestUnmarshalTruncatedYieldsUnexpectedEOF(t *testing.T) {
	withTestFactory(t, "Patient", func() Resource { return &fakePatient{} })

	full := `{"resourceType":"Patient","id":"pat-trunc","extra":{"a":[1,2,3]}}`
	// Truncate at a range of lengths so both the discriminator peek and the full-object
	// decode see a clipped buffer. len(full)-1 clips inside the closing braces; a short
	// prefix clips inside the resourceType key itself.
	for _, n := range []int{1, 5, 10, len(full) - 20, len(full) - 1} {
		truncated := []byte(full[:n])

		if _, err := UnmarshalResource(truncated); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("UnmarshalResource(truncated len=%d): err = %v, want io.ErrUnexpectedEOF", n, err)
		}
		if _, err := Unmarshal[*fakePatient](truncated); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("Unmarshal[*fakePatient](truncated len=%d): err = %v, want io.ErrUnexpectedEOF", n, err)
		}
	}

	// Empty input is the degenerate truncation: the value never began.
	if _, err := UnmarshalResource([]byte{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("UnmarshalResource(empty): err = %v, want io.ErrUnexpectedEOF", err)
	}

	// A truncated resource array (the contained-decode path) maps the same way.
	if _, err := UnmarshalResourceSlice([]byte(`[{"resourceType":"Patient"`)); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("UnmarshalResourceSlice(truncated): err = %v, want io.ErrUnexpectedEOF", err)
	}

	// A structurally malformed payload that is not merely cut short must stay a plain
	// decode error: a trailing stray character after a complete value is a syntax error
	// at an offset inside the buffer, never truncation.
	if _, err := UnmarshalResource([]byte(`{"resourceType":"Patient"}}`)); errors.Is(err, io.ErrUnexpectedEOF) {
		t.Error("a trailing-brace syntax error was misreported as truncation (io.ErrUnexpectedEOF)")
	}
}

// TestAsCheckedDowncast covers the As[T] checked downcast: a matching dynamic type
// succeeds, a non-matching one returns (zero, false), and a nil Resource returns
// (zero, false) instead of panicking.
func TestAsCheckedDowncast(t *testing.T) {
	var r Resource = &fakePatient{ID: "pat-1"}

	p, ok := As[*fakePatient](r)
	if !ok || p == nil || p.ID != "pat-1" {
		t.Fatalf("As[*fakePatient] = (%+v, %v), want the patient and true", p, ok)
	}

	if _, ok := As[*fakeObservation](r); ok {
		t.Error("As[*fakeObservation] on a Patient should return false")
	}

	if _, ok := As[*fakePatient](nil); ok {
		t.Error("As[*fakePatient](nil) should return false, not panic")
	}

	// A typed-nil pointer carried in the interface must fail closed: handing back a
	// nil pointer with ok=true is a dereference footgun.
	var typedNil *fakePatient
	var carrier Resource = typedNil
	if _, ok := As[*fakePatient](carrier); ok {
		t.Error("As of a typed-nil *fakePatient should return false, not (nil, true)")
	}
}

// TestUnmarshalNonPointerTargetErrors asserts Unmarshal instantiated with a
// non-pointer Resource (the bare interface) returns an error instead of panicking,
// honouring the never-panic-on-misuse contract.
func TestUnmarshalNonPointerTargetErrors(t *testing.T) {
	if _, err := Unmarshal[Resource]([]byte(`{"resourceType":"Patient"}`)); err == nil {
		t.Error("Unmarshal[Resource] (a non-pointer target) should return an error, not panic")
	}
}

// TestUnmarshalResourceDispatch covers the registry path: a registered resourceType
// dispatches to its factory and decodes into the concrete type, while an unknown or
// absent resourceType returns ErrUnknownResourceType and a nil Resource.
func TestUnmarshalResourceDispatch(t *testing.T) {
	withTestFactory(t, "Patient", func() Resource { return &fakePatient{} })

	r, err := UnmarshalResource([]byte(`{"resourceType":"Patient","id":"pat-9"}`))
	if err != nil {
		t.Fatalf("UnmarshalResource(Patient): %v", err)
	}
	p, ok := r.(*fakePatient)
	if !ok {
		t.Fatalf("UnmarshalResource returned %T, want *fakePatient", r)
	}
	if p.ID != "pat-9" {
		t.Errorf("decoded id = %q, want pat-9", p.ID)
	}

	if _, err := UnmarshalResource([]byte(`{"resourceType":"Nonexistent"}`)); !errors.Is(err, ErrUnknownResourceType) {
		t.Errorf("UnmarshalResource of an unknown type: err = %v, want ErrUnknownResourceType", err)
	}
	if _, err := UnmarshalResource([]byte(`{"id":"x"}`)); !errors.Is(err, ErrUnknownResourceType) {
		t.Errorf("UnmarshalResource of a discriminator-less payload: err = %v, want ErrUnknownResourceType", err)
	}
}

// TestUnmarshalResourceSliceDispatch covers the repeating-resource decode path that
// backs DomainResource.contained: each array element dispatches through its factory to
// the concrete type, a JSON null yields a nil slice, and an element whose resourceType
// is unregistered fails the whole decode (never a partial slice) with a clear,
// errors.Is-matchable ErrUnknownResourceType.
func TestUnmarshalResourceSliceDispatch(t *testing.T) {
	withTestFactory(t, "Patient", func() Resource { return &fakePatient{} })

	got, err := UnmarshalResourceSlice([]byte(`[{"resourceType":"Patient","id":"a"},{"resourceType":"Patient","id":"b"}]`))
	if err != nil {
		t.Fatalf("UnmarshalResourceSlice: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("decoded %d resources, want 2", len(got))
	}
	for i, want := range []string{"a", "b"} {
		p, ok := got[i].(*fakePatient)
		if !ok {
			t.Fatalf("element %d is %T, want *fakePatient", i, got[i])
		}
		if p.ID != want {
			t.Errorf("element %d id = %q, want %q", i, p.ID, want)
		}
	}

	nilSlice, err := UnmarshalResourceSlice([]byte(`null`))
	if err != nil {
		t.Fatalf("UnmarshalResourceSlice(null): %v", err)
	}
	if nilSlice != nil {
		t.Errorf("UnmarshalResourceSlice(null) = %v, want nil", nilSlice)
	}

	partial, err := UnmarshalResourceSlice([]byte(`[{"resourceType":"Patient"},{"resourceType":"Nonexistent"}]`))
	if !errors.Is(err, ErrUnknownResourceType) {
		t.Errorf("UnmarshalResourceSlice with an unknown element: err = %v, want ErrUnknownResourceType", err)
	}
	if partial != nil {
		t.Errorf("a failed slice decode returned %v, want nil (no partial slice)", partial)
	}
}

// TestRegisterFactoryRejectsDuplicate asserts the registry fails loudly on a
// duplicate registration, the build-time defect a release generator would commit if
// it emitted two factories for one resourceType.
func TestRegisterFactoryRejectsDuplicate(t *testing.T) {
	withTestFactory(t, "Dup", func() Resource { return &fakePatient{} })

	defer func() {
		if recover() == nil {
			t.Error("RegisterFactory should panic on a duplicate resourceType")
		}
	}()
	RegisterFactory("Dup", func() Resource { return &fakePatient{} })
}

// withTestFactory registers a factory for the duration of one test and removes it
// afterwards, so the registry is left untouched for other tests. Tests are the one
// place that exercises registration directly; they clean up after themselves. The
// direct map reads and the delete take the registry lock so the helper never races
// the package's own readers.
func withTestFactory(t *testing.T, resourceType string, f Factory) {
	t.Helper()
	if _, ok := lookupFactory(resourceType); ok {
		t.Fatalf("registry already has %q; the test registry was not cleaned up", resourceType)
	}
	RegisterFactory(resourceType, f)
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, resourceType)
		registryMu.Unlock()
	})
}
