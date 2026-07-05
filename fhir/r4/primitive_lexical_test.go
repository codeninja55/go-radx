package r4_test

// Behavioural tests for the R4 validation depth: the official R4 primitive regexes
// (whose dateTime pattern makes the timezone offset mandatory whenever a time-of-day is
// present, and whose fractional seconds are unbounded where R5 caps them at nine
// digits), nested-backbone required presence, and the recursive Extension.url
// requirement. The R4 tree previously had no test package; these pin the release's own
// lexical rules rather than assuming R5's.

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
)

func lexStr(s string) *string { return &s }

// hasIssueAt reports whether the outcome carries an error issue whose expression is
// exactly path.
func hasIssueAt(oo *fhir.OperationOutcome, path string) bool {
	for _, issue := range oo.Issue {
		if issue.Expression == path && issue.Severity == fhir.SeverityError {
			return true
		}
	}
	return false
}

// TestValidateDateLexicalR4 rejects an out-of-range month and an impossible calendar
// day at Patient.birthDate while the partial-precision forms pass.
func TestValidateDateLexicalR4(t *testing.T) {
	for _, bad := range []string{"2026-13-01", "2026-02-30", "2026-1-01"} {
		if oo := r4.Validate(&r4.Patient{BirthDate: lexStr(bad)}); !hasIssueAt(oo, "Patient.birthDate") {
			t.Errorf("birthDate %q should be a lexical issue, got %+v", bad, oo.Issue)
		}
	}
	for _, good := range []string{"1905", "1973-06", "1905-08-23"} {
		if oo := r4.Validate(&r4.Patient{BirthDate: lexStr(good)}); oo.HasErrors() {
			t.Errorf("birthDate %q should be valid, got %+v", good, oo.Issue)
		}
	}
}

// TestValidateDateTimeLexicalR4 pins the two R4-specific rules: the R4 regex itself
// rejects a time-of-day without an offset, and R4's fractional seconds are unbounded —
// a ten-digit fraction that R5 rejects is valid R4.
func TestValidateDateTimeLexicalR4(t *testing.T) {
	for _, bad := range []string{"2015-02-07T13:28:17", "2015-02-07T24:00:00+02:00"} {
		v := r4.FHIRDateTime(bad)
		if oo := r4.Validate(&r4.Patient{DeceasedDateTime: &v}); !hasIssueAt(oo, "Patient.deceasedDateTime") {
			t.Errorf("deceasedDateTime %q should be a lexical issue, got %+v", bad, oo.Issue)
		}
	}
	for _, good := range []string{"2015-02-07", "2015-02-07T13:28:17+02:00", "2015-02-07T13:28:17.1234567890+02:00"} {
		v := r4.FHIRDateTime(good)
		if oo := r4.Validate(&r4.Patient{DeceasedDateTime: &v}); oo.HasErrors() {
			t.Errorf("deceasedDateTime %q should be valid R4, got %+v", good, oo.Issue)
		}
	}
}

// TestValidateTimeLexicalR4 enforces the R4 time rules on the boxed Observation valueTime
// branch: no 24:00:00, no timezone, leap second allowed.
func TestValidateTimeLexicalR4(t *testing.T) {
	status := r4.ObservationStatusFinal
	obs := func(s string) *r4.Observation {
		v := r4.FHIRTime(s)
		return &r4.Observation{Status: &status, Code: &r4.CodeableConcept{}, ValueTime: &v}
	}
	for _, bad := range []string{"24:00:00", "13:28", "13:28:17+02:00", "13:28:61"} {
		if oo := r4.Validate(obs(bad)); !hasIssueAt(oo, "Observation.valueTime") {
			t.Errorf("valueTime %q should be a lexical issue, got %+v", bad, oo.Issue)
		}
	}
	for _, good := range []string{"13:28:17", "23:59:60", "00:00:00.000001"} {
		if oo := r4.Validate(obs(good)); oo.HasErrors() {
			t.Errorf("valueTime %q should be valid, got %+v", good, oo.Issue)
		}
	}
}

// TestValidateInstantLexicalR4 enforces the R4 instant rules on Slot.start: at least
// second precision with a mandatory timezone offset.
func TestValidateInstantLexicalR4(t *testing.T) {
	free := r4.SlotStatusFree
	slot := func(start string) *r4.Slot {
		return &r4.Slot{
			Schedule: &r4.Reference{},
			Status:   &free,
			Start:    lexStr(start),
			End:      lexStr("2026-01-01T11:00:00Z"),
		}
	}
	for _, bad := range []string{"2026-01-01T10:00:00", "2026-01-01", "2026-01-01T10:00Z"} {
		if oo := r4.Validate(slot(bad)); !hasIssueAt(oo, "Slot.start") {
			t.Errorf("Slot.start %q should be a lexical issue, got %+v", bad, oo.Issue)
		}
	}
	if oo := r4.Validate(slot("2026-01-01T10:00:00.5+10:00")); oo.HasErrors() {
		t.Errorf("a valid instant should pass, got %+v", oo.Issue)
	}
}

// TestValidateNestedBackboneRequiredR4 proves the nested-required walk ships for R4
// too: a transaction entry's empty request reports its missing method and url at the
// indexed occurrence path.
func TestValidateNestedBackboneRequiredR4(t *testing.T) {
	bt := r4.BundleTypeTransaction
	bundle := &r4.Bundle{Type: &bt, Entry: []r4.BundleEntry{
		{Request: &r4.BundleEntryRequest{}},
	}}
	oo := r4.Validate(bundle)
	if !hasIssueAt(oo, "Bundle.entry[0].request.method") || !hasIssueAt(oo, "Bundle.entry[0].request.url") {
		t.Errorf("an empty entry.request should report missing method and url, got %+v", oo.Issue)
	}
}

// TestValidateExtensionURLRequiredR4 proves the recursive Extension.url requirement
// ships for R4 too.
func TestValidateExtensionURLRequiredR4(t *testing.T) {
	p := &r4.Patient{DomainResource: r4.DomainResource{
		Extension: []r4.Extension{{}},
	}}
	if oo := r4.Validate(p); !hasIssueAt(oo, "Patient.extension[0].url") {
		t.Errorf("an extension without url should be a required issue, got %+v", oo.Issue)
	}
}
