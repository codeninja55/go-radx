package convert

import (
	"crypto/sha1" // #nosec G505 -- deterministic UID derivation (a stable name hash), not a security primitive
	"fmt"
	"math/big"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// comprehensiveSRSOPClass is the SOP Class UID DiagnosticReportToSR stamps on the
// produced document. A DiagnosticReport with coded and measured Observations is a
// Comprehensive SR, which is the richest of the three SR IODs go-radx round-trips.
const comprehensiveSRSOPClass = "1.2.840.10008.5.1.4.1.1.88.33"

// DiagnosticReportToSR converts a FHIR R5 DiagnosticReport together with its
// Observations back to a DICOM Structured Report document, the inverse of
// SRToDiagnosticReportR5. The report's code becomes the root CONTAINER's Concept Name
// Code Sequence; the conclusion becomes a bare (un-coded) TEXT child; each Observation
// becomes a measurement leaf through observationToContentItem (a string Observation
// becomes a concept-named TEXT leaf, so it re-imports as an Observation, not narrative).
// The document-level status,
// content date/time, and patient identity are written from the report's status,
// effectiveDateTime, and subject identifier.
//
// The Study, Series, and SOP Instance UIDs are minted deterministically: when
// WithUIDRoot supplies an organisation root they are derived under it from the report's
// identity, so the same report mints byte-identical UIDs across runs; absent a root the
// document carries no UIDs and the absence is recorded, because go-radx ships no default
// registered root. A report whose code does not map to a Concept Name Code Sequence is
// rejected fail-closed (ErrMalformedSource): the SR document root requires one.
func DiagnosticReportToSR(dr *r5.DiagnosticReport, observations []*r5.Observation, opts ...Option) (*dicom.DataSet, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if dr == nil {
		return nil, nil, fmt.Errorf("%w: DiagnosticReport is nil", ErrMalformedSource)
	}

	rootConcept := conceptNameFor(dr.Code)
	if rootConcept.IsZero() {
		return nil, nil, fmt.Errorf("%w: DiagnosticReport.code does not map to a Concept Name Code Sequence (0040,A043) for the required SR document root",
			ErrMalformedSource)
	}

	root := &dicom.ContentItem{
		ValueType:   dicom.ValueTypeContainer,
		ConceptName: rootConcept,
	}

	if dr.Conclusion != nil && strings.TrimSpace(*dr.Conclusion) != "" {
		// The conclusion is emitted as a bare (un-coded) TEXT child: the forward path
		// routes a TEXT leaf with no Concept Name Code Sequence to
		// DiagnosticReport.conclusion, while a concept-named TEXT leaf is an Observation.
		// Giving the conclusion a concept name would re-import it as a string Observation
		// and lose the conclusion, so it carries none.
		root.Children = append(root.Children, dicom.ContentItem{
			ValueType:    dicom.ValueTypeText,
			Relationship: dicom.RelationshipContains,
			Text:         *dr.Conclusion,
		})
	}

	for i := range observations {
		item, ok := observationToContentItem(observations[i], report)
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
	if err := mintSRIdentity(ds, cfg, dr, report); err != nil {
		return nil, nil, err
	}
	writeSRStatus(ds, dr)
	writeSRContentDateTime(ds, dr)
	writeSRPatientID(ds, dr, report)

	rep, err := cfg.finalize(report)
	return ds, rep, err
}

// mintSRIdentity mints the Study, Series, and SOP Instance UIDs under the configured
// organisation root, deriving each deterministically from the report's identity so the
// same report yields byte-identical UIDs. Absent a configured root, no UID is minted
// (go-radx ships no default registered root) and the absence is recorded. An over-long
// or otherwise invalid root is rejected fail-closed: silently truncating a conformant
// root could drop the role-specific suffix and mint identical or malformed UIDs, so the
// caller is told its root cannot mint a 64-character UID rather than receiving corrupted
// identity data.
func mintSRIdentity(ds *dicom.DataSet, cfg config, dr *r5.DiagnosticReport, report *Report) error {
	if cfg.uidRoot == "" {
		report.defaulted("SOPInstanceUID (0008,0018)", "",
			"no WithUIDRoot was supplied and go-radx ships no default registered root; the SR document carries no minted UIDs")
		return nil
	}
	// Reuse the dicom UID generator's safe-prefix-length and validity check (root must be
	// a valid UID prefix that leaves room for a suffix within the 64-character field).
	// This rejects an over-long root with the same bound the generator enforces rather
	// than truncating it here.
	if _, err := dicom.NewUIDGenerator(cfg.uidRoot); err != nil {
		return fmt.Errorf("%w: WithUIDRoot cannot mint a conformant SR UID: %v", ErrMalformedSource, err)
	}
	seed := drIdentitySeed(dr)
	ds.SetString(dicom.TagStudyInstanceUID, string(mintUID(cfg.uidRoot, seed, "study")))
	ds.SetString(dicom.TagSeriesInstanceUID, string(mintUID(cfg.uidRoot, seed, "series")))
	ds.SetString(dicom.TagSOPInstanceUID, string(mintUID(cfg.uidRoot, seed, "instance")))
	ds.SetString(dicom.TagSeriesNumber, "1")
	ds.SetString(dicom.TagInstanceNumber, "1")
	return nil
}

// drIdentitySeed returns a stable seed string for UID minting. The report's DICOM UID
// identifier (a urn:oid: value carried by the SR-sourced forward path) is preferred so
// a round trip re-derives the same UIDs; absent it, the code and conclusion form the
// seed. The seed is a logical identity, never a patient value.
func drIdentitySeed(dr *r5.DiagnosticReport) string {
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

// mintUID derives a conformant UID under root by hashing the seed and a role label
// (study/series/instance), then rendering the hash as a decimal suffix. The derivation
// uses no randomness and no wall clock, so the same (root, seed) always mints the same
// triple; the role label keeps the three UIDs of one document distinct.
//
// The root is validated for length by the caller (mintSRIdentity), so after the
// separating dot the 64-character field always leaves room for a suffix. When the full
// decimal suffix would overflow the field, only the suffix is trimmed — never the root
// or the dot — so the UID can never end in a dot and the role-specific leading digits
// that keep the three UIDs distinct are preserved.
func mintUID(root dicom.UID, seed, role string) dicom.UID {
	h := sha1.New() // #nosec G401 -- deterministic UID derivation (a stable name hash), not a security primitive
	h.Write([]byte(seed))
	h.Write([]byte{0})
	h.Write([]byte(role))
	sum := h.Sum(nil)

	prefix := string(root)
	if !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	suffix := new(big.Int).SetBytes(sum).String()
	if budget := maxUIDLen - len(prefix); len(suffix) > budget {
		suffix = suffix[:budget]
	}
	return dicom.UID(prefix + suffix)
}

// maxUIDLen is the PS3.5 UID character cap, mirrored here so a minted UID never exceeds
// the 64-character field.
const maxUIDLen = 64

// writeSRStatus writes CompletionFlag (0040,A491) and VerificationFlag (0040,A493) from
// the report status, the inverse of srStatus: final maps to COMPLETE+VERIFIED, every
// other status to COMPLETE+UNVERIFIED, since a reported document is structurally complete
// even when not yet verified.
func writeSRStatus(ds *dicom.DataSet, dr *r5.DiagnosticReport) {
	ds.SetString(dicom.TagCompletionFlag, "COMPLETE")
	if dr.Status != nil && *dr.Status == r5.DiagnosticReportStatusFinal {
		ds.SetString(dicom.TagVerificationFlag, "VERIFIED")
		return
	}
	ds.SetString(dicom.TagVerificationFlag, "UNVERIFIED")
}

// writeSRContentDateTime writes ContentDate (0008,0023), ContentTime (0008,0033), and
// TimezoneOffsetFromUTC (0008,0201) from the report's effectiveDateTime, the inverse of
// the combineDateTime forward mapping. A date-only effective value writes only the date;
// a value carrying a time splits into date, time, and the document-level offset so the
// forward path re-derives the same FHIR dateTime.
func writeSRContentDateTime(ds *dicom.DataSet, dr *r5.DiagnosticReport) {
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

// splitDICOMOffset peels a trailing "&ZZXX" (+/-HHMM) UTC offset off a DICOM datetime
// lexical, returning the body and the offset ("" when absent). It scans from the right
// so a leading-sign in the body is never mistaken for the offset.
func splitDICOMOffset(lexical string) (body, offset string) {
	for i := len(lexical) - 1; i > 0; i-- {
		if lexical[i] == '+' || lexical[i] == '-' {
			return lexical[:i], lexical[i:]
		}
	}
	return lexical, ""
}

// writeSRPatientID writes PatientID (0010,0020) from the report subject's logical
// Reference.identifier, the inverse of dicomPatientSubjectR5. A subject carried as a
// resolvable Reference.reference URL has no DICOM PatientID home and is recorded, never
// fabricated into an identifier.
func writeSRPatientID(ds *dicom.DataSet, dr *r5.DiagnosticReport, report *Report) {
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
