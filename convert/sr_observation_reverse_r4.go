package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// ObservationToContentItemR4 maps one FHIR R4 Observation back to a DICOM SR
// content-item leaf, the R4 twin of ObservationToContentItem (R5) and the inverse
// of ContentItemToObservationR4. The Observation.code becomes the leaf's Concept
// Name Code Sequence, and the leaf's ValueType and value are chosen by the
// Observation's value[x] branch identically to R5.
//
// ok is false for an Observation with no code or whose value[x] is unset or of an
// unsupported branch — the round trip names what could not be re-encoded on the
// supplied report rather than emitting a value-less leaf.
func ObservationToContentItemR4(o *r4.Observation, _ ...Option) (dicom.ContentItem, *Report, bool) {
	report := &Report{}
	item, ok := observationToContentItemR4(o, report)
	return item, report, ok
}

// observationToContentItemR4 is the loss-aware R4 reverse leaf converter shared by
// the exported entry point and DiagnosticReportToSRR4. The dateTime lexical helper
// fhirDateTimeToDICOM is release-agnostic and shared.
func observationToContentItemR4(o *r4.Observation, report *Report) (dicom.ContentItem, bool) {
	if o == nil {
		return dicom.ContentItem{}, false
	}
	concept := conceptNameForR4(o.Code)
	if concept.IsZero() {
		report.dropped("Observation.code",
			"Observation has no code; a DICOM SR content item requires a Concept Name Code Sequence, so the leaf was not emitted")
		return dicom.ContentItem{}, false
	}

	item := dicom.ContentItem{
		Relationship: dicom.RelationshipContains,
		ConceptName:  concept,
	}

	value, ok := o.Value()
	if !ok {
		report.dropped("Observation.value[x]",
			"Observation carries no value; the SR leaf would have no value, so it was not emitted")
		return dicom.ContentItem{}, false
	}

	switch v := value.(type) {
	case r4.Quantity:
		if !setNumValueR4(&item, v, report) {
			return dicom.ContentItem{}, false
		}
	case r4.CodeableConcept:
		code := conceptNameForR4(&v)
		if code.IsZero() {
			report.dropped("Observation.valueCodeableConcept",
				"the coded value carried no Coding; a DICOM CODE item requires a Concept Code Sequence, so the leaf was not emitted")
			return dicom.ContentItem{}, false
		}
		item.ValueType = dicom.ValueTypeCode
		item.Code = code
	case r4.FHIRString:
		item.ValueType = dicom.ValueTypeText
		item.Text = string(v)
	case r4.FHIRTime:
		// A DICOM SR TIME content item carries a (0040,A122) Time value of VR TM, which
		// the dicom.DT value cannot hold from outside the dicom package. The time is
		// recorded as reported loss rather than emitted on a wrong-typed leaf, matching
		// the R5 reverse path's one un-re-encodable value[x] branch.
		report.dropped("Observation.valueTime",
			"a FHIR time maps to a DICOM SR TIME content item (0040,A122) of VR TM, which the reverse content-item builder cannot carry; the leaf was not emitted")
		return dicom.ContentItem{}, false
	case r4.FHIRDateTime:
		if !setDateTimeValueR4(&item, string(v), report) {
			return dicom.ContentItem{}, false
		}
	default:
		report.dropped("Observation.value[x]",
			"the value[x] branch has no DICOM SR content-item form; the leaf was not emitted")
		return dicom.ContentItem{}, false
	}

	return item, true
}

// setNumValueR4 fills a NUM leaf from an R4 FHIR Quantity, the R4 twin of
// setNumValue. ok is false when the Quantity carries no value.
func setNumValueR4(item *dicom.ContentItem, q r4.Quantity, report *Report) bool {
	if q.Value == nil || q.Value.String() == "" {
		report.dropped("Observation.valueQuantity",
			"the Quantity carried no value; a DICOM NUM item requires a Measured Value, so the leaf was not emitted")
		return false
	}
	item.ValueType = dicom.ValueTypeNum
	item.MeasuredValue = *q.Value
	item.MeasurementUnits = quantityUnitsR4(q)
	return true
}

// quantityUnitsR4 maps an R4 FHIR Quantity's units back to a DICOM
// MeasurementUnits ConceptNameCode, the R4 twin of quantityUnits. The system maps
// back through the shared schemeDesignatorFor helper.
func quantityUnitsR4(q r4.Quantity) dicom.ConceptNameCode {
	units := dicom.ConceptNameCode{}
	if q.Code != nil {
		units.CodeValue = *q.Code
	}
	if q.Unit != nil {
		units.CodeMeaning = *q.Unit
	}
	if q.System != nil {
		units.CodingSchemeDesignator = schemeDesignatorFor(*q.System)
	}
	return units
}

// setDateTimeValueR4 fills a DATE or DATETIME leaf from an R4 FHIR dateTime, the R4
// twin of setDateTimeValue. The lexical conversion helper fhirDateTimeToDICOM is
// shared. ok is false when the value is not at least a full calendar date.
func setDateTimeValueR4(item *dicom.ContentItem, value string, report *Report) bool {
	lexical, ok := fhirDateTimeToDICOM(value)
	if !ok {
		report.dropped("Observation.valueDateTime",
			"the dateTime value is not at least a full calendar date; the SR leaf was not emitted")
		return false
	}
	dt, err := dicom.ParseDT(lexical)
	if err != nil {
		report.dropped("Observation.valueDateTime",
			"the dateTime value does not re-encode as a DICOM DT; the SR leaf was not emitted")
		return false
	}
	if indexByte(value, 'T') >= 0 {
		item.ValueType = dicom.ValueTypeDateTime
	} else {
		item.ValueType = dicom.ValueTypeDate
	}
	item.DateTime = dt
	return true
}
