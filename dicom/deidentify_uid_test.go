package dicom

import "testing"

// A repeated source UID must map to one consistent replacement within a Deidentify
// call so the Study/Series/Instance reference graph survives de-identification
// (Codex DCM-013: referential integrity after remap). Here the same StudyInstanceUID
// appears at the top level and is referenced again inside a nested item; both must
// rewrite to the same new UID.
func TestDeidentifyUIDRemapIsConsistent(t *testing.T) {
	const study = "1.2.840.113619.2.55.3.1"
	const series = "1.2.840.113619.2.55.3.2"

	ref := NewDataSet()
	ref.SetString(TagReferencedSOPInstanceUID, study) // references the study UID

	top := NewDataSet()
	top.SetString(TagStudyInstanceUID, study)
	top.SetString(TagSeriesInstanceUID, series)
	// A second attribute that carries the same study UID, to prove consistency.
	top.SetString(TagFrameOfReferenceUID, study)
	top.Set(Element{Tag: TagSharedFunctionalGroupsSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(ref))})

	prof := NewProfile(testGenerator(t))
	clean, err := prof.Deidentify(top)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	newStudy, _ := clean.GetString(TagStudyInstanceUID)
	newSeries, _ := clean.GetString(TagSeriesInstanceUID)
	newFOR, _ := clean.GetString(TagFrameOfReferenceUID)

	if newStudy == study || newStudy == "" {
		t.Errorf("StudyInstanceUID not remapped: %q", newStudy)
	}
	if newSeries == series || newSeries == newStudy {
		t.Errorf("SeriesInstanceUID remap collided or was kept: %q (study %q)", newSeries, newStudy)
	}
	// Same source UID -> same replacement, at every occurrence.
	if newFOR != newStudy {
		t.Errorf("FrameOfReferenceUID %q != remapped study UID %q; remap is not consistent", newFOR, newStudy)
	}
	seq, _ := clean.GetSequence(TagSharedFunctionalGroupsSequence)
	var item Item
	for it := range seq.Items() {
		item = it
	}
	refUID, _ := item.DataSet.GetString(TagReferencedSOPInstanceUID)
	if refUID != newStudy {
		t.Errorf("nested ReferencedSOPInstanceUID %q != remapped study UID %q; reference graph broken", refUID, newStudy)
	}
}

// New UIDs must be conformant (PS3.5: <= 64 chars, valid components).
func TestDeidentifyRemappedUIDsAreValid(t *testing.T) {
	gen, err := NewUIDGenerator("1.2.840.99999")
	if err != nil {
		t.Fatalf("NewUIDGenerator: %v", err)
	}
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3")
	ds.SetString(TagSOPInstanceUID, "1.2.4")

	clean, err := NewProfile(gen).Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	for _, tg := range []Tag{TagStudyInstanceUID, TagSOPInstanceUID} {
		u, _ := clean.GetUID(tg)
		if err := u.Validate(); err != nil {
			t.Errorf("remapped %s = %q is not a valid UID: %v", tg, u, err)
		}
	}
}

// WithRetainUIDs leaves UIDs untouched (the PS3.15 Retain UIDs sub-option).
func TestDeidentifyRetainUIDsKeepsThem(t *testing.T) {
	const study = "1.2.3.4.5"
	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, study)

	prof := NewProfile(nil, WithRetainUIDs())
	clean, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify with WithRetainUIDs: %v", err)
	}
	if v, _ := clean.GetString(TagStudyInstanceUID); v != study {
		t.Errorf("StudyInstanceUID = %q, want unchanged %q under WithRetainUIDs", v, study)
	}
}
