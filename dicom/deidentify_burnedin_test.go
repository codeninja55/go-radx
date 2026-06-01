package dicom

import (
	"errors"
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
