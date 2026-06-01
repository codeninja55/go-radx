package dicom

import (
	"errors"
	"testing"
)

// testGenerator returns a deterministic-root UID generator for tests. The 2.25.
// random root is fail-closed and needs no registration.
func testGenerator(t *testing.T) *UIDGenerator {
	t.Helper()
	return NewRandomUIDGenerator()
}

func TestDeidentifySetsRequiredMetadata(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagPatientName, "Doe^Jane")

	prof := NewProfile(testGenerator(t))
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	if v, _ := clean.GetString(TagPatientIdentityRemoved); v != "YES" {
		t.Errorf("PatientIdentityRemoved = %q, want YES", v)
	}
	if v, ok := clean.GetString(TagDeidentificationMethod); !ok || v == "" {
		t.Errorf("DeidentificationMethod = %q,%v, want a non-empty description", v, ok)
	}
	// The Basic Profile code 113100 must appear in the method code sequence.
	seq, ok := clean.GetSequence(TagDeidentificationMethodCodeSequence)
	if !ok {
		t.Fatal("DeidentificationMethodCodeSequence absent")
	}
	foundBasic := false
	for it := range seq.Items() {
		if cv, _ := it.DataSet.GetString(TagCodeValue); cv == "113100" {
			foundBasic = true
		}
	}
	if !foundBasic {
		t.Error("DeidentificationMethodCodeSequence is missing the Basic Profile code 113100")
	}
}

func TestDeidentifyAppliesTopLevelActions(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagPatientName, "Doe^Jane")           // D: replaced, non-empty
	ds.SetString(TagPatientBirthDate, "19800101")      // Z: zero-length
	ds.SetString(TagPatientBirthTime, "120000")        // X: removed
	ds.SetString(TagAccessionNumber, "ACC123")         // Z
	ds.SetString(TagStudyDescription, "Brain w/ John") // C: cleaned (zeroed in v1)

	prof := NewProfile(testGenerator(t))
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	// D: present, non-empty, and not the original value.
	if v, ok := clean.GetString(TagPatientName); !ok || v == "" || v == "Doe^Jane" {
		t.Errorf("PatientName after D = %q,%v, want a non-empty dummy != original", v, ok)
	}
	// Z: present but zero-length.
	if e, ok := clean.Get(TagPatientBirthDate); !ok {
		t.Error("PatientBirthDate (Z) should remain present as zero-length")
	} else if e.Value.EncodedLen(nil) != 0 {
		t.Errorf("PatientBirthDate (Z) length = %d, want 0", e.Value.EncodedLen(nil))
	}
	// X: removed.
	if _, ok := clean.Get(TagPatientBirthTime); ok {
		t.Error("PatientBirthTime (X) should be removed")
	}
	// Z: zero-length.
	if e, _ := clean.Get(TagAccessionNumber); e.Value.EncodedLen(nil) != 0 {
		t.Errorf("AccessionNumber (Z) length = %d, want 0", e.Value.EncodedLen(nil))
	}
	// C: identity removed (zeroed) in v1.
	if e, ok := clean.Get(TagStudyDescription); !ok {
		t.Error("StudyDescription (C) should remain present")
	} else if e.Value.EncodedLen(nil) != 0 {
		t.Errorf("StudyDescription (C) length = %d, want 0 (v1 clean = zero)", e.Value.EncodedLen(nil))
	}
}

func TestDeidentifyNeverMutatesInput(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagPatientName, "Doe^Jane")
	ds.SetString(TagStudyInstanceUID, "1.2.3.4.5")
	ds.SetString(TagPatientBirthTime, "120000")

	// Snapshot the source by deep clone before de-identifying.
	before := ds.Clone()

	prof := NewProfile(testGenerator(t))
	if _, err := prof.Deidentify(ds); err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	if !dataSetsEqual(ds, before) {
		t.Error("Deidentify mutated its input dataset (Codex DCM-016 deep-copy invariant)")
	}
}

// dataSetsEqual compares two datasets by tag set and string-rendered values for the
// text VRs used in these tests. It is a test helper, not a general equality.
func dataSetsEqual(a, b *DataSet) bool {
	if a.Len() != b.Len() {
		return false
	}
	for e := range a.All() {
		be, ok := b.Get(e.Tag)
		if !ok || be.VR != e.VR {
			return false
		}
		av, aok := a.GetStrings(e.Tag)
		bv, bok := b.GetStrings(e.Tag)
		if aok != bok {
			return false
		}
		if aok {
			if len(av) != len(bv) {
				return false
			}
			for i := range av {
				if av[i] != bv[i] {
					return false
				}
			}
		}
	}
	return true
}

func TestNewProfileRequiresGeneratorForUIDRemap(t *testing.T) {
	// With a nil generator and the default (UID remap on), Deidentify must fail
	// closed rather than mint nothing or panic.
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	prof := NewProfile(nil)
	_, err := prof.Deidentify(ds)
	if err == nil {
		t.Fatal("Deidentify with nil generator and UID remap on should error, not silently keep UIDs")
	}
	if !errors.Is(err, errNoUIDGenerator) {
		t.Errorf("error = %v, want errNoUIDGenerator", err)
	}
}
