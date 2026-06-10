package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// DICOMToImagingStudyR4 builds a FHIR R4 ImagingStudy from one or more DICOM
// instances of the same study, the R4 twin of DICOMToImagingStudyR5. The DICOM
// reading, series grouping, count recomputation, and instance de-duplication are
// identical; the R4 output differs in three load-bearing ways the R4 resource
// model imposes:
//
//   - ImagingStudy.modality and series.modality are Coding in R4 (CodeableConcept
//     in R5), so the modality maps to a single Coding rather than a concept.
//   - R4 has no CodeableReference: where R5 carries the reason as
//     ImagingStudy.reason (CodeableReference), R4 splits it into reasonCode
//     (CodeableConcept) for coded and free-text reasons and reasonReference for a
//     resolvable reason; this converter populates reasonCode only.
//   - series.bodySite and series.laterality are Coding in R4 (CodeableConcept in R5).
//
// The Study Instance UID becomes the study identifier via UIDIdentifierR4 (never a
// Reference URL). subject carries the DICOM PatientID logically, or the
// WithSubjectR4 reference when supplied. status has no DICOM source and is
// defaulted to "available". Attributes outside the General Study/Series/SOP-Common
// modules are recorded in Report.Dropped.
func DICOMToImagingStudyR4(instances []*dicom.DataSet, opts ...Option) (*r4.ImagingStudy, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if len(instances) == 0 {
		return nil, nil, fmt.Errorf("%w: no instances supplied", ErrMalformedSource)
	}

	studyUID, ok := studyInstanceUID(instances)
	if !ok {
		return nil, nil, fmt.Errorf("%w: instances have no Study Instance UID (0020,000D)", ErrMissingIdentifier)
	}

	study := &r4.ImagingStudy{}
	study.Identifier = []r4.Identifier{UIDIdentifierR4(studyUID)}

	status := r4.ImagingStudyStatusAvailable
	study.Status = &status
	report.defaulted("ImagingStudy.status", "available", "DICOM has no study status; defaulted per convert.md")

	first := instances[0]
	if when := combineDateTime(first, dicom.TagStudyDate, dicom.TagStudyTime, report, "ImagingStudy.started"); when != "" {
		study.Started = &when
	}
	if desc, has := first.GetString(dicom.TagStudyDescription); has && desc != "" {
		d := desc
		study.Description = &d
	}
	if ref := referrerReferenceR4(first); ref != nil {
		study.Referrer = ref
	}
	study.ProcedureCode = codeSequenceConceptsR4(first, dicom.TagProcedureCodeSequence)
	study.ReasonCode = appendStudyReasonR4(study.ReasonCode, first, report)

	series, modalities := groupSeriesR4(instances, report)
	study.Series = series

	numSeries := int32(len(series)) // #nosec G115 -- a study's in-memory series count is far below int32
	study.NumberOfSeries = &numSeries
	var numInstances int32
	for i := range series {
		numInstances += int32(len(series[i].Instance)) // #nosec G115 -- a series' in-memory instance count is far below int32
	}
	study.NumberOfInstances = &numInstances

	// Study-level modality is the union of the series modalities (R4 Coding).
	for _, m := range modalities {
		study.Modality = append(study.Modality, modalityCodingR4(m))
	}

	study.Subject = dicomPatientSubjectR4(cfg, first, report, "ImagingStudy.subject")

	rep, err := cfg.finalize(report)
	return study, rep, err
}

