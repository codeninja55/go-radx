package convert

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// canonicalORU is a representative ORU^R01 with header, patient, an observation
// request (OBR), and a mix of OBX value types: a numeric result with units, a
// reference range, and an abnormal flag (NM); a coded result (CWE); and a free-
// text narrative (TX). The OBR observation date/time carries a -0500 offset so it
// renders as a full FHIR dateTime.
const canonicalORU = "MSH|^~\\&|LIS|HOSP|EMR|HOSP|202605311230-0500||ORU^R01|MSGORU1|P|2.4\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
	"OBR|1|PLACER123|FILLER456|24331-1^LIPID PANEL^LN|||202605311231-0500\r" +
	"OBX|1|NM|2093-3^CHOLESTEROL^LN||242|mg/dL^mg/dL^UCUM|0-200|H|||F\r" +
	"OBX|2|CWE|664-3^SPECIMEN APPEARANCE^LN||CLEAR^Clear^L|||N|||F\r" +
	"OBX|3|TX|8251-1^NARRATIVE^LN||Sample mildly lipemic.|||||\r"

func TestORUToDiagnosticReportR5(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(canonicalORU))
	if err != nil {
		t.Fatalf("parse ORU: %v", err)
	}

	dr, obs, report, err := ORUToDiagnosticReportR5(msg)
	if err != nil {
		t.Fatalf("ORUToDiagnosticReportR5: %v", err)
	}
	if dr == nil {
		t.Fatal("DiagnosticReport is nil")
	}
	_ = report

	// OBR-4 becomes the required DiagnosticReport.code.
	if dr.Code == nil || len(dr.Code.Coding) == 0 || dr.Code.Coding[0].Code == nil ||
		*dr.Code.Coding[0].Code != "24331-1" {
		t.Errorf("Code = %+v, want one coding 24331-1", dr.Code)
	}

	// OBR-25 absent -> status defaulted to final and recorded.
	if dr.Status == nil || *dr.Status != r5.DiagnosticReportStatusFinal {
		t.Errorf("Status = %v, want final", dr.Status)
	}

	// PID-3 becomes a logical subject reference — identifier only, never a URL.
	if dr.Subject == nil || dr.Subject.Reference != nil {
		t.Fatalf("Subject = %+v, want a logical identifier reference (no URL)", dr.Subject)
	}
	if dr.Subject.Identifier == nil || dr.Subject.Identifier.Value == nil ||
		*dr.Subject.Identifier.Value != "555-44-4444" {
		t.Errorf("Subject.Identifier.Value = %v, want 555-44-4444", dr.Subject.Identifier)
	}

	// OBR-7 with an offset becomes effectiveDateTime.
	if dr.EffectiveDateTime == nil || string(*dr.EffectiveDateTime) != "2026-05-31T12:31:00-05:00" {
		t.Errorf("EffectiveDateTime = %v, want 2026-05-31T12:31:00-05:00", dr.EffectiveDateTime)
	}

	// Three OBX rows become three Observations, each linked from result[].
	if len(obs) != 3 {
		t.Fatalf("Observations = %d, want 3", len(obs))
	}
	if len(dr.Result) != 3 {
		t.Errorf("result[] = %d links, want 3", len(dr.Result))
	}
	for i, o := range obs {
		if o.ID == nil || *o.ID == "" {
			t.Errorf("Observation[%d] has no id for the result link", i)
		}
	}

	// OBX-1 NM -> valueQuantity carrying the units and a reference range and an
	// abnormal-flag interpretation.
	num := obs[0]
	if num.ValueQuantity == nil || num.ValueQuantity.Value == nil ||
		num.ValueQuantity.Value.String() != "242" {
		t.Errorf("NM valueQuantity = %+v, want 242", num.ValueQuantity)
	}
	if num.ValueQuantity.Code == nil || *num.ValueQuantity.Code != "mg/dL" {
		t.Errorf("NM valueQuantity.code = %v, want mg/dL", num.ValueQuantity.Code)
	}
	if len(num.ReferenceRange) != 1 || num.ReferenceRange[0].Low == nil ||
		num.ReferenceRange[0].High == nil {
		t.Errorf("NM referenceRange = %+v, want one range with low and high", num.ReferenceRange)
	}
	if num.ReferenceRange[0].Low.Value == nil || num.ReferenceRange[0].Low.Value.String() != "0" {
		t.Errorf("referenceRange low = %+v, want 0", num.ReferenceRange[0].Low)
	}
	if num.ReferenceRange[0].High.Value == nil || num.ReferenceRange[0].High.Value.String() != "200" {
		t.Errorf("referenceRange high = %+v, want 200", num.ReferenceRange[0].High)
	}
	if len(num.Interpretation) != 1 || len(num.Interpretation[0].Coding) == 0 ||
		num.Interpretation[0].Coding[0].Code == nil || *num.Interpretation[0].Coding[0].Code != "H" {
		t.Errorf("NM interpretation = %+v, want one coding H", num.Interpretation)
	}

	// OBX-2 CWE -> valueCodeableConcept.
	code := obs[1]
	if code.ValueCodeableConcept == nil || len(code.ValueCodeableConcept.Coding) == 0 ||
		code.ValueCodeableConcept.Coding[0].Code == nil || *code.ValueCodeableConcept.Coding[0].Code != "CLEAR" {
		t.Errorf("CWE valueCodeableConcept = %+v, want coding CLEAR", code.ValueCodeableConcept)
	}

	// OBX-3 TX -> valueString.
	text := obs[2]
	if text.ValueString == nil || string(*text.ValueString) != "Sample mildly lipemic." {
		t.Errorf("TX valueString = %v, want the narrative text", text.ValueString)
	}

	// The DiagnosticReport and each Observation validate by construction.
	if oo := fhir.Validate(dr); oo.HasErrors() {
		t.Errorf("DiagnosticReport fails validation: %+v", oo.Issue)
	}
	for i, o := range obs {
		if oo := fhir.Validate(o); oo.HasErrors() {
			t.Errorf("Observation[%d] fails validation: %+v", i, oo.Issue)
		}
	}
}

