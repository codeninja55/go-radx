package dimse

import "github.com/codeninja55/go-radx/dicom"

// verificationSOPClass is the Verification SOP Class UID (PS3.4 A.4), the abstract syntax
// of a C-ECHO presentation context.
const verificationSOPClass dicom.SOPClassUID = "1.2.840.10008.1.1"

// Query/Retrieve, Worklist, MPPS, and Storage Commitment SOP Class UIDs, verified against the
// DICOM registry (PS3.6 Annex A) and pynetdicom's sop_class exports. These are the abstract
// syntaxes the M3 presentation-context presets propose.
const (
	patientRootFindSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.1.1" // Patient Root Q/R — FIND
	patientRootMoveSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.1.2" // Patient Root Q/R — MOVE
	patientRootGetSOPClass  dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.1.3" // Patient Root Q/R — GET
	studyRootFindSOPClass   dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.2.1" // Study Root Q/R — FIND
	studyRootMoveSOPClass   dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.2.2" // Study Root Q/R — MOVE
	studyRootGetSOPClass    dicom.SOPClassUID = "1.2.840.10008.5.1.4.1.2.2.3" // Study Root Q/R — GET

	modalityWorklistSOPClass           dicom.SOPClassUID = "1.2.840.10008.5.1.4.31"  // Modality Worklist Information Model — FIND
	modalityPerformedStepSOPClass      dicom.SOPClassUID = "1.2.840.10008.3.1.2.3.3" // Modality Performed Procedure Step (MPPS)
	storageCommitmentPushModelSOPClass dicom.SOPClassUID = "1.2.840.10008.1.20.1"    // Storage Commitment Push Model
)

// queryRetrieveSOPClasses is the Patient Root and Study Root Query/Retrieve Information Model set
// (FIND, MOVE, GET each), the abstract syntaxes QueryRetrieveContexts proposes. It is the go-radx
// conformance subset of pynetdicom's 13-class QueryRetrievePresentationContexts (the
// PatientStudyOnly, CompositeInstance, and RepositoryQuery models are out of v1 scope).
var queryRetrieveSOPClasses = []dicom.SOPClassUID{
	patientRootFindSOPClass,
	patientRootMoveSOPClass,
	patientRootGetSOPClass,
	studyRootFindSOPClass,
	studyRootMoveSOPClass,
	studyRootGetSOPClass,
}

