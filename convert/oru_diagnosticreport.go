package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// observationInterpretationSystem is the FHIR code system the abnormal-flag
// interpretation (OBX-8) carries. The HL7 v2 abnormal-flag codes (Table 0078) and
// the FHIR ObservationInterpretation codes (H, L, HH, LL, N, A, AA, ...) share the
// same symbols, so the flag is carried verbatim under this system rather than
// translated.
const observationInterpretationSystem = "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"

// ORUToDiagnosticReportR5 converts an HL7 v2 observation result message (ORU^R01)
// to a FHIR R5 DiagnosticReport and the set of Observations carrying its results.
// The first OBR becomes the DiagnosticReport (its OBR-4 the required code, OBR-25
// the status, OBR-7 the effective date/time), and each OBX that follows becomes one
// Observation through OBXToObservationR5, linked from DiagnosticReport.result by an
// intra-call urn:uuid logical reference. A message that is not an ORU is rejected
// with ErrUnsupportedSource; an ORU with no OBR is rejected with ErrMalformedSource
// because DiagnosticReport.code is required and has no other source.
//
// The result links are derived deterministically from the message control ID and
// each OBX's position, so the same input yields byte-identical output. subject
// carries the PID-3 patient identity logically (Reference.identifier), or the
// WithSubjectR5 reference when supplied — never a fabricated Reference.reference URL
// (the identity rule).
func ORUToDiagnosticReportR5(msg *hl7v2.Message, opts ...Option) (*r5.DiagnosticReport, []*r5.Observation, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if msg == nil {
		return nil, nil, nil, fmt.Errorf("%w: message is nil", ErrMalformedSource)
	}

	oru, ok := hl7v2.AsORU(msg)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: MSH-9.1 is not ORU", ErrUnsupportedSource)
	}

	groups := collectResultGroups(oru)
	if len(groups) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: ORU carries no OBR for the required DiagnosticReport.code",
			ErrMalformedSource)
	}
	for i := range groups[1:] {
		// v1 maps the first result group to one DiagnosticReport; a panel split
		// across additional OBRs is recorded by locus, not silently merged.
		report.dropped(
			fmt.Sprintf("OBR (result group %d)", i+2),
			"v1 DiagnosticReport maps a single OBR; extra result groups are not mapped",
		)
	}

	group := groups[0]
	dr := &r5.DiagnosticReport{}

	// DiagnosticReport.code is required by the R5 model; OBR-4 is its only source.
	// A result group with no OBR-4 would produce an invalid resource, so fail
	// closed rather than emit a code-less report.
	code := cweConcept(group.Order.UniversalServiceID)
	if code == nil {
		return nil, nil, nil, fmt.Errorf("%w: OBR-4 UniversalServiceID is empty; DiagnosticReport.code is required",
			ErrMalformedSource)
	}
	dr.Code = code

	status := diagnosticReportStatus(msg, report)
	dr.Status = &status

	if when := hl7DateTimeToFHIR(group.Order.ObservationDateTime, report, "DiagnosticReport.effectiveDateTime"); when != "" {
		dr.SetEffectiveDateTime(r5.FHIRDateTime(when))
	}

	if subject := patientSubjectR5(cfg, msg, report, "DiagnosticReport.subject"); subject != nil {
		dr.Subject = subject
	}

	controlID := ""
	if h, hasMSH := msg.MSH(); hasMSH {
		controlID = h.ControlID
	}

	var observations []*r5.Observation
	for i := range group.Observations {
		o, obxReport, ok := obxToObservationR5(group.Observations[i], report)
		if !ok {
			// An OBX with no OBX-3 identifier has no FHIR home (Observation.code is
			// required); record the loss by locus so the OBX-5 clinical value is not
			// lost silently and strict-loss can escalate it.
			report.dropped(
				fmt.Sprintf("OBX (observation %d)", i+1),
				"OBX has no OBX-3 observation identifier for the required Observation.code; the row was not mapped",
			)
			continue
		}
		_ = obxReport
		id := deterministicObservationUUID(controlID, len(observations))
		o.ID = &id
		observations = append(observations, o)
		ref := "urn:uuid:" + id
		dr.Result = append(dr.Result, r5.Reference{Reference: &ref})
	}

	rep, err := cfg.finalize(report)
	return dr, observations, rep, err
}