// TestORUToDiagnosticReportR5RejectsNonResult rejects a message that is not an ORU.
func TestORUToDiagnosticReportR5RejectsNonResult(t *testing.T) {
	const orm = "MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r"
	msg, err := hl7v2.Parse([]byte(orm))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}
	if _, _, _, err := ORUToDiagnosticReportR5(msg); err == nil {
		t.Fatal("error = nil, want ErrUnsupportedSource for a non-ORU message")
	}
}

// TestORUToDiagnosticReportR5NilMessage rejects a nil message fail-closed.
func TestORUToDiagnosticReportR5NilMessage(t *testing.T) {
	if _, _, _, err := ORUToDiagnosticReportR5(nil); err == nil {
		t.Fatal("error = nil, want ErrMalformedSource for a nil message")
	}
}

// TestORUToDiagnosticReportR5NoOBR fails closed when there is no OBR to supply the
// required DiagnosticReport.code.
func TestORUToDiagnosticReportR5NoOBR(t *testing.T) {
	const oru = "MSH|^~\\&|LIS|HOSP|EMR|HOSP|202605311230||ORU^R01|M|P|2.4\r" +
		"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r"
	msg, err := hl7v2.Parse([]byte(oru))
	if err != nil {
		t.Fatalf("parse ORU: %v", err)
	}
	if _, _, _, err := ORUToDiagnosticReportR5(msg); err == nil {
		t.Fatal("error = nil, want a fail-closed reject when no OBR is present")
	}
}

