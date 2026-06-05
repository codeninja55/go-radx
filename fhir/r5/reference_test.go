package r5_test

import (
	"errors"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// containerWithContained builds a Patient carrying one contained Observation with the
// given id, the shape "#id" resolution and contained-integrity checks operate on.
func containerWithContained(containedID string) *r5.Patient {
	p := &r5.Patient{}
	obs := &r5.Observation{}
	obs.ID = strptr(containedID)
	p.Contained = []fhir.Resource{obs}
	return p
}

func TestBundleResolveByFullURL(t *testing.T) {
	target := &r5.Observation{}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: target},
		r5.CollectionEntry{FullURL: "urn:uuid:obs-2", Resource: &r5.Observation{}},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	got, ok := bundle.Resolve("urn:uuid:obs-1")
	if !ok {
		t.Fatalf("Resolve(urn:uuid:obs-1) not found")
	}
	if obs, ok := fhir.As[*r5.Observation](got); !ok || obs != target {
		t.Errorf("Resolve returned the wrong resource")
	}
}

func TestBundleResolveContainedFragment(t *testing.T) {
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:pat-1", Resource: containerWithContained("obs-c")},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	got, ok := bundle.Resolve("#obs-c")
	if !ok {
		t.Fatalf("Resolve(#obs-c) not found")
	}
	if got.ResourceType() != "Observation" {
		t.Errorf("resolved type = %s, want Observation", got.ResourceType())
	}
}

func TestBundleResolveExternalURLNotFound(t *testing.T) {
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: &r5.Observation{}},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	if _, ok := bundle.Resolve("https://example.org/fhir/Patient/external"); ok {
		t.Errorf("external URL resolved, want not found")
	}
}

func TestResolveContainedMalformedReturnsError(t *testing.T) {
	p := &r5.Patient{}
	// A typed-nil contained resource is malformed: it cannot be addressed by id.
	var nilObs *r5.Observation
	p.Contained = []fhir.Resource{nilObs}

	_, err := p.ResolveContained("anything")
	if !errors.Is(err, r5.ErrContained) {
		t.Fatalf("err = %v, want ErrContained", err)
	}
}

func TestResolveContainedNoIDReturnsError(t *testing.T) {
	p := &r5.Patient{}
	// A contained resource with no id is malformed for "#id" resolution.
	p.Contained = []fhir.Resource{&r5.Observation{}}

	_, err := p.ResolveContained("obs-c")
	if !errors.Is(err, r5.ErrContained) {
		t.Fatalf("err = %v, want ErrContained naming the missing id", err)
	}
}

func TestResolveContainedFound(t *testing.T) {
	p := containerWithContained("obs-c")
	got, err := p.ResolveContained("obs-c")
	if err != nil {
		t.Fatalf("ResolveContained: %v", err)
	}
	if got == nil || got.ResourceType() != "Observation" {
		t.Errorf("resolved = %v, want Observation", got)
	}
}

