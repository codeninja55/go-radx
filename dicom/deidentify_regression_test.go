package dicom

import (
	"testing"
)

// TestDCM013BasicProfileDeidentification is the named regression for Codex DCM-013:
// the prototype's Basic profile was a sparse, top-level-only, no-op-pixel-cleaning
// implementation that falsely claimed PS3.15 compliance. This test de-identifies a
// dataset carrying confidentiality attributes both at the top level and nested inside
// sequences, and asserts the full corrected behaviour in one place:
//
//   - every targeted attribute is removed/replaced at BOTH levels (recursive coverage)
//   - a repeated source UID maps to one consistent new UID (referential integrity)
//   - dates are gone by default
//   - the (0012,006x) de-identification metadata is set, including code 113100
//   - the input DataSet is unmodified (deep copy)
//   - burned-in pixel PHI is honoured (here: absent, so de-identification proceeds)
func TestDCM013BasicProfileDeidentification(t *testing.T) {
	const sharedUID = "1.2.840.113619.STUDY"

	// Nested item carrying PHI and a back-reference to the shared study UID.
	nested := NewDataSet()
	nested.SetString(TagPatientName, "Nested^Patient")     // D
	nested.SetString(TagPatientBirthTime, "070000")        // X
	nested.SetString(TagInstitutionName, "Nested General") // X
	nested.SetString(TagOperatorsName, "Nested^Operator")  // Z
	nested.SetString(TagStudyDate, "20240101")             // Z
	nested.SetString(TagReferencedSOPInstanceUID, sharedUID)

	// A structural sequence (kept) holds the nested item, so the profile must recurse
	// into it rather than zeroing it. This is the exact shape the prototype missed.
	src := NewDataSet()
	src.SetString(TagPatientName, "Top^Patient")     // D
	src.SetString(TagPatientID, "MRN-12345")         // D
	src.SetString(TagPatientBirthDate, "19700101")   // Z
	src.SetString(TagPatientBirthTime, "120000")     // X
	src.SetString(TagAccessionNumber, "ACC-7788")    // Z
	src.SetString(TagReferringPhysicianName, "Dr X") // Z
	src.SetString(TagStudyDate, "20240101")          // Z
	src.SetString(TagStudyInstanceUID, sharedUID)    // U
	src.SetString(TagSeriesInstanceUID, "1.2.840.113619.SERIES")
	src.SetString(TagSOPInstanceUID, "1.2.840.113619.SOP")
	src.SetString(TagFrameOfReferenceUID, sharedUID) // U, same as study -> same remap
	// A private tag, removed by default.
	src.Set(Element{Tag: NewTag(0x0009, 0x0010), VR: VRLO, Value: NewStrings(VRLO, "PRIVCO")})
	src.Set(Element{Tag: NewTag(0x0009, 0x1001), VR: VRLO, Value: NewStrings(VRLO, "private secret")})
	src.Set(Element{
		Tag:   TagSharedFunctionalGroupsSequence,
		VR:    VRSQ,
		Value: NewSequenceValue(NewSequence(nested)),
	})

	// Snapshot the source to prove it is never mutated.
	before := src.Clone()

	gen, err := NewUIDGenerator("1.2.840.99999")
	if err != nil {
		t.Fatalf("NewUIDGenerator: %v", err)
	}
	clean, err := NewProfile(gen).Deidentify(src)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	// 1. Top-level X/Z/D actions.
	if _, ok := clean.Get(TagPatientBirthTime); ok {
		t.Error("top PatientBirthTime (X) survived")
	}
	if e, _ := clean.Get(TagAccessionNumber); e.Value.EncodedLen(nil) != 0 {
		t.Error("top AccessionNumber (Z) is not zero-length")
	}
	if v, _ := clean.GetString(TagPatientName); v == "Top^Patient" || v == "" {
		t.Errorf("top PatientName (D) = %q, want non-empty dummy != original", v)
	}
	if v, _ := clean.GetString(TagPatientID); v == "MRN-12345" || v == "" {
		t.Errorf("top PatientID (D) = %q, want non-empty dummy != original", v)
	}

	// 2. Dates gone by default.
	if e, ok := clean.Get(TagStudyDate); ok && e.Value.EncodedLen(nil) != 0 {
		t.Error("top StudyDate retained temporal data by default")
	}
	if e, ok := clean.Get(TagPatientBirthDate); ok && e.Value.EncodedLen(nil) != 0 {
		t.Error("top PatientBirthDate retained temporal data by default")
	}

	// 3. Private tags removed.
	if _, ok := clean.Get(NewTag(0x0009, 0x1001)); ok {
		t.Error("private data element survived")
	}
	if _, ok := clean.Get(NewTag(0x0009, 0x0010)); ok {
		t.Error("private creator survived")
	}

	// 4. UID remap is consistent: study and frame-of-reference shared one source UID,
	//    so they must share one replacement, and the nested back-reference too.
	newStudy, _ := clean.GetString(TagStudyInstanceUID)
	newFOR, _ := clean.GetString(TagFrameOfReferenceUID)
	if newStudy == sharedUID || newStudy == "" {
		t.Errorf("StudyInstanceUID not remapped: %q", newStudy)
	}
	if newFOR != newStudy {
		t.Errorf("FrameOfReferenceUID %q != study %q; remap inconsistent", newFOR, newStudy)
	}

	// 5. Recursive coverage: descend into the kept structural sequence.
	seq, ok := clean.GetSequence(TagSharedFunctionalGroupsSequence)
	if !ok || seq.Len() != 1 {
		t.Fatalf("SharedFunctionalGroupsSequence missing or empty: ok=%v", ok)
	}
	var item Item
	for it := range seq.Items() {
		item = it
	}
	if _, ok := item.DataSet.Get(TagPatientBirthTime); ok {
		t.Error("nested PatientBirthTime (X) survived (DCM-013)")
	}
	if _, ok := item.DataSet.Get(TagInstitutionName); ok {
		t.Error("nested InstitutionName (X) survived (DCM-013)")
	}
	if v, _ := item.DataSet.GetString(TagPatientName); v == "Nested^Patient" || v == "" {
		t.Errorf("nested PatientName (D) = %q, want non-empty dummy != original (DCM-013)", v)
	}
	if e, ok := item.DataSet.Get(TagStudyDate); ok && e.Value.EncodedLen(nil) != 0 {
		t.Error("nested StudyDate retained temporal data (DCM-013)")
	}
	// The nested back-reference must follow the same remap as the top-level study UID.
	if refUID, _ := item.DataSet.GetString(TagReferencedSOPInstanceUID); refUID != newStudy {
		t.Errorf("nested ReferencedSOPInstanceUID %q != remapped study %q; reference graph broken (DCM-013)", refUID, newStudy)
	}

	// 6. De-identification metadata.
	if v, _ := clean.GetString(TagPatientIdentityRemoved); v != "YES" {
		t.Errorf("PatientIdentityRemoved = %q, want YES", v)
	}
	if v, ok := clean.GetString(TagDeidentificationMethod); !ok || v == "" {
		t.Error("DeidentificationMethod not set")
	}
	mseq, ok := clean.GetSequence(TagDeidentificationMethodCodeSequence)
	if !ok {
		t.Fatal("DeidentificationMethodCodeSequence absent")
	}
	hasBasic := false
	for it := range mseq.Items() {
		if cv, _ := it.DataSet.GetString(TagCodeValue); cv == "113100" {
			hasBasic = true
		}
	}
	if !hasBasic {
		t.Error("DeidentificationMethodCodeSequence missing Basic Profile code 113100")
	}

	// 7. Input never mutated (deep copy).
	if !dataSetsEqual(src, before) {
		t.Error("Deidentify mutated its input dataset (deep-copy invariant)")
	}
}

