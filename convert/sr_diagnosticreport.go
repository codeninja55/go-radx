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
// FHIR R5 DiagnosticReport. This M2 slice is narrative-only: it maps the document
// identity, status, code, category, effective date/time, subject, and the
// concatenated TEXT narrative to DiagnosticReport.conclusion. The full
// measurement walk that emits []*r5.Observation from the SR content items is
// deferred to M7 (docs/plans walking-skeleton, Increment 12 reviewer correction).
//
// The SOP Instance UID becomes the report identifier via UIDIdentifierR5 (never a
// Reference URL). subject carries the DICOM PatientID logically, or the
// WithSubjectR5 reference when supplied. Attributes outside the narrow
// narrative-only mapping are recorded in Report.Dropped.
func SRToDiagnosticReportR5(sr *dicom.DataSet, opts ...Option) (*r5.DiagnosticReport, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if sr == nil {
		return nil, nil, fmt.Errorf("%w: SR dataset is nil", ErrMalformedSource)
	}

	root, err := dicom.ParseSR(sr)
	if err != nil {
		// ParseSR fails closed on a non-SR IOD or a malformed tree; surface it as
		// an unsupported source so the caller does not get a partial report.
		return nil, nil, fmt.Errorf("%w: %v", ErrUnsupportedSource, err)
	}

	sopInstanceUID, ok := sr.GetUID(dicom.TagSOPInstanceUID)
	if !ok || sopInstanceUID == "" {
		return nil, nil, fmt.Errorf("%w: SR has no SOP Instance UID (0008,0018)", ErrMissingIdentifier)
	}

	dr := &r5.DiagnosticReport{}
	id := UIDIdentifierR5(sopInstanceUID)
	dr.Identifier = []r5.Identifier{id}

	dr.Status = srStatus(sr, report)
	dr.Category = []r5.CodeableConcept{imagingCategory()}

	// DiagnosticReport.code is required by the R5 model; the root CONTAINER's
	// Concept Name Code Sequence is its only source. A supported SR that omits it
	// would produce an invalid resource, so fail closed rather than emit a
	// code-less report.
	code := conceptNameCode(root.ConceptName)
	if code == nil {
		return nil, nil, fmt.Errorf("%w: SR root has no Concept Name Code Sequence (0040,A043) for the required DiagnosticReport.code",
			ErrMalformedSource)
	}
	dr.Code = code

	if when := combineDateTime(sr, dicom.TagContentDate, dicom.TagContentTime, report, "DiagnosticReport.effectiveDateTime"); when != "" {
		dr.EffectiveDateTime = &when
	}

	if conclusion := narrative(root); conclusion != "" {
		dr.Conclusion = &conclusion
	}

	dr.Subject = dicomPatientSubjectR5(cfg, sr, report, "DiagnosticReport.subject")

	rep, ferr := cfg.finalize(report)
	return dr, rep, ferr
}

// srStatus maps CompletionFlag (0040,A491) and VerificationFlag (0040,A493) to a
// FHIR DiagnosticReport.status. COMPLETE+VERIFIED is final; a PARTIAL or
// UNVERIFIED document is preliminary. An absent CompletionFlag defaults to
// preliminary and is recorded.
func srStatus(sr *dicom.DataSet, report *Report) string {
	completion, hasCompletion := sr.GetString(dicom.TagCompletionFlag)
	verification, _ := sr.GetString(dicom.TagVerificationFlag)
	if !hasCompletion {
		report.defaulted("DiagnosticReport.status", "preliminary",
			"SR has no CompletionFlag (0040,A491); defaulted")
		return "preliminary"
	}
	if completion == "COMPLETE" && verification == "VERIFIED" {
		return "final"
	}
	return "preliminary"
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
