package convert

import (
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// patientSubjectR4 resolves the Patient subject Reference for an HL7-sourced R4
// converter, the R4 twin of patientSubjectR5. When WithSubjectR4 supplied an
// explicit reference, that is used verbatim. Otherwise the patient's HL7 identity
// (PID-3 CX) is carried as a logical Reference.identifier — never a fabricated
// Reference.reference URL (the identity rule). When the source carries no patient
// identity at all, subject is left unset and a Defaulted entry records the absence.
func patientSubjectR4(cfg config, msg *hl7v2.Message, report *Report, targetPath string) *r4.Reference {
	if cfg.subjectR4 != nil {
		ref := *cfg.subjectR4
		return &ref
	}

	if msg != nil {
		if pid, ok := msg.PID(); ok && pid.PatientID.ID != "" {
			id := cxToIdentifierR4(pid.PatientID)
			refType := patientReferenceType
			return &r4.Reference{
				Type:       &refType,
				Identifier: &id,
			}
		}
	}

	report.defaulted(targetPath, "",
		"source carries no patient identity and no WithSubjectR4 was supplied; subject left unset")
	return nil
}

// cxToIdentifierR4 maps an HL7 v2 CX (ID + assigning authority) to a FHIR R4
// Identifier, the R4 twin of cxToIdentifierR5. The system is derived from the
// assigning authority (HD), the value from the ID component. It is never a
// Reference.reference URL (the identity rule). The Coding/system is carried
// verbatim; go-radx does not translate assigning-authority namespaces.
func cxToIdentifierR4(cx hl7v2.CX) r4.Identifier {
	id := r4.Identifier{}
	value := cx.ID
	id.Value = &value
	if system := assigningAuthoritySystem(cx.AssigningAuthority); system != "" {
		id.System = &system
	}
	return id
}
