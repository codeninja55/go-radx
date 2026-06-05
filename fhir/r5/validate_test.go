package r5_test

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// phiSentinel is a synthetic patient-data value seeded into resources under validation.
// No issue diagnostic or expression may ever contain it: validation messages name
// elements, paths, types, and codes, never patient values (PRD §9.1). It is not real
// PHI — it is a recognisable token that would only appear in an issue if the engine
// leaked a field value.
const phiSentinel = "ZZSENTINEL-DO-NOT-LEAK-9981"

// assertNoPHI fails the test if any issue carries the seeded sentinel in its diagnostic
// or expression, proving the OperationOutcome is PHI-clean.
func assertNoPHI(t *testing.T, oo *fhir.OperationOutcome) {
	t.Helper()
	for _, issue := range oo.Issue {
		if strings.Contains(issue.Diagnostics, phiSentinel) || strings.Contains(issue.Expression, phiSentinel) {
			t.Errorf("validation issue leaked the PHI sentinel: %+v", issue)
		}
	}
}

// TestValidateRequiredAbsentRealResource reports a real resource's absent required
// elements by element path, with no value in the message. Observation requires status
// and code; an empty Observation reports both.
func TestValidateRequiredAbsentRealResource(t *testing.T) {
	oo := fhir.Validate(&r5.Observation{})
	paths := map[string]bool{}
	for _, issue := range oo.Issue {
		if issue.Code == fhir.IssueTypeRequired {
			paths[issue.Expression] = true
		}
	}
	if !paths["Observation.status"] || !paths["Observation.code"] {
		t.Fatalf("expected required issues for Observation.status and Observation.code, got %+v", oo.Issue)
	}
}

// TestValidateRequiredFalseIsPresentRealResource is the FHIR-007 behavioural regression on
// a real generated resource: Substance.instance is a required boolean; a Substance whose
// instance is validly false is present (a non-nil *bool) and must not be reported missing.
func TestValidateRequiredFalseIsPresentRealResource(t *testing.T) {
	s := &r5.Substance{
		Instance: boolPtr(false), // present and false — presence is the pointer, not the value
		Code:     &r5.CodeableReference{},
	}
	oo := fhir.Validate(s)
	for _, issue := range oo.Issue {
		if issue.Expression == "Substance.instance" {
			t.Errorf("a present required false was reported missing (FHIR-007 regression): %+v", issue)
		}
	}
}

// TestValidateChoiceMutualExclusionDirectWrite is the FHIR-001 regression caught at
// validation: writing two suffixed choice storage fields directly (bypassing the
// mutually-exclusive setters) is reported as one choice issue. The setters would have
// cleared the sibling; a direct field write does not, and Validate is the gate that
// catches it.
func TestValidateChoiceMutualExclusionDirectWrite(t *testing.T) {
	p := &r5.Patient{
		// A direct two-branch write the setters make unrepresentable through the API.
		DeceasedBoolean:  func() *r5.FHIRBoolean { v := r5.FHIRBoolean(true); return &v }(),
		DeceasedDateTime: func() *r5.FHIRDateTime { v := r5.FHIRDateTime("2020-01-01"); return &v }(),
	}
	oo := fhir.Validate(p)
	choiceIssues := 0
	for _, issue := range oo.Issue {
		if issue.Expression == "Patient.deceased[x]" {
			choiceIssues++
			if issue.Code != fhir.IssueTypeStructure {
				t.Errorf("a choice violation should be a structure issue, got %q", issue.Code)
			}
		}
	}
	if choiceIssues != 1 {
		t.Fatalf("expected exactly one choice mutual-exclusion issue, got %d: %+v", choiceIssues, oo.Issue)
	}
	assertNoPHI(t, oo)
}

