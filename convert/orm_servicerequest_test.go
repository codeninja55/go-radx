package convert

import (
	"errors"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// richORM exercises the broadened ServiceRequest mapping: an ORC carrying a
// Quantity/Timing priority (ORC-7 component 6 "R"), an ordering provider (ORC-12
// XCN), a PID-3 subject, a PV1-19 visit number, and an OBR with a requested
// date/time (OBR-6) and a reason for study (OBR-31). The timestamps carry a -0500
// offset so the times render as full FHIR dateTimes.
const richORM = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230-0500||ORM^O01|MSG00010|P|2.4\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
	"PV1|1|O|RADIOLOGY||||||||||||||||VISIT-9001\r" +
	"ORC|NW|PLACER123|FILLER456||||^^^^^R||202605311230-0500|||1234^WELBY^MARCUS^^^DR\r" +
	"OBR|1|PLACER123|FILLER456|36643-5^CHEST XRAY^LN||202605311231-0500|||||||||||||||||||||||||R07.9^Chest pain^I10\r"

// canonicalORM is a representative ORM^O01 with header, patient, common order,
// and one observation request, shaped after the hl7v2 package fixtures.
const canonicalORM = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230||ORM^O01|MSG00001|P|2.4\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
	"ORC|NW|PLACER123|FILLER456||||||202605311230\r" +
	"OBR|1|PLACER123|FILLER456|36643-5^CHEST XRAY^LN|||202605311231\r"

// multiOrderORM carries two ORC+OBR groups — the fail-closed case.
const multiOrderORM = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230||ORM^O01|MSG00002|P|2.4\r" +
	"ORC|NW|P1|F1||||||202605311230\r" +
	"OBR|1|P1|F1|AAA^A^LN\r" +
	"ORC|NW|P2|F2||||||202605311230\r" +
	"OBR|1|P2|F2|BBB^B^LN\r"

func TestORMToServiceRequestR5(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}

	sr, report, err := ORMToServiceRequestR5(msg)
	if err != nil {
		t.Fatalf("ORMToServiceRequestR5: %v", err)
	}
	if sr == nil {
		t.Fatal("ServiceRequest is nil")
	}

	// ORC-2 (Placer) and ORC-3 (Filler) become the identifiers.
	wantIDs := map[string]bool{"PLACER123": false, "FILLER456": false}
	for _, id := range sr.Identifier {
		if id.Value != nil {
			if _, ok := wantIDs[*id.Value]; ok {
				wantIDs[*id.Value] = true
			}
		}
	}
	for v, seen := range wantIDs {
		if !seen {
			t.Errorf("identifier %q not present; got %d identifiers", v, len(sr.Identifier))
		}
	}

	// ORC-1 NW maps to active.
	if sr.Status == nil || *sr.Status != r5.RequestStatusActive {
		t.Errorf("Status = %v, want active", sr.Status)
	}

	// intent is defaulted to order and recorded in Report.Defaulted.
	if sr.Intent == nil || *sr.Intent != r5.RequestIntentOrder {
		t.Errorf("Intent = %v, want order", sr.Intent)
	}
	if !hasDefault(report, "ServiceRequest.intent", "order") {
		t.Errorf("Report.Defaulted does not record the intent default: %+v", report.Defaulted)
	}

	// OBR-4 becomes the code as a CodeableReference.Concept (R5).
	if sr.Code == nil || sr.Code.Concept == nil || len(sr.Code.Concept.Coding) == 0 {
		t.Fatalf("Code not populated: %+v", sr.Code)
	}
	gotCode := sr.Code.Concept.Coding[0].Code
	if gotCode == nil || *gotCode != "36643-5" {
		t.Errorf("Code.code = %v, want 36643-5", gotCode)
	}

	// ORC-9 (202605311230, minute precision, no timezone offset) becomes authoredOn
	// as a FHIR date. FHIR forbids a timezone-less time, and the source supplied no
	// offset, so the time is dropped (recorded) rather than fabricated.
	if sr.AuthoredOn == nil || *sr.AuthoredOn != "2026-05-31" {
		t.Errorf("AuthoredOn = %v, want 2026-05-31 (time dropped for lack of offset)", sr.AuthoredOn)
	}
	if !hasDroppedContaining(report, "ServiceRequest.authoredOn") {
		t.Errorf("Report.Dropped does not record the dropped authoredOn time: %+v", report.Dropped)
	}

	// PID-3 becomes a logical subject reference — identifier only, never a URL.
	if sr.Subject == nil {
		t.Fatal("Subject is nil; PID-3 identity should be carried logically")
	}
	if sr.Subject.Reference != nil {
		t.Errorf("Subject.Reference = %q, want nil (identity rule: never a URL)", *sr.Subject.Reference)
	}
	if sr.Subject.Identifier == nil || sr.Subject.Identifier.Value == nil ||
		*sr.Subject.Identifier.Value != "555-44-4444" {
		t.Errorf("Subject.Identifier.Value = %v, want 555-44-4444", sr.Subject.Identifier)
	}
}

