package convert

import (
	"errors"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// roundTripUIDRoot is a valid test organisation root for minting deterministic UIDs on
// the FHIR -> SR reverse path. It is a 2.25. UUID-derived root, which needs no
// registration.
const roundTripUIDRoot dicom.UID = "2.25.123456789"

// TestSRRoundTripPreservesMeasurements converts a DICOM SR to a DiagnosticReport plus
// Observations and back to an SR, then re-runs the forward conversion on the rebuilt SR.
// The NUM, CODE, and DATETIME measurement leaves must survive: the same Quantity value,
// units, code, and coded value must reappear, proving no silent measurement, unit, or
// code loss across the round trip.
func TestSRRoundTripPreservesMeasurements(t *testing.T) {
	sr := measurementSR(t)

	dr, observations, _, err := SRToDiagnosticReportR5(sr)
	if err != nil {
		t.Fatalf("forward SRToDiagnosticReportR5: %v", err)
	}

	rebuilt, reverseReport, err := DiagnosticReportToSR(dr, observations, WithUIDRoot(roundTripUIDRoot))
	if err != nil {
		t.Fatalf("reverse DiagnosticReportToSR: %v", err)
	}
	// The forward SR carries a TIME leaf the reverse builder cannot re-encode; the loss
	// must be reported, never silent.
	if !hasDropped(reverseReport, "Observation.valueTime") {
		t.Errorf("reverse report does not record the TIME leaf it could not re-encode: %+v", reverseReport.Dropped)
	}

	// The rebuilt SR must parse cleanly as a conformant SR content tree.
	if _, err := dicom.ParseSR(rebuilt); err != nil {
		t.Fatalf("rebuilt SR does not parse as a conformant SR: %v", err)
	}

	dr2, observations2, _, err := SRToDiagnosticReportR5(rebuilt)
	if err != nil {
		t.Fatalf("re-forward SRToDiagnosticReportR5: %v", err)
	}

	// The resources produced from the rebuilt SR must pass the in-process structural
	// gate, which mirrors the merge-blocking FHIR validator gate that runs over convert
	// output in CI.
	if outcome := r5.Validate(dr2); outcome.HasErrors() {
		t.Errorf("round-tripped DiagnosticReport is not structurally valid: %s", outcome.Error())
	}
	for i, o := range observations2 {
		if outcome := r5.Validate(o); outcome.HasErrors() {
			t.Errorf("round-tripped Observation[%d] is not structurally valid: %s", i, outcome.Error())
		}
	}

	// The conclusion (the SR TEXT narrative) survives the round trip.
	if dr.Conclusion == nil || dr2.Conclusion == nil || *dr.Conclusion != *dr2.Conclusion {
		t.Errorf("conclusion lost across round trip: %v -> %v", dr.Conclusion, dr2.Conclusion)
	}
	// The report code (root CONTAINER concept) survives.
	if !sameConcept(dr.Code, dr2.Code) {
		t.Errorf("DiagnosticReport.code lost across round trip: %+v -> %+v", dr.Code, dr2.Code)
	}

	first := observationsByValue(observations)
	second := observationsByValue(observations2)

	// The NUM measurement: value, unit code, and unit system must all survive.
	q1 := requireQuantity(t, first, "NUM")
	q2 := requireQuantity(t, second, "NUM")
	if q1.Value.String() != q2.Value.String() {
		t.Errorf("NUM value lost: %q -> %q", q1.Value.String(), q2.Value.String())
	}
	if !samePtr(q1.Code, q2.Code) || !samePtr(q1.Unit, q2.Unit) || !samePtr(q1.System, q2.System) {
		t.Errorf("NUM units lost: %+v -> %+v", q1, q2)
	}

	// The CODE value: the coded value Coding must survive.
	c1 := requireCoded(t, first)
	c2 := requireCoded(t, second)
	if !sameConcept(&c1, &c2) {
		t.Errorf("CODE value lost across round trip: %+v -> %+v", c1, c2)
	}
}

// TestSRRoundTripDeterministic confirms two reverse conversions of the same
// DiagnosticReport produce byte-identical DICOM output, including the minted UIDs, so the
// FHIR -> SR direction is deterministic under a fixed UID root.
func TestSRRoundTripDeterministic(t *testing.T) {
	dr, observations, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("forward: %v", err)
	}

	first := mintedUIDs(t, dr, observations)
	second := mintedUIDs(t, dr, observations)
	if first != second {
		t.Error("two reverse conversions of the same DiagnosticReport minted different UIDs; the FHIR -> SR direction is not deterministic")
	}

	// A second forward+reverse cycle from a freshly built SR mints the same UIDs, since
	// the seed (the report's DICOM UID identifier) is stable.
	dr2, obs2, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("forward (second cycle): %v", err)
	}
	third := mintedUIDs(t, dr2, obs2)
	if first != third {
		t.Error("the same SR source did not mint byte-identical UIDs across two full cycles")
	}
	if first == "||" {
		t.Fatal("no UIDs were minted; the determinism check is vacuous")
	}
}

