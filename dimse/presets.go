package dimse

import "github.com/codeninja55/go-radx/dicom"

// verificationSOPClass is the Verification SOP Class UID (PS3.4 A.4), the abstract syntax
// of a C-ECHO presentation context.
const verificationSOPClass dicom.SOPClassUID = "1.2.840.10008.1.1"

// validatedStorageSOPClasses is the curated, radiology-first Storage SOP Class set go-radx
// declares conformance for as both Storage SCU and Storage SCP. It is the "Supported
// (validated) Storage SOP Classes" table in docs/conformance/dicom.md — 36 classes,
// intentionally narrower than the 120-class pynetdicom selected-Storage floor. Each is
// round-trip-tested and interop-verified; the rest are reachable only via AllStorageContexts.
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