// TestValidateBindingCodeRealResource reports an out-of-set required-binding code on a
// real resource. A Patient.gender set to a value outside the AdministrativeGender set is a
// value issue; the message names the binding, never the offending value.
func TestValidateBindingCodeRealResource(t *testing.T) {
	bad := r5.AdministrativeGender(phiSentinel) // an out-of-set value seeded with the sentinel
	p := &r5.Patient{Gender: &bad}
	oo := fhir.Validate(p)
	found := false
	for _, issue := range oo.Issue {
		if issue.Expression == "Patient.gender" && issue.Code == fhir.IssueTypeValue {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a binding value issue for Patient.gender, got %+v", oo.Issue)
	}
	assertNoPHI(t, oo) // the bad code value must not appear in any message
}

// TestValidateValidPatientIsClean confirms a structurally valid resource produces no
// error issues (Patient has no required top-level elements; a well-formed gender passes).
func TestValidateValidPatientIsClean(t *testing.T) {
	gender := r5.AdministrativeGenderFemale
	p := &r5.Patient{Gender: &gender, Active: boolPtr(true)}
	if oo := fhir.Validate(p); oo.HasErrors() {
		t.Fatalf("a valid Patient should have no error issues, got %+v", oo.Issue)
	}
}

// TestValidateBundleInvariants exercises the Bundle bdl-* extra checks: a document bundle
// whose first entry is not a Composition, and a total on a non-searchset bundle, are both
// reported. The Bundle descriptor composes these via BundleValidateExtra.
func TestValidateBundleInvariants(t *testing.T) {
	bt := r5.BundleTypeDocument
	total := int32(1)
	patient := &r5.Patient{Active: boolPtr(true)}
	b := &r5.Bundle{
		Type:  &bt,
		Total: &total, // bdl-1: total not allowed on a document bundle
		Entry: []r5.BundleEntry{
			{Resource: func() *fhir.Resource { var r fhir.Resource = patient; return &r }()}, // first entry is not a Composition (bdl-3)
		},
	}
	oo := fhir.Validate(b)
	var sawTotal, sawFirstEntry bool
	for _, issue := range oo.Issue {
		if issue.Expression == "Bundle.total" {
			sawTotal = true
		}
		if strings.HasPrefix(issue.Expression, "Bundle.entry[0]") {
			sawFirstEntry = true
		}
	}
	if !sawTotal {
		t.Error("expected a bdl-1 issue for total on a document bundle")
	}
	if !sawFirstEntry {
		t.Error("expected a bdl-3 issue for a non-Composition first entry")
	}
	assertNoPHI(t, oo)
}

// TestValidateBundleFullURLUniqueness exercises bdl-7/bdl-8: an exact duplicate fullUrl in a
// collection is flagged, but the same fullUrl on two resources with different meta.versionId
// is allowed (the versioned clause), and a history bundle is exempt entirely.
func TestValidateBundleFullURLUniqueness(t *testing.T) {
	makePatient := func(version string) *r5.Patient {
		p := &r5.Patient{Active: boolPtr(true)}
		if version != "" {
			p.Meta = &r5.Meta{VersionId: strPtr(version)}
		}
		return p
	}
	entryAt := func(url string, p *r5.Patient) r5.BundleEntry {
		return r5.BundleEntry{FullUrl: strPtr(url), Resource: func() *fhir.Resource { var r fhir.Resource = p; return &r }()}
	}

	// An exact duplicate fullUrl with no distinguishing version is a bdl-7 violation.
	dup := r5.BundleTypeCollection
	bDup := &r5.Bundle{Type: &dup, Entry: []r5.BundleEntry{
		entryAt("urn:uuid:1", makePatient("")),
		entryAt("urn:uuid:1", makePatient("")),
	}}
	if !hasIssue(fhir.Validate(bDup), "Bundle.entry[1].fullUrl", fhir.IssueTypeInvalid) {
		t.Error("expected a bdl-7 duplicate-fullUrl issue for an exact duplicate")
	}

	// The same fullUrl on two different versions is allowed.
	versioned := r5.BundleTypeCollection
	bVersioned := &r5.Bundle{Type: &versioned, Entry: []r5.BundleEntry{
		entryAt("urn:uuid:1", makePatient("1")),
		entryAt("urn:uuid:1", makePatient("2")),
	}}
	for _, issue := range fhir.Validate(bVersioned).Issue {
		if issue.Expression == "Bundle.entry[1].fullUrl" {
			t.Errorf("two distinct versions sharing a fullUrl should be allowed (bdl-7), got %+v", issue)
		}
	}

	// A history bundle is exempt from fullUrl uniqueness entirely.
	hist := r5.BundleTypeHistory
	bHist := &r5.Bundle{Type: &hist, Entry: []r5.BundleEntry{
		entryAt("urn:uuid:1", makePatient("")),
		entryAt("urn:uuid:1", makePatient("")),
	}}
	for _, issue := range fhir.Validate(bHist).Issue {
		if issue.Expression == "Bundle.entry[1].fullUrl" {
			t.Errorf("a history bundle should be exempt from fullUrl uniqueness (bdl-8), got %+v", issue)
		}
	}
}

// TestValidateBundleReferenceIntegrity confirms Validate composes the reference-integrity
// walk: a dangling "#id" reference inside a bundle entry is reported as a not-found issue.
func TestValidateBundleReferenceIntegrity(t *testing.T) {
	bt := r5.BundleTypeCollection
	obs := &r5.Observation{
		Status:  func() *r5.ObservationStatus { v := r5.ObservationStatusFinal; return &v }(),
		Code:    &r5.CodeableConcept{},
		Subject: &r5.Reference{Reference: strPtr("#missing")}, // resolves to nothing in the bundle
	}
	b := &r5.Bundle{
		Type:  &bt,
		Entry: []r5.BundleEntry{{Resource: func() *fhir.Resource { var r fhir.Resource = obs; return &r }()}},
	}
	oo := fhir.Validate(b)
	found := false
	for _, issue := range oo.Issue {
		if issue.Code == fhir.IssueTypeNotFound {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a not-found issue for a dangling #id reference, got %+v", oo.Issue)
	}
	assertNoPHI(t, oo)
}

// TestValidateBundleNilEntryResourceNoPanic is the never-panic regression for a malformed
// in-memory Bundle: a document/message bundle whose first entry holds a *fhir.Resource that
// points at a nil interface (a valid Go construction a caller can build directly). The
// first-entry check must report it as an issue, never dereference the nil interface and
// panic (PRD §9.3).
func TestValidateBundleNilEntryResourceNoPanic(t *testing.T) {
	for _, bt := range []r5.BundleType{r5.BundleTypeDocument, r5.BundleTypeMessage} {
		t.Run(string(bt), func(t *testing.T) {
			bundleType := bt
			var nilResource fhir.Resource // a nil interface boxed in a non-nil pointer
			b := &r5.Bundle{
				Type:  &bundleType,
				Entry: []r5.BundleEntry{{Resource: &nilResource}},
			}
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("Validate panicked on a nil-interface entry resource: %v", p)
				}
			}()
			oo := fhir.Validate(b)
			if !hasIssue(oo, "Bundle.entry[0]", fhir.IssueTypeInvalid) {
				t.Errorf("expected a bdl-3 first-entry issue for a nil entry resource, got %+v", oo.Issue)
			}
		})
	}
}

