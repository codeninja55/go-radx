package convert

import (
	"errors"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// canonicalADT is an ADT^A01 (admit) carrying patient demographics (PID), an event
// (EVN), and a patient visit (PV1). PID-8 "F" maps to the female gender; PV1-2 "I"
// inpatient maps to the encounter class.
const canonicalADT = "MSH|^~\\&|ADT1|HOSP|EMR|HOSP|202605311230-0500||ADT^A01|MSGADT1|P|2.4\r" +
	"EVN|A01|202605311230-0500\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F|||123 MAIN ST^^METROPOLIS^NY^12345^USA\r" +
	"PV1|1|I|WARD3^301^A||||||||||||||||VISIT-9001\r"

func TestADTToPatientR5(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(canonicalADT))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}

	pat, report, err := ADTToPatientR5(msg)
	if err != nil {
		t.Fatalf("ADTToPatientR5: %v", err)
	}
	if pat == nil {
		t.Fatal("Patient is nil")
	}
	_ = report

	// PID-3 becomes a logical Patient.identifier.
	if len(pat.Identifier) == 0 || pat.Identifier[0].Value == nil || *pat.Identifier[0].Value != "555-44-4444" {
		t.Errorf("identifier = %+v, want 555-44-4444", pat.Identifier)
	}

	// PID-5 becomes a HumanName.
	if len(pat.Name) == 0 || pat.Name[0].Family == nil || *pat.Name[0].Family != "EVERYWOMAN" {
		t.Errorf("name family = %+v, want EVERYWOMAN", pat.Name)
	}
	if len(pat.Name[0].Given) == 0 || pat.Name[0].Given[0] != "EVE" {
		t.Errorf("name given = %+v, want EVE", pat.Name)
	}

	// PID-8 "F" maps to the female gender, value-set-safe.
	if pat.Gender == nil || *pat.Gender != r5.AdministrativeGenderFemale {
		t.Errorf("gender = %v, want female", pat.Gender)
	}

	// PID-7 becomes the birthDate (day precision).
	if pat.BirthDate == nil || *pat.BirthDate != "1962-03-20" {
		t.Errorf("birthDate = %v, want 1962-03-20", pat.BirthDate)
	}

	// PID-11 becomes an Address.
	if len(pat.Address) == 0 || pat.Address[0].City == nil || *pat.Address[0].City != "METROPOLIS" {
		t.Errorf("address city = %+v, want METROPOLIS", pat.Address)
	}

	if oo := fhir.Validate(pat); oo.HasErrors() {
		t.Errorf("Patient fails validation: %+v", oo.Issue)
	}
}

// TestADTToPatientR5UnknownGenderSubstituted confirms a PID-8 code outside HL7
// Table 0001 is mapped to the value-set-safe "unknown" and recorded as a
// Substitution that names the concept, never the raw code.
func TestADTToPatientR5UnknownGenderSubstituted(t *testing.T) {
	const adt = "MSH|^~\\&|ADT1|HOSP|EMR|HOSP|202605311230||ADT^A01|M|P|2.4\r" +
		"PID|||555-44-4444^^^HOSP^MR||DOE^JANE||19700101|ZZ\r" +
		"PV1|1|I\r"
	msg, err := hl7v2.Parse([]byte(adt))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}
	pat, report, err := ADTToPatientR5(msg)
	if err != nil {
		t.Fatalf("ADTToPatientR5: %v", err)
	}
	if pat.Gender == nil || *pat.Gender != r5.AdministrativeGenderUnknown {
		t.Errorf("gender = %v, want unknown for an out-of-table code", pat.Gender)
	}
	if !hasSubstitutionContaining(report, "Patient.gender") {
		t.Errorf("Report.Substituted does not record the gender approximation: %+v", report.Substituted)
	}
	for _, s := range report.Substituted {
		if strings.Contains(s.Concept+s.Approximation, "ZZ") {
			t.Errorf("Report leaks the raw gender code: %+v", s)
		}
	}
	if oo := fhir.Validate(pat); oo.HasErrors() {
		t.Errorf("Patient fails validation with an unknown gender: %+v", oo.Issue)
	}
}

