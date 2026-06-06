package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// ObservationToContentItem maps one FHIR R5 Observation back to a DICOM SR content-item
// leaf, the inverse of ContentItemToObservationR5. The Observation.code becomes the
// leaf's Concept Name Code Sequence, and the leaf's ValueType and value are chosen by
// the Observation's value[x] branch: valueQuantity to NUM (MeasuredValue plus
// MeasurementUnits), valueCodeableConcept to CODE, valueString to TEXT, valueTime to
// TIME, and valueDateTime to DATE or DATETIME by whether the value carries a time. The
// produced item carries the CONTAINS relationship, matching how a measurement leaf sits
// under the document root.
//
// ok is false for an Observation with no code (an SR content item requires a Concept
// Name Code Sequence) or whose value[x] is unset or of an unsupported branch — the round
// trip names what could not be re-encoded on the supplied report rather than emitting a
// value-less leaf.
func ObservationToContentItem(o *r5.Observation, _ ...Option) (dicom.ContentItem, *Report, bool) {
	report := &Report{}
	item, ok := observationToContentItem(o, report)
	return item, report, ok
}

// observationToContentItem is the loss-aware reverse leaf converter shared by the
// exported entry point and DiagnosticReportToSR. It records on report the FHIR element
// that had no DICOM SR home, never the clinical value it carried.
func observationToContentItem(o *r5.Observation, report *Report) (dicom.ContentItem, bool) {
	if o == nil {
		return dicom.ContentItem{}, false
	}
	concept := conceptNameFor(o.Code)
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
	case r5.Quantity:
		if !setNumValue(&item, v, report) {
			return dicom.ContentItem{}, false
		}
	case r5.CodeableConcept:
		code := conceptNameFor(&v)
		if code.IsZero() {
			report.dropped("Observation.valueCodeableConcept",
				"the coded value carried no Coding; a DICOM CODE item requires a Concept Code Sequence, so the leaf was not emitted")
			return dicom.ContentItem{}, false
		}
		item.ValueType = dicom.ValueTypeCode
		item.Code = code
	case r5.FHIRString:
		item.ValueType = dicom.ValueTypeText
		item.Text = string(v)
	case r5.FHIRTime:
		// A DICOM SR TIME content item carries a (0040,A122) Time value of VR TM, which
		// the dicom.DT value cannot hold from outside the dicom package (a TM lexical is
		// not a constructible DT and the package exposes no lexical-DT constructor). The
		// time is recorded as reported loss rather than emitted on a wrong-typed leaf, so
		// the round trip is honest about the one value[x] branch it cannot re-encode here.
		report.dropped("Observation.valueTime",
			"a FHIR time maps to a DICOM SR TIME content item (0040,A122) of VR TM, which the reverse content-item builder cannot carry; the leaf was not emitted")
		return dicom.ContentItem{}, false
	case r5.FHIRDateTime:
		if !setDateTimeValue(&item, string(v), report) {
			return dicom.ContentItem{}, false
		}
	default:
		report.dropped("Observation.value[x]",
			"the value[x] branch has no DICOM SR content-item form; the leaf was not emitted")
		return dicom.ContentItem{}, false
	}

	return item, true
}

// setNumValue fills a NUM leaf from a FHIR Quantity: Quantity.value becomes the
// MeasuredValue and the Quantity.code/unit/system become the MeasurementUnits triplet,
// the inverse of measurementQuantity. ok is false when the Quantity carries no value,
// since a NUM item with no MeasuredValue has nothing to round-trip.
func setNumValue(item *dicom.ContentItem, q r5.Quantity, report *Report) bool {
	if q.Value == nil || q.Value.String() == "" {
		report.dropped("Observation.valueQuantity",
			"the Quantity carried no value; a DICOM NUM item requires a Measured Value, so the leaf was not emitted")
		return false
	}
	item.ValueType = dicom.ValueTypeNum
	item.MeasuredValue = *q.Value
	item.MeasurementUnits = quantityUnits(q)
	return true
}

// quantityUnits maps a FHIR Quantity's units (code, unit text, system) back to a DICOM
// MeasurementUnits ConceptNameCode, the inverse of the units half of measurementQuantity.
// A Quantity with no units yields the zero ConceptNameCode.
func quantityUnits(q r5.Quantity) dicom.ConceptNameCode {
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

// setDateTimeValue fills a DATE or DATETIME leaf from a FHIR dateTime: a date-only value
// produces a DATE item, a value carrying a time produces a DATETIME item. ok is false
// when the value is not at least a full calendar date.
func setDateTimeValue(item *dicom.ContentItem, value string, report *Report) bool {
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