// TestValidateNeverLeaksPHIAcrossResources seeds the sentinel into many fields of several
// resources and asserts no issue ever surfaces it, the property the Phase 0 PHI sweep
// depends on: a validation message names paths and codes, never patient values.
func TestValidateNeverLeaksPHIAcrossResources(t *testing.T) {
	resources := []fhir.Resource{
		&r5.Patient{
			Gender: func() *r5.AdministrativeGender { v := r5.AdministrativeGender(phiSentinel); return &v }(),
		},
		&r5.Observation{Code: &r5.CodeableConcept{Text: strPtr(phiSentinel)}},
		&r5.Substance{Instance: boolPtr(false)},
		&r5.Bundle{},
	}
	for _, r := range resources {
		assertNoPHI(t, fhir.Validate(r))
	}
}

// TestValidateWorkflowResourcesHaveDescriptors confirms the workflow resources
// (ServiceRequest, DiagnosticReport, ImagingStudy) carry registered descriptors rather
// than only a warning: an empty resource reports its required code elements, an out-of-set
// status is a binding value issue, and a fully-populated resource is clean. The generator
// emits these descriptors from the StructureDefinition, so the required-element set is the
// faithful FHIR one (for example R5 also requires ServiceRequest.subject).
func TestValidateWorkflowResourcesHaveDescriptors(t *testing.T) {
	// An empty ServiceRequest reports its required code elements.
	oo := fhir.Validate(&r5.ServiceRequest{})
	required := map[string]bool{}
	for _, issue := range oo.Issue {
		if issue.Code == fhir.IssueTypeRequired {
			required[issue.Expression] = true
		}
	}
	if !required["ServiceRequest.status"] || !required["ServiceRequest.intent"] {
		t.Fatalf("expected required issues for ServiceRequest status and intent, got %+v", oo.Issue)
	}

	// An empty DiagnosticReport and ImagingStudy each report their required status.
	if oo := fhir.Validate(&r5.DiagnosticReport{}); !hasRequired(oo, "DiagnosticReport.status") {
		t.Errorf("expected a required issue for DiagnosticReport.status, got %+v", oo.Issue)
	}
	if oo := fhir.Validate(&r5.ImagingStudy{}); !hasRequired(oo, "ImagingStudy.status") {
		t.Errorf("expected a required issue for ImagingStudy.status, got %+v", oo.Issue)
	}

	// An out-of-set ServiceRequest status is a binding value issue.
	badStatus := r5.RequestStatus("banana")
	badIntent := r5.RequestIntentOrder
	bad := fhir.Validate(&r5.ServiceRequest{Status: &badStatus, Intent: &badIntent})
	if !hasIssue(bad, "ServiceRequest.status", fhir.IssueTypeValue) {
		t.Errorf("expected a binding value issue for an out-of-set ServiceRequest.status, got %+v", bad.Issue)
	}

	// A well-formed ServiceRequest is clean. R5 requires status, intent, and subject,
	// so a clean instance carries all three.
	goodStatus := r5.RequestStatusActive
	goodIntent := r5.RequestIntentOrder
	good := fhir.Validate(&r5.ServiceRequest{
		Status:  &goodStatus,
		Intent:  &goodIntent,
		Subject: &r5.Reference{Reference: strPtr("Patient/pat-1")},
	})
	if good.HasErrors() {
		t.Errorf("a valid ServiceRequest should have no error issues, got %+v", good.Issue)
	}
}

