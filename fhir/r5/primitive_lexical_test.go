package r5_test

// Behavioural tests for the R5 validation depth shipped by the generated descriptors:
// date/dateTime/time/instant lexical checks (the official R5 primitive regexes plus the
// offset-with-time and valid-calendar-date prose rules), required-element presence
// inside backbone elements at every depth, and the recursive Extension.url requirement.
// Every assertion drives r5.Validate over a decoded-shape resource, the same gate a
// wire resource passes through.

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
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

// TestValidateDateLexicalR5 rejects an out-of-range month, an impossible calendar day,
// and a malformed shape at Patient.birthDate, while the spec's partial-precision forms
// (year, year-month, full date) pass. The issue names the path and the type, never the
// value — a birth date is PHI.
func TestValidateDateLexicalR5(t *testing.T) {
	for _, bad := range []string{"2026-13-01", "2026-02-30", "2026-1-01", "01-01-2026", "2026-01-01T10:00:00Z"} {
		p := &r5.Patient{BirthDate: lexStr(bad)}
		oo := r5.Validate(p)
		if !hasIssueAt(oo, "Patient.birthDate") {
			t.Errorf("birthDate %q should be a lexical issue, got %+v", bad, oo.Issue)
			continue
		}
		for _, issue := range oo.Issue {
			if strings.Contains(issue.Diagnostics, bad) {
				t.Errorf("diagnostic echoes the date value (PHI): %q", issue.Diagnostics)
			}
		}
	}
	for _, good := range []string{"1905", "1973-06", "1905-08-23", "2024-02-29"} {
		if oo := r5.Validate(&r5.Patient{BirthDate: lexStr(good)}); oo.HasErrors() {
			t.Errorf("birthDate %q should be valid, got %+v", good, oo.Issue)
		}
	}
}

// TestValidateDateTimeLexicalR5 enforces the R5 dateTime rules on the boxed
// Patient.deceasedDateTime choice branch: a value carrying a time-of-day must carry a
// populated timezone offset (the datatypes.html SHALL the published regex alone does
// not enforce), a date-only value must NOT carry an offset (the converse, which the
// published R5 offset group wrongly admits and R4 rejects), and R5 caps fractional
// seconds at nine digits.
func TestValidateDateTimeLexicalR5(t *testing.T) {
	bad := []string{
		"2015-02-07T13:28:17",                  // time-of-day without an offset
		"2015-02-07T13:28:17+",                 // the published-regex erratum's bare sign
		"2015-02-07T13:28:17.1234567890+02:00", // ten fractional digits; R5 caps at nine
		"2015-02-07T24:00:00+02:00",            // hour 24 does not exist
		"2026-05Z",                             // year-month with an offset but no time
		"2026-01-15Z",                          // date-only with an offset but no time
		"2026-01+02:00",                        // year-month with an offset but no time
	}
	for _, s := range bad {
		v := r5.FHIRDateTime(s)
		oo := r5.Validate(&r5.Patient{DeceasedDateTime: &v})
		if !hasIssueAt(oo, "Patient.deceasedDateTime") {
			t.Errorf("deceasedDateTime %q should be a lexical issue, got %+v", s, oo.Issue)
		}
	}
	good := []string{"2015", "2015-02", "2015-02-07", "2015-02-07T13:28:17+02:00", "2015-02-07T13:28:17.239Z"}
	for _, s := range good {
		v := r5.FHIRDateTime(s)
		if oo := r5.Validate(&r5.Patient{DeceasedDateTime: &v}); oo.HasErrors() {
			t.Errorf("deceasedDateTime %q should be valid, got %+v", s, oo.Issue)
		}
	}
}

// TestValidateTimeLexicalR5 enforces the R5 time rules on the boxed Observation
// valueTime branch: no 24:00:00, no timezone, leap second allowed.
func TestValidateTimeLexicalR5(t *testing.T) {
	status := r5.ObservationStatusFinal
	obs := func(s string) *r5.Observation {
		v := r5.FHIRTime(s)
		return &r5.Observation{Status: &status, Code: &r5.CodeableConcept{}, ValueTime: &v}
	}
	for _, bad := range []string{"24:00:00", "13:28", "13:28:17+02:00", "13:28:61"} {
		if oo := r5.Validate(obs(bad)); !hasIssueAt(oo, "Observation.valueTime") {
			t.Errorf("valueTime %q should be a lexical issue, got %+v", bad, oo.Issue)
		}
	}
	for _, good := range []string{"13:28:17", "23:59:60", "00:00:00.000000001"} {
		if oo := r5.Validate(obs(good)); oo.HasErrors() {
			t.Errorf("valueTime %q should be valid, got %+v", good, oo.Issue)
		}
	}
}