// TestORMToServiceRequestR5MultiOrderFailsClosed is the fail-closed regression:
// a multi-order ORM is rejected with ErrUnsupportedSource (the v1 single-order
// limit), never a partial result mapping only the first order.
func TestORMToServiceRequestR5MultiOrderFailsClosed(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(multiOrderORM))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}

	sr, report, err := ORMToServiceRequestR5(msg)
	if !errors.Is(err, ErrUnsupportedSource) {
		t.Fatalf("error = %v, want ErrUnsupportedSource", err)
	}
	if sr != nil {
		t.Errorf("ServiceRequest = %+v, want nil on a fail-closed reject", sr)
	}
	if report != nil {
		t.Errorf("Report = %+v, want nil on a fail-closed reject", report)
	}
}

// TestORMToServiceRequestR5RejectsNonOrder rejects a message that is not an ORM.
func TestORMToServiceRequestR5RejectsNonOrder(t *testing.T) {
	const oru = "MSH|^~\\&|A|B|C|D|202605311230||ORU^R01|M1|P|2.4\r"
	msg, err := hl7v2.Parse([]byte(oru))
	if err != nil {
		t.Fatalf("parse ORU: %v", err)
	}
	if _, _, err := ORMToServiceRequestR5(msg); !errors.Is(err, ErrUnsupportedSource) {
		t.Fatalf("error = %v, want ErrUnsupportedSource", err)
	}
}

// TestORMToServiceRequestR5WithSubjectUsesExplicitReference confirms WithSubjectR5
// overrides the PID-3 logical identity.
func TestORMToServiceRequestR5WithSubjectUsesExplicitReference(t *testing.T) {
	msg, _ := hl7v2.Parse([]byte(canonicalORM))
	want := "Patient/pat-42"
	ref := r5.Reference{Reference: &want}

	sr, _, err := ORMToServiceRequestR5(msg, WithSubjectR5(ref))
	if err != nil {
		t.Fatalf("ORMToServiceRequestR5: %v", err)
	}
	if sr.Subject == nil || sr.Subject.Reference == nil || *sr.Subject.Reference != want {
		t.Errorf("Subject.Reference = %v, want %q", sr.Subject, want)
	}
}

// TestORMToServiceRequestR5RejectsUnsupportedTrigger rejects an ORM with a
// trigger event outside the v1 scope (ORM^O01 / OMG^O19), fail-closed.
func TestORMToServiceRequestR5RejectsUnsupportedTrigger(t *testing.T) {
	const ormR01 = "MSH|^~\\&|A|B|C|D|202605311230||ORM^R01|M1|P|2.4\r" +
		"ORC|NW|P1|F1\r" +
		"OBR|1|P1|F1|AAA^A^LN\r"
	msg, err := hl7v2.Parse([]byte(ormR01))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}
	if _, _, err := ORMToServiceRequestR5(msg); !errors.Is(err, ErrUnsupportedSource) {
		t.Fatalf("error = %v, want ErrUnsupportedSource for an ORM^R01 trigger", err)
	}
}