// collectResultGroups materialises the ORU's result groups so the count can be
// checked before mapping (the iterator is consumed once).
func collectResultGroups(oru hl7v2.ORU) []hl7v2.ResultGroup {
	var groups []hl7v2.ResultGroup
	for g := range oru.Results() {
		groups = append(groups, g)
	}
	return groups
}

// diagnosticReportStatus maps OBR-25 Result Status (HL7 Table 0123) to a FHIR
// DiagnosticReport.status: F→final, P→preliminary, C→corrected, X→cancelled. An
// absent OBR-25 defaults to final and is recorded; an unrecognised code defaults to
// final and is recorded, because the binding is required and an out-of-set code
// would fail validation.
func diagnosticReportStatus(msg *hl7v2.Message, report *Report) r5.DiagnosticReportStatus {
	code := getField(msg, "OBR-25")
	switch code {
	case "F":
		return r5.DiagnosticReportStatusFinal
	case "P":
		return r5.DiagnosticReportStatusPreliminary
	case "C":
		return r5.DiagnosticReportStatusCorrected
	case "X":
		return r5.DiagnosticReportStatusCancelled
	case "":
		report.defaulted("DiagnosticReport.status", "final",
			"ORU has no OBR-25 result status; defaulted")
		return r5.DiagnosticReportStatusFinal
	default:
		report.defaulted("DiagnosticReport.status", "final",
			"OBR-25 result status is not in HL7 Table 0123; defaulted (the binding is required)")
		return r5.DiagnosticReportStatusFinal
	}
}

// OBXToObservationR5 maps one HL7 v2 OBX result to a FHIR R5 Observation. OBX-3
// becomes the required Observation.code, OBX-2 (ValueType) dispatches the value
// into the matching value[x], OBX-8 abnormal flags become Observation.interpretation,
// and OBX-7 a reference range. The Observation is emitted with status "final"; a
// per-result status is OBX-11, but the v1 mapping treats a reported result as final.
//
// ok is false only when the OBX has no observation identifier (OBX-3), which is the
// required Observation.code; an unmappable value type leaves value[x] unset (recorded)
// rather than rejecting the observation, so the coded identity survives.
func OBXToObservationR5(obx hl7v2.OBX, _ ...Option) (*r5.Observation, *Report, bool) {
	report := &Report{}
	o, _, ok := obxToObservationR5(obx, report)
	return o, report, ok
}

// obxToObservationR5 is the loss-aware OBX converter shared by ORUToDiagnosticReportR5
// and the exported single-leaf entry point. It records loss on the supplied report.
func obxToObservationR5(obx hl7v2.OBX, report *Report) (*r5.Observation, *Report, bool) {
	code := cweConcept(obx.ObservationID)
	if code == nil {
		return nil, report, false
	}

	o := &r5.Observation{Code: code}
	status := r5.ObservationStatusFinal
	o.Status = &status

	setObservationValue(o, obx, report)
	if interp := abnormalFlagInterpretation(obx.AbnormalFlags); len(interp) > 0 {
		o.Interpretation = interp
	}
	if rr, ok := parseReferenceRange(obx.ReferenceRange); ok {
		o.ReferenceRange = []r5.ObservationReferenceRange{rr}
	}
	return o, report, true
}

