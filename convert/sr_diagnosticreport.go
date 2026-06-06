package convert

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// imagingCategorySystem is the FHIR code system for the DiagnosticReport service
// section category. An SR document (Modality SR) is an imaging report (IMG).
const (
	imagingCategorySystem  = "http://terminology.hl7.org/CodeSystem/v2-0074"
	imagingCategoryCode    = "IMG"
	imagingCategoryDisplay = "Diagnostic Imaging"
)

// SRToDiagnosticReportR5 converts a DICOM Structured Report document dataset to a
// FHIR R5 DiagnosticReport and the set of Observations carrying its measurements. It
// maps the document identity, status, code, category, effective date/time, subject,
// and the concatenated TEXT narrative to DiagnosticReport.conclusion, then walks the
// SR content tree for measurement leaves: each NUM, CODE, and date/time leaf becomes
// one Observation (via ContentItemToObservationR5), linked from DiagnosticReport.result
// by an intra-call urn:uuid logical reference. TEXT items remain the narrative
// conclusion rather than separate observations, so a finding is represented once.
//
// The result links are derived deterministically from the SR SOP Instance UID and each
// leaf's position, so the same input yields byte-identical output. The SOP Instance UID
// becomes the report identifier via UIDIdentifierR5 (never a Reference URL). subject
// carries the DICOM PatientID logically, or the WithSubjectR5 reference when supplied.
// Attributes outside the supported mapping are recorded in Report.Dropped.
func SRToDiagnosticReportR5(sr *dicom.DataSet, opts ...Option) (*r5.DiagnosticReport, []*r5.Observation, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if sr == nil {
		return nil, nil, nil, fmt.Errorf("%w: SR dataset is nil", ErrMalformedSource)
	}

	root, err := dicom.ParseSR(sr)
	if err != nil {
		// ParseSR fails closed on a non-SR IOD or a malformed tree; surface it as
		// an unsupported source so the caller does not get a partial report.
		return nil, nil, nil, fmt.Errorf("%w: %v", ErrUnsupportedSource, err)
	}

	sopInstanceUID, ok := sr.GetUID(dicom.TagSOPInstanceUID)
	if !ok || sopInstanceUID == "" {
		return nil, nil, nil, fmt.Errorf("%w: SR has no SOP Instance UID (0008,0018)", ErrMissingIdentifier)
	}

	dr := &r5.DiagnosticReport{}
	id := UIDIdentifierR5(sopInstanceUID)
	dr.Identifier = []r5.Identifier{id}

	status := srStatus(sr, report)
	dr.Status = &status
	dr.Category = []r5.CodeableConcept{imagingCategory()}

	// DiagnosticReport.code is required by the R5 model; the root CONTAINER's
	// Concept Name Code Sequence is its only source. A supported SR that omits it
	// would produce an invalid resource, so fail closed rather than emit a
	// code-less report.
	code := conceptNameCode(root.ConceptName)
	if code == nil {
		return nil, nil, nil, fmt.Errorf("%w: SR root has no Concept Name Code Sequence (0040,A043) for the required DiagnosticReport.code",
			ErrMalformedSource)
	}
	dr.Code = code

	if when := combineDateTime(sr, dicom.TagContentDate, dicom.TagContentTime, report, "DiagnosticReport.effectiveDateTime"); when != "" {
		// effective[x] is a choice group; the sealed setter picks the dateTime branch
		// and clears every sibling, so the resource never holds two branches at once.
		dr.SetEffectiveDateTime(r5.FHIRDateTime(when))
	}

	if conclusion := narrative(root); conclusion != "" {
		dr.Conclusion = &conclusion
	}

	dr.Subject = dicomPatientSubjectR5(cfg, sr, report, "DiagnosticReport.subject")

	observations := measurementObservations(root, sr, string(sopInstanceUID), report)
	for _, o := range observations {
		ref := "urn:uuid:" + *o.ID
		dr.Result = append(dr.Result, r5.Reference{Reference: &ref})
	}

	rep, ferr := cfg.finalize(report)
	return dr, observations, rep, ferr
}

