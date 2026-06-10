package convert

import (
	"fmt"
	"math"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// rfc3986System is the SOP Class UID Coding system for ImagingStudy series
// instances (an OID expressed as a URN), per the R5 mapping table.
const rfc3986System = "urn:ietf:rfc:3986"

// maxUnsignedInt is the largest value the generated FHIR unsignedInt field (an
// *int32) can hold. FHIR unsignedInt is a non-negative 32-bit integer; the
// generated Go type is int32, so the representable maximum is math.MaxInt32. A
// DICOM IS value above it is not cast (the cast would overflow int32 and silently
// corrupt the number); it is dropped and recorded.
const maxUnsignedInt = int64(math.MaxInt32)

// DICOMToImagingStudyR5 builds a FHIR R5 ImagingStudy from one or more DICOM
// instances of the same study. Pass every available instance dataset; the
// converter groups by Series Instance UID and SOP Instance UID, recomputes
// numberOfSeries/numberOfInstances from the distinct UIDs it sees, and
// de-duplicates instances. A single-instance call is valid.
//
// The Study Instance UID becomes the study identifier via UIDIdentifierR5 (never
// a Reference URL). subject carries the DICOM PatientID logically, or the
// WithSubjectR5 reference when supplied — it is never a fabricated
// Reference.reference URL (the identity rule). status has no DICOM source and is
// defaulted to "available", recorded in Report.Defaulted. Attributes outside the
// General Study/Series/SOP-Common modules are recorded in Report.Dropped:
// ImagingStudy is an index, not a copy of the dataset.
func DICOMToImagingStudyR5(instances []*dicom.DataSet, opts ...Option) (*r5.ImagingStudy, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if len(instances) == 0 {
		return nil, nil, fmt.Errorf("%w: no instances supplied", ErrMalformedSource)
	}

	studyUID, ok := studyInstanceUID(instances)
	if !ok {
		return nil, nil, fmt.Errorf("%w: instances have no Study Instance UID (0020,000D)", ErrMissingIdentifier)
	}

	study := &r5.ImagingStudy{}
	study.Identifier = []r5.Identifier{UIDIdentifierR5(studyUID)}

	// status has no DICOM source; default it and record the decision.
	status := r5.ImagingStudyStatusAvailable
	study.Status = &status
	report.defaulted("ImagingStudy.status", "available", "DICOM has no study status; defaulted per convert.md")

	// Read the study-level attributes from the first instance. These attributes are
	// identical across a study's instances in conformant data, so index 0 is
	// authoritative.
	// TODO(M7): repair a study-level gap (PatientID/StudyDate/StudyDescription
	// absent on instances[0] but present on a later instance) the way series
	// modality is repaired, for partial/malformed inputs (Codex round-3 finding).
	first := instances[0]
	if when := combineDateTime(first, dicom.TagStudyDate, dicom.TagStudyTime, report, "ImagingStudy.started"); when != "" {
		study.Started = &when
	}
	if desc, has := first.GetString(dicom.TagStudyDescription); has && desc != "" {
		d := desc
		study.Description = &d
	}
	if ref := referrerReference(first); ref != nil {
		study.Referrer = ref
	}
	study.Procedure = codeSequenceConcepts(first, dicom.TagProcedureCodeSequence)
	study.Reason = appendStudyReason(study.Reason, first, report)

	series, modalities := groupSeries(instances, report)
	study.Series = series

	// numberOfSeries/numberOfInstances are recomputed from the distinct UIDs seen,
	// never copied from a source attribute that may be stale.
	numSeries := int32(len(series)) // #nosec G115 -- a study's in-memory series count is far below int32
	study.NumberOfSeries = &numSeries
	var numInstances int32
	for i := range series {
		numInstances += int32(len(series[i].Instance)) // #nosec G115 -- a series' in-memory instance count is far below int32
	}
	study.NumberOfInstances = &numInstances

	// Study-level modality is the union of the series modalities (R5 CodeableConcept).
	for _, m := range modalities {
		study.Modality = append(study.Modality, modalityConcept(m))
	}

	study.Subject = dicomPatientSubjectR5(cfg, first, report, "ImagingStudy.subject")

	rep, err := cfg.finalize(report)
	return study, rep, err
}

// studyInstanceUID returns the Study Instance UID shared by the instances. The
// first non-empty value wins; this slice does not validate that every instance
// agrees (a mixed-study call is the caller's error to make).
func studyInstanceUID(instances []*dicom.DataSet) (dicom.UID, bool) {
	for _, ds := range instances {
		if uid, ok := ds.GetUID(dicom.TagStudyInstanceUID); ok && uid != "" {
			return uid, true
		}
	}
	return "", false
}

// groupSeries groups the instances by Series Instance UID, then by SOP Instance
// UID within each series, preserving first-seen order and de-duplicating
// instances. It returns the series list and the ordered set of distinct
// modalities seen across all series (for the study-level modality union).
func groupSeries(instances []*dicom.DataSet, report *Report) ([]r5.ImagingStudySeries, []string) {
	type seriesAcc struct {
		series   *r5.ImagingStudySeries
		seenSOP  map[string]struct{}
		modality string // resolved from the first instance that carries Modality
	}
	order := make([]string, 0, len(instances))
	accByUID := make(map[string]*seriesAcc)

	for _, ds := range instances {
		seriesUID, ok := ds.GetString(dicom.TagSeriesInstanceUID)
		if !ok || seriesUID == "" {
			report.dropped("DICOM (0020,000E) SeriesInstanceUID",
				"instance has no Series Instance UID and cannot be placed in a series")
			continue
		}

		acc := accByUID[seriesUID]
		if acc == nil {
			s := newSeries(ds, seriesUID, report)
			acc = &seriesAcc{series: &s, seenSOP: make(map[string]struct{})}
			accByUID[seriesUID] = acc
			order = append(order, seriesUID)
		}
		// series.modality is required; take it from the first instance of the
		// series that carries Modality, repairing a first-instance gap from a later
		// one rather than emitting an empty required element.
		if acc.modality == "" {
			if m, has := ds.GetString(dicom.TagModality); has && m != "" {
				acc.modality = m
			}
		}

		sopUID, ok := ds.GetString(dicom.TagSOPInstanceUID)
		if !ok || sopUID == "" {
			report.dropped("DICOM (0008,0018) SOPInstanceUID",
				"instance has no SOP Instance UID and cannot be recorded")
			continue
		}
		if _, dup := acc.seenSOP[sopUID]; dup {
			continue // de-duplicate a repeated SOP Instance UID
		}
		inst, ok := newInstance(ds, sopUID, report)
		if !ok {
			// series.instance.sopClass is required by the R5 model; an instance with
			// no SOP Class UID would produce an invalid resource, so drop it
			// (recorded) rather than emit an empty required field (fail-closed).
			continue
		}
		acc.seenSOP[sopUID] = struct{}{}
		acc.series.Instance = append(acc.series.Instance, inst)
	}

	out := make([]r5.ImagingStudySeries, 0, len(order))
	modalityOrder := make([]string, 0, 4)
	seenModality := make(map[string]struct{})
	for _, uid := range order {
		acc := accByUID[uid]
		if acc.modality == "" {
			// series.modality is required; no instance in this series carried a
			// Modality, so drop the whole series rather than emit an invalid one.
			report.dropped("DICOM (0008,0060) Modality",
				"no instance of series "+uid+" carries Modality, which series.modality requires; series dropped")
			continue
		}
		modality := modalityConcept(acc.modality)
		acc.series.Modality = &modality
		if _, dup := seenModality[acc.modality]; !dup {
			seenModality[acc.modality] = struct{}{}
			modalityOrder = append(modalityOrder, acc.modality)
		}
		out = append(out, *acc.series)
	}
	return out, modalityOrder
}

// newSeries builds an ImagingStudySeries from one instance's series-level
// attributes (the General Series module). The required modality is resolved by
// groupSeries (which can repair a first-instance gap from a later instance), so
// it is not set here. Number and optional descriptors are set when present.
func newSeries(ds *dicom.DataSet, seriesUID string, report *Report) r5.ImagingStudySeries {
	uid := seriesUID
	s := r5.ImagingStudySeries{UID: &uid}

	if n, has := ds.GetInt(dicom.TagSeriesNumber); has {
		switch {
		case n < 0:
			report.dropped("DICOM (0020,0011) SeriesNumber",
				"series number is negative; unsignedInt cannot represent it")
		case n > maxUnsignedInt:
			report.dropped("DICOM (0020,0011) SeriesNumber",
				"series number exceeds the FHIR unsignedInt range; not represented")
		default:
			num := int32(n)
			s.Number = &num
		}
	}
	if desc, has := ds.GetString(dicom.TagSeriesDescription); has && desc != "" {
		d := desc
		s.Description = &d
	}
	if body, has := ds.GetString(dicom.TagBodyPartExamined); has && body != "" {
		s.BodySite = &r5.CodeableReference{Concept: plainConcept(body)}
	}
	if lat, has := ds.GetString(dicom.TagLaterality); has && lat != "" {
		s.Laterality = lateralityConcept(lat)
	}
	if when := combineDateTime(ds, dicom.TagSeriesDate, dicom.TagSeriesTime, report, "ImagingStudy.series.started"); when != "" {
		s.Started = &when
	}
	return s
}

// newInstance builds an ImagingStudySeriesInstance from one instance's SOP-Common
// attributes. The SOP Class UID becomes a Coding under urn:ietf:rfc:3986 per the
// R5 mapping table. ok is false when the dataset carries no SOP Class UID: that
// element is required on series.instance, so the caller drops the instance rather
// than emit an invalid resource (the fail-closed rule); the absence is recorded.
func newInstance(ds *dicom.DataSet, sopUID string, report *Report) (r5.ImagingStudySeriesInstance, bool) {
	class, has := ds.GetString(dicom.TagSOPClassUID)
	if !has || class == "" {
		report.dropped("DICOM (0008,0016) SOPClassUID",
			"instance has no SOP Class UID, which series.instance.sopClass requires; instance dropped")
		return r5.ImagingStudySeriesInstance{}, false
	}

	uid := sopUID
	inst := r5.ImagingStudySeriesInstance{UID: &uid}
	system := rfc3986System
	code := "urn:oid:" + class
	inst.SopClass = &r5.Coding{System: &system, Code: &code}

	if n, has := ds.GetInt(dicom.TagInstanceNumber); has {
		switch {
		case n < 0:
			report.dropped("DICOM (0020,0013) InstanceNumber",
				"instance number is negative; unsignedInt cannot represent it")
		case n > maxUnsignedInt:
			report.dropped("DICOM (0020,0013) InstanceNumber",
				"instance number exceeds the FHIR unsignedInt range; not represented")
		default:
			num := int32(n)
			inst.Number = &num
		}
	}
	return inst, true
}

// modalityConcept builds a CodeableConcept for a DICOM Modality CS value. The
// modality code uses the DICOM modality code system; the value is carried
// verbatim.
func modalityConcept(modality string) r5.CodeableConcept {
	system := "http://dicom.nema.org/resources/ontology/DCM"
	code := modality
	return r5.CodeableConcept{Coding: []r5.Coding{{System: &system, Code: &code}}}
}

// plainConcept builds a CodeableConcept carrying only a free-text rendering, used
// for a DICOM CS value that has no coding system (e.g. BodyPartExamined).
func plainConcept(text string) *r5.CodeableConcept {
	t := text
	return &r5.CodeableConcept{Text: &t}
}

// dicomDCMSystem is the FHIR system URI for the DICOM controlled-terminology
// (DCM) coding scheme, the home of enumerated DICOM CS values such as Laterality.
const dicomDCMSystem = "http://dicom.nema.org/resources/ontology/DCM"

// lateralityConcept builds a CodeableConcept for a DICOM Laterality CS value
// (e.g. "L", "R"), carrying the code under the DICOM coding system.
func lateralityConcept(laterality string) *r5.CodeableConcept {
	system := dicomDCMSystem
	code := laterality
	return &r5.CodeableConcept{Coding: []r5.Coding{{System: &system, Code: &code}}}
}

// referrerReference builds the ImagingStudy.referrer Reference from
// ReferringPhysicianName (0008,0090). The name is carried as Reference.display
// only — a referring physician PN is not a server-resolvable identity, so it is
// never a fabricated Reference.reference URL (the identity rule). Absent or empty
// the reference is nil.
func referrerReference(ds *dicom.DataSet) *r5.Reference {
	name, has := ds.GetString(dicom.TagReferringPhysicianName)
	if !has || name == "" {
		return nil
	}
	display := name
	refType := practitionerReferenceType
	return &r5.Reference{Type: &refType, Display: &display}
}

// practitionerReferenceType is the FHIR resource type the referrer Reference
// points at.
const practitionerReferenceType = "Practitioner"

// appendStudyReason appends the ImagingStudy.reason CodeableReferences derived
// from the coded ReasonForRequestedProcedureCodeSequence (0040,100A) and the
// free-text ReasonForStudy (0032,1030). A coded reason becomes a concept Coding;
// the free-text reason becomes a text-only concept so the clinical reason is
// preserved without inventing a code.
func appendStudyReason(reasons []r5.CodeableReference, ds *dicom.DataSet, report *Report) []r5.CodeableReference {
	reasons = append(reasons, codeSequenceConcepts(ds, dicom.TagReasonForRequestedProcedureCodeSequence)...)
	if text, has := ds.GetString(dicom.TagReasonForStudy); has && text != "" {
		reasons = append(reasons, r5.CodeableReference{Concept: plainConcept(text)})
	}
	return reasons
}

// codeSequenceConcepts reads a DICOM code sequence (one or more coded-entry items,
// each carrying a code value plus CodingSchemeDesignator/CodeMeaning) at tag t and
// renders each item as a CodeableReference whose Concept carries the coded entry.
// An absent or non-sequence attribute yields nil; an item carrying none of the
// three standard code-value forms is skipped.
func codeSequenceConcepts(ds *dicom.DataSet, t dicom.Tag) []r5.CodeableReference {
	seq, ok := ds.GetSequence(t)
	if !ok {
		return nil
	}
	var out []r5.CodeableReference
	for item := range seq.Items() {
		if item.DataSet == nil {
			continue
		}
		if concept := codeItemConcept(item.DataSet); concept != nil {
			out = append(out, r5.CodeableReference{Concept: concept})
		}
	}
	return out
}

// codeItemConcept maps one DICOM coded-entry item to a FHIR CodeableConcept, with
// the CodingSchemeDesignator resolved to its registered FHIR system URI by the
// shared schemeDesignatorSystem helper. The DICOM Code Sequence Macro (PS3.3
// Table 8.8-1) admits three standard code-value forms — Code Value (0008,0100),
// Long Code Value (0008,0119), and URN Code Value (0008,0120) — exactly one of
// which is present per item; the value carries verbatim into Coding.code. nil when
// the item carries none of the three, so a malformed item never produces a
// value-less Coding.
func codeItemConcept(item *dicom.DataSet) *r5.CodeableConcept {
	value, has := codeItemValue(item)
	if !has {
		return nil
	}
	coding := r5.Coding{}
	code := value
	coding.Code = &code
	if scheme, ok := item.GetString(dicom.TagCodingSchemeDesignator); ok && scheme != "" {
		system := schemeDesignatorSystem(scheme)
		coding.System = &system
	}
	cc := &r5.CodeableConcept{}
	if meaning, ok := item.GetString(dicom.TagCodeMeaning); ok && meaning != "" {
		display := meaning
		coding.Display = &display
		text := meaning
		cc.Text = &text
	}
	cc.Coding = []r5.Coding{coding}
	return cc
}

// codeItemValue returns the code value of a DICOM coded-entry item, reading the
// three standard forms of the Code Sequence Macro in the order the standard lists
// them: Code Value, then Long Code Value, then URN Code Value. The first non-empty
// form wins. The boolean is false when the item carries none of them.
func codeItemValue(item *dicom.DataSet) (string, bool) {
	for _, t := range []dicom.Tag{dicom.TagCodeValue, dicom.TagLongCodeValue, dicom.TagURNCodeValue} {
		if v, ok := item.GetString(t); ok && v != "" {
			return v, true
		}
	}
	return "", false
}
