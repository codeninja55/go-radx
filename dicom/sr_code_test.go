package dicom

import "testing"

func TestConceptNameCodeRoundTrip(t *testing.T) {
	want := ConceptNameCode{
		CodeValue:              "121071",
		CodingSchemeDesignator: "DCM",
		CodeMeaning:            "Finding",
	}
	item := codeItem(want)
	got, ok := readCode(item)
	if !ok {
		t.Fatal("readCode should resolve a populated code item")
	}
	if got != want {
		t.Errorf("readCode = %+v, want %+v", got, want)
	}
}

func TestReadCodeFromSequence(t *testing.T) {
	want := ConceptNameCode{CodeValue: "12345", CodingSchemeDesignator: "SCT", CodeMeaning: "Lesion"}
	ds := NewDataSet()
	ds.Set(Element{
		Tag:   TagConceptNameCodeSequence,
		VR:    VRSQ,
		Value: NewSequenceValue(NewSequence(codeItem(want))),
	})
	got, ok := readCodeSeq(ds, TagConceptNameCodeSequence)
	if !ok {
		t.Fatal("readCodeSeq should resolve the single-item code sequence")
	}
	if got != want {
		t.Errorf("readCodeSeq = %+v, want %+v", got, want)
	}
	if _, ok := readCodeSeq(ds, TagConceptCodeSequence); ok {
		t.Error("readCodeSeq of an absent sequence should report ok == false")
	}
}

func TestWriteCodeSeq(t *testing.T) {
	c := ConceptNameCode{CodeValue: "R-00339", CodingSchemeDesignator: "SRT", CodeMeaning: "Normal"}
	ds := NewDataSet()
	writeCodeSeq(ds, TagConceptNameCodeSequence, c)
	got, ok := readCodeSeq(ds, TagConceptNameCodeSequence)
	if !ok || got != c {
		t.Errorf("writeCodeSeq round-trip = (%+v,%v), want (%+v,true)", got, ok, c)
	}
}

func TestReferencedSOPRoundTrip(t *testing.T) {
	want := ReferencedSOPInstance{
		SOPClassUID:    SOPClassUID("1.2.840.10008.5.1.4.1.1.2"),
		SOPInstanceUID: SOPInstanceUID("1.2.3.4.5.6.7.8.9"),
	}
	ds := NewDataSet()
	writeReferencedSOPSeq(ds, []ReferencedSOPInstance{want})
	got, ok := readReferencedSOPSeq(ds)
	if !ok {
		t.Fatal("readReferencedSOPSeq should resolve a populated sequence")
	}
	if len(got) != 1 || got[0] != want {
		t.Errorf("readReferencedSOPSeq = %+v, want [%+v]", got, want)
	}
}

func TestReadReferencedSOPSeqMultiple(t *testing.T) {
	refs := []ReferencedSOPInstance{
		{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "1.1.1"},
		{SOPClassUID: "1.2.840.10008.5.1.4.1.1.4", SOPInstanceUID: "2.2.2"},
	}
	ds := NewDataSet()
	writeReferencedSOPSeq(ds, refs)
	got, ok := readReferencedSOPSeq(ds)
	if !ok || len(got) != 2 {
		t.Fatalf("readReferencedSOPSeq returned %d items, want 2", len(got))
	}
	for i := range refs {
		if got[i] != refs[i] {
			t.Errorf("ref[%d] = %+v, want %+v", i, got[i], refs[i])
		}
	}
}
