package convert

import (
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ObservationToOBXR4 maps one FHIR R4 Observation back to an HL7 v2 OBX segment,
// the R4 twin of ObservationToOBX (R5) and the inverse of OBXToObservationR4. The
// Observation.code becomes OBX-3, the value[x] branch chooses OBX-2 and renders
// OBX-5 identically to R5, the interpretation maps back to OBX-8, and a numeric
// reference range to OBX-7. The result status OBX-11 is "F" (final).
//
// ok is false for an Observation with no code, since OBX-3 is the observation
// identifier the forward path requires. A value[x] branch with no OBX form leaves
// OBX-2/5 unset and records the loss by element path, never the clinical value.
func ObservationToOBXR4(o *r4.Observation, _ ...Option) (hl7v2.OBX, *Report, bool) {
	report := &Report{}
	obx, ok := observationToOBXR4(o, report)
	return obx, report, ok
}

// observationToOBXR4 is the loss-aware R4 reverse OBX converter. It records on
// report the FHIR element that had no HL7 OBX home, never the clinical value.
func observationToOBXR4(o *r4.Observation, report *Report) (hl7v2.OBX, bool) {
	if o == nil {
		return hl7v2.OBX{}, false
	}
	id, ok := conceptToCWER4(o.Code)
	if !ok {
		report.dropped("Observation.code",
			"Observation has no code; OBX-3 is the required observation identifier, so no OBX was emitted")
		return hl7v2.OBX{}, false
	}

	obx := hl7v2.OBX{
		ObservationID: id,
		ResultStatus:  "F",
	}

	setOBXValueR4(&obx, o, report)
	obx.AbnormalFlags = interpretationFlagsR4(o.Interpretation)
	if rr, ok := referenceRangeTextR4(o.ReferenceRange); ok {
		obx.ReferenceRange = rr
	}
	return obx, true
}

// setOBXValueR4 dispatches the R4 Observation's value[x] into OBX-2 and OBX-5, the
// R4 twin of setOBXValue. The lexical and inline CWE rendering helpers are shared.
func setOBXValueR4(obx *hl7v2.OBX, o *r4.Observation, report *Report) {
	value, ok := o.Value()
	if !ok {
		report.dropped("Observation.value[x]",
			"Observation carries no value; OBX-5 was left unset")
		return
	}
	switch v := value.(type) {
	case r4.Quantity:
		if v.Value == nil || v.Value.String() == "" {
			report.dropped("Observation.valueQuantity",
				"the Quantity carried no value; OBX-5 was left unset")
			return
		}
		obx.ValueType = "NM"
		obx.Value = []string{v.Value.String()}
		obx.Units = quantityToCWER4(v)
	case r4.CodeableConcept:
		cwe, ok := conceptToCWER4(&v)
		if !ok {
			report.dropped("Observation.valueCodeableConcept",
				"the coded value carried no code or text; OBX-5 was left unset")
			return
		}
		obx.ValueType = "CWE"
		obx.Value = []string{renderInlineCWE(cwe)}
	case r4.FHIRString:
		obx.ValueType = "ST"
		obx.Value = []string{string(v)}
	case r4.FHIRDateTime:
		lexical, lok := fhirDateTimeToDICOM(string(v))
		if !lok {
			report.dropped("Observation.valueDateTime",
				"the dateTime is not a well-formed FHIR dateTime; OBX-5 was left unset")
			return
		}
		obx.ValueType = "TS"
		obx.Value = []string{lexical}
	case r4.FHIRTime:
		obx.ValueType = "TM"
		obx.Value = []string{stripColons(string(v))}
	default:
		report.dropped("Observation.value[x]",
			"the value[x] branch has no HL7 OBX-5 form; OBX-5 was left unset")
	}
}

// quantityToCWER4 maps an R4 FHIR Quantity's units back to an OBX-6 CWE, the R4
// twin of quantityToCWE. The system maps back through the shared obxUnitSystem
// helper so a UCUM quantity round-trips its "UCUM" coding system.
func quantityToCWER4(q r4.Quantity) hl7v2.CWE {
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

// conceptToCWER4 maps the first Coding of an R4 CodeableConcept back to an HL7 CWE,
// the R4 twin of conceptToCWE. The Coding.display, or the CodeableConcept.text when
// the Coding has none, becomes CWE-2. ok is false only for a CWE with neither a
// code nor text.
func conceptToCWER4(cc *r4.CodeableConcept) (hl7v2.CWE, bool) {
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
	return cwe, true
}

// interpretationFlagsR4 maps R4 Observation.interpretation back to OBX-8 abnormal
// flags, the R4 twin of interpretationFlags. The FHIR and HL7 codes share the same
// symbols, so each interpretation Coding's code is carried verbatim.
func interpretationFlagsR4(interp []r4.CodeableConcept) []string {
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

// referenceRangeTextR4 maps the first R4 Observation.referenceRange back to an
// OBX-7 value, the R4 twin of referenceRangeText. A low/high Quantity pair renders
// "low-high"; a text-only range renders its text verbatim. ok is false when no
// reference range is set.
func referenceRangeTextR4(ranges []r4.ObservationReferenceRange) (string, bool) {
	if len(ranges) == 0 {
		return "", false
	}
	rr := ranges[0]
	if rr.Text != nil && *rr.Text != "" {
		return *rr.Text, true
	}
	low := boundaryStringR4(rr.Low)
	high := boundaryStringR4(rr.High)
	if low == "" && high == "" {
		return "", false
	}
	return low + "-" + high, true
}

// boundaryStringR4 renders an R4 reference-range boundary Quantity's value, or ""
// when the boundary is unset or value-less, the R4 twin of boundaryString.
func boundaryStringR4(q *r4.Quantity) string {
	if q == nil || q.Value == nil {
		return ""
	}
	return q.Value.String()
}
