package dimse

import "github.com/codeninja55/go-radx/dicom"

// Additional Query/Retrieve Information Model SOP Class UIDs, verified against the DICOM registry
// (PS3.6 Annex A) and pynetdicom's sop_class exports (the `_QR_CLASSES` table). These reuse the
// existing C-FIND/C-GET/C-MOVE machinery; only the SOP-class registration and per-model level
// handling are new.
const (
	// patientStudyOnlyFindSOPClass is the Patient/Study Only Q/R Information Model — FIND
	// (PS3.4 C.6.3, retired in the current edition but still admitted by pynetdicom). It is a
	// two-level model: PATIENT and STUDY only (no SERIES/IMAGE).
	patientStudyOnlyFindSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.3.1"
	patientStudyOnlyMoveSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.3.2"
	patientStudyOnlyGetSOPClass  dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.3.3"

	// compositeInstanceRootMoveSOPClass and compositeInstanceRootGetSOPClass are the Composite
	// Instance Root Retrieve Information Model — MOVE and GET (PS3.4 C.6.5). This model has no FIND
	// service. It is a two-level retrieve rooted at the SOP Instance UID: the IMAGE level and the
	// FRAME level (single-frame extraction from a multi-frame instance).
	compositeInstanceRootMoveSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.4.2"
	compositeInstanceRootGetSOPClass  dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.4.3"

	// compositeInstanceRetrieveWithoutBulkGetSOPClass is the Composite Instance Retrieve Without Bulk
	// Data Information Model — GET (PS3.4 C.6.6). It is GET-only and IMAGE-level only: it retrieves an
	// instance with its bulk-data attributes (e.g. Pixel Data) omitted.
	compositeInstanceRetrieveWithoutBulkGetSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.5.3"
)

// Exported Query/Retrieve Information Model SOP Class UID aliases, so a caller may name a model in
// WithQueryModel (and an SCP recognises it). They follow the ModalityWorklistInformationModelFind
// precedent: the unexported constants are the wire values, these are the public names.
const (
	// PatientStudyOnlyQueryRetrieveInformationModelFind is the Patient/Study Only Q/R Information
	// Model — FIND SOP Class UID (PS3.4 C.6.3). A two-level (PATIENT/STUDY) C-FIND model.
	PatientStudyOnlyQueryRetrieveInformationModelFind = patientStudyOnlyFindSOPClass
	// PatientStudyOnlyQueryRetrieveInformationModelMove is the Patient/Study Only Q/R Information
	// Model — MOVE SOP Class UID (PS3.4 C.6.3).
	PatientStudyOnlyQueryRetrieveInformationModelMove = patientStudyOnlyMoveSOPClass
	// PatientStudyOnlyQueryRetrieveInformationModelGet is the Patient/Study Only Q/R Information
	// Model — GET SOP Class UID (PS3.4 C.6.3).
	PatientStudyOnlyQueryRetrieveInformationModelGet = patientStudyOnlyGetSOPClass

	// CompositeInstanceRootRetrieveInformationModelMove is the Composite Instance Root Retrieve
	// Information Model — MOVE SOP Class UID (PS3.4 C.6.5). An IMAGE/FRAME-level retrieve model.
	CompositeInstanceRootRetrieveInformationModelMove = compositeInstanceRootMoveSOPClass
	// CompositeInstanceRootRetrieveInformationModelGet is the Composite Instance Root Retrieve
	// Information Model — GET SOP Class UID (PS3.4 C.6.5). An IMAGE/FRAME-level retrieve model.
	CompositeInstanceRootRetrieveInformationModelGet = compositeInstanceRootGetSOPClass

	// CompositeInstanceRetrieveWithoutBulkDataGet is the Composite Instance Retrieve Without Bulk
	// Data Information Model — GET SOP Class UID (PS3.4 C.6.6). An IMAGE-level, GET-only model.
	CompositeInstanceRetrieveWithoutBulkDataGet = compositeInstanceRetrieveWithoutBulkGetSOPClass
)

// modelLevels maps each levelled Query/Retrieve Information Model to the set of Query/Retrieve Levels
// (0008,0052) it admits, per PS3.4 Annex C. A model not in this map (e.g. Modality Worklist) carries
// no level and is exempt from the per-model level check. The check is what makes the additional
// models meaningful: a Composite Instance Root retrieve at the PATIENT level, or a Patient/Study Only
// query at the IMAGE level, is a malformed request a compliant peer rejects, so go-radx fails closed
// before any wire I/O rather than emitting a model/level pair no SCP will honour.
var modelLevels = map[dicom.SOPClassUID]map[QueryLevel]struct{}{
	// Patient Root: PATIENT/STUDY/SERIES/IMAGE (PS3.4 C.6.1).
	patientRootFindSOPClass: {QueryLevelPatient: {}, QueryLevelStudy: {}, QueryLevelSeries: {}, QueryLevelImage: {}},
	patientRootMoveSOPClass: {QueryLevelPatient: {}, QueryLevelStudy: {}, QueryLevelSeries: {}, QueryLevelImage: {}},
	patientRootGetSOPClass:  {QueryLevelPatient: {}, QueryLevelStudy: {}, QueryLevelSeries: {}, QueryLevelImage: {}},
	// Study Root: STUDY/SERIES/IMAGE — no PATIENT level (PS3.4 C.6.2).
	studyRootFindSOPClass: {QueryLevelStudy: {}, QueryLevelSeries: {}, QueryLevelImage: {}},
	studyRootMoveSOPClass: {QueryLevelStudy: {}, QueryLevelSeries: {}, QueryLevelImage: {}},
	studyRootGetSOPClass:  {QueryLevelStudy: {}, QueryLevelSeries: {}, QueryLevelImage: {}},
	// Patient/Study Only: PATIENT/STUDY only (PS3.4 C.6.3).
	patientStudyOnlyFindSOPClass: {QueryLevelPatient: {}, QueryLevelStudy: {}},
	patientStudyOnlyMoveSOPClass: {QueryLevelPatient: {}, QueryLevelStudy: {}},
	patientStudyOnlyGetSOPClass:  {QueryLevelPatient: {}, QueryLevelStudy: {}},
	// Composite Instance Root Retrieve: IMAGE and FRAME (PS3.4 C.6.5).
	compositeInstanceRootMoveSOPClass: {QueryLevelImage: {}, QueryLevelFrame: {}},
	compositeInstanceRootGetSOPClass:  {QueryLevelImage: {}, QueryLevelFrame: {}},
	// Composite Instance Retrieve Without Bulk Data: IMAGE only (PS3.4 C.6.6).
	compositeInstanceRetrieveWithoutBulkGetSOPClass: {QueryLevelImage: {}},
}

// validateModelLevel reports whether level is admissible for the given Query/Retrieve Information
// Model. A model with no level table (the Modality Worklist model) is exempt; for a levelled model an
// out-of-range level is a *ValidationError naming the constraint so the caller fails closed before
// any wire I/O (PRD §9.2; PS3.4 Annex C).
func validateModelLevel(model dicom.SOPClassUID, level QueryLevel) error {
	levels, ok := modelLevels[model]
	if !ok {
		return nil // a flat (level-less) model such as Modality Worklist
	}
	if _, ok := levels[level]; !ok {
		return &ValidationError{Detail: "Query/Retrieve Level " + level.String() +
			" is not valid for the information model " + string(model)}
	}
	return nil
}