// measurementObservations walks the SR content tree depth-first in document order and
// emits one Observation per measurement leaf (NUM, CODE, or date/time), assigning each
// a deterministic urn:uuid logical id derived from the SR SOP Instance UID and the
// leaf's position. TEXT leaves are skipped here: they form the narrative conclusion,
// not separate observations. CONTAINER and coordinate/reference items are structure and
// produce no observation. The dataset's TimezoneOffsetFromUTC (0008,0201) is resolved
// once and applied to any DATETIME leaf lacking an inline offset; leaves whose value
// cannot become a conformant value[x] are recorded on report rather than emitted.
func measurementObservations(root *dicom.ContentItem, ds *dicom.DataSet, sopInstanceUID string, report *Report) []*r5.Observation {
	tzOffset, hasTZ := fhirTimezoneOffset(ds)
	var observations []*r5.Observation
	index := 0
	var walk func(items []dicom.ContentItem)
	walk = func(items []dicom.ContentItem) {
		for i := range items {
			it := items[i]
			if it.ValueType != dicom.ValueTypeText {
				if o, ok := contentItemToObservationR5(it, tzOffset, hasTZ, report); ok {
					id := deterministicObservationUUID(sopInstanceUID, index)
					o.ID = &id
					observations = append(observations, o)
					index++
				}
			}
			walk(it.Children)
		}
	}
	walk(root.Children)
	return observations
}

// srStatus maps CompletionFlag (0040,A491) and VerificationFlag (0040,A493) to a
// FHIR DiagnosticReport.status. COMPLETE+VERIFIED is final; a PARTIAL or
// UNVERIFIED document is preliminary. An absent CompletionFlag defaults to
// preliminary and is recorded.
func srStatus(sr *dicom.DataSet, report *Report) r5.DiagnosticReportStatus {
	completion, hasCompletion := sr.GetString(dicom.TagCompletionFlag)
	verification, _ := sr.GetString(dicom.TagVerificationFlag)
	if !hasCompletion {
		report.defaulted("DiagnosticReport.status", "preliminary",
			"SR has no CompletionFlag (0040,A491); defaulted")
		return r5.DiagnosticReportStatusPreliminary
	}
	if completion == "COMPLETE" && verification == "VERIFIED" {
		return r5.DiagnosticReportStatusFinal
	}
	return r5.DiagnosticReportStatusPreliminary
}

// imagingCategory builds the imaging DiagnosticReport category.
func imagingCategory() r5.CodeableConcept {
	system := imagingCategorySystem
	code := imagingCategoryCode
	display := imagingCategoryDisplay
	return r5.CodeableConcept{
		Coding: []r5.Coding{{System: &system, Code: &code, Display: &display}},
	}
}

// conceptNameCode maps a DICOM SR ConceptNameCode triplet to a FHIR
// CodeableConcept with one Coding, or nil when the concept is empty. The scheme
// designator is carried as the system verbatim; go-radx does not translate code
// systems.
func conceptNameCode(c dicom.ConceptNameCode) *r5.CodeableConcept {
	if c.IsZero() {
		return nil
	}
	coding := r5.Coding{}
	if c.CodeValue != "" {
		code := c.CodeValue
		coding.Code = &code
	}
	if c.CodingSchemeDesignator != "" {
		system := schemeDesignatorSystem(c.CodingSchemeDesignator)
		coding.System = &system
	}
	if c.CodeMeaning != "" {
		display := c.CodeMeaning
		coding.Display = &display
	}
	cc := &r5.CodeableConcept{Coding: []r5.Coding{coding}}
	if c.CodeMeaning != "" {
		text := c.CodeMeaning
		cc.Text = &text
	}
	return cc
}

// schemeDesignatorSystem maps a DICOM coding-scheme designator to its registered
// FHIR system URI. An unknown designator is carried under urn:dicom:scheme: so
// the value is preserved rather than dropped (docs/reference/convert.md).
func schemeDesignatorSystem(designator string) string {
	switch designator {
	case "DCM":
		return "http://dicom.nema.org/resources/ontology/DCM"
	case "SCT":
		return "http://snomed.info/sct"
	case "LN":
		return "http://loinc.org"
	default:
		return "urn:dicom:scheme:" + designator
	}
}

// narrative concatenates the TEXT content items of the SR tree into a single
// markdown conclusion, in document order. Non-text items contribute structure,
// not narrative, and are skipped here (their full mapping is M7).
func narrative(root *dicom.ContentItem) string {
	var parts []string
	var walk func(items []dicom.ContentItem)
	walk = func(items []dicom.ContentItem) {
		for i := range items {
			it := items[i]
			if it.ValueType == dicom.ValueTypeText && strings.TrimSpace(it.Text) != "" {
				parts = append(parts, it.Text)
			}
			walk(it.Children)
		}
	}
	walk(root.Children)
	return strings.Join(parts, "\n\n")
}
