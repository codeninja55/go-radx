package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ORUToDiagnosticReportR4 converts an HL7 v2 observation result message (ORU^R01)
// to a FHIR R4 DiagnosticReport and the set of Observations carrying its results,
// the R4 twin of ORUToDiagnosticReportR5. The OBR/OBX reading, the
// single-result-group boundary, the deterministic urn:uuid result linking, and the
// identity rule are identical. The DiagnosticReport and Observation resource models
// for the v1 mapping's fields are the same shape in R4 and R5, so the twin differs
// only in the release sub-package its types come from.
//
// subject carries the PID-3 patient identity logically (Reference.identifier), or
// the WithSubjectR4 reference when supplied — never a fabricated Reference.reference
// URL.
func ORUToDiagnosticReportR4(msg *hl7v2.Message, opts ...Option) (*r4.DiagnosticReport, []*r4.Observation, *Report, error) {
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
		report.dropped(
			fmt.Sprintf("OBR (result group %d)", i+2),
			"v1 DiagnosticReport maps a single OBR; extra result groups are not mapped",
		)
	}

	group := groups[0]
	dr := &r4.DiagnosticReport{}

	// DiagnosticReport.code is required by the R4 model; OBR-4 is its only source.
	code := cweConceptR4(group.Order.UniversalServiceID)
	if code == nil {
		return nil, nil, nil, fmt.Errorf("%w: OBR-4 UniversalServiceID is empty; DiagnosticReport.code is required",
			ErrMalformedSource)
	}
	dr.Code = code

	status := diagnosticReportStatusR4(msg, report)
	dr.Status = &status

	if when := hl7DateTimeToFHIR(group.Order.ObservationDateTime, report, "DiagnosticReport.effectiveDateTime"); when != "" {
		dr.SetEffectiveDateTime(r4.FHIRDateTime(when))
	}

	if subject := patientSubjectR4(cfg, msg, report, "DiagnosticReport.subject"); subject != nil {
		dr.Subject = subject
	}

	controlID := ""
	if h, hasMSH := msg.MSH(); hasMSH {
		controlID = h.ControlID
	}

	var observations []*r4.Observation
	for i := range group.Observations {
		o, ok := obxToObservationR4(group.Observations[i], report)
		if !ok {
			report.dropped(
				fmt.Sprintf("OBX (observation %d)", i+1),
				"OBX has no OBX-3 observation identifier for the required Observation.code; the row was not mapped",
			)
			continue
		}
		id := deterministicObservationUUID(controlID, len(observations))
		o.ID = &id
		observations = append(observations, o)
		ref := "urn:uuid:" + id
		dr.Result = append(dr.Result, r4.Reference{Reference: &ref})
	}

	rep, err := cfg.finalize(report)
	return dr, observations, rep, err
}

// diagnosticReportStatusR4 maps OBR-25 Result Status (HL7 Table 0123) to an R4
// DiagnosticReport.status, the R4 twin of diagnosticReportStatus. The mapping and
// the defaulting rules are identical.
func diagnosticReportStatusR4(msg *hl7v2.Message, report *Report) r4.DiagnosticReportStatus {
	code := getField(msg, "OBR-25")
	switch code {
	case "F":
		return r4.DiagnosticReportStatusFinal
	case "P":
		return r4.DiagnosticReportStatusPreliminary
	case "C":
		return r4.DiagnosticReportStatusCorrected
	case "X":
		return r4.DiagnosticReportStatusCancelled
	case "":
		report.defaulted("DiagnosticReport.status", "final",
			"ORU has no OBR-25 result status; defaulted")
		return r4.DiagnosticReportStatusFinal
	default:
		report.defaulted("DiagnosticReport.status", "final",
			"OBR-25 result status is not in HL7 Table 0123; defaulted (the binding is required)")
		return r4.DiagnosticReportStatusFinal
	}
}

// OBXToObservationR4 maps one HL7 v2 OBX result to a FHIR R4 Observation, the R4
// twin of OBXToObservationR5. OBX-3 becomes the required Observation.code, OBX-2
// dispatches the value into value[x], OBX-8 abnormal flags become
// Observation.interpretation, and OBX-7 a reference range.
//
// ok is false only when the OBX has no observation identifier (OBX-3); an
// unmappable value type leaves value[x] unset (recorded) rather than rejecting the
// observation.
func OBXToObservationR4(obx hl7v2.OBX, _ ...Option) (*r4.Observation, *Report, bool) {
	report := &Report{}
	o, ok := obxToObservationR4(obx, report)
	return o, report, ok
}

// obxToObservationR4 is the loss-aware R4 OBX converter shared by
// ORUToDiagnosticReportR4 and the exported single-leaf entry point.
func obxToObservationR4(obx hl7v2.OBX, report *Report) (*r4.Observation, bool) {
	code := cweConceptR4(obx.ObservationID)
	if code == nil {
		return nil, false
	}

	o := &r4.Observation{Code: code}
	status := r4.ObservationStatusFinal
	o.Status = &status

	setObservationValueR4(o, obx, report)
	if interp := abnormalFlagInterpretationR4(obx.AbnormalFlags); len(interp) > 0 {
		o.Interpretation = interp
	}
	if rr, ok := parseReferenceRangeR4(obx.ReferenceRange); ok {
		o.ReferenceRange = []r4.ObservationReferenceRange{rr}
	}
	return o, true
}

