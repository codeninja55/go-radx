package r5_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestBundleEntryResourceDecodesConcreteType decodes a searchset Bundle whose entries
// carry two different resource types and asserts each entry.resource is the correct
// concrete type behind the fhir.Resource interface, recoverable with fhir.As[T]. This is
// the polymorphic decode the standard codec cannot perform: it has no concrete type to
// unmarshal a resource object into the interface, so the generated UnmarshalJSON routes
// the raw JSON through fhir.UnmarshalResource (resourceType peek then registry dispatch).
func TestBundleEntryResourceDecodesConcreteType(t *testing.T) {
	const payload = `{"resourceType":"Bundle","type":"searchset","entry":[` +
		`{"resource":{"resourceType":"Patient","id":"p1"}},` +
		`{"resource":{"resourceType":"Observation","id":"o1","status":"final","code":{"text":"heart rate"}}}` +
		`]}`

	var bundle r5.Bundle
	if err := json.Unmarshal([]byte(payload), &bundle); err != nil {
		t.Fatalf("decode searchset Bundle: %v", err)
	}
	if len(bundle.Entry) != 2 {
		t.Fatalf("decoded %d entries, want 2", len(bundle.Entry))
	}

	if bundle.Entry[0].Resource == nil {
		t.Fatal("entry[0].Resource is nil after decode")
	}
	patient, ok := fhir.As[*r5.Patient](*bundle.Entry[0].Resource)
	if !ok {
		t.Fatalf("entry[0] is %T, want *r5.Patient", *bundle.Entry[0].Resource)
	}
	if patient.ID == nil || *patient.ID != "p1" {
		t.Errorf("entry[0] Patient id = %v, want \"p1\"", patient.ID)
	}

	if bundle.Entry[1].Resource == nil {
		t.Fatal("entry[1].Resource is nil after decode")
	}
	observation, ok := fhir.As[*r5.Observation](*bundle.Entry[1].Resource)
	if !ok {
		t.Fatalf("entry[1] is %T, want *r5.Observation", *bundle.Entry[1].Resource)
	}
	if observation.ID == nil || *observation.ID != "o1" {
		t.Errorf("entry[1] Observation id = %v, want \"o1\"", observation.ID)
	}
}

// TestBundleEntryResourceRoundTrips decodes a multi-resource-type Bundle, re-encodes it,
// decodes the re-encoded bytes, and asserts the typed value is unchanged — the
// decode -> marshal -> decode invariant for a polymorphic entry.resource field.
func TestBundleEntryResourceRoundTrips(t *testing.T) {
	const payload = `{"resourceType":"Bundle","type":"searchset","entry":[` +
		`{"resource":{"resourceType":"Patient","id":"p1"}},` +
		`{"resource":{"resourceType":"Observation","id":"o1","status":"final","code":{"text":"heart rate"}}}` +
		`]}`

	var first r5.Bundle
	if err := json.Unmarshal([]byte(payload), &first); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	reencoded, err := json.Marshal(&first)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	var second r5.Bundle
	if err := json.Unmarshal(reencoded, &second); err != nil {
		t.Fatalf("second decode: %v", err)
	}
	if !reflect.DeepEqual(&first, &second) {
		t.Errorf("Bundle did not round-trip structurally:\n first  %#v\n second %#v", first, second)
	}
}

// TestContainedDecodesConcreteType decodes a resource carrying a contained resource and
// asserts the contained slot holds the correct concrete type. The contained field lives
// in the embedded DomainResource base; the embedding resource's generated UnmarshalJSON
// is where the polymorphic decode runs, because a base type defines no UnmarshalJSON.
func TestContainedDecodesConcreteType(t *testing.T) {
	const payload = `{"resourceType":"Patient","id":"p1","contained":[` +
		`{"resourceType":"Organization","id":"org1"},` +
		`{"resourceType":"Practitioner","id":"prac1"}` +
		`]}`

	var patient r5.Patient
	if err := json.Unmarshal([]byte(payload), &patient); err != nil {
		t.Fatalf("decode Patient with contained: %v", err)
	}
	if len(patient.Contained) != 2 {
		t.Fatalf("decoded %d contained, want 2", len(patient.Contained))
	}

	org, ok := fhir.As[*r5.Organization](patient.Contained[0])
	if !ok {
		t.Fatalf("contained[0] is %T, want *r5.Organization", patient.Contained[0])
	}
	if org.ID == nil || *org.ID != "org1" {
		t.Errorf("contained[0] Organization id = %v, want \"org1\"", org.ID)
	}

	prac, ok := fhir.As[*r5.Practitioner](patient.Contained[1])
	if !ok {
		t.Fatalf("contained[1] is %T, want *r5.Practitioner", patient.Contained[1])
	}
	if prac.ID == nil || *prac.ID != "prac1" {
		t.Errorf("contained[1] Practitioner id = %v, want \"prac1\"", prac.ID)
	}
}