// setObservationValue dispatches the OBX value into the Observation's value[x] by
// OBX-2 ValueType: NM→valueQuantity, CE/CWE→valueCodeableConcept, TX/ST/FT→valueString,
// DT/DATE→valueDateTime, TM/TIME→valueTime. value[x] is a single choice element, so
// only the first OBX-5 repetition is mapped; any further repetitions are recorded by
// locus. An unrecognised value type or a non-empty unparseable value likewise leaves
// value[x] unset and records the loss by locus only — the raw clinical value is never
// named in the report (PRD §9.1).
func setObservationValue(o *r5.Observation, obx hl7v2.OBX, report *Report) {
	raw := firstValue(obx.Value)
	if len(obx.Value) > 1 {
		// FHIR Observation.value[x] is a single choice element; the leading OBX-5
		// repetition maps to it and the remainder have no home in this resource.
		// Record the count of unconverted repetitions by locus (never the values)
		// so the loss is reported and strict-loss can escalate it.
		report.dropped(
			fmt.Sprintf("OBX-5 ObservationValue (%d additional repetitions)", len(obx.Value)-1),
			"FHIR Observation.value[x] holds a single value; OBX-5 repetitions beyond the first were not mapped",
		)
	}
	switch obx.ValueType {
	case "NM":
		if q, ok := numericQuantity(raw, obx.Units); ok {
			o.SetValueQuantity(q)
			return
		}
		report.dropped("OBX-5 ObservationValue (NM)",
			"the numeric result value is not a valid FHIR decimal; valueQuantity was not set")
	case "CE", "CWE":
		if cc := codedValueConcept(raw); cc != nil {
			o.SetValueCodeableConcept(*cc)
			return
		}
		report.dropped("OBX-5 ObservationValue (CWE)",
			"the coded result value carried no code; valueCodeableConcept was not set")
	case "TX", "ST", "FT":
		if raw != "" {
			o.SetValueString(r5.FHIRString(raw))
			return
		}
	case "DT", "DATE", "TS":
		if when, ok := dtmDateTimeValue(raw, report); ok {
			o.SetValueDateTime(r5.FHIRDateTime(when))
			return
		}
		if raw != "" {
			report.dropped("OBX-5 ObservationValue (DT/TS)",
				"the date/time result value is not a valid FHIR dateTime; valueDateTime was not set")
		}
	case "TM", "TIME":
		if when, ok := timeValue(raw); ok {
			o.SetValueTime(r5.FHIRTime(when))
			return
		}
		if raw != "" {
			report.dropped("OBX-5 ObservationValue (TM)",
				"the time result value is not a valid FHIR time; valueTime was not set")
		}
	default:
		report.dropped("OBX-5 ObservationValue",
			"the OBX-2 value type is not in the supported set (NM, CE/CWE, TX/ST/FT, DT, TM); value[x] was not set")
	}
}

// numericQuantity builds a FHIR Quantity from an OBX-5 NM value and its OBX-6 units
// (a CWE). The value is carried as a Decimal so its lexical precision survives; the
// units' code, text, and coding system map to Quantity.code, Quantity.unit, and
// Quantity.system. A units coding system of UCUM resolves to the UCUM URI; any other
// is carried verbatim. ok is false when the value is not a valid FHIR decimal.
func numericQuantity(raw string, units hl7v2.CWE) (r5.Quantity, bool) {
	dec, err := fhir.ParseDecimal(raw)
	if err != nil {
		return r5.Quantity{}, false
	}
	q := r5.Quantity{}
	value := dec
	q.Value = &value
	if units.Code != "" {
		c := units.Code
		q.Code = &c
	}
	if units.Text != "" {
		u := units.Text
		q.Unit = &u
	}
	if units.CodingSystem != "" {
		system := observationUnitSystem(units.CodingSystem)
		q.System = &system
	}
	return q, true
}

// observationUnitSystem maps an OBX-6 units coding system to its FHIR system URI.
// UCUM is the system FHIR Quantity binds to; any other identifier is carried
// verbatim so the value is preserved rather than dropped.
func observationUnitSystem(system string) string {
	if system == "UCUM" {
		return ucumSystem
	}
	return system
}

// codedValueConcept maps an OBX-5 CWE-typed value (rendered as "code^text^system")
// to a FHIR CodeableConcept, or nil when the value carries no code.
func codedValueConcept(raw string) *r5.CodeableConcept {
	cwe := parseInlineCWE(raw)
	return cweConcept(cwe)
}

// dtmDateTimeValue parses an OBX-5 DT/DATE value as an HL7 DTM and renders it as a
// FHIR dateTime, preserving precision and dropping a timezone-less time per
// hl7DateTimeToFHIR. ok is false when the value does not parse.
func dtmDateTimeValue(raw string, report *Report) (string, bool) {
	dtm, err := hl7v2.ParseDTM(raw)
	if err != nil {
		return "", false
	}
	when := hl7DateTimeToFHIR(dtm, report, "Observation.valueDateTime")
	return when, when != ""
}

// timeValue parses an OBX-5 TM value (an HL7 timestamp truncated to a time) and
// renders it as a FHIR time "hh:mm:ss". FHIR time carries no timezone. ok is false
// when the value does not carry at least an hour.
func timeValue(raw string) (string, bool) {
	dtm, err := hl7v2.ParseDTM("19700101" + raw)
	if err != nil {
		return "", false
	}
	t, prec, ok := dtm.Time()
	if !ok || prec < hl7v2.PrecisionHour {
		return "", false
	}
	return pad2(t.Hour()) + ":" + pad2(t.Minute()) + ":" + pad2(t.Second()), true
}