// TestDeidentifyOnRealFixture de-identifies a parsed real DICOM file end to end as a
// feature test, confirming the profile runs on genuine data and sets the metadata.
func TestDeidentifyOnRealFixture(t *testing.T) {
	f, err := ReadFile("../testdata/dicom/liver.dcm")
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	srcStudy, hadStudy := f.DataSet.GetString(TagStudyInstanceUID)

	gen, err := NewUIDGenerator("1.2.840.99999")
	if err != nil {
		t.Fatalf("NewUIDGenerator: %v", err)
	}
	clean, err := NewProfile(gen, WithAllowBurnedInPixelData()).Deidentify(f.DataSet)
	if err != nil {
		t.Fatalf("Deidentify real fixture: %v", err)
	}

	if v, _ := clean.GetString(TagPatientIdentityRemoved); v != "YES" {
		t.Errorf("PatientIdentityRemoved = %q, want YES", v)
	}
	if hadStudy {
		if v, _ := clean.GetString(TagStudyInstanceUID); v == srcStudy {
			t.Error("StudyInstanceUID was not remapped on the real fixture")
		}
	}
	// PatientName, if present, must not retain its original value.
	if orig, ok := f.DataSet.GetString(TagPatientName); ok && orig != "" {
		if v, _ := clean.GetString(TagPatientName); v == orig {
			t.Error("PatientName survived on the real fixture")
		}
	}
}
