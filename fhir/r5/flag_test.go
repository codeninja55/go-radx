package r5_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestFlagMarshalAlwaysEmitsResourceType is the FHIR-004 regression against a real
// generated resource: a zero-value Flag must serialise "resourceType":"Flag", never
// the empty string the prototype emitted.
func TestFlagMarshalAlwaysEmitsResourceType(t *testing.T) {
	data, err := json.Marshal(&r5.Flag{})
	if err != nil {
		t.Fatalf("marshal zero-value Flag: %v", err)
	}
	if got := string(data); !strings.Contains(got, `"resourceType":"Flag"`) {
		t.Errorf("zero-value Flag marshalled to %s, want it to carry \"resourceType\":\"Flag\"", got)
	}
	if strings.Contains(string(data), `"resourceType":""`) {
		t.Error("Flag marshalled an empty resourceType (FHIR-004 regression)")
	}
}

// TestFlagUnmarshalCheckedIdentity exercises the identity API end to end against the
// generated Flag resource: the checked decode succeeds for a Flag payload and fails
// with ErrResourceTypeMismatch for a non-Flag payload (FHIR-003), so the registry and
// generic functions are proven against generated code, not only in-package fakes.
func TestFlagUnmarshalCheckedIdentity(t *testing.T) {
	payload := []byte(`{"resourceType":"Flag","status":"active"}`)

	flag, err := fhir.Unmarshal[*r5.Flag](payload)
	if err != nil {
		t.Fatalf("Unmarshal[*r5.Flag] of a Flag payload: %v", err)
	}
	if flag == nil || flag.Status == nil || *flag.Status != "active" {
		t.Fatalf("decoded Flag = %+v, want status active", flag)
	}

	wrong := []byte(`{"resourceType":"Observation"}`)
	if _, err := fhir.Unmarshal[*r5.Flag](wrong); !errors.Is(err, fhir.ErrResourceTypeMismatch) {
		t.Errorf("Unmarshal[*r5.Flag] of an Observation payload: err = %v, want ErrResourceTypeMismatch", err)
	}
}

// TestFlagRegistryDispatch proves the generated registry init() registered Flag so
// fhir.UnmarshalResource dispatches a "Flag" payload to the concrete *r5.Flag type,
// and that As[T] downcasts the returned interface.
func TestFlagRegistryDispatch(t *testing.T) {
	r, err := fhir.UnmarshalResource([]byte(`{"resourceType":"Flag","status":"inactive"}`))
	if err != nil {
		t.Fatalf("UnmarshalResource(Flag): %v", err)
	}
	if r.ResourceType() != "Flag" {
		t.Fatalf("dispatched resource type = %q, want Flag", r.ResourceType())
	}

	flag, ok := fhir.As[*r5.Flag](r)
	if !ok {
		t.Fatalf("As[*r5.Flag] on the dispatched resource returned false")
	}
	if flag.Status == nil || *flag.Status != "inactive" {
		t.Errorf("downcast Flag status = %v, want inactive", flag.Status)
	}
}

// TestFlagRoundTrip asserts a Flag survives a marshal/unmarshal cycle through the
// checked identity API with its fields intact.
func TestFlagRoundTrip(t *testing.T) {
	status := r5.FlagStatus("active")
	original := &r5.Flag{Status: &status}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal Flag: %v", err)
	}
	decoded, err := fhir.Unmarshal[*r5.Flag](data)
	if err != nil {
		t.Fatalf("round-trip Unmarshal[*r5.Flag]: %v", err)
	}
	if decoded.Status == nil || *decoded.Status != status {
		t.Errorf("round-tripped status = %v, want %q", decoded.Status, status)
	}
}