// TestADTToPatientR5BirthDateTimePrecisionDropped confirms a PID-7 birth date that
// carries time precision Patient.birthDate (a date) cannot hold records a Dropped
// entry naming the concept, never the raw value.
func TestADTToPatientR5BirthDateTimePrecisionDropped(t *testing.T) {
	const adt = "MSH|^~\\&|ADT1|HOSP|EMR|HOSP|202605311230||ADT^A01|M|P|2.4\r" +
		"PID|||555-44-4444^^^HOSP^MR||DOE^JANE||196203201415|F\r" +
		"PV1|1|I\r"
	msg, err := hl7v2.Parse([]byte(adt))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}
	pat, report, err := ADTToPatientR5(msg)
	if err != nil {
		t.Fatalf("ADTToPatientR5: %v", err)
	}
	// The date components survive; the time-of-birth precision is dropped.
	if pat.BirthDate == nil || *pat.BirthDate != "1962-03-20" {
		t.Errorf("birthDate = %v, want 1962-03-20 (time precision reduced)", pat.BirthDate)
	}
	if !hasDroppedContaining(report, "PID-7") {
		t.Errorf("Report.Dropped does not record the dropped birth-date time precision: %+v", report.Dropped)
	}
	for _, d := range report.Dropped {
		if strings.Contains(d.Source+d.Reason, "1415") || strings.Contains(d.Source+d.Reason, "196203201415") {
			t.Errorf("Report leaks the raw birth-date value: %+v", d)
		}
	}
	if oo := fhir.Validate(pat); oo.HasErrors() {
		t.Errorf("Patient fails validation: %+v", oo.Issue)
	}
}

// TestADTToPatientR5StrictLossEscalates confirms WithStrictLoss escalates a lossy
// birth-date reduction to a *LossError instead of a silent Report.Dropped entry.
func TestADTToPatientR5StrictLossEscalates(t *testing.T) {
	const adt = "MSH|^~\\&|ADT1|HOSP|EMR|HOSP|202605311230||ADT^A01|M|P|2.4\r" +
		"PID|||555-44-4444^^^HOSP^MR||DOE^JANE||196203201415|F\r" +
		"PV1|1|I\r"
	msg, err := hl7v2.Parse([]byte(adt))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}
	_, _, err = ADTToPatientR5(msg, WithStrictLoss())
	if err == nil {
		t.Fatal("error = nil, want a *LossError under WithStrictLoss for a lossy birth date")
	}
	var le *LossError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LossError", err)
	}
	// The default (lenient) path must keep returning a nil error.
	if _, _, err := ADTToPatientR5(msg); err != nil {
		t.Errorf("lenient ADTToPatientR5 returned an error: %v", err)
	}
}

func TestParseAdministrativeGender(t *testing.T) {
	cases := []struct {
		in            string
		want          r5.AdministrativeGender
		wantSubstitue bool
	}{
		{"M", r5.AdministrativeGenderMale, false},
		{"F", r5.AdministrativeGenderFemale, false},
		{"O", r5.AdministrativeGenderOther, false},
		{"A", r5.AdministrativeGenderOther, false},
		{"N", r5.AdministrativeGenderOther, false},
		{"U", r5.AdministrativeGenderUnknown, false},
		{"", r5.AdministrativeGenderUnknown, false},
		{"X", r5.AdministrativeGenderUnknown, true},
	}
	for _, c := range cases {
		got, substituted := ParseAdministrativeGender(c.in)
		if got != c.want {
			t.Errorf("ParseAdministrativeGender(%q) = %q, want %q", c.in, got, c.want)
		}
		if substituted != c.wantSubstitue {
			t.Errorf("ParseAdministrativeGender(%q) substituted = %v, want %v", c.in, substituted, c.wantSubstitue)
		}
	}
}

