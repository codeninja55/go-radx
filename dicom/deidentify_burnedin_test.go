package dicom

import (
	"errors"
	"path/filepath"
	"testing"
)

// When BurnedInAnnotation is YES the profile fails closed: it returns
// ErrBurnedInPixelData rather than reporting a complete de-identification while
// leaving identifying text rendered into the pixels (Codex DCM-013).
func TestDeidentifyFailsClosedOnBurnedInPixelData(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagPatientName, "Doe^Jane")
	ds.SetString(TagBurnedInAnnotation, "YES")

	clean, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if !errors.Is(err, ErrBurnedInPixelData) {
		t.Fatalf("Deidentify error = %v, want ErrBurnedInPixelData", err)
	}
	if clean != nil {
		t.Error("Deidentify should return a nil dataset alongside ErrBurnedInPixelData")
	}
	// The source must be untouched even on the fail-closed path.
	if v, _ := ds.GetString(TagPatientName); v != "Doe^Jane" {
		t.Error("Deidentify mutated input on the fail-closed path")
	}
}

// The error must not echo any PHI value (PRD §8.2: name structure, never values).
func TestBurnedInErrorCarriesNoPHI(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagPatientName, "Sensitive^Name")
	ds.SetString(TagBurnedInAnnotation, "YES")

	_, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if err == nil {
		t.Fatal("expected ErrBurnedInPixelData")
	}
	if msg := err.Error(); contains(msg, "Sensitive") || contains(msg, "Name") {
		t.Errorf("burned-in error leaked PHI: %q", msg)
	}
}

// WithAllowBurnedInPixelData accepts the residual risk and lets de-identification
// proceed; the rest of the profile still runs.
func TestDeidentifyAllowBurnedInPixelData(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagPatientName, "Doe^Jane")
	ds.SetString(TagBurnedInAnnotation, "YES")

	clean, err := NewProfile(testGenerator(t), WithAllowBurnedInPixelData()).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify with opt-out: %v", err)
	}
	if v, _ := clean.GetString(TagPatientName); v == "Doe^Jane" {
		t.Error("de-identification did not run under the burned-in opt-out")
	}
	if v, _ := clean.GetString(TagPatientIdentityRemoved); v != "YES" {
		t.Error("metadata not set under the burned-in opt-out")
	}
}

// A deferred BurnedInAnnotation whose source can no longer be read must fail closed:
// the accessor would report a failed deferred load as absent, so checkBurnedIn must
// inspect the element directly and treat an unverifiable flag as burned-in PHI.
func TestDeidentifyBurnedInDeferredUnreadableFailsClosed(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone.dcm") // never created, so Load always fails
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.Set(Element{Tag: TagBurnedInAnnotation, VR: VRCS, Value: &DeferredValue{
		tag: TagBurnedInAnnotation, vr: VRCS, path: gone, offset: 0, length: 4,
	}})

	_, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if !errors.Is(err, ErrBurnedInPixelData) {
		t.Fatalf("Deidentify with an unreadable deferred BurnedInAnnotation = %v, want ErrBurnedInPixelData", err)
	}
}

// A present-but-non-string BurnedInAnnotation (a malformed OB/UN encoding of a CS
// attribute) cannot be confirmed clean and must fail closed, never be read as absent.
func TestDeidentifyBurnedInNonStringFailsClosed(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.Set(Element{Tag: TagBurnedInAnnotation, VR: VROB, Value: NewBytes(VROB, []byte("YES"))})

	_, err := NewProfile(testGenerator(t)).Deidentify(ds)
	if !errors.Is(err, ErrBurnedInPixelData) {
		t.Fatalf("Deidentify with a non-string BurnedInAnnotation = %v, want ErrBurnedInPixelData", err)
	}
}

// BurnedInAnnotation NO or absent must not trigger the fail-closed path.
func TestDeidentifyNoBurnedInProceeds(t *testing.T) {
	for _, val := range []string{"NO", ""} {
		ds := NewDataSet()
		ds.SetString(TagStudyInstanceUID, "1.2.3")
		ds.SetString(TagPatientName, "Doe^Jane")
		if val != "" {
			ds.SetString(TagBurnedInAnnotation, val)
		}
		if _, err := NewProfile(testGenerator(t)).Deidentify(ds); err != nil {
			t.Errorf("BurnedInAnnotation=%q: unexpected error %v", val, err)
		}
	}
}
