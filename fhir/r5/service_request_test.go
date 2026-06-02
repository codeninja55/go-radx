package r5_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

func TestServiceRequestResourceType(t *testing.T) {
	sr := &r5.ServiceRequest{}
	if got := sr.ResourceType(); got != "ServiceRequest" {
		t.Errorf("ResourceType() = %q, want ServiceRequest", got)
	}
	// It satisfies the release-agnostic fhir.Resource interface.
	var _ fhir.Resource = sr
}

func TestServiceRequestMarshalEmitsResourceTypeFirst(t *testing.T) {
	sr := &r5.ServiceRequest{
		Identifier: []r5.Identifier{{System: strptr("urn:placer"), Value: strptr("ORD-1")}},
		Status:     "active",
		Intent:     "order",
		Code:       &r5.CodeableReference{Concept: &r5.CodeableConcept{Text: strptr("CT Chest")}},
		Subject:    &r5.Reference{Type: strptr("Patient"), Identifier: &r5.Identifier{System: strptr("urn:mrn"), Value: strptr("123")}},
		AuthoredOn: strptr("2026-06-01T10:00:00Z"),
		Requester:  &r5.Reference{Display: strptr("Dr Smith")},
	}
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.HasPrefix(got, `{"resourceType":"ServiceRequest"`) {
		t.Errorf("ServiceRequest JSON does not start with resourceType: %s", got)
	}
	want := `{"resourceType":"ServiceRequest","identifier":[{"system":"urn:placer","value":"ORD-1"}],` +
		`"status":"active","intent":"order","code":{"concept":{"text":"CT Chest"}},` +
		`"subject":{"type":"Patient","identifier":{"system":"urn:mrn","value":"123"}},` +
		`"authoredOn":"2026-06-01T10:00:00Z","requester":{"display":"Dr Smith"}}`
	if got != want {
		t.Errorf("ServiceRequest JSON\n got = %s\nwant = %s", got, want)
	}
}

func TestServiceRequestMarshalOmitsEmpty(t *testing.T) {
	sr := &r5.ServiceRequest{Status: "active", Intent: "order"}
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"resourceType":"ServiceRequest","status":"active","intent":"order"}`
	if string(b) != want {
		t.Errorf("ServiceRequest JSON = %s, want %s", b, want)
	}
}
