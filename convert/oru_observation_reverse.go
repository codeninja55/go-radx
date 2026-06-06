package convert

import (
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ObservationToOBX maps one FHIR R5 Observation back to an HL7 v2 OBX segment, the
// inverse of OBXToObservationR5. The Observation.code becomes OBX-3, the value[x] branch
// chooses OBX-2 (the value type) and renders OBX-5: a Quantity to NM plus OBX-6 units, a
// CodeableConcept to CWE, a string to ST, a dateTime to TS, and a time to TM. The
// interpretation maps back to OBX-8 abnormal flags and a numeric reference range to
// OBX-7. The result status OBX-11 is "F" (final), matching the status the forward path
// assigns a reported result.
//
// ok is false for an Observation with no code, since OBX-3 is the observation identifier
// the forward path requires. A value[x] branch with no OBX form leaves OBX-2/5 unset and
// records the loss by element path, never the clinical value.
func ObservationToOBX(o *r5.Observation, _ ...Option) (hl7v2.OBX, *Report, bool) {
	report := &Report{}
	obx, ok := observationToOBX(o, report)
	return obx, report, ok
}

// observationToOBX is the loss-aware reverse OBX converter. It records on report the
// FHIR element that had no HL7 OBX home, never the clinical value it carried.
func observationToOBX(o *r5.Observation, report *Report) (hl7v2.OBX, bool) {
	if o == nil {
		return hl7v2.OBX{}, false
	}
	id, ok := conceptToCWE(o.Code)
	if !ok {
		report.dropped("Observation.code",
			"Observation has no code; OBX-3 is the required observation identifier, so no OBX was emitted")
		return hl7v2.OBX{}, false
	}

	obx := hl7v2.OBX{
		ObservationID: id,
		ResultStatus:  "F",
	}

	setOBXValue(&obx, o, report)
	obx.AbnormalFlags = interpretationFlags(o.Interpretation)
	if rr, ok := referenceRangeText(o.ReferenceRange); ok {
		obx.ReferenceRange = rr
	}
	return obx, true
}

// setOBXValue dispatches the Observation's value[x] into OBX-2 (value type) and OBX-5
// (value), the inverse of setObservationValue. A Quantity also fills OBX-6 units. An
// unset or unsupported value[x] leaves OBX-2/5 empty and records the loss by element
// path only.
func setOBXValue(obx *hl7v2.OBX, o *r5.Observation, report *Report) {
	value, ok := o.Value()
	if !ok {
		report.dropped("Observation.value[x]",
			"Observation carries no value; OBX-5 was left unset")
		return
	}
	switch v := value.(type) {
	case r5.Quantity:
		if v.Value == nil || v.Value.String() == "" {
			report.dropped("Observation.valueQuantity",
				"the Quantity carried no value; OBX-5 was left unset")
			return
		}
		obx.ValueType = "NM"
		obx.Value = []string{v.Value.String()}
		obx.Units = quantityToCWE(v)
	case r5.CodeableConcept:
		cwe, hasCode := conceptToCWE(&v)
		if !hasCode {
			report.dropped("Observation.valueCodeableConcept",
				"the coded value carried no Coding; OBX-5 was left unset")
			return
		}
		obx.ValueType = "CWE"
		obx.Value = []string{renderInlineCWE(cwe)}
	case r5.FHIRString:
		obx.ValueType = "ST"
		obx.Value = []string{string(v)}
	case r5.FHIRDateTime:
		lexical, lok := fhirDateTimeToDICOM(string(v))
		if !lok {
			report.dropped("Observation.valueDateTime",
				"the dateTime is not at least a full calendar date; OBX-5 was left unset")
			return
		}
		obx.ValueType = "TS"
		obx.Value = []string{lexical}
	case r5.FHIRTime:
		obx.ValueType = "TM"
		obx.Value = []string{stripColons(string(v))}
	default:
		report.dropped("Observation.value[x]",
			"the value[x] branch has no HL7 OBX-5 form; OBX-5 was left unset")
	}
}

// quantityToCWE maps a FHIR Quantity's units back to an OBX-6 CWE, the inverse of the
// units half of numericQuantity: Quantity.code to CWE-1, Quantity.unit to CWE-2, and the
// system back through the UCUM-aware inverse so a UCUM quantity round-trips its "UCUM"
// coding system.
func quantityToCWE(q r5.Quantity) hl7v2.CWE {
	cwe := hl7v2.CWE{}
	if q.Code != nil {
		cwe.Code = *q.Code
	}
	if q.Unit != nil {
		cwe.Text = *q.Unit
	}
	if q.System != nil {
		cwe.CodingSystem = obxUnitSystem(*q.System)
	}
	return cwe
}

// obxUnitSystem is the inverse of observationUnitSystem: the UCUM system URI maps back to
// the "UCUM" coding-system identifier, and any other URI is carried verbatim.
func obxUnitSystem(system string) string {
	if system == ucumSystem {
		return "UCUM"
	}
	return system
}

// conceptToCWE maps the first Coding of a FHIR CodeableConcept back to an HL7 CWE
// (code, text, coding system), the inverse of cweConcept. The Coding.display, or the
// CodeableConcept.text when the Coding has none, becomes CWE-2. ok is false when the
// concept carries no Coding with a code, since an OBX-3 with no code has no observation
// identity.
func conceptToCWE(cc *r5.CodeableConcept) (hl7v2.CWE, bool) {
	if cc == nil || len(cc.Coding) == 0 {
		return hl7v2.CWE{}, false
	}
	coding := cc.Coding[0]
	cwe := hl7v2.CWE{}
	if coding.Code != nil {
		cwe.Code = *coding.Code
	}
	switch {
	case coding.Display != nil:
		cwe.Text = *coding.Display
	case cc.Text != nil:
		cwe.Text = *cc.Text
	}
	if coding.System != nil {
		cwe.CodingSystem = *coding.System
	}
	if cwe.Code == "" && cwe.Text == "" {
		return hl7v2.CWE{}, false
	}
	return cwe, cwe.Code != ""
}

// renderInlineCWE renders a CWE as the canonical "code^text^system" OBX-5 value form
// the forward parseInlineCWE reads, so a coded result value round-trips.
func renderInlineCWE(cwe hl7v2.CWE) string {
	return cwe.Code + "^" + cwe.Text + "^" + cwe.CodingSystem
}

// interpretationFlags maps Observation.interpretation back to OBX-8 abnormal flags, the
// inverse of abnormalFlagInterpretation: the FHIR and HL7 codes share the same symbols,
// so each interpretation Coding's code is carried verbatim.
func interpretationFlags(interp []r5.CodeableConcept) []string {
	var out []string
	for _, cc := range interp {
		for _, c := range cc.Coding {
			if c.Code != nil && *c.Code != "" {
				out = append(out, *c.Code)
			}
		}
	}
	return out
}

// referenceRangeText maps the first Observation.referenceRange back to an OBX-7 value,
// the inverse of parseReferenceRange. A low/high Quantity pair renders "low-high"; a
// text-only range renders its text verbatim. ok is false when no reference range is set.
func referenceRangeText(ranges []r5.ObservationReferenceRange) (string, bool) {
	if len(ranges) == 0 {
		return "", false
	}
	rr := ranges[0]
	if rr.Text != nil && *rr.Text != "" {
		return *rr.Text, true
	}
	low := boundaryString(rr.Low)
	high := boundaryString(rr.High)
	if low == "" && high == "" {
		return "", false
	}
	return low + "-" + high, true
}

// boundaryString renders a reference-range boundary Quantity's value, or "" when the
// boundary is unset or value-less.
func boundaryString(q *r5.Quantity) string {
	if q == nil || q.Value == nil {
		return ""
	}
	return q.Value.String()
}