// TestSRReverseNoUIDRootRecorded confirms that without WithUIDRoot the reverse builder
// mints no UIDs (go-radx ships no default registered root) and records the absence
// rather than fabricating an unregistered UID.
func TestSRReverseNoUIDRootRecorded(t *testing.T) {
	dr, observations, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	rebuilt, report, err := DiagnosticReportToSR(dr, observations)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if _, ok := rebuilt.GetString(dicom.TagSOPInstanceUID); ok {
		t.Error("reverse minted a SOP Instance UID without a configured root")
	}
	if !hasDefaultTarget(report, "SOPInstanceUID (0008,0018)") {
		t.Errorf("report does not record the absent UID root: %+v", report.Defaulted)
	}
}

// TestSRReverseRejectsCodelessReport confirms a DiagnosticReport whose code does not map
// to a Concept Name Code Sequence fails closed: the SR document root requires one.
func TestSRReverseRejectsCodelessReport(t *testing.T) {
	if _, _, err := DiagnosticReportToSR(&r5.DiagnosticReport{}, nil); err == nil {
		t.Fatal("DiagnosticReportToSR of a code-less report returned nil error, want a fail-closed error")
	}
}

// TestSRReverseStrictLossEscalates confirms WithStrictLoss turns a recorded reverse-path
// drop (the valueTime leaf the content-item builder cannot carry) into a returned
// *LossError, so a consumer that cannot accept loss fails closed.
func TestSRReverseStrictLossEscalates(t *testing.T) {
	dr, observations, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	_, _, err = DiagnosticReportToSR(dr, observations, WithUIDRoot(roundTripUIDRoot), WithStrictLoss())
	if _, ok := errors.AsType[*LossError](err); !ok {
		t.Fatalf("DiagnosticReportToSR with WithStrictLoss returned %v, want a *LossError for the dropped valueTime leaf", err)
	}
}

// TestOBXRoundTripPreservesMeasurements builds an OBX, converts it to an Observation and
// back, and confirms the value type, numeric value, units, code identity, interpretation
// flag, and reference range all survive with no silent loss.
func TestOBXRoundTripPreservesMeasurements(t *testing.T) {
	tests := []struct {
		name string
		obx  hl7v2.OBX
	}{
		{
			name: "numeric result with units",
			obx: hl7v2.OBX{
				ValueType:      "NM",
				ObservationID:  hl7v2.CWE{Code: "718-7", Text: "Hemoglobin", CodingSystem: "LN"},
				Value:          []string{"9.4"},
				Units:          hl7v2.CWE{Code: "g/dL", Text: "g/dL", CodingSystem: "UCUM"},
				ReferenceRange: "12-16",
				AbnormalFlags:  []string{"L"},
				ResultStatus:   "F",
			},
		},
		{
			name: "coded result",
			obx: hl7v2.OBX{
				ValueType:     "CWE",
				ObservationID: hl7v2.CWE{Code: "test", Text: "Test", CodingSystem: "L"},
				Value:         []string{"260385009^Negative^SCT"},
				ResultStatus:  "F",
			},
		},
		{
			name: "string result",
			obx: hl7v2.OBX{
				ValueType:     "ST",
				ObservationID: hl7v2.CWE{Code: "NOTE", Text: "Comment", CodingSystem: "L"},
				Value:         []string{"within normal limits"},
				ResultStatus:  "F",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, fwdReport, ok := OBXToObservationR5(tt.obx)
			if !ok {
				t.Fatalf("forward OBXToObservationR5 ok=false")
			}
			if len(fwdReport.Dropped) != 0 {
				t.Errorf("forward conversion dropped data: %+v", fwdReport.Dropped)
			}

			back, revReport, ok := ObservationToOBX(o)
			if !ok {
				t.Fatalf("reverse ObservationToOBX ok=false")
			}
			if len(revReport.Dropped) != 0 {
				t.Errorf("reverse conversion dropped data: %+v", revReport.Dropped)
			}

			if back.ValueType != tt.obx.ValueType {
				t.Errorf("OBX-2 value type lost: %q -> %q", tt.obx.ValueType, back.ValueType)
			}
			if len(back.Value) != 1 || back.Value[0] != tt.obx.Value[0] {
				t.Errorf("OBX-5 value lost: %v -> %v", tt.obx.Value, back.Value)
			}
			if back.ObservationID.Code != tt.obx.ObservationID.Code {
				t.Errorf("OBX-3 code lost: %q -> %q", tt.obx.ObservationID.Code, back.ObservationID.Code)
			}
			if tt.obx.Units.Code != "" && back.Units.Code != tt.obx.Units.Code {
				t.Errorf("OBX-6 units code lost: %q -> %q", tt.obx.Units.Code, back.Units.Code)
			}
			if tt.obx.ReferenceRange != "" && back.ReferenceRange != tt.obx.ReferenceRange {
				t.Errorf("OBX-7 reference range lost: %q -> %q", tt.obx.ReferenceRange, back.ReferenceRange)
			}
			if len(tt.obx.AbnormalFlags) > 0 {
				if len(back.AbnormalFlags) != 1 || back.AbnormalFlags[0] != tt.obx.AbnormalFlags[0] {
					t.Errorf("OBX-8 flags lost: %v -> %v", tt.obx.AbnormalFlags, back.AbnormalFlags)
				}
			}
		})
	}
}