func TestCheckReferenceIntegrityReportsDanglingFragment(t *testing.T) {
	obs := &r5.Observation{Subject: &r5.Reference{Reference: strptr("#missing")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if !outcome.HasErrors() {
		t.Fatalf("expected a dangling-reference error, got none")
	}
	if outcome.Error() == nil {
		t.Errorf("Error() = nil, want non-nil for a dangling reference")
	}
}

func TestCheckReferenceIntegrityResolvesContainedFragment(t *testing.T) {
	p := containerWithContained("obs-c")
	p.ManagingOrganization = &r5.Reference{Reference: strptr("#obs-c")}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:pat-1", Resource: p},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if outcome.HasErrors() {
		t.Errorf("contained reference should resolve, got %v", outcome.Error())
	}
}

func TestCheckReferenceIntegrityResolvesIntraBundleFullURL(t *testing.T) {
	obs := &r5.Observation{Subject: &r5.Reference{Reference: strptr("urn:uuid:pat-1")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:pat-1", Resource: &r5.Patient{}},
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if outcome.HasErrors() {
		t.Errorf("intra-bundle fullUrl reference should resolve, got %v", outcome.Error())
	}
}

func TestCheckReferenceIntegrityReportsDanglingURN(t *testing.T) {
	// A urn:uuid: reference is intra-bundle (URNs are not network-dereferenceable), so a
	// urn that matches no entry fullUrl is a dangling reference, not an external one.
	obs := &r5.Observation{Subject: &r5.Reference{Reference: strptr("urn:uuid:not-present")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if !outcome.HasErrors() {
		t.Errorf("a dangling urn:uuid: reference should be flagged, not skipped as external")
	}
}

func TestResolveContainedReportsMalformedEvenWhenMatchFound(t *testing.T) {
	p := containerWithContained("obs-c")
	// Append a malformed (typed-nil) slot AFTER the matching one.
	var nilObs *r5.Observation
	p.Contained = append(p.Contained, nilObs)

	_, err := p.ResolveContained("obs-c")
	if !errors.Is(err, r5.ErrContained) {
		t.Fatalf("err = %v, want ErrContained even though the id matched a clean slot", err)
	}
}

func TestCheckReferenceIntegrityIgnoresExternalURL(t *testing.T) {
	obs := &r5.Observation{Subject: &r5.Reference{Reference: strptr("https://example.org/fhir/Patient/ext")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if outcome.HasErrors() {
		t.Errorf("external absolute URL should not be flagged, got %v", outcome.Error())
	}
}

func TestCheckReferenceIntegrityFlagsMalformedAuthorityURL(t *testing.T) {
	// A scheme with an empty authority is not a resolvable external URL; it must be
	// reported as dangling rather than skipped.
	obs := &r5.Observation{Subject: &r5.Reference{Reference: strptr("https:///Patient/x")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	if !bundle.CheckReferenceIntegrity().HasErrors() {
		t.Errorf("a scheme with an empty authority should be flagged, not skipped as external")
	}
}

func TestCheckReferenceIntegrityReportsDanglingRelative(t *testing.T) {
	obs := &r5.Observation{Subject: &r5.Reference{Reference: strptr("Patient/not-in-bundle")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if !outcome.HasErrors() {
		t.Errorf("a relative reference matching no entry fullUrl should be flagged")
	}
}

func TestCheckReferenceIntegrityAggregatesMultipleDangling(t *testing.T) {
	obs1 := &r5.Observation{Subject: &r5.Reference{Reference: strptr("#missing-1")}}
	obs2 := &r5.Observation{Subject: &r5.Reference{Reference: strptr("Patient/missing-2")}}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:obs-1", Resource: obs1},
		r5.CollectionEntry{FullURL: "urn:uuid:obs-2", Resource: obs2},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if len(outcome.Issue) < 2 {
		t.Fatalf("expected at least 2 aggregated issues, got %d", len(outcome.Issue))
	}
}

func TestCheckReferenceIntegrityReportsMalformedContained(t *testing.T) {
	p := &r5.Patient{}
	var nilObs *r5.Observation
	p.Contained = []fhir.Resource{nilObs}
	bundle, err := r5.NewCollection(
		r5.CollectionEntry{FullURL: "urn:uuid:pat-1", Resource: p},
	)
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	outcome := bundle.CheckReferenceIntegrity()
	if !outcome.HasErrors() {
		t.Errorf("a malformed contained resource should be reported, not silently skipped")
	}
}

func TestHasErrorsNilSafe(t *testing.T) {
	var oo *r5.OperationOutcome
	if oo.HasErrors() {
		t.Errorf("nil outcome HasErrors() = true, want false")
	}
	if oo.Error() != nil {
		t.Errorf("nil outcome Error() != nil")
	}
}

func TestHasErrorsInformationOnly(t *testing.T) {
	sev := r5.IssueSeverityInformation
	oo := &r5.OperationOutcome{
		Issue: []r5.OperationOutcomeIssue{{Severity: &sev}},
	}
	if oo.HasErrors() {
		t.Errorf("information-only outcome HasErrors() = true, want false")
	}
	if oo.Error() != nil {
		t.Errorf("information-only outcome Error() != nil")
	}
}

func TestErrorNamesSeverity(t *testing.T) {
	sev := r5.IssueSeverityError
	oo := &r5.OperationOutcome{
		Issue: []r5.OperationOutcomeIssue{{Severity: &sev, Diagnostics: strptr("unresolved local reference \"#x\"")}},
	}
	err := oo.Error()
	if err == nil {
		t.Fatal("Error() = nil, want non-nil")
	}
}