// abnormalFlagInterpretation maps OBX-8 abnormal flags (HL7 Table 0078) to FHIR
// Observation.interpretation CodeableConcepts. The HL7 and FHIR codes share the same
// symbols, so each flag is carried verbatim under the ObservationInterpretation
// system; an empty flag list yields no interpretation.
func abnormalFlagInterpretation(flags []string) []r5.CodeableConcept {
	var out []r5.CodeableConcept
	for _, flag := range flags {
		if flag == "" {
			continue
		}
		code := flag
		system := observationInterpretationSystem
		out = append(out, r5.CodeableConcept{
			Coding: []r5.Coding{{System: &system, Code: &code}},
		})
	}
	return out
}

// parseReferenceRange maps an OBX-7 reference range to a FHIR ObservationReferenceRange.
// A "low-high" range becomes low and high Quantities; a bare value or an unparseable
// form is carried as referenceRange.text so the range is preserved rather than dropped.
// ok is false when OBX-7 is empty.
func parseReferenceRange(raw string) (r5.ObservationReferenceRange, bool) {
	if raw == "" {
		return r5.ObservationReferenceRange{}, false
	}
	low, high, isRange := splitRange(raw)
	if !isRange {
		text := raw
		return r5.ObservationReferenceRange{Text: &text}, true
	}
	rr := r5.ObservationReferenceRange{}
	if q, ok := boundaryQuantity(low); ok {
		rr.Low = &q
	}
	if q, ok := boundaryQuantity(high); ok {
		rr.High = &q
	}
	if rr.Low == nil && rr.High == nil {
		text := raw
		return r5.ObservationReferenceRange{Text: &text}, true
	}
	return rr, true
}

// boundaryQuantity builds a bare-value FHIR Quantity from a reference-range
// boundary. ok is false when the boundary is empty or not a valid decimal.
func boundaryQuantity(s string) (r5.Quantity, bool) {
	if s == "" {
		return r5.Quantity{}, false
	}
	dec, err := fhir.ParseDecimal(s)
	if err != nil {
		return r5.Quantity{}, false
	}
	value := dec
	return r5.Quantity{Value: &value}, true
}

// splitRange splits an HL7 reference range of the form "low-high" into its bounds.
// A leading '-' on either bound (a negative number) is respected: the split point
// is the first '-' that is not at the start and not immediately after another sign.
// isRange is false when the string carries no interior separator.
func splitRange(raw string) (low, high string, isRange bool) {
	for i := 1; i < len(raw); i++ {
		if raw[i] == '-' && raw[i-1] != '-' && raw[i-1] != '+' {
			return raw[:i], raw[i+1:], true
		}
	}
	return "", "", false
}

// firstValue returns the first OBX-5 repetition, or "".
func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// cweConcept maps an HL7 CWE to a FHIR CodeableConcept, or nil when the CWE carries
// neither a code nor text. The Coding is carried across verbatim — go-radx does not
// translate between code systems — and the text becomes CodeableConcept.text.
func cweConcept(cwe hl7v2.CWE) *r5.CodeableConcept {
	if cwe.Code == "" && cwe.Text == "" {
		return nil
	}
	coding := r5.Coding{}
	if cwe.Code != "" {
		code := cwe.Code
		coding.Code = &code
	}
	if cwe.Text != "" {
		display := cwe.Text
		coding.Display = &display
	}
	if cwe.CodingSystem != "" {
		system := cwe.CodingSystem
		coding.System = &system
	}
	cc := &r5.CodeableConcept{Coding: []r5.Coding{coding}}
	if cwe.Text != "" {
		text := cwe.Text
		cc.Text = &text
	}
	return cc
}

// parseInlineCWE parses a canonical "code^text^system" CWE rendering, the form an
// OBX-5 CWE value carries as a single repetition string. A bare value with no '^'
// becomes the code alone.
func parseInlineCWE(raw string) hl7v2.CWE {
	parts := splitComponents(raw)
	cwe := hl7v2.CWE{}
	if len(parts) > 0 {
		cwe.Code = parts[0]
	}
	if len(parts) > 1 {
		cwe.Text = parts[1]
	}
	if len(parts) > 2 {
		cwe.CodingSystem = parts[2]
	}
	return cwe
}

// splitComponents splits a canonical component-delimited string on the standard '^'
// component separator.
func splitComponents(raw string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == '^' {
			parts = append(parts, raw[start:i])
			start = i + 1
		}
	}
	return append(parts, raw[start:])
}