// groupSeriesR4 groups the instances by Series Instance UID, then by SOP Instance
// UID within each series, the R4 twin of groupSeries. The grouping logic is
// identical; the emitted series carry R4 Coding modality, bodySite, and laterality.
func groupSeriesR4(instances []*dicom.DataSet, report *Report) ([]r4.ImagingStudySeries, []string) {
	type seriesAcc struct {
		series   *r4.ImagingStudySeries
		seenSOP  map[string]struct{}
		modality string
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
			s := newSeriesR4(ds, seriesUID, report)
			acc = &seriesAcc{series: &s, seenSOP: make(map[string]struct{})}
			accByUID[seriesUID] = acc
			order = append(order, seriesUID)
		}
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
			continue
		}
		inst, ok := newInstanceR4(ds, sopUID, report)
		if !ok {
			continue
		}
		acc.seenSOP[sopUID] = struct{}{}
		acc.series.Instance = append(acc.series.Instance, inst)
	}

	out := make([]r4.ImagingStudySeries, 0, len(order))
	modalityOrder := make([]string, 0, 4)
	seenModality := make(map[string]struct{})
	for _, uid := range order {
		acc := accByUID[uid]
		if acc.modality == "" {
			report.dropped("DICOM (0008,0060) Modality",
				"no instance of series "+uid+" carries Modality, which series.modality requires; series dropped")
			continue
		}
		modality := modalityCodingR4(acc.modality)
		acc.series.Modality = &modality
		if _, dup := seenModality[acc.modality]; !dup {
			seenModality[acc.modality] = struct{}{}
			modalityOrder = append(modalityOrder, acc.modality)
		}
		out = append(out, *acc.series)
	}
	return out, modalityOrder
}

// newSeriesR4 builds an R4 ImagingStudySeries from one instance's series-level
// attributes, the R4 twin of newSeries. The required modality is resolved by
// groupSeriesR4. bodySite and laterality are R4 Coding (CodeableConcept in R5).
func newSeriesR4(ds *dicom.DataSet, seriesUID string, report *Report) r4.ImagingStudySeries {
	uid := seriesUID
	s := r4.ImagingStudySeries{UID: &uid}

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
		s.BodySite = bodyPartCodingR4(body)
	}
	if lat, has := ds.GetString(dicom.TagLaterality); has && lat != "" {
		s.Laterality = lateralityCodingR4(lat)
	}
	if when := combineDateTime(ds, dicom.TagSeriesDate, dicom.TagSeriesTime, report, "ImagingStudy.series.started"); when != "" {
		s.Started = &when
	}
	return s
}