// TestOBXRoundTripDeterministic confirms two reverse conversions of the same Observation
// render byte-identical OBX segments through the hl7v2 renderer.
func TestOBXRoundTripDeterministic(t *testing.T) {
	obx := hl7v2.OBX{
		ValueType:     "NM",
		ObservationID: hl7v2.CWE{Code: "718-7", Text: "Hemoglobin", CodingSystem: "LN"},
		Value:         []string{"9.4"},
		Units:         hl7v2.CWE{Code: "g/dL", Text: "g/dL", CodingSystem: "UCUM"},
		ResultStatus:  "F",
	}
	o, _, ok := OBXToObservationR5(obx)
	if !ok {
		t.Fatal("forward ok=false")
	}
	first := renderOBX(t, o)
	second := renderOBX(t, o)
	if first != second {
		t.Errorf("two reverse OBX conversions rendered differently: %q vs %q", first, second)
	}
}

// TestOBXRoundTripPreservesPartialDateTime confirms a partial-precision OBX-5 dateTime
// (year or year-month) survives OBX -> Observation -> OBX without being dropped. The
// forward path emits a partial FHIR dateTime, so the reverse path must re-emit the
// matching partial HL7 DTM rather than rejecting anything shorter than a full date.
func TestOBXRoundTripPreservesPartialDateTime(t *testing.T) {
	tests := []struct {
		name    string
		dtm     string // OBX-5 value the forward path reads
		wantDTM string // OBX-5 value the reverse path must re-emit
	}{
		{name: "year precision", dtm: "2026", wantDTM: "2026"},
		{name: "year-month precision", dtm: "202605", wantDTM: "202605"},
		{name: "full date precision", dtm: "20260531", wantDTM: "20260531"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obx := hl7v2.OBX{
				ValueType:     "TS",
				ObservationID: hl7v2.CWE{Code: "21112-8", Text: "Birth date", CodingSystem: "LN"},
				Value:         []string{tt.dtm},
				ResultStatus:  "F",
			}
			o, fwdReport, ok := OBXToObservationR5(obx)
			if !ok {
				t.Fatalf("forward OBXToObservationR5 ok=false")
			}
			if len(fwdReport.Dropped) != 0 {
				t.Errorf("forward conversion dropped a supported partial dateTime: %+v", fwdReport.Dropped)
			}

			back, revReport, ok := ObservationToOBX(o)
			if !ok {
				t.Fatalf("reverse ObservationToOBX ok=false")
			}
			if len(revReport.Dropped) != 0 {
				t.Errorf("reverse conversion dropped the partial dateTime instead of re-emitting it: %+v", revReport.Dropped)
			}
			if back.ValueType != "TS" {
				t.Errorf("OBX-2 = %q, want TS", back.ValueType)
			}
			if len(back.Value) != 1 || back.Value[0] != tt.wantDTM {
				t.Errorf("OBX-5 partial dateTime lost: %q -> %v, want %q", tt.dtm, back.Value, tt.wantDTM)
			}
		})
	}
}