// TestORUStatusMapping confirms OBR-25 result status maps to DiagnosticReport.status.
func TestORUStatusMapping(t *testing.T) {
	cases := []struct {
		obr25 string
		want  r5.DiagnosticReportStatus
	}{
		{"F", r5.DiagnosticReportStatusFinal},
		{"P", r5.DiagnosticReportStatusPreliminary},
		{"C", r5.DiagnosticReportStatusCorrected},
		{"X", r5.DiagnosticReportStatusCancelled},
	}
	for _, c := range cases {
		oru := "MSH|^~\\&|LIS|HOSP|EMR|HOSP|202605311230||ORU^R01|M|P|2.4\r" +
			"OBR|1|P1|F1|24331-1^LIPID^LN|||||||||||||||||||||" + c.obr25 + "\r" +
			"OBX|1|TX|8251-1^N^LN||text|||||\r"
		msg, err := hl7v2.Parse([]byte(oru))
		if err != nil {
			t.Fatalf("parse ORU(%s): %v", c.obr25, err)
		}
		dr, _, _, err := ORUToDiagnosticReportR5(msg)
		if err != nil {
			t.Fatalf("ORUToDiagnosticReportR5(%s): %v", c.obr25, err)
		}
		if dr.Status == nil || *dr.Status != c.want {
			t.Errorf("status(%s) = %v, want %v", c.obr25, dr.Status, c.want)
		}
	}
}

// TestOBXToObservationR5ValueTypeDispatch confirms OBX-2 governs value[x] selection.
func TestOBXToObservationR5ValueTypeDispatch(t *testing.T) {
	cases := []struct {
		valueType string
		value     string
		check     func(*testing.T, *r5.Observation)
	}{
		{"NM", "3.14", func(t *testing.T, o *r5.Observation) {
			if o.ValueQuantity == nil || o.ValueQuantity.Value == nil || o.ValueQuantity.Value.String() != "3.14" {
				t.Errorf("NM -> valueQuantity = %+v", o.ValueQuantity)
			}
		}},
		{"ST", "a string", func(t *testing.T, o *r5.Observation) {
			if o.ValueString == nil || string(*o.ValueString) != "a string" {
				t.Errorf("ST -> valueString = %v", o.ValueString)
			}
		}},
		{"FT", "formatted", func(t *testing.T, o *r5.Observation) {
			if o.ValueString == nil || string(*o.ValueString) != "formatted" {
				t.Errorf("FT -> valueString = %v", o.ValueString)
			}
		}},
		{"DT", "20260531", func(t *testing.T, o *r5.Observation) {
			if o.ValueDateTime == nil || string(*o.ValueDateTime) != "2026-05-31" {
				t.Errorf("DT -> valueDateTime = %v", o.ValueDateTime)
			}
		}},
		{"TM", "1230", func(t *testing.T, o *r5.Observation) {
			if o.ValueTime == nil || string(*o.ValueTime) != "12:30:00" {
				t.Errorf("TM -> valueTime = %v", o.ValueTime)
			}
		}},
	}
	for _, c := range cases {
		obx := hl7v2.OBX{
			ValueType:     c.valueType,
			ObservationID: hl7v2.CWE{Code: "X", Text: "test", CodingSystem: "LN"},
			Value:         []string{c.value},
		}
		o, _, ok := obxToObservationR5(obx, &Report{})
		if !ok {
			t.Fatalf("obxToObservationR5(%s) returned ok=false", c.valueType)
		}
		c.check(t, o)
		if oo := fhir.Validate(o); oo.HasErrors() {
			t.Errorf("Observation(%s) fails validation: %+v", c.valueType, oo.Issue)
		}
	}
}

// TestOBXUnknownValueTypeDropped confirms an unknown OBX-2 value type produces no
// value[x] and records the drop by locus, never the raw value.
func TestOBXUnknownValueTypeDropped(t *testing.T) {
	obx := hl7v2.OBX{
		ValueType:     "ZZ",
		ObservationID: hl7v2.CWE{Code: "X", Text: "secret-observed", CodingSystem: "LN"},
		Value:         []string{"secret-value-42"},
	}
	o, report, ok := obxToObservationR5(obx, &Report{})
	if !ok {
		t.Fatal("obxToObservationR5 returned ok=false for a coded leaf with an unknown value type")
	}
	if o.ValueQuantity != nil || o.ValueString != nil || o.ValueCodeableConcept != nil {
		t.Errorf("unknown value type set a value[x]: %+v", o)
	}
	if !hasDroppedContaining(report, "OBX-5") {
		t.Errorf("Report.Dropped does not record the unmapped OBX value: %+v", report.Dropped)
	}
	for _, d := range report.Dropped {
		if strings.Contains(d.Source+d.Reason, "secret") {
			t.Errorf("Report leaks a raw OBX value: %+v", d)
		}
	}
}
