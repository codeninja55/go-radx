package convert

import (
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// patientReferenceType is the FHIR resource type a subject Reference points at.
const patientReferenceType = "Patient"

// patientSubjectR5 resolves the Patient subject Reference for an HL7-sourced
// converter. When WithSubjectR5 supplied an explicit reference, that is used
// verbatim. Otherwise the patient's HL7 identity (PID-3 CX) is carried as a
// logical Reference.identifier — never a fabricated Reference.reference URL (the
// identity rule). When the source carries no patient identity at all, subject is
// left unset and a Defaulted entry records the absence.
//
// targetPath names the FHIR element (e.g. "ServiceRequest.subject") for the
// Defaulted entry, so the diagnostic is element-named per the report model.
func patientSubjectR5(cfg config, msg *hl7v2.Message, report *Report, targetPath string) *r5.Reference {
	if cfg.subjectR5 != nil {
		ref := *cfg.subjectR5
		return &ref
	}

	if msg != nil {
		if pid, ok := msg.PID(); ok && pid.PatientID.ID != "" {
			id := cxToIdentifierR5(pid.PatientID)
			refType := patientReferenceType
			return &r5.Reference{
				Type:       &refType,
				Identifier: &id,
			}
		}
	}

	report.defaulted(targetPath, "",
		"source carries no patient identity and no WithSubjectR5 was supplied; subject left unset")
	return nil
}

// cxToIdentifierR5 maps an HL7 v2 CX (ID + assigning authority) to a FHIR
// Identifier. The system is derived from the assigning authority (HD), the value
// from the ID component. It is never a Reference.reference URL (the identity
// rule). The Coding/system is carried verbatim; go-radx does not translate
// assigning-authority namespaces.
func cxToIdentifierR5(cx hl7v2.CX) r5.Identifier {
	id := r5.Identifier{}
	value := cx.ID
	id.Value = &value
	if system := assigningAuthoritySystem(cx.AssigningAuthority); system != "" {
		id.System = &system
	}
	return id
}

// assigningAuthoritySystem derives an Identifier.system from an HL7 HD. A
// universal ID (HD-2) is preferred when present; otherwise the namespace ID
// (HD-1) is used as a local system name. An empty HD yields an empty system,
// leaving the Identifier system-less rather than inventing one.
func assigningAuthoritySystem(hd hl7v2.HD) string {
	if hd.UniversalID != "" {
		return hd.UniversalID
	}
	return hd.NamespaceID
}