func TestADTToEncounterR5(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(canonicalADT))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}

	enc, report, err := ADTToEncounterR5(msg)
	if err != nil {
		t.Fatalf("ADTToEncounterR5: %v", err)
	}
	if enc == nil {
		t.Fatal("Encounter is nil")
	}
	_ = report

	// A01 (admit) maps to in-progress.
	if enc.Status == nil || *enc.Status != r5.EncounterStatusInProgress {
		t.Errorf("status = %v, want in-progress for A01", enc.Status)
	}

	// PV1-19 becomes a logical Encounter.identifier.
	if len(enc.Identifier) == 0 || enc.Identifier[0].Value == nil || *enc.Identifier[0].Value != "VISIT-9001" {
		t.Errorf("identifier = %+v, want VISIT-9001", enc.Identifier)
	}

	// PID-3 becomes a logical subject reference — never a URL.
	if enc.Subject == nil || enc.Subject.Reference != nil {
		t.Fatalf("subject = %+v, want a logical identifier reference (no URL)", enc.Subject)
	}
	if enc.Subject.Identifier == nil || enc.Subject.Identifier.Value == nil ||
		*enc.Subject.Identifier.Value != "555-44-4444" {
		t.Errorf("subject identifier = %+v, want 555-44-4444", enc.Subject.Identifier)
	}

	if oo := fhir.Validate(enc); oo.HasErrors() {
		t.Errorf("Encounter fails validation: %+v", oo.Issue)
	}
}

// TestADTToEncounterR5TriggerStatusMapping confirms the trigger-event -> status map
// and that each approximation records a Substitution.
func TestADTToEncounterR5TriggerStatusMapping(t *testing.T) {
	cases := []struct {
		event           string
		want            r5.EncounterStatus
		wantSubstituted bool
	}{
		{"A01", r5.EncounterStatusInProgress, false},
		{"A03", r5.EncounterStatusCompleted, false},
		{"A11", r5.EncounterStatusCancelled, false},
		{"A99", r5.EncounterStatusUnknown, true},
	}
	for _, c := range cases {
		adt := "MSH|^~\\&|A|H|E|H|202605311230||ADT^" + c.event + "|M|P|2.4\r" +
			"EVN|" + c.event + "|202605311230\r" +
			"PID|||555-44-4444^^^HOSP^MR||DOE^JANE||19700101|F\r" +
			"PV1|1|I\r"
		msg, err := hl7v2.Parse([]byte(adt))
		if err != nil {
			t.Fatalf("parse ADT(%s): %v", c.event, err)
		}
		enc, report, err := ADTToEncounterR5(msg)
		if err != nil {
			t.Fatalf("ADTToEncounterR5(%s): %v", c.event, err)
		}
		if enc.Status == nil || *enc.Status != c.want {
			t.Errorf("status(%s) = %v, want %v", c.event, enc.Status, c.want)
		}
		if c.wantSubstituted && !hasSubstitutionContaining(report, "Encounter.status") {
			t.Errorf("status(%s) did not record a Substitution: %+v", c.event, report.Substituted)
		}
		if oo := fhir.Validate(enc); oo.HasErrors() {
			t.Errorf("Encounter(%s) fails validation: %+v", c.event, oo.Issue)
		}
	}
}

// TestADTToPatientR5RejectsNonADT rejects a message that is not an ADT.
func TestADTToPatientR5RejectsNonADT(t *testing.T) {
	const orm = "MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r"
	msg, err := hl7v2.Parse([]byte(orm))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}
	if _, _, err := ADTToPatientR5(msg); err == nil {
		t.Fatal("error = nil, want ErrUnsupportedSource for a non-ADT message")
	}
	if _, _, err := ADTToEncounterR5(msg); err == nil {
		t.Fatal("error = nil, want ErrUnsupportedSource for a non-ADT message")
	}
}

// hasSubstitutionContaining reports whether any substitution's Concept contains sub.
func hasSubstitutionContaining(r *Report, sub string) bool {
	for _, s := range r.Substituted {
		if strings.Contains(s.Concept, sub) {
			return true
		}
	}
	return false
}