// hasRequired reports whether the outcome carries a required issue at the given path.
func hasRequired(oo *fhir.OperationOutcome, path string) bool {
	return hasIssue(oo, path, fhir.IssueTypeRequired)
}

// hasIssue reports whether the outcome carries an issue at the given path with the given code.
func hasIssue(oo *fhir.OperationOutcome, path string, code fhir.IssueType) bool {
	for _, issue := range oo.Issue {
		if issue.Expression == path && issue.Code == code {
			return true
		}
	}
	return false
}

// FuzzValidateNeverPanics drives Validate with resources decoded from arbitrary JSON. A
// malformed, partial, or hostile payload that decodes into a resource (or fails to decode)
// must never make Validate panic: a structurally broken value yields an OperationOutcome,
// never a crash (PRD §9.3). The fuzzer decodes the bytes through the release registry —
// which exercises every registered descriptor as the corpus explores resourceTypes — then
// validates whatever resource came back.
func FuzzValidateNeverPanics(f *testing.F) {
	f.Add([]byte(`{"resourceType":"Patient"}`))
	f.Add([]byte(`{"resourceType":"Patient","deceasedBoolean":true,"deceasedDateTime":"2020"}`))
	f.Add([]byte(`{"resourceType":"Observation"}`))
	f.Add([]byte(`{"resourceType":"Bundle","type":"document","total":1,"entry":[{}]}`))
	f.Add([]byte(`{"resourceType":"Bundle","type":"collection","entry":[{"resource":{"resourceType":"Observation","status":"final","subject":{"reference":"#x"}}}]}`))
	f.Add([]byte(`{"resourceType":"Substance","instance":false}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))
	f.Add([]byte(`{"resourceType":"NotAType"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		// UnmarshalResource may fail (unknown type, malformed JSON); when it returns a
		// resource, Validate must handle whatever shape decoded without panicking.
		r, err := fhir.UnmarshalResource(data)
		if err != nil {
			return
		}
		oo := fhir.Validate(r)
		// The outcome must never leak a value into a message even from a fuzzed input;
		// issue messages are paths and codes by construction.
		_ = oo
	})
}

// FuzzValidateTypedNeverPanics drives Validate directly with a typed Patient whose choice
// and code fields are populated from fuzzed strings, so the descriptor closures run over
// adversarial field values without going through JSON decode. It pins that the closures
// themselves never panic on an arbitrary (including empty) value.
func FuzzValidateTypedNeverPanics(f *testing.F) {
	f.Add("male", "", "")
	f.Add("banana", "true", "2020-01-01")
	f.Add("", "x", "y")
	f.Fuzz(func(t *testing.T, gender, deceasedBool, deceasedTime string) {
		p := &r5.Patient{}
		if gender != "" {
			g := r5.AdministrativeGender(gender)
			p.Gender = &g
		}
		if deceasedBool != "" {
			b := r5.FHIRBoolean(deceasedBool == "true")
			p.DeceasedBoolean = &b
		}
		if deceasedTime != "" {
			dt := r5.FHIRDateTime(deceasedTime)
			p.DeceasedDateTime = &dt
		}
		_ = fhir.Validate(p)
	})
}