// TestValidateInstantLexicalR5 enforces the instant rules on Slot.start: at least
// second precision with a mandatory timezone offset.
func TestValidateInstantLexicalR5(t *testing.T) {
	free := r5.SlotStatusFree
	slot := func(start string) *r5.Slot {
		return &r5.Slot{
			Schedule: &r5.Reference{},
			Status:   &free,
			Start:    lexStr(start),
			End:      lexStr("2026-01-01T11:00:00Z"),
		}
	}
	for _, bad := range []string{"2026-01-01T10:00:00", "2026-01-01", "2026-01-01T10:00Z"} {
		if oo := r5.Validate(slot(bad)); !hasIssueAt(oo, "Slot.start") {
			t.Errorf("Slot.start %q should be a lexical issue, got %+v", bad, oo.Issue)
		}
	}
	if oo := r5.Validate(slot("2026-01-01T10:00:00.5+10:00")); oo.HasErrors() {
		t.Errorf("a valid instant should pass, got %+v", oo.Issue)
	}
}

// TestValidateNestedBackboneRequiredR5 proves required-element presence now reaches
// inside backbone elements: a transaction entry's empty request reports its missing
// method and url at the indexed occurrence path, an ImagingStudy series without a uid
// reports it one level down, and a present-but-empty string is present (the FHIR-007
// presence rule holds at depth: presence is the pointer, never the value).
func TestValidateNestedBackboneRequiredR5(t *testing.T) {
	bt := r5.BundleTypeTransaction
	bundle := &r5.Bundle{Type: &bt, Entry: []r5.BundleEntry{
		{Request: &r5.BundleEntryRequest{}},
	}}
	oo := r5.Validate(bundle)
	if !hasIssueAt(oo, "Bundle.entry[0].request.method") || !hasIssueAt(oo, "Bundle.entry[0].request.url") {
		t.Errorf("an empty entry.request should report missing method and url, got %+v", oo.Issue)
	}

	// Present-but-empty is present: an empty-string url is not "missing" (its validity
	// is a different rule), so only the absent method is reported.
	verb := r5.HTTPVerbGET
	bundle.Entry[0].Request.URL = lexStr("")
	bundle.Entry[0].Request.Method = &verb
	if oo := r5.Validate(bundle); oo.HasErrors() {
		t.Errorf("a present empty url and a set method should raise no required issue, got %+v", oo.Issue)
	}

	registered := r5.ImagingStudyStatusAvailable
	study := &r5.ImagingStudy{
		Status:  &registered,
		Subject: &r5.Reference{},
		Series:  []r5.ImagingStudySeries{{Modality: &r5.CodeableConcept{}}},
	}
	oo = r5.Validate(study)
	if !hasIssueAt(oo, "ImagingStudy.series[0].uid") {
		t.Errorf("a series without uid should report ImagingStudy.series[0].uid, got %+v", oo.Issue)
	}
}

// TestValidateExtensionURLRequiredR5 proves the fhir.resources 8.3.0-aligned
// Extension.url requirement: an extension without its url — top-level or nested inside
// another extension — is a required issue at the indexed path.
func TestValidateExtensionURLRequiredR5(t *testing.T) {
	p := &r5.Patient{DomainResource: r5.DomainResource{
		Extension: []r5.Extension{{}},
	}}
	if oo := r5.Validate(p); !hasIssueAt(oo, "Patient.extension[0].url") {
		t.Errorf("an extension without url should be a required issue, got %+v", oo.Issue)
	}

	nested := &r5.Patient{DomainResource: r5.DomainResource{
		Extension: []r5.Extension{{
			URL:     lexStr("http://example.org/outer"),
			Element: r5.Element{Extension: []r5.Extension{{}}},
		}},
	}}
	if oo := r5.Validate(nested); !hasIssueAt(oo, "Patient.extension[0].extension[0].url") {
		t.Errorf("a nested extension without url should be a required issue, got %+v", oo.Issue)
	}

	valid := &r5.Patient{DomainResource: r5.DomainResource{
		Extension: []r5.Extension{{URL: lexStr("http://example.org/defined")}},
	}}
	if oo := r5.Validate(valid); oo.HasErrors() {
		t.Errorf("an extension carrying its url should pass, got %+v", oo.Issue)
	}
}
