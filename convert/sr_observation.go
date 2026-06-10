package convert

import (
	"crypto/sha1" // #nosec G505 -- RFC 4122 version-5 UUIDs are defined over SHA-1; identifier derivation, not a security primitive
	"encoding/binary"
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// ucumSystem is the UCUM unit-of-measure system URI a NUM measurement's units carry
// when their coding-scheme designator is UCUM. FHIR Quantity binds its code to UCUM,
// so a measurement's units map to Quantity.code (the UCUM symbol), Quantity.unit (the
// human-readable meaning), and Quantity.system (this URI).
const ucumSystem = "http://unitsofmeasure.org"

// ContentItemToObservationR5 maps one DICOM SR content-item leaf to a FHIR R5
// Observation. The leaf's Concept Name Code Sequence becomes the required
// Observation.code, and the leaf value maps to value[x] by ValueType: NUM to
// valueQuantity (MeasuredValue plus MeasurementUnits), CODE to valueCodeableConcept,
// TEXT to valueString, TIME to valueTime, and DATE/DATETIME to valueDateTime. The
// Observation is emitted with status "final"; the SR document's verification state
// governs the enclosing DiagnosticReport.status, while a leaf observation is a
// completed measurement.
//
// ok is false for a content item that is structure rather than an observation — a
// CONTAINER, a spatial/temporal coordinate, a waveform, or a referenced instance —
// or for a leaf whose value could not be rendered into a conformant value[x] (for
// example a date-only-precision DATETIME with a time but no timezone, or a NUM with
// no numeric value). The returned Observation has no id; the caller assigns the
// deterministic urn:uuid logical id it links from DiagnosticReport.result.
func ContentItemToObservationR5(item dicom.ContentItem, _ ...Option) (*r5.Observation, bool) {
	return contentItemToObservationR5(item, "", false, &Report{})
}

// contentItemToObservationR5 is the loss-aware leaf converter. tzOffset is the FHIR
// timezone suffix resolved from the dataset's TimezoneOffsetFromUTC (0008,0201);
// hasTZ reports whether the dataset carried one. A DATETIME leaf without an inline
// offset borrows the dataset offset so a timestamped value is not degraded to
// date-only when the document states its zone. A leaf whose value cannot become a
// conformant value[x] is dropped on report rather than emitted as a null-valued
// resource.
func contentItemToObservationR5(item dicom.ContentItem, tzOffset string, hasTZ bool, report *Report) (*r5.Observation, bool) {
	code := conceptNameCode(item.ConceptName)
	if code == nil {
		// Without a concept name there is no Observation.code, which is required; the
		// leaf is not a conformant observation on its own.
		return nil, false
	}

	o := &r5.Observation{Code: code}
	status := r5.ObservationStatusFinal
	o.Status = &status

	switch item.ValueType {
	case dicom.ValueTypeNum:
		q, ok := measurementQuantity(item)
		if !ok {
			// A Quantity without a value would marshal valueQuantity.value: null, which
			// is non-conformant; drop the leaf and record the loss instead.
			report.dropped(dicomTagSource(dicom.TagNumericValue),
				"NUM content item has no usable numeric value; a value-less Quantity is non-conformant, so the Observation was dropped")
			return nil, false
		}
		o.SetValueQuantity(q)
	case dicom.ValueTypeCode:
		cc := conceptNameCode(item.Code)
		if cc == nil {
			return nil, false
		}
		o.SetValueCodeableConcept(*cc)
	case dicom.ValueTypeText:
		o.SetValueString(r5.FHIRString(item.Text))
	case dicom.ValueTypeTime:
		when, ok := srTimeToFHIR(item.DateTime)
		if !ok {
			return nil, false
		}
		o.SetValueTime(r5.FHIRTime(when))
	case dicom.ValueTypeDate, dicom.ValueTypeDateTime:
		when, ok := srDateTimeToFHIR(item.DateTime, tzOffset, hasTZ)
		if !ok {
			return nil, false
		}
		o.SetValueDateTime(r5.FHIRDateTime(when))
	default:
		return nil, false
	}

	return o, true
}

// measurementQuantity builds a FHIR Quantity from a NUM leaf's MeasuredValue and
// MeasurementUnits. The numeric value is carried as a Decimal so its lexical
// precision survives; the units' code, meaning, and scheme map to Quantity.code,
// Quantity.unit, and Quantity.system. A units scheme of UCUM resolves to the UCUM
// system URI; any other scheme is carried verbatim under its registered or
// urn:dicom:scheme: URI so the value is preserved. ok is false when the leaf carries
// no numeric value, since a Quantity with a null value is non-conformant.
func measurementQuantity(item dicom.ContentItem) (r5.Quantity, bool) {
	if item.MeasuredValue.String() == "" {
		return r5.Quantity{}, false
	}
	q := r5.Quantity{}
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

// measurementUnitSystem maps a measurement-units coding-scheme designator to its FHIR
// system URI. UCUM is the system FHIR Quantity binds to, so it resolves to the UCUM
// URI; every other scheme falls through to the shared schemeDesignatorSystem mapping.
func measurementUnitSystem(designator string) string {
	if designator == "UCUM" {
		return ucumSystem
	}
	return schemeDesignatorSystem(designator)
}

// srDateTimeToFHIR renders a DICOM SR DATE/DATETIME value as a FHIR dateTime. A
// date-only value yields "YYYY-MM-DD". A value carrying a time is rendered with a
// timezone, because FHIR dateTime forbids a timezone-less time: the inline &ZZXX
// offset takes precedence, and when the value omits one the dataset-level
// TimezoneOffsetFromUTC (0008,0201) supplied as datasetOffset is applied. Only when
// no offset is available at all does the value fall back to date-only. The lexical
// fractional second is preserved verbatim so a sub-second timestamp is not truncated.
// ok is false when the value lacks a full calendar date, so a partial-precision value
// is never widened into a fabricated date.
func srDateTimeToFHIR(dt dicom.DT, datasetOffset string, hasDatasetOffset bool) (string, bool) {
	if dt.Precision() < dicom.DTPrecisionDay {
		return "", false
	}
	t, ok := dt.Time()
	if !ok {
		return "", false
	}
	date := pad4(t.Year()) + "-" + pad2(int(t.Month())) + "-" + pad2(t.Day())
	if dt.Precision() < dicom.DTPrecisionHour {
		return date, true
	}

	offset := ""
	switch {
	case dt.HasOffset():
		offset = offsetSuffix(dt.OffsetSeconds())
	case hasDatasetOffset:
		offset = datasetOffset
	default:
		// No inline offset and no document-level offset: a timezone-less time has no
		// valid FHIR dateTime form, so fall back to the date rather than fabricate a
		// zone, preserving the value at day precision.
		return date, true
	}

	timePart := pad2(t.Hour()) + ":" + pad2(t.Minute()) + ":" + pad2(t.Second()) + fractionOf(dt.String())
	return date + "T" + timePart + offset, true
}

// srTimeToFHIR renders a DICOM SR TIME content value as a FHIR time ("hh:mm:ss" with
// an optional fractional second). The TIME value is parsed not as a DT but as a TM:
// an SR TIME content item carries a (0040,A122) Time value whose lexical form is the
// HHMMSS[.FFFFFF] of VR TM, which the content-tree parser preserves on the DT's
// lexical string even though it is not a valid DT. FHIR time carries no timezone, so
// the dataset offset does not apply. ok is false when the value does not parse as a
// time or lacks at least an hour.
func srTimeToFHIR(dt dicom.DT) (string, bool) {
	tm, err := dicom.ParseTM(dt.String())
	if err != nil {
		return "", false
	}
	t, ok := tm.Time()
	if !ok {
		return "", false
	}
	return pad2(t.Hour()) + ":" + pad2(t.Minute()) + ":" + pad2(t.Second()) + fractionOf(tm.String()), true
}

// offsetSuffix renders a signed UTC offset in seconds as a FHIR timezone suffix: "Z"
// for zero, otherwise "+hh:mm" or "-hh:mm".
func offsetSuffix(seconds int) string {
	if seconds == 0 {
		return "Z"
	}
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	hh := seconds / 3600
	mm := (seconds % 3600) / 60
	return sign + pad2(hh) + ":" + pad2(mm)
}

// observationUUIDNamespace is the fixed RFC 4122 namespace UUID under which an SR
// content item's logical Observation id is derived. It is a constant so the derived
// ids depend only on the SR's identity and the item's position, never on the process
// or the wall clock.
var observationUUIDNamespace = [16]byte{
	0x6b, 0xf6, 0x1e, 0x2c, 0x4a, 0x3d, 0x4e, 0x57,
	0x9b, 0x21, 0x0f, 0x8c, 0x1d, 0x2e, 0x3f, 0x40,
}

// deterministicObservationUUID derives an RFC 4122 version-5 (name-based, SHA-1) UUID
// from the SR SOP Instance UID and the content item's positional index, so the same SR
// always yields byte-identical Observation ids and the DiagnosticReport.result links
// are stable across runs. The derivation uses no randomness and no wall-clock time.
func deterministicObservationUUID(sopInstanceUID string, index int) string {
	var idx [8]byte
	binary.BigEndian.PutUint64(idx[:], uint64(index)) // #nosec G115 -- index is a non-negative positional counter

	h := sha1.New() // #nosec G401 -- RFC 4122 version-5 UUIDs are defined over SHA-1; this derives a stable identifier, not a security primitive
	h.Write(observationUUIDNamespace[:])
	h.Write([]byte(sopInstanceUID))
	h.Write([]byte{0}) // separator so a UID ending in digits cannot collide with the index
	h.Write(idx[:])
	sum := h.Sum(nil)

	var u [16]byte
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | 0x50 // version 5
	u[8] = (u[8] & 0x3f) | 0x80 // RFC 4122 variant

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