// TestContainedRoundTrips decodes a resource with contained resources, re-encodes, and
// re-decodes, asserting the typed value is unchanged across the round trip.
func TestContainedRoundTrips(t *testing.T) {
	const payload = `{"resourceType":"Patient","id":"p1","contained":[` +
		`{"resourceType":"Organization","id":"org1"}` +
		`]}`

	var first r5.Patient
	if err := json.Unmarshal([]byte(payload), &first); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	reencoded, err := json.Marshal(&first)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	var second r5.Patient
	if err := json.Unmarshal(reencoded, &second); err != nil {
		t.Fatalf("second decode: %v", err)
	}
	if !reflect.DeepEqual(&first, &second) {
		t.Errorf("Patient with contained did not round-trip structurally")
	}
}

// TestPolymorphicDecodeRejectsUnknownResourceType asserts a resource-typed field whose
// element carries an unregistered resourceType fails the decode with a clear,
// errors.Is-matchable ErrUnknownResourceType rather than panicking or silently dropping
// the element. Both the single (entry.resource) and repeating (contained) shapes are
// covered.
func TestPolymorphicDecodeRejectsUnknownResourceType(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		decode  func([]byte) error
	}{
		{
			name:    "contained unknown type",
			payload: `{"resourceType":"Patient","contained":[{"resourceType":"NotAResource"}]}`,
			decode:  func(b []byte) error { var p r5.Patient; return json.Unmarshal(b, &p) },
		},
		{
			name: "entry.resource unknown type",
			payload: `{"resourceType":"Bundle","type":"collection","entry":[` +
				`{"resource":{"resourceType":"NotAResource"}}]}`,
			decode: func(b []byte) error { var bundle r5.Bundle; return json.Unmarshal(b, &bundle) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.decode([]byte(tc.payload))
			if err == nil {
				t.Fatal("decode of unknown resourceType succeeded, want error")
			}
			if !errors.Is(err, fhir.ErrUnknownResourceType) {
				t.Errorf("error %v does not match ErrUnknownResourceType", err)
			}
		})
	}
}

// TestPolymorphicDecodeRejectsAbsentResourceType asserts a resource-typed field whose
// element omits resourceType fails the decode with ErrUnknownResourceType (a resource
// with no discriminator cannot be dispatched), not a panic.
func TestPolymorphicDecodeRejectsAbsentResourceType(t *testing.T) {
	const payload = `{"resourceType":"Bundle","type":"collection","entry":[{"resource":{"id":"x"}}]}`
	var bundle r5.Bundle
	err := json.Unmarshal([]byte(payload), &bundle)
	if err == nil {
		t.Fatal("decode of entry.resource with no resourceType succeeded, want error")
	}
	if !errors.Is(err, fhir.ErrUnknownResourceType) {
		t.Errorf("error %v does not match ErrUnknownResourceType", err)
	}
}

// TestContainedSliceDecodeIsAllOrNothing asserts UnmarshalResourceSlice never returns a
// partial contained slice: a valid first element followed by an unregistered second one
// fails the whole decode, so a caller never sees a silently-truncated slice.
func TestContainedSliceDecodeIsAllOrNothing(t *testing.T) {
	const payload = `{"resourceType":"Patient","contained":[` +
		`{"resourceType":"Organization","id":"org1"},` +
		`{"resourceType":"NotAResource"}` +
		`]}`
	var patient r5.Patient
	err := json.Unmarshal([]byte(payload), &patient)
	if err == nil {
		t.Fatal("decode succeeded with a partial contained slice, want error")
	}
	if !errors.Is(err, fhir.ErrUnknownResourceType) {
		t.Errorf("error %v does not match ErrUnknownResourceType", err)
	}
	if patient.Contained != nil {
		t.Errorf("contained = %v, want nil (no partial slice on failure)", patient.Contained)
	}
}