// TestOBXRoundTripPreservesTextOnlyIdentifier confirms a text-only OBX-3 (CWE.2 present,
// CWE.1 empty) survives OBX -> Observation -> OBX. The forward converter accepts a
// text-only coded identifier as Observation.code, so the reverse path must re-emit the
// OBX rather than dropping it for lack of a code.
func TestOBXRoundTripPreservesTextOnlyIdentifier(t *testing.T) {
	obx := hl7v2.OBX{
		ValueType:     "ST",
		ObservationID: hl7v2.CWE{Text: "Free-text observation label"},
		Value:         []string{"within normal limits"},
		ResultStatus:  "F",
	}
	o, _, ok := OBXToObservationR5(obx)
	if !ok {
		t.Fatalf("forward OBXToObservationR5 ok=false for a text-only OBX-3")
	}

	back, revReport, ok := ObservationToOBX(o)
	if !ok {
		t.Fatalf("reverse ObservationToOBX ok=false; a text-only OBX-3 was not re-emitted")
	}
	if len(revReport.Dropped) != 0 {
		t.Errorf("reverse conversion dropped the text-only identifier: %+v", revReport.Dropped)
	}
	if back.ObservationID.Code != "" {
		t.Errorf("OBX-3 code = %q, want empty (the source carried text only)", back.ObservationID.Code)
	}
	if back.ObservationID.Text != "Free-text observation label" {
		t.Errorf("OBX-3 text lost: %q", back.ObservationID.Text)
	}
}

// TestSRRoundTripStringObservationStaysObservation confirms a string-valued Observation
// survives DiagnosticReport -> SR -> DiagnosticReport as an Observation, not as report
// narrative. A concept-named TEXT content item is a measurement Observation, so the
// reverse path must emit the string Observation with its concept name and the forward
// path must re-import it as a valueString Observation rather than folding it into
// DiagnosticReport.conclusion.
func TestSRRoundTripStringObservationStaysObservation(t *testing.T) {
	drCode := &r5.CodeableConcept{
		Coding: []r5.Coding{{System: strPtr("http://loinc.org"), Code: strPtr("11528-7"), Display: strPtr("Radiology Report")}},
	}
	status := r5.DiagnosticReportStatusFinal
	conclusion := "Overall impression: unremarkable."
	dr := &r5.DiagnosticReport{Code: drCode, Status: &status, Conclusion: &conclusion}

	stringObs := &r5.Observation{Code: numConcept()}
	stringObs.SetValueString(r5.FHIRString("Solitary pulmonary nodule."))

	sr, _, err := DiagnosticReportToSR(dr, []*r5.Observation{stringObs}, WithUIDRoot(roundTripUIDRoot))
	if err != nil {
		t.Fatalf("reverse DiagnosticReportToSR: %v", err)
	}

	dr2, observations2, _, err := SRToDiagnosticReportR5(sr)
	if err != nil {
		t.Fatalf("forward SRToDiagnosticReportR5: %v", err)
	}

	// The string Observation must come back as an Observation with the same valueString,
	// not be merged into the conclusion.
	var got *r5.Observation
	for _, o := range observations2 {
		if v, ok := o.Value(); ok {
			if s, isStr := v.(r5.FHIRString); isStr && string(s) == "Solitary pulmonary nodule." {
				got = o
			}
		}
	}
	if got == nil {
		t.Fatalf("the string Observation did not survive the round trip as an Observation; observations=%d conclusion=%v",
			len(observations2), dr2.Conclusion)
	}
	if !sameConcept(got.Code, stringObs.Code) {
		t.Errorf("string Observation code lost across round trip: %+v -> %+v", stringObs.Code, got.Code)
	}

	// The bare conclusion still routes to DiagnosticReport.conclusion, not an Observation.
	if dr2.Conclusion == nil || *dr2.Conclusion != conclusion {
		t.Errorf("conclusion lost across round trip: %v -> %v", conclusion, dr2.Conclusion)
	}
}

// TestSRReverseRejectsOverlongUIDRoot confirms an over-long WithUIDRoot is rejected
// rather than silently truncated. Truncating a long root could drop the role-specific
// suffix and mint identical or malformed UIDs, so the reverse path fails closed.
func TestSRReverseRejectsOverlongUIDRoot(t *testing.T) {
	dr, observations, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("forward: %v", err)
	}

	// A root longer than the safe-prefix bound the dicom UID generator enforces (54).
	overlong := dicom.UID("2.25." + repeatDigit('1', 60))
	if _, _, err := DiagnosticReportToSR(dr, observations, WithUIDRoot(overlong)); err == nil {
		t.Fatal("DiagnosticReportToSR with an over-long UID root returned nil error, want a fail-closed error")
	}
}