// TestORMToServiceRequestR5BroadenedMapping covers the priority, encounter,
// requester, reason, and occurrence elements, and confirms the produced
// ServiceRequest validates by construction.
func TestORMToServiceRequestR5BroadenedMapping(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(richORM))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}

	sr, _, err := ORMToServiceRequestR5(msg)
	if err != nil {
		t.Fatalf("ORMToServiceRequestR5: %v", err)
	}

	// ORC-7 component 6 "R" maps to routine.
	if sr.Priority == nil || *sr.Priority != r5.RequestPriorityRoutine {
		t.Errorf("Priority = %v, want routine", sr.Priority)
	}

	// PV1-19 Visit Number becomes a logical encounter reference (never a URL).
	if sr.Encounter == nil {
		t.Fatal("Encounter is nil; PV1-19 should map to a logical encounter reference")
	}
	if sr.Encounter.Reference != nil {
		t.Errorf("Encounter.Reference = %q, want nil (identity rule)", *sr.Encounter.Reference)
	}
	if sr.Encounter.Identifier == nil || sr.Encounter.Identifier.Value == nil ||
		*sr.Encounter.Identifier.Value != "VISIT-9001" {
		t.Errorf("Encounter.Identifier.Value = %v, want VISIT-9001", sr.Encounter.Identifier)
	}

	// ORC-12 Ordering Provider becomes the requester: identifier + display name,
	// never a fabricated URL.
	if sr.Requester == nil {
		t.Fatal("Requester is nil; ORC-12 should map to requester")
	}
	if sr.Requester.Reference != nil {
		t.Errorf("Requester.Reference = %q, want nil (identity rule)", *sr.Requester.Reference)
	}
	if sr.Requester.Identifier == nil || sr.Requester.Identifier.Value == nil ||
		*sr.Requester.Identifier.Value != "1234" {
		t.Errorf("Requester.Identifier.Value = %v, want 1234", sr.Requester.Identifier)
	}
	if sr.Requester.Display == nil || *sr.Requester.Display != "WELBY^MARCUS" {
		t.Errorf("Requester.Display = %v, want WELBY^MARCUS", sr.Requester.Display)
	}

	// OBR-31 Reason for Study becomes a CodeableReference concept.
	if len(sr.Reason) != 1 || sr.Reason[0].Concept == nil ||
		len(sr.Reason[0].Concept.Coding) == 0 || sr.Reason[0].Concept.Coding[0].Code == nil ||
		*sr.Reason[0].Concept.Coding[0].Code != "R07.9" {
		t.Errorf("Reason = %+v, want one concept coded R07.9", sr.Reason)
	}

	// OBR-6 Requested Date/Time (with offset) becomes occurrenceDateTime.
	occ, ok := sr.Occurrence()
	if !ok {
		t.Fatal("Occurrence is unset; OBR-6 should map to occurrenceDateTime")
	}
	if dt, isDT := occ.(r5.FHIRDateTime); !isDT || string(dt) != "2026-05-31T12:31:00-05:00" {
		t.Errorf("occurrenceDateTime = %v, want 2026-05-31T12:31:00-05:00", occ)
	}

	if oo := r5.Validate(sr); oo.HasErrors() {
		t.Errorf("ServiceRequest fails validation: %+v", oo.Issue)
	}
}

// TestORMToServiceRequestR5UnknownPriorityDropped confirms an unrecognised HL7
// priority code is not emitted as an invalid FHIR code (the priority binding is
// required); the value is dropped and recorded instead, named by locus only.
func TestORMToServiceRequestR5UnknownPriorityDropped(t *testing.T) {
	const orm = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230-0500||ORM^O01|MSGZ|P|2.4\r" +
		"ORC|NW|P1|F1||||^^^^^Z||202605311230-0500\r" +
		"OBR|1|P1|F1|AAA^A^LN\r"
	msg, err := hl7v2.Parse([]byte(orm))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}
	sr, report, err := ORMToServiceRequestR5(msg)
	if err != nil {
		t.Fatalf("ORMToServiceRequestR5: %v", err)
	}
	if sr.Priority != nil {
		t.Errorf("Priority = %v, want nil for an unrecognised HL7 priority", sr.Priority)
	}
	if !hasDroppedContaining(report, "priority") {
		t.Errorf("Report.Dropped does not record the unmapped priority: %+v", report.Dropped)
	}
	// The drop names the locus only — never the raw clinical value.
	for _, d := range report.Dropped {
		if strings.Contains(d.Source, "Z") || strings.Contains(d.Reason, "Z") {
			t.Errorf("Report leaks a raw priority value: %+v", d)
		}
	}
}

// TestORMToServiceRequestR5StatPriority confirms the stat/asap priority codes map.
func TestORMToServiceRequestR5StatPriority(t *testing.T) {
	cases := []struct {
		code string
		want r5.RequestPriority
	}{
		{"S", r5.RequestPriorityStat},
		{"A", r5.RequestPriorityAsap},
		{"R", r5.RequestPriorityRoutine},
		{"P", r5.RequestPriorityRoutine}, // preferred -> routine per convert.md
	}
	for _, c := range cases {
		orm := "MSH|^~\\&|R|H|P|H|202605311230-0500||ORM^O01|M|P|2.4\r" +
			"ORC|NW|P1|F1||||^^^^^" + c.code + "||202605311230-0500\r" +
			"OBR|1|P1|F1|AAA^A^LN\r"
		msg, err := hl7v2.Parse([]byte(orm))
		if err != nil {
			t.Fatalf("parse ORM(%s): %v", c.code, err)
		}
		sr, _, err := ORMToServiceRequestR5(msg)
		if err != nil {
			t.Fatalf("ORMToServiceRequestR5(%s): %v", c.code, err)
		}
		if sr.Priority == nil || *sr.Priority != c.want {
			t.Errorf("priority(%s) = %v, want %v", c.code, sr.Priority, c.want)
		}
	}
}

// hasDroppedContaining reports whether any dropped field's Source contains sub.
func hasDroppedContaining(r *Report, sub string) bool {
	for _, d := range r.Dropped {
		if strings.Contains(d.Source, sub) {
			return true
		}
	}
	return false
}

// hasDefault reports whether the report records a default for target with value.
func hasDefault(r *Report, target, value string) bool {
	for _, d := range r.Defaulted {
		if d.Target == target && d.Value == value {
			return true
		}
	}
	return false
}