// validatedStorageSOPClasses is the curated, radiology-first Storage SOP Class set go-radx
// declares conformance for as both Storage SCU and Storage SCP. It is the "Supported
// (validated) Storage SOP Classes" table in docs/conformance/dicom.md — 36 classes,
// intentionally narrower than the 120-class pynetdicom selected-Storage floor. Each is
// round-trip-tested and interop-verified. A transport-only forwarding preset over the full
// registered Storage set is documented as NOT YET SHIPPED in docs/conformance/dicom.md.
var validatedStorageSOPClasses = []dicom.SOPClassUID{
	"1.2.840.10008.5.1.4.1.1.1",      // Computed Radiography Image Storage
	"1.2.840.10008.5.1.4.1.1.1.1",    // Digital X-Ray Image Storage — For Presentation
	"1.2.840.10008.5.1.4.1.1.1.1.1",  // Digital X-Ray Image Storage — For Processing
	"1.2.840.10008.5.1.4.1.1.1.2",    // Digital Mammography X-Ray Image Storage — For Presentation
	"1.2.840.10008.5.1.4.1.1.1.2.1",  // Digital Mammography X-Ray Image Storage — For Processing
	"1.2.840.10008.5.1.4.1.1.13.1.3", // Breast Tomosynthesis Image Storage
	"1.2.840.10008.5.1.4.1.1.2",      // CT Image Storage
	"1.2.840.10008.5.1.4.1.1.2.1",    // Enhanced CT Image Storage
	"1.2.840.10008.5.1.4.1.1.2.2",    // Legacy Converted Enhanced CT Image Storage
	"1.2.840.10008.5.1.4.1.1.4",      // MR Image Storage
	"1.2.840.10008.5.1.4.1.1.4.1",    // Enhanced MR Image Storage
	"1.2.840.10008.5.1.4.1.1.4.3",    // Enhanced MR Color Image Storage
	"1.2.840.10008.5.1.4.1.1.4.4",    // Legacy Converted Enhanced MR Image Storage
	"1.2.840.10008.5.1.4.1.1.6.1",    // Ultrasound Image Storage
	"1.2.840.10008.5.1.4.1.1.3.1",    // Ultrasound Multi-frame Image Storage
	"1.2.840.10008.5.1.4.1.1.12.1",   // X-Ray Angiographic Image Storage
	"1.2.840.10008.5.1.4.1.1.12.1.1", // Enhanced XA Image Storage
	"1.2.840.10008.5.1.4.1.1.12.2",   // X-Ray Radiofluoroscopic Image Storage
	"1.2.840.10008.5.1.4.1.1.12.2.1", // Enhanced XRF Image Storage
	"1.2.840.10008.5.1.4.1.1.20",     // Nuclear Medicine Image Storage
	"1.2.840.10008.5.1.4.1.1.128",    // Positron Emission Tomography Image Storage
	"1.2.840.10008.5.1.4.1.1.130",    // Enhanced PET Image Storage
	"1.2.840.10008.5.1.4.1.1.7",      // Secondary Capture Image Storage
	"1.2.840.10008.5.1.4.1.1.7.1",    // Multi-frame Single Bit Secondary Capture Image Storage
	"1.2.840.10008.5.1.4.1.1.7.2",    // Multi-frame Grayscale Byte Secondary Capture Image Storage
	"1.2.840.10008.5.1.4.1.1.7.3",    // Multi-frame Grayscale Word Secondary Capture Image Storage
	"1.2.840.10008.5.1.4.1.1.7.4",    // Multi-frame True Color Secondary Capture Image Storage
	"1.2.840.10008.5.1.4.1.1.66.4",   // Segmentation Storage
	"1.2.840.10008.5.1.4.1.1.30",     // Parametric Map Storage
	"1.2.840.10008.5.1.4.1.1.11.1",   // Grayscale Softcopy Presentation State Storage
	"1.2.840.10008.5.1.4.1.1.11.2",   // Color Softcopy Presentation State Storage
	"1.2.840.10008.5.1.4.1.1.88.11",  // Basic Text SR Storage
	"1.2.840.10008.5.1.4.1.1.88.22",  // Enhanced SR Storage
	"1.2.840.10008.5.1.4.1.1.88.33",  // Comprehensive SR Storage
	"1.2.840.10008.5.1.4.1.1.88.59",  // Key Object Selection Document Storage
	"1.2.840.10008.5.1.4.1.1.104.1",  // Encapsulated PDF Storage
}

// extendedQueryRetrieveSOPClasses is the additional Query/Retrieve Information Model set beyond the
// Patient Root / Study Root core: the Patient/Study Only model (FIND/MOVE/GET), the Composite
// Instance Root Retrieve model (MOVE/GET), and the Composite Instance Retrieve Without Bulk Data
// model (GET). They reuse the same C-FIND/C-GET/C-MOVE machinery; only their level handling differs
// (PS3.4 C.6.3, C.6.5, C.6.6).
var extendedQueryRetrieveSOPClasses = []dicom.SOPClassUID{
	patientStudyOnlyFindSOPClass,
	patientStudyOnlyMoveSOPClass,
	patientStudyOnlyGetSOPClass,
	compositeInstanceRootMoveSOPClass,
	compositeInstanceRootGetSOPClass,
	compositeInstanceRetrieveWithoutBulkGetSOPClass,
}

// contextsFor builds presentation contexts for the given abstract syntaxes, assigning the
// odd IDs 1, 3, 5, … (PS3.8 9.3.2.2) and proposing DefaultTransferSyntaxes for each. It
// returns a fresh slice so callers may mutate it without sharing state.
func contextsFor(abstracts []dicom.SOPClassUID) []PresentationContext {
	contexts := make([]PresentationContext, 0, len(abstracts))
	id := uint8(1)
	for _, as := range abstracts {
		contexts = append(contexts, NewPresentationContext(id, as))
		id += 2
	}
	return contexts
}

