package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// dicomPatientSubjectR4 resolves the Patient subject Reference for a DICOM-sourced
// R4 converter, the R4 twin of dicomPatientSubjectR5. WithSubjectR4 takes
// precedence. Otherwise the DICOM PatientID (0010,0020) and IssuerOfPatientID
// (0010,0021) are carried as a logical Reference.identifier — never a fabricated
// Reference.reference URL (the identity rule). When the dataset carries no
// PatientID, subject is left unset (nil) and a Defaulted entry records the absence.
func dicomPatientSubjectR4(cfg config, ds *dicom.DataSet, report *Report, targetPath string) *r4.Reference {
	if cfg.subjectR4 != nil {
		ref := *cfg.subjectR4
		return &ref
	}

	patientID, ok := ds.GetString(dicom.TagPatientID)
	if !ok || patientID == "" {
		report.defaulted(targetPath, "",
			"dataset carries no PatientID (0010,0020) and no WithSubjectR4 was supplied; subject left unset")
		return nil
	}

	value := patientID
	id := r4.Identifier{Value: &value}
	if issuer, has := ds.GetString(dicom.TagIssuerOfPatientID); has && issuer != "" {
		system := issuer
		id.System = &system
	}
	refType := patientReferenceType
	return &r4.Reference{Type: &refType, Identifier: &id}
}
