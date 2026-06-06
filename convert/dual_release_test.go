package convert_test

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestDualReleaseImportNoInitPanic is the regression for the dual-release init panic:
// importing both fhir/r4 and fhir/r5 must not collide at package init. Before the
// registries were release-scoped, both release packages registered the same
// resourceType keys ("Account", "Patient", ...) into one shared root-package registry,
// and RegisterFactory panics on a duplicate, so any binary importing both releases
// crashed during init before main. The mere fact that this test package compiles and
// its init runs proves the collision is gone; the body asserts each release's
// release-scoped entry points dispatch to that release's own type space.
//
// The pre-fix code fails this test by panicking at init ("RegisterFactory: duplicate
// factory for resourceType Account"), before any test body runs.
func TestDualReleaseImportNoInitPanic(t *testing.T) {
	const patientJSON = `{"resourceType":"Patient","id":"dual-1"}`

	r4Patient, err := r4.UnmarshalResource([]byte(patientJSON))
	if err != nil {
		t.Fatalf("r4.UnmarshalResource(Patient): %v", err)
	}
	if _, ok := r4Patient.(*r4.Patient); !ok {
		t.Fatalf("r4.UnmarshalResource returned %T, want *r4.Patient", r4Patient)
	}

	r5Patient, err := r5.UnmarshalResource([]byte(patientJSON))
	if err != nil {
		t.Fatalf("r5.UnmarshalResource(Patient): %v", err)
	}
	if _, ok := r5Patient.(*r5.Patient); !ok {
		t.Fatalf("r5.UnmarshalResource returned %T, want *r5.Patient", r5Patient)
	}
}

// TestDualReleaseValidateIsReleaseScoped proves Validate consumes the release-local
// descriptor registry: an Observation missing its required status is reported by each
// release's own Validate, and a complete one passes, with each release validating its
// own concrete type. A shared registry could not tell the two releases' Observation
// descriptors apart; release-scoped registries can.
func TestDualReleaseValidateIsReleaseScoped(t *testing.T) {
	if err := r4.Validate(&r4.Observation{}).Error(); err == nil || !strings.Contains(err.Error(), "Observation.status") {
		t.Errorf("r4.Validate(empty Observation) missing required-status issue: %v", err)
	}
	if err := r5.Validate(&r5.Observation{}).Error(); err == nil || !strings.Contains(err.Error(), "Observation.status") {
		t.Errorf("r5.Validate(empty Observation) missing required-status issue: %v", err)
	}

	r4Status := r4.ObservationStatusFinal
	r4Complete := &r4.Observation{Status: &r4Status, Code: &r4.CodeableConcept{Text: stringPtr("synthetic")}}
	if oo := r4.Validate(r4Complete); oo.HasErrors() {
		t.Errorf("r4.Validate(complete Observation) reported errors: %v", oo.Error())
	}
	r5Status := r5.ObservationStatusFinal
	r5Complete := &r5.Observation{Status: &r5Status, Code: &r5.CodeableConcept{Text: stringPtr("synthetic")}}
	if oo := r5.Validate(r5Complete); oo.HasErrors() {
		t.Errorf("r5.Validate(complete Observation) reported errors: %v", oo.Error())
	}
}

func stringPtr(s string) *string { return &s }
