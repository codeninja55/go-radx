package dimse

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestVerificationContextsCount pins the Verification preset at exactly one context for
// the Verification SOP Class, proposing the default transfer syntaxes (dimse.md
// "Presentation contexts and presets"; docs/conformance/dicom.md preset summary line 238).
func TestVerificationContextsCount(t *testing.T) {
	got := VerificationContexts()
	if len(got) != 1 {
		t.Fatalf("VerificationContexts() returned %d contexts, want 1", len(got))
	}
	pc := got[0]
	if pc.AbstractSyntax != dicom.SOPClassUID("1.2.840.10008.1.1") {
		t.Errorf("Verification abstract syntax = %q, want 1.2.840.10008.1.1", pc.AbstractSyntax)
	}
	if pc.ID%2 == 0 {
		t.Errorf("context ID %d is even; presentation context IDs must be odd (PS3.8 9.3.2.2)", pc.ID)
	}
	if len(pc.TransferSyntaxes) != len(DefaultTransferSyntaxes) {
		t.Errorf("Verification context proposes %d transfer syntaxes, want the default %d",
			len(pc.TransferSyntaxes), len(DefaultTransferSyntaxes))
	}
}

// storageSOPClasses is the validated radiology Storage set from
// docs/conformance/dicom.md "Supported (validated) Storage SOP Classes" (lines 129-166),
// 36 SOP Classes; the preset summary (line 239) pins the count at 36.
var storageSOPClasses = []dicom.SOPClassUID{
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

// TestStorageContextsCount pins the validated radiology Storage set at exactly 36 contexts
// (docs/conformance/dicom.md preset summary line 239) and verifies the exact SOP Class set.
func TestStorageContextsCount(t *testing.T) {
	got := StorageContexts()
	if len(got) != 36 {
		t.Fatalf("StorageContexts() returned %d contexts, want 36 (docs/conformance/dicom.md line 239)", len(got))
	}
	if len(storageSOPClasses) != 36 {
		t.Fatalf("test fixture lists %d SOP Classes, want 36", len(storageSOPClasses))
	}
	for i, want := range storageSOPClasses {
		if got[i].AbstractSyntax != want {
			t.Errorf("StorageContexts()[%d].AbstractSyntax = %q, want %q", i, got[i].AbstractSyntax, want)
		}
		if len(got[i].TransferSyntaxes) != len(DefaultTransferSyntaxes) {
			t.Errorf("StorageContexts()[%d] proposes %d transfer syntaxes, want the default %d",
				i, len(got[i].TransferSyntaxes), len(DefaultTransferSyntaxes))
		}
	}
}

// TestStorageContextsOddUniqueIDs verifies every context ID is odd (PS3.8 9.3.2.2) and
// unique, so an A-ASSOCIATE-RQ proposing the set has no colliding or even IDs.
func TestStorageContextsOddUniqueIDs(t *testing.T) {
	seen := map[uint8]bool{}
	for _, pc := range StorageContexts() {
		if pc.ID%2 == 0 {
			t.Errorf("context ID %d is even; IDs must be odd (PS3.8 9.3.2.2)", pc.ID)
		}
		if seen[pc.ID] {
			t.Errorf("duplicate context ID %d", pc.ID)
		}
		seen[pc.ID] = true
	}
}

// TestPresetsReturnFreshSlices verifies each preset returns a fresh slice so a caller may
// mutate it without affecting later calls (dimse.md "Presets").
func TestPresetsReturnFreshSlices(t *testing.T) {
	a := StorageContexts()
	b := StorageContexts()
	if len(a) == 0 {
		t.Fatal("StorageContexts() returned an empty slice")
	}
	a[0].ID = 250
	if b[0].ID == 250 {
		t.Error("StorageContexts() returns a shared slice; callers can corrupt each other")
	}
}

// TestNewPresetContextCounts pins the M3 query/retrieve, worklist, MPPS, and storage-commitment
// presets at their conformance counts (dimse.md "Presets": QueryRetrieve 6, BasicWorklist 1,
// ModalityPerformed 1, StorageCommitment 1) and checks the IDs are odd (PS3.8 9.3.2.2) and each
// proposes the default transfer syntaxes.
func TestNewPresetContextCounts(t *testing.T) {
	cases := []struct {
		name string
		got  []PresentationContext
		want int
	}{
		{"QueryRetrieveContexts", QueryRetrieveContexts(), 6},
		{"BasicWorklistContexts", BasicWorklistContexts(), 1},
		{"ModalityPerformedContexts", ModalityPerformedContexts(), 1},
		{"StorageCommitmentContexts", StorageCommitmentContexts(), 1},
	}
	for _, c := range cases {
		if len(c.got) != c.want {
			t.Errorf("%s returned %d contexts, want %d", c.name, len(c.got), c.want)
		}
		for _, pc := range c.got {
			if pc.ID%2 == 0 {
				t.Errorf("%s context ID %d is even; IDs must be odd (PS3.8 9.3.2.2)", c.name, pc.ID)
			}
			if len(pc.TransferSyntaxes) != len(DefaultTransferSyntaxes) {
				t.Errorf("%s context %d proposes %d transfer syntaxes, want the default %d",
					c.name, pc.ID, len(pc.TransferSyntaxes), len(DefaultTransferSyntaxes))
			}
		}
	}

	// Fresh slice each call, matching the existing presets: mutating one return must not corrupt
	// the next.
	a := QueryRetrieveContexts()
	b := QueryRetrieveContexts()
	a[0].ID = 250
	if b[0].ID == 250 {
		t.Error("QueryRetrieveContexts() returns a shared slice; callers can corrupt each other")
	}
}

// TestNewPresetAbstractSyntaxes verifies the exact SOP Class UIDs each M3 preset proposes,
// against the DICOM registry and pynetdicom's presentation-context exports.
func TestNewPresetAbstractSyntaxes(t *testing.T) {
	qr := QueryRetrieveContexts()
	wantQR := []dicom.SOPClassUID{
		"1.2.840.10008.5.1.4.1.2.1.1", // Patient Root Q/R — FIND
		"1.2.840.10008.5.1.4.1.2.1.2", // Patient Root Q/R — MOVE
		"1.2.840.10008.5.1.4.1.2.1.3", // Patient Root Q/R — GET
		"1.2.840.10008.5.1.4.1.2.2.1", // Study Root Q/R — FIND
		"1.2.840.10008.5.1.4.1.2.2.2", // Study Root Q/R — MOVE
		"1.2.840.10008.5.1.4.1.2.2.3", // Study Root Q/R — GET
	}
	for i, want := range wantQR {
		if qr[i].AbstractSyntax != want {
			t.Errorf("QueryRetrieveContexts()[%d].AbstractSyntax = %q, want %q", i, qr[i].AbstractSyntax, want)
		}
	}
	if got := BasicWorklistContexts()[0].AbstractSyntax; got != dicom.SOPClassUID("1.2.840.10008.5.1.4.31") {
		t.Errorf("BasicWorklistContexts abstract syntax = %q, want Modality Worklist 1.2.840.10008.5.1.4.31", got)
	}
	if got := ModalityPerformedContexts()[0].AbstractSyntax; got != dicom.SOPClassUID("1.2.840.10008.3.1.2.3.3") {
		t.Errorf("ModalityPerformedContexts abstract syntax = %q, want MPPS 1.2.840.10008.3.1.2.3.3", got)
	}
	if got := StorageCommitmentContexts()[0].AbstractSyntax; got != dicom.SOPClassUID("1.2.840.10008.1.20.1") {
		t.Errorf("StorageCommitmentContexts abstract syntax = %q, want Storage Commitment Push Model 1.2.840.10008.1.20.1", got)
	}
}
