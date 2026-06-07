package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// SRToDiagnosticReportR4 converts a DICOM Structured Report document dataset to a
// FHIR R4 DiagnosticReport and the set of Observations carrying its measurements,
// the R4 twin of SRToDiagnosticReportR5. The SR parsing, content-tree walk,
// narrative/measurement classification, deterministic urn:uuid result linking, and
// the identity rule are identical (the content walk and UUID derivation are
// release-agnostic and shared). The DiagnosticReport and Observation resource
// models for the v1 mapping's fields are the same shape in R4 and R5, so the twin
// differs only in the release sub-package its types come from.
//
// The SOP Instance UID becomes the report identifier via UIDIdentifierR4 (never a
// Reference URL). subject carries the DICOM PatientID logically, or the
// WithSubjectR4 reference when supplied. Attributes outside the supported mapping
// are recorded in Report.Dropped.
func SRToDiagnosticReportR4(sr *dicom.DataSet, opts ...Option) (*r4.DiagnosticReport, []*r4.Observation, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if sr == nil {
		return nil, nil, nil, fmt.Errorf("%w: SR dataset is nil", ErrMalformedSource)
	}

	root, err := dicom.ParseSR(sr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %v", ErrUnsupportedSource, err)
	}

	sopInstanceUID, ok := sr.GetUID(dicom.TagSOPInstanceUID)
	if !ok || sopInstanceUID == "" {
		return nil, nil, nil, fmt.Errorf("%w: SR has no SOP Instance UID (0008,0018)", ErrMissingIdentifier)
	}

	dr := &r4.DiagnosticReport{}
	id := UIDIdentifierR4(sopInstanceUID)
	dr.Identifier = []r4.Identifier{id}

	status := srStatusR4(sr, report)
	dr.Status = &status
	dr.Category = []r4.CodeableConcept{imagingCategoryR4()}

	// DiagnosticReport.code is required by the R4 model; the root CONTAINER's
	// Concept Name Code Sequence is its only source. A supported SR that omits it
	// would produce an invalid resource, so fail closed rather than emit a
	// code-less report.
	code := conceptNameCodeR4(root.ConceptName)
	if code == nil {
		return nil, nil, nil, fmt.Errorf("%w: SR root has no Concept Name Code Sequence (0040,A043) for the required DiagnosticReport.code",
			ErrMalformedSource)
	}
	dr.Code = code

	if when := combineDateTime(sr, dicom.TagContentDate, dicom.TagContentTime, report, "DiagnosticReport.effectiveDateTime"); when != "" {
		dr.SetEffectiveDateTime(r4.FHIRDateTime(when))
	}

	if conclusion := narrative(root); conclusion != "" {
		dr.Conclusion = &conclusion
	}

	dr.Subject = dicomPatientSubjectR4(cfg, sr, report, "DiagnosticReport.subject")

	observations := measurementObservationsR4(root, sr, string(sopInstanceUID), report)
	for _, o := range observations {
		ref := "urn:uuid:" + *o.ID
		dr.Result = append(dr.Result, r4.Reference{Reference: &ref})
	}

	rep, ferr := cfg.finalize(report)
	return dr, observations, rep, ferr
}

// measurementObservationsR4 walks the SR content tree and emits one R4 Observation
// per measurement leaf, the R4 twin of measurementObservations. The walk, the
// narrative/measurement classification (the shared isConclusionText), and the
// deterministic urn:uuid id (the shared deterministicObservationUUID) are
// identical; only the leaf converter and the Observation type are R4.
func measurementObservationsR4(root *dicom.ContentItem, ds *dicom.DataSet, sopInstanceUID string, report *Report) []*r4.Observation {
	tzOffset, hasTZ := fhirTimezoneOffset(ds)
	var observations []*r4.Observation
	index := 0
	var walk func(items []dicom.ContentItem)
	walk = func(items []dicom.ContentItem) {
		for i := range items {
			it := items[i]
			if !isConclusionText(it) {
				if o, ok := contentItemToObservationR4(it, tzOffset, hasTZ, report); ok {
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

// srStatusR4 maps CompletionFlag (0040,A491) and VerificationFlag (0040,A493) to an
// R4 DiagnosticReport.status, the R4 twin of srStatus. The mapping is identical.
func srStatusR4(sr *dicom.DataSet, report *Report) r4.DiagnosticReportStatus {
	completion, hasCompletion := sr.GetString(dicom.TagCompletionFlag)
	verification, _ := sr.GetString(dicom.TagVerificationFlag)
	if !hasCompletion {
		report.defaulted("DiagnosticReport.status", "preliminary",
			"SR has no CompletionFlag (0040,A491); defaulted")
		return r4.DiagnosticReportStatusPreliminary
	}
	if completion == "COMPLETE" && verification == "VERIFIED" {
		return r4.DiagnosticReportStatusFinal
	}
	return r4.DiagnosticReportStatusPreliminary
}

// imagingCategoryR4 builds the imaging DiagnosticReport category, the R4 twin of
// imagingCategory.
func imagingCategoryR4() r4.CodeableConcept {
	system := imagingCategorySystem
	code := imagingCategoryCode
	display := imagingCategoryDisplay
	return r4.CodeableConcept{
		Coding: []r4.Coding{{System: &system, Code: &code, Display: &display}},
	}
}

// conceptNameCodeR4 maps a DICOM SR ConceptNameCode triplet to an R4
// CodeableConcept, the R4 twin of conceptNameCode. The scheme designator resolves
// through the shared schemeDesignatorSystem helper.
func conceptNameCodeR4(c dicom.ConceptNameCode) *r4.CodeableConcept {
	if c.IsZero() {
		return nil
	}
	coding := r4.Coding{}
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
	cc := &r4.CodeableConcept{Coding: []r4.Coding{coding}}
	if c.CodeMeaning != "" {
		text := c.CodeMeaning
		cc.Text = &text
	}
	return cc
}
