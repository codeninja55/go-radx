package convert

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// DiagnosticReportToSRR4 converts a FHIR R4 DiagnosticReport together with its
// Observations back to a DICOM Structured Report document, the R4 twin of
// DiagnosticReportToSR (R5) and the inverse of SRToDiagnosticReportR4. The report's
// code becomes the root CONTAINER's Concept Name Code Sequence; the conclusion
// becomes a bare (un-coded) TEXT child; each Observation becomes a measurement leaf
// through observationToContentItemR4. The document-level status, content date/time,
// and patient identity are written from the report's status, effectiveDateTime, and
// subject identifier.
//
// The Study, Series, and SOP Instance UIDs are minted deterministically under the
// WithUIDRoot organisation root, identically to the R5 reverse path (the UID minting
// is release-agnostic); absent a root the document carries no UIDs and the absence is
// recorded. A report whose code does not map to a Concept Name Code Sequence is
// rejected fail-closed (ErrMalformedSource).
func DiagnosticReportToSRR4(dr *r4.DiagnosticReport, observations []*r4.Observation, opts ...Option) (*dicom.DataSet, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if dr == nil {
		return nil, nil, fmt.Errorf("%w: DiagnosticReport is nil", ErrMalformedSource)
	}

	rootConcept := conceptNameForR4(dr.Code)
	if rootConcept.IsZero() {
		return nil, nil, fmt.Errorf("%w: DiagnosticReport.code does not map to a Concept Name Code Sequence (0040,A043) for the required SR document root",
			ErrMalformedSource)
	}

	root := &dicom.ContentItem{
		ValueType:   dicom.ValueTypeContainer,
		ConceptName: rootConcept,
	}

	if dr.Conclusion != nil && strings.TrimSpace(*dr.Conclusion) != "" {
		root.Children = append(root.Children, dicom.ContentItem{
			ValueType:    dicom.ValueTypeText,
			Relationship: dicom.RelationshipContains,
			Text:         *dr.Conclusion,
		})
	}

	for i := range observations {
		item, ok := observationToContentItemR4(observations[i], report)
		if !ok {
			report.dropped(
				fmt.Sprintf("Observation (result %d)", i+1),
				"the Observation did not re-encode as a DICOM SR content item; it was not added to the document",
			)
			continue
		}
		root.Children = append(root.Children, item)
	}

	ds := dicom.NewDataSet()
	if err := dicom.BuildSR(ds, root); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrMalformedSource, err)
	}

	ds.SetString(dicom.TagSOPClassUID, comprehensiveSRSOPClass)
	ds.SetString(dicom.TagModality, "SR")
	if err := mintSRIdentityR4(ds, cfg, dr, report); err != nil {
		return nil, nil, err
	}
	writeSRStatusR4(ds, dr)
	writeSRContentDateTimeR4(ds, dr)
	writeSRPatientIDR4(ds, dr, report)

	rep, err := cfg.finalize(report)
	return ds, rep, err
}

// mintSRIdentityR4 mints the Study, Series, and SOP Instance UIDs under the
// configured organisation root, the R4 twin of mintSRIdentity. The UID minting
// (mintUID) is release-agnostic and shared; only the identity seed reads R4 fields.
func mintSRIdentityR4(ds *dicom.DataSet, cfg config, dr *r4.DiagnosticReport, report *Report) error {
	if cfg.uidRoot == "" {
		report.defaulted("SOPInstanceUID (0008,0018)", "",
			"no WithUIDRoot was supplied and go-radx ships no default registered root; the SR document carries no minted UIDs")
		return nil
	}
	if _, err := dicom.NewUIDGenerator(cfg.uidRoot); err != nil {
		return fmt.Errorf("%w: WithUIDRoot cannot mint a conformant SR UID: %v", ErrMalformedSource, err)
	}
	seed := drIdentitySeedR4(dr)
	ds.SetString(dicom.TagStudyInstanceUID, string(mintUID(cfg.uidRoot, seed, "study")))
	ds.SetString(dicom.TagSeriesInstanceUID, string(mintUID(cfg.uidRoot, seed, "series")))
	ds.SetString(dicom.TagSOPInstanceUID, string(mintUID(cfg.uidRoot, seed, "instance")))
	ds.SetString(dicom.TagSeriesNumber, "1")
	ds.SetString(dicom.TagInstanceNumber, "1")
	return nil
}

