package fhir

import (
	"encoding/json"
	"errors"
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
