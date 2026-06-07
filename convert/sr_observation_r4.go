package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// ContentItemToObservationR4 maps one DICOM SR content-item leaf to a FHIR R4
// Observation, the R4 twin of ContentItemToObservationR5. The leaf's Concept Name
// Code Sequence becomes the required Observation.code, and the leaf value maps to
// value[x] by ValueType identically to R5. The Observation is emitted with status
// "final".
//
// ok is false for a content item that is structure rather than an observation, or
// for a leaf whose value could not be rendered into a conformant value[x]. The
// returned Observation has no id; the caller assigns the deterministic urn:uuid
// logical id it links from DiagnosticReport.result.
func ContentItemToObservationR4(item dicom.ContentItem, _ ...Option) (*r4.Observation, bool) {
	return contentItemToObservationR4(item, "", false, &Report{})
}

// contentItemToObservationR4 is the loss-aware R4 leaf converter, the R4 twin of
// contentItemToObservationR5. The temporal helpers srTimeToFHIR and srDateTimeToFHIR
// are release-agnostic and shared; only the Observation type and its value[x]
// setters are R4.
func contentItemToObservationR4(item dicom.ContentItem, tzOffset string, hasTZ bool, report *Report) (*r4.Observation, bool) {
	code := conceptNameCodeR4(item.ConceptName)
	if code == nil {
		return nil, false
	}

	o := &r4.Observation{Code: code}
	status := r4.ObservationStatusFinal
	o.Status = &status

	switch item.ValueType {
	case dicom.ValueTypeNum:
		q, ok := measurementQuantityR4(item)
		if !ok {
			report.dropped(dicomTagSource(dicom.TagNumericValue),
				"NUM content item has no usable numeric value; a value-less Quantity is non-conformant, so the Observation was dropped")
			return nil, false
		}
		o.SetValueQuantity(q)
	case dicom.ValueTypeCode:
		cc := conceptNameCodeR4(item.Code)
		if cc == nil {
			return nil, false
		}
		o.SetValueCodeableConcept(*cc)
	case dicom.ValueTypeText:
		o.SetValueString(r4.FHIRString(item.Text))
	case dicom.ValueTypeTime:
		when, ok := srTimeToFHIR(item.DateTime)
		if !ok {
			return nil, false
		}
		o.SetValueTime(r4.FHIRTime(when))
	case dicom.ValueTypeDate, dicom.ValueTypeDateTime:
		when, ok := srDateTimeToFHIR(item.DateTime, tzOffset, hasTZ)
		if !ok {
			return nil, false
		}
		o.SetValueDateTime(r4.FHIRDateTime(when))
	default:
		return nil, false
	}

	return o, true
}

// measurementQuantityR4 builds an R4 FHIR Quantity from a NUM leaf's MeasuredValue
// and MeasurementUnits, the R4 twin of measurementQuantity. The numeric value is
// carried as a Decimal so its lexical precision survives; the units' code, meaning,
// and scheme map to Quantity.code, Quantity.unit, and Quantity.system through the
// shared measurementUnitSystem helper. ok is false when the leaf carries no numeric
// value.
func measurementQuantityR4(item dicom.ContentItem) (r4.Quantity, bool) {
	if item.MeasuredValue.String() == "" {
		return r4.Quantity{}, false
	}
	q := r4.Quantity{}
	value := fhir.Decimal(item.MeasuredValue)
	q.Value = &value

	units := item.MeasurementUnits
	if units.CodeValue != "" {
		c := units.CodeValue
		q.Code = &c
	}
	if units.CodeMeaning != "" {
		u := units.CodeMeaning
		q.Unit = &u
	}
	if units.CodingSchemeDesignator != "" {
		system := measurementUnitSystem(units.CodingSchemeDesignator)
		q.System = &system
	}
	return q, true
}