// drIdentitySeedR4 returns a stable seed string for UID minting, the R4 twin of
// drIdentitySeed. The report's DICOM UID identifier is preferred so a round trip
// re-derives the same UIDs; absent it, the code and conclusion form the seed. The
// seed is a logical identity, never a patient value.
func drIdentitySeedR4(dr *r4.DiagnosticReport) string {
	for _, id := range dr.Identifier {
		if id.System != nil && *id.System == dicomUIDSystem && id.Value != nil {
			return *id.Value
		}
	}
	var b strings.Builder
	if dr.Code != nil {
		for _, c := range dr.Code.Coding {
			if c.System != nil {
				b.WriteString(*c.System)
			}
			b.WriteByte('|')
			if c.Code != nil {
				b.WriteString(*c.Code)
			}
			b.WriteByte('|')
		}
	}
	if dr.Conclusion != nil {
		b.WriteString(*dr.Conclusion)
	}
	return b.String()
}

// writeSRStatusR4 writes CompletionFlag and VerificationFlag from the report
// status, the R4 twin of writeSRStatus: final maps to COMPLETE+VERIFIED, every
// other status to COMPLETE+UNVERIFIED.
func writeSRStatusR4(ds *dicom.DataSet, dr *r4.DiagnosticReport) {
	ds.SetString(dicom.TagCompletionFlag, "COMPLETE")
	if dr.Status != nil && *dr.Status == r4.DiagnosticReportStatusFinal {
		ds.SetString(dicom.TagVerificationFlag, "VERIFIED")
		return
	}
	ds.SetString(dicom.TagVerificationFlag, "UNVERIFIED")
}

// writeSRContentDateTimeR4 writes ContentDate, ContentTime, and
// TimezoneOffsetFromUTC from the report's effectiveDateTime, the R4 twin of
// writeSRContentDateTime. The lexical conversion (fhirDateTimeToDICOM) and offset
// splitting (splitDICOMOffset) are shared.
func writeSRContentDateTimeR4(ds *dicom.DataSet, dr *r4.DiagnosticReport) {
	if dr.EffectiveDateTime == nil {
		return
	}
	lexical, ok := fhirDateTimeToDICOM(string(*dr.EffectiveDateTime))
	if !ok {
		return
	}
	body, offset := splitDICOMOffset(lexical)
	if len(body) < 8 {
		return
	}
	ds.SetString(dicom.TagContentDate, body[:8])
	if len(body) > 8 {
		ds.SetString(dicom.TagContentTime, body[8:])
	}
	if offset != "" {
		ds.SetString(dicom.TagTimezoneOffsetFromUTC, offset)
	}
}

// writeSRPatientIDR4 writes PatientID from the report subject's logical
// Reference.identifier, the R4 twin of writeSRPatientID. A subject carried as a
// resolvable Reference.reference URL has no DICOM PatientID home and is recorded,
// never fabricated into an identifier.
func writeSRPatientIDR4(ds *dicom.DataSet, dr *r4.DiagnosticReport, report *Report) {
	if dr.Subject == nil {
		return
	}
	if dr.Subject.Identifier != nil && dr.Subject.Identifier.Value != nil && *dr.Subject.Identifier.Value != "" {
		ds.SetString(dicom.TagPatientID, *dr.Subject.Identifier.Value)
		return
	}
	if dr.Subject.Reference != nil {
		report.dropped("DiagnosticReport.subject",
			"the subject is a Reference.reference URL with no logical identifier; a DICOM PatientID (0010,0020) cannot be derived from a resolvable reference")
	}
}