// newInstanceR4 builds an R4 ImagingStudySeriesInstance from one instance's
// SOP-Common attributes, the R4 twin of newInstance. The SOP Class UID becomes a
// Coding under urn:ietf:rfc:3986. ok is false when the dataset carries no SOP
// Class UID (the required series.instance.sopClass), so the caller drops the
// instance rather than emit an invalid resource.
func newInstanceR4(ds *dicom.DataSet, sopUID string, report *Report) (r4.ImagingStudySeriesInstance, bool) {
	class, has := ds.GetString(dicom.TagSOPClassUID)
	if !has || class == "" {
		report.dropped("DICOM (0008,0016) SOPClassUID",
			"instance has no SOP Class UID, which series.instance.sopClass requires; instance dropped")
		return r4.ImagingStudySeriesInstance{}, false
	}

	uid := sopUID
	inst := r4.ImagingStudySeriesInstance{UID: &uid}
	system := rfc3986System
	code := "urn:oid:" + class
	inst.SopClass = &r4.Coding{System: &system, Code: &code}

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

// modalityCodingR4 builds an R4 Coding for a DICOM Modality CS value under the
// DICOM modality code system, the R4 counterpart of modalityConcept. R4 binds
// ImagingStudy.modality and series.modality to Coding, not CodeableConcept.
func modalityCodingR4(modality string) r4.Coding {
	system := dicomDCMSystem
	code := modality
	return r4.Coding{System: &system, Code: &code}
}

// bodyPartCodingR4 builds an R4 Coding for a DICOM BodyPartExamined CS value. The
// value is carried as the Coding.code under the DICOM body-part code system so the
// defined term is preserved; R4 binds series.bodySite to Coding.
func bodyPartCodingR4(body string) *r4.Coding {
	system := dicomBodyPartSystem
	code := body
	return &r4.Coding{System: &system, Code: &code}
}

// dicomBodyPartSystem is the FHIR system URI for the DICOM body-part-examined
// defined-term code system. R4 carries BodyPartExamined as a Coding, so the value
// needs a system rather than the free-text concept R5 uses.
const dicomBodyPartSystem = "http://dicom.nema.org/resources/ontology/DCM"

// lateralityCodingR4 builds an R4 Coding for a DICOM Laterality CS value under the
// DICOM coding system, the R4 counterpart of lateralityConcept.
func lateralityCodingR4(laterality string) *r4.Coding {
	system := dicomDCMSystem
	code := laterality
	return &r4.Coding{System: &system, Code: &code}
}

// referrerReferenceR4 builds the ImagingStudy.referrer Reference from
// ReferringPhysicianName (0008,0090), the R4 twin of referrerReference. The name
// is carried as Reference.display only — never a fabricated Reference.reference URL
// (the identity rule).
func referrerReferenceR4(ds *dicom.DataSet) *r4.Reference {
	name, has := ds.GetString(dicom.TagReferringPhysicianName)
	if !has || name == "" {
		return nil
	}
	display := name
	refType := practitionerReferenceType
	return &r4.Reference{Type: &refType, Display: &display}
}

// appendStudyReasonR4 appends the ImagingStudy.reasonCode CodeableConcepts derived
// from the coded ReasonForRequestedProcedureCodeSequence (0040,100A) and the
// free-text ReasonForStudy (0032,1030). It is the R4 counterpart of
// appendStudyReason: where R5 carries both under the single reason
// CodeableReference, R4 has no CodeableReference, so a coded or free-text reason
// becomes a reasonCode CodeableConcept (reasonReference is reserved for a
// resolvable reason the source does not carry).
func appendStudyReasonR4(reasons []r4.CodeableConcept, ds *dicom.DataSet, report *Report) []r4.CodeableConcept {
	reasons = append(reasons, codeSequenceConceptsR4(ds, dicom.TagReasonForRequestedProcedureCodeSequence)...)
	if text, has := ds.GetString(dicom.TagReasonForStudy); has && text != "" {
		t := text
		reasons = append(reasons, r4.CodeableConcept{Text: &t})
	}
	return reasons
}

// codeSequenceConceptsR4 reads a DICOM code sequence at tag t and renders each item
// as an R4 CodeableConcept, the R4 twin of codeSequenceConcepts (which produces R5
// CodeableReferences). The item-reading helper codeItemValue is release-agnostic
// and shared.
func codeSequenceConceptsR4(ds *dicom.DataSet, t dicom.Tag) []r4.CodeableConcept {
	seq, ok := ds.GetSequence(t)
	if !ok {
		return nil
	}
	var out []r4.CodeableConcept
	for item := range seq.Items() {
		if item.DataSet == nil {
			continue
		}
		if concept := codeItemConceptR4(item.DataSet); concept != nil {
			out = append(out, *concept)
		}
	}
	return out
}

// codeItemConceptR4 maps one DICOM coded-entry item to an R4 CodeableConcept, the
// R4 twin of codeItemConcept. The shared codeItemValue reads the three standard
// Code Sequence Macro value forms; the scheme designator resolves through the
// shared schemeDesignatorSystem helper.
func codeItemConceptR4(item *dicom.DataSet) *r4.CodeableConcept {
	value, has := codeItemValue(item)
	if !has {
		return nil
	}
	coding := r4.Coding{}
	code := value
	coding.Code = &code
	if scheme, ok := item.GetString(dicom.TagCodingSchemeDesignator); ok && scheme != "" {
		system := schemeDesignatorSystem(scheme)
		coding.System = &system
	}
	cc := &r4.CodeableConcept{}
	if meaning, ok := item.GetString(dicom.TagCodeMeaning); ok && meaning != "" {
		display := meaning
		coding.Display = &display
		text := meaning
		cc.Text = &text
	}
	cc.Coding = []r4.Coding{coding}
	return cc
}