// setObservationValueR4 dispatches the OBX value into the R4 Observation's value[x]
// by OBX-2 ValueType, the R4 twin of setObservationValue. The temporal and inline
// CWE parsing helpers are release-agnostic and shared; only the value[x] setters
// and Quantity/CodeableConcept types are R4.
func setObservationValueR4(o *r4.Observation, obx hl7v2.OBX, report *Report) {
	raw := firstValue(obx.Value)
	if len(obx.Value) > 1 {
		report.dropped(
			fmt.Sprintf("OBX-5 ObservationValue (%d additional repetitions)", len(obx.Value)-1),
			"FHIR Observation.value[x] holds a single value; OBX-5 repetitions beyond the first were not mapped",
		)
	}
	switch obx.ValueType {
	case "NM":
		if q, ok := numericQuantityR4(raw, obx.Units); ok {
			o.SetValueQuantity(q)
			return
		}
		report.dropped("OBX-5 ObservationValue (NM)",
			"the numeric result value is not a valid FHIR decimal; valueQuantity was not set")
	case "CE", "CWE":
		if cc := codedValueConceptR4(raw); cc != nil {
			o.SetValueCodeableConcept(*cc)
			return
		}
		report.dropped("OBX-5 ObservationValue (CWE)",
			"the coded result value carried no code; valueCodeableConcept was not set")
	case "TX", "ST", "FT":
		if raw != "" {
			o.SetValueString(r4.FHIRString(raw))
			return
		}
	case "DT", "DATE", "TS":
		if when, ok := dtmDateTimeValue(raw, report); ok {
			o.SetValueDateTime(r4.FHIRDateTime(when))
			return
		}
		if raw != "" {
			report.dropped("OBX-5 ObservationValue (DT/TS)",
				"the date/time result value is not a valid FHIR dateTime; valueDateTime was not set")
		}
	case "TM", "TIME":
		if when, ok := timeValue(raw); ok {
			o.SetValueTime(r4.FHIRTime(when))
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

// numericQuantityR4 builds an R4 FHIR Quantity from an OBX-5 NM value and its OBX-6
// units, the R4 twin of numericQuantity. The value is carried as a Decimal; the
// units map through the shared observationUnitSystem helper. ok is false when the
// value is not a valid FHIR decimal.
func numericQuantityR4(raw string, units hl7v2.CWE) (r4.Quantity, bool) {
	dec, err := fhir.ParseDecimal(raw)
	if err != nil {
		return r4.Quantity{}, false
	}
	q := r4.Quantity{}
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

// codedValueConceptR4 maps an OBX-5 CWE-typed value to an R4 CodeableConcept, the
// R4 twin of codedValueConcept. The inline CWE parsing helper is shared.
func codedValueConceptR4(raw string) *r4.CodeableConcept {
	cwe := parseInlineCWE(raw)
	return cweConceptR4(cwe)
}

// abnormalFlagInterpretationR4 maps OBX-8 abnormal flags to R4
// Observation.interpretation CodeableConcepts, the R4 twin of
// abnormalFlagInterpretation. The HL7 and FHIR codes share the same symbols, so
// each flag is carried verbatim under the ObservationInterpretation system.
func abnormalFlagInterpretationR4(flags []string) []r4.CodeableConcept {
	var out []r4.CodeableConcept
	for _, flag := range flags {
		if flag == "" {
			continue
		}
		code := flag
		system := observationInterpretationSystem
		out = append(out, r4.CodeableConcept{
			Coding: []r4.Coding{{System: &system, Code: &code}},
		})
	}
	return out
}

// parseReferenceRangeR4 maps an OBX-7 reference range to an R4
// ObservationReferenceRange, the R4 twin of parseReferenceRange. The range-splitting
// helper splitRange is shared.
func parseReferenceRangeR4(raw string) (r4.ObservationReferenceRange, bool) {
	if raw == "" {
		return r4.ObservationReferenceRange{}, false
	}
	low, high, isRange := splitRange(raw)
	if !isRange {
		text := raw
		return r4.ObservationReferenceRange{Text: &text}, true
	}
	rr := r4.ObservationReferenceRange{}
	if q, ok := boundaryQuantityR4(low); ok {
		rr.Low = &q
	}
	if q, ok := boundaryQuantityR4(high); ok {
		rr.High = &q
	}
	if rr.Low == nil && rr.High == nil {
		text := raw
		return r4.ObservationReferenceRange{Text: &text}, true
	}
	return rr, true
}

// boundaryQuantityR4 builds a bare-value R4 FHIR Quantity from a reference-range
// boundary, the R4 twin of boundaryQuantity. ok is false when the boundary is empty
// or not a valid decimal.
func boundaryQuantityR4(s string) (r4.Quantity, bool) {
	if s == "" {
		return r4.Quantity{}, false
	}
	dec, err := fhir.ParseDecimal(s)
	if err != nil {
		return r4.Quantity{}, false
	}
	value := dec
	return r4.Quantity{Value: &value}, true
}

// cweConceptR4 maps an HL7 CWE to an R4 CodeableConcept, the R4 twin of cweConcept,
// or nil when the CWE carries neither a code nor text. The Coding is carried
// verbatim and the text becomes CodeableConcept.text.
func cweConceptR4(cwe hl7v2.CWE) *r4.CodeableConcept {
	if cwe.Code == "" && cwe.Text == "" {
		return nil
	}
	coding := r4.Coding{}
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
	cc := &r4.CodeableConcept{Coding: []r4.Coding{coding}}
	if cwe.Text != "" {
		text := cwe.Text
		cc.Text = &text
	}
	return cc
}