// VerificationContexts returns the single presentation context for the Verification SOP
// Class (C-ECHO), proposing the default transfer syntaxes. A fresh slice is returned each
// call (dimse.md "Presets").
func VerificationContexts() []PresentationContext {
	return contextsFor([]dicom.SOPClassUID{verificationSOPClass})
}

// StorageContexts returns the 36-class validated radiology Storage set, each context
// proposing the default transfer syntaxes (docs/conformance/dicom.md preset summary). It
// is intentionally narrower than the pynetdicom selected-Storage floor and stays well
// under the 128-context A-ASSOCIATE-RQ limit. A fresh slice is returned each call.
func StorageContexts() []PresentationContext {
	return contextsFor(validatedStorageSOPClasses)
}

// QueryRetrieveContexts returns the six Patient Root and Study Root Query/Retrieve Information
// Model contexts (FIND, MOVE, GET each), the abstract syntaxes a C-FIND/C-MOVE/C-GET SCU proposes,
// each proposing the default transfer syntaxes (dimse.md "Presets"). A fresh slice is returned
// each call.
func QueryRetrieveContexts() []PresentationContext {
	return contextsFor(queryRetrieveSOPClasses)
}

// QueryRetrieveWithStorageContexts returns the Query/Retrieve Information Model contexts together
// with the validated radiology Storage contexts, assigned contiguous non-colliding presentation
// context IDs in a single pass. It is the preset a same-association C-GET SCU proposes: the GET
// model context carries the C-GET-RQ, while the Storage contexts give the SCP's sub-operation
// C-STOREs (which arrive back on the SAME association) an accepted context to use. Combining the two
// preset slices directly would collide their IDs (each restarts numbering at 1), so this builds the
// union with one ID sequence. The C-GET SCU pairs it with WithRoleSelection proposing the Storage
// SCP role for the SOP Classes it expects to receive. A fresh slice is returned each call.
func QueryRetrieveWithStorageContexts() []PresentationContext {
	combined := make([]dicom.SOPClassUID, 0, len(queryRetrieveSOPClasses)+len(validatedStorageSOPClasses))
	combined = append(combined, queryRetrieveSOPClasses...)
	combined = append(combined, validatedStorageSOPClasses...)
	return contextsFor(combined)
}

// ExtendedQueryRetrieveContexts returns the additional Query/Retrieve Information Model contexts: the
// Patient/Study Only model (FIND/MOVE/GET), the Composite Instance Root Retrieve model (MOVE/GET),
// and the Composite Instance Retrieve Without Bulk Data model (GET). These complement the Patient
// Root / Study Root core returned by QueryRetrieveContexts and reuse the same C-FIND/C-GET/C-MOVE
// machinery (PS3.4 C.6.3, C.6.5, C.6.6). A fresh slice is returned each call.
func ExtendedQueryRetrieveContexts() []PresentationContext {
	return contextsFor(extendedQueryRetrieveSOPClasses)
}

// BasicWorklistContexts returns the single Modality Worklist Information Model — FIND context, the
// abstract syntax an MWL C-FIND SCU proposes, proposing the default transfer syntaxes (dimse.md
// "Presets"). A fresh slice is returned each call.
func BasicWorklistContexts() []PresentationContext {
	return contextsFor([]dicom.SOPClassUID{modalityWorklistSOPClass})
}

// ModalityPerformedContexts returns the single Modality Performed Procedure Step (MPPS) context,
// the abstract syntax an MPPS SCU proposes, proposing the default transfer syntaxes (dimse.md
// "Presets"). A fresh slice is returned each call.
func ModalityPerformedContexts() []PresentationContext {
	return contextsFor([]dicom.SOPClassUID{modalityPerformedStepSOPClass})
}

// StorageCommitmentContexts returns the single Storage Commitment Push Model context, the abstract
// syntax a Storage Commitment SCU proposes, proposing the default transfer syntaxes (dimse.md
// "Presets"). A fresh slice is returned each call.
func StorageCommitmentContexts() []PresentationContext {
	return contextsFor([]dicom.SOPClassUID{storageCommitmentPushModelSOPClass})
}