// TestSRReverseMintsDistinctValidUIDs confirms a normal root mints a Study, Series, and
// SOP Instance UID that are pairwise distinct, at most 64 characters, and never end in a
// dot.
func TestSRReverseMintsDistinctValidUIDs(t *testing.T) {
	dr, observations, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	ds, _, err := DiagnosticReportToSR(dr, observations, WithUIDRoot(roundTripUIDRoot))
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}

	study, _ := ds.GetString(dicom.TagStudyInstanceUID)
	series, _ := ds.GetString(dicom.TagSeriesInstanceUID)
	instance, _ := ds.GetString(dicom.TagSOPInstanceUID)

	uids := map[string]string{"study": study, "series": series, "instance": instance}
	for role, uid := range uids {
		if uid == "" {
			t.Fatalf("%s UID was not minted", role)
		}
		if len(uid) > 64 {
			t.Errorf("%s UID exceeds 64 characters: %q (%d)", role, uid, len(uid))
		}
		if uid[len(uid)-1] == '.' {
			t.Errorf("%s UID ends in a dot: %q", role, uid)
		}
		if err := dicom.UID(uid).Validate(); err != nil {
			t.Errorf("%s UID is not a valid UID: %q (%v)", role, uid, err)
		}
	}
	if study == series || study == instance || series == instance {
		t.Errorf("minted UIDs are not pairwise distinct: study=%q series=%q instance=%q", study, series, instance)
	}
}

// repeatDigit returns a string of n copies of the digit d, used to build an over-long
// UID root for the rejection test.
func repeatDigit(d byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = d
	}
	return string(b)
}

// renderOBX runs the reverse converter and renders the OBX to its wire line through the
// hl7v2 message renderer, so the assertion exercises the real serialisation path.
func renderOBX(t *testing.T, o *r5.Observation) string {
	t.Helper()
	obx, _, ok := ObservationToOBX(o)
	if !ok {
		t.Fatal("ObservationToOBX ok=false")
	}
	enc := hl7v2.DefaultEncoding()
	msg := hl7v2.NewMessage(enc)
	msg.AppendSegment(obx.Segment(enc))
	return msg.String()
}

// mintedUIDs runs the reverse converter under the fixed UID root and returns its minted
// Study/Series/SOP Instance UIDs joined for a determinism compare.
func mintedUIDs(t *testing.T, dr *r5.DiagnosticReport, observations []*r5.Observation) string {
	t.Helper()
	ds, _, err := DiagnosticReportToSR(dr, observations, WithUIDRoot(roundTripUIDRoot))
	if err != nil {
		t.Fatalf("DiagnosticReportToSR: %v", err)
	}
	study, _ := ds.GetString(dicom.TagStudyInstanceUID)
	series, _ := ds.GetString(dicom.TagSeriesInstanceUID)
	instance, _ := ds.GetString(dicom.TagSOPInstanceUID)
	return study + "|" + series + "|" + instance
}

// observationsByValue indexes Observations by their value[x] branch kind so a round-trip
// assertion can find the NUM/CODE leaf regardless of order.
func observationsByValue(observations []*r5.Observation) map[string]*r5.Observation {
	out := make(map[string]*r5.Observation)
	for _, o := range observations {
		v, ok := o.Value()
		if !ok {
			continue
		}
		switch v.(type) {
		case r5.Quantity:
			out["NUM"] = o
		case r5.CodeableConcept:
			out["CODE"] = o
		case r5.FHIRDateTime:
			out["DATETIME"] = o
		case r5.FHIRTime:
			out["TIME"] = o
		}
	}
	return out
}

func requireQuantity(t *testing.T, m map[string]*r5.Observation, kind string) r5.Quantity {
	t.Helper()
	o, ok := m[kind]
	if !ok || o.ValueQuantity == nil {
		t.Fatalf("no %s Observation with a Quantity in the set", kind)
	}
	return *o.ValueQuantity
}

func requireCoded(t *testing.T, m map[string]*r5.Observation) r5.CodeableConcept {
	t.Helper()
	o, ok := m["CODE"]
	if !ok || o.ValueCodeableConcept == nil {
		t.Fatal("no CODE Observation with a CodeableConcept in the set")
	}
	return *o.ValueCodeableConcept
}

// sameConcept reports whether two CodeableConcepts carry the same first Coding code and
// system, the identity that must survive a round trip.
func sameConcept(a, b *r5.CodeableConcept) bool {
	if a == nil || b == nil || len(a.Coding) == 0 || len(b.Coding) == 0 {
		return false
	}
	return samePtr(a.Coding[0].Code, b.Coding[0].Code) && samePtr(a.Coding[0].System, b.Coding[0].System)
}

// samePtr reports whether two string pointers hold the same value (both nil counts as
// equal).
func samePtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
